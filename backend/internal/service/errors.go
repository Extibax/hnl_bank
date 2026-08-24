package service

import "errors"

var (
	// ErrAccountNotFound indicates the requested account does not exist.
	ErrAccountNotFound = errors.New("account not found")
	// ErrInsufficientFunds indicates an operation would overdraw an account.
	ErrInsufficientFunds = errors.New("insufficient funds")
	// ErrNegativeAmount indicates an amount that is not greater than zero.
	ErrNegativeAmount = errors.New("amount must be greater than zero")
	// ErrSameAccount indicates a transfer from and to the same account.
	ErrSameAccount = errors.New("cannot transfer to the same account")
	// ErrDestinationNotFound indicates the transfer destination does not exist.
	ErrDestinationNotFound = errors.New("destination account not found")
)
