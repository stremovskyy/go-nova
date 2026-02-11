package go_nova

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"path"

	"github.com/stremovskyy/go-nova/acquiring"
	"github.com/stremovskyy/go-nova/checkout"
	"github.com/stremovskyy/go-nova/comfort"
	"github.com/stremovskyy/go-nova/consts"
	"github.com/stremovskyy/go-nova/internal/httpclient"
	"github.com/stremovskyy/go-nova/log"
	"github.com/stremovskyy/recorder"
)

// Client is the main NovaPay SDK client.
//
// It supports:
//   - External API: Acquiring + Checkout
//   - Comfort API
//
// Requests are signed automatically with x-sign.
type Client struct {
	cfg config

	externalHTTP *httpclient.Client
	comfortHTTP  *httpclient.Client

	acquiring *AcquiringService
	comfort   *ComfortService
	checkout  *CheckoutService
}

func NewClient(opts ...Option) (Nova, error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(&cfg); err != nil {
			return nil, err
		}
	}

	comfortHeaders := map[string]string{}
	if cfg.comfortMerchantID != "" {
		comfortHeaders[consts.HeaderXMerchantID] = cfg.comfortMerchantID
	}

	c := &Client{cfg: cfg}
	c.externalHTTP = httpclient.New(cfg.httpClient, cfg.externalSigner, cfg.logger, cfg.retryAttempts, cfg.retryWait, nil, cfg.recorder)
	c.comfortHTTP = httpclient.New(cfg.httpClient, cfg.comfortSigner, cfg.logger, cfg.retryAttempts, cfg.retryWait, comfortHeaders, cfg.recorder)

	c.acquiring = &AcquiringService{c: c}
	c.comfort = &ComfortService{c: c}
	c.checkout = &CheckoutService{c: c}
	return c, nil
}

// NewDefaultClient is a convenience wrapper around NewClient() with default configuration.
func NewDefaultClient() (Nova, error) {
	return NewClient()
}

// NewClientWithRecorder mirrors go-ipay style constructor and attaches recorder.
func NewClientWithRecorder(rec recorder.Recorder, opts ...Option) (Nova, error) {
	opts = append([]Option{WithRecorder(rec)}, opts...)
	return NewClient(opts...)
}

func (c *Client) Acquiring() *AcquiringService { return c.acquiring }
func (c *Client) Comfort() *ComfortService     { return c.comfort }
func (c *Client) Checkout() *CheckoutService   { return c.checkout }

// SetLogLevel updates SDK log level when current logger supports it.
func (c *Client) SetLogLevel(level log.Level) {
	if c == nil || c.cfg.logger == nil {
		return
	}
	if l, ok := c.cfg.logger.(interface{ SetLevel(log.Level) }); ok {
		l.SetLevel(level)
	}
}

// Sign signs request payload for External API (Acquiring/Checkout).
func (c *Client) Sign(body []byte) (string, error) {
	if c == nil || c.cfg.externalSigner == nil {
		return "", errors.New("client is not initialized")
	}
	return c.cfg.externalSigner.Sign(body)
}

// SignComfort signs request payload for Comfort API.
func (c *Client) SignComfort(body []byte) (string, error) {
	if c == nil || c.cfg.comfortSigner == nil {
		return "", errors.New("client is not initialized")
	}
	return c.cfg.comfortSigner.Sign(body)
}

// Verify verifies x-sign using configured external public key.
func (c *Client) Verify(body []byte, xSign string) error {
	if c == nil || c.cfg.externalSigner == nil {
		return errors.New("client is not initialized")
	}
	return c.cfg.externalSigner.Verify(body, xSign)
}

// VerifyComfort verifies x-sign using configured comfort public key.
func (c *Client) VerifyComfort(body []byte, xSign string) error {
	if c == nil || c.cfg.comfortSigner == nil {
		return errors.New("client is not initialized")
	}
	return c.cfg.comfortSigner.Verify(body, xSign)
}

func joinURL(base string, p string) (string, error) {
	u, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("invalid base url %q: %w", base, err)
	}
	u.Path = path.Join(u.Path, p)
	return u.String(), nil
}

func wrapAPIError(err error) error {
	if err == nil {
		return nil
	}
	var hs *httpclient.HTTPStatusError
	if errors.As(err, &hs) {
		return &APIError{StatusCode: hs.StatusCode, Body: hs.Body}
	}
	return err
}

