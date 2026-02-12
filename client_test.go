package go_nova

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stremovskyy/go-nova/acquiring"
	"github.com/stremovskyy/go-nova/comfort"
	"github.com/stremovskyy/go-nova/internal/signature"
	sdklog "github.com/stremovskyy/go-nova/log"
	"github.com/stremovskyy/recorder"
)

func TestExternalAndComfortSigningAndHeaders(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		_ = r.Body.Close()
		if err != nil {
			http.Error(w, "read body", http.StatusBadRequest)
			t.Errorf("read body: %v", err)
			return
		}

		switch r.URL.Path {
		case "/external/v1/session":
			if got := r.Header.Get("x-merchant-id"); got != "" {
				http.Error(w, "unexpected x-merchant-id", http.StatusBadRequest)
				t.Errorf("external request must not contain x-merchant-id, got %q", got)
				return
			}
			if err := (&signature.RSASigner{PublicKey: &key.PublicKey, Hash: signature.HashSHA256}).Verify(body, r.Header.Get("x-sign")); err != nil {
				http.Error(w, "bad external signature", http.StatusBadRequest)
				t.Errorf("external request signature verify failed: %v", err)
				return
			}
			_, _ = w.Write([]byte(`{"id":"session-id"}`))
		case "/comfort/v1/operations/create":
			if got := r.Header.Get("x-merchant-id"); got != "42" {
				http.Error(w, "missing x-merchant-id", http.StatusBadRequest)
				t.Errorf("comfort request must contain x-merchant-id=42, got %q", got)
				return
			}
			if err := (&signature.RSASigner{PublicKey: &key.PublicKey, Hash: signature.HashSHA1}).Verify(body, r.Header.Get("x-sign")); err != nil {
				http.Error(w, "bad comfort signature", http.StatusBadRequest)
				t.Errorf("comfort request signature verify failed: %v", err)
				return
			}

			var payload map[string]json.RawMessage
			if err := json.Unmarshal(body, &payload); err != nil {
				http.Error(w, "bad comfort body", http.StatusBadRequest)
				t.Errorf("decode comfort body: %v", err)
				return
			}
			if _, ok := payload["RAW_BODY"]; !ok {
				http.Error(w, "missing RAW_BODY", http.StatusBadRequest)
				t.Errorf("comfort create payload must include RAW_BODY field")
				return
			}
			_, _ = w.Write([]byte(`[{"guid":"guid-1","public_id":"public-1"}]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	client, err := NewClient(
		WithPrivateKey(key),
		WithAcquiringBaseURL(ts.URL+"/external"),
		WithComfortBaseURL(ts.URL+"/comfort"),
		WithComfortMerchantID("42"),
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	_, err = client.Acquiring().CreateSession(context.Background(), &acquiring.CreateSessionRequest{
		MerchantID:  "1",
		ClientPhone: "+380982850620",
	})
	if err != nil {
		t.Fatalf("acquiring create session: %v", err)
	}

	ops, err := client.Comfort().CreateOperations(context.Background(), comfort.CreateOperationsRequest{
		RawBody: []comfort.CreateOperationItem{
			{Amount: "1.00"},
		},
	})
	if err != nil {
		t.Fatalf("comfort create operations: %v", err)
	}
	if len(ops) != 1 || ops[0].GUID == "" || ops[0].PublicID == "" {
		t.Fatalf("unexpected comfort create operations response: %+v", ops)
	}
}

func TestValidateCreateSessionDoesNotRequireCallbackURL(t *testing.T) {
	err := validateCreateSession(&acquiring.CreateSessionRequest{
		MerchantID:  "1",
		ClientPhone: "+380982850620",
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestValidateComfortCreateOperationsRawBody(t *testing.T) {
	err := validateComfortCreateOperations(comfort.CreateOperationsRequest{})
	ve, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T (%v)", err, err)
	}
	if len(ve.Fields) != 1 || ve.Fields[0].Field != "RAW_BODY" {
		t.Fatalf("unexpected validation fields for empty RAW_BODY: %+v", ve.Fields)
	}

	err = validateComfortCreateOperations(comfort.CreateOperationsRequest{
		RawBody: []comfort.CreateOperationItem{
			{Amount: ""},
		},
	})
	ve, ok = err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T (%v)", err, err)
	}
	if len(ve.Fields) != 1 || ve.Fields[0].Field != "RAW_BODY[0].amount" {
		t.Fatalf("unexpected validation fields: %+v", ve.Fields)
	}
}

func TestDryRunSkipsHTTPCall(t *testing.T) {
	var hitCount int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hitCount, 1)
		_, _ = w.Write([]byte(`{"id":"session-id"}`))
	}))
	defer ts.Close()

	client, err := NewClient(
		WithAcquiringBaseURL(ts.URL),
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	var (
		called    bool
		gotMethod string
		gotURL    string
		gotReq    *acquiring.CreateSessionRequest
	)

	_, err = client.Acquiring().CreateSession(context.Background(), &acquiring.CreateSessionRequest{
		MerchantID:  "1",
		ClientPhone: "+380982850620",
	}, DryRun(func(method string, url string, payload any) {
		called = true
		gotMethod = method
		gotURL = url
		if v, ok := payload.(*acquiring.CreateSessionRequest); ok {
			gotReq = v
		}
	}))
	if err != nil {
		t.Fatalf("create session dry run: %v", err)
	}

	if !called {
		t.Fatalf("dry run handler was not called")
	}
	if gotMethod != "POST" {
		t.Fatalf("unexpected method: %q", gotMethod)
	}
	if gotURL != ts.URL+"/v1/session" {
		t.Fatalf("unexpected url: %q", gotURL)
	}
	if gotReq == nil || gotReq.MerchantID != "1" {
		t.Fatalf("unexpected payload: %+v", gotReq)
	}
	if atomic.LoadInt32(&hitCount) != 0 {
		t.Fatalf("expected no HTTP calls, got %d", hitCount)
	}
}

func TestNewClientWithRecorderRecordsTraffic(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	rec := &testRecorder{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":"session-id"}`))
	}))
	defer ts.Close()

	client, err := NewClientWithRecorder(rec,
		WithPrivateKey(key),
		WithAcquiringBaseURL(ts.URL),
	)
	if err != nil {
		t.Fatalf("new client with recorder: %v", err)
	}

	_, err = client.Acquiring().CreateSession(context.Background(), &acquiring.CreateSessionRequest{
		MerchantID:  "1",
		ClientPhone: "+380982850620",
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	if rec.requestCount != 1 {
		t.Fatalf("expected 1 recorded request, got %d", rec.requestCount)
	}
	if rec.responseCount != 1 {
		t.Fatalf("expected 1 recorded response, got %d", rec.responseCount)
	}
	if rec.errorCount != 0 {
		t.Fatalf("expected 0 recorded errors, got %d", rec.errorCount)
	}
}

func TestSetLogLevelEnablesDebugLogging(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	logger := &testLogger{level: sdklog.LevelInfo}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":"session-id"}`))
	}))
	defer ts.Close()

	client, err := NewClient(
		WithPrivateKey(key),
		WithLogger(logger),
		WithAcquiringBaseURL(ts.URL),
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	// Before enabling debug there should be no debug logs.
	_, err = client.Acquiring().CreateSession(context.Background(), &acquiring.CreateSessionRequest{
		MerchantID:  "1",
		ClientPhone: "+380982850620",
	})
	if err != nil {
		t.Fatalf("create session before debug: %v", err)
	}
	if logger.debugCount != 0 {
		t.Fatalf("expected 0 debug logs before enabling debug, got %d", logger.debugCount)
	}

	client.SetLogLevel(sdklog.LevelDebug)

	_, err = client.Acquiring().CreateSession(context.Background(), &acquiring.CreateSessionRequest{
		MerchantID:  "1",
		ClientPhone: "+380982850620",
	})
	if err != nil {
		t.Fatalf("create session after debug: %v", err)
	}
	if logger.debugCount == 0 {
		t.Fatalf("expected debug logs after enabling debug level")
	}
}

