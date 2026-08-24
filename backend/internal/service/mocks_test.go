package service_test

import (
	"context"

	"github.com/juanbedoya/hnl-bank/backend/internal/model"
)

type mockUserRepo struct {
	createFn    func(ctx context.Context, u *model.User) error
	findByEmail func(ctx context.Context, email string) (*model.User, error)
	findByID    func(ctx context.Context, id string) (*model.User, error)
}

func (m *mockUserRepo) Create(ctx context.Context, u *model.User) error {
	if m.createFn == nil {
		return nil
	}
	return m.createFn(ctx, u)
}

func (m *mockUserRepo) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	if m.findByEmail == nil {
		return nil, nil
	}
	return m.findByEmail(ctx, email)
}

func (m *mockUserRepo) FindByID(ctx context.Context, id string) (*model.User, error) {
	if m.findByID == nil {
		return nil, nil
	}
	return m.findByID(ctx, id)
}

const testAccountID = 1

type mockAccountRepo struct {
	createFn     func(ctx context.Context, a *model.Account) error
	findByIDFn   func(ctx context.Context, id string) (*model.Account, error)
	findByUserFn func(ctx context.Context, userID string) ([]model.Account, error)
	findByNumFn  func(ctx context.Context, number string) (*model.Account, error)
	getBalFn     func(ctx context.Context, id [16]byte) (int64, error)
}

func (m *mockAccountRepo) Create(ctx context.Context, a *model.Account) error {
	if m.createFn == nil {
		return nil
	}
	return m.createFn(ctx, a)
}

func (m *mockAccountRepo) FindByID(ctx context.Context, id string) (*model.Account, error) {
	if m.findByIDFn == nil {
		return nil, nil
	}
	return m.findByIDFn(ctx, id)
}

func (m *mockAccountRepo) FindByUserID(ctx context.Context, userID string) ([]model.Account, error) {
	if m.findByUserFn == nil {
		return nil, nil
	}
	return m.findByUserFn(ctx, userID)
}

func (m *mockAccountRepo) FindByNumber(ctx context.Context, number string) (*model.Account, error) {
	if m.findByNumFn == nil {
		return nil, nil
	}
	return m.findByNumFn(ctx, number)
}

func (m *mockAccountRepo) GetBalance(ctx context.Context, id [16]byte) (int64, error) {
	if m.getBalFn == nil {
		return 0, nil
	}
	return m.getBalFn(ctx, id)
}

type mockTxRepo struct {
	createTransferFn func(ctx context.Context, id, debitID, creditID [16]byte, amount uint64, ledger, code uint32, flags uint16) error
	recordFn         func(ctx context.Context, tx *model.Transaction, fromUserID, toUserID string) error
	historyFn        func(ctx context.Context, userID, accountNumber string, limit, offset int) ([]model.Transaction, error)
	countFn          func(ctx context.Context, userID, accountNumber string) (int, error)
}

func (m *mockTxRepo) CreateTransfer(ctx context.Context, id, debitID, creditID [16]byte, amount uint64, ledger, code uint32, flags uint16) error {
	if m.createTransferFn == nil {
		return nil
	}
	return m.createTransferFn(ctx, id, debitID, creditID, amount, ledger, code, flags)
}

func (m *mockTxRepo) Record(ctx context.Context, tx *model.Transaction, fromUserID, toUserID string) error {
	if m.recordFn == nil {
		return nil
	}
	return m.recordFn(ctx, tx, fromUserID, toUserID)
}

func (m *mockTxRepo) History(ctx context.Context, userID, accountNumber string, limit, offset int) ([]model.Transaction, error) {
	if m.historyFn == nil {
		return nil, nil
	}
	return m.historyFn(ctx, userID, accountNumber, limit, offset)
}

func (m *mockTxRepo) CountHistory(ctx context.Context, userID, accountNumber string) (int, error) {
	if m.countFn == nil {
		return 0, nil
	}
	return m.countFn(ctx, userID, accountNumber)
}
