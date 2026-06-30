package robokassa

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/laschenkov67/paykit"
	"github.com/laschenkov67/paykit/internal/signing"
)

func (p *Provider) ParseWebhook(r *http.Request) (*paykit.WebhookEvent, error) {
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("%w: %s", paykit.ErrInvalidRequest, err)
	}
	outSum := r.Form.Get("OutSum")
	invID := r.Form.Get("InvId")
	sig := strings.ToLower(r.Form.Get("SignatureValue"))
	if outSum == "" || invID == "" || sig == "" {
		return nil, paykit.ErrInvalidRequest
	}
	want := md5hex(fmt.Sprintf("%s:%s:%s", outSum, invID, p.pass2))
	if !signing.EqualHex(want, sig) {
		return nil, paykit.ErrInvalidSignature
	}
	amount, _ := paykit.ParseMajor(outSum, "RUB")
	pay := &paykit.Payment{
		ID:       invID,
		OrderID:  invID,
		Status:   paykit.StatusSucceeded,
		Amount:   amount,
		Provider: "robokassa",
	}
	return &paykit.WebhookEvent{
		Type: paykit.EventPaymentSucceeded, Provider: "robokassa", Payment: pay,
	}, nil
}

func AckResponse(invID string) string { return "OK" + invID }

func (p *Provider) WebhookAck(ev *paykit.WebhookEvent) (int, []byte) {
	invID := ""
	if ev != nil && ev.Payment != nil {
		invID = ev.Payment.OrderID
	}
	return http.StatusOK, []byte(AckResponse(invID))
}
