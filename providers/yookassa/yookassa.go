package yookassa

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/laschenkov67/paykit"
	"github.com/laschenkov67/paykit/internal/httpx"
)

const defaultBaseURL = "https://api.yookassa.ru/v3"

type Provider struct {
	shopID string
	secret string
	cfg    *paykit.Config
	base   string
}

func New(shopID, secretKey string, opts ...paykit.Option) (*Provider, error) {
	if shopID == "" || secretKey == "" {
		return nil, errors.New("yookassa: shopID and secretKey are required")
	}
	cfg := paykit.NewConfig(opts...)
	base := cfg.BaseURL
	if base == "" {
		base = defaultBaseURL
	}
	return &Provider{shopID: shopID, secret: secretKey, cfg: cfg, base: base}, nil
}

func (p *Provider) Name() string { return "yookassa" }

type ykAmount struct {
	Value    string `json:"value"`
	Currency string `json:"currency"`
}

type ykConfirmation struct {
	Type      string `json:"type"`
	ReturnURL string `json:"return_url,omitempty"`
	URL       string `json:"confirmation_url,omitempty"`
}

type ykPayment struct {
	ID           string            `json:"id"`
	Status       string            `json:"status"`
	Amount       ykAmount          `json:"amount"`
	Description  string            `json:"description"`
	Confirmation *ykConfirmation   `json:"confirmation,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	Paid         bool              `json:"paid"`
}

type ykCreateReq struct {
	Amount       ykAmount          `json:"amount"`
	Description  string            `json:"description,omitempty"`
	Capture      bool              `json:"capture"`
	Confirmation ykConfirmation    `json:"confirmation"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

func (p *Provider) CreatePayment(ctx context.Context, req paykit.CreatePaymentRequest) (*paykit.Payment, error) {
	body := ykCreateReq{
		Amount:      ykAmount{Value: req.Amount.Major(), Currency: req.Amount.Currency},
		Description: req.Description,
		Capture:     !req.TwoStage,
		Confirmation: ykConfirmation{
			Type:      "redirect",
			ReturnURL: req.ReturnURL,
		},
		Metadata: cloneMeta(req.Metadata, req.OrderID),
	}

	idem := req.IdempotencyKey
	if idem == "" {
		idem = newIdempotencyKey()
	}

	headers := map[string]string{
		"Idempotence-Key": idem,
		"Authorization":   basicAuth(p.shopID, p.secret),
		"User-Agent":      p.cfg.UserAgent,
	}

	var out ykPayment
	status, raw, err := httpx.DoJSON(ctx, p.cfg.HTTPClient,
		http.MethodPost, p.base+"/payments", headers, body, &out)
	if err != nil {
		return nil, mapHTTPError("yookassa", status, raw, err)
	}
	return mapPayment(&out, raw, req.OrderID), nil
}

func (p *Provider) GetPayment(ctx context.Context, id string) (*paykit.Payment, error) {
	headers := map[string]string{
		"Authorization": basicAuth(p.shopID, p.secret),
		"User-Agent":    p.cfg.UserAgent,
	}
	var out ykPayment
	status, raw, err := httpx.DoJSON(ctx, p.cfg.HTTPClient,
		http.MethodGet, p.base+"/payments/"+id, headers, nil, &out)
	if err != nil {
		return nil, mapHTTPError("yookassa", status, raw, err)
	}
	return mapPayment(&out, raw, ""), nil
}

func (p *Provider) CapturePayment(ctx context.Context, id string, amount *paykit.Money) (*paykit.Payment, error) {
	var body any
	if amount != nil {
		body = map[string]any{"amount": ykAmount{Value: amount.Major(), Currency: amount.Currency}}
	} else {
		body = map[string]any{}
	}
	headers := map[string]string{
		"Idempotence-Key": newIdempotencyKey(),
		"Authorization":   basicAuth(p.shopID, p.secret),
		"User-Agent":      p.cfg.UserAgent,
	}
	var out ykPayment
	status, raw, err := httpx.DoJSON(ctx, p.cfg.HTTPClient,
		http.MethodPost, fmt.Sprintf("%s/payments/%s/capture", p.base, id), headers, body, &out)
	if err != nil {
		return nil, mapHTTPError("yookassa", status, raw, err)
	}
	return mapPayment(&out, raw, ""), nil
}

func (p *Provider) CancelPayment(ctx context.Context, id string) (*paykit.Payment, error) {
	headers := map[string]string{
		"Idempotence-Key": newIdempotencyKey(),
		"Authorization":   basicAuth(p.shopID, p.secret),
		"User-Agent":      p.cfg.UserAgent,
	}
	var out ykPayment
	status, raw, err := httpx.DoJSON(ctx, p.cfg.HTTPClient,
		http.MethodPost, fmt.Sprintf("%s/payments/%s/cancel", p.base, id), headers, map[string]any{}, &out)
	if err != nil {
		return nil, mapHTTPError("yookassa", status, raw, err)
	}
	return mapPayment(&out, raw, ""), nil
}

func (p *Provider) Refund(ctx context.Context, paymentID string, amount *paykit.Money) (*paykit.Refund, error) {
	if amount == nil {
		return nil, errors.New("yookassa: refund amount is required")
	}
	body := map[string]any{
		"payment_id": paymentID,
		"amount":     ykAmount{Value: amount.Major(), Currency: amount.Currency},
	}
	headers := map[string]string{
		"Idempotence-Key": newIdempotencyKey(),
		"Authorization":   basicAuth(p.shopID, p.secret),
		"User-Agent":      p.cfg.UserAgent,
	}
	var out struct {
		ID        string    `json:"id"`
		PaymentID string    `json:"payment_id"`
		Status    string    `json:"status"`
		Amount    ykAmount  `json:"amount"`
		CreatedAt time.Time `json:"created_at"`
	}
	status, raw, err := httpx.DoJSON(ctx, p.cfg.HTTPClient,
		http.MethodPost, p.base+"/refunds", headers, body, &out)
	if err != nil {
		return nil, mapHTTPError("yookassa", status, raw, err)
	}
	return &paykit.Refund{
		ID:        out.ID,
		PaymentID: out.PaymentID,
		Amount:    paykit.Money{Amount: amount.Amount, Currency: out.Amount.Currency},
		Status:    mapStatus(out.Status),
		CreatedAt: out.CreatedAt,
		Provider:  "yookassa",
		Raw:       raw,
	}, nil
}

func mapStatus(s string) paykit.PaymentStatus {
	switch s {
	case "pending":
		return paykit.StatusPending
	case "waiting_for_capture":
		return paykit.StatusAuthorized
	case "succeeded":
		return paykit.StatusSucceeded
	case "canceled":
		return paykit.StatusCanceled
	}
	return paykit.PaymentStatus(s)
}

func mapPayment(p *ykPayment, raw json.RawMessage, orderID string) *paykit.Payment {
	amount, _ := paykit.ParseMajor(p.Amount.Value, p.Amount.Currency)
	url := ""
	if p.Confirmation != nil {
		url = p.Confirmation.URL
	}
	if orderID == "" {
		orderID = p.Metadata["order_id"]
	}
	return &paykit.Payment{
		ID:          p.ID,
		OrderID:     orderID,
		Status:      mapStatus(p.Status),
		Amount:      amount,
		PaymentURL:  url,
		Description: p.Description,
		Metadata:    p.Metadata,
		CreatedAt:   p.CreatedAt,
		Provider:    "yookassa",
		Raw:         raw,
	}
}

func cloneMeta(m map[string]string, orderID string) map[string]string {
	out := make(map[string]string, len(m)+1)
	for k, v := range m {
		out[k] = v
	}
	if orderID != "" {
		out["order_id"] = orderID
	}
	return out
}

func basicAuth(user, pass string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+pass))
}

func newIdempotencyKey() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func mapHTTPError(provider string, status int, body []byte, err error) error {
	pe := &paykit.ProviderError{Provider: provider, StatusCode: status, Body: body, Message: err.Error()}
	var apiErr struct {
		Type        string `json:"type"`
		Code        string `json:"code"`
		Description string `json:"description"`
	}
	if json.Unmarshal(body, &apiErr) == nil {
		pe.Code = apiErr.Code
		if apiErr.Description != "" {
			pe.Message = apiErr.Description
		}
	}
	return pe
}
