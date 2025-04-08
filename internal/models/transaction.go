package models

type Transaction struct {
	ID             string  `json:"id" firestore:"id,omitempty"`
	UserID         string  `json:"userId" firestore:"userId"`
	Amount         float64 `json:"amount" firestore:"amount"`
	Description    string  `json:"description" firestore:"description"`
	CategoryID     string  `json:"categoryId" firestore:"categoryId"`
	Type           string  `json:"type" firestore:"type"`
	Date           string  `json:"date" firestore:"date"`
	Status         string  `json:"status" firestore:"status"`
	IsRecurring    bool    `json:"isRecurring" firestore:"isRecurring"`
	RecurringDate  int     `json:"recurringDate,omitempty" firestore:"recurringDate,omitempty"`
	CreatedAt      string  `json:"createdAt,omitempty" firestore:"createdAt,omitempty"`
	UpdatedAt      string  `json:"updatedAt,omitempty" firestore:"updatedAt,omitempty"`
	ShoppingListID string  `json:"shoppingListId,omitempty" firestore:"shoppingListId,omitempty"`
}

type TransactionResponse struct {
	Transactions []Transaction `json:"transactions"`
}

type CategorySummary struct {
	CategoryID       string  `json:"categoryId"`
	Amount           float64 `json:"amount"`
	Percentage       float64 `json:"percentage"`
	TransactionCount int     `json:"transactionCount"`
}

type MonthlySummary struct {
	Month             string            `json:"month"`
	TotalExpenses     float64           `json:"totalExpenses"`
	TotalIncome       float64           `json:"totalIncome"`
	Delta             float64           `json:"delta"`
	CategorySummaries []CategorySummary `json:"categorySummaries"`
}

type TransactionSummaryResponse struct {
	Summary *MonthlySummary `json:"summary"`
}
