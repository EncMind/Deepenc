package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
)

// AzureKeyVaultService manages encryption keys using Azure Key Vault with TEE memory caching
type AzureKeyVaultService struct {
	client      *azsecrets.Client
	keyVaultURL string
	secretName  string

	// TEE memory cache (no TTL for simplicity)
	cacheMutex sync.RWMutex
	cachedKey  *ecdsa.PrivateKey
	cacheTime  time.Time
}

// NewAzureKeyVaultService creates a new Azure Key Vault service with managed identity authentication
func NewAzureKeyVaultService(keyVaultURL, secretName string) (*AzureKeyVaultService, error) {
	log.Printf("[KeyVault] Initializing Azure Key Vault service")
	log.Printf("[KeyVault] Key Vault URL: %s", keyVaultURL)
	log.Printf("[KeyVault] Secret Name: %s", secretName)

	// Use managed identity for authentication (works inside TEE with proper Azure setup)
	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure credential: %w", err)
	}

	// Create Key Vault client
	client, err := azsecrets.NewClient(keyVaultURL, credential, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Key Vault client: %w", err)
	}

	service := &AzureKeyVaultService{
		client:      client,
		keyVaultURL: keyVaultURL,
		secretName:  secretName,
	}

	log.Printf("[KeyVault] Azure Key Vault service initialized successfully")
	return service, nil
}

// GetOrCreateMasterKey retrieves the master key from cache or Azure Key Vault, creating if necessary
func (kvs *AzureKeyVaultService) GetOrCreateMasterKey() (*ecdsa.PrivateKey, error) {
	// Check TEE memory cache first
	kvs.cacheMutex.RLock()
	if kvs.cachedKey != nil {
		log.Printf("[KeyVault] Using cached master key (cached at: %v)", kvs.cacheTime)
		defer kvs.cacheMutex.RUnlock()
		return kvs.cachedKey, nil
	}
	kvs.cacheMutex.RUnlock()

	// Cache miss - acquire write lock and check again (double-checked locking)
	kvs.cacheMutex.Lock()
	defer kvs.cacheMutex.Unlock()

	if kvs.cachedKey != nil {
		log.Printf("[KeyVault] Key was cached by another goroutine")
		return kvs.cachedKey, nil
	}

	log.Printf("[KeyVault] Cache miss - retrieving key from Azure Key Vault")

	// Try to retrieve existing key from Azure Key Vault
	privateKey, err := kvs.getPrivateKeyFromVault()
	if err != nil {
		log.Printf("[KeyVault] Failed to retrieve existing key: %v", err)
		log.Printf("[KeyVault] Creating new master key pair")

		// Generate new key pair
		privateKey, err = kvs.generateAndStoreNewKey()
		if err != nil {
			return nil, fmt.Errorf("failed to generate and store new key: %w", err)
		}
	} else {
		log.Printf("[KeyVault] Successfully retrieved existing master key from vault")
	}

	// Cache the key in TEE memory (no TTL for simplicity)
	kvs.cachedKey = privateKey
	kvs.cacheTime = time.Now()

	log.Printf("[KeyVault] Master key cached in TEE memory")
	return privateKey, nil
}

// getPrivateKeyFromVault retrieves the private key from Azure Key Vault
func (kvs *AzureKeyVaultService) getPrivateKeyFromVault() (*ecdsa.PrivateKey, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Printf("[KeyVault] Fetching secret '%s' from Key Vault", kvs.secretName)

	response, err := kvs.client.GetSecret(ctx, kvs.secretName, "", nil)
	if err != nil {
		if azErr, ok := err.(*azcore.ResponseError); ok && azErr.StatusCode == 404 {
			return nil, fmt.Errorf("secret not found in Key Vault")
		}
		return nil, fmt.Errorf("failed to get secret from Key Vault: %w", err)
	}

	if response.Value == nil {
		return nil, fmt.Errorf("secret value is nil")
	}

	// Parse PEM-encoded private key
	privateKey, err := parsePrivateKeyFromPEM(*response.Value)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key from Key Vault: %w", err)
	}

	log.Printf("[KeyVault] Successfully parsed private key from Key Vault")
	return privateKey, nil
}

