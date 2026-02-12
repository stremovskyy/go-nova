# go-nova

Go SDK for NovaPay APIs:

- Acquiring API
- Checkout API
- Comfort API

The client signs outgoing requests (`x-sign`) and can verify incoming callback signatures.

## Installation

```bash
go get github.com/stremovskyy/go-nova@latest
```

## Requirements

- Go `1.23+`
- NovaPay merchant private RSA key (for request signing)
- NovaPay public RSA key (for callback signature verification)
- Comfort merchant id for Comfort API (`x-merchant-id`)

## Quick Start (Acquiring)

```go
package main

import (
	"context"
	"fmt"
	"log"

	go_nova "github.com/stremovskyy/go-nova"
	"github.com/stremovskyy/go-nova/acquiring"
)

func main() {
	client, err := go_nova.NewClient(
		go_nova.WithPrivateKeyFile("./merchant-private.pem"),
	)
	if err != nil {
		log.Fatal(err)
	}

	callbackURL := "https://example.com/novapay/callback"
	session, err := client.Acquiring().CreateSession(context.Background(), &acquiring.CreateSessionRequest{
		MerchantID:  "1",
		ClientPhone: "+380670000000",
		CallbackURL: &callbackURL,
	})
	if err != nil {
		log.Fatal(err)
	}

	payment, err := client.Acquiring().AddPayment(context.Background(), &acquiring.AddPaymentRequest{
		MerchantID: "1",
		SessionID:  session.ID,
		Amount:     100.50,
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Payment URL:", payment.URL)
}
```

## Verify Callback Signature

```go
body := []byte(`{"id":"..."}`)
xSign := "signature-from-x-sign-header"

client, err := go_nova.NewClient(
	go_nova.WithPublicKeyFile("./novapay-public.pem"),
)
if err != nil {
	log.Fatal(err)
}

if err := client.Verify(body, xSign); err != nil {
	// invalid signature
}
```

## Services

### Acquiring

- `CreateSession`
- `AddPayment`
- `VoidSession`
- `CompleteHold`
- `ExpireSession`
- `ConfirmDeliveryHold`
- `PrintExpressWaybill`
- `GetStatus`
- `DeliveryPrice`
- `Do` (manual signed call)

### Checkout

- `CreateSession`
- `AddPayment`
- `VoidSession`
- `GetStatus`
- `ExpireSession`
- `Do` (manual signed call)

### Comfort

Requires `go_nova.WithComfortMerchantID("...")`.

- `CreateOperations`
- `RefundOperations`
- `OperationsStatus`
- `ChangeRecipientData`
- `Balance`
- `ExportOperations`
- `Do` (manual signed call)

## Configuration Options

Common options:

- `WithPrivateKeyFile` / `WithPrivateKeyPEM`
- `WithPublicKeyFile` / `WithPublicKeyPEM`
- `WithTimeout`
- `WithRetry`
- `WithHTTPClient`
- `WithLogger`

Base URLs:

- `WithAcquiringBaseURL`
- `WithCheckoutBaseURL`
- `WithComfortBaseURL`

Useful constants:

- test acquiring URL: `consts.DefaultAcquiringBaseURL`
- production acquiring URL: `consts.ProductionAcquiringURL`
- default comfort URL: `consts.DefaultComfortBaseURL`

## Dry Run Mode

You can skip HTTP requests and inspect payloads:

```go
_, _ = client.Acquiring().CreateSession(ctx, req, go_nova.DryRun())
```

## Errors

- `*go_nova.ValidationError`: invalid or missing request fields
- `*go_nova.APIError`: non-2xx API response with status/body

## Examples

Run examples from repository root:

```bash
export NOVAPAY_PRIVATE_KEY_PATH=/absolute/path/to/merchant-private.pem
export NOVAPAY_PUBLIC_KEY_PATH=/absolute/path/to/novapay-public.pem

go run ./examples/acquiring_create_session
go run ./examples/acquiring_add_payment
go run ./examples/verify_postback
```

## Development

```bash
go test ./...
```

## Security Notes

- Never commit real private keys to git.
- Keep credentials in environment variables or secret managers.

## License

[MIT](LICENSE)
