package tinkoff

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/laschenkov67/paykit"
)

type tinkoffNotification struct {
	TerminalKey string `json:"TerminalKey"`
	OrderID     string `json:"OrderId"`
	Success     bool   `json:"Success"`
	Status      string `json:"Status"`
	PaymentID   int64  `json:"PaymentId"`
	ErrorCode   string `json:"ErrorCode"`
	Amount      int64  `json:"Amount"`
	Token       string `json:"Token"`
}

func (p *Provider) ParseWebhook(r *http.Request) (*paykit.WebhookEvent, error) {
	defer r.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("tinkoff webhook: %w", err)
	}
	var n tinkoffNotification
	if err := json.Unmarshal(raw, &n); err != nil {
		return nil, fmt.Errorf("%w: %s", paykit.ErrInvalidRequest, err)
	}

	var asMap map[string]any
	if err := json.Unmarshal(raw, &asMap); err != nil {
		return nil, err
	}
	params := make(map[string]string, len(asMap))
	for k, v := range asMap {
		if k == "Token" {
			continue
		}
		switch t := v.(type) {
		case string:
			params[k] = t
		case bool:
			params[k] = strconv.FormatBool(t)
		case float64:
			params[k] = strconv.FormatInt(int64(t), 10)
		}
	}
	if !p.Verify(params, n.Token) {
		return nil, paykit.ErrInvalidSignature
	}

	pay := &paykit.Payment{
		ID:       strconv.FormatInt(n.PaymentID, 10),
		OrderID:  n.OrderID,
		Status:   mapStatus(n.Status),
		Amount:   paykit.Money{Amount: n.Amount, Currency: "RUB"},
		Provider: "tinkoff",
		Raw:      raw,
	}
	ev := &paykit.WebhookEvent{Provider: "tinkoff", Raw: raw, Payment: pay}
	switch pay.Status {
	case paykit.StatusSucceeded:
		ev.Type = paykit.EventPaymentSucceeded
	case paykit.StatusCanceled:
		ev.Type = paykit.EventPaymentCanceled
	case paykit.StatusFailed:
		ev.Type = paykit.EventPaymentFailed
	case paykit.StatusRefunded:
		ev.Type = paykit.EventRefundSucceeded
	default:
		ev.Type = paykit.EventPaymentPending
	}
	return ev, nil
}

func (p *Provider) WebhookAck(_ *paykit.WebhookEvent) (int, []byte) {
	return http.StatusOK, []byte("OK")
}
