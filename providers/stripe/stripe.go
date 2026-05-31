// API: https://stripe.com/docs/api
package stripe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/laschenkov67/paykit"
)

const defaultBaseURL = "https://api.stripe.com/v1"

type Provider struct {
	secretKey     string
	webhookSecret string
	cfg           *paykit.Config
	base          string
}

func New(secretKey, webhookSecret string, opts ...paykit.Option) (*Provider, error) {
	if secretKey == "" {
		return nil, errors.New("stripe: secretKey is required")
	}
	cfg := paykit.NewConfig(opts...)
	base := cfg.BaseURL
	if base == "" {
		base = defaultBaseURL
	}
	return &Provider{secretKey: secretKey, webhookSecret: webhookSecret, cfg: cfg, base: base}, nil
}

func (p *Provider) Name() string { return "stripe" }

type stripeSession struct {
	ID            string            `json:"id"`
	URL           string            `json:"url"`
	PaymentIntent string            `json:"payment_intent"`
	AmountTotal   int64             `json:"amount_total"`
	Currency      string            `json:"currency"`
	Status        string            `json:"status"`
	Created       int64             `json:"created"`
	Metadata      map[string]string `json:"metadata"`
}

func (p *Provider) CreatePayment(ctx context.Context, req paykit.CreatePaymentRequest) (*paykit.Payment, error) {
	form := url.Values{}
	form.Set("mode", "payment")
	form.Set("success_url", emptyOr(req.ReturnURL, "https://example.com/success"))
	form.Set("cancel_url", emptyOr(req.ReturnURL, "https://example.com/cancel"))
	form.Set("line_items[0][quantity]", "1")
	form.Set("line_items[0][price_data][currency]", strings.ToLower(req.Amount.Currency))
	form.Set("line_items[0][price_data][unit_amount]", strconv.FormatInt(req.Amount.Amount, 10))
	form.Set("line_items[0][price_data][product_data][name]", emptyOr(req.Description, "Order "+req.OrderID))
	form.Set("client_reference_id", req.OrderID)
	for k, v := range req.Metadata {
		form.Set("metadata["+k+"]", v)
	}
	if req.Customer != nil && req.Customer.Email != "" {
		form.Set("customer_email", req.Customer.Email)
	}

	var s stripeSession
	raw, err := p.doForm(ctx, http.MethodPost, "/checkout/sessions", form, &s)
	if err != nil {
		return nil, err
	}
	return &paykit.Payment{
		ID:         s.PaymentIntent,
		OrderID:    req.OrderID,
		Status:     mapSessionStatus(s.Status),
		Amount:     req.Amount,
		PaymentURL: s.URL,
		Provider:   "stripe",
		CreatedAt:  time.Unix(s.Created, 0).UTC(),
		Metadata:   s.Metadata,
		Raw:        raw,
	}, nil
}

type stripePI struct {
	ID       string            `json:"id"`
	Amount   int64             `json:"amount"`
	Currency string            `json:"currency"`
	Status   string            `json:"status"`
	Created  int64             `json:"created"`
	Metadata map[string]string `json:"metadata"`
}

func (p *Provider) GetPayment(ctx context.Context, id string) (*paykit.Payment, error) {
	var pi stripePI
	raw, err := p.doForm(ctx, http.MethodGet, "/payment_intents/"+id, nil, &pi)
	if err != nil {
		return nil, err
	}
	return piToPayment(&pi, raw), nil
}

func (p *Provider) CapturePayment(ctx context.Context, id string, amount *paykit.Money) (*paykit.Payment, error) {
	form := url.Values{}
	if amount != nil {
		form.Set("amount_to_capture", strconv.FormatInt(amount.Amount, 10))
	}
	var pi stripePI
	raw, err := p.doForm(ctx, http.MethodPost, "/payment_intents/"+id+"/capture", form, &pi)
	if err != nil {
		return nil, err
	}
	return piToPayment(&pi, raw), nil
}

