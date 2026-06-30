package cloudpayments

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"crypto/hmac"
	"crypto/sha256"

	"github.com/laschenkov67/paykit"
)

func (p *Provider) ParseWebhook(r *http.Request) (*paykit.WebhookEvent, error) {
	defer r.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return nil, err
	}

	got := r.Header.Get("Content-HMAC")
	if got == "" {
		got = r.Header.Get("X-Content-HMAC")
	}
	mac := hmac.New(sha256.New, []byte(p.apiPass))
	mac.Write(raw)
	want := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(got), []byte(want)) {
		return nil, paykit.ErrInvalidSignature
	}

	form, err := url.ParseQuery(string(raw))
	if err != nil {
		return nil, fmt.Errorf("%w: %s", paykit.ErrInvalidRequest, err)
	}

	amount, _ := paykit.ParseMajor(form.Get("Amount"), strings.ToUpper(form.Get("Currency")))
	pay := &paykit.Payment{
		ID:       form.Get("TransactionId"),
		OrderID:  form.Get("InvoiceId"),
		Amount:   amount,
		Status:   mapStatus(form.Get("Status")),
		Provider: "cloudpayments",
		Raw:      raw,
	}
	ev := &paykit.WebhookEvent{Provider: "cloudpayments", Raw: raw, Payment: pay}
	switch pay.Status {
	case paykit.StatusSucceeded:
		ev.Type = paykit.EventPaymentSucceeded
	case paykit.StatusCanceled, paykit.StatusFailed:
		ev.Type = paykit.EventPaymentFailed
	case paykit.StatusRefunded:
		ev.Type = paykit.EventRefundSucceeded
	default:
		ev.Type = paykit.EventPaymentPending
	}
	return ev, nil
}

func (p *Provider) WebhookAck(_ *paykit.WebhookEvent) (int, []byte) {
	return http.StatusOK, []byte(`{"code":0}`)
}
