package service

import (
	"context"

	"github.com/juanbedoya/hnl-bank/backend/internal/model"
	"github.com/juanbedoya/hnl-bank/backend/internal/repository"
)

// AccountService exposes account listing and balance queries.
type AccountService interface {
	GetAccounts(ctx context.Context, userID string) ([]model.AccountWithBalance, error)
	GetBalance(ctx context.Context, userID, accountID string) (int64, string, error)
}

type accountService struct {
	accounts repository.AccountRepository
}

// NewAccountService builds an AccountService.
func NewAccountService(accounts repository.AccountRepository) AccountService {
	return &accountService{accounts: accounts}
}

func (s *accountService) GetAccounts(ctx context.Context, userID string) ([]model.AccountWithBalance, error) {
	accs, err := s.accounts.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]model.AccountWithBalance, 0, len(accs))
	for _, a := range accs {
		bal, err := s.accounts.GetBalance(ctx, a.TigerBeetleID)
		if err != nil {
			return nil, err
		}
		out = append(out, model.AccountWithBalance{Account: a, Balance: bal})
	}
	return out, nil
}

func (s *accountService) GetBalance(ctx context.Context, userID, accountID string) (int64, string, error) {
	a, err := s.accounts.FindByID(ctx, accountID)
	if err != nil {
		return 0, "", ErrAccountNotFound
	}
	if a.UserID != userID {
		return 0, "", ErrUnauthorized
	}
	bal, err := s.accounts.GetBalance(ctx, a.TigerBeetleID)
	if err != nil {
		return 0, "", err
	}
	return bal, a.Currency, nil
}
