package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/juanbedoya/hnl-bank/backend/internal/model"
	"github.com/juanbedoya/hnl-bank/backend/internal/repository"
	"github.com/juanbedoya/hnl-bank/backend/internal/service"
	"golang.org/x/crypto/bcrypt"
)

func hashPassword(t *testing.T, pass string) string {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}
	return string(h)
}

func TestRegisterDuplicateEmail(t *testing.T) {
	repo := &mockUserRepo{createFn: func(ctx context.Context, u *model.User) error {
		return repository.ErrEmailExists
	}}
	svc := service.NewAuthService(repo, "secret")
	_, err := svc.Register(context.Background(), "a@b.com", "pass", "Name")
	if !errors.Is(err, service.ErrEmailExists) {
		t.Fatalf("expected ErrEmailExists, got %v", err)
	}
}

func TestRegisterCreatesHashedPassword(t *testing.T) {
	var stored model.User
	repo := &mockUserRepo{createFn: func(ctx context.Context, u *model.User) error {
		stored = *u
		return nil
	}}
	svc := service.NewAuthService(repo, "secret")
	u, err := svc.Register(context.Background(), "a@b.com", "pass123", "Alberto")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Email != "a@b.com" {
		t.Fatalf("email mismatch: %s", u.Email)
	}
	if stored.PasswordHash == "pass123" {
		t.Fatal("password stored in plaintext")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte("pass123")); err != nil {
		t.Fatalf("hash does not match password: %v", err)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	repo := &mockUserRepo{findByEmail: func(ctx context.Context, email string) (*model.User, error) {
		return &model.User{
			ID:           "u1",
			Email:        email,
			PasswordHash: hashPassword(t, "correct"),
			FullName:     "Name",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}, nil
	}}
	svc := service.NewAuthService(repo, "secret")
	_, _, err := svc.Login(context.Background(), "a@b.com", "wrong")
	if !errors.Is(err, service.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLoginSuccessReturnsValidToken(t *testing.T) {
	repo := &mockUserRepo{findByEmail: func(ctx context.Context, email string) (*model.User, error) {
		return &model.User{ID: "u1", Email: email, PasswordHash: hashPassword(t, "secretpass")}, nil
	}}
	svc := service.NewAuthService(repo, "secret")
	u, token, err := svc.Login(context.Background(), "a@b.com", "secretpass")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
	if u.ID != "u1" {
		t.Fatalf("wrong user returned: %s", u.ID)
	}
	userID, err := svc.ValidateToken(context.Background(), token)
	if err != nil {
		t.Fatalf("token should validate: %v", err)
	}
	if userID != "u1" {
		t.Fatalf("token subject mismatch: %s", userID)
	}
}

func TestLogoutBlacklistsToken(t *testing.T) {
	repo := &mockUserRepo{findByEmail: func(ctx context.Context, email string) (*model.User, error) {
		return &model.User{ID: "u1", Email: email, PasswordHash: hashPassword(t, "secretpass")}, nil
	}}
	svc := service.NewAuthService(repo, "secret")
	_, token, err := svc.Login(context.Background(), "a@b.com", "secretpass")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if err := svc.Logout(context.Background(), token); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if _, err := svc.ValidateToken(context.Background(), token); !errors.Is(err, service.ErrInvalidToken) {
		t.Fatalf("expected token invalidated, got %v", err)
	}
}
