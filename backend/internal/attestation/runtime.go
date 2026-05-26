package attestation

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// buildRuntimeNonce returns a nonce derived from runtime metadata and the serialized payload used in logging.
func buildRuntimeNonce(serviceName string) (string, string, error) {
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", "", fmt.Errorf("failed to generate nonce entropy: %w", err)
	}

	payload := map[string]any{
		"nonce":     hex.EncodeToString(randomBytes),
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"service":   serviceName,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal runtime payload: %w", err)
	}

	// Nonce must be <= 32 alphanumeric characters; use 16-byte random hex (32 chars).
	nonce := hex.EncodeToString(randomBytes)
	return nonce, string(payloadBytes), nil
}
