package repository

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

// NewPostgresDB opens a connection pool and applies schema migrations.
func NewPostgresDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	if err := migrate(db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return db, nil
}

const usersSchema = `CREATE TABLE IF NOT EXISTS users (
	id UUID PRIMARY KEY,
	email TEXT UNIQUE NOT NULL,
	password_hash TEXT NOT NULL,
	full_name TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);`

const userAccountsSchema = `CREATE TABLE IF NOT EXISTS user_accounts (
	id UUID PRIMARY KEY,
	user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	account_number TEXT UNIQUE NOT NULL,
	tigerbeetle_id BYTEA NOT NULL,
	account_type TEXT NOT NULL,
	currency TEXT NOT NULL DEFAULT 'USD',
	created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);`

const transactionsSchema = `CREATE TABLE IF NOT EXISTS transactions (
	id UUID PRIMARY KEY,
	from_user_id UUID,
	to_user_id UUID,
	from_account TEXT,
	to_account TEXT,
	amount BIGINT NOT NULL,
	type TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	timestamp TIMESTAMPTZ NOT NULL DEFAULT now(),
	status TEXT NOT NULL DEFAULT 'completed'
);
CREATE INDEX IF NOT EXISTS idx_transactions_from_user ON transactions(from_user_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_transactions_to_user ON transactions(to_user_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_transactions_from_acct ON transactions(from_account);
CREATE INDEX IF NOT EXISTS idx_transactions_to_acct ON transactions(to_account);`

func migrate(db *sql.DB) error {
	for _, schema := range []string{usersSchema, userAccountsSchema, transactionsSchema} {
		if _, err := db.Exec(schema); err != nil {
			return err
		}
	}
	return nil
}

// isUniqueViolation reports whether err is a PostgreSQL unique constraint error.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	type pqError interface {
		SQLState() string
	}
	if pe, ok := err.(pqError); ok {
		return pe.SQLState() == "23505"
	}
	return false
}
