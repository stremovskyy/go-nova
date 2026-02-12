package httpclient

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNextRequestIDIsUUIDv4(t *testing.T) {
	id := nextRequestID()

	parsed, err := uuid.Parse(id)
	if err != nil {
		t.Fatalf("request_id must be a valid UUID, got %q: %v", id, err)
	}
	if parsed.Version() != 4 {
		t.Fatalf("request_id must be UUID v4, got version %d (%q)", parsed.Version(), id)
	}
	if parsed.Variant() != uuid.RFC4122 {
		t.Fatalf("request_id must use RFC4122 variant, got %v (%q)", parsed.Variant(), id)
	}
}

func TestIsRetryable(t *testing.T) {
	if isRetryable(nil, nil) {
		t.Fatalf("nil error must not be retryable")
	}
	if isRetryable(errors.New("boom"), nil) {
		t.Fatalf("plain non-network error must not be retryable")
	}
	if !isRetryable(&HTTPStatusError{StatusCode: http.StatusInternalServerError}, nil) {
		t.Fatalf("500 should be retryable")
	}
	if !isRetryable(&HTTPStatusError{StatusCode: http.StatusTooManyRequests}, nil) {
		t.Fatalf("429 should be retryable")
	}
	if isRetryable(&HTTPStatusError{StatusCode: http.StatusBadRequest}, nil) {
		t.Fatalf("400 must not be retryable")
	}
}

func TestDoJSONDoesNotRetrySignerErrors(t *testing.T) {
	signer := &countingFailSigner{}
	c := New(&http.Client{Timeout: 250 * time.Millisecond}, signer, nil, 3, 10*time.Millisecond, nil, nil, false)

	_, _, err := c.DoJSON(context.Background(), http.MethodPost, "http://example.com", map[string]any{"ok": true}, nil)
	if err == nil {
		t.Fatalf("expected signer error")
	}

	if calls := atomic.LoadInt32(&signer.calls); calls != 1 {
		t.Fatalf("expected exactly one signing attempt, got %d", calls)
	}
}

type countingFailSigner struct {
	calls int32
}

func (s *countingFailSigner) Sign([]byte) (string, error) {
	atomic.AddInt32(&s.calls, 1)
	return "", errors.New("sign failed")
}
