package paykit

import (
	"encoding/json"
	"time"
)

type PaymentStatus string

const (
	StatusPending    PaymentStatus = "pending"
	StatusAuthorized PaymentStatus = "authorized"
	StatusSucceeded  PaymentStatus = "succeeded"
	StatusCanceled   PaymentStatus = "canceled"
	StatusFailed     PaymentStatus = "failed"
	StatusRefunded   PaymentStatus = "refunded"
)

type Payment struct {
	ID          string
	OrderID     string
	Status      PaymentStatus
	Amount      Money
	PaymentURL  string
	Description string
	Metadata    map[string]string
	CreatedAt   time.Time
	Provider    string
	Raw         json.RawMessage
}

func (p *Payment) IsTerminal() bool {
	switch p.Status {
	case StatusSucceeded, StatusFailed, StatusCanceled, StatusRefunded:
		return true
	}
	return false
}