// generateAndStoreNewKey generates a new ECC key pair and attempts to store it in Azure Key Vault
func (kvs *AzureKeyVaultService) generateAndStoreNewKey() (*ecdsa.PrivateKey, error) {
	log.Printf("[KeyVault] Generating new ECC P-256 key pair")

	// Generate new ECC P-256 private key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ECC key: %w", err)
	}

	// Convert to PEM format for storage
	pemData, err := privateKeyToPEM(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encode private key to PEM: %w", err)
	}

	// Try to store in Azure Key Vault (best effort)
	err = kvs.storePrivateKeyInVault(pemData)
	if err != nil {
		log.Printf("[KeyVault] Warning: Failed to store key in Key Vault (will use local key): %v", err)
		log.Printf("[KeyVault] Continuing with locally generated key for this session")
		// Don't return error - use the generated key locally
	} else {
		log.Printf("[KeyVault] New master key generated and stored successfully")
	}

	return privateKey, nil
}

// storePrivateKeyInVault stores the PEM-encoded private key in Azure Key Vault
func (kvs *AzureKeyVaultService) storePrivateKeyInVault(pemData string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Printf("[KeyVault] Storing private key in Key Vault as secret '%s'", kvs.secretName)

	secretParams := azsecrets.SetSecretParameters{
		Value: &pemData,
		SecretAttributes: &azsecrets.SecretAttributes{
			Enabled: &[]bool{true}[0],
		},
		ContentType: &[]string{"application/x-pem-file"}[0],
		Tags: map[string]*string{
			"purpose":   &[]string{"deepenc-master-key"}[0],
			"algorithm": &[]string{"ECC-P256"}[0],
			"created":   &[]string{time.Now().UTC().Format(time.RFC3339)}[0],
		},
	}

	_, err := kvs.client.SetSecret(ctx, kvs.secretName, secretParams, nil)
	if err != nil {
		return fmt.Errorf("failed to store secret in Key Vault: %w", err)
	}

	log.Printf("[KeyVault] Private key stored successfully in Key Vault")
	return nil
}

// ClearCache clears the TEE memory cache (useful for testing or forced refresh)
func (kvs *AzureKeyVaultService) ClearCache() {
	kvs.cacheMutex.Lock()
	defer kvs.cacheMutex.Unlock()

	if kvs.cachedKey != nil {
		log.Printf("[KeyVault] Clearing TEE memory cache")
		kvs.cachedKey = nil
		kvs.cacheTime = time.Time{}
	}
}

// GetCacheInfo returns information about the current cache state (for monitoring/debugging)
func (kvs *AzureKeyVaultService) GetCacheInfo() (bool, time.Time) {
	kvs.cacheMutex.RLock()
	defer kvs.cacheMutex.RUnlock()

	return kvs.cachedKey != nil, kvs.cacheTime
}

// privateKeyToPEM converts an ECDSA private key to PEM format
func privateKeyToPEM(privateKey *ecdsa.PrivateKey) (string, error) {
	// Marshal private key to PKCS#8 format
	privateKeyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to marshal private key: %w", err)
	}

	// Create PEM block
	pemBlock := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privateKeyBytes,
	}

	// Encode to PEM format
	pemData := pem.EncodeToMemory(pemBlock)
	return string(pemData), nil
}

// parsePrivateKeyFromPEM parses a PEM-encoded ECDSA private key
func parsePrivateKeyFromPEM(pemData string) (*ecdsa.PrivateKey, error) {
	// Decode PEM block
	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	if block.Type != "PRIVATE KEY" {
		return nil, fmt.Errorf("invalid PEM block type: %s", block.Type)
	}

	// Parse PKCS#8 private key
	privateKeyInterface, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse PKCS#8 private key: %w", err)
	}

	// Assert to ECDSA private key
	privateKey, ok := privateKeyInterface.(*ecdsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("parsed key is not an ECDSA private key")
	}

	// Verify it's P-256 curve
	if privateKey.Curve != elliptic.P256() {
		return nil, fmt.Errorf("private key is not using P-256 curve")
	}

	return privateKey, nil
}
