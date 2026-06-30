package cloudpayments_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/laschenkov67/paykit"
	"github.com/laschenkov67/paykit/providers/cloudpayments"
)

func TestGetPaymentAmountPrecision(t *testing.T) {
	// 19.99 cannot be represented exactly in float64; a naive int64(x*100)
	// conversion truncates it to 1998 instead of rounding to 1999.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"Success":true,"Model":{"TransactionId":777,"Amount":19.99,"Currency":"RUB","Status":"Completed"}}`))
	}))
	defer srv.Close()

	p, _ := cloudpayments.New("pub", "secret", paykit.WithBaseURL(srv.URL))
	pay, err := p.GetPayment(context.Background(), "777")
	if err != nil {
		t.Fatal(err)
	}
	if pay.Amount.Amount != 1999 {
		t.Fatalf("Amount=%d want 1999", pay.Amount.Amount)
	}
}

func TestWebhook(t *testing.T) {
	p, _ := cloudpayments.New("pub", "secret")
	form := url.Values{}
	form.Set("TransactionId", "777")
	form.Set("InvoiceId", "ord-9")
	form.Set("Amount", "100.50")
	form.Set("Currency", "RUB")
	form.Set("Status", "Completed")
	body := form.Encode()

	mac := hmac.New(sha256.New, []byte("secret"))
	mac.Write([]byte(body))
	sig := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	r := httptest.NewRequest("POST", "/wh", strings.NewReader(body))
	r.Header.Set("Content-HMAC", sig)

	ev, err := p.ParseWebhook(r)
	if err != nil {
		t.Fatal(err)
	}
	if ev.Type != paykit.EventPaymentSucceeded || ev.Payment.Amount.Amount != 10050 {
		t.Fatalf("bad event: %+v", ev.Payment)
	}
}
