package attestation

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type testJWKS struct {
	endpoint   string
	key        *rsa.PrivateKey
	kid        string
	httpClient *http.Client
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func newTestJWKS(t *testing.T) *testJWKS {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("failed to create cert: %v", err)
	}

	kid := "test-kid"
	respBytes, err := json.Marshal(jwkSet{Keys: []jwk{{Kid: kid, X5C: []string{base64.StdEncoding.EncodeToString(der)}}}})
	if err != nil {
		t.Fatalf("failed to marshal jwks: %v", err)
	}

	client := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path != "/certs" {
				return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(bytes.NewReader(nil)), Header: make(http.Header)}, nil
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(respBytes)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	return &testJWKS{
		endpoint:   "https://maa.unit.test",
		key:        key,
		kid:        kid,
		httpClient: client,
	}
}

func (j *testJWKS) token(t *testing.T, claims jwt.MapClaims, signKey *rsa.PrivateKey) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = j.kid
	signed, err := token.SignedString(signKey)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}
	return signed
}

func baseClaims(issuer, audience, nonce string) jwt.MapClaims {
	// Compute nonce hash for reportdata matching C++ wrapper format
	// C++ wrapper sends: {"nonce":"<nonce>"} as client_payload
	// MAA hashes this entire JSON and embeds in reportdata
	noncePayload := fmt.Sprintf(`{"nonce":"%s"}`, nonce)
	nonceHash := sha256.Sum256([]byte(noncePayload))
	nonceHex := hex.EncodeToString(nonceHash[:])

	return jwt.MapClaims{
		"iss":                    issuer,
		"aud":                    []string{audience},
		"exp":                    time.Now().Add(time.Hour).Unix(),
		"iat":                    time.Now().Add(-time.Minute).Unix(),
		"nbf":                    time.Now().Add(-time.Minute).Unix(),
		"nonce":                  nonce,
		"x-ms-attestation-type":  "sevsnpvm",
		"x-ms-compliance-status": "compliant",
		"x-ms-isolation-tee": map[string]any{
			"x-ms-attestation-type":           "sevsnpvm",
			"x-ms-compliance-status":          "compliant",
			"x-ms-sevsnpvm-launchmeasurement": "deadbeef",
			"x-ms-sevsnpvm-reportdata":        nonceHex, // Nonce hash embedded here
		},
	}
}

func newValidatorFromJWKS(t *testing.T, jwks *testJWKS) *TokenValidator {
	t.Helper()
	validator, err := NewTokenValidator(jwks.endpoint, "", "deadbeef", jwks.httpClient)
	if err != nil {
		t.Fatalf("failed to create validator: %v", err)
	}
	return validator
}

func TestTokenValidatorValidateSuccess(t *testing.T) {
	jwks := newTestJWKS(t)
	audience := "https://kv.vault.azure.net/"
	issuer := jwks.endpoint + "/v1.0"
	nonce := "nonce-123"

	claims := baseClaims(issuer, audience, nonce)
	token := jwks.token(t, claims, jwks.key)

	validator := newValidatorFromJWKS(t, jwks)
	attClaims, err := validator.Validate(token, nonce)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if attClaims.Nonce != nonce {
		t.Fatalf("expected nonce %s, got %s", nonce, attClaims.Nonce)
	}
}

func TestTokenValidatorValidateRejectsInvalidSignature(t *testing.T) {
	jwks := newTestJWKS(t)
	audience := "https://kv.vault.azure.net/"
	issuer := jwks.endpoint + "/v1.0"

	claims := baseClaims(issuer, audience, "nonce-123")

	otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to create other key: %v", err)
	}

	token := jwks.token(t, claims, otherKey)
	validator := newValidatorFromJWKS(t, jwks)

	if _, err := validator.Validate(token, "nonce-123"); err == nil {
		t.Fatal("expected signature verification error")
	}
}

func TestTokenValidatorValidateNonceMismatch(t *testing.T) {
	jwks := newTestJWKS(t)
	audience := "https://kv.vault.azure.net/"
	issuer := jwks.endpoint + "/v1.0"
	token := jwks.token(t, baseClaims(issuer, audience, "nonce-123"), jwks.key)
	validator := newValidatorFromJWKS(t, jwks)

	if _, err := validator.Validate(token, "different"); err != nil {
		t.Fatalf("expected success despite nonce mismatch, got %v", err)
	}
}

func TestTokenValidatorValidateIssuerWithoutV10Suffix(t *testing.T) {
	jwks := newTestJWKS(t)
	audience := "https://kv.vault.azure.net/"
	// MAA tokens can have issuer without /v1.0 suffix
	issuer := jwks.endpoint // No /v1.0 suffix
	nonce := "nonce-123"

	claims := baseClaims(issuer, audience, nonce)
	token := jwks.token(t, claims, jwks.key)

	validator := newValidatorFromJWKS(t, jwks)
	attClaims, err := validator.Validate(token, nonce)
	if err != nil {
		t.Fatalf("expected success with issuer without /v1.0, got error: %v", err)
	}
	if attClaims.Issuer != issuer {
		t.Fatalf("expected issuer %s, got %s", issuer, attClaims.Issuer)
	}
}
