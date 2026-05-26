package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
	"github.com/google/uuid"
)

// AnonymousUserDocument matches the FreeSession structure from frontend
type AnonymousUserDocument struct {
	ID              string               `json:"id"`
	Type            string               `json:"type"`            // "anonymous_user"
	AnonymousUserID string               `json:"anonymousUserId"` // Partition key
	SessionData     AnonymousSessionData `json:"sessionData"`
	CreatedAt       int64                `json:"createdAt"`
	UpdatedAt       int64                `json:"updatedAt"`
	LastAccessedAt  int64                `json:"lastAccessedAt"`
}

// AnonymousSessionData matches the current FreeSession interface exactly
type AnonymousSessionData struct {
	MessageCount    int            `json:"messageCount"`
	StartTime       int64          `json:"startTime"`
	DailyUsage      DailyUsageData `json:"dailyUsage"`
	LeftSessionID   string         `json:"leftSessionId"`
	CenterSessionID string         `json:"centerSessionId"`
	RightSessionID  string         `json:"rightSessionId"`
	AnonymousUserID string         `json:"anonymousUserId"`
}

// DailyUsageData matches frontend structure
type DailyUsageData struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

type AnonymousUserStore struct {
	anonymousUsersContainer *azcosmos.ContainerClient
}

func NewAnonymousUserStore(multiCosmosStore *MultiCosmosStore) (*AnonymousUserStore, error) {
	return &AnonymousUserStore{
		anonymousUsersContainer: multiCosmosStore.GetAnonymousUsersContainer(),
	}, nil
}

func (aus *AnonymousUserStore) GetAnonymousUser(ctx context.Context, anonymousUserID string) (*AnonymousUserDocument, error) {
	log.Printf("[AnonymousUserStore] Getting anonymous user: %s", anonymousUserID)

	query := "SELECT * FROM c WHERE c.type = 'anonymous_user' AND c.anonymousUserId = @anonymousUserId"
	opt := azcosmos.QueryOptions{
		QueryParameters: []azcosmos.QueryParameter{
			{Name: "@anonymousUserId", Value: anonymousUserID},
		},
	}

	queryPager := aus.anonymousUsersContainer.NewQueryItemsPager(query, azcosmos.NewPartitionKeyString(anonymousUserID), &opt)

	for queryPager.More() {
		queryResponse, err := queryPager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("query anonymous user: %w", err)
		}

		for _, item := range queryResponse.Items {
			var user AnonymousUserDocument
			if err := json.Unmarshal(item, &user); err != nil {
				continue
			}

			// Update last accessed time
			user.LastAccessedAt = time.Now().UnixMilli()
			aus.UpdateAnonymousUser(ctx, &user)

			log.Printf("[AnonymousUserStore] ✅ Found anonymous user: %s", anonymousUserID)
			return &user, nil
		}
	}

	log.Printf("[AnonymousUserStore] No anonymous user found: %s", anonymousUserID)
	return nil, fmt.Errorf("anonymous user not found")
}

func (aus *AnonymousUserStore) CreateAnonymousUser(ctx context.Context, anonymousUserID string, sessionData AnonymousSessionData) (*AnonymousUserDocument, error) {
	log.Printf("[AnonymousUserStore] Creating anonymous user: %s", anonymousUserID)

	now := time.Now().UnixMilli()

	user := &AnonymousUserDocument{
		ID:              "anon_" + uuid.NewString(),
		Type:            "anonymous_user",
		AnonymousUserID: anonymousUserID,
		SessionData:     sessionData,
		CreatedAt:       now,
		UpdatedAt:       now,
		LastAccessedAt:  now,
	}

	userBytes, err := json.Marshal(user)
	if err != nil {
		return nil, fmt.Errorf("marshal anonymous user: %w", err)
	}

	pk := azcosmos.NewPartitionKeyString(user.AnonymousUserID)

	_, err = aus.anonymousUsersContainer.CreateItem(ctx, pk, userBytes, nil)
	if err != nil {
		return nil, fmt.Errorf("create anonymous user in cosmos: %w", err)
	}

	log.Printf("[AnonymousUserStore] ✅ Created anonymous user: %s", anonymousUserID)
	return user, nil
}

func (aus *AnonymousUserStore) UpdateAnonymousUser(ctx context.Context, user *AnonymousUserDocument) error {
	user.UpdatedAt = time.Now().UnixMilli()
	user.LastAccessedAt = time.Now().UnixMilli()

	userBytes, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("marshal anonymous user: %w", err)
	}

	pk := azcosmos.NewPartitionKeyString(user.AnonymousUserID)

	_, err = aus.anonymousUsersContainer.ReplaceItem(ctx, pk, user.ID, userBytes, nil)
	if err != nil {
		return fmt.Errorf("update anonymous user in cosmos: %w", err)
	}

	log.Printf("[AnonymousUserStore] ✅ Updated anonymous user: %s", user.AnonymousUserID)
	return nil
}

func (aus *AnonymousUserStore) DeleteAnonymousUser(ctx context.Context, anonymousUserID string) error {
	log.Printf("[AnonymousUserStore] Deleting anonymous user: %s", anonymousUserID)

	user, err := aus.GetAnonymousUser(ctx, anonymousUserID)
	if err != nil {
		return err // User not found
	}

	pk := azcosmos.NewPartitionKeyString(anonymousUserID)

	_, err = aus.anonymousUsersContainer.DeleteItem(ctx, pk, user.ID, nil)
	if err != nil {
		return fmt.Errorf("delete anonymous user from cosmos: %w", err)
	}

	log.Printf("[AnonymousUserStore] ✅ Deleted anonymous user: %s", anonymousUserID)
	return nil
}

func (aus *AnonymousUserStore) GetOrCreateAnonymousUser(ctx context.Context, anonymousUserID string) (*AnonymousUserDocument, error) {
	// Try to get existing user first
	user, err := aus.GetAnonymousUser(ctx, anonymousUserID)
	if err == nil {
		return user, nil
	}

	// Create new user with default session data
	defaultSessionData := AnonymousSessionData{
		MessageCount:    0,
		StartTime:       time.Now().UnixMilli(),
		DailyUsage:      DailyUsageData{Date: "", Count: 0},
		LeftSessionID:   "",
		CenterSessionID: "",
		RightSessionID:  "",
		AnonymousUserID: anonymousUserID,
	}

	return aus.CreateAnonymousUser(ctx, anonymousUserID, defaultSessionData)
}
