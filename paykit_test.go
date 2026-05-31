package paykit_test

import (
	"testing"

	"github.com/laschenkov67/paykit"
)

func TestMoneyMajor(t *testing.T) {
	cases := []struct {
		in   paykit.Money
		want string
	}{
		{paykit.RUB(199_00), "199.00"},
		{paykit.RUB(1), "0.01"},
		{paykit.RUB(0), "0.00"},
		{paykit.RUB(-150), "-1.50"},
		{paykit.USD(99_99), "99.99"},
	}
	for _, c := range cases {
		if got := c.in.Major(); got != c.want {
			t.Errorf("Major(%v)=%q want %q", c.in, got, c.want)
		}
	}
}

func TestParseMajor(t *testing.T) {
	cases := []struct {
		in       string
		currency string
		want     int64
	}{
		{"199.00", "RUB", 19900},
		{"199", "RUB", 19900},
		{"1.5", "USD", 150},
		{"0.01", "EUR", 1},
	}
	for _, c := range cases {
		m, err := paykit.ParseMajor(c.in, c.currency)
		if err != nil {
			t.Fatalf("ParseMajor(%q): %v", c.in, err)
		}
		if m.Amount != c.want || m.Currency != c.currency {
			t.Errorf("ParseMajor(%q)=%v want %d %s", c.in, m, c.want, c.currency)
		}
	}
}

func TestPaymentIsTerminal(t *testing.T) {
	p := &paykit.Payment{Status: paykit.StatusSucceeded}
	if !p.IsTerminal() {
		t.Fatal("expected terminal")
	}
	p.Status = paykit.StatusPending
	if p.IsTerminal() {
		t.Fatal("expected non-terminal")
	}
}
