package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
)

// ProcessedWebhookEvent represents a webhook event that has been processed
// to prevent duplicate processing due to Stripe retries
type ProcessedWebhookEvent struct {
	ID          string `json:"id"`   // event_id (e.g., "evt_abc123")
	Type        string `json:"type"` // Always "webhook_event"
	UserID      string `json:"userId,omitempty"`
	ThreadID    string `json:"threadId,omitempty"`
	EventType   string `json:"eventType"`   // Stripe event type (e.g., "customer.subscription.updated")
	ProcessedAt int64  `json:"processedAt"` // Unix timestamp in milliseconds
	CreatedAt   int64  `json:"createdAt"`   // Unix timestamp in milliseconds
}

// WebhookEventStore handles storing and checking processed webhook events
type WebhookEventStore struct {
	container         *azcosmos.ContainerClient
	partitionKeyField string
}

const webhookPartitionValue = "webhooks"

// NewWebhookEventStore creates a new webhook event store from a container client
// This works with both single-container (cosmos.container) and multi-container (multiCosmos.GetUsersContainer())
func NewWebhookEventStore(container *azcosmos.ContainerClient, partitionKeyField string) (*WebhookEventStore, error) {
	if container == nil {
		return nil, fmt.Errorf("container client is required")
	}
	pkField := strings.TrimPrefix(partitionKeyField, "/")
	if pkField != "userId" && pkField != "threadId" {
		return nil, fmt.Errorf("unsupported partition key field: %s (expected userId or threadId)", pkField)
	}
	return &WebhookEventStore{
		container:         container,
		partitionKeyField: pkField,
	}, nil
}

// IsEventProcessed checks if a webhook event has already been processed
func (wes *WebhookEventStore) IsEventProcessed(ctx context.Context, eventID string) (bool, error) {
	log.Printf("[WebhookEventStore] Checking if event %s has been processed", eventID)

	pk := azcosmos.NewPartitionKeyString(webhookPartitionValue)

	// Try to read the event directly
	_, err := wes.container.ReadItem(ctx, pk, eventID, nil)
	if err != nil {
		// Check if it's a "not found" error using proper Cosmos error checking
		if isCosmosNotFound(err) {
			log.Printf("[WebhookEventStore] Event %s not found - first time processing", eventID)
			return false, nil
		}
		// Other errors
		log.Printf("[WebhookEventStore] Error checking event %s: %v", eventID, err)
		return false, fmt.Errorf("check event: %w", err)
	}

	log.Printf("[WebhookEventStore] Event %s already processed - skipping", eventID)
	return true, nil
}

// MarkEventProcessed marks a webhook event as processed
func (wes *WebhookEventStore) MarkEventProcessed(ctx context.Context, eventID, eventType string) error {
	log.Printf("[WebhookEventStore] Marking event %s as processed", eventID)

	now := time.Now().UnixMilli()
	event := ProcessedWebhookEvent{
		ID:          eventID,
		Type:        "webhook_event",
		EventType:   eventType,
		ProcessedAt: now,
		CreatedAt:   now,
	}

	switch wes.partitionKeyField {
	case "userId":
		event.UserID = webhookPartitionValue
	case "threadId":
		event.ThreadID = webhookPartitionValue
	default:
		return fmt.Errorf("unsupported partition key field: %s (expected userId or threadId)", wes.partitionKeyField)
	}

	eventBytes, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal webhook event: %w", err)
	}

	pk := azcosmos.NewPartitionKeyString(webhookPartitionValue)

	_, err = wes.container.CreateItem(ctx, pk, eventBytes, nil)
	if err != nil {
		log.Printf("[WebhookEventStore] Failed to mark event as processed: %v", err)
		return fmt.Errorf("create webhook event: %w", err)
	}

	log.Printf("[WebhookEventStore] Event %s marked as processed successfully", eventID)
	return nil
}

// CleanupOldEvents removes processed events older than the specified duration
// This should be called periodically to prevent unbounded growth
func (wes *WebhookEventStore) CleanupOldEvents(ctx context.Context, olderThan time.Duration) error {
	log.Printf("[WebhookEventStore] Cleaning up events older than %v", olderThan)

	cutoff := time.Now().Add(-olderThan).UnixMilli()

	query := fmt.Sprintf(
		"SELECT c.id FROM c WHERE c.type = 'webhook_event' AND c.createdAt < %d",
		cutoff,
	)

	pk := azcosmos.NewPartitionKeyString(webhookPartitionValue)
	queryOptions := azcosmos.QueryOptions{
		QueryParameters: []azcosmos.QueryParameter{},
	}

	queryPager := wes.container.NewQueryItemsPager(query, pk, &queryOptions)

	deletedCount := 0
	for queryPager.More() {
		response, err := queryPager.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("query old events: %w", err)
		}

		for _, item := range response.Items {
			var doc struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal(item, &doc); err != nil {
				log.Printf("[WebhookEventStore] Failed to unmarshal event: %v", err)
				continue
			}

			if _, err := wes.container.DeleteItem(ctx, pk, doc.ID, nil); err != nil {
				log.Printf("[WebhookEventStore] Failed to delete event %s: %v", doc.ID, err)
			} else {
				deletedCount++
			}
		}
	}

	log.Printf("[WebhookEventStore] Cleaned up %d old webhook events", deletedCount)
	return nil
}

// isCosmosNotFound checks if the error is a Cosmos DB 404 not found error
// Uses proper error type checking instead of fragile string matching
func isCosmosNotFound(err error) bool {
	if err == nil {
		return false
	}

	var responseErr *azcore.ResponseError
	if errors.As(err, &responseErr) {
		return responseErr.StatusCode == http.StatusNotFound
	}

	return false
}
