package repositories

import (
	"context"
	"math"

	"cloud.google.com/go/firestore"
	"github.com/adrianghub/portfelik-bff/internal/logger"
	"github.com/adrianghub/portfelik-bff/internal/models"
	"github.com/adrianghub/portfelik-bff/internal/utils"
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
	groupIDs, ok := userData["groupIds"].([]any)
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

func (r *TransactionRepository) GetTransactionSummaryByMonth(
	ctx context.Context,
	userID string,
	startDate, endDate string,
	categoryRepository *CategoryRepository,
) (*models.MonthlySummary, error) {
	transactions, err := r.GetTransactionsByDateRange(ctx, userID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	// Get all categories for this user (including shared ones)
	categories, err := categoryRepository.GetAllUserCategories(ctx, userID)
	if err != nil {
		r.logger.Error("Error fetching categories: %v", err)
		// Continue with empty categories - we'll use category IDs as names
		categories = []models.Category{}
	}

	// Create a map for quick category lookup
	categoryMap := make(map[string]models.Category)
	for _, cat := range categories {
		categoryMap[cat.ID] = cat
	}

	// Extract the month from the start date
	month, err := utils.ExtractMonthKey(startDate)
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

		// Get category name or use ID if not found
		categoryName := categoryID
		if cat, found := categoryMap[categoryID]; found {
			categoryName = cat.Name
		}

		monthlySummary.CategorySummaries = append(monthlySummary.CategorySummaries, models.CategorySummary{
			CategoryID:       categoryID,
			CategoryName:     categoryName,
			Amount:           amount,
			Percentage:       percentage,
			TransactionCount: categoryTransactionCounts[categoryID],
		})
	}

	return monthlySummary, nil
}

// GetSharedTransactionSummaryWithCategoryNames returns shared transaction summary with category names
func (r *TransactionRepository) GetSharedTransactionSummaryByMonth(
	ctx context.Context,
	userID string,
	startDate, endDate string,
	categoryRepository *CategoryRepository,
) (*models.MonthlySummary, error) {
	transactions, err := r.GetSharedTransactionsByDateRange(ctx, userID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	categories, err := categoryRepository.GetAllUserCategories(ctx, userID)
	if err != nil {
		r.logger.Error("Error fetching categories: %v", err)
		categories = []models.Category{}
	}

	categoryMap := make(map[string]models.Category)
	for _, cat := range categories {
		categoryMap[cat.ID] = cat
	}

	month, err := utils.ExtractMonthKey(startDate)
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

		// Get category name or use ID if not found
		categoryName := categoryID
		if cat, found := categoryMap[categoryID]; found {
			categoryName = cat.Name
		}

		monthlySummary.CategorySummaries = append(monthlySummary.CategorySummaries, models.CategorySummary{
			CategoryID:       categoryID,
			CategoryName:     categoryName,
			Amount:           amount,
			Percentage:       percentage,
			TransactionCount: categoryTransactionCounts[categoryID],
		})
	}

	return monthlySummary, nil
}
