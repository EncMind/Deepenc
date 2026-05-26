package attestation

import (
	"context"
)

type asyncResult struct {
	token string
	err   error
}

func runAttestationWithContext(ctx context.Context, fn func() (string, error)) (string, error) {
	resultCh := make(chan asyncResult, 1)

	go func() {
		token, err := fn()
		resultCh <- asyncResult{token: token, err: err}
	}()

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case res := <-resultCh:
		return res.token, res.err
	}
}
