package paykit

import (
	"fmt"
	"strconv"
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
	var major, minor int64
	var err error
	dot := -1
	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			dot = i
			break
		}
	}
	if dot < 0 {
		major, err = strconv.ParseInt(s, 10, 64)
		if err != nil {
			return Money{}, fmt.Errorf("paykit: parse money: %w", err)
		}
		return Money{Amount: major * 100, Currency: currency}, nil
	}
	major, err = strconv.ParseInt(s[:dot], 10, 64)
	if err != nil {
		return Money{}, fmt.Errorf("paykit: parse money: %w", err)
	}
	frac := s[dot+1:]
	if len(frac) == 1 {
		frac += "0"
	}
	if len(frac) > 2 {
		frac = frac[:2]
	}
	minor, err = strconv.ParseInt(frac, 10, 64)
	if err != nil {
		return Money{}, fmt.Errorf("paykit: parse money: %w", err)
	}
	if major < 0 {
		minor = -minor
	}
	return Money{Amount: major*100 + minor, Currency: currency}, nil
}
