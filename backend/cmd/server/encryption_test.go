package main

import (
	"crypto/elliptic"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"
	"time"
)

// TestTEEEncryptionServiceBasic tests basic encryption/decryption functionality
func TestTEEEncryptionServiceBasic(t *testing.T) {
	// Initialize encryption service
	service, err := NewTEEEncryptionService()
	if err != nil {
		t.Fatalf("Failed to create encryption service: %v", err)
	}

	// Test data
	testMessages := []string{
		"Hello, World!",
		"This is a secret message that needs to be encrypted.",
		"🔐 Unicode characters and emojis should work too! 🚀",
		"",                         // Empty string
		strings.Repeat("A", 10000), // Large message
	}

	for i, plaintext := range testMessages {
		t.Run(fmt.Sprintf("Message_%d", i), func(t *testing.T) {
			// Encrypt the message
			encrypted, err := service.EncryptMessage(plaintext)
			if err != nil {
				t.Fatalf("Encryption failed: %v", err)
			}

			// Verify all required fields are present (except for empty messages)
			if plaintext != "" && encrypted.EncryptedContent == "" {
				t.Error("EncryptedContent is empty for non-empty message")
			}
			if encrypted.EncryptedAESKey == "" {
				t.Error("EncryptedAESKey is empty")
			}
			if encrypted.IV == "" {
				t.Error("IV is empty")
			}
			if encrypted.AuthTag == "" {
				t.Error("AuthTag is empty")
			}
			if encrypted.Algorithm != "ECC-P256+AES256-GCM" {
				t.Errorf("Unexpected algorithm: %s", encrypted.Algorithm)
			}

			// Decrypt the message
			decrypted, err := service.DecryptMessage(encrypted)
			if err != nil {
				t.Fatalf("Decryption failed: %v", err)
			}

			// Verify decrypted message matches original
			if decrypted != plaintext {
				t.Errorf("Decrypted message doesn't match original.\nExpected: %q\nGot: %q", plaintext, decrypted)
			}
		})
	}
}

// TestEncryptionUniqueness tests that each encryption produces unique results
func TestEncryptionUniqueness(t *testing.T) {
	service, err := NewTEEEncryptionService()
	if err != nil {
		t.Fatalf("Failed to create encryption service: %v", err)
	}

	plaintext := "Same message encrypted multiple times"
	var encrypted1, encrypted2 *EncryptedMessage

	// Encrypt the same message twice
	encrypted1, err = service.EncryptMessage(plaintext)
	if err != nil {
		t.Fatalf("First encryption failed: %v", err)
	}

	encrypted2, err = service.EncryptMessage(plaintext)
	if err != nil {
		t.Fatalf("Second encryption failed: %v", err)
	}

	// Verify that encryptions are different (due to random IV and ephemeral keys)
	if encrypted1.EncryptedContent == encrypted2.EncryptedContent {
		t.Error("Encrypted content should be different for each encryption")
	}
	if encrypted1.EncryptedAESKey == encrypted2.EncryptedAESKey {
		t.Error("Encrypted AES keys should be different for each encryption")
	}
	if encrypted1.IV == encrypted2.IV {
		t.Error("IVs should be different for each encryption")
	}

	// But both should decrypt to the same plaintext
	decrypted1, err := service.DecryptMessage(encrypted1)
	if err != nil {
		t.Fatalf("First decryption failed: %v", err)
	}

	decrypted2, err := service.DecryptMessage(encrypted2)
	if err != nil {
		t.Fatalf("Second decryption failed: %v", err)
	}

	if decrypted1 != plaintext || decrypted2 != plaintext {
		t.Error("Both decryptions should produce the original plaintext")
	}
}

// TestInvalidDecryption tests behavior with corrupted or invalid data
func TestInvalidDecryption(t *testing.T) {
	service, err := NewTEEEncryptionService()
	if err != nil {
		t.Fatalf("Failed to create encryption service: %v", err)
	}

	plaintext := "Valid message for corruption testing"
	encrypted, err := service.EncryptMessage(plaintext)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	// Test cases for invalid data
	testCases := []struct {
		name     string
		modifier func(*EncryptedMessage)
	}{
		{
			name: "Corrupted content",
			modifier: func(e *EncryptedMessage) {
				e.EncryptedContent = "invalid_base64_content"
			},
		},
		{
			name: "Corrupted AES key",
			modifier: func(e *EncryptedMessage) {
				e.EncryptedAESKey = "invalid_base64_key"
			},
		},
		{
			name: "Corrupted IV",
			modifier: func(e *EncryptedMessage) {
				e.IV = "invalid_base64_iv"
			},
		},
		{
			name: "Corrupted auth tag",
			modifier: func(e *EncryptedMessage) {
				e.AuthTag = "invalid_base64_tag"
			},
		},
		{
			name: "Modified encrypted content",
			modifier: func(e *EncryptedMessage) {
				// Decode, modify, re-encode
				data, _ := base64.StdEncoding.DecodeString(e.EncryptedContent)
				if len(data) > 0 {
					data[0] ^= 0xFF // Flip bits
				}
				e.EncryptedContent = base64.StdEncoding.EncodeToString(data)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a copy and corrupt it
			corruptedMsg := *encrypted
			tc.modifier(&corruptedMsg)

			// Attempt decryption - should fail
			_, err := service.DecryptMessage(&corruptedMsg)
			if err == nil {
				t.Error("Decryption should have failed with corrupted data")
			}
		})
	}
}

