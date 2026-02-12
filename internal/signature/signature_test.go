package signature

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"strings"
	"testing"
)

func TestDecodeSignatureBase64(t *testing.T) {
	want := []byte("abc123")
	std := base64.StdEncoding.EncodeToString(want)
	raw := strings.TrimRight(std, "=")
	trimmed := " \n\t" + std + "\r\n"

	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{name: "standard", input: std},
		{name: "raw", input: raw},
		{name: "trimmed", input: trimmed},
		{name: "empty", input: "  \t", wantErr: "empty signature"},
		{name: "invalid", input: "not-base64", wantErr: "invalid base64 signature"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := decodeSignatureBase64(tc.input)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("decode failed: %v", err)
			}
			if string(got) != string(want) {
				t.Fatalf("decoded value mismatch: got %q want %q", got, want)
			}
		})
	}
}

func TestRSASignerVerifyAcceptsTrimmedAndUnpaddedSignature(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	signer := &RSASigner{PrivateKey: key, Hash: HashSHA256}
	verifier := &RSASigner{PublicKey: &key.PublicKey, Hash: HashSHA256}

	body := []byte(`{"id":"123","status":"ok"}`)
	sig, err := signer.Sign(body)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	if err := verifier.Verify(body, " \n"+sig+"\r\n"); err != nil {
		t.Fatalf("verify trimmed signature: %v", err)
	}

	raw := strings.TrimRight(sig, "=")
	if err := verifier.Verify(body, raw); err != nil {
		t.Fatalf("verify raw signature: %v", err)
	}
}
