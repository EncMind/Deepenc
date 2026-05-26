package attestation

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

// Manager handles attestation lifecycle
type attestationClient interface {
	FetchToken(ctx context.Context, nonce string) (string, error)
	Close() error
}

type claimsValidator interface {
	Validate(token, expectedNonce string) (*AttestationClaims, error)
}

type NonceGenerator func(serviceName string) (nonce string, logPayload string, err error)

type Manager struct {
	client    attestationClient
	validator claimsValidator
	config    Config

	// Cached attestation results
	mu     sync.RWMutex
	token  string
	claims *AttestationClaims
	nonce  string
}

// NewManager creates a new attestation manager
func NewManager(cfg Config) (*Manager, error) {
	cfg.normalize()

	// Create CGO attestation client
	client, err := NewClient(cfg.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to create attestation client: %w", err)
	}

	// Create token validator
	validator, err := NewTokenValidator(cfg.Endpoint, cfg.KeyVaultURL, cfg.ExpectedMeasurement, cfg.HTTPClient)
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to create token validator: %w", err)
	}

	log.Printf("[Attestation] Manager initialized with endpoint: %s", cfg.Endpoint)

	return &Manager{
		client:    client,
		validator: validator,
		config:    cfg,
	}, nil
}

// Init performs initial attestation
func (m *Manager) Init(ctx context.Context, nonceFn ...NonceGenerator) error {
	log.Printf("[Attestation] Starting attestation flow...")

	// Generate nonce
	generator := buildRuntimeNonce
	if len(nonceFn) > 0 && nonceFn[0] != nil {
		generator = nonceFn[0]
	}
	nonce, noncePayload, err := generator(serviceNameFromEnv())
	if err != nil {
		return fmt.Errorf("failed to generate nonce: %w", err)
	}
	log.Printf("[Attestation] Generated nonce payload: %s", noncePayload)

	// Fetch attestation token via CGO
	token, err := m.client.FetchToken(ctx, nonce)
	if err != nil {
		return fmt.Errorf("attestation failed: %w", err)
	}

	// Validate token and extract claims
	claims, err := m.validator.Validate(token, nonce)
	if err != nil {
		return fmt.Errorf("token validation failed: %w", err)
	}

	// Cache results
	m.mu.Lock()
	m.token = token
	m.claims = claims
	m.nonce = nonce
	m.mu.Unlock()

	log.Printf("[Attestation] ✅ Attestation successful | type=%s | tee=%s | compliance=%s | measurement=%s",
		claims.AttestationType,
		claims.IsolationTEE,
		claims.ComplianceStatus,
		truncateString(claims.LaunchMeasurement, 32))

	return nil
}

// RequireAttestation ensures attestation has been performed and returns claims
func (m *Manager) RequireAttestation(ctx context.Context) (*AttestationClaims, error) {
	m.mu.RLock()
	claims := m.claims
	m.mu.RUnlock()

	if claims == nil {
		return nil, fmt.Errorf("attestation not performed - call Init() first")
	}

	if claims.Expired() {
		return nil, fmt.Errorf("attestation token expired at %s", claims.ExpiresAt.Format(time.RFC3339))
	}

	return claims, nil
}

// Token returns the cached attestation token
func (m *Manager) Token() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.token
}

// Close releases resources
func (m *Manager) Close() error {
	if m.client != nil {
		return m.client.Close()
	}
	return nil
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func serviceNameFromEnv() string {
	hostname := strings.TrimSpace(os.Getenv("HOSTNAME"))
	if hostname == "" {
		return "deepenc-backend"
	}
	return hostname
}