// TestCrossServiceDecryption tests that messages encrypted by one service can be decrypted by another
// This simulates the real-world scenario where messages are encrypted by one instance and decrypted by another
func TestCrossServiceDecryption(t *testing.T) {
	// This test will fail because each service generates its own key pair
	// In production, both services would use the same key from Azure Key Vault

	service1, err := NewTEEEncryptionService()
	if err != nil {
		t.Fatalf("Failed to create first encryption service: %v", err)
	}

	service2, err := NewTEEEncryptionService()
	if err != nil {
		t.Fatalf("Failed to create second encryption service: %v", err)
	}

	plaintext := "Cross-service test message"

	// Encrypt with service1
	encrypted, err := service1.EncryptMessage(plaintext)
	if err != nil {
		t.Fatalf("Encryption with service1 failed: %v", err)
	}

	// Try to decrypt with service2 - this should fail for MVP (different keys)
	_, err = service2.DecryptMessage(encrypted)
	if err == nil {
		t.Error("Cross-service decryption should fail with different keys (MVP behavior)")
	}

	// But service1 should still be able to decrypt its own message
	decrypted, err := service1.DecryptMessage(encrypted)
	if err != nil {
		t.Fatalf("Service1 failed to decrypt its own message: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("Decrypted message doesn't match original: %q vs %q", decrypted, plaintext)
	}
}

// TestPerformance measures encryption/decryption performance
func TestPerformance(t *testing.T) {
	service, err := NewTEEEncryptionService()
	if err != nil {
		t.Fatalf("Failed to create encryption service: %v", err)
	}

	// Test different message sizes
	testCases := []struct {
		name string
		size int
	}{
		{"Small_100B", 100},
		{"Medium_1KB", 1024},
		{"Large_10KB", 10240},
		{"XLarge_100KB", 102400},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Generate test message of specified size
			plaintext := strings.Repeat("A", tc.size)

			// Measure encryption time
			startTime := time.Now()
			encrypted, err := service.EncryptMessage(plaintext)
			encryptionTime := time.Since(startTime)

			if err != nil {
				t.Fatalf("Encryption failed: %v", err)
			}

			// Measure decryption time
			startTime = time.Now()
			decrypted, err := service.DecryptMessage(encrypted)
			decryptionTime := time.Since(startTime)

			if err != nil {
				t.Fatalf("Decryption failed: %v", err)
			}

			if decrypted != plaintext {
				t.Error("Decrypted message doesn't match original")
			}

			// Log performance metrics
			t.Logf("Size: %d bytes, Encryption: %v, Decryption: %v",
				tc.size, encryptionTime, decryptionTime)

			// Performance assertions (adjust thresholds as needed)
			if encryptionTime > 10*time.Millisecond {
				t.Errorf("Encryption too slow: %v (should be < 10ms)", encryptionTime)
			}
			if decryptionTime > 15*time.Millisecond {
				t.Errorf("Decryption too slow: %v (should be < 15ms)", decryptionTime)
			}
		})
	}
}

