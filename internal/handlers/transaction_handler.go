package handlers

import (
	"encoding/json"
	"math"
	"net/http"
	"sort"

	firebaseauth "firebase.google.com/go/v4/auth"
	localauth "github.com/adrianghub/portfelik-bff/internal/auth"
	"github.com/adrianghub/portfelik-bff/internal/logger"
	"github.com/adrianghub/portfelik-bff/internal/models"
	"github.com/adrianghub/portfelik-bff/internal/repositories"
	"github.com/go-chi/chi/v5"
)

type TransactionHandler struct {
	transactionRepository *repositories.TransactionRepository
	categoryRepository    *repositories.CategoryRepository
	logger                *logger.Logger
}

func NewTransactionHandler(
	transactionRepository *repositories.TransactionRepository,
	categoryRepository *repositories.CategoryRepository,
	logger *logger.Logger,
) *TransactionHandler {
	return &TransactionHandler{
		transactionRepository: transactionRepository,
		categoryRepository:    categoryRepository,
		logger:                logger,
	}
}

func (h *TransactionHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.getAllTransactions)
	r.Get("/summary", h.getAllTransactionSummary)

	return r
}

func (h *TransactionHandler) getAllTransactions(w http.ResponseWriter, r *http.Request) {
	tokenValue := r.Context().Value(localauth.UserContextKey)
	if tokenValue == nil {
		h.logger.Error("No auth token found in request context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	token, ok := tokenValue.(*firebaseauth.Token)
	if !ok {
		h.logger.Error("Invalid token type in context")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	userID := token.UID
	h.logger.Info("Fetching all transactions for user: %s", userID)

	queryParams := r.URL.Query()
	startDate := queryParams.Get("startDate")
	endDate := queryParams.Get("endDate")
	categoryId := queryParams.Get("category")

	// Fetch both user and shared transactions concurrently
	userTransactionsChan := make(chan []models.Transaction)
	sharedTransactionsChan := make(chan []models.Transaction)
	userErrChan := make(chan error)
	sharedErrChan := make(chan error)

	go func() {
		transactions, err := h.transactionRepository.GetTransactionsByDateRange(r.Context(), userID, startDate, endDate, categoryId)
		userTransactionsChan <- transactions
		userErrChan <- err
	}()

	go func() {
		transactions, err := h.transactionRepository.GetSharedTransactionsByDateRange(r.Context(), userID, startDate, endDate, categoryId)
		sharedTransactionsChan <- transactions
		sharedErrChan <- err
	}()

	userTransactions := <-userTransactionsChan
	sharedTransactions := <-sharedTransactionsChan
	userErr := <-userErrChan
	sharedErr := <-sharedErrChan

	if userErr != nil {
		h.logger.Error("Failed to get user transactions: %v", userErr)
		http.Error(w, "Failed to get transactions", http.StatusInternalServerError)
		return
	}

	if sharedErr != nil {
		h.logger.Error("Failed to get shared transactions: %v", sharedErr)
		http.Error(w, "Failed to get transactions", http.StatusInternalServerError)
		return
	}

	// Combine user and shared transactions
	allTransactions := append(userTransactions, sharedTransactions...)
	if allTransactions == nil {
		allTransactions = []models.Transaction{}
	}

	response := models.TransactionResponse{
		Transactions: allTransactions,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func (h *TransactionHandler) getAllTransactionSummary(w http.ResponseWriter, r *http.Request) {
	tokenValue := r.Context().Value(localauth.UserContextKey)
	if tokenValue == nil {
		h.logger.Error("No auth token found in request context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	token, ok := tokenValue.(*firebaseauth.Token)
	if !ok {
		h.logger.Error("Invalid token type in context")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	userID := token.UID
	h.logger.Info("Fetching all transaction summaries for user: %s", userID)

	// Get query parameters
	queryParams := r.URL.Query()
	startDate := queryParams.Get("startDate")
	endDate := queryParams.Get("endDate")
	categoryId := queryParams.Get("category")

	// Fetch both user and shared transaction summaries concurrently
	userSummaryChan := make(chan *models.MonthlySummary)
	sharedSummaryChan := make(chan *models.MonthlySummary)
	userErrChan := make(chan error)
	sharedErrChan := make(chan error)

	go func() {
		summary, err := h.transactionRepository.GetTransactionSummaryByMonth(
			r.Context(),
			userID,
			startDate,
			endDate,
			categoryId,
			h.categoryRepository,
		)
		userSummaryChan <- summary
		userErrChan <- err
	}()

	go func() {
		summary, err := h.transactionRepository.GetSharedTransactionSummaryByMonth(
			r.Context(),
			userID,
			startDate,
			endDate,
			categoryId,
			h.categoryRepository,
		)
		sharedSummaryChan <- summary
		sharedErrChan <- err
	}()

	userSummary := <-userSummaryChan
	sharedSummary := <-sharedSummaryChan
	userErr := <-userErrChan
	sharedErr := <-sharedErrChan

	if userErr != nil {
		h.logger.Error("Failed to get user transaction summary: %v", userErr)
		http.Error(w, "Failed to get transaction summary", http.StatusInternalServerError)
		return
	}

	if sharedErr != nil {
		h.logger.Error("Failed to get shared transaction summary: %v", sharedErr)
		http.Error(w, "Failed to get transaction summary", http.StatusInternalServerError)
		return
	}

	// Combine user and shared summaries
	combinedSummary := h.combineSummaries(userSummary, sharedSummary)

	response := models.TransactionSummaryResponse{
		Summary: combinedSummary,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// combineSummaries combines user and shared transaction summaries
func (h *TransactionHandler) combineSummaries(userSummary, sharedSummary *models.MonthlySummary) *models.MonthlySummary {
	if userSummary == nil && sharedSummary == nil {
		return nil // Or return an empty summary object
	}
	if userSummary == nil {
		return sharedSummary
	}
	if sharedSummary == nil {
		return userSummary
	}

	combined := &models.MonthlySummary{
		Month:         userSummary.Month,
		TotalExpenses: userSummary.TotalExpenses + sharedSummary.TotalExpenses,
		TotalIncome:   userSummary.TotalIncome + sharedSummary.TotalIncome,
		Delta:         (userSummary.TotalIncome + sharedSummary.TotalIncome) - (userSummary.TotalExpenses + sharedSummary.TotalExpenses),
	}

	// Combine category summaries
	categoryMap := make(map[string]models.CategorySummary)

	// Add user categories
	for _, cat := range userSummary.CategorySummaries {
		categoryMap[cat.CategoryID] = cat
	}

	// Add or merge shared categories
	for _, sharedCat := range sharedSummary.CategorySummaries {
		if existingCat, ok := categoryMap[sharedCat.CategoryID]; ok {
			// Category exists, merge amounts and counts
			existingCat.Amount += sharedCat.Amount
			existingCat.TransactionCount += sharedCat.TransactionCount
			categoryMap[sharedCat.CategoryID] = existingCat
		} else {
			// New category, add it
			categoryMap[sharedCat.CategoryID] = sharedCat
		}
	}

	// Recalculate percentages based on combined totals
	combinedTotalCategorizedAmount := 0.0
	for _, cat := range categoryMap {
		combinedTotalCategorizedAmount += math.Abs(cat.Amount)
	}

	combinedCategories := make([]models.CategorySummary, 0, len(categoryMap))
	for _, cat := range categoryMap {
		if combinedTotalCategorizedAmount != 0 {
			cat.Percentage = (math.Abs(cat.Amount) / combinedTotalCategorizedAmount) * 100
		} else {
			cat.Percentage = 0
		}
		combinedCategories = append(combinedCategories, cat)
	}

	// Sort combined categories (e.g., by amount or name)
	sort.Slice(combinedCategories, func(i, j int) bool {
		// Example: Sort by absolute amount descending
		return math.Abs(combinedCategories[i].Amount) > math.Abs(combinedCategories[j].Amount)
	})

	combined.CategorySummaries = combinedCategories

	return combined
}
