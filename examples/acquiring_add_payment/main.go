package main

import (
	"context"
	"fmt"
	stdlog "log"
	"os"

	"github.com/google/uuid"

	go_nova "github.com/stremovskyy/go-nova"
	"github.com/stremovskyy/go-nova/acquiring"
	"github.com/stremovskyy/go-nova/examples/internal/dotenv"
	"github.com/stremovskyy/go-nova/internal/utils"
)

func main() {
	if _, err := dotenv.LoadNearest(".env"); err != nil {
		stdlog.Fatalf("load .env: %v", err)
	}

	privateKeyPath := os.Getenv("NOVAPAY_PRIVATE_KEY_PATH")
	if privateKeyPath == "" {
		stdlog.Fatal("set NOVAPAY_PRIVATE_KEY_PATH to your merchant private key PEM path")
	}

	client, err := go_nova.NewClient(
		go_nova.WithPrivateKeyFile(privateKeyPath),
		go_nova.WithLogHTTPBodies(true), // enable request/response JSON in debug logs
	)
	if err != nil {
		stdlog.Fatal(err)
	}

	// client.SetLogLevel(log.LevelDebug)

	ctx := context.Background()

	// 1) Create session
	callbackURL := "https://webhook.site/130f09be-9e16-4716-88e9-fa3014c38032"
	session, err := client.Acquiring().CreateSession(
		ctx, &acquiring.CreateSessionRequest{
			MerchantID:  "2",
			ClientPhone: "+380982850620",
			CallbackURL: &callbackURL,
		},
	)
	if err != nil {
		stdlog.Fatal(err)
	}

	// 2) Add payment and get payment URL
	payment, err := client.Acquiring().AddPayment(
		ctx, &acquiring.AddPaymentRequest{
			MerchantID: "2",
			SessionID:  session.ID,
			Amount:     1.25,
			ExternalID: utils.Ref(uuid.New().String()),
		},
	)
	if err != nil {
		stdlog.Fatal(err)
	}

	fmt.Println("Pay URL:", payment.URL)
}
