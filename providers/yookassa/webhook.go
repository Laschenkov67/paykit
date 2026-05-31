package yookassa

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/laschenkov67/paykit"
)

type ykNotification struct {
	Type   string          `json:"type"`
	Event  string          `json:"event"`
	Object json.RawMessage `json:"object"`
}

func (p *Provider) ParseWebhook(r *http.Request) (*paykit.WebhookEvent, error) {
	defer r.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("yookassa webhook: read body: %w", err)
	}
	var n ykNotification
	if err := json.Unmarshal(raw, &n); err != nil {
		return nil, fmt.Errorf("%w: %s", paykit.ErrInvalidRequest, err)
	}
	ev := &paykit.WebhookEvent{Provider: "yookassa", Raw: raw}

	switch n.Event {
	case "payment.succeeded", "payment.waiting_for_capture",
		"payment.canceled", "payment.pending":
		var pp ykPayment
		if err := json.Unmarshal(n.Object, &pp); err != nil {
			return nil, fmt.Errorf("yookassa webhook: %w", err)
		}
		pay := mapPayment(&pp, n.Object, "")
		ev.Payment = pay
		switch n.Event {
		case "payment.succeeded":
			ev.Type = paykit.EventPaymentSucceeded
		case "payment.canceled":
			ev.Type = paykit.EventPaymentCanceled
		default:
			ev.Type = paykit.EventPaymentPending
		}
	case "refund.succeeded":
		ev.Type = paykit.EventRefundSucceeded
	default:
		ev.Type = paykit.EventUnknown
	}
	return ev, nil
}

func AllowedIPs() []string {
	return []string{
		"185.71.76.0/27",
		"185.71.77.0/27",
		"77.75.153.0/25",
		"77.75.156.11/32",
		"77.75.156.35/32",
		"77.75.154.128/25",
		"2a02:5180::/32",
	}
}