func ensureComfortReady(c *Client) error {
	if c == nil {
		return errors.New("client is nil")
	}
	if c.cfg.comfortMerchantID == "" {
		return errors.New("comfort merchant id is not configured; use WithComfortMerchantID(...)")
	}
	return nil
}

// =========================
// Acquiring (External API)
// =========================

type AcquiringService struct{ c *Client }

// CreateSession creates a payment session.
func (s *AcquiringService) CreateSession(ctx context.Context, req *acquiring.CreateSessionRequest, runOpts ...RunOption) (*acquiring.CreateSessionResponse, error) {
	if s == nil || s.c == nil {
		return nil, errors.New("client is nil")
	}
	if req == nil {
		return nil, &ValidationError{Fields: []FieldError{{Field: "request", Message: "is nil"}}}
	}
	if err := validateCreateSession(req); err != nil {
		return nil, err
	}

	full, err := joinURL(s.c.cfg.acquiringBaseURL, consts.AcquiringCreateSessionPath)
	if err != nil {
		return nil, err
	}
	if shouldDryRun(runOpts, "POST", full, req) {
		return nil, nil
	}
	var out acquiring.CreateSessionResponse
	_, _, err = s.c.externalHTTP.DoJSON(ctx, "POST", full, req, &out)
	if err != nil {
		return nil, wrapAPIError(err)
	}
	return &out, nil
}

// AddPayment adds order information and returns payment URL.
func (s *AcquiringService) AddPayment(ctx context.Context, req *acquiring.AddPaymentRequest, runOpts ...RunOption) (*acquiring.AddPaymentResponse, error) {
	if s == nil || s.c == nil {
		return nil, errors.New("client is nil")
	}
	if req == nil {
		return nil, &ValidationError{Fields: []FieldError{{Field: "request", Message: "is nil"}}}
	}
	if err := validateAddPayment(req); err != nil {
		return nil, err
	}

	full, err := joinURL(s.c.cfg.acquiringBaseURL, consts.AcquiringAddPaymentPath)
	if err != nil {
		return nil, err
	}
	if shouldDryRun(runOpts, "POST", full, req) {
		return nil, nil
	}
	var out acquiring.AddPaymentResponse
	_, _, err = s.c.externalHTTP.DoJSON(ctx, "POST", full, req, &out)
	if err != nil {
		return nil, wrapAPIError(err)
	}
	return &out, nil
}

// VoidSession voids or refunds blocked/charged funds.
func (s *AcquiringService) VoidSession(ctx context.Context, req *acquiring.SessionRequest, runOpts ...RunOption) error {
	if s == nil || s.c == nil {
		return errors.New("client is nil")
	}
	if req == nil {
		return &ValidationError{Fields: []FieldError{{Field: "request", Message: "is nil"}}}
	}
	if err := validateSessionRequest(req); err != nil {
		return err
	}

	full, err := joinURL(s.c.cfg.acquiringBaseURL, consts.AcquiringVoidSessionPath)
	if err != nil {
		return err
	}
	if shouldDryRun(runOpts, "POST", full, req) {
		return nil
	}
	_, _, err = s.c.externalHTTP.DoJSON(ctx, "POST", full, req, nil)
	return wrapAPIError(err)
}

// CompleteHold confirms previously blocked funds.
func (s *AcquiringService) CompleteHold(ctx context.Context, req *acquiring.CompleteHoldRequest, runOpts ...RunOption) error {
	if s == nil || s.c == nil {
		return errors.New("client is nil")
	}
	if req == nil {
		return &ValidationError{Fields: []FieldError{{Field: "request", Message: "is nil"}}}
	}
	if err := validateCompleteHold(req); err != nil {
		return err
	}

	full, err := joinURL(s.c.cfg.acquiringBaseURL, consts.AcquiringCompleteHoldPath)
	if err != nil {
		return err
	}
	if shouldDryRun(runOpts, "POST", full, req) {
		return nil
	}
	_, _, err = s.c.externalHTTP.DoJSON(ctx, "POST", full, req, nil)
	return wrapAPIError(err)
}

