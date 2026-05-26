//go:build !linux || !cgo
// +build !linux !cgo

package attestation

import (
	"context"
	"fmt"
)

// Client stub for non-CGO builds
type Client struct{}

// NewClient returns error on non-CGO builds
func NewClient(endpoint string) (*Client, error) {
	return nil, fmt.Errorf("CGO attestation not available - must build with CGO_ENABLED=1 on Linux with azguestattestation1 library installed")
}

// FetchToken stub
func (c *Client) FetchToken(ctx context.Context, nonce string) (string, error) {
	return "", fmt.Errorf("CGO attestation not available")
}

// Close stub
func (c *Client) Close() error {
	return nil
}
