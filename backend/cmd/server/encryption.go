package main

import (
    "context"
    "crypto/aes"
    "crypto/cipher"
    "crypto/ecdsa"
    "crypto/elliptic"
    "crypto/rand"
    "crypto/sha256"
    "encoding/base64"
    "fmt"
    "log"
    "os"
    "strings"
    "time"

    "deepenc/internal/attestation"
)

// EncryptedMessage represents an encrypted message with all necessary components
type EncryptedMessage struct {
	EncryptedContent string `json:"encryptedContent"`
	EncryptedAESKey  string `json:"encryptedAESKey"`
	IV               string `json:"iv"`
	AuthTag          string `json:"authTag"`
	Algorithm        string `json:"algorithm"`
}

// TEEEncryptionService handles all encryption/decryption operations within the TEE
type TEEEncryptionService struct {
	masterPrivateKey *ecdsa.PrivateKey
	masterPublicKey  *ecdsa.PublicKey
	keyVaultService  *AzureKeyVaultService
	attestor         *attestation.Manager
}

// NewTEEEncryptionService creates a new encryption service with Azure Key Vault integration
func NewTEEEncryptionService() (*TEEEncryptionService, error) {
    log.Printf("[Encryption] Initializing TEE Encryption Service with Azure Key Vault")

    // Initialize Azure Key Vault service
    keyVaultURL := strings.TrimSpace(os.Getenv("AZURE_KEY_VAULT_URL"))
    secretName := strings.TrimSpace(os.Getenv("AZURE_KEY_VAULT_SECRET_NAME"))
    if keyVaultURL == "" || secretName == "" {
        return nil, fmt.Errorf("missing Key Vault configuration: AZURE_KEY_VAULT_URL and AZURE_KEY_VAULT_SECRET_NAME are required")
    }

    // Perform MAA attestation before accessing Key Vault
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
    defer cancel()

    maaEndpoint := strings.TrimSpace(os.Getenv("MAA_ENDPOINT"))
    expectedMeasurement := strings.TrimSpace(os.Getenv("MAA_POLICY_MEASUREMENT"))

    attestor, err := attestation.NewManager(attestation.Config{
        Endpoint:            maaEndpoint,
        KeyVaultURL:         keyVaultURL,
        ExpectedMeasurement: expectedMeasurement,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to configure attestation manager: %w", err)
    }

    if err := attestor.Init(ctx); err != nil {
        attestor.Close() // Cleanup on failure
        return nil, fmt.Errorf("MAA attestation failed: %w", err)
    }

    claims, err := attestor.RequireAttestation(ctx)
    if err != nil {
        attestor.Close()
        return nil, fmt.Errorf("attestation claims unavailable: %w", err)
    }

    log.Printf("[MAA] ✅ Attestation successful | type=%s | tee=%s | compliance=%s | measurement=%s...",
        claims.AttestationType,
        claims.IsolationTEE,
        claims.ComplianceStatus,
        claims.LaunchMeasurement[:min(32, len(claims.LaunchMeasurement))],
    )

    // Initialize Key Vault service (attestation token will be added via policy)
    keyVaultService, err := NewAzureKeyVaultService(keyVaultURL, secretName)
    if err != nil {
        attestor.Close()
        return nil, fmt.Errorf("failed to initialize Azure Key Vault: %w", err)
    }

    // TODO: Pass attestation token to Key Vault service via attestation-based access policy
    _ = attestor.Token() // Token available for future use

	// Get or create master key from Azure Key Vault
    privateKey, err := keyVaultService.GetOrCreateMasterKey()
    if err != nil {
        attestor.Close()
        return nil, fmt.Errorf("failed to get master key from Azure Key Vault: %w", err)
    }

	log.Printf("[Encryption] TEE Encryption Service initialized with Azure Key Vault integration")

	return &TEEEncryptionService{
		masterPrivateKey: privateKey,
		masterPublicKey:  &privateKey.PublicKey,
		keyVaultService:  keyVaultService,
		attestor:         attestor,
	}, nil
}

// NewTEEEncryptionServiceWithConfig creates encryption service with custom Key Vault configuration
func NewTEEEncryptionServiceWithConfig(keyVaultURL, secretName string) (*TEEEncryptionService, error) {
	log.Printf("[Encryption] Initializing TEE Encryption Service with custom config")
	log.Printf("[Encryption] Key Vault URL: %s, Secret: %s", keyVaultURL, secretName)

	keyVaultService, err := NewAzureKeyVaultService(keyVaultURL, secretName)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Azure Key Vault: %w", err)
	}

	privateKey, err := keyVaultService.GetOrCreateMasterKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get master key from Azure Key Vault: %w", err)
	}

	log.Printf("[Encryption] TEE Encryption Service initialized with custom configuration")

	return &TEEEncryptionService{
		masterPrivateKey: privateKey,
		masterPublicKey:  &privateKey.PublicKey,
		keyVaultService:  keyVaultService,
	}, nil
}