// ExpireSession force-expires a payment session.
func (s *AcquiringService) ExpireSession(ctx context.Context, req *acquiring.SessionRequest, runOpts ...RunOption) error {
	if s == nil || s.c == nil {
		return errors.New("client is nil")
	}
	if req == nil {
		return &ValidationError{Fields: []FieldError{{Field: "request", Message: "is nil"}}}
	}
	if err := validateSessionRequest(req); err != nil {
		return err
	}

	full, err := joinURL(s.c.cfg.acquiringBaseURL, consts.AcquiringExpireSessionPath)
	if err != nil {
		return err
	}
	if shouldDryRun(runOpts, "POST", full, req) {
		return nil
	}
	_, _, err = s.c.externalHTTP.DoJSON(ctx, "POST", full, req, nil)
	return wrapAPIError(err)
}

// ConfirmDeliveryHold confirms protected payment based on delivery status.
func (s *AcquiringService) ConfirmDeliveryHold(ctx context.Context, req *acquiring.SessionRequest, runOpts ...RunOption) (*acquiring.ConfirmDeliveryHoldResponse, error) {
	if s == nil || s.c == nil {
		return nil, errors.New("client is nil")
	}
	if req == nil {
		return nil, &ValidationError{Fields: []FieldError{{Field: "request", Message: "is nil"}}}
	}
	if err := validateSessionRequest(req); err != nil {
		return nil, err
	}

	full, err := joinURL(s.c.cfg.acquiringBaseURL, consts.AcquiringConfirmDeliveryPath)
	if err != nil {
		return nil, err
	}
	if shouldDryRun(runOpts, "POST", full, req) {
		return nil, nil
	}
	var out acquiring.ConfirmDeliveryHoldResponse
	_, _, err = s.c.externalHTTP.DoJSON(ctx, "POST", full, req, &out)
	if err != nil {
		return nil, wrapAPIError(err)
	}
	return &out, nil
}

// PrintExpressWaybill returns express waybill file stream.
func (s *AcquiringService) PrintExpressWaybill(ctx context.Context, req *acquiring.SessionRequest, runOpts ...RunOption) ([]byte, error) {
	if s == nil || s.c == nil {
		return nil, errors.New("client is nil")
	}
	if req == nil {
		return nil, &ValidationError{Fields: []FieldError{{Field: "request", Message: "is nil"}}}
	}
	if err := validateSessionRequest(req); err != nil {
		return nil, err
	}

	full, err := joinURL(s.c.cfg.acquiringBaseURL, consts.AcquiringPrintExpressWaybillPath)
	if err != nil {
		return nil, err
	}
	if shouldDryRun(runOpts, "POST", full, req) {
		return nil, nil
	}
	_, raw, err := s.c.externalHTTP.DoJSON(ctx, "POST", full, req, nil)
	if err != nil {
		return nil, wrapAPIError(err)
	}
	return raw, nil
}

// GetStatus returns current session status/details.
func (s *AcquiringService) GetStatus(ctx context.Context, req *acquiring.SessionRequest, runOpts ...RunOption) (*acquiring.GetStatusResponse, error) {
	if s == nil || s.c == nil {
		return nil, errors.New("client is nil")
	}
	if req == nil {
		return nil, &ValidationError{Fields: []FieldError{{Field: "request", Message: "is nil"}}}
	}
	if err := validateSessionRequest(req); err != nil {
		return nil, err
	}

	full, err := joinURL(s.c.cfg.acquiringBaseURL, consts.AcquiringGetStatusPath)
	if err != nil {
		return nil, err
	}
	if shouldDryRun(runOpts, "POST", full, req) {
		return nil, nil
	}
	var out acquiring.GetStatusResponse
	_, _, err = s.c.externalHTTP.DoJSON(ctx, "POST", full, req, &out)
	if err != nil {
		return nil, wrapAPIError(err)
	}
	return &out, nil
}

// DeliveryPrice calculates delivery price.
func (s *AcquiringService) DeliveryPrice(ctx context.Context, req *acquiring.DeliveryPriceRequest, runOpts ...RunOption) (acquiring.DeliveryPriceResponse, error) {
	if s == nil || s.c == nil {
		return nil, errors.New("client is nil")
	}
	if req == nil {
		return nil, &ValidationError{Fields: []FieldError{{Field: "request", Message: "is nil"}}}
	}
	if err := validateDeliveryPrice(req); err != nil {
		return nil, err
	}

	full, err := joinURL(s.c.cfg.acquiringBaseURL, consts.AcquiringDeliveryPricePath)
	if err != nil {
		return nil, err
	}
	if shouldDryRun(runOpts, "POST", full, req) {
		return nil, nil
	}
	var out acquiring.DeliveryPriceResponse
	_, _, err = s.c.externalHTTP.DoJSON(ctx, "POST", full, req, &out)
	if err != nil {
		return nil, wrapAPIError(err)
	}
	return out, nil
}

