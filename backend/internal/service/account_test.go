package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/juanbedoya/hnl-bank/backend/internal/model"
	"github.com/juanbedoya/hnl-bank/backend/internal/service"
)

func TestGetAccountsEmpty(t *testing.T) {
	repo := &mockAccountRepo{findByUserFn: func(ctx context.Context, userID string) ([]model.Account, error) {
		return nil, nil
	}}
	svc := service.NewAccountService(repo)
	accs, err := svc.GetAccounts(context.Background(), "u1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(accs) != 0 {
		t.Fatalf("expected 0 accounts, got %d", len(accs))
	}
}

func TestGetAccountsWithBalances(t *testing.T) {
	acct := model.Account{ID: "a1", UserID: "u1", AccountNumber: "4001-1", AccountType: "savings", Currency: "USD"}
	repo := &mockAccountRepo{
		findByUserFn: func(ctx context.Context, userID string) ([]model.Account, error) {
			return []model.Account{acct}, nil
		},
		getBalFn: func(ctx context.Context, id [16]byte) (int64, error) {
			return 123456, nil
		},
	}
	svc := service.NewAccountService(repo)
	accs, err := svc.GetAccounts(context.Background(), "u1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(accs) != 1 {
		t.Fatalf("expected 1 account, got %d", len(accs))
	}
	if accs[0].Balance != 123456 {
		t.Fatalf("balance mismatch: %d", accs[0].Balance)
	}
}

func TestGetBalanceUnauthorized(t *testing.T) {
	repo := &mockAccountRepo{findByIDFn: func(ctx context.Context, id string) (*model.Account, error) {
		return &model.Account{ID: id, UserID: "owner"}, nil
	}}
	svc := service.NewAccountService(repo)
	_, _, err := svc.GetBalance(context.Background(), "intruder", "a1")
	if !errors.Is(err, service.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}
