package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/stremovskyy/go-nova/internal/jsonutil"
	"github.com/stremovskyy/go-nova/log"
	"github.com/stremovskyy/recorder"
)

// Signer produces the NovaPay x-sign header value.
//
// The SDK signs the exact request body bytes it sends.
type Signer interface {
	Sign(body []byte) (string, error)
}

// Client is a small HTTP helper with JSON marshal/unmarshal and retry support.
// It is internal on purpose: the public API lives in the root package.
type Client struct {
	httpClient     *http.Client
	signer         Signer
	logger         log.Logger
	logBodies      bool
	retryAttempts  int
	retryWait      time.Duration
	defaultHeaders map[string]string
	recorder       recorder.Recorder
}

// New creates an internal HTTP client.
func New(httpClient *http.Client, signer Signer, logger log.Logger, retryAttempts int, retryWait time.Duration, defaultHeaders map[string]string, rec recorder.Recorder, logBodies bool) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	if logger == nil {
		logger = log.NopLogger{}
	}
	if retryAttempts <= 0 {
		retryAttempts = 1
	}
	if retryWait <= 0 {
		retryWait = 300 * time.Millisecond
	}
	return &Client{
		httpClient:     httpClient,
		signer:         signer,
		logger:         logger,
		logBodies:      logBodies,
		retryAttempts:  retryAttempts,
		retryWait:      retryWait,
		defaultHeaders: cloneHeaders(defaultHeaders),
		recorder:       rec,
	}
}

// DoJSON sends a request to url and unmarshals the JSON response into out (if out != nil).
// It returns the http response and the raw response body.
func (c *Client) DoJSON(ctx context.Context, method, url string, body any, out any) (*http.Response, []byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	var lastErr error
	wait := c.retryWait
	for attempt := 1; attempt <= c.retryAttempts; attempt++ {
		c.logger.Debugf("[NovaPay HTTP] request: method=%s url=%s attempt=%d/%d", method, url, attempt, c.retryAttempts)
		resp, raw, err := c.doOnce(ctx, method, url, body, out)
		if err == nil {
			if resp != nil {
				c.logger.Debugf("[NovaPay HTTP] response: method=%s url=%s status=%d response=%s", method, url, resp.StatusCode, logBody(raw, c.logBodies))
			}
			return resp, raw, nil
		}
		lastErr = err

		// Retry only on transient errors.
		if !isRetryable(err, resp) || attempt == c.retryAttempts {
			if resp != nil {
				c.logger.Errorf("[NovaPay HTTP] request failed: method=%s url=%s status=%d err=%v response=%s", method, url, resp.StatusCode, err, logBody(raw, c.logBodies))
			} else {
				c.logger.Errorf("[NovaPay HTTP] request failed: method=%s url=%s err=%v", method, url, err)
			}
			return resp, raw, err
		}
		c.logger.Warnf("[NovaPay HTTP] request retry: method=%s url=%s attempt=%d wait=%s err=%v", method, url, attempt, wait, err)
		select {
		case <-ctx.Done():
			return resp, raw, ctx.Err()
		case <-time.After(wait):
			wait *= 2
		}
	}
	return nil, nil, lastErr
}

func (c *Client) doOnce(ctx context.Context, method, url string, body any, out any) (*http.Response, []byte, error) {
	requestID := nextRequestID()

	bodyBytes, err := prepareBody(body)
	if err != nil {
		c.recordError(ctx, requestID, err)
		return nil, nil, err
	}
	// NovaPay signature is calculated on the request body.
	sigInput := bodyBytes
	if sigInput == nil {
		sigInput = []byte{}
	}

	var reader io.Reader
	if bodyBytes != nil {
		reader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		c.recordError(ctx, requestID, err)
		return nil, nil, err
	}

	req.Header.Set("Accept", "application/json")
	if bodyBytes != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range c.defaultHeaders {
		if k == "" || v == "" {
			continue
		}
		req.Header.Set(k, v)
	}
	if c.signer != nil {
		sig, err := c.signer.Sign(sigInput)
		if err != nil {
			c.recordError(ctx, requestID, err)
			return nil, nil, err
		}
		req.Header.Set("x-sign", sig)
	}

	c.logger.Debugf("[NovaPay HTTP] request prepared: request_id=%s method=%s url=%s payload=%s", requestID, method, url, logBody(sigInput, c.logBodies))

	c.recordRequest(ctx, requestID, sigInput)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.recordError(ctx, requestID, err)
		return nil, nil, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		c.recordError(ctx, requestID, err)
		return resp, nil, err
	}
	c.recordResponse(ctx, requestID, raw)

	c.logger.Debugf("[NovaPay HTTP] response received: request_id=%s method=%s url=%s status=%d response=%s", requestID, method, url, resp.StatusCode, logBody(raw, c.logBodies))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		statusErr := &HTTPStatusError{StatusCode: resp.StatusCode, Body: raw}
		c.recordError(ctx, requestID, statusErr)
		return resp, raw, statusErr
	}

	if out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			decErr := fmt.Errorf("decode json response: %w", err)
			c.recordError(ctx, requestID, decErr)
			return resp, raw, decErr
		}
	}

	return resp, raw, nil
}

