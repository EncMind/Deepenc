package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"os"
	"testing"
	"time"
)

// In CI/sandbox environments Azure Key Vault access is not available. Ensure the
// tests do not attempt outbound calls by default.
func init() {
	_ = os.Setenv("SKIP_AZURE_TESTS", "true")
}

func newFallbackEncryptionService(t *testing.T) *TEEEncryptionService {
	t.Helper()

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate test key: %v", err)
	}

	return &TEEEncryptionService{
		masterPrivateKey: privateKey,
		masterPublicKey:  &privateKey.PublicKey,
		// keyVaultService left nil to simulate fallback behaviour.
	}
}

// TestTEEEncryptionServiceFallback verifies the service works with locally generated keys
// when Azure Key Vault is unavailable.
func TestTEEEncryptionServiceFallback(t *testing.T) {
	t.Run("FallbackToLocalKeyGeneration", func(t *testing.T) {
		encService := newFallbackEncryptionService(t)

		if encService.masterPrivateKey == nil {
			t.Fatal("masterPrivateKey should not be nil")
		}
		if encService.masterPublicKey == nil {
			t.Fatal("masterPublicKey should not be nil")
		}

		originalMessage := "Test message with fallback key"

		encrypted, err := encService.EncryptMessage(originalMessage)
		if err != nil {
			t.Fatalf("encryption failed with fallback key: %v", err)
		}

		decrypted, err := encService.DecryptMessage(encrypted)
		if err != nil {
			t.Fatalf("decryption failed with fallback key: %v", err)
		}

		if decrypted != originalMessage {
			t.Errorf("message mismatch after round-trip: got %q, want %q", decrypted, originalMessage)
		}

		if encService.IsUsingKeyVault() {
			t.Error("service should report Key Vault disabled during fallback")
		}
	})
}

// TestKeyVaultServiceComponents exercises PEM encode/decode helpers with fallback keys.
func TestKeyVaultServiceComponents(t *testing.T) {
	t.Run("PEMEncoding", func(t *testing.T) {
		encService := newFallbackEncryptionService(t)

		originalKey := encService.masterPrivateKey
		pemData, err := privateKeyToPEM(originalKey)
		if err != nil {
			t.Fatalf("failed to convert key to PEM: %v", err)
		}
		if pemData == "" {
			t.Fatal("PEM data should not be empty")
		}

		parsedKey, err := parsePrivateKeyFromPEM(pemData)
		if err != nil {
			t.Fatalf("failed to parse key from PEM: %v", err)
		}

		testMessage := "PEM round-trip test"

		encService1 := &TEEEncryptionService{
			masterPrivateKey: originalKey,
			masterPublicKey:  &originalKey.PublicKey,
		}
		encrypted, err := encService1.EncryptMessage(testMessage)
		if err != nil {
			t.Fatalf("encryption with original key failed: %v", err)
		}

		encService2 := &TEEEncryptionService{
			masterPrivateKey: parsedKey,
			masterPublicKey:  &parsedKey.PublicKey,
		}
		decrypted, err := encService2.DecryptMessage(encrypted)
		if err != nil {
			t.Fatalf("decryption with parsed key failed: %v", err)
		}

		if decrypted != testMessage {
			t.Error("PEM round-trip failed - decrypted message mismatch")
		}
	})
}

// TestEncryptionPerformanceComparison compares encryption/decryption performance using fallback keys.
func TestEncryptionPerformanceComparison(t *testing.T) {
	t.Run("FallbackPerformance", func(t *testing.T) {
		encService := newFallbackEncryptionService(t)

		testMessage := "Performance test message for fallback mode"
		iterations := 50

		start := time.Now()
		for i := 0; i < iterations; i++ {
			if _, err := encService.EncryptMessage(testMessage); err != nil {
				t.Fatalf("encryption failed on iteration %d: %v", i, err)
			}
		}
		encryptTime := time.Since(start)

		encrypted, err := encService.EncryptMessage(testMessage)
		if err != nil {
			t.Fatalf("failed to encrypt test message: %v", err)
		}

		start = time.Now()
		for i := 0; i < iterations; i++ {
			if _, err := encService.DecryptMessage(encrypted); err != nil {
				t.Fatalf("decryption failed on iteration %d: %v", i, err)
			}
		}
		decryptTime := time.Since(start)

		t.Logf("Fallback encryption avg: %v", encryptTime/time.Duration(iterations))
		t.Logf("Fallback decryption avg: %v", decryptTime/time.Duration(iterations))
	})
}

// TestCacheManagement confirms cache helpers return an error when Key Vault is disabled.
func TestCacheManagement(t *testing.T) {
	t.Run("CacheUnavailableWithoutKeyVault", func(t *testing.T) {
		encService := newFallbackEncryptionService(t)

		isCached, cacheTime, err := encService.GetCacheInfo()
		if err == nil {
			t.Fatal("expected GetCacheInfo to fail without Key Vault")
		}
		if isCached {
			t.Error("cache should not be populated without Key Vault")
		}
		if !cacheTime.IsZero() {
			t.Error("cache time should be zero when Key Vault is unavailable")
		}
	})
}
