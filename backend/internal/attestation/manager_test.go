package attestation

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeClient struct {
	token             string
	err               error
	usedNonce         string
	blockUntilCtxDone bool
}

func (f *fakeClient) FetchToken(ctx context.Context, nonce string) (string, error) {
	f.usedNonce = nonce
	if f.blockUntilCtxDone {
		<-ctx.Done()
		return "", ctx.Err()
	}
	return f.token, f.err
}

func (f *fakeClient) Close() error { return nil }

type fakeValidator struct {
	claims    *AttestationClaims
	err       error
	usedToken string
	usedNonce string
}

func (f *fakeValidator) Validate(token, expectedNonce string) (*AttestationClaims, error) {
	f.usedToken = token
	f.usedNonce = expectedNonce
	if f.err != nil {
		return nil, f.err
	}
	return f.claims, nil
}

func TestManagerInitUsesCustomNonceGenerator(t *testing.T) {
	fClient := &fakeClient{token: "token"}
	claims := &AttestationClaims{
		Issuer:            "https://example.com/v1.0",
		Audience:          []string{"https://vault"},
		AttestationType:   "sevsnpvm",
		ComplianceStatus:  "compliant",
		LaunchMeasurement: "deadbeef",
		Nonce:             "static-nonce",
		ExpiresAt:         time.Now().Add(time.Hour),
	}
	fValidator := &fakeValidator{claims: claims}

	mgr := &Manager{
		client:    fClient,
		validator: fValidator,
		config: Config{
			Endpoint:            "https://example.com",
			KeyVaultURL:         "https://vault",
			ExpectedMeasurement: "deadbeef",
		},
	}

	nonceCalled := false
	nonceFn := func(string) (string, string, error) {
		nonceCalled = true
		return "static-nonce", "payload", nil
	}

	if err := mgr.Init(context.Background(), nonceFn); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !nonceCalled {
		t.Fatal("expected custom nonce generator to be used")
	}
	if fClient.usedNonce != "static-nonce" {
		t.Fatalf("client received nonce %s", fClient.usedNonce)
	}
	if fValidator.usedNonce != "static-nonce" {
		t.Fatalf("validator saw nonce %s", fValidator.usedNonce)
	}
}

func TestManagerInitPropagatesValidatorError(t *testing.T) {
	fClient := &fakeClient{token: "token"}
	fValidator := &fakeValidator{err: errors.New("bad token")}
	mgr := &Manager{client: fClient, validator: fValidator, config: Config{Endpoint: "https://example.com"}}

	err := mgr.Init(context.Background(), func(string) (string, string, error) {
		return "nonce", "payload", nil
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestManagerInitHonorsContextCancellation(t *testing.T) {
	fClient := &fakeClient{blockUntilCtxDone: true}
	mgr := &Manager{client: fClient, validator: &fakeValidator{}, config: Config{Endpoint: "https://example.com"}}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err := mgr.Init(ctx, func(string) (string, string, error) {
		return "nonce", "payload", nil
	})
	if err == nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context deadline exceeded, got %v", err)
	}
}
