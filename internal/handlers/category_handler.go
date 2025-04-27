package handlers

import (
	"encoding/json"
	"net/http"

	firebaseauth "firebase.google.com/go/v4/auth"
	localauth "github.com/adrianghub/portfelik-bff/internal/auth"
	"github.com/adrianghub/portfelik-bff/internal/logger"
	"github.com/adrianghub/portfelik-bff/internal/models"
	"github.com/adrianghub/portfelik-bff/internal/repositories"
	"github.com/go-chi/chi/v5"
)

type CategoryHandler struct {
	categoryRepository *repositories.CategoryRepository
	logger             *logger.Logger
}

func NewCategoryHandler(categoryRepository *repositories.CategoryRepository, logger *logger.Logger) *CategoryHandler {
	return &CategoryHandler{
		categoryRepository: categoryRepository,
		logger:             logger,
	}
}

func (h *CategoryHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.getAllUserCategories)

	return r
}

func (h *CategoryHandler) getAllUserCategories(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Category request received, checking auth token")

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
	h.logger.Info("Fetching categories for user: %s", userID)

	categories, err := h.categoryRepository.GetAllUserCategories(r.Context(), userID)
	if err != nil {
		h.logger.Error("Failed to get categories: %v", err)
		http.Error(w, "Failed to get categories", http.StatusInternalServerError)
		return
	}

	if categories == nil {
		categories = []models.Category{}
	}

	response := models.CategoryResponse{
		Categories: categories,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