// HTTPStatusError indicates a non-2xx response.
type HTTPStatusError struct {
	StatusCode int
	Body       []byte
}

func (e *HTTPStatusError) Error() string {
	if e == nil {
		return "http status error"
	}
	if len(e.Body) == 0 {
		return fmt.Sprintf("unexpected status: %d", e.StatusCode)
	}
	// Limit in error string.
	b := e.Body
	if len(b) > 512 {
		b = b[:512]
	}
	return fmt.Sprintf("unexpected status: %d: %s", e.StatusCode, string(b))
}

func isRetryable(err error, resp *http.Response) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var hs *HTTPStatusError
	if errors.As(err, &hs) {
		// Retry 5xx and rate limiting.
		return hs.StatusCode == http.StatusTooManyRequests || (hs.StatusCode >= 500 && hs.StatusCode != http.StatusNotImplemented)
	}

	// Retry only transport-level errors.
	var ue *url.Error
	if errors.As(err, &ue) {
		if errors.Is(ue.Err, context.Canceled) || errors.Is(ue.Err, context.DeadlineExceeded) {
			return false
		}
		var ne net.Error
		if errors.As(ue.Err, &ne) {
			return true
		}
		return true
	}
	var ne net.Error
	if errors.As(err, &ne) {
		return true
	}
	return false
}

func prepareBody(body any) ([]byte, error) {
	if body == nil {
		return nil, nil
	}
	switch v := body.(type) {
	case []byte:
		if len(v) == 0 {
			return []byte{}, nil
		}
		out := make([]byte, len(v))
		copy(out, v)
		return out, nil
	case string:
		return []byte(v), nil
	default:
		b, err := jsonutil.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("marshal json body: %w", err)
		}
		return b, nil
	}
}

func cloneHeaders(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func nextRequestID() string {
	return uuid.NewString()
}

func (c *Client) recordRequest(ctx context.Context, requestID string, body []byte) {
	if c == nil || c.recorder == nil {
		return
	}
	if err := c.recorder.RecordRequest(ctx, nil, requestID, body, nil); err != nil {
		c.logger.Warnf("[NovaPay HTTP] cannot record request: %v", err)
	}
}

func (c *Client) recordResponse(ctx context.Context, requestID string, body []byte) {
	if c == nil || c.recorder == nil {
		return
	}
	if err := c.recorder.RecordResponse(ctx, nil, requestID, body, nil); err != nil {
		c.logger.Warnf("[NovaPay HTTP] cannot record response: %v", err)
	}
}

func (c *Client) recordError(ctx context.Context, requestID string, err error) {
	if c == nil || c.recorder == nil || err == nil {
		return
	}
	if recErr := c.recorder.RecordError(ctx, nil, requestID, err, nil); recErr != nil {
		c.logger.Warnf("[NovaPay HTTP] cannot record error: %v", recErr)
	}
}

func summarizeBytes(b []byte) string {
	return fmt.Sprintf("size=%d bytes", len(b))
}

func logBody(b []byte, verbose bool) string {
	if !verbose {
		return summarizeBytes(b)
	}

	if pretty, ok := prettyJSONPreview(b); ok {
		return pretty
	}
	return previewBytes(b)
}

func prettyJSONPreview(b []byte) (string, bool) {
	if len(b) == 0 || !json.Valid(b) {
		return "", false
	}

	var out bytes.Buffer
	if err := json.Indent(&out, b, "", "  "); err != nil {
		return "", false
	}
	return truncate(out.String(), 4096), true
}

func previewBytes(b []byte) string {
	if len(b) == 0 {
		return "<empty>"
	}
	s := strings.TrimSpace(string(b))
	if s == "" {
		return "<empty>"
	}
	if !utf8.ValidString(s) {
		return fmt.Sprintf("<binary size=%d bytes>", len(b))
	}
	return truncate(s, 4096)
}

func truncate(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "...(truncated)"
}
