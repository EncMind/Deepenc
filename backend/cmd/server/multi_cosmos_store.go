package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
)

// MultiCosmosStore manages separate containers for different data types
type MultiCosmosStore struct {
	client                  *azcosmos.Client
	db                      *azcosmos.DatabaseClient
	usersContainer          *azcosmos.ContainerClient
	threadsContainer        *azcosmos.ContainerClient
	messagesContainer       *azcosmos.ContainerClient
	anonymousUsersContainer *azcosmos.ContainerClient
	dbID                    string
	usersPartitionKey       string
}

// Container definitions
const (
	UsersContainerID          = "users"
	ThreadsContainerID        = "threads"
	MessagesContainerID       = "messages"
	AnonymousUsersContainerID = "anonymous_users"
)

func NewMultiCosmosStore(ctx context.Context) (*MultiCosmosStore, error) {
	// Check if we should use emulator mode first
	isEmu := isEmulatorMode()
	if isEmu {
		configureForEmulator()
	}

	endpoint := strings.TrimSpace(envOr("COSMOS_ENDPOINT", "https://localhost:8081"))
	key := strings.TrimSpace(envOr("COSMOS_KEY", emulatorKey))
	dbID := envOr("COSMOS_DB", "deepenc")

	insecure := strings.EqualFold(os.Getenv("COSMOS_EMULATOR_INSECURE"), "1") ||
		strings.EqualFold(os.Getenv("COSMOS_EMULATOR_INSECURE"), "true")
	if isEmu {
		log.Printf("[multi-cosmos] 🔧 EMULATOR: %s", endpoint)
		log.Printf("[multi-cosmos] ⚠️ Using emulator key (dev only)")
	} else {
		log.Printf("[multi-cosmos] 🔐 Azure Cosmos: %s", endpoint)
	}
	log.Printf("[multi-cosmos] Config: db=%s insecure=%v", dbID, insecure)

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

	// Create containers with appropriate partition keys
	containers := map[string]string{
		UsersContainerID:          "/userId",          // Partition by Firebase UID
		ThreadsContainerID:        "/threadId",        // Partition by thread ID
		MessagesContainerID:       "/threadId",        // Partition by thread ID for message locality
		AnonymousUsersContainerID: "/anonymousUserId", // Partition by anonymous user ID
	}

	containerClients := make(map[string]*azcosmos.ContainerClient)

	// Create containers one by one with delays to avoid emulator issues
	containerOrder := []string{MessagesContainerID, UsersContainerID, ThreadsContainerID, AnonymousUsersContainerID}

	for i, containerID := range containerOrder {
		partitionKey := containers[containerID]

		// Add delay between container creation to help emulator
		if i > 0 {
			log.Printf("[multi-cosmos] ⏳ Waiting before creating next container...")
			time.Sleep(2 * time.Second)
		}

		log.Printf("[multi-cosmos] 🔄 Creating container: %s", containerID)

		// Create container
		props := azcosmos.ContainerProperties{
			ID: containerID,
			PartitionKeyDefinition: azcosmos.PartitionKeyDefinition{
				Paths: []string{partitionKey},
				Kind:  azcosmos.PartitionKeyKindHash,
			},
		}

		// Retry logic for container creation (no throughput for serverless compatibility)
		var createErr error
		for attempt := 1; attempt <= 3; attempt++ {
			_, createErr = dbClient.CreateContainer(ctx, props, nil)

			if createErr == nil || isConflict(createErr) {
				break // Success or container already exists
			}

			log.Printf("[multi-cosmos] ⚠️ Container creation attempt %d failed: %v", attempt, createErr)
			if attempt < 3 {
				time.Sleep(time.Duration(attempt*2) * time.Second)
			}
		}

		if createErr != nil && !isConflict(createErr) {
			return nil, fmt.Errorf("create container %s after 3 attempts: %w", containerID, createErr)
		}

		// Get container client
		conClient, err := dbClient.NewContainer(containerID)
		if err != nil {
			return nil, fmt.Errorf("container client %s: %w", containerID, err)
		}
		containerClients[containerID] = conClient

		log.Printf("[multi-cosmos] ✅ Container ready: %s (partition: %s)", containerID, partitionKey)
	}

	return &MultiCosmosStore{
		client:                  client,
		db:                      dbClient,
		usersContainer:          containerClients[UsersContainerID],
		threadsContainer:        containerClients[ThreadsContainerID],
		messagesContainer:       containerClients[MessagesContainerID],
		anonymousUsersContainer: containerClients[AnonymousUsersContainerID],
		dbID:                    dbID,
		usersPartitionKey:       containers[UsersContainerID],
	}, nil
}

// Container getters
func (ms *MultiCosmosStore) GetUsersContainer() *azcosmos.ContainerClient {
	return ms.usersContainer
}

func (ms *MultiCosmosStore) GetUsersPartitionKey() string {
	return ms.usersPartitionKey
}

func (ms *MultiCosmosStore) GetThreadsContainer() *azcosmos.ContainerClient {
	return ms.threadsContainer
}

func (ms *MultiCosmosStore) GetMessagesContainer() *azcosmos.ContainerClient {
	return ms.messagesContainer
}

func (ms *MultiCosmosStore) GetAnonymousUsersContainer() *azcosmos.ContainerClient {
	return ms.anonymousUsersContainer
}

// Test connection by creating a test document in each container
func (ms *MultiCosmosStore) TestConnection(ctx context.Context) error {
	testTime := time.Now().Unix()

	// Test users container
	testUser := map[string]interface{}{
		"id":     fmt.Sprintf("test-user-%d", testTime),
		"userId": fmt.Sprintf("test-user-%d", testTime),
		"email":  "test@example.com",
		"type":   "test",
	}
	userBytes, _ := json.Marshal(testUser)
	pk := azcosmos.NewPartitionKeyString(testUser["userId"].(string))
	if _, err := ms.usersContainer.CreateItem(ctx, pk, userBytes, nil); err != nil {
		return fmt.Errorf("test users container: %w", err)
	}

	// Test threads container
	testThread := map[string]interface{}{
		"id":       fmt.Sprintf("test-thread-%d", testTime),
		"threadId": fmt.Sprintf("test-thread-%d", testTime),
		"title":    "Test Thread",
		"type":     "test",
	}
	threadBytes, _ := json.Marshal(testThread)
	pk = azcosmos.NewPartitionKeyString(testThread["threadId"].(string))
	if _, err := ms.threadsContainer.CreateItem(ctx, pk, threadBytes, nil); err != nil {
		return fmt.Errorf("test threads container: %w", err)
	}

	// Test messages container
	testMessage := map[string]interface{}{
		"id":       fmt.Sprintf("test-message-%d", testTime),
		"threadId": fmt.Sprintf("test-thread-%d", testTime),
		"content":  "Test message",
		"type":     "test",
	}
	msgBytes, _ := json.Marshal(testMessage)
	pk = azcosmos.NewPartitionKeyString(testMessage["threadId"].(string))
	if _, err := ms.messagesContainer.CreateItem(ctx, pk, msgBytes, nil); err != nil {
		return fmt.Errorf("test messages container: %w", err)
	}

	log.Printf("[multi-cosmos] ✅ All containers tested successfully")
	return nil
}

// Ping database
func (ms *MultiCosmosStore) Ping(ctx context.Context) error {
	_, err := ms.db.Read(ctx, nil)
	return err
}
