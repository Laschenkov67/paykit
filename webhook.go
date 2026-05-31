package paykit

import "encoding/json"

type EventType string

const (
	EventPaymentPending   EventType = "payment.pending"
	EventPaymentSucceeded EventType = "payment.succeeded"
	EventPaymentFailed    EventType = "payment.failed"
	EventPaymentCanceled  EventType = "payment.canceled"
	EventRefundSucceeded  EventType = "refund.succeeded"
	EventUnknown          EventType = "unknown"
)

type WebhookEvent struct {
	Type     EventType
	Provider string
	Payment  *Payment // populated for payment.* events
	Refund   *Refund  // populated for refund.* events
	Raw      json.RawMessage
}
