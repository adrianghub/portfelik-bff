package repositories

import (
	"context"
	"math"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/panizinko/portfelik-bff/internal/logger"
	"github.com/panizinko/portfelik-bff/internal/models"
	"google.golang.org/api/iterator"
)

const (
	transactionsCollection = "transactions"
)

type TransactionRepository struct {
	client *firestore.Client
	logger *logger.Logger
}

func NewTransactionRepository(client *firestore.Client, logger *logger.Logger) *TransactionRepository {
	return &TransactionRepository{
		client: client,
		logger: logger,
	}
}

func (r *TransactionRepository) GetTransactionsByDateRange(
	ctx context.Context,
	userID string,
	startDate, endDate string,
) ([]models.Transaction, error) {
	var transactions []models.Transaction

	query := r.client.Collection(transactionsCollection).
		Where("userId", "==", userID).
		OrderBy("date", firestore.Desc)

	if startDate != "" && endDate != "" {
		query = r.client.Collection(transactionsCollection).
			Where("userId", "==", userID).
			Where("date", ">=", startDate).
			Where("date", "<=", endDate).
			OrderBy("date", firestore.Desc)
	}

	iter := query.Documents(ctx)
	defer iter.Stop()

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var transaction models.Transaction
		if err := doc.DataTo(&transaction); err != nil {
			return nil, err
		}
		transaction.ID = doc.Ref.ID

		transactions = append(transactions, transaction)
	}

	return transactions, nil
}

func (r *TransactionRepository) GetAllTransactionsByDateRange(
	ctx context.Context,
	startDate, endDate string,
) ([]models.Transaction, error) {
	var transactions []models.Transaction

	query := r.client.Collection(transactionsCollection).
		OrderBy("date", firestore.Desc)

	if startDate != "" && endDate != "" {
		query = r.client.Collection(transactionsCollection).
			Where("date", ">=", startDate).
			Where("date", "<=", endDate).
			OrderBy("date", firestore.Desc)
	}

	iter := query.Documents(ctx)
	defer iter.Stop()

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var transaction models.Transaction
		if err := doc.DataTo(&transaction); err != nil {
			return nil, err
		}
		transaction.ID = doc.Ref.ID

		transactions = append(transactions, transaction)
	}

	return transactions, nil
}

func (r *TransactionRepository) GetSharedTransactionsByDateRange(
	ctx context.Context,
	userID string,
	startDate, endDate string,
) ([]models.Transaction, error) {
	userDoc, err := r.client.Collection("users").Doc(userID).Get(ctx)
	if err != nil {
		return nil, err
	}

	userData := userDoc.Data()
	groupIDs, ok := userData["groupIds"].([]interface{})
	if !ok || len(groupIDs) == 0 {
		return []models.Transaction{}, nil
	}

	// Get member IDs from groups
	memberIDs := make(map[string]bool)
	for _, gID := range groupIDs {
		groupID, ok := gID.(string)
		if !ok {
			continue
		}

		groupDoc, err := r.client.Collection("user-groups").Doc(groupID).Get(ctx)
		if err != nil {
			continue
		}

		groupData := groupDoc.Data()
		members, ok := groupData["memberIds"].([]interface{})
		if !ok {
			continue
		}

		for _, m := range members {
			memberID, ok := m.(string)
			if ok && memberID != userID {
				memberIDs[memberID] = true
			}
		}
	}

	if len(memberIDs) == 0 {
		return []models.Transaction{}, nil
	}

	// Convert to array for firestore "in" query
	memberIDsArray := make([]string, 0, len(memberIDs))
	for id := range memberIDs {
		memberIDsArray = append(memberIDsArray, id)
	}

	query := r.client.Collection(transactionsCollection).
		Where("userId", "in", memberIDsArray).
		OrderBy("date", firestore.Desc)

	if startDate != "" && endDate != "" {
		query = r.client.Collection(transactionsCollection).
			Where("userId", "in", memberIDsArray).
			Where("date", ">=", startDate).
			Where("date", "<=", endDate).
			OrderBy("date", firestore.Desc)
	}

	iter := query.Documents(ctx)
	defer iter.Stop()

	var transactions []models.Transaction
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var transaction models.Transaction
		if err := doc.DataTo(&transaction); err != nil {
			return nil, err
		}
		transaction.ID = doc.Ref.ID

		transactions = append(transactions, transaction)
	}

	return transactions, nil
}

func extractMonthKey(dateStr string) (string, error) {
	var t time.Time
	var err error

	t, err = time.Parse(time.RFC3339, dateStr)
	if err != nil {
		t, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			return "", err
		}
	}

	return t.Format("2006-01"), nil
}

