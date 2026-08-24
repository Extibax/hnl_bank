package model

import "time"

// Account represents a financial account. The authoritative balance lives in
// TigerBeetle (identified by TigerBeetleID); the rich metadata lives in PG.
type Account struct {
	ID            string    `json:"id"`
	UserID        string    `json:"user_id"`
	AccountNumber string    `json:"account_number"`
	TigerBeetleID [16]byte  `json:"-"`
	AccountType   string    `json:"account_type"`
	Currency      string    `json:"currency"`
	CreatedAt     time.Time `json:"created_at"`
}

// AccountWithBalance carries an account plus its current balance in cents.
type AccountWithBalance struct {
	Account
	Balance int64 `json:"-"`
}

// AccountBalance is the API view of a single balance query.
type AccountBalance struct {
	Balance  string `json:"balance"`
	Currency string `json:"currency"`
}

// AccountResponse is the API view of an account with a formatted balance.
type AccountResponse struct {
	ID            string    `json:"id"`
	AccountNumber string    `json:"account_number"`
	AccountType   string    `json:"account_type"`
	Currency      string    `json:"currency"`
	Balance       string    `json:"balance"`
	CreatedAt     time.Time `json:"created_at"`
}
