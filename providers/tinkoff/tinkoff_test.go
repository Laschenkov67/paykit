package tinkoff_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/laschenkov67/paykit"
	"github.com/laschenkov67/paykit/providers/tinkoff"
)

func TestInitAndWebhook(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Init" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"Success":    true,
			"PaymentId":  "555",
			"OrderId":    "ord-1",
			"Amount":     19900,
			"Status":     "NEW",
			"PaymentURL": "https://tinkoff.example/pay/555",
		})
	}))
	defer srv.Close()

	p, _ := tinkoff.New("TERM", "PASS", paykit.WithBaseURL(srv.URL))
	pay, err := p.CreatePayment(context.Background(), paykit.CreatePaymentRequest{
		OrderID: "ord-1", Amount: paykit.RUB(199_00), Description: "Test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if pay.ID != "555" || pay.PaymentURL == "" {
		t.Fatalf("bad payment: %+v", pay)
	}

	params := map[string]string{
		"TerminalKey": "TERM",
		"OrderId":     "ord-1",
		"Success":     "true",
		"Status":      "CONFIRMED",
		"PaymentId":   "555",
		"Amount":      strconv.Itoa(19900),
		"ErrorCode":   "0",
	}
	token := signFor(t, p, params)
	wh := map[string]any{
		"TerminalKey": "TERM",
		"OrderId":     "ord-1",
		"Success":     true,
		"Status":      "CONFIRMED",
		"PaymentId":   555,
		"Amount":      19900,
		"ErrorCode":   "0",
		"Token":       token,
	}
	body, _ := json.Marshal(wh)
	r := httptest.NewRequest(http.MethodPost, "/wh", bytes.NewReader(body))
	ev, err := p.ParseWebhook(r)
	if err != nil {
		t.Fatal(err)
	}
	if ev.Type != paykit.EventPaymentSucceeded {
		t.Fatalf("type=%s", ev.Type)
	}

	r2 := httptest.NewRequest(http.MethodPost, "/wh",
		strings.NewReader(strings.Replace(string(body), token, "deadbeef", 1)))
	if _, err := p.ParseWebhook(r2); err == nil {
		t.Fatal("expected signature error")
	}
}

func signFor(t *testing.T, p *tinkoff.Provider, params map[string]string) string {
	t.Helper()
	keys := make([]string, 0, len(params)+1)
	all := make(map[string]string, len(params)+1)
	for k, v := range params {
		all[k] = v
		keys = append(keys, k)
	}

	return p.SignForTest(all)
}