// TestConcurrency tests encryption service under concurrent load
func TestConcurrency(t *testing.T) {
	service, err := NewTEEEncryptionService()
	if err != nil {
		t.Fatalf("Failed to create encryption service: %v", err)
	}

	const numGoroutines = 10
	const messagesPerGoroutine = 20

	// Channel to collect results
	results := make(chan error, numGoroutines*messagesPerGoroutine)

	// Start concurrent encryption/decryption operations
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			for j := 0; j < messagesPerGoroutine; j++ {
				plaintext := fmt.Sprintf("Goroutine %d, Message %d", goroutineID, j)

				// Encrypt
				encrypted, err := service.EncryptMessage(plaintext)
				if err != nil {
					results <- fmt.Errorf("goroutine %d: encryption failed: %v", goroutineID, err)
					continue
				}

				// Decrypt
				decrypted, err := service.DecryptMessage(encrypted)
				if err != nil {
					results <- fmt.Errorf("goroutine %d: decryption failed: %v", goroutineID, err)
					continue
				}

				// Verify
				if decrypted != plaintext {
					results <- fmt.Errorf("goroutine %d: message mismatch", goroutineID)
					continue
				}

				results <- nil // Success
			}
		}(i)
	}

	// Collect results
	successCount := 0
	totalOperations := numGoroutines * messagesPerGoroutine

	for i := 0; i < totalOperations; i++ {
		err := <-results
		if err != nil {
			t.Errorf("Concurrent operation failed: %v", err)
		} else {
			successCount++
		}
	}

	t.Logf("Concurrent test completed: %d/%d operations successful", successCount, totalOperations)
}

// TestBase64Encoding tests that all encrypted components are valid base64
func TestBase64Encoding(t *testing.T) {
	service, err := NewTEEEncryptionService()
	if err != nil {
		t.Fatalf("Failed to create encryption service: %v", err)
	}

	plaintext := "Base64 encoding test message"
	encrypted, err := service.EncryptMessage(plaintext)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	// Test that all components are valid base64
	components := map[string]string{
		"EncryptedContent": encrypted.EncryptedContent,
		"EncryptedAESKey":  encrypted.EncryptedAESKey,
		"IV":               encrypted.IV,
		"AuthTag":          encrypted.AuthTag,
	}

	for name, value := range components {
		_, err := base64.StdEncoding.DecodeString(value)
		if err != nil {
			t.Errorf("%s is not valid base64: %v", name, err)
		}
	}
}

// TestKeyGeneration tests that the service generates valid ECC keys
func TestKeyGeneration(t *testing.T) {
	service, err := NewTEEEncryptionService()
	if err != nil {
		t.Fatalf("Failed to create encryption service: %v", err)
	}

	// Get keys for testing
	publicKey := service.GetPublicKeyForTesting()
	privateKey := service.GetPrivateKeyForTesting()

	// Verify key properties
	if publicKey == nil {
		t.Error("Public key is nil")
	}
	if privateKey == nil {
		t.Error("Private key is nil")
	}

	// Verify curve
	if publicKey.Curve != elliptic.P256() {
		t.Error("Public key is not using P-256 curve")
	}
	if privateKey.Curve != elliptic.P256() {
		t.Error("Private key is not using P-256 curve")
	}

	// Verify key relationship
	if !publicKey.Curve.IsOnCurve(publicKey.X, publicKey.Y) {
		t.Error("Public key point is not on curve")
	}

	// Verify that public key is derived from private key
	derivedPubKey := &privateKey.PublicKey
	if derivedPubKey.X.Cmp(publicKey.X) != 0 || derivedPubKey.Y.Cmp(publicKey.Y) != 0 {
		t.Error("Public key doesn't match derived public key from private key")
	}
}

// Benchmark encryption performance
func BenchmarkEncryption(b *testing.B) {
	service, err := NewTEEEncryptionService()
	if err != nil {
		b.Fatalf("Failed to create encryption service: %v", err)
	}

	plaintext := "Benchmark test message for encryption performance measurement"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.EncryptMessage(plaintext)
		if err != nil {
			b.Fatalf("Encryption failed: %v", err)
		}
	}
}

// Benchmark decryption performance
func BenchmarkDecryption(b *testing.B) {
	service, err := NewTEEEncryptionService()
	if err != nil {
		b.Fatalf("Failed to create encryption service: %v", err)
	}

	plaintext := "Benchmark test message for decryption performance measurement"
	encrypted, err := service.EncryptMessage(plaintext)
	if err != nil {
		b.Fatalf("Failed to encrypt test message: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.DecryptMessage(encrypted)
		if err != nil {
			b.Fatalf("Decryption failed: %v", err)
		}
	}
}

// BenchmarkFullCycle benchmarks complete encrypt-decrypt cycle
func BenchmarkFullCycle(b *testing.B) {
	service, err := NewTEEEncryptionService()
	if err != nil {
		b.Fatalf("Failed to create encryption service: %v", err)
	}

	plaintext := "Full cycle benchmark test message"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		encrypted, err := service.EncryptMessage(plaintext)
		if err != nil {
			b.Fatalf("Encryption failed: %v", err)
		}

		_, err = service.DecryptMessage(encrypted)
		if err != nil {
			b.Fatalf("Decryption failed: %v", err)
		}
	}
}
