package attestation

import (
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
)

const attestationHeader = "x-ms-attestation-token"

// NewKeyVaultPolicy creates a pipeline policy that forwards the MAA JWT to Key Vault.
func NewKeyVaultPolicy(token string) policy.Policy {
	return &keyVaultPolicy{token: token}
}

type keyVaultPolicy struct {
	token string
}

func (p *keyVaultPolicy) Do(req *policy.Request) (*http.Response, error) {
	if p.token != "" {
		req.Raw().Header.Set(attestationHeader, p.token)
	}
	return req.Next()
}
