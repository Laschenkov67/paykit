package stripe_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/laschenkov67/paykit"
	"github.com/laschenkov67/paykit/providers/stripe"
)

func TestWebhookSignature(t *testing.T) {
	secret := "whsec_test"
	p, _ := stripe.New("sk_test_x", secret)

	body := `{"id":"evt_1","type":"payment_intent.succeeded","data":{"object":{"id":"pi_1","amount":1000,"currency":"usd","status":"succeeded","created":1700000000,"metadata":{"order_id":"o-1"}}}}`
	ts := fmt.Sprintf("%d", time.Now().Unix())

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts + "." + body))
	v1 := hex.EncodeToString(mac.Sum(nil))

	r := httptest.NewRequest("POST", "/wh", strings.NewReader(body))
	r.Header.Set("Stripe-Signature", "t="+ts+",v1="+v1)

	ev, err := p.ParseWebhook(r)
	if err != nil {
		t.Fatal(err)
	}
	if ev.Type != paykit.EventPaymentSucceeded || ev.Payment.ID != "pi_1" {
		t.Fatalf("bad event: %+v", ev)
	}

	// bad signature
	r2 := httptest.NewRequest("POST", "/wh", strings.NewReader(body))
	r2.Header.Set("Stripe-Signature", "t="+ts+",v1=deadbeef")
	if _, err := p.ParseWebhook(r2); err == nil {
		t.Fatal("expected signature error")
	}
}