// EncryptMessage encrypts a plaintext message using hybrid ECC + AES-GCM encryption
func (tes *TEEEncryptionService) EncryptMessage(plaintext string) (*EncryptedMessage, error) {
	log.Printf("[Encryption] Encrypting message of length: %d", len(plaintext))

	// 1. Generate random AES-256 key
	aesKey := make([]byte, 32)
	if _, err := rand.Read(aesKey); err != nil {
		return nil, fmt.Errorf("failed to generate AES key: %w", err)
	}

	// 2. Generate random IV for AES-GCM
	iv := make([]byte, 12)
	if _, err := rand.Read(iv); err != nil {
		return nil, fmt.Errorf("failed to generate IV: %w", err)
	}

	// 3. Encrypt message with AES-GCM
	encryptedContent, authTag, err := tes.encryptWithAESGCM([]byte(plaintext), aesKey, iv)
	if err != nil {
		return nil, fmt.Errorf("AES encryption failed: %w", err)
	}

	// 4. Encrypt AES key with ECC public key
	encryptedAESKey, err := tes.encryptWithECC(aesKey, tes.masterPublicKey)
	if err != nil {
		return nil, fmt.Errorf("ECC encryption failed: %w", err)
	}

	// 5. Clear sensitive data from memory
	clearBytes(aesKey)

	result := &EncryptedMessage{
		EncryptedContent: base64.StdEncoding.EncodeToString(encryptedContent),
		EncryptedAESKey:  base64.StdEncoding.EncodeToString(encryptedAESKey),
		IV:               base64.StdEncoding.EncodeToString(iv),
		AuthTag:          base64.StdEncoding.EncodeToString(authTag),
		Algorithm:        "ECC-P256+AES256-GCM",
	}

	log.Printf("[Encryption] Message encrypted successfully")
	return result, nil
}

// DecryptMessage decrypts an encrypted message using the master private key
func (tes *TEEEncryptionService) DecryptMessage(encMsg *EncryptedMessage) (string, error) {
	log.Printf("[Encryption] Decrypting message with algorithm: %s", encMsg.Algorithm)

	// 1. Decode base64 components
	encryptedContent, err := base64.StdEncoding.DecodeString(encMsg.EncryptedContent)
	if err != nil {
		return "", fmt.Errorf("failed to decode content: %w", err)
	}

	encryptedAESKey, err := base64.StdEncoding.DecodeString(encMsg.EncryptedAESKey)
	if err != nil {
		return "", fmt.Errorf("failed to decode AES key: %w", err)
	}

	iv, err := base64.StdEncoding.DecodeString(encMsg.IV)
	if err != nil {
		return "", fmt.Errorf("failed to decode IV: %w", err)
	}

	authTag, err := base64.StdEncoding.DecodeString(encMsg.AuthTag)
	if err != nil {
		return "", fmt.Errorf("failed to decode auth tag: %w", err)
	}

	// 2. Decrypt AES key using ECC private key
	aesKey, err := tes.decryptWithECC(encryptedAESKey, tes.masterPrivateKey)
	if err != nil {
		return "", fmt.Errorf("ECC decryption failed: %w", err)
	}
	defer clearBytes(aesKey)

	// 3. Decrypt message using AES-GCM
	plaintext, err := tes.decryptWithAESGCM(encryptedContent, authTag, aesKey, iv)
	if err != nil {
		return "", fmt.Errorf("AES decryption failed: %w", err)
	}

	log.Printf("[Encryption] Message decrypted successfully, length: %d", len(plaintext))
	return string(plaintext), nil
}

// encryptWithAESGCM encrypts data using AES-256-GCM
func (tes *TEEEncryptionService) encryptWithAESGCM(plaintext, key, iv []byte) ([]byte, []byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}

	ciphertext := gcm.Seal(nil, iv, plaintext, nil)

	// Split ciphertext and auth tag
	authTagSize := gcm.Overhead()
	if len(ciphertext) < authTagSize {
		return nil, nil, fmt.Errorf("ciphertext too short")
	}

	actualCiphertext := ciphertext[:len(ciphertext)-authTagSize]
	authTag := ciphertext[len(ciphertext)-authTagSize:]

	return actualCiphertext, authTag, nil
}

// decryptWithAESGCM decrypts data using AES-256-GCM
func (tes *TEEEncryptionService) decryptWithAESGCM(ciphertext, authTag, key, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Combine ciphertext and auth tag for GCM
	fullCiphertext := append(ciphertext, authTag...)

	plaintext, err := gcm.Open(nil, iv, fullCiphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("GCM authentication failed: %w", err)
	}

	return plaintext, nil
}