// Do performs a signed request against Acquiring base URL.
func (s *AcquiringService) Do(ctx context.Context, method string, endpointPath string, body any, out any, runOpts ...RunOption) error {
	if s == nil || s.c == nil {
		return errors.New("client is nil")
	}
	full, err := joinURL(s.c.cfg.acquiringBaseURL, endpointPath)
	if err != nil {
		return err
	}
	if shouldDryRun(runOpts, method, full, body) {
		return nil
	}
	_, _, err = s.c.externalHTTP.DoJSON(ctx, method, full, body, out)
	return wrapAPIError(err)
}

// =========================
// Comfort API
// =========================

type ComfortService struct{ c *Client }

// CreateOperations sends payout instructions.
func (s *ComfortService) CreateOperations(ctx context.Context, req comfort.CreateOperationsRequest, runOpts ...RunOption) ([]comfort.CreateOperationsResponseItem, error) {
	if s == nil || s.c == nil {
		return nil, errors.New("client is nil")
	}
	if err := ensureComfortReady(s.c); err != nil {
		return nil, err
	}
	if err := validateComfortCreateOperations(req); err != nil {
		return nil, err
	}

	full, err := joinURL(s.c.cfg.comfortBaseURL, consts.ComfortCreateOperationsPath)
	if err != nil {
		return nil, err
	}
	if shouldDryRun(runOpts, "POST", full, req) {
		return nil, nil
	}
	var out []comfort.CreateOperationsResponseItem
	_, _, err = s.c.comfortHTTP.DoJSON(ctx, "POST", full, req, &out)
	if err != nil {
		return nil, wrapAPIError(err)
	}
	return out, nil
}

// RefundOperations requests operation refund by public IDs.
func (s *ComfortService) RefundOperations(ctx context.Context, req *comfort.RefundOperationsRequest, runOpts ...RunOption) ([]string, error) {
	if s == nil || s.c == nil {
		return nil, errors.New("client is nil")
	}
	if err := ensureComfortReady(s.c); err != nil {
		return nil, err
	}
	if err := validateComfortRefundOperations(req); err != nil {
		return nil, err
	}

	full, err := joinURL(s.c.cfg.comfortBaseURL, consts.ComfortRefundOperationsPath)
	if err != nil {
		return nil, err
	}
	if shouldDryRun(runOpts, "POST", full, req) {
		return nil, nil
	}
	var out []string
	_, _, err = s.c.comfortHTTP.DoJSON(ctx, "POST", full, req, &out)
	if err != nil {
		return nil, wrapAPIError(err)
	}
	return out, nil
}

// OperationsStatus checks status by operation GUID.
func (s *ComfortService) OperationsStatus(ctx context.Context, req *comfort.OperationsStatusRequest, runOpts ...RunOption) (*comfort.OperationsStatusResponse, error) {
	if s == nil || s.c == nil {
		return nil, errors.New("client is nil")
	}
	if err := ensureComfortReady(s.c); err != nil {
		return nil, err
	}
	if req == nil {
		req = &comfort.OperationsStatusRequest{}
	}

	full, err := joinURL(s.c.cfg.comfortBaseURL, consts.ComfortOperationsStatusPath)
	if err != nil {
		return nil, err
	}
	if shouldDryRun(runOpts, "POST", full, req) {
		return nil, nil
	}
	var out comfort.OperationsStatusResponse
	_, _, err = s.c.comfortHTTP.DoJSON(ctx, "POST", full, req, &out)
	if err != nil {
		return nil, wrapAPIError(err)
	}
	return &out, nil
}

