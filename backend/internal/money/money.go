package money

import (
	"fmt"
	"strconv"
	"strings"
)

// ToCents converts a decimal amount string such as "1234.56" into integer
// cents. It tolerates an optional sign and more than two decimals are truncated.
func ToCents(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("amount is empty")
	}
	neg := false
	if strings.HasPrefix(s, "-") {
		neg = true
		s = s[1:]
	} else if strings.HasPrefix(s, "+") {
		s = s[1:]
	}
	parts := strings.SplitN(s, ".", 2)
	intStr := parts[0]
	fracStr := "00"
	if len(parts) == 2 {
		fracStr = parts[1]
		if len(fracStr) > 2 {
			fracStr = fracStr[:2]
		}
		for len(fracStr) < 2 {
			fracStr += "0"
		}
	}
	if intStr == "" {
		intStr = "0"
	}
	intVal, err := strconv.ParseInt(intStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid integer part: %w", err)
	}
	fracVal, err := strconv.ParseInt(fracStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid fraction part: %w", err)
	}
	cents := intVal*100 + fracVal
	if neg {
		cents = -cents
	}
	return cents, nil
}

// FromCents renders integer cents as a decimal string such as "1234.56".
func FromCents(v int64) string {
	if v < 0 {
		return "-" + FromCents(-v)
	}
	return fmt.Sprintf("%d.%02d", v/100, v%100)
}
