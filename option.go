package go_nova

import (
	"crypto/rsa"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/stremovskyy/go-nova/consts"
	"github.com/stremovskyy/go-nova/internal/signature"
	"github.com/stremovskyy/go-nova/log"
	"github.com/stremovskyy/recorder"
)

type Option func(*config) error

type config struct {
	acquiringBaseURL  string
	checkoutBaseURL   string
	comfortBaseURL    string
	comfortMerchantID string

	httpClient *http.Client
	logger     log.Logger
	logBodies  bool

	retryAttempts int
	retryWait     time.Duration
	recorder      recorder.Recorder

	externalSigner *signature.RSASigner
	comfortSigner  *signature.RSASigner
}

func defaultConfig() config {
	return config{
		acquiringBaseURL: consts.DefaultAcquiringBaseURL,
		checkoutBaseURL:  consts.DefaultAcquiringBaseURL,
		comfortBaseURL:   consts.DefaultComfortBaseURL,
		httpClient:       &http.Client{Timeout: 30 * time.Second},
		logger:           log.NewDefault(),
		retryAttempts:    1,
		retryWait:        300 * time.Millisecond,
		// External API docs use SHA-256.
		externalSigner: &signature.RSASigner{Hash: signature.HashSHA256},
		// Comfort API docs use SHA-1.
		comfortSigner: &signature.RSASigner{Hash: signature.HashSHA1},
	}
}

// WithHTTPClient sets a custom *http.Client.
func WithHTTPClient(client *http.Client) Option {
	return func(cfg *config) error {
		if client == nil {
			return errors.New("http client is nil")
		}
		cfg.httpClient = client
		return nil
	}
}

// WithClient is an alias of WithHTTPClient, kept for API parity with go-ipay.
func WithClient(client *http.Client) Option {
	return WithHTTPClient(client)
}

// WithTimeout sets http client timeout.
func WithTimeout(timeout time.Duration) Option {
	return func(cfg *config) error {
		if timeout <= 0 {
			return errors.New("timeout must be > 0")
		}
		cfg.httpClient.Timeout = timeout
		return nil
	}
}

func WithLogger(logger log.Logger) Option {
	return func(cfg *config) error {
		if logger == nil {
			cfg.logger = log.NopLogger{}
			return nil
		}
		cfg.logger = logger
		return nil
	}
}

// WithLogHTTPBodies enables verbose request/response body logging for debugging.
//
// Disabled by default because bodies may contain sensitive data.
func WithLogHTTPBodies(enabled bool) Option {
	return func(cfg *config) error {
		cfg.logBodies = enabled
		return nil
	}
}

// WithRecorder attaches a recorder, similar to go-ipay.
func WithRecorder(r recorder.Recorder) Option {
	return func(cfg *config) error {
		cfg.recorder = r
		return nil
	}
}

func WithRetry(attempts int, wait time.Duration) Option {
	return func(cfg *config) error {
		if attempts <= 0 {
			return errors.New("retry attempts must be > 0")
		}
		if wait <= 0 {
			return errors.New("retry wait must be > 0")
		}
		cfg.retryAttempts = attempts
		cfg.retryWait = wait
		return nil
	}
}

func WithAcquiringBaseURL(baseURL string) Option {
	return func(cfg *config) error {
		if baseURL == "" {
			return errors.New("acquiring base url is empty")
		}
		cfg.acquiringBaseURL = baseURL
		return nil
	}
}

func WithCheckoutBaseURL(baseURL string) Option {
	return func(cfg *config) error {
		if baseURL == "" {
			return errors.New("checkout base url is empty")
		}
		cfg.checkoutBaseURL = baseURL
		return nil
	}
}

func WithComfortBaseURL(baseURL string) Option {
	return func(cfg *config) error {
		if baseURL == "" {
			return errors.New("comfort base url is empty")
		}
		cfg.comfortBaseURL = baseURL
		return nil
	}
}

// WithComfortMerchantID sets x-merchant-id header value used for Comfort API requests.
func WithComfortMerchantID(merchantID string) Option {
	return func(cfg *config) error {
		merchantID = strings.TrimSpace(merchantID)
		if merchantID == "" {
			return errors.New("comfort merchant id is empty")
		}
		cfg.comfortMerchantID = merchantID
		return nil
	}
}

// WithSignatureHash sets the hash algorithm used for x-sign for all APIs.
//
// Kept for backwards compatibility. Prefer API-specific hash options.
func WithSignatureHash(hash signature.HashAlgorithm) Option {
	return func(cfg *config) error {
		cfg.externalSigner.Hash = hash
		cfg.comfortSigner.Hash = hash
		return nil
	}
}

// WithExternalSignatureHash sets the hash algorithm used for Acquiring/Checkout x-sign.
func WithExternalSignatureHash(hash signature.HashAlgorithm) Option {
	return func(cfg *config) error {
		cfg.externalSigner.Hash = hash
		return nil
	}
}

// WithComfortSignatureHash sets the hash algorithm used for Comfort x-sign.
func WithComfortSignatureHash(hash signature.HashAlgorithm) Option {
	return func(cfg *config) error {
		cfg.comfortSigner.Hash = hash
		return nil
	}
}

// WithPrivateKeyPEM configures the RSA private key used to sign requests.
func WithPrivateKeyPEM(pemBytes []byte) Option {
	return func(cfg *config) error {
		k, err := signature.ParseRSAPrivateKeyPEM(pemBytes)
		if err != nil {
			return err
		}
		cfg.externalSigner.PrivateKey = k
		cfg.comfortSigner.PrivateKey = k
		return nil
	}
}

// WithPrivateKeyFile reads a PEM file and sets it as the signing key.
func WithPrivateKeyFile(path string) Option {
	return func(cfg *config) error {
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		k, err := signature.ParseRSAPrivateKeyPEM(b)
		if err != nil {
			return err
		}
		cfg.externalSigner.PrivateKey = k
		cfg.comfortSigner.PrivateKey = k
		return nil
	}
}

// WithPublicKeyPEM configures the RSA public key used to verify incoming signatures.
func WithPublicKeyPEM(pemBytes []byte) Option {
	return func(cfg *config) error {
		k, err := signature.ParseRSAPublicKeyPEM(pemBytes)
		if err != nil {
			return err
		}
		cfg.externalSigner.PublicKey = k
		cfg.comfortSigner.PublicKey = k
		return nil
	}
}

// WithPublicKeyFile reads a PEM file and sets it as the verification key.
func WithPublicKeyFile(path string) Option {
	return func(cfg *config) error {
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		k, err := signature.ParseRSAPublicKeyPEM(b)
		if err != nil {
			return err
		}
		cfg.externalSigner.PublicKey = k
		cfg.comfortSigner.PublicKey = k
		return nil
	}
}

// WithPrivateKey allows setting already parsed RSA private key.
func WithPrivateKey(key *rsa.PrivateKey) Option {
	return func(cfg *config) error {
		if key == nil {
			return errors.New("private key is nil")
		}
		cfg.externalSigner.PrivateKey = key
		cfg.comfortSigner.PrivateKey = key
		return nil
	}
}
