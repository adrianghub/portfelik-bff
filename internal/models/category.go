package models

// Category represents a transaction category
type Category struct {
	ID     string `json:"id" firestore:"id,omitempty"`
	Name   string `json:"name" firestore:"name"`
	Type   string `json:"type" firestore:"type"`
	UserID string `json:"userId" firestore:"userId,omitempty"`
}

type CategoryResponse struct {
	Categories []Category `json:"categories"`
}
