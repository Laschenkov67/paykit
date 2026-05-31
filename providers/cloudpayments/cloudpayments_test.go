package cloudpayments_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/laschenkov67/paykit"
	"github.com/laschenkov67/paykit/providers/cloudpayments"
)

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
