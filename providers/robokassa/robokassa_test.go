package robokassa_test

import (
	"context"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"crypto/md5"
	"encoding/hex"

	"github.com/laschenkov67/paykit"
	"github.com/laschenkov67/paykit/providers/robokassa"
)

func TestCreateAndWebhook(t *testing.T) {
	p, _ := robokassa.New("login", "p1", "p2")
	pay, err := p.CreatePayment(context.Background(), paykit.CreatePaymentRequest{
		OrderID: "100", Amount: paykit.RUB(199_00), Description: "Test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(pay.PaymentURL, "SignatureValue=") {
		t.Fatal("no signature in URL")
	}

	form := url.Values{}
	form.Set("OutSum", "199.00")
	form.Set("InvId", "100")
	sum := md5.Sum([]byte("199.00:100:p2"))
	form.Set("SignatureValue", hex.EncodeToString(sum[:]))
	r := httptest.NewRequest("POST", "/wh", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	ev, err := p.ParseWebhook(r)
	if err != nil {
		t.Fatal(err)
	}
	if ev.Type != paykit.EventPaymentSucceeded || ev.Payment.OrderID != "100" {
		t.Fatalf("bad ev: %+v", ev)
	}
}
