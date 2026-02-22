package cqrs

// ---------- User queries ----------

// GetUserQuery fetches a single user by ID, subject to ownership check.
type GetUserQuery struct {
	UserID           string
	RequestingUserID string
}

// ---------- Account queries ----------

// GetAccountQuery fetches a single account by account number.
type GetAccountQuery struct {
	AccountNumber    string
	RequestingUserID string
}

// ListAccountsQuery fetches all accounts belonging to a user.
type ListAccountsQuery struct {
	UserID string
}

// ---------- Transaction queries ----------

// GetTransactionQuery fetches a single transaction.
type GetTransactionQuery struct {
	TransactionID string
	AccountNumber string
	UserID        string
}

// ListTransactionsQuery fetches all transactions for an account.
type ListTransactionsQuery struct {
	AccountNumber string
	UserID        string
}
