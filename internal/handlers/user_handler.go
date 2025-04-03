package handlers

import (
	"encoding/json"
	"net/http"

	"firebase.google.com/go/v4/auth"
	"github.com/go-chi/chi/v5"
	"github.com/panizinko/portfelik-bff/internal/logger"
)

type UserHandler struct {
	firebaseAuth *auth.Client
	logger       *logger.Logger
}

type UserProfile struct {
	UID         string `json:"uid"`
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
	PhotoURL    string `json:"photoURL,omitempty"`
}

func NewUserHandler(firebaseAuth *auth.Client, logger *logger.Logger) *UserHandler {
	return &UserHandler{
		firebaseAuth: firebaseAuth,
		logger:       logger,
	}
}

func (h *UserHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/me", h.getCurrentUser)
	r.Get("/{uid}", h.getUserByID)

	return r
}

func (h *UserHandler) getCurrentUser(w http.ResponseWriter, r *http.Request) {
	token := r.Context().Value("user").(*auth.Token)
	uid := token.UID

	user, err := h.firebaseAuth.GetUser(r.Context(), uid)
	if err != nil {
		h.logger.Error("Failed to get user: %v", err)
		http.Error(w, "Failed to get user", http.StatusInternalServerError)
		return
	}

	profile := UserProfile{
		UID:         user.UID,
		Email:       user.Email,
		DisplayName: user.DisplayName,
		PhotoURL:    user.PhotoURL,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(profile); err != nil {
		h.logger.Error("Failed to encode response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func (h *UserHandler) getUserByID(w http.ResponseWriter, r *http.Request) {
	uid := chi.URLParam(r, "uid")
	if uid == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	user, err := h.firebaseAuth.GetUser(r.Context(), uid)
	if err != nil {
		h.logger.Error("Failed to get user: %v", err)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	profile := UserProfile{
		UID:         user.UID,
		Email:       user.Email,
		DisplayName: user.DisplayName,
		PhotoURL:    user.PhotoURL,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(profile); err != nil {
		h.logger.Error("Failed to encode response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