// ChangeRecipientData updates recipient data for operation.
func (s *ComfortService) ChangeRecipientData(ctx context.Context, req *comfort.ChangeRecipientDataRequest, runOpts ...RunOption) error {
	if s == nil || s.c == nil {
		return errors.New("client is nil")
	}
	if err := ensureComfortReady(s.c); err != nil {
		return err
	}
	if err := validateComfortChangeRecipientData(req); err != nil {
		return err
	}

	full, err := joinURL(s.c.cfg.comfortBaseURL, consts.ComfortChangeRecipientDataPath)
	if err != nil {
		return err
	}
	if shouldDryRun(runOpts, "POST", full, req) {
		return nil
	}
	_, _, err = s.c.comfortHTTP.DoJSON(ctx, "POST", full, req, nil)
	return wrapAPIError(err)
}

// Balance queries current comfort API balance.
func (s *ComfortService) Balance(ctx context.Context, runOpts ...RunOption) (*comfort.BalanceResponse, error) {
	if s == nil || s.c == nil {
		return nil, errors.New("client is nil")
	}
	if err := ensureComfortReady(s.c); err != nil {
		return nil, err
	}

	full, err := joinURL(s.c.cfg.comfortBaseURL, consts.ComfortBalancePath)
	if err != nil {
		return nil, err
	}
	if shouldDryRun(runOpts, "GET", full, nil) {
		return nil, nil
	}
	var out comfort.BalanceResponse
	_, _, err = s.c.comfortHTTP.DoJSON(ctx, "GET", full, nil, &out)
	if err != nil {
		return nil, wrapAPIError(err)
	}
	return &out, nil
}

// ExportOperations requests operations export file generation.
func (s *ComfortService) ExportOperations(ctx context.Context, req *comfort.ExportOperationsRequest, runOpts ...RunOption) (*comfort.ExportOperationsResponse, error) {
	if s == nil || s.c == nil {
		return nil, errors.New("client is nil")
	}
	if err := ensureComfortReady(s.c); err != nil {
		return nil, err
	}
	if err := validateComfortExport(req); err != nil {
		return nil, err
	}

	full, err := joinURL(s.c.cfg.comfortBaseURL, consts.ComfortExportOperationsPath)
	if err != nil {
		return nil, err
	}
	if shouldDryRun(runOpts, "POST", full, req) {
		return nil, nil
	}
	var out comfort.ExportOperationsResponse
	_, _, err = s.c.comfortHTTP.DoJSON(ctx, "POST", full, req, &out)
	if err != nil {
		return nil, wrapAPIError(err)
	}
	return &out, nil
}

// Do performs a signed request against the Comfort base URL.
func (s *ComfortService) Do(ctx context.Context, method string, endpointPath string, body any, out any, runOpts ...RunOption) error {
	if s == nil || s.c == nil {
		return errors.New("client is nil")
	}
	if err := ensureComfortReady(s.c); err != nil {
		return err
	}
	full, err := joinURL(s.c.cfg.comfortBaseURL, endpointPath)
	if err != nil {
		return err
	}
	if shouldDryRun(runOpts, method, full, body) {
		return nil
	}
	_, _, err = s.c.comfortHTTP.DoJSON(ctx, method, full, body, out)
	return wrapAPIError(err)
}

// =========================
// Checkout API
// =========================

type CheckoutService struct{ c *Client }

// CreateSession creates checkout session.
func (s *CheckoutService) CreateSession(ctx context.Context, req *checkout.CreateSessionRequest, runOpts ...RunOption) (checkout.GenericResponse, error) {
	if s == nil || s.c == nil {
		return nil, errors.New("client is nil")
	}
	if req == nil {
		return nil, &ValidationError{Fields: []FieldError{{Field: "request", Message: "is nil"}}}
	}
	if err := validateCheckoutCreateSession(req); err != nil {
		return nil, err
	}

	full, err := joinURL(s.c.cfg.checkoutBaseURL, consts.CheckoutCreateSessionPath)
	if err != nil {
		return nil, err
	}
	if shouldDryRun(runOpts, "POST", full, req) {
		return nil, nil
	}
	var out checkout.GenericResponse
	_, _, err = s.c.externalHTTP.DoJSON(ctx, "POST", full, req, &out)
	if err != nil {
		return nil, wrapAPIError(err)
	}
	return out, nil
}

