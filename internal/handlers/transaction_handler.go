package handlers

import (
	"encoding/json"
	"net/http"

	firebaseauth "firebase.google.com/go/v4/auth"
	"github.com/go-chi/chi/v5"
	localauth "github.com/panizinko/portfelik-bff/internal/auth"
	"github.com/panizinko/portfelik-bff/internal/logger"
	"github.com/panizinko/portfelik-bff/internal/models"
	"github.com/panizinko/portfelik-bff/internal/repositories"
)

type TransactionHandler struct {
	transactionRepository *repositories.TransactionRepository
	logger                *logger.Logger
}

func NewTransactionHandler(transactionRepository *repositories.TransactionRepository, logger *logger.Logger) *TransactionHandler {
	return &TransactionHandler{
		transactionRepository: transactionRepository,
		logger:                logger,
	}
}

func (h *TransactionHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.getTransactionsByDateRange)
	r.Get("/shared", h.getSharedTransactionsByDateRange)
	r.Get("/summary", h.getTransactionSummaryByMonth)
	r.Get("/summary/shared", h.getSharedTransactionSummaryByMonth)

	return r
}

func (h *TransactionHandler) getTransactionsByDateRange(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Transaction request received, checking auth token")
	h.logger.Info("Request headers: %v", r.Header)

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
	h.logger.Info("Fetching transactions for user: %s", userID)
	h.logger.Info("Token claims: %v", token.Claims)
	h.logger.Info("Token issued at: %v, expires at: %v", token.IssuedAt, token.Expires)

	var transactions = []models.Transaction{}
	var err error

	startDate := r.URL.Query().Get("startDate")
	endDate := r.URL.Query().Get("endDate")

	isAdmin := false
	claims := token.Claims
	if role, ok := claims["role"]; ok {
		isAdmin = role == "admin"
	}

	h.logger.Info("Is admin: %v", isAdmin)

	if isAdmin {
		transactions, err = h.transactionRepository.GetAllTransactionsByDateRange(r.Context(), startDate, endDate)
	} else {
		transactions, err = h.transactionRepository.GetTransactionsByDateRange(r.Context(), userID, startDate, endDate)
	}

	if err != nil {
		h.logger.Error("Failed to get transactions: %v", err)
		http.Error(w, "Failed to get transactions", http.StatusInternalServerError)
		return
	}

	if transactions == nil {
		transactions = []models.Transaction{}
	}

	response := models.TransactionResponse{
		Transactions: transactions,
	}

	w.Header().Set("Content-Type", "application/json")
	responseJSON, err := json.Marshal(response)
	if err != nil {
		h.logger.Error("Failed to encode response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

	_, err = w.Write(responseJSON)
	if err != nil {
		h.logger.Error("Failed to write response: %v", err)
	}
}

func (h *TransactionHandler) getSharedTransactionsByDateRange(w http.ResponseWriter, r *http.Request) {
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

	var err error

	startDate := r.URL.Query().Get("startDate")
	endDate := r.URL.Query().Get("endDate")

	transactions, err := h.transactionRepository.GetSharedTransactionsByDateRange(r.Context(), userID, startDate, endDate)
	if err != nil {
		h.logger.Error("Failed to get shared transactions: %v", err)
		http.Error(w, "Failed to get shared transactions", http.StatusInternalServerError)
		return
	}

	if transactions == nil {
		transactions = []models.Transaction{}
	}

	response := models.TransactionResponse{
		Transactions: transactions,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func (h *TransactionHandler) getTransactionSummaryByMonth(w http.ResponseWriter, r *http.Request) {
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

	startDate := r.URL.Query().Get("startDate")
	endDate := r.URL.Query().Get("endDate")

	var summaries []models.MonthlySummary
	var err error

	isAdmin := false
	claims := token.Claims
	if role, ok := claims["role"]; ok {
		isAdmin = role == "admin"
	}

	if isAdmin {
		summaries, err = h.transactionRepository.GetAllTransactionSummaryByMonth(r.Context(), startDate, endDate)
	} else {
		summaries, err = h.transactionRepository.GetTransactionSummaryByMonth(r.Context(), userID, startDate, endDate)
	}

	if err != nil {
		h.logger.Error("Failed to get transaction summary: %v", err)
		http.Error(w, "Failed to get transaction summary", http.StatusInternalServerError)
		return
	}

	if summaries == nil {
		summaries = []models.MonthlySummary{}
	}

	response := models.TransactionSummaryResponse{
		Summaries: summaries,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func (h *TransactionHandler) getSharedTransactionSummaryByMonth(w http.ResponseWriter, r *http.Request) {
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

	startDate := r.URL.Query().Get("startDate")
	endDate := r.URL.Query().Get("endDate")

	summaries, err := h.transactionRepository.GetSharedTransactionSummaryByMonth(r.Context(), userID, startDate, endDate)
	if err != nil {
		h.logger.Error("Failed to get shared transaction summary: %v", err)
		http.Error(w, "Failed to get shared transaction summary", http.StatusInternalServerError)
		return
	}

	if summaries == nil {
		summaries = []models.MonthlySummary{}
	}

	response := models.TransactionSummaryResponse{
		Summaries: summaries,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