func (r *TransactionRepository) GetTransactionSummaryByMonth(
	ctx context.Context,
	userID string,
	startDate, endDate string,
) (*models.MonthlySummary, error) {
	transactions, err := r.GetTransactionsByDateRange(ctx, userID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	// Extract the month from the start date since we know it's always from the beginning of the month
	month, err := extractMonthKey(startDate)
	if err != nil {
		return nil, err
	}

	monthlySummary := &models.MonthlySummary{
		Month:             month,
		TotalExpenses:     0,
		TotalIncome:       0,
		Delta:             0,
		CategorySummaries: []models.CategorySummary{},
	}

	categoryAmounts := make(map[string]float64)
	categoryTransactionCounts := make(map[string]int)

	for _, transaction := range transactions {
		if transaction.Type == "expense" {
			monthlySummary.TotalExpenses += math.Abs(transaction.Amount)
			categoryAmounts[transaction.CategoryID] += math.Abs(transaction.Amount)
			categoryTransactionCounts[transaction.CategoryID]++
		} else if transaction.Type == "income" {
			monthlySummary.TotalIncome += transaction.Amount
		}
	}

	monthlySummary.Delta = monthlySummary.TotalIncome - monthlySummary.TotalExpenses

	for categoryID, amount := range categoryAmounts {
		percentage := 0.0
		if monthlySummary.TotalExpenses > 0 {
			percentage = (amount / monthlySummary.TotalExpenses) * 100
		}

		monthlySummary.CategorySummaries = append(monthlySummary.CategorySummaries, models.CategorySummary{
			CategoryID:       categoryID,
			Amount:           amount,
			Percentage:       percentage,
			TransactionCount: categoryTransactionCounts[categoryID],
		})
	}

	return monthlySummary, nil
}

// GetAllTransactionSummaryByMonth retrieves and aggregates all transaction data by month for admin users
func (r *TransactionRepository) GetAllTransactionSummaryByMonth(
	ctx context.Context,
	startDate, endDate string,
) ([]models.MonthlySummary, error) {
	transactions, err := r.GetAllTransactionsByDateRange(ctx, startDate, endDate)
	if err != nil {
		return nil, err
	}

	monthlySummaries := make(map[string]*models.MonthlySummary)

	for _, transaction := range transactions {
		month, err := extractMonthKey(transaction.Date)
		if err != nil {
			r.logger.Error("Invalid date format: %s, error: %v", transaction.Date, err)
			continue
		}

		if _, exists := monthlySummaries[month]; !exists {
			monthlySummaries[month] = &models.MonthlySummary{
				Month:             month,
				TotalExpenses:     0,
				TotalIncome:       0,
				Delta:             0,
				CategorySummaries: []models.CategorySummary{},
			}
		}

		if transaction.Type == "expense" {
			monthlySummaries[month].TotalExpenses += math.Abs(transaction.Amount)
		} else if transaction.Type == "income" {
			monthlySummaries[month].TotalIncome += transaction.Amount
		}
	}

	for _, summary := range monthlySummaries {
		summary.Delta = summary.TotalIncome - summary.TotalExpenses

		categoryAmounts := make(map[string]float64)
		categoryTransactionCounts := make(map[string]int)

		for _, transaction := range transactions {
			if transaction.Type == "expense" {
				categoryAmounts[transaction.CategoryID] += math.Abs(transaction.Amount)
				categoryTransactionCounts[transaction.CategoryID]++
			}
		}

		for categoryID, amount := range categoryAmounts {
			percentage := 0.0
			if summary.TotalExpenses > 0 {
				percentage = (amount / summary.TotalExpenses) * 100
			}

			summary.CategorySummaries = append(summary.CategorySummaries, models.CategorySummary{
				CategoryID:       categoryID,
				Amount:           amount,
				Percentage:       percentage,
				TransactionCount: categoryTransactionCounts[categoryID],
			})
		}
	}

	result := make([]models.MonthlySummary, 0, len(monthlySummaries))
	for _, summary := range monthlySummaries {
		result = append(result, *summary)
	}

	return result, nil
}

func (r *TransactionRepository) GetSharedTransactionSummaryByMonth(
	ctx context.Context,
	userID string,
	startDate, endDate string,
) (*models.MonthlySummary, error) {
	transactions, err := r.GetSharedTransactionsByDateRange(ctx, userID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	// Extract the month from the start date since we know it's always from the beginning of the month
	month, err := extractMonthKey(startDate)
	if err != nil {
		return nil, err
	}

	monthlySummary := &models.MonthlySummary{
		Month:             month,
		TotalExpenses:     0,
		TotalIncome:       0,
		Delta:             0,
		CategorySummaries: []models.CategorySummary{},
	}

	categoryAmounts := make(map[string]float64)
	categoryTransactionCounts := make(map[string]int)

	for _, transaction := range transactions {
		if transaction.Type == "expense" {
			monthlySummary.TotalExpenses += math.Abs(transaction.Amount)
			categoryAmounts[transaction.CategoryID] += math.Abs(transaction.Amount)
			categoryTransactionCounts[transaction.CategoryID]++
		} else if transaction.Type == "income" {
			monthlySummary.TotalIncome += transaction.Amount
		}
	}

	monthlySummary.Delta = monthlySummary.TotalIncome - monthlySummary.TotalExpenses

	for categoryID, amount := range categoryAmounts {
		percentage := 0.0
		if monthlySummary.TotalExpenses > 0 {
			percentage = (amount / monthlySummary.TotalExpenses) * 100
		}

		monthlySummary.CategorySummaries = append(monthlySummary.CategorySummaries, models.CategorySummary{
			CategoryID:       categoryID,
			Amount:           amount,
			Percentage:       percentage,
			TransactionCount: categoryTransactionCounts[categoryID],
		})
	}

	return monthlySummary, nil
}
