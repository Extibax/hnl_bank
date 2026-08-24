package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/juanbedoya/hnl-bank/backend/internal/model"
)

// AccountRepository persists account metadata in PostgreSQL and the
// authoritative balance in TigerBeetle.
type AccountRepository interface {
	Create(ctx context.Context, a *model.Account) error
	FindByID(ctx context.Context, id string) (*model.Account, error)
	FindByUserID(ctx context.Context, userID string) ([]model.Account, error)
	FindByNumber(ctx context.Context, number string) (*model.Account, error)
	GetBalance(ctx context.Context, tigerBeetleID [16]byte) (int64, error)
}

type accountRepo struct {
	db *sql.DB
	tb TigerBeetleRepository
}

// NewAccountRepository builds an AccountRepository bridging PG and TigerBeetle.
func NewAccountRepository(db *sql.DB, tb TigerBeetleRepository) AccountRepository {
	return &accountRepo{db: db, tb: tb}
}

func (r *accountRepo) Create(ctx context.Context, a *model.Account) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO user_accounts
		(id, user_id, account_number, tigerbeetle_id, account_type, currency, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		a.ID, a.UserID, a.AccountNumber, a.TigerBeetleID[:], a.AccountType, a.Currency, a.CreatedAt)
	if err != nil {
		return err
	}
	// Create the counterpart account in TigerBeetle. If it fails, roll back the
	// PG row to keep the two databases consistent.
	acctFlags := uint16(typesAccountFlagsDebitsMustNotExceedCredits)
	if err := r.tb.CreateAccount(ctx, a.TigerBeetleID, 1, 1, acctFlags); err != nil {
		_, _ = r.db.ExecContext(ctx, `DELETE FROM user_accounts WHERE id = $1`, a.ID)
		return err
	}
	return nil
}

func (r *accountRepo) FindByID(ctx context.Context, id string) (*model.Account, error) {
	return r.scanOne(ctx, `SELECT id, user_id, account_number, tigerbeetle_id, account_type, currency, created_at
		FROM user_accounts WHERE id = $1`, id)
}

func (r *accountRepo) FindByNumber(ctx context.Context, number string) (*model.Account, error) {
	return r.scanOne(ctx, `SELECT id, user_id, account_number, tigerbeetle_id, account_type, currency, created_at
		FROM user_accounts WHERE account_number = $1`, number)
}

func (r *accountRepo) FindByUserID(ctx context.Context, userID string) ([]model.Account, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, user_id, account_number, tigerbeetle_id, account_type, currency, created_at
		FROM user_accounts WHERE user_id = $1 ORDER BY created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Account
	for rows.Next() {
		var a model.Account
		if err := scanAccount(rows, &a); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *accountRepo) GetBalance(ctx context.Context, tigerBeetleID [16]byte) (int64, error) {
	return r.tb.Balance(ctx, tigerBeetleID)
}

func (r *accountRepo) scanOne(ctx context.Context, query string, arg any) (*model.Account, error) {
	var a model.Account
	err := scanAccount(r.db.QueryRowContext(ctx, query, arg), &a)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrAccountNotFound
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func scanAccount(row interface{ Scan(...any) error }, a *model.Account) error {
	var tbID []byte
	if err := row.Scan(&a.ID, &a.UserID, &a.AccountNumber, &tbID, &a.AccountType, &a.Currency, &a.CreatedAt); err != nil {
		return err
	}
	copy(a.TigerBeetleID[:], tbID)
	return nil
}

// typesAccountFlagsDebitsMustNotExceedCredits mirrors the TigerBeetle account
// flag (1 << 1) that prevents debiting beyond credits.
const typesAccountFlagsDebitsMustNotExceedCredits uint16 = 1 << 1