func TestSetLogLevelInfoSuppressesDebugLogging(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	logger := &testLogger{level: sdklog.LevelDebug}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":"session-id"}`))
	}))
	defer ts.Close()

	client, err := NewClient(
		WithPrivateKey(key),
		WithLogger(logger),
		WithAcquiringBaseURL(ts.URL),
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	// Confirm we get debug when set to debug.
	_, err = client.Acquiring().CreateSession(context.Background(), &acquiring.CreateSessionRequest{
		MerchantID:  "1",
		ClientPhone: "+380982850620",
	})
	if err != nil {
		t.Fatalf("create session at debug: %v", err)
	}
	if logger.debugCount == 0 {
		t.Fatalf("expected debug logs at debug level")
	}

	logger.debugCount = 0
	client.SetLogLevel(sdklog.LevelInfo)

	_, err = client.Acquiring().CreateSession(context.Background(), &acquiring.CreateSessionRequest{
		MerchantID:  "1",
		ClientPhone: "+380982850620",
	})
	if err != nil {
		t.Fatalf("create session at info: %v", err)
	}
	if logger.debugCount != 0 {
		t.Fatalf("expected debug logs to be suppressed at info level, got %d", logger.debugCount)
	}
}

type testRecorder struct {
	requestCount  int
	responseCount int
	errorCount    int
}

func (t *testRecorder) RecordRequest(context.Context, *string, string, []byte, map[string]string) error {
	t.requestCount++
	return nil
}

func (t *testRecorder) RecordResponse(context.Context, *string, string, []byte, map[string]string) error {
	t.responseCount++
	return nil
}

func (t *testRecorder) RecordError(context.Context, *string, string, error, map[string]string) error {
	t.errorCount++
	return nil
}

func (t *testRecorder) RecordMetrics(context.Context, *string, string, map[string]string, map[string]string) error {
	return nil
}

func (t *testRecorder) GetRequest(context.Context, string) ([]byte, error) {
	return nil, nil
}

func (t *testRecorder) GetResponse(context.Context, string) ([]byte, error) {
	return nil, nil
}

func (t *testRecorder) FindByTag(context.Context, string) ([]string, error) {
	return nil, nil
}

func (t *testRecorder) Async() recorder.AsyncRecorder {
	return nil
}

type testLogger struct {
	level      sdklog.Level
	debugCount int
	infoCount  int
	warnCount  int
	errCount   int
}

func (t *testLogger) SetLevel(level sdklog.Level) {
	t.level = level
}

func (t *testLogger) Debugf(string, ...any) {
	if t.level <= sdklog.LevelDebug {
		t.debugCount++
	}
}

func (t *testLogger) Infof(string, ...any) {
	if t.level <= sdklog.LevelInfo {
		t.infoCount++
	}
}

func (t *testLogger) Warnf(string, ...any) {
	if t.level <= sdklog.LevelWarn {
		t.warnCount++
	}
}

func (t *testLogger) Errorf(string, ...any) {
	if t.level <= sdklog.LevelError {
		t.errCount++
	}
}
