package stripe

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/laschenkov67/paykit"
	"github.com/laschenkov67/paykit/internal/signing"
)

func (p *Provider) ParseWebhook(r *http.Request) (*paykit.WebhookEvent, error) {
	defer r.Body.Close()
	if p.webhookSecret == "" {
		return nil, errors.New("stripe: webhook secret is not configured")
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if err := p.verifySignature(r.Header.Get("Stripe-Signature"), raw); err != nil {
		return nil, err
	}

	var ev struct {
		ID   string `json:"id"`
		Type string `json:"type"`
		Data struct {
			Object json.RawMessage `json:"object"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &ev); err != nil {
		return nil, fmt.Errorf("%w: %s", paykit.ErrInvalidRequest, err)
	}

	out := &paykit.WebhookEvent{Provider: "stripe", Raw: raw}
	switch ev.Type {
	case "payment_intent.succeeded":
		var pi stripePI
		_ = json.Unmarshal(ev.Data.Object, &pi)
		out.Type = paykit.EventPaymentSucceeded
		out.Payment = piToPayment(&pi, ev.Data.Object)
	case "payment_intent.payment_failed":
		var pi stripePI
		_ = json.Unmarshal(ev.Data.Object, &pi)
		out.Type = paykit.EventPaymentFailed
		out.Payment = piToPayment(&pi, ev.Data.Object)
	case "payment_intent.canceled":
		var pi stripePI
		_ = json.Unmarshal(ev.Data.Object, &pi)
		out.Type = paykit.EventPaymentCanceled
		out.Payment = piToPayment(&pi, ev.Data.Object)
	case "charge.refunded":
		out.Type = paykit.EventRefundSucceeded
	default:
		out.Type = paykit.EventUnknown
	}
	return out, nil
}

// WebhookAck satisfies paykit.Provider. Stripe only needs a plain 2xx
// response to consider a notification delivered.
func (p *Provider) WebhookAck(_ *paykit.WebhookEvent) (int, []byte) {
	return http.StatusOK, nil
}

func (p *Provider) verifySignature(header string, body []byte) error {
	if header == "" {
		return paykit.ErrInvalidSignature
	}
	var ts, v1 string
	for _, part := range strings.Split(header, ",") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "t":
			ts = kv[1]
		case "v1":
			v1 = kv[1]
		}
	}
	if ts == "" || v1 == "" {
		return paykit.ErrInvalidSignature
	}
	t, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return paykit.ErrInvalidSignature
	}
	if time.Since(time.Unix(t, 0)) > 5*time.Minute {
		return paykit.ErrInvalidSignature
	}
	signedPayload := fmt.Sprintf("%s.%s", ts, body)
	want := signing.HMACSHA256Hex([]byte(p.webhookSecret), []byte(signedPayload))
	if !signing.EqualHex(v1, want) {
		return paykit.ErrInvalidSignature
	}
	return nil
}
