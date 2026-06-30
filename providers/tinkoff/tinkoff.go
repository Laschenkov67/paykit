// API: https://www.tinkoff.ru/kassa/develop/api/payments/
package tinkoff

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/laschenkov67/paykit"
	"github.com/laschenkov67/paykit/internal/httpx"
	"github.com/laschenkov67/paykit/internal/signing"
)

const defaultBaseURL = "https://securepay.tinkoff.ru/v2"

type Provider struct {
	terminalKey string
	password    string
	cfg         *paykit.Config
	base        string
}

func New(terminalKey, password string, opts ...paykit.Option) (*Provider, error) {
	if terminalKey == "" || password == "" {
		return nil, errors.New("tinkoff: terminalKey and password are required")
	}
	cfg := paykit.NewConfig(opts...)
	base := cfg.BaseURL
	if base == "" {
		base = defaultBaseURL
	}
	return &Provider{terminalKey: terminalKey, password: password, cfg: cfg, base: base}, nil
}

func (p *Provider) Name() string { return "tinkoff" }

func (p *Provider) sign(params map[string]string) string {
	params["Password"] = p.password
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var buf []byte
	for _, k := range keys {
		buf = append(buf, params[k]...)
	}
	delete(params, "Password")
	sum := sha256.Sum256(buf)
	return hex.EncodeToString(sum[:])
}

type initReq struct {
	TerminalKey     string            `json:"TerminalKey"`
	Amount          int64             `json:"Amount"`
	OrderID         string            `json:"OrderId"`
	Description     string            `json:"Description,omitempty"`
	PayType         string            `json:"PayType,omitempty"`
	SuccessURL      string            `json:"SuccessURL,omitempty"`
	FailURL         string            `json:"FailURL,omitempty"`
	NotificationURL string            `json:"NotificationURL,omitempty"`
	Token           string            `json:"Token"`
	DATA            map[string]string `json:"DATA,omitempty"`
}

type initResp struct {
	Success    bool   `json:"Success"`
	ErrorCode  string `json:"ErrorCode"`
	Message    string `json:"Message"`
	Details    string `json:"Details"`
	PaymentID  string `json:"PaymentId"`
	OrderID    string `json:"OrderId"`
	Amount     int64  `json:"Amount"`
	Status     string `json:"Status"`
	PaymentURL string `json:"PaymentURL"`
}

func (p *Provider) SignForTest(params map[string]string) string {
	return p.sign(params)
}

func (p *Provider) CreatePayment(ctx context.Context, req paykit.CreatePaymentRequest) (*paykit.Payment, error) {
	payType := ""
	if req.TwoStage {
		payType = "T" // two-stage: authorize only, capture later via CapturePayment
	}
	signParams := map[string]string{
		"TerminalKey": p.terminalKey,
		"Amount":      strconv.FormatInt(req.Amount.Amount, 10),
		"OrderId":     req.OrderID,
		"Description": req.Description,
		"SuccessURL":  req.ReturnURL,
	}
	if payType != "" {
		signParams["PayType"] = payType
	}
	body := initReq{
		TerminalKey: p.terminalKey,
		Amount:      req.Amount.Amount,
		OrderID:     req.OrderID,
		Description: req.Description,
		PayType:     payType,
		SuccessURL:  req.ReturnURL,
		Token:       p.sign(signParams),
		DATA:        req.Metadata,
	}

	var out initResp
	status, raw, err := httpx.DoJSON(ctx, p.cfg.HTTPClient,
		http.MethodPost, p.base+"/Init", headers(p.cfg.UserAgent), body, &out)
	if err != nil || !out.Success {
		return nil, mapErr("tinkoff", status, raw, out.ErrorCode, out.Message, err)
	}
	return &paykit.Payment{
		ID:         out.PaymentID,
		OrderID:    out.OrderID,
		Status:     mapStatus(out.Status),
		Amount:     paykit.Money{Amount: out.Amount, Currency: req.Amount.Currency},
		PaymentURL: out.PaymentURL,
		Provider:   "tinkoff",
		CreatedAt:  time.Now().UTC(),
		Raw:        raw,
	}, nil
}

func (p *Provider) GetPayment(ctx context.Context, id string) (*paykit.Payment, error) {
	signParams := map[string]string{
		"TerminalKey": p.terminalKey,
		"PaymentId":   id,
	}
	body := map[string]any{
		"TerminalKey": p.terminalKey,
		"PaymentId":   id,
		"Token":       p.sign(signParams),
	}
	var out initResp
	status, raw, err := httpx.DoJSON(ctx, p.cfg.HTTPClient,
		http.MethodPost, p.base+"/GetState", headers(p.cfg.UserAgent), body, &out)
	if err != nil || !out.Success {
		return nil, mapErr("tinkoff", status, raw, out.ErrorCode, out.Message, err)
	}
	return &paykit.Payment{
		ID:       out.PaymentID,
		OrderID:  out.OrderID,
		Status:   mapStatus(out.Status),
		Amount:   paykit.Money{Amount: out.Amount, Currency: "RUB"},
		Provider: "tinkoff",
		Raw:      raw,
	}, nil
}

