// API: https://developers.cloudpayments.ru/
package cloudpayments

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/laschenkov67/paykit"
	"github.com/laschenkov67/paykit/internal/httpx"
)

const defaultBaseURL = "https://api.cloudpayments.ru"

type Provider struct {
	publicID string
	apiPass  string
	cfg      *paykit.Config
	base     string
}

func New(publicID, apiPassword string, opts ...paykit.Option) (*Provider, error) {
	if publicID == "" || apiPassword == "" {
		return nil, errors.New("cloudpayments: publicID and apiPassword are required")
	}
	cfg := paykit.NewConfig(opts...)
	base := cfg.BaseURL
	if base == "" {
		base = defaultBaseURL
	}
	return &Provider{publicID: publicID, apiPass: apiPassword, cfg: cfg, base: base}, nil
}

func (p *Provider) Name() string { return "cloudpayments" }

type cpResp struct {
	Success bool            `json:"Success"`
	Message string          `json:"Message"`
	Model   json.RawMessage `json:"Model"`
}

type cpTransaction struct {
	TransactionID int64   `json:"TransactionId"`
	Amount        float64 `json:"Amount"`
	Currency      string  `json:"Currency"`
	InvoiceID     string  `json:"InvoiceId"`
	Status        string  `json:"Status"`
	CreatedDate   string  `json:"CreatedDate"`
}

func (p *Provider) CreatePayment(ctx context.Context, req paykit.CreatePaymentRequest) (*paykit.Payment, error) {
	body := map[string]any{
		"Amount":      float64(req.Amount.Amount) / 100,
		"Currency":    req.Amount.Currency,
		"Description": req.Description,
		"InvoiceId":   req.OrderID,
		"JsonData":    req.Metadata,
	}
	if req.Customer != nil {
		body["Email"] = req.Customer.Email
	}

	var out struct {
		cpResp
		Model struct {
			Number string `json:"Number"`
			URL    string `json:"Url"`
		} `json:"Model"`
	}
	status, raw, err := httpx.DoJSON(ctx, p.cfg.HTTPClient,
		http.MethodPost, p.base+"/orders/create", p.headers(), body, &out)
	if err != nil || !out.Success {
		return nil, &paykit.ProviderError{
			Provider: "cloudpayments", StatusCode: status, Body: raw,
			Message: firstNonEmpty(out.Message, errString(err)),
		}
	}
	return &paykit.Payment{
		ID:         out.Model.Number,
		OrderID:    req.OrderID,
		Status:     paykit.StatusPending,
		Amount:     req.Amount,
		PaymentURL: out.Model.URL,
		Provider:   "cloudpayments",
		CreatedAt:  time.Now().UTC(),
		Raw:        raw,
	}, nil
}

func (p *Provider) GetPayment(ctx context.Context, id string) (*paykit.Payment, error) {
	body := map[string]any{"TransactionId": id}
	var out struct {
		cpResp
		Model cpTransaction `json:"Model"`
	}
	status, raw, err := httpx.DoJSON(ctx, p.cfg.HTTPClient,
		http.MethodPost, p.base+"/payments/get", p.headers(), body, &out)
	if err != nil || !out.Success {
		return nil, &paykit.ProviderError{
			Provider: "cloudpayments", StatusCode: status, Body: raw,
			Message: firstNonEmpty(out.Message, errString(err)),
		}
	}
	return &paykit.Payment{
		ID:       fmt.Sprintf("%d", out.Model.TransactionID),
		OrderID:  out.Model.InvoiceID,
		Status:   mapStatus(out.Model.Status),
		Amount:   paykit.Money{Amount: int64(out.Model.Amount * 100), Currency: out.Model.Currency},
		Provider: "cloudpayments",
		Raw:      raw,
	}, nil
}

