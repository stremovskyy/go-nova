package main

import (
	"context"
	"fmt"
	stdlog "log"
	"os"

	go_nova "github.com/stremovskyy/go-nova"
	"github.com/stremovskyy/go-nova/acquiring"
	"github.com/stremovskyy/go-nova/examples/internal/dotenv"
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
	)
	if err != nil {
		stdlog.Fatal(err)
	}

	req := &acquiring.CreateSessionRequest{
		MerchantID:  "1",
		ClientPhone: "+380982850620",
	}
	callbackURL := "https://example.com/novapay/callback"
	req.CallbackURL = &callbackURL

	res, err := client.Acquiring().CreateSession(context.Background(), req)
	if err != nil {
		stdlog.Fatal(err)
	}

	fmt.Println("Session ID:", res.ID)
}
