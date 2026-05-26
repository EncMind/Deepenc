//go:build linux && cgo
// +build linux,cgo

package attestation

/*
#cgo LDFLAGS: -L${SRCDIR}/cgo -lattestation_wrapper -lazguestattestation -lz -lstdc++
#include "cgo/attestation_wrapper.h"
#include <stdlib.h>
*/
import "C"
import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"unsafe"
)

// Client handles Azure attestation using CGO
type Client struct {
	ctx      *C.AttestationContext
	endpoint string
}

// NewClient creates a new attestation client
func NewClient(endpoint string) (*Client, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("attestation endpoint cannot be empty")
	}

	ctx := C.create_attestation_client()
	if ctx == nil {
		return nil, fmt.Errorf("failed to initialize attestation client - check azguestattestation1 library")
	}

	log.Printf("[Attestation] ✅ CGO client initialized for endpoint: %s", endpoint)

	return &Client{
		ctx:      ctx,
		endpoint: endpoint,
	}, nil
}

// FetchToken performs attestation and returns JWT token
func (c *Client) FetchToken(ctx context.Context, nonce string) (string, error) {
	if nonce == "" {
		return "", fmt.Errorf("nonce cannot be empty")
	}

	token, err := runAttestationWithContext(ctx, func() (string, error) {
		cEndpoint := C.CString(c.endpoint)
		defer C.free(unsafe.Pointer(cEndpoint))

		cNonce := C.CString(nonce)
		defer C.free(unsafe.Pointer(cNonce))

		log.Printf("[Attestation] Requesting attestation token with nonce: %s", nonce)

		cToken := C.perform_attestation(c.ctx, cEndpoint, cNonce)
		if cToken == nil {
			return "", c.buildDetailedError()
		}
		defer C.free(unsafe.Pointer(cToken))

		result := C.GoString(cToken)
		log.Printf("[Attestation] ✅ Token received, length: %d bytes", len(result))

		return result, nil
	})
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			log.Printf("[Attestation] ⚠️  Attestation timed out: %v", err)
		}
		return "", err
	}
	return token, nil
}

// buildDetailedError creates helpful error message based on environment
func (c *Client) buildDetailedError() error {
	// Check TPM device
	if _, err := os.Stat("/dev/tpmrm0"); os.IsNotExist(err) {
		return fmt.Errorf("attestation failed: TPM device /dev/tpmrm0 not found - are you running on an Azure CVM?")
	}

	// Check TPM permissions
	if file, err := os.Open("/dev/tpmrm0"); err != nil {
		if os.IsPermission(err) {
			return fmt.Errorf("attestation failed: permission denied to /dev/tpmrm0 - server must run as root or in 'tss' group")
		}
	} else {
		file.Close()
	}

	// Generic error
	return fmt.Errorf("attestation failed: check TPM access, network connectivity to %s, and system logs", c.endpoint)
}

// Close releases attestation client resources
func (c *Client) Close() error {
	if c.ctx != nil {
		C.destroy_attestation_client(c.ctx)
		c.ctx = nil
		log.Printf("[Attestation] Client closed")
	}
	return nil
}