func (p *Provider) CancelPayment(ctx context.Context, id string) (*paykit.Payment, error) {
	var pi stripePI
	raw, err := p.doForm(ctx, http.MethodPost, "/payment_intents/"+id+"/cancel", url.Values{}, &pi)
	if err != nil {
		return nil, err
	}
	return piToPayment(&pi, raw), nil
}

func (p *Provider) Refund(ctx context.Context, paymentID string, amount *paykit.Money) (*paykit.Refund, error) {
	form := url.Values{"payment_intent": {paymentID}}
	if amount != nil {
		form.Set("amount", strconv.FormatInt(amount.Amount, 10))
	}
	var r struct {
		ID       string `json:"id"`
		Amount   int64  `json:"amount"`
		Currency string `json:"currency"`
		Status   string `json:"status"`
		Created  int64  `json:"created"`
	}
	raw, err := p.doForm(ctx, http.MethodPost, "/refunds", form, &r)
	if err != nil {
		return nil, err
	}
	return &paykit.Refund{
		ID:        r.ID,
		PaymentID: paymentID,
		Amount:    paykit.Money{Amount: r.Amount, Currency: strings.ToUpper(r.Currency)},
		Status:    paykit.StatusRefunded,
		CreatedAt: time.Unix(r.Created, 0).UTC(),
		Provider:  "stripe",
		Raw:       raw,
	}, nil
}

func (p *Provider) doForm(ctx context.Context, method, path string, form url.Values, out any) (json.RawMessage, error) {
	var body strings.Reader
	if form != nil {
		body = *strings.NewReader(form.Encode())
	}
	req, err := http.NewRequestWithContext(ctx, method, p.base+path, &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+p.secretKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", p.cfg.UserAgent)
	resp, err := p.cfg.HTTPClient.Do(req)
	if err != nil {
		return nil, &paykit.ProviderError{Provider: "stripe", Message: err.Error()}
	}
	defer resp.Body.Close()
	raw := readAll(resp.Body)
	if resp.StatusCode/100 != 2 {
		var apiErr struct {
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
				Type    string `json:"type"`
			} `json:"error"`
		}
		_ = json.Unmarshal(raw, &apiErr)
		return raw, &paykit.ProviderError{
			Provider: "stripe", StatusCode: resp.StatusCode,
			Code: apiErr.Error.Code, Message: apiErr.Error.Message, Body: raw,
		}
	}
	if out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			return raw, fmt.Errorf("stripe: decode: %w", err)
		}
	}
	return raw, nil
}

func piToPayment(pi *stripePI, raw json.RawMessage) *paykit.Payment {
	return &paykit.Payment{
		ID:        pi.ID,
		OrderID:   pi.Metadata["order_id"],
		Amount:    paykit.Money{Amount: pi.Amount, Currency: strings.ToUpper(pi.Currency)},
		Status:    mapPIStatus(pi.Status),
		Metadata:  pi.Metadata,
		CreatedAt: time.Unix(pi.Created, 0).UTC(),
		Provider:  "stripe",
		Raw:       raw,
	}
}

func mapPIStatus(s string) paykit.PaymentStatus {
	switch s {
	case "succeeded":
		return paykit.StatusSucceeded
	case "requires_capture":
		return paykit.StatusAuthorized
	case "canceled":
		return paykit.StatusCanceled
	case "processing", "requires_action", "requires_confirmation", "requires_payment_method":
		return paykit.StatusPending
	}
	return paykit.StatusPending
}

func mapSessionStatus(s string) paykit.PaymentStatus {
	switch s {
	case "complete":
		return paykit.StatusSucceeded
	case "expired":
		return paykit.StatusCanceled
	}
	return paykit.StatusPending
}

func emptyOr(a, b string) string {
	if a == "" {
		return b
	}
	return a
}

func readAll(r interface{ Read([]byte) (int, error) }) []byte {
	out := make([]byte, 0, 4096)
	buf := make([]byte, 4096)
	for {
		n, err := r.Read(buf)
		out = append(out, buf[:n]...)
		if err != nil {
			break
		}
	}
	return out
}
