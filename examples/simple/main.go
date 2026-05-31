package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/laschenkov67/paykit"
	"github.com/laschenkov67/paykit/providers/yookassa"
)

func main() {
	yk, err := yookassa.New(os.Getenv("YK_SHOP_ID"), os.Getenv("YK_SECRET"))
	if err != nil {
		log.Fatal(err)
	}
	pay, err := yk.CreatePayment(context.Background(), paykit.CreatePaymentRequest{
		OrderID:     "demo-1",
		Amount:      paykit.RUB(199_00),
		Description: "Demo payment",
		ReturnURL:   "https://example.com/return",
		Metadata:    map[string]string{"source": "demo"},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Pay here: %s\n", pay.PaymentURL)
}
