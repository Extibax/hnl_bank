package service

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"github.com/juanbedoya/hnl-bank/backend/internal/id"
	"github.com/juanbedoya/hnl-bank/backend/internal/model"
	"github.com/juanbedoya/hnl-bank/backend/internal/repository"
)

const transferFlagsNone uint16 = 0

// TransactionService handles deposits, withdrawals, transfers and history.
type TransactionService interface {
	Deposit(ctx context.Context, userID, accountID string, amount int64) error
	Withdraw(ctx context.Context, userID, accountID string, amount int64) error
	Transfer(ctx context.Context, userID, fromAccountID, toAccount string, amount int64) error
	History(ctx context.Context, userID, accountID string, limit, offset int) ([]model.Transaction, int, error)
}

type transactionService struct {
	accounts repository.AccountRepository
	repo     repository.TransactionRepository
	seq      uint64
}

// NewTransactionService builds a TransactionService.
func NewTransactionService(accounts repository.AccountRepository, repo repository.TransactionRepository) TransactionService {
	return &transactionService{accounts: accounts, repo: repo}
}

func (s *transactionService) Deposit(ctx context.Context, userID, accountID string, amount int64) error {
	if amount <= 0 {
		return ErrNegativeAmount
	}
	a, err := s.accounts.FindByID(ctx, accountID)
	if err != nil {
		return ErrAccountNotFound
	}
	if a.UserID != userID {
		return ErrUnauthorized
	}
	external := repository.ExternalID
	transferID := repository.NewTransferID(atomic.AddUint64(&s.seq, 1))
	if err := s.repo.CreateTransfer(ctx, transferID, external, a.TigerBeetleID, uint64(amount), 1, 1, transferFlagsNone); err != nil {
		return err
	}
	tx := &model.Transaction{
		ID:          id.New(),
		FromAccount: "EXTERNAL",
		ToAccount:   a.AccountNumber,
		Amount:      amount,
		Type:        model.TypeDeposit,
		Description: "Deposito",
		Timestamp:   time.Now().UTC(),
		Status:      model.StatusCompleted,
	}
	return s.repo.Record(ctx, tx, a.UserID, userID)
}

func (s *transactionService) Withdraw(ctx context.Context, userID, accountID string, amount int64) error {
	if amount <= 0 {
		return ErrNegativeAmount
	}
	a, err := s.accounts.FindByID(ctx, accountID)
	if err != nil {
		return ErrAccountNotFound
	}
	if a.UserID != userID {
		return ErrUnauthorized
	}
	if bal, err := s.accounts.GetBalance(ctx, a.TigerBeetleID); err == nil && bal < amount {
		return ErrInsufficientFunds
	}
	external := repository.ExternalID
	transferID := repository.NewTransferID(atomic.AddUint64(&s.seq, 1))
	if err := s.repo.CreateTransfer(ctx, transferID, a.TigerBeetleID, external, uint64(amount), 1, 1, transferFlagsNone); err != nil {
		if errors.Is(err, repository.ErrInsufficientFunds) {
			return ErrInsufficientFunds
		}
		return err
	}
	tx := &model.Transaction{
		ID:          id.New(),
		FromAccount: a.AccountNumber,
		ToAccount:   "EXTERNAL",
		Amount:      amount,
		Type:        model.TypeWithdraw,
		Description: "Retiro",
		Timestamp:   time.Now().UTC(),
		Status:      model.StatusCompleted,
	}
	return s.repo.Record(ctx, tx, userID, userID)
}

func (s *transactionService) Transfer(ctx context.Context, userID, fromAccountID, toAccount string, amount int64) error {
	if amount <= 0 {
		return ErrNegativeAmount
	}
	from, err := s.accounts.FindByID(ctx, fromAccountID)
	if err != nil {
		return ErrAccountNotFound
	}
	if from.UserID != userID {
		return ErrUnauthorized
	}
	dst, err := s.accounts.FindByNumber(ctx, toAccount)
	if err != nil {
		return ErrDestinationNotFound
	}
	if from.AccountNumber == dst.AccountNumber {
		return ErrSameAccount
	}
	if bal, err := s.accounts.GetBalance(ctx, from.TigerBeetleID); err == nil && bal < amount {
		return ErrInsufficientFunds
	}
	transferID := repository.NewTransferID(atomic.AddUint64(&s.seq, 1))
	if err := s.repo.CreateTransfer(ctx, transferID, from.TigerBeetleID, dst.TigerBeetleID, uint64(amount), 1, 1, transferFlagsNone); err != nil {
		if errors.Is(err, repository.ErrInsufficientFunds) {
			return ErrInsufficientFunds
		}
		return err
	}
	tx := &model.Transaction{
		ID:          id.New(),
		FromAccount: from.AccountNumber,
		ToAccount:   dst.AccountNumber,
		Amount:      amount,
		Type:        model.TypeTransfer,
		Description: "Transferencia",
		Timestamp:   time.Now().UTC(),
		Status:      model.StatusCompleted,
	}
	return s.repo.Record(ctx, tx, userID, dst.UserID)
}

func (s *transactionService) History(ctx context.Context, userID, accountID string, limit, offset int) ([]model.Transaction, int, error) {
	var accountNumber string
	if accountID != "" {
		acct, err := s.accounts.FindByID(ctx, accountID)
		if err != nil {
			acct, err = s.accounts.FindByNumber(ctx, accountID)
			if err != nil {
				return nil, 0, ErrAccountNotFound
			}
		}
		if acct.UserID != userID {
			return nil, 0, ErrUnauthorized
		}
		accountNumber = acct.AccountNumber
	}
	txs, err := s.repo.History(ctx, userID, accountNumber, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	count, err := s.repo.CountHistory(ctx, userID, accountNumber)
	if err != nil {
		return nil, 0, err
	}
	return txs, count, nil
}