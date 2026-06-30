package paykit_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/laschenkov67/paykit"
)

// fakeProvider implements paykit.Provider with a configurable webhook ack,
// used to verify Manager.HandleWebhook actually writes it back to the client.
type fakeProvider struct {
	name    string
	ackCode int
	ackBody []byte
}

func (f *fakeProvider) Name() string { return f.name }

func (f *fakeProvider) CreatePayment(_ context.Context, _ paykit.CreatePaymentRequest) (*paykit.Payment, error) {
	return nil, nil
}

func (f *fakeProvider) GetPayment(_ context.Context, _ string) (*paykit.Payment, error) {
	return nil, nil
}

func (f *fakeProvider) CapturePayment(_ context.Context, _ string, _ *paykit.Money) (*paykit.Payment, error) {
	return nil, nil
}

func (f *fakeProvider) CancelPayment(_ context.Context, _ string) (*paykit.Payment, error) {
	return nil, nil
}

func (f *fakeProvider) Refund(_ context.Context, _ string, _ *paykit.Money) (*paykit.Refund, error) {
	return nil, nil
}

func (f *fakeProvider) ParseWebhook(_ *http.Request) (*paykit.WebhookEvent, error) {
	return &paykit.WebhookEvent{Provider: f.name, Type: paykit.EventPaymentSucceeded}, nil
}

func (f *fakeProvider) WebhookAck(_ *paykit.WebhookEvent) (int, []byte) {
	code := f.ackCode
	if code == 0 {
		code = http.StatusOK
	}
	return code, f.ackBody
}

func TestManagerHandleWebhookWritesProviderAck(t *testing.T) {
	mgr := paykit.NewManager()
	mgr.Register(&fakeProvider{name: "fake", ackBody: []byte("OK123")})

	mux := http.NewServeMux()
	mux.HandleFunc("POST /webhook/{provider}", mgr.HandleWebhook(nil))

	r := httptest.NewRequest(http.MethodPost, "/webhook/fake", strings.NewReader("{}"))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
	if w.Body.String() != "OK123" {
		t.Fatalf("body=%q want %q", w.Body.String(), "OK123")
	}
}