// AddPayment adds products into checkout session.
func (s *CheckoutService) AddPayment(ctx context.Context, req *checkout.AddPaymentRequest, runOpts ...RunOption) (checkout.GenericResponse, error) {
	if s == nil || s.c == nil {
		return nil, errors.New("client is nil")
	}
	if req == nil {
		return nil, &ValidationError{Fields: []FieldError{{Field: "request", Message: "is nil"}}}
	}
	if err := validateCheckoutAddPayment(req); err != nil {
		return nil, err
	}

	full, err := joinURL(s.c.cfg.checkoutBaseURL, consts.CheckoutAddPaymentPath)
	if err != nil {
		return nil, err
	}
	if shouldDryRun(runOpts, "POST", full, req) {
		return nil, nil
	}
	var out checkout.GenericResponse
	_, _, err = s.c.externalHTTP.DoJSON(ctx, "POST", full, req, &out)
	if err != nil {
		return nil, wrapAPIError(err)
	}
	return out, nil
}

// VoidSession voids checkout session.
func (s *CheckoutService) VoidSession(ctx context.Context, req *checkout.SessionRequest, runOpts ...RunOption) error {
	if s == nil || s.c == nil {
		return errors.New("client is nil")
	}
	if req == nil {
		return &ValidationError{Fields: []FieldError{{Field: "request", Message: "is nil"}}}
	}
	if err := validateCheckoutSessionRequest(req); err != nil {
		return err
	}

	full, err := joinURL(s.c.cfg.checkoutBaseURL, consts.CheckoutVoidSessionPath)
	if err != nil {
		return err
	}
	if shouldDryRun(runOpts, "POST", full, req) {
		return nil
	}
	_, _, err = s.c.externalHTTP.DoJSON(ctx, "POST", full, req, nil)
	return wrapAPIError(err)
}

// GetStatus returns checkout session status.
func (s *CheckoutService) GetStatus(ctx context.Context, req *checkout.SessionRequest, runOpts ...RunOption) (checkout.GenericResponse, error) {
	if s == nil || s.c == nil {
		return nil, errors.New("client is nil")
	}
	if req == nil {
		return nil, &ValidationError{Fields: []FieldError{{Field: "request", Message: "is nil"}}}
	}
	if err := validateCheckoutSessionRequest(req); err != nil {
		return nil, err
	}

	full, err := joinURL(s.c.cfg.checkoutBaseURL, consts.CheckoutGetStatusPath)
	if err != nil {
		return nil, err
	}
	if shouldDryRun(runOpts, "POST", full, req) {
		return nil, nil
	}
	var out checkout.GenericResponse
	_, _, err = s.c.externalHTTP.DoJSON(ctx, "POST", full, req, &out)
	if err != nil {
		return nil, wrapAPIError(err)
	}
	return out, nil
}

// ExpireSession force-expires checkout session.
func (s *CheckoutService) ExpireSession(ctx context.Context, req *checkout.SessionRequest, runOpts ...RunOption) error {
	if s == nil || s.c == nil {
		return errors.New("client is nil")
	}
	if req == nil {
		return &ValidationError{Fields: []FieldError{{Field: "request", Message: "is nil"}}}
	}
	if err := validateCheckoutSessionRequest(req); err != nil {
		return err
	}

	full, err := joinURL(s.c.cfg.checkoutBaseURL, consts.CheckoutExpireSessionPath)
	if err != nil {
		return err
	}
	if shouldDryRun(runOpts, "POST", full, req) {
		return nil
	}
	_, _, err = s.c.externalHTTP.DoJSON(ctx, "POST", full, req, nil)
	return wrapAPIError(err)
}

// Do performs a signed request against Checkout base URL.
func (s *CheckoutService) Do(ctx context.Context, method string, path string, body any, out any, runOpts ...RunOption) error {
	if s == nil || s.c == nil {
		return errors.New("client is nil")
	}
	full, err := joinURL(s.c.cfg.checkoutBaseURL, path)
	if err != nil {
		return err
	}
	if shouldDryRun(runOpts, method, full, body) {
		return nil
	}
	_, _, err = s.c.externalHTTP.DoJSON(ctx, method, full, body, out)
	return wrapAPIError(err)
}

// =========================
// Validation
// =========================

func validateCreateSession(req *acquiring.CreateSessionRequest) error {
	ve := &ValidationError{}
	if req.MerchantID == "" {
		ve.Add("merchant_id", "is required")
	}
	if req.ClientPhone == "" {
		ve.Add("client_phone", "is required")
	}
	if ve.HasErrors() {
		return ve
	}
	return nil
}

