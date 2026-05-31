package stripe

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/laschenkov67/paykit"
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
	mac := hmac.New(sha256.New, []byte(p.webhookSecret))
	fmt.Fprintf(mac, "%s.%s", ts, body)
	want := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(v1), []byte(want)) {
		return paykit.ErrInvalidSignature
	}
	return nil
}
