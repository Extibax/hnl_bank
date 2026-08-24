package model

import "time"

// TransactionType constants.
const (
	TypeDeposit  = "deposit"
	TypeWithdraw = "withdraw"
	TypeTransfer = "transfer"
)

// Status constants.
const (
	StatusCompleted = "completed"
)

// Transaction represents a historical movement. Amount is stored in integer
// cents.
type Transaction struct {
	ID          string    `json:"id"`
	FromAccount string    `json:"from_account"`
	ToAccount   string    `json:"to_account"`
	Amount      int64     `json:"-"`
	AmountStr   string    `json:"amount"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Timestamp   time.Time `json:"timestamp"`
	Status      string    `json:"status"`
}
