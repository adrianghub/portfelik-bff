package repositories

import (
	"context"
	"sort"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/adrianghub/portfelik-bff/internal/logger"
	"github.com/adrianghub/portfelik-bff/internal/models"
	"google.golang.org/api/iterator"
)

const (
	categoriesCollection = "categories"
)

type CategoryRepository struct {
	client *firestore.Client
	logger *logger.Logger
}

func NewCategoryRepository(client *firestore.Client, logger *logger.Logger) *CategoryRepository {
	return &CategoryRepository{
		client: client,
		logger: logger,
	}
}

func (r *CategoryRepository) GetUserCategories(ctx context.Context, userID string) ([]models.Category, error) {
	r.logger.Info("Getting categories for user: %s", userID)

	var categories []models.Category

	query := r.client.Collection(categoriesCollection).
		Where("userId", "==", userID).
		OrderBy("name", firestore.Asc)

	iter := query.Documents(ctx)
	defer iter.Stop()

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			r.logger.Error("Error iterating categories: %v", err)
			return nil, err
		}

		var category models.Category
		if err := doc.DataTo(&category); err != nil {
			r.logger.Error("Error converting document to category: %v", err)
			return nil, err
		}
		category.ID = doc.Ref.ID

		categories = append(categories, category)
	}

	return categories, nil
}

func (r *CategoryRepository) GetSharedCategories(ctx context.Context, userID string) ([]models.Category, error) {
	r.logger.Info("Getting shared categories for user: %s", userID)

	userDoc, err := r.client.Collection("users").Doc(userID).Get(ctx)
	if err != nil {
		r.logger.Error("Error getting user document: %v", err)
		return nil, err
	}

	userData := userDoc.Data()
	groupIDs, ok := userData["groupIds"].([]any)
	if !ok || len(groupIDs) == 0 {
		return []models.Category{}, nil
	}

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
		members, ok := groupData["memberIds"].([]any)
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
		return []models.Category{}, nil
	}

	// Convert to array for firestore "in" query
	memberIDsArray := make([]string, 0, len(memberIDs))
	for id := range memberIDs {
		memberIDsArray = append(memberIDsArray, id)
	}

	// Query for shared categories
	query := r.client.Collection(categoriesCollection).
		Where("userId", "in", memberIDsArray).
		OrderBy("name", firestore.Asc)

	iter := query.Documents(ctx)
	defer iter.Stop()

	var categories []models.Category
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			r.logger.Error("Error iterating shared categories: %v", err)
			return nil, err
		}

		var category models.Category
		if err := doc.DataTo(&category); err != nil {
			r.logger.Error("Error converting document to shared category: %v", err)
			return nil, err
		}
		category.ID = doc.Ref.ID

		categories = append(categories, category)
	}

	return categories, nil
}

// GetAllUserCategories returns both user's own categories and shared categories
func (r *CategoryRepository) GetAllUserCategories(ctx context.Context, userID string) ([]models.Category, error) {
	r.logger.Info("Getting all categories for user: %s", userID)

	ownCategories, err := r.GetUserCategories(ctx, userID)
	if err != nil {
		return nil, err
	}

	sharedCategories, err := r.GetSharedCategories(ctx, userID)
	if err != nil {
		return nil, err
	}

	allCategories := append(ownCategories, sharedCategories...)

	uniqueCategories := make(map[string]models.Category)
	for _, cat := range allCategories {
		uniqueCategories[cat.ID] = cat
	}

	result := make([]models.Category, 0, len(uniqueCategories))
	for _, cat := range uniqueCategories {
		result = append(result, cat)
	}

	sort.Slice(result, func(i, j int) bool {
		return strings.ToLower(result[i].Name) < strings.ToLower(result[j].Name)
	})

	return result, nil
}