func (p *Provider) CapturePayment(ctx context.Context, id string, amount *paykit.Money) (*paykit.Payment, error) {
	signParams := map[string]string{
		"TerminalKey": p.terminalKey,
		"PaymentId":   id,
	}
	if amount != nil {
		signParams["Amount"] = strconv.FormatInt(amount.Amount, 10)
	}
	body := map[string]any{
		"TerminalKey": p.terminalKey,
		"PaymentId":   id,
		"Token":       p.sign(signParams),
	}
	if amount != nil {
		body["Amount"] = amount.Amount
	}
	var out initResp
	status, raw, err := httpx.DoJSON(ctx, p.cfg.HTTPClient,
		http.MethodPost, p.base+"/Confirm", headers(p.cfg.UserAgent), body, &out)
	if err != nil || !out.Success {
		return nil, mapErr("tinkoff", status, raw, out.ErrorCode, out.Message, err)
	}
	return &paykit.Payment{ID: out.PaymentID, OrderID: out.OrderID,
		Status: mapStatus(out.Status), Provider: "tinkoff", Raw: raw}, nil
}

func (p *Provider) CancelPayment(ctx context.Context, id string) (*paykit.Payment, error) {
	signParams := map[string]string{"TerminalKey": p.terminalKey, "PaymentId": id}
	body := map[string]any{
		"TerminalKey": p.terminalKey,
		"PaymentId":   id,
		"Token":       p.sign(signParams),
	}
	var out initResp
	status, raw, err := httpx.DoJSON(ctx, p.cfg.HTTPClient,
		http.MethodPost, p.base+"/Cancel", headers(p.cfg.UserAgent), body, &out)
	if err != nil || !out.Success {
		return nil, mapErr("tinkoff", status, raw, out.ErrorCode, out.Message, err)
	}
	return &paykit.Payment{ID: out.PaymentID, OrderID: out.OrderID,
		Status: mapStatus(out.Status), Provider: "tinkoff", Raw: raw}, nil
}

func (p *Provider) Refund(ctx context.Context, paymentID string, amount *paykit.Money) (*paykit.Refund, error) {
	signParams := map[string]string{"TerminalKey": p.terminalKey, "PaymentId": paymentID}
	body := map[string]any{
		"TerminalKey": p.terminalKey,
		"PaymentId":   paymentID,
	}
	if amount != nil {
		signParams["Amount"] = strconv.FormatInt(amount.Amount, 10)
		body["Amount"] = amount.Amount
	}
	body["Token"] = p.sign(signParams)
	var out initResp
	status, raw, err := httpx.DoJSON(ctx, p.cfg.HTTPClient,
		http.MethodPost, p.base+"/Cancel", headers(p.cfg.UserAgent), body, &out)
	if err != nil || !out.Success {
		return nil, mapErr("tinkoff", status, raw, out.ErrorCode, out.Message, err)
	}
	a := paykit.Money{Currency: "RUB"}
	if amount != nil {
		a = *amount
	}
	return &paykit.Refund{
		ID: out.PaymentID, PaymentID: paymentID, Amount: a,
		Status: paykit.StatusRefunded, CreatedAt: time.Now().UTC(),
		Provider: "tinkoff", Raw: raw,
	}, nil
}

func mapStatus(s string) paykit.PaymentStatus {
	switch s {
	case "NEW", "FORM_SHOWED", "AUTHORIZING", "3DS_CHECKING", "3DS_CHECKED":
		return paykit.StatusPending
	case "AUTHORIZED":
		return paykit.StatusAuthorized
	case "CONFIRMED":
		return paykit.StatusSucceeded
	case "REVERSED", "CANCELED":
		return paykit.StatusCanceled
	case "REFUNDED", "PARTIAL_REFUNDED":
		return paykit.StatusRefunded
	case "REJECTED", "DEADLINE_EXPIRED":
		return paykit.StatusFailed
	}
	return paykit.PaymentStatus(s)
}

func headers(ua string) map[string]string {
	return map[string]string{"User-Agent": ua}
}

func mapErr(provider string, status int, body []byte, code, msg string, err error) error {
	pe := &paykit.ProviderError{Provider: provider, StatusCode: status, Code: code, Message: msg, Body: body}
	if pe.Message == "" && err != nil {
		pe.Message = err.Error()
	}
	return pe
}

func (p *Provider) Verify(params map[string]string, token string) bool {
	return signing.EqualHex(p.sign(params), token)
}
