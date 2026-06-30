package paykit

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

type Provider interface {
	Name() string

	CreatePayment(ctx context.Context, req CreatePaymentRequest) (*Payment, error)

	GetPayment(ctx context.Context, id string) (*Payment, error)

	CapturePayment(ctx context.Context, id string, amount *Money) (*Payment, error)

	CancelPayment(ctx context.Context, id string) (*Payment, error)

	Refund(ctx context.Context, paymentID string, amount *Money) (*Refund, error)

	ParseWebhook(r *http.Request) (*WebhookEvent, error)

	// WebhookAck returns the HTTP status code and response body the provider
	// expects after a webhook notification has been successfully processed.
	// PSPs that don't see their required acknowledgement (e.g. a literal "OK"
	// body) will keep retrying the same notification, so a generic 200 with
	// an empty body is not safe for every provider. ev is the event returned
	// by the preceding ParseWebhook call.
	WebhookAck(ev *WebhookEvent) (statusCode int, body []byte)
}

type CreatePaymentRequest struct {
	OrderID        string            // your internal order identifier
	Amount         Money             // amount + ISO-4217 currency
	Description    string            // human-readable payment purpose
	Customer       *Customer         // optional customer info (email / phone)
	ReturnURL      string            // where the user is redirected after payment
	Metadata       map[string]string // arbitrary key/value, persisted by PSP
	TwoStage       bool              // false = one-stage, capture immediately (default); true = authorize only, settle later via CapturePayment
	IdempotencyKey string            // optional; auto-generated when empty
}

type Customer struct {
	ID    string
	Email string
	Phone string
	Name  string
}

type Refund struct {
	ID        string
	PaymentID string
	Amount    Money
	Status    PaymentStatus
	CreatedAt time.Time
	Provider  string
	Raw       json.RawMessage
}
