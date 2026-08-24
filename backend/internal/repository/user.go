package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/juanbedoya/hnl-bank/backend/internal/model"
)

// ErrEmailExists is returned when creating a user with a duplicate email.
var ErrEmailExists = errors.New("email already exists")

var errUserNotFound = errors.New("user not found")

// ErrUserNotFound is returned when a user cannot be located.
var ErrUserNotFound = errUserNotFound

// UserRepository persists users in PostgreSQL.
type UserRepository interface {
	Create(ctx context.Context, u *model.User) error
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	FindByID(ctx context.Context, id string) (*model.User, error)
}

type userRepo struct {
	db *sql.DB
}

// NewUserRepository returns a UserRepository backed by the given database.
func NewUserRepository(db *sql.DB) UserRepository {
	return &userRepo{db: db}
}

func (r *userRepo) Create(ctx context.Context, u *model.User) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO users (id, email, password_hash, full_name, created_at, updated_at)
	VALUES ($1, $2, $3, $4, $5, $6)`,
		u.ID, u.Email, u.PasswordHash, u.FullName, u.CreatedAt, u.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrEmailExists
		}
		return err
	}
	return nil
}

func (r *userRepo) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	return r.scanOne(ctx, `SELECT id, email, password_hash, full_name, created_at, updated_at
	FROM users WHERE email = $1`, email)
}

func (r *userRepo) FindByID(ctx context.Context, id string) (*model.User, error) {
	return r.scanOne(ctx, `SELECT id, email, password_hash, full_name, created_at, updated_at
	FROM users WHERE id = $1`, id)
}

func (r *userRepo) scanOne(ctx context.Context, query string, arg any) (*model.User, error) {
	var u model.User
	err := r.db.QueryRowContext(ctx, query, arg).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.FullName, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}
