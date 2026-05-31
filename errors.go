package paykit

import "errors"

var (
	ErrInvalidSignature = errors.New("paykit: invalid webhook signature")
	ErrInvalidRequest   = errors.New("paykit: invalid request")
	ErrNotFound         = errors.New("paykit: payment not found")
	ErrUnauthorized     = errors.New("paykit: unauthorized (bad credentials)")
	ErrProvider         = errors.New("paykit: provider error")
)

type ProviderError struct {
	Provider   string
	StatusCode int
	Code       string // PSP-specific code if any
	Message    string
	Body       []byte
}

func (e *ProviderError) Error() string {
	if e.Code != "" {
		return "paykit: " + e.Provider + ": " + e.Code + ": " + e.Message
	}
	return "paykit: " + e.Provider + ": " + e.Message
}

func (e *ProviderError) Unwrap() error { return ErrProvider }
