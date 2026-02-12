package main

import (
	"encoding/json"
	"io"
	stdlog "log"
	"net/http"
	"os"

	go_nova "github.com/stremovskyy/go-nova"
	"github.com/stremovskyy/go-nova/acquiring"
)

func main() {
	publicKeyPath := os.Getenv("NOVAPAY_PUBLIC_KEY_PATH")
	if publicKeyPath == "" {
		stdlog.Fatal("set NOVAPAY_PUBLIC_KEY_PATH to NovaPay public key PEM path")
	}

	client, err := go_nova.NewClient(
		go_nova.WithPublicKeyFile(publicKeyPath),
	)
	if err != nil {
		stdlog.Fatal(err)
	}

	http.HandleFunc("/novapay/callback", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		xSign := r.Header.Get("x-sign")
		if err := client.Verify(body, xSign); err != nil {
			http.Error(w, "invalid signature", http.StatusUnauthorized)
			return
		}

		var pb acquiring.Postback
		if err := json.Unmarshal(body, &pb); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		totalAmount := 0.0
		for _, p := range pb.Payments {
			totalAmount += p.Amount
		}

		stdlog.Printf("postback: id=%s status=%s paytype=%s amount=%.2f", pb.ID, pb.Status, pb.Paytype, totalAmount)
		w.WriteHeader(http.StatusOK)
	})

	stdlog.Println("listening on :8080")
	stdlog.Fatal(http.ListenAndServe(":8080", nil))
}
