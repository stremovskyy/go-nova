package go_nova

import "github.com/stremovskyy/go-nova/log"

// Nova is the main SDK interface, mirroring the top-level style used in go-ipay.
type Nova interface {
	Acquiring() *AcquiringService
	Comfort() *ComfortService
	Checkout() *CheckoutService

	Sign(body []byte) (string, error)
	SignComfort(body []byte) (string, error)
	Verify(body []byte, xSign string) error
	VerifyComfort(body []byte, xSign string) error

	SetLogLevel(level log.Level)
}

var _ Nova = (*Client)(nil)
