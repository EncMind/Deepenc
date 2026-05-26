package attestation

import (
	"net/http"
	"strings"
	"time"
)

const (
	defaultMAAEndpoint = "https://sharedeus2.eus2.attest.azure.net"
)

// Config drives how the attestation manager fetches and validates attestation tokens.
type Config struct {
	Endpoint            string       // MAA endpoint URL (required)
	KeyVaultURL         string       // Azure Key Vault URL (required)
	ExpectedMeasurement string       // Expected launch measurement (optional)
	HTTPClient          *http.Client // HTTP client for token validation (optional)
}

func (c *Config) normalize() {
	if c.Endpoint == "" {
		c.Endpoint = defaultMAAEndpoint
	}
	if c.HTTPClient == nil {
		c.HTTPClient = defaultHTTPClient()
	}
	c.KeyVaultURL = strings.ToLower(strings.TrimSpace(c.KeyVaultURL))
	c.ExpectedMeasurement = strings.ToLower(strings.TrimSpace(c.ExpectedMeasurement))
}

func defaultHTTPClient() *http.Client {
	return &http.Client{Timeout: 20 * time.Second}
}