// encryptWithECC encrypts data using ECIES (Elliptic Curve Integrated Encryption Scheme)
func (tes *TEEEncryptionService) encryptWithECC(data []byte, publicKey *ecdsa.PublicKey) ([]byte, error) {
	// Generate ephemeral key pair
	ephemeralPrivateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	// Perform ECDH key agreement
	sharedX, _ := publicKey.Curve.ScalarMult(publicKey.X, publicKey.Y, ephemeralPrivateKey.D.Bytes())

	// Derive encryption key using SHA-256 (simple KDF for MVP)
	sharedSecret := sharedX.Bytes()
	hash := sha256.Sum256(sharedSecret)
	encryptionKey := hash[:]

	// Encrypt data with derived key using AES-GCM
	block, err := aes.NewCipher(encryptionKey[:32])
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nil, nonce, data, nil)

	// Combine ephemeral public key + nonce + ciphertext
	ephemeralPubKeyBytes := elliptic.Marshal(publicKey.Curve, ephemeralPrivateKey.PublicKey.X, ephemeralPrivateKey.PublicKey.Y)

	result := make([]byte, 0, len(ephemeralPubKeyBytes)+len(nonce)+len(ciphertext))
	result = append(result, ephemeralPubKeyBytes...)
	result = append(result, nonce...)
	result = append(result, ciphertext...)

	return result, nil
}

// decryptWithECC decrypts data using ECIES
func (tes *TEEEncryptionService) decryptWithECC(encryptedData []byte, privateKey *ecdsa.PrivateKey) ([]byte, error) {
	curve := elliptic.P256()
	pubKeySize := (curve.Params().BitSize+7)/8*2 + 1 // 65 bytes for P-256 uncompressed

	if len(encryptedData) < pubKeySize+12 { // pubkey + min nonce size
		return nil, fmt.Errorf("encrypted data too short")
	}

	// Extract components
	ephemeralPubKeyBytes := encryptedData[:pubKeySize]
	nonce := encryptedData[pubKeySize : pubKeySize+12]
	ciphertext := encryptedData[pubKeySize+12:]

	// Reconstruct ephemeral public key
	ephemeralPubX, ephemeralPubY := elliptic.Unmarshal(curve, ephemeralPubKeyBytes)
	if ephemeralPubX == nil {
		return nil, fmt.Errorf("invalid ephemeral public key")
	}

	// Perform ECDH key agreement
	sharedX, _ := curve.ScalarMult(ephemeralPubX, ephemeralPubY, privateKey.D.Bytes())

	// Derive decryption key
	sharedSecret := sharedX.Bytes()
	hash := sha256.Sum256(sharedSecret)
	decryptionKey := hash[:]

	// Decrypt with AES-GCM
	block, err := aes.NewCipher(decryptionKey[:32])
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("GCM decryption failed: %w", err)
	}

	return plaintext, nil
}

// clearBytes securely clears sensitive data from memory
func clearBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

// GetPublicKeyForTesting returns the public key for testing purposes
func (tes *TEEEncryptionService) GetPublicKeyForTesting() *ecdsa.PublicKey {
	return tes.masterPublicKey
}

// GetPrivateKeyForTesting returns the private key for testing purposes
func (tes *TEEEncryptionService) GetPrivateKeyForTesting() *ecdsa.PrivateKey {
	return tes.masterPrivateKey
}

// RefreshKeyFromVault refreshes the master key from Azure Key Vault (clears cache and reloads)
func (tes *TEEEncryptionService) RefreshKeyFromVault() error {
	if tes.keyVaultService == nil {
		return fmt.Errorf("no Azure Key Vault service available")
	}

	log.Printf("[Encryption] Refreshing master key from Azure Key Vault")

	// Clear the cache to force reload
	tes.keyVaultService.ClearCache()

	// Get fresh key from vault
	privateKey, err := tes.keyVaultService.GetOrCreateMasterKey()
	if err != nil {
		return fmt.Errorf("failed to refresh key from vault: %w", err)
	}

	// Update the encryption service with new key
	tes.masterPrivateKey = privateKey
	tes.masterPublicKey = &privateKey.PublicKey

	log.Printf("[Encryption] Master key refreshed successfully")
	return nil
}

// GetCacheInfo returns information about the Key Vault cache state
func (tes *TEEEncryptionService) GetCacheInfo() (bool, time.Time, error) {
	if tes.keyVaultService == nil {
		return false, time.Time{}, fmt.Errorf("no Azure Key Vault service available")
	}

	isCached, cacheTime := tes.keyVaultService.GetCacheInfo()
	return isCached, cacheTime, nil
}

// IsUsingKeyVault returns true if the service is using Azure Key Vault
func (tes *TEEEncryptionService) IsUsingKeyVault() bool {
	return tes.keyVaultService != nil
}

// Close releases attestation resources
func (tes *TEEEncryptionService) Close() error {
	if tes.attestor != nil {
		return tes.attestor.Close()
	}
	return nil
}
