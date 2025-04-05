package models

type Transaction struct {
	ID             string  `json:"id" firestore:"id,omitempty"`
	UserID         string  `json:"userId" firestore:"userId"`
	Amount         float64 `json:"amount" firestore:"amount"`
	Description    string  `json:"description" firestore:"description"`
	CategoryID     string  `json:"categoryId" firestore:"categoryId"`
	Type           string  `json:"type" firestore:"type"`
	Date           string  `json:"date" firestore:"date"`
	CreatedAt      string  `json:"createdAt,omitempty" firestore:"createdAt,omitempty"`
	UpdatedAt      string  `json:"updatedAt,omitempty" firestore:"updatedAt,omitempty"`
	ShoppingListID string  `json:"shoppingListId,omitempty" firestore:"shoppingListId,omitempty"`
}

type TransactionResponse struct {
	Transactions []Transaction `json:"transactions"`
}
