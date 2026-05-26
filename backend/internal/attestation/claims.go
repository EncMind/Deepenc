package attestation

import "time"

// AttestationClaims represents the subset of MAA JWT claims the server cares about.
type AttestationClaims struct {
	Issuer            string
	Audience          []string
	ExpiresAt         time.Time
	IssuedAt          time.Time
	NotBefore         time.Time
	Nonce             string
	ComplianceStatus  string
	AttestationType   string
	IsolationTEE      string // x-ms-isolation-tee: "sevsnpvm" for SEV-SNP VMs
	Measurement       string
	LaunchMeasurement string
	ReportData        string
	HostData          string
}

// Expired reports whether the attestation token has expired.
func (c *AttestationClaims) Expired() bool {
	if c == nil || c.ExpiresAt.IsZero() {
		return true
	}
	return time.Now().After(c.ExpiresAt)
}
