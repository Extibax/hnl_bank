package service

import "github.com/juanbedoya/hnl-bank/backend/internal/money"

func moneyFromCents(v int64) string {
	return money.FromCents(v)
}

func moneyToCents(s string) (int64, error) {
	return money.ToCents(s)
}
