package repository

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	tb "github.com/tigerbeetle/tigerbeetle-go"
)

var (
	// ErrInsufficientFunds is returned when a debit exceeds the account balance.
	ErrInsufficientFunds = errors.New("insufficient funds")
	// ErrAccountNotFound is returned when a TigerBeetle account cannot be located.
	ErrAccountNotFound = errors.New("account not found")
)

// TigerBeetleRepository wraps the TigerBeetle client for accounts and transfers.
type TigerBeetleRepository interface {
	CreateAccount(ctx context.Context, id [16]byte, ledger, code uint32, flags uint16) error
	CreateTransfer(ctx context.Context, id, debitID, creditID [16]byte, amount uint64, ledger, code uint32, flags uint16) error
	Balance(ctx context.Context, id [16]byte) (int64, error)
	EnsureExternalAccount(ctx context.Context) error
	Close()
}

type tbRepo struct {
	client tb.Client
}

// NewTigerBeetleClient connects to a TigerBeetle cluster at the given address.
func NewTigerBeetleClient(address string) (tb.Client, error) {
	clusterID := tb.Uint128{}
	client, err := tb.NewClient(clusterID, []string{address})
	if err != nil {
		return nil, fmt.Errorf("new tigerbeetle client: %w", err)
	}
	return client, nil
}

// NewTigerBeetleRepository wraps an existing client as a repository.
func NewTigerBeetleRepository(client tb.Client) TigerBeetleRepository {
	return &tbRepo{client: client}
}

// ExternalID is the id of the dedicated external counterparty account.
var ExternalID = func() [16]byte { return tb.ToUint128(1 << 40).Bytes() }()

// NewAccountID returns a random, non-zero 16-byte TigerBeetle account id.
func NewAccountID() [16]byte {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	if b == [16]byte{} {
		b[0] = 1
	}
	return b
}

// Uint128FromUint64 builds a 16-byte id from a sequential counter.
func Uint128FromUint64(n uint64) [16]byte {
	return tb.ToUint128(n).Bytes()
}

// NewTransferID returns a unique, time-sorted 16-byte transfer id.
func NewTransferID(seq uint64) [16]byte {
	// Reuse the TB monotonic generator for guaranteed uniqueness.
	_ = seq
	return tb.ID().Bytes()
}

func toUint128(b [16]byte) tb.Uint128 {
	return tb.BytesToUint128(b)
}

func (r *tbRepo) CreateAccount(ctx context.Context, id [16]byte, ledger, code uint32, flags uint16) error {
	accts := []tb.Account{{
		ID:     toUint128(id),
		Ledger: ledger,
		Code:   uint16(code),
		Flags:  flags,
	}}
	results, err := r.client.CreateAccounts(accts)
	if err != nil {
		return fmt.Errorf("create account: %w", err)
	}
	if len(results) > 0 {
		st := results[0].Status
		if st != tb.AccountCreated && st != tb.AccountExists {
			return fmt.Errorf("create account status: %s", st.String())
		}
	}
	return nil
}

func (r *tbRepo) CreateTransfer(ctx context.Context, id, debitID, creditID [16]byte, amount uint64, ledger, code uint32, flags uint16) error {
	trs := []tb.Transfer{{
		ID:              toUint128(id),
		DebitAccountID:  toUint128(debitID),
		CreditAccountID: toUint128(creditID),
		Amount:          tb.ToUint128(amount),
		Ledger:          ledger,
		Code:            uint16(code),
		Flags:           flags,
	}}
	results, err := r.client.CreateTransfers(trs)
	if err != nil {
		return fmt.Errorf("create transfer: %w", err)
	}
	if len(results) > 0 {
		switch results[0].Status {
		case tb.TransferCreated:
			return nil
		case tb.TransferExceedsCredits:
			return ErrInsufficientFunds
		default:
			return fmt.Errorf("create transfer status: %s", results[0].Status.String())
		}
	}
	return nil
}

func (r *tbRepo) Balance(ctx context.Context, id [16]byte) (int64, error) {
	accounts, err := r.client.LookupAccounts([]tb.Uint128{toUint128(id)})
	if err != nil {
		return 0, err
	}
	if len(accounts) == 0 {
		return 0, ErrAccountNotFound
	}
	acct := accounts[0]
	creditsLo, _ := acct.CreditsPosted.Uint64()
	debitsLo, _ := acct.DebitsPosted.Uint64()
	cpLo, _ := acct.CreditsPending.Uint64()
	dpLo, _ := acct.DebitsPending.Uint64()
	balance := int64(creditsLo) - int64(debitsLo) + int64(cpLo) - int64(dpLo)
	if balance < 0 {
		balance = 0
	}
	return balance, nil
}

func (r *tbRepo) EnsureExternalAccount(ctx context.Context) error {
	// The external account has no constraints so it can absorb arbitrary
	// inbound and outbound flows and go negative.
	return r.CreateAccount(ctx, ExternalID, 1, 1, 0)
}

func (r *tbRepo) Close() {
	r.client.Close()
}

var _ = time.Now