package main

import (
	"context"
	"fmt"
	stdlog "log"
	"os"

	go_nova "github.com/stremovskyy/go-nova"
	"github.com/stremovskyy/go-nova/acquiring"
	"github.com/stremovskyy/go-nova/log"
)

func main() {
	privateKeyPath := os.Getenv("NOVAPAY_PRIVATE_KEY_PATH")
	if privateKeyPath == "" {
		stdlog.Fatal("set NOVAPAY_PRIVATE_KEY_PATH to your merchant private key PEM path")
	}

	client, err := go_nova.NewClient(
		go_nova.WithPrivateKeyFile(privateKeyPath),
	)
	if err != nil {
		stdlog.Fatal(err)
	}

	client.SetLogLevel(log.LevelDebug)

	ctx := context.Background()

	// 1) Create session
	callbackURL := "https://webhook.site/e7048bac-3cbd-4b77-ac00-7b625add5dd8"
	session, err := client.Acquiring().CreateSession(
		ctx, &acquiring.CreateSessionRequest{
			MerchantID:  "1",
			ClientPhone: "+38068365465",
			CallbackURL: &callbackURL,
		},
	)
	if err != nil {
		stdlog.Fatal(err)
	}

	// 2) Add payment and get payment URL
	payment, err := client.Acquiring().AddPayment(
		ctx, &acquiring.AddPaymentRequest{
			MerchantID: "1",
			SessionID:  session.ID,
			Amount:     1.25,
		},
	)
	if err != nil {
		stdlog.Fatal(err)
	}

	fmt.Println("Pay URL:", payment.URL)
}
