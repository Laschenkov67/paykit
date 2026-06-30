package paykit

import (
	"fmt"
	"strconv"
	"strings"
)

type Money struct {
	Amount   int64
	Currency string
}

func RUB(kopecks int64) Money { return Money{Amount: kopecks, Currency: "RUB"} }

func USD(cents int64) Money { return Money{Amount: cents, Currency: "USD"} }

func EUR(cents int64) Money { return Money{Amount: cents, Currency: "EUR"} }

func (m Money) Major() string {
	sign := ""
	a := m.Amount
	if a < 0 {
		sign = "-"
		a = -a
	}
	major := a / 100
	minor := a % 100
	return fmt.Sprintf("%s%d.%02d", sign, major, minor)
}

func (m Money) String() string { return m.Major() + " " + m.Currency }

func ParseMajor(s, currency string) (Money, error) {
	neg := strings.HasPrefix(s, "-")
	body := s
	if neg {
		body = s[1:]
	}
	dot := strings.IndexByte(body, '.')
	majorStr, frac := body, ""
	if dot >= 0 {
		majorStr, frac = body[:dot], body[dot+1:]
	}
	major, err := strconv.ParseInt(majorStr, 10, 64)
	if err != nil {
		return Money{}, fmt.Errorf("paykit: parse money: %w", err)
	}
	if len(frac) == 1 {
		frac += "0"
	}
	if len(frac) > 2 {
		frac = frac[:2]
	}
	var minor int64
	if frac != "" {
		minor, err = strconv.ParseInt(frac, 10, 64)
		if err != nil {
			return Money{}, fmt.Errorf("paykit: parse money: %w", err)
		}
	}
	amount := major*100 + minor
	if neg {
		amount = -amount
	}
	return Money{Amount: amount, Currency: currency}, nil
}
