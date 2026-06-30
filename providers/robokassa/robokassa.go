package robokassa

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/laschenkov67/paykit"
)

const (
	defaultPayURL = "https://auth.robokassa.ru/Merchant/Index.aspx"
	stateAPIURL   = "https://auth.robokassa.ru/Merchant/WebService/Service.asmx/OpStateExt"
)

type Provider struct {
	login  string
	pass1  string // for payment URL
	pass2  string // for result/webhook signature
	cfg    *paykit.Config
	payURL string
	isTest bool
}

func New(login, pass1, pass2 string, opts ...paykit.Option) (*Provider, error) {
	if login == "" || pass1 == "" || pass2 == "" {
		return nil, errors.New("robokassa: login, pass1 and pass2 are required")
	}
	cfg := paykit.NewConfig(opts...)
	pay := cfg.BaseURL
	if pay == "" {
		pay = defaultPayURL
	}
	return &Provider{login: login, pass1: pass1, pass2: pass2, cfg: cfg, payURL: pay}, nil
}

func (p *Provider) Name() string { return "robokassa" }

func (p *Provider) SetTestMode(t bool) { p.isTest = t }

func (p *Provider) CreatePayment(_ context.Context, req paykit.CreatePaymentRequest) (*paykit.Payment, error) {
	if req.OrderID == "" {
		return nil, errors.New("robokassa: OrderID is required (InvId)")
	}
	invID, err := strconv.ParseInt(req.OrderID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("robokassa: OrderID must be numeric (InvId): %w", err)
	}
	amount := req.Amount.Major()
	// signature: MerchantLogin:OutSum:InvId:Password1
	sig := md5hex(fmt.Sprintf("%s:%s:%d:%s", p.login, amount, invID, p.pass1))

	q := url.Values{}
	q.Set("MerchantLogin", p.login)
	q.Set("OutSum", amount)
	q.Set("InvId", req.OrderID)
	q.Set("Description", req.Description)
	q.Set("SignatureValue", sig)
	q.Set("Culture", "ru")
	if req.Amount.Currency != "" && req.Amount.Currency != "RUB" {
		q.Set("OutSumCurrency", req.Amount.Currency)
	}
	if req.ReturnURL != "" {
		q.Set("SuccessURL", req.ReturnURL)
	}
	if p.isTest {
		q.Set("IsTest", "1")
	}
	return &paykit.Payment{
		ID:         req.OrderID,
		OrderID:    req.OrderID,
		Status:     paykit.StatusPending,
		Amount:     req.Amount,
		PaymentURL: p.payURL + "?" + q.Encode(),
		Provider:   "robokassa",
		CreatedAt:  time.Now().UTC(),
	}, nil
}

type opStateResp struct {
	Result struct {
		Code int `xml:"Code"`
	} `xml:"Result"`
	State struct {
		Code int `xml:"Code"`
	} `xml:"State"`
}

func (p *Provider) GetPayment(ctx context.Context, invID string) (*paykit.Payment, error) {
	sig := md5hex(fmt.Sprintf("%s:%s:%s", p.login, invID, p.pass2))
	q := url.Values{
		"MerchantLogin": {p.login},
		"InvoiceID":     {invID},
		"Signature":     {sig},
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, stateAPIURL+"?"+q.Encode(), nil)
	resp, err := p.cfg.HTTPClient.Do(req)
	if err != nil {
		return nil, &paykit.ProviderError{Provider: "robokassa", Message: err.Error()}
	}
	defer resp.Body.Close()

	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &paykit.ProviderError{Provider: "robokassa", Message: err.Error()}
	}
	var state opStateResp
	if err := xml.Unmarshal(buf, &state); err != nil {
		return nil, fmt.Errorf("robokassa: parse state response: %w", err)
	}
	return &paykit.Payment{
		ID:       invID,
		OrderID:  invID,
		Status:   mapStateCode(state.State.Code),
		Provider: "robokassa",
		Raw:      buf,
	}, nil
}

func (p *Provider) CapturePayment(_ context.Context, _ string, _ *paykit.Money) (*paykit.Payment, error) {
	return nil, errors.New("robokassa: CapturePayment is not supported")
}

func (p *Provider) CancelPayment(_ context.Context, _ string) (*paykit.Payment, error) {
	return nil, errors.New("robokassa: CancelPayment is not supported")
}

func (p *Provider) Refund(_ context.Context, _ string, _ *paykit.Money) (*paykit.Refund, error) {
	return nil, errors.New("robokassa: Refund must be performed manually in the merchant cabinet")
}

func md5hex(s string) string {
	sum := md5.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}

func mapStateCode(code int) paykit.PaymentStatus {
	switch code {
	case 100:
		return paykit.StatusSucceeded
	case 50:
		return paykit.StatusPending
	case 60:
		return paykit.StatusRefunded
	case 80:
		return paykit.StatusCanceled
	}
	return paykit.StatusPending
}
