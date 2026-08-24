package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/juanbedoya/hnl-bank/backend/internal/model"
)

// TransactionRepository performs TigerBeetle transfers and records rich history
// in PostgreSQL for querying/pagination.
type TransactionRepository interface {
	CreateTransfer(ctx context.Context, id, debitID, creditID [16]byte, amount uint64, ledger, code uint32, flags uint16) error
	Record(ctx context.Context, tx *model.Transaction, fromUserID, toUserID string) error
	History(ctx context.Context, userID, accountNumber string, limit, offset int) ([]model.Transaction, error)
	CountHistory(ctx context.Context, userID, accountNumber string) (int, error)
}

type txRepo struct {
	db *sql.DB
	tb TigerBeetleRepository
}

// NewTransactionRepository builds a TransactionRepository over PG and TB.
func NewTransactionRepository(db *sql.DB, tb TigerBeetleRepository) TransactionRepository {
	return &txRepo{db: db, tb: tb}
}

func (r *txRepo) CreateTransfer(ctx context.Context, id, debitID, creditID [16]byte, amount uint64, ledger, code uint32, flags uint16) error {
	return r.tb.CreateTransfer(ctx, id, debitID, creditID, amount, ledger, code, flags)
}

func (r *txRepo) Record(ctx context.Context, tx *model.Transaction, fromUserID, toUserID string) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO transactions
		(id, from_user_id, to_user_id, from_account, to_account, amount, type, description, timestamp, status)
		VALUES ($1, NULLIF($2, '')::uuid, NULLIF($3, '')::uuid, $4, $5, $6, $7, $8, $9, $10)`,
		tx.ID, nullIfEmpty(fromUserID), nullIfEmpty(toUserID), tx.FromAccount, tx.ToAccount,
		tx.Amount, tx.Type, tx.Description, tx.Timestamp, tx.Status)
	return err
}

func (r *txRepo) History(ctx context.Context, userID, accountNumber string, limit, offset int) ([]model.Transaction, error) {
	query := `SELECT id, from_user_id, to_user_id, from_account, to_account, amount, type, description, timestamp, status
		FROM transactions
		WHERE (from_user_id = $1 OR to_user_id = $1)`
	args := []any{userID}
	if accountNumber != "" {
		query += ` AND (from_account = $2 OR to_account = $2)`
		args = append(args, accountNumber)
	}
	query += ` ORDER BY timestamp DESC LIMIT $3 OFFSET $4`
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Transaction
	for rows.Next() {
		var t model.Transaction
		var fromUID, toUID sql.NullString
		if err := rows.Scan(&t.ID, &fromUID, &toUID, &t.FromAccount, &t.ToAccount,
			&t.Amount, &t.Type, &t.Description, &t.Timestamp, &t.Status); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *txRepo) CountHistory(ctx context.Context, userID, accountNumber string) (int, error) {
	query := `SELECT COUNT(*) FROM transactions WHERE (from_user_id = $1 OR to_user_id = $1)`
	args := []any{userID}
	if accountNumber != "" {
		query += ` AND (from_account = $2 OR to_account = $2)`
		args = append(args, accountNumber)
	}
	var count int
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// ensure time import used (timestamp scanning)
var _ = time.Time{}
