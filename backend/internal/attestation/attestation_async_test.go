package attestation

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRunAttestationWithContextSuccess(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	token, err := runAttestationWithContext(ctx, func() (string, error) {
		return "token", nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "token" {
		t.Fatalf("expected token, got %s", token)
	}
}

func TestRunAttestationWithContextTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	token, err := runAttestationWithContext(ctx, func() (string, error) {
		time.Sleep(100 * time.Millisecond)
		return "token", nil
	})
	if err == nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}
	if token != "" {
		t.Fatalf("expected empty token on timeout, got %s", token)
	}
}
