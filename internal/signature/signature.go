package signature

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
)

// HashAlgorithm controls which hash is used for RSA PKCS#1 v1.5 signatures.
//
// NovaPay External API uses SHA-256 for x-sign, while Comfort API uses SHA-1.
// The SDK supports both algorithms.
type HashAlgorithm string

const (
	HashSHA256 HashAlgorithm = "SHA-256"
	HashSHA1   HashAlgorithm = "SHA-1"
)

func digest(algo HashAlgorithm, data []byte) (hash crypto.Hash, sum []byte, err error) {
	switch algo {
	case HashSHA256, "", "sha256", "SHA256":
		h := sha256.Sum256(data)
		return crypto.SHA256, h[:], nil
	case HashSHA1, "sha1", "SHA1":
		h := sha1.Sum(data)
		return crypto.SHA1, h[:], nil
	default:
		return 0, nil, fmt.Errorf("unsupported signature hash algorithm: %q", algo)
	}
}

// RSASigner signs and/or verifies NovaPay x-sign signatures using RSA PKCS#1 v1.5.
//
// If PrivateKey is nil, Sign will return an error.
// If PublicKey is nil, Verify will return an error.
type RSASigner struct {
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
	Hash       HashAlgorithm
}

func (s *RSASigner) Sign(body []byte) (string, error) {
	if s == nil || s.PrivateKey == nil {
		return "", errors.New("signature: private key is not configured")
	}
	h, sum, err := digest(s.Hash, body)
	if err != nil {
		return "", err
	}
	sig, err := rsa.SignPKCS1v15(rand.Reader, s.PrivateKey, h, sum)
	if err != nil {
		return "", fmt.Errorf("signature: rsa sign: %w", err)
	}
	return base64.StdEncoding.EncodeToString(sig), nil
}

func (s *RSASigner) Verify(body []byte, signatureBase64 string) error {
	if s == nil || s.PublicKey == nil {
		return errors.New("signature: public key is not configured")
	}
	sig, err := decodeSignatureBase64(signatureBase64)
	if err != nil {
		return err
	}
	h, sum, err := digest(s.Hash, body)
	if err != nil {
		return err
	}
	if err := rsa.VerifyPKCS1v15(s.PublicKey, h, sum, sig); err != nil {
		return fmt.Errorf("signature: verify failed: %w", err)
	}
	return nil
}

func decodeSignatureBase64(signatureBase64 string) ([]byte, error) {
	signatureBase64 = strings.TrimSpace(signatureBase64)
	if signatureBase64 == "" {
		return nil, errors.New("signature: empty signature")
	}
	sig, err := base64.StdEncoding.DecodeString(signatureBase64)
	if err == nil {
		return sig, nil
	}
	// Some integrations/proxies may strip trailing "=" padding.
	sig, rawErr := base64.RawStdEncoding.DecodeString(signatureBase64)
	if rawErr == nil {
		return sig, nil
	}
	return nil, fmt.Errorf("signature: invalid base64 signature: std=%v; raw=%v", err, rawErr)
}

// ParseRSAPrivateKeyPEM parses a PEM encoded RSA private key.
// It supports both PKCS#1 ("RSA PRIVATE KEY") and PKCS#8 ("PRIVATE KEY").
func ParseRSAPrivateKeyPEM(pemBytes []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("signature: invalid PEM (no block)")
	}
	switch block.Type {
	case "RSA PRIVATE KEY":
		k, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("signature: parse PKCS#1 private key: %w", err)
		}
		return k, nil
	case "PRIVATE KEY":
		keyAny, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("signature: parse PKCS#8 private key: %w", err)
		}
		k, ok := keyAny.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("signature: PKCS#8 key is not RSA (got %T)", keyAny)
		}
		return k, nil
	default:
		return nil, fmt.Errorf("signature: unsupported private key type: %q", block.Type)
	}
}

// ParseRSAPublicKeyPEM parses a PEM encoded RSA public key.
// It supports both PKIX ("PUBLIC KEY") and PKCS#1 ("RSA PUBLIC KEY").
func ParseRSAPublicKeyPEM(pemBytes []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("signature: invalid PEM (no block)")
	}
	switch block.Type {
	case "PUBLIC KEY":
		keyAny, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("signature: parse PKIX public key: %w", err)
		}
		k, ok := keyAny.(*rsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("signature: PKIX key is not RSA (got %T)", keyAny)
		}
		return k, nil
	case "RSA PUBLIC KEY":
		k, err := x509.ParsePKCS1PublicKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("signature: parse PKCS#1 public key: %w", err)
		}
		return k, nil
	default:
		return nil, fmt.Errorf("signature: unsupported public key type: %q", block.Type)
	}
}
