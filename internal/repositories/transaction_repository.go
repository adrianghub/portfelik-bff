package repositories

import (
	"context"

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
