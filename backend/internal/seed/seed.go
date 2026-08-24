package seed

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/juanbedoya/hnl-bank/backend/internal/id"
	"github.com/juanbedoya/hnl-bank/backend/internal/model"
	"github.com/juanbedoya/hnl-bank/backend/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

//go:embed data/seed.json
var seedData []byte

type seedUser struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Password  string    `json:"password"`
	FullName  string    `json:"full_name"`
	CreatedAt time.Time `json:"created_at"`
}

type seedAccount struct {
	AccountNumber  string  `json:"account_number"`
	UserID         string  `json:"user_id"`
	InitialBalance float64 `json:"initial_balance"`
	Currency       string  `json:"currency"`
	AccountType    string  `json:"account_type"`
}

type seedTransaction struct {
	FromAccount string    `json:"from_account"`
	ToAccount   string    `json:"to_account"`
	Amount      float64   `json:"amount"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Timestamp   time.Time `json:"timestamp"`
	Status      string    `json:"status"`
}

type seedDoc struct {
	Users        []seedUser        `json:"users"`
	Accounts     []seedAccount     `json:"accounts"`
	Transactions []seedTransaction `json:"transactions"`
}

// Run loads the demo JSON into PostgreSQL and TigerBeetle. It is a no-op if the
// users table already has rows, so a second start never re-seeds.
func Run(ctx context.Context, db *sql.DB, accounts repository.AccountRepository, txRepo repository.TransactionRepository, tb repository.TigerBeetleRepository) error {
	var userCount int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&userCount); err != nil {
		return err
	}
	if userCount > 0 {
		return nil
	}

	var doc seedDoc
	if err := json.Unmarshal(seedData, &doc); err != nil {
		return fmt.Errorf("parse seed data: %w", err)
	}

	// Insert users. The dataset contains duplicate emails, so we append a unique
	// suffix to keep all rows while preserving the email unique constraint.
	usedEmails := make(map[string]bool)
	for _, su := range doc.Users {
		hash, err := bcrypt.GenerateFromPassword([]byte(su.Password), bcrypt.MinCost)
		if err != nil {
			return fmt.Errorf("hash password for %s: %w", su.Email, err)
		}
		email := uniqueEmail(su.Email, usedEmails)
		if _, err := db.ExecContext(ctx, `INSERT INTO users (id, email, password_hash, full_name, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6)`, su.ID, email, string(hash), su.FullName, su.CreatedAt, su.CreatedAt); err != nil {
			return fmt.Errorf("insert user %s: %w", su.Email, err)
		}
	}

	// Insert accounts in PG + TigerBeetle, and seed each initial balance from the
	// external account.
	acctToUser := make(map[string]string)
	var seq uint64
	for _, sa := range doc.Accounts {
		seq++
		tbID := repository.Uint128FromUint64(seq)
		acct := &model.Account{
			ID:            id.New(),
			UserID:        sa.UserID,
			AccountNumber: sa.AccountNumber,
			TigerBeetleID: tbID,
			AccountType:   sa.AccountType,
			Currency:      sa.Currency,
			CreatedAt:     time.Now().UTC(),
		}
		if err := accounts.Create(ctx, acct); err != nil {
			return fmt.Errorf("create account %s: %w", sa.AccountNumber, err)
		}
		acctToUser[sa.AccountNumber] = sa.UserID

		var external [16]byte
		external = repository.ExternalID
		initialCents := centsFromFloat(sa.InitialBalance)
		if err := txRepo.CreateTransfer(ctx, repository.NewTransferID(seq), external, tbID, uint64(initialCents), 1, 1, 0); err != nil {
			return fmt.Errorf("seed initial balance for %s: %w", sa.AccountNumber, err)
		}
	}

	// Seed the historical transactions into PostgreSQL for history/querying.
	for _, st := range doc.Transactions {
		fromUser := acctToUser[st.FromAccount]
		toUser := acctToUser[st.ToAccount]
		tx := &model.Transaction{
			ID:          id.New(),
			FromAccount: st.FromAccount,
			ToAccount:   st.ToAccount,
			Amount:      centsFromFloat(st.Amount),
			Type:        st.Type,
			Description: st.Description,
			Timestamp:   st.Timestamp,
			Status:      st.Status,
		}
		if err := txRepo.Record(ctx, tx, fromUser, toUser); err != nil {
			return fmt.Errorf("seed transaction: %w", err)
		}
	}

	logSeed(doc.Users, doc.Accounts, doc.Transactions)
	return nil
}

func centsFromFloat(f float64) int64 {
	return int64(math.Round(f * 100))
}

func logSeed(users []seedUser, accounts []seedAccount, transactions []seedTransaction) {
	fmt.Printf("seeded %d users, %d accounts, %d transactions\n", len(users), len(accounts), len(transactions))
}

// uniqueEmail returns base if unused, otherwise appends a +N suffix.
func uniqueEmail(base string, used map[string]bool) string {
	if !used[base] {
		used[base] = true
		return base
	}
	for i := 2; ; i++ {
		cand := suffixEmail(base, i)
		if !used[cand] {
			used[cand] = true
			return cand
		}
	}
}

func suffixEmail(email string, n int) string {
	at := strings.Index(email, "@")
	if at < 0 {
		return fmt.Sprintf("%s+%d", email, n)
	}
	return email[:at] + "+" + strconv.Itoa(n) + email[at:]
}