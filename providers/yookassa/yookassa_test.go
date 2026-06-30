package yookassa_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/laschenkov67/paykit"
	"github.com/laschenkov67/paykit/providers/yookassa"
)

func TestCreatePayment(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/payments" || r.Method != http.MethodPost {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Idempotence-Key") == "" {
			t.Fatal("Idempotence-Key not set")
		}
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Basic ") {
			t.Fatal("Authorization not Basic")
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if capture, _ := body["capture"].(bool); !capture {
			t.Fatalf("capture should default to true (one-stage) when TwoStage is unset, got %v", body["capture"])
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":     "pay_123",
			"status": "pending",
			"amount": map[string]string{"value": "199.00", "currency": "RUB"},
			"confirmation": map[string]string{
				"type":             "redirect",
				"confirmation_url": "https://yookassa.example/qr",
			},
			"created_at": "2024-01-02T03:04:05Z",
			"metadata":   map[string]string{"order_id": "ord-1"},
		})
	}))
	defer srv.Close()

	p, err := yookassa.New("shop", "secret", paykit.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	pay, err := p.CreatePayment(context.Background(), paykit.CreatePaymentRequest{
		OrderID:     "ord-1",
		Amount:      paykit.RUB(199_00),
		Description: "Test",
		ReturnURL:   "https://shop.example/return",
	})
	if err != nil {
		t.Fatal(err)
	}
	if pay.ID != "pay_123" || pay.PaymentURL == "" || pay.Status != paykit.StatusPending {
		t.Fatalf("bad payment: %+v", pay)
	}
}

func TestCreatePaymentTwoStageDoesNotCaptureImmediately(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if capture, ok := body["capture"].(bool); !ok || capture {
			t.Fatalf("capture should be false for a two-stage payment, got %v", body["capture"])
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "pay_124", "status": "pending",
			"amount": map[string]string{"value": "199.00", "currency": "RUB"},
		})
	}))
	defer srv.Close()

	p, _ := yookassa.New("shop", "secret", paykit.WithBaseURL(srv.URL))
	if _, err := p.CreatePayment(context.Background(), paykit.CreatePaymentRequest{
		OrderID: "ord-2", Amount: paykit.RUB(199_00), TwoStage: true,
	}); err != nil {
		t.Fatal(err)
	}
}

func TestParseWebhook(t *testing.T) {
	body := `{"type":"notification","event":"payment.succeeded","object":{
		"id":"pay_1","status":"succeeded","amount":{"value":"10.00","currency":"RUB"},
		"created_at":"2024-01-02T03:04:05Z","metadata":{"order_id":"ord-7"}}}`
	r := httptest.NewRequest(http.MethodPost, "/wh", strings.NewReader(body))
	r.RemoteAddr = "77.75.156.11:443" // within yookassa.AllowedIPs()
	p, _ := yookassa.New("a", "b")
	ev, err := p.ParseWebhook(r)
	if err != nil {
		t.Fatal(err)
	}
	if ev.Type != paykit.EventPaymentSucceeded {
		t.Fatalf("type=%s", ev.Type)
	}
	if ev.Payment.OrderID != "ord-7" {
		t.Fatalf("order_id mismatch: %+v", ev.Payment)
	}
}

func TestParseWebhookRejectsUntrustedIP(t *testing.T) {
	body := `{"type":"notification","event":"payment.succeeded","object":{
		"id":"pay_1","status":"succeeded","amount":{"value":"10.00","currency":"RUB"},
		"created_at":"2024-01-02T03:04:05Z","metadata":{"order_id":"ord-7"}}}`
	r := httptest.NewRequest(http.MethodPost, "/wh", strings.NewReader(body))
	r.RemoteAddr = "8.8.8.8:443" // not in yookassa.AllowedIPs()
	p, _ := yookassa.New("a", "b")
	if _, err := p.ParseWebhook(r); err == nil {
		t.Fatal("expected error for untrusted source IP")
	}
}