func validateAddPayment(req *acquiring.AddPaymentRequest) error {
	ve := &ValidationError{}
	if req.MerchantID == "" {
		ve.Add("merchant_id", "is required")
	}
	if req.SessionID == "" {
		ve.Add("session_id", "is required")
	}
	if req.Amount <= 0 {
		ve.Add("amount", "must be > 0")
	}
	if req.Delivery != nil {
		if req.UseHold == nil || !*req.UseHold {
			ve.Add("use_hold", "must be true when delivery is provided")
		}
		d := req.Delivery
		if d.VolumeWeight <= 0 {
			ve.Add("delivery.volume_weight", "must be > 0")
		}
		if d.Weight <= 0 {
			ve.Add("delivery.weight", "must be > 0")
		}
		if d.RecipientCity == "" {
			ve.Add("delivery.recipient_city", "is required")
		}
		if d.RecipientWarehouse == "" {
			ve.Add("delivery.recipient_warehouse", "is required")
		}
	}
	for i, p := range req.Products {
		if p.Description == "" {
			ve.Add(fmt.Sprintf("products[%d].description", i), "is required")
		}
		if p.Count <= 0 {
			ve.Add(fmt.Sprintf("products[%d].count", i), "must be > 0")
		}
		if p.Price <= 0 {
			ve.Add(fmt.Sprintf("products[%d].price", i), "must be > 0")
		}
	}
	if ve.HasErrors() {
		return ve
	}
	return nil
}

func validateSessionRequest(req *acquiring.SessionRequest) error {
	ve := &ValidationError{}
	if req.MerchantID == "" {
		ve.Add("merchant_id", "is required")
	}
	if req.SessionID == "" {
		ve.Add("session_id", "is required")
	}
	if ve.HasErrors() {
		return ve
	}
	return nil
}

func validateCompleteHold(req *acquiring.CompleteHoldRequest) error {
	ve := &ValidationError{}
	if req.MerchantID == "" {
		ve.Add("merchant_id", "is required")
	}
	if req.SessionID == "" {
		ve.Add("session_id", "is required")
	}
	if req.Amount != nil && *req.Amount <= 0 {
		ve.Add("amount", "must be > 0")
	}
	for i, op := range req.Operations {
		if op.ID == "" {
			ve.Add(fmt.Sprintf("operations[%d].id", i), "is required")
		}
		if op.Amount <= 0 {
			ve.Add(fmt.Sprintf("operations[%d].amount", i), "must be > 0")
		}
		if op.RecipientIdentifier == "" {
			ve.Add(fmt.Sprintf("operations[%d].recipient_identifier", i), "is required")
		}
	}
	if ve.HasErrors() {
		return ve
	}
	return nil
}

func validateDeliveryPrice(req *acquiring.DeliveryPriceRequest) error {
	ve := &ValidationError{}
	if req.MerchantID == "" {
		ve.Add("merchant_id", "is required")
	}
	if req.RecipientCity == "" {
		ve.Add("recipient_city", "is required")
	}
	if req.RecipientWarehouse == "" {
		ve.Add("recipient_warehouse", "is required")
	}
	if req.VolumeWeight <= 0 {
		ve.Add("volume_weight", "must be > 0")
	}
	if req.Weight <= 0 {
		ve.Add("weight", "must be > 0")
	}
	if req.Amount <= 0 {
		ve.Add("amount", "must be > 0")
	}
	if ve.HasErrors() {
		return ve
	}
	return nil
}

func validateCheckoutCreateSession(req *checkout.CreateSessionRequest) error {
	ve := &ValidationError{}
	if req.MerchantID == "" {
		ve.Add("merchant_id", "is required")
	}
	if req.CallbackURL == "" {
		ve.Add("callback_url", "is required")
	}
	createWaybill := req.CreateExpressWaybill != nil && *req.CreateExpressWaybill
	if createWaybill && req.Delivery == nil {
		ve.Add("delivery", "is required when create_express_waybill is true")
	}
	if req.Delivery != nil {
		if !createWaybill {
			ve.Add("create_express_waybill", "must be true when delivery is provided")
		}
		if req.Delivery.VolumeWeight <= 0 {
			ve.Add("delivery.volume_weight", "must be > 0")
		}
		if req.Delivery.Weight <= 0 {
			ve.Add("delivery.weight", "must be > 0")
		}
	}
	if ve.HasErrors() {
		return ve
	}
	return nil
}

