package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
)

const emulatorKey = "C2y6yDjf5/R+ob0N8A7Cgv30VRDJIWEHLM+4QDU5DE2nQ9nDuVTqobD4b8mGGyPMbIZnqyMsEcaGQy67XIw/Jw=="

type CosmosStore struct {
	client       *azcosmos.Client
	db           *azcosmos.DatabaseClient
	container    *azcosmos.ContainerClient
	dbID         string
	containerID  string
	partitionKey string
}

// Simple document for connection test
type MessageDoc struct {
	ID        string `json:"id"`
	UserID    string `json:"userId"`
	ThreadID  string `json:"threadId"` // partition key
	Provider  string `json:"provider"`
	Model     string `json:"model"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	CreatedAt int64  `json:"createdAt"`
}

func isEmulatorMode() bool {
	endpoint := os.Getenv("COSMOS_ENDPOINT")
	if endpoint == "" {
		return true // Default to emulator if no endpoint specified
	}
	// If it's an Azure Cosmos DB endpoint, definitely not emulator
	if strings.Contains(endpoint, "documents.azure.com") {
		return false
	}
	// Only consider it emulator mode if it explicitly points to localhost/127.0.0.1 with port 8081
	return strings.Contains(endpoint, "localhost:8081") ||
		strings.Contains(endpoint, "127.0.0.1:8081")
}

func configureForEmulator() {
	if isEmulatorMode() {
		if os.Getenv("COSMOS_KEY") == "" {
			_ = os.Setenv("COSMOS_KEY", emulatorKey)
		}
		if os.Getenv("COSMOS_ENDPOINT") == "" {
			_ = os.Setenv("COSMOS_ENDPOINT", "https://localhost:8081")
		}
		if os.Getenv("COSMOS_EMULATOR_INSECURE") == "" {
			_ = os.Setenv("COSMOS_EMULATOR_INSECURE", "true")
			log.Println("[cosmos] Auto-enabled COSMOS_EMULATOR_INSECURE for emulator mode")
		}
		log.Println("[cosmos] Auto-configured for emulator mode")
	}
}

func NewCosmosStore(ctx context.Context) (*CosmosStore, error) {
	// Check if we should use emulator mode first
	isEmu := isEmulatorMode()
	if isEmu {
		configureForEmulator()
	}

	endpoint := strings.TrimSpace(envOr("COSMOS_ENDPOINT", "https://localhost:8081"))
	key := strings.TrimSpace(envOr("COSMOS_KEY", emulatorKey))
	dbID := envOr("COSMOS_DB", "deepenc")
	collID := envOr("COSMOS_CONTAINER", "messages")
	pkPath := envOr("COSMOS_PARTITION_KEY", "/threadId")

	insecure := strings.EqualFold(os.Getenv("COSMOS_EMULATOR_INSECURE"), "1") ||
		strings.EqualFold(os.Getenv("COSMOS_EMULATOR_INSECURE"), "true")
	if isEmu {
		log.Printf("[cosmos] 🔧 EMULATOR: %s", endpoint)
		log.Printf("[cosmos] ⚠️ Using emulator key (dev only)")
	} else {
		log.Printf("[cosmos] 🔐 Azure Cosmos: %s", endpoint)
	}
	log.Printf("[cosmos] Config: db=%s container=%s pk=%s insecure=%v", dbID, collID, pkPath, insecure)

	transport, err := buildTransport(endpoint, insecure)
	if err != nil {
		return nil, fmt.Errorf("tls transport: %w", err)
	}
	cred, err := azcosmos.NewKeyCredential(key)
	if err != nil {
		return nil, fmt.Errorf("key credential: %w", err)
	}
	client, err := azcosmos.NewClientWithKey(endpoint, cred, &azcosmos.ClientOptions{
		ClientOptions: azcore.ClientOptions{Transport: transport},
	})
	if err != nil {
		if isEmu {
			return nil, fmt.Errorf("cosmos client (emulator): %w\nTip: set COSMOS_EMULATOR_INSECURE=true", err)
		}
		return nil, fmt.Errorf("cosmos client: %w", err)
	}

	// Ensure DB (no throughput for serverless compatibility)
	dbProps := azcosmos.DatabaseProperties{ID: dbID}
	if _, err = client.CreateDatabase(ctx, dbProps, nil); err != nil && !isConflict(err) {
		return nil, fmt.Errorf("create database: %w", err)
	}

	dbClient, err := client.NewDatabase(dbID)
	if err != nil {
		return nil, fmt.Errorf("db client: %w", err)
	}

	// Ensure container (no throughput for serverless compatibility)
	props := azcosmos.ContainerProperties{
		ID: collID,
		PartitionKeyDefinition: azcosmos.PartitionKeyDefinition{
			Paths: []string{pkPath},
			Kind:  azcosmos.PartitionKeyKindHash,
		},
	}
	if _, err = dbClient.CreateContainer(ctx, props, nil); err != nil && !isConflict(err) {
		return nil, fmt.Errorf("create container: %w", err)
	}

	conClient, err := dbClient.NewContainer(collID)
	if err != nil {
		return nil, fmt.Errorf("container client: %w", err)
	}

	return &CosmosStore{
		client:       client,
		db:           dbClient,
		container:    conClient,
		dbID:         dbID,
		containerID:  collID,
		partitionKey: pkPath,
	}, nil
}

func (s *CosmosStore) SaveMessage(ctx context.Context, m MessageDoc) error {
	if m.ThreadID == "" {
		return errors.New("missing threadId (partition key)")
	}
	body, err := json.Marshal(m)
	if err != nil {
		return err
	}
	pk := azcosmos.NewPartitionKeyString(m.ThreadID)
	_, err = s.container.CreateItem(ctx, pk, body, nil)
	return err
}

func (s *CosmosStore) TestConnection(ctx context.Context) error {
	testDoc := MessageDoc{
		ID:        fmt.Sprintf("test-%d", time.Now().Unix()),
		UserID:    "test-user",
		ThreadID:  "test-thread",
		Provider:  "test",
		Model:     "test-model",
		Role:      "system",
		Content:   "Connection test at " + time.Now().Format(time.RFC3339),
		CreatedAt: nowMS(),
	}
	return s.SaveMessage(ctx, testDoc)
}

// Ping (no writes)
func (s *CosmosStore) Ping(ctx context.Context) error {
	_, err := s.db.Read(ctx, nil)
	return err
}

func nowMS() int64 { return time.Now().UnixMilli() }

func envOr(k, def string) string {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return def
	}
	return v
}

func isConflict(err error) bool {
	var re *azcore.ResponseError
	return errors.As(err, &re) && re.StatusCode == http.StatusConflict
}

func isNotFound(err error) bool {
	var re *azcore.ResponseError
	return errors.As(err, &re) && re.StatusCode == http.StatusNotFound
}

// TLS transport
func buildTransport(endpoint string, insecure bool) (policy.Transporter, error) {
	u, _ := url.Parse(endpoint)
	host := ""
	if u != nil {
		host = strings.ToLower(u.Hostname())
	}
	isEmu := host == "localhost" || host == "127.0.0.1" || strings.Contains(endpoint, ":8081")

	if isEmu {
		if insecure {
			log.Printf("[tls] 🔓 emulator: skip verify (dev only)")
			return &http.Client{
				Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}, // #nosec G402
				Timeout:   30 * time.Second,
			}, nil
		}

		pool, _ := x509.SystemCertPool()
		if pool == nil {
			pool = x509.NewCertPool()
		}
		certPaths := []string{
			os.Getenv("COSMOS_EMULATOR_CERT_PATH"),
			filepath.Join(os.Getenv("HOME"), "emulatorcert.crt"),
			filepath.Join(os.Getenv("HOME"), ".cosmos", "emulatorcert.crt"),
			"/usr/local/share/ca-certificates/emulatorcert.crt",
		}
		loaded := false
		for _, p := range certPaths {
			if p == "" {
				continue
			}
			if data, err := os.ReadFile(p); err == nil {
				if pool.AppendCertsFromPEM(data) {
					log.Printf("[tls] ✅ loaded emulator cert: %s", p)
					loaded = true
					break
				}
			}
		}
		if !loaded {
			log.Printf("[tls] ⚠️ no emulator cert; falling back to insecure")
			return &http.Client{
				Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}, // #nosec G402
				Timeout:   30 * time.Second,
			}, nil
		}
		return &http.Client{
			Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}},
			Timeout:   30 * time.Second,
		}, nil
	}

	log.Printf("[tls] 🔐 prod: system CAs")
	return &http.Client{
		Transport: &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12}},
		Timeout:   30 * time.Second,
	}, nil
}
