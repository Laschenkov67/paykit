package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/laschenkov67/paykit"
	"github.com/laschenkov67/paykit/providers/stripe"
	"github.com/laschenkov67/paykit/providers/yookassa"
)

func main() {
	mgr := paykit.NewManager()

	yk, _ := yookassa.New(os.Getenv("YK_SHOP_ID"), os.Getenv("YK_SECRET"))
	mgr.Register(yk)

	st, _ := stripe.New(os.Getenv("STRIPE_KEY"), os.Getenv("STRIPE_WHSEC"))
	mgr.Register(st)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /webhook/{provider}", mgr.HandleWebhook(func(ev *paykit.WebhookEvent) {
		fmt.Printf("event=%s provider=%s payment=%+v\n", ev.Type, ev.Provider, ev.Payment)
		// ... here you update your DB, emit domain events, etc.
	}))

	log.Println("listening :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