func validateCheckoutAddPayment(req *checkout.AddPaymentRequest) error {
	ve := &ValidationError{}
	if req.MerchantID == "" {
		ve.Add("merchant_id", "is required")
	}
	if req.SessionID == "" {
		ve.Add("session_id", "is required")
	}
	if req.Amount <= 0 {
		ve.Add("amount", "must be > 0")
	}
	for i, p := range req.Products {
		if p.Count <= 0 {
			ve.Add(fmt.Sprintf("products[%d].count", i), "must be > 0")
		}
		if p.Price <= 0 {
			ve.Add(fmt.Sprintf("products[%d].price", i), "must be > 0")
		}
	}
	if ve.HasErrors() {
		return ve
	}
	return nil
}

func validateCheckoutSessionRequest(req *checkout.SessionRequest) error {
	ve := &ValidationError{}
	if req.MerchantID == "" {
		ve.Add("merchant_id", "is required")
	}
	if req.SessionID == "" {
		ve.Add("session_id", "is required")
	}
	if ve.HasErrors() {
		return ve
	}
	return nil
}

func validateComfortCreateOperations(req comfort.CreateOperationsRequest) error {
	ve := &ValidationError{}
	for i, op := range req.RawBody {
		if op.Amount == "" {
			ve.Add(fmt.Sprintf("RAW_BODY[%d].amount", i), "is required")
		}
		if op.Recipient != nil {
			r := op.Recipient
			if r.LastName == "" {
				ve.Add(fmt.Sprintf("RAW_BODY[%d].recipient.last_name", i), "is required")
			}
			if r.FirstName == "" {
				ve.Add(fmt.Sprintf("RAW_BODY[%d].recipient.first_name", i), "is required")
			}
			if r.Patronymic == "" {
				ve.Add(fmt.Sprintf("RAW_BODY[%d].recipient.patronymic", i), "is required")
			}
			if r.Phone == "" {
				ve.Add(fmt.Sprintf("RAW_BODY[%d].recipient.phone", i), "is required")
			}
		}
	}
	if ve.HasErrors() {
		return ve
	}
	return nil
}

func validateComfortRefundOperations(req *comfort.RefundOperationsRequest) error {
	ve := &ValidationError{}
	if req == nil {
		ve.Add("request", "is nil")
		return ve
	}
	if len(req.RawBody) == 0 {
		ve.Add("RAW_BODY", "must contain at least one operation id")
		return ve
	}
	for i, id := range req.RawBody {
		if id == "" {
			ve.Add(fmt.Sprintf("RAW_BODY[%d]", i), "is required")
		}
	}
	if ve.HasErrors() {
		return ve
	}
	return nil
}

func validateComfortChangeRecipientData(req *comfort.ChangeRecipientDataRequest) error {
	ve := &ValidationError{}
	if req == nil {
		ve.Add("request", "is nil")
		return ve
	}
	if req.GUID == "" {
		ve.Add("guid", "is required")
	}
	if req.Recipient.LastName == "" {
		ve.Add("recipient.last_name", "is required")
	}
	if req.Recipient.FirstName == "" {
		ve.Add("recipient.first_name", "is required")
	}
	if req.Recipient.Patronymic == "" {
		ve.Add("recipient.patronymic", "is required")
	}
	if ve.HasErrors() {
		return ve
	}
	return nil
}

func validateComfortExport(req *comfort.ExportOperationsRequest) error {
	ve := &ValidationError{}
	if req == nil {
		ve.Add("request", "is nil")
		return ve
	}
	if req.FromDate == "" {
		ve.Add("from_date", "is required")
	}
	if req.ToDate == "" {
		ve.Add("to_date", "is required")
	}
	if req.RecepientEmail == "" {
		ve.Add("recepient_email", "is required")
	}
	if req.Format != nil {
		switch *req.Format {
		case comfort.ExportFormatCSV, comfort.ExportFormatJSON, comfort.ExportFormatXLSX:
		default:
			ve.Add("format", "must be one of CSV, JSON, XLSX")
		}
	}
	if ve.HasErrors() {
		return ve
	}
	return nil
}
