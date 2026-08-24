package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/juanbedoya/hnl-bank/backend/internal/model"
	"github.com/juanbedoya/hnl-bank/backend/internal/service"
)

func account(id, userID, number string) *model.Account {
	return &model.Account{ID: id, UserID: userID, AccountNumber: number, AccountType: "savings", Currency: "USD"}
}

func TestDepositNegativeAmount(t *testing.T) {
	acctRepo := &mockAccountRepo{}
	txRepo := &mockTxRepo{}
	svc := service.NewTransactionService(acctRepo, txRepo)
	if err := svc.Deposit(context.Background(), "u1", "a1", -5); !errors.Is(err, service.ErrNegativeAmount) {
		t.Fatalf("expected ErrNegativeAmount, got %v", err)
	}
}

func TestWithdrawInsufficientFunds(t *testing.T) {
	acctRepo := &mockAccountRepo{
		findByIDFn: func(ctx context.Context, id string) (*model.Account, error) {
			return account(id, "u1", "4001-1"), nil
		},
		getBalFn: func(ctx context.Context, id [16]byte) (int64, error) {
			return 50, nil
		},
	}
	txRepo := &mockTxRepo{}
	svc := service.NewTransactionService(acctRepo, txRepo)
	if err := svc.Withdraw(context.Background(), "u1", "a1", 100); !errors.Is(err, service.ErrInsufficientFunds) {
		t.Fatalf("expected ErrInsufficientFunds, got %v", err)
	}
}

func TestWithdrawSuccess(t *testing.T) {
	var transferCalled bool
	acctRepo := &mockAccountRepo{
		findByIDFn: func(ctx context.Context, id string) (*model.Account, error) {
			return account(id, "u1", "4001-1"), nil
		},
		getBalFn: func(ctx context.Context, id [16]byte) (int64, error) {
			return 1000, nil
		},
	}
	txRepo := &mockTxRepo{createTransferFn: func(ctx context.Context, id, debitID, creditID [16]byte, amount uint64, ledger, code uint32, flags uint16) error {
		transferCalled = true
		return nil
	}}
	svc := service.NewTransactionService(acctRepo, txRepo)
	if err := svc.Withdraw(context.Background(), "u1", "a1", 100); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !transferCalled {
		t.Fatal("expected transfer to be created")
	}
}

func TestTransferToNonexistentDestination(t *testing.T) {
	acctRepo := &mockAccountRepo{
		findByIDFn: func(ctx context.Context, id string) (*model.Account, error) {
			return account(id, "u1", "4001-1"), nil
		},
		findByNumFn: func(ctx context.Context, number string) (*model.Account, error) {
			return nil, errors.New("not found")
		},
	}
	txRepo := &mockTxRepo{}
	svc := service.NewTransactionService(acctRepo, txRepo)
	if err := svc.Transfer(context.Background(), "u1", "a1", "4001-9999", 50); !errors.Is(err, service.ErrDestinationNotFound) {
		t.Fatalf("expected ErrDestinationNotFound, got %v", err)
	}
}

func TestTransferToSameAccount(t *testing.T) {
	acctRepo := &mockAccountRepo{
		findByIDFn: func(ctx context.Context, id string) (*model.Account, error) {
			return account(id, "u1", "4001-1"), nil
		},
		findByNumFn: func(ctx context.Context, number string) (*model.Account, error) {
			return account("a1", "u1", "4001-1"), nil
		},
	}
	txRepo := &mockTxRepo{}
	svc := service.NewTransactionService(acctRepo, txRepo)
	if err := svc.Transfer(context.Background(), "u1", "a1", "4001-1", 50); !errors.Is(err, service.ErrSameAccount) {
		t.Fatalf("expected ErrSameAccount, got %v", err)
	}
}

func TestTransferSuccess(t *testing.T) {
	acctRepo := &mockAccountRepo{
		findByIDFn: func(ctx context.Context, id string) (*model.Account, error) {
			return account(id, "u1", "4001-1"), nil
		},
		findByNumFn: func(ctx context.Context, number string) (*model.Account, error) {
			return account("a2", "u2", "4001-2"), nil
		},
		getBalFn: func(ctx context.Context, id [16]byte) (int64, error) {
			return 500, nil
		},
	}
	var recordCalled bool
	txRepo := &mockTxRepo{
		createTransferFn: func(ctx context.Context, id, debitID, creditID [16]byte, amount uint64, ledger, code uint32, flags uint16) error {
			return nil
		},
		recordFn: func(ctx context.Context, tx *model.Transaction, fromUserID, toUserID string) error {
			recordCalled = true
			return nil
		},
	}
	svc := service.NewTransactionService(acctRepo, txRepo)
	if err := svc.Transfer(context.Background(), "u1", "a1", "4001-2", 50); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !recordCalled {
		t.Fatal("expected transaction to be recorded")
	}
}