func (p *Provider) CapturePayment(ctx context.Context, id string, amount *paykit.Money) (*paykit.Payment, error) {
	body := map[string]any{"TransactionId": id}
	if amount != nil {
		body["Amount"] = float64(amount.Amount) / 100
	}
	var out cpResp
	status, raw, err := httpx.DoJSON(ctx, p.cfg.HTTPClient,
		http.MethodPost, p.base+"/payments/confirm", p.headers(), body, &out)
	if err != nil || !out.Success {
		return nil, &paykit.ProviderError{
			Provider: "cloudpayments", StatusCode: status, Body: raw,
			Message: firstNonEmpty(out.Message, errString(err)),
		}
	}
	return &paykit.Payment{ID: id, Status: paykit.StatusSucceeded, Provider: "cloudpayments", Raw: raw}, nil
}

func (p *Provider) CancelPayment(ctx context.Context, id string) (*paykit.Payment, error) {
	body := map[string]any{"TransactionId": id}
	var out cpResp
	status, raw, err := httpx.DoJSON(ctx, p.cfg.HTTPClient,
		http.MethodPost, p.base+"/payments/void", p.headers(), body, &out)
	if err != nil || !out.Success {
		return nil, &paykit.ProviderError{
			Provider: "cloudpayments", StatusCode: status, Body: raw,
			Message: firstNonEmpty(out.Message, errString(err)),
		}
	}
	return &paykit.Payment{ID: id, Status: paykit.StatusCanceled, Provider: "cloudpayments", Raw: raw}, nil
}

func (p *Provider) Refund(ctx context.Context, paymentID string, amount *paykit.Money) (*paykit.Refund, error) {
	if amount == nil {
		return nil, errors.New("cloudpayments: refund amount is required")
	}
	body := map[string]any{
		"TransactionId": paymentID,
		"Amount":        float64(amount.Amount) / 100,
	}
	var out cpResp
	status, raw, err := httpx.DoJSON(ctx, p.cfg.HTTPClient,
		http.MethodPost, p.base+"/payments/refund", p.headers(), body, &out)
	if err != nil || !out.Success {
		return nil, &paykit.ProviderError{
			Provider: "cloudpayments", StatusCode: status, Body: raw,
			Message: firstNonEmpty(out.Message, errString(err)),
		}
	}
	return &paykit.Refund{
		PaymentID: paymentID, Amount: *amount,
		Status: paykit.StatusRefunded, CreatedAt: time.Now().UTC(),
		Provider: "cloudpayments", Raw: raw,
	}, nil
}

func (p *Provider) headers() map[string]string {
	return map[string]string{
		"Authorization": "Basic " + basic(p.publicID, p.apiPass),
		"User-Agent":    p.cfg.UserAgent,
	}
}

func mapStatus(s string) paykit.PaymentStatus {
	switch s {
	case "Completed":
		return paykit.StatusSucceeded
	case "Authorized":
		return paykit.StatusAuthorized
	case "Cancelled", "Voided":
		return paykit.StatusCanceled
	case "Declined":
		return paykit.StatusFailed
	case "Refunded":
		return paykit.StatusRefunded
	}
	return paykit.StatusPending
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
func errString(e error) string {
	if e == nil {
		return ""
	}
	return e.Error()
}

func basic(user, pass string) string {
	const tbl = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	in := []byte(user + ":" + pass)
	out := make([]byte, 0, (len(in)+2)/3*4)
	for i := 0; i < len(in); i += 3 {
		var n uint32
		n |= uint32(in[i]) << 16
		if i+1 < len(in) {
			n |= uint32(in[i+1]) << 8
		}
		if i+2 < len(in) {
			n |= uint32(in[i+2])
		}
		out = append(out, tbl[(n>>18)&0x3F], tbl[(n>>12)&0x3F])
		if i+1 < len(in) {
			out = append(out, tbl[(n>>6)&0x3F])
		} else {
			out = append(out, '=')
		}
		if i+2 < len(in) {
			out = append(out, tbl[n&0x3F])
		} else {
			out = append(out, '=')
		}
	}
	return string(out)
}
