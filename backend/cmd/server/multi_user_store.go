package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
	"github.com/google/uuid"
)

// UserDocument for the users container
type UserDocument struct {
	ID            string          `json:"id"`
	Type          string          `json:"type"`     // "user"
	UserID        string          `json:"userId"`   // Firebase UID (partition key)
	ThreadID      string          `json:"threadId"` // For backward compatibility
	Email         string          `json:"email"`
	DisplayName   string          `json:"displayName"`
	PhotoURL      string          `json:"photoURL"`
	CreatedAt     int64           `json:"createdAt"`
	UpdatedAt     int64           `json:"updatedAt"`
	LastLoginAt   int64           `json:"lastLoginAt"`
	IsActive      bool            `json:"isActive"`
	Preferences   UserPreferences `json:"preferences"`
	Subscription  Subscription    `json:"subscription"`
	UsageTracking UsageTracking   `json:"usageTracking"`
	FreeTrial     FreeTrialWindow `json:"freeTrial,omitempty"`
}

type MultiUserStore struct {
	usersContainer *azcosmos.ContainerClient
}

func (u *UserDocument) SyncFreeTrial(now time.Time) bool {
	if u == nil {
		return false
	}

	changed := false
	if u.FreeTrial.ensureWindow(now) {
		changed = true
	}

	tokenCalc := NewTokenCalculator()
	desiredLimit := tokenCalc.GetTokenLimitForTier(u.EffectiveTier(now))
	if u.UsageTracking.TokensLimit != desiredLimit {
		u.UsageTracking.TokensLimit = desiredLimit
		changed = true
	}

	return changed
}

func (u *UserDocument) EffectiveTier(now time.Time) string {
	if u == nil {
		return "free"
	}

	if u.Subscription.StripeSubscriptionID != "" && u.Subscription.Tier != "" {
		return u.Subscription.Tier
	}

	if u.FreeTrial.Active(now) {
		return "pro"
	}

	if u.Subscription.Tier == "" {
		return "free"
	}

	return u.Subscription.Tier
}

func (u *UserDocument) FreeTrialActive(now time.Time) bool {
	if u == nil {
		return false
	}

	if u.Subscription.StripeSubscriptionID != "" {
		return false
	}

	return u.FreeTrial.Active(now)
}

func NewMultiUserStore(multiCosmosStore *MultiCosmosStore) (*MultiUserStore, error) {
	return &MultiUserStore{
		usersContainer: multiCosmosStore.GetUsersContainer(),
	}, nil
}

func (mus *MultiUserStore) CreateUser(ctx context.Context, firebaseUID, email, displayName, photoURL string) (*UserDocument, error) {
	currentTime := time.Now()
	now := currentTime.UnixMilli()

	// Use email as displayName if displayName is empty or "Anonymous User"
	if (displayName == "" || displayName == "Anonymous User") && email != "" {
		displayName = email
	}

	user := &UserDocument{
		ID:          "user_" + uuid.NewString(),
		Type:        "user",
		UserID:      firebaseUID,
		ThreadID:    firebaseUID,
		Email:       email,
		DisplayName: displayName,
		PhotoURL:    photoURL,
		CreatedAt:   now,
		UpdatedAt:   now,
		LastLoginAt: now,
		IsActive:    true,
		Preferences: UserPreferences{
			DefaultModels: DefaultModels{
				OpenAI:    "gpt-4o-mini",
				Anthropic: "claude-3-5-sonnet-20241022",
				Gemini:    "gemini-2.0-flash-exp",
			},
			Theme:                "light",
			Language:             "en",
			NotificationsEnabled: true,
		},
		Subscription: Subscription{
			Tier:                 "free",
			Status:               "active",
			StripeCustomerID:     "",
			StripeSubscriptionID: "",
			CurrentPeriodStart:   now,
			CurrentPeriodEnd:     now + (30 * 24 * 60 * 60 * 1000), // 30 days from now
			CancelAtPeriodEnd:    false,
			CreatedAt:            now,
			UpdatedAt:            now,
		},
		UsageTracking: UsageTracking{
			CurrentPeriodStart: now,
			CurrentPeriodEnd:   now + (30 * 24 * 60 * 60 * 1000), // 30 days from now
			TokensUsed:         0,
			TokensLimit:        NewTokenCalculator().GetTokenLimitForTier("free"),
			ProviderUsage:      make(map[string]ProviderUsage),
			DailyUsage:         []DailyUsage{},
			LastResetAt:        now,
		},
	}

	user.SyncFreeTrial(time.Now())

	userBytes, err := json.Marshal(user)
	if err != nil {
		return nil, fmt.Errorf("marshal user: %w", err)
	}

	pk := azcosmos.NewPartitionKeyString(user.UserID)

	_, err = mus.usersContainer.CreateItem(ctx, pk, userBytes, nil)
	if err != nil {
		return nil, fmt.Errorf("create user in cosmos: %w", err)
	}

	log.Printf("✅ Created user in users container: %s (%s)", user.UserID, user.Email)
	return user, nil
}

func (mus *MultiUserStore) GetUser(ctx context.Context, firebaseUID string) (*UserDocument, error) {
	query := "SELECT * FROM c WHERE c.type = 'user' AND c.userId = @userId"
	opt := azcosmos.QueryOptions{
		QueryParameters: []azcosmos.QueryParameter{
			{Name: "@userId", Value: firebaseUID},
		},
	}

	queryPager := mus.usersContainer.NewQueryItemsPager(query, azcosmos.NewPartitionKeyString(firebaseUID), &opt)

	for queryPager.More() {
		queryResponse, err := queryPager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("query user: %w", err)
		}

		for _, item := range queryResponse.Items {
			var user UserDocument
			if err := json.Unmarshal(item, &user); err != nil {
				continue
			}

			now := time.Now()
			updated := false

			// Ensure billing period is initialized (fix for 12/31/1969 issue)
			if user.UsageTracking.CurrentPeriodEnd <= 0 {
				nowMillis := now.UnixMilli()
				log.Printf("[MultiUserStore] Auto-initializing billing period for user: %s", firebaseUID)
				user.UsageTracking.CurrentPeriodStart = nowMillis
				user.UsageTracking.CurrentPeriodEnd = nowMillis + (30 * 24 * 60 * 60 * 1000) // 30 days from now
				user.UsageTracking.LastResetAt = nowMillis
				updated = true
			}

			if user.SyncFreeTrial(now) {
				updated = true
			}

			if updated {
				if err := mus.UpdateUser(ctx, &user); err != nil {
					log.Printf("[MultiUserStore] Failed to persist complimentary access changes: %v", err)
				}
			}

			return &user, nil
		}
	}

	// User not found - check if this is an anonymous user that should be auto-created
	// Anonymous users have firebaseUID starting with "anon_" or similar patterns
	if len(firebaseUID) > 0 && (strings.HasPrefix(firebaseUID, "anon_") || strings.HasPrefix(firebaseUID, "anonymous_") || len(firebaseUID) > 20) {
		log.Printf("[MultiUserStore] Auto-creating anonymous user for ID: %s", firebaseUID)

		// Create anonymous user with free tier defaults
		anonymousUser, err := mus.CreateUser(ctx, firebaseUID, "", "Anonymous User", "")
		if err != nil {
			log.Printf("[MultiUserStore] Failed to auto-create anonymous user: %v", err)
			return nil, fmt.Errorf("failed to create anonymous user: %w", err)
		}

		log.Printf("[MultiUserStore] ✅ Successfully auto-created anonymous user: %s", firebaseUID)
		return anonymousUser, nil
	}

	return nil, fmt.Errorf("user not found")
}

func (mus *MultiUserStore) UpdateUser(ctx context.Context, user *UserDocument) error {
	user.UpdatedAt = time.Now().UnixMilli()

	userBytes, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("marshal user: %w", err)
	}

	pk := azcosmos.NewPartitionKeyString(user.UserID)

	_, err = mus.usersContainer.ReplaceItem(ctx, pk, user.ID, userBytes, nil)
	if err != nil {
		return fmt.Errorf("update user in cosmos: %w", err)
	}

	return nil
}

func (mus *MultiUserStore) UpdateLastLogin(ctx context.Context, firebaseUID string) error {
	user, err := mus.GetUser(ctx, firebaseUID)
	if err != nil {
		return err
	}

	user.LastLoginAt = time.Now().UnixMilli()
	return mus.UpdateUser(ctx, user)
}

func (mus *MultiUserStore) DeactivateUser(ctx context.Context, firebaseUID string) error {
	user, err := mus.GetUser(ctx, firebaseUID)
	if err != nil {
		return err
	}

	user.IsActive = false
	return mus.UpdateUser(ctx, user)
}

func (mus *MultiUserStore) GetUserPreferences(ctx context.Context, firebaseUID string) (*UserPreferences, error) {
	user, err := mus.GetUser(ctx, firebaseUID)
	if err != nil {
		return nil, err
	}

	return &user.Preferences, nil
}

func (mus *MultiUserStore) UpdateUserPreferences(ctx context.Context, firebaseUID string, preferences *UserPreferences) error {
	user, err := mus.GetUser(ctx, firebaseUID)
	if err != nil {
		return err
	}

	user.Preferences = *preferences
	return mus.UpdateUser(ctx, user)
}

// Subscription and usage management methods
func (mus *MultiUserStore) UpdateSubscription(ctx context.Context, firebaseUID string, subscription *Subscription) error {
	log.Printf("[MultiUserStore] Updating subscription for user: %s to tier: %s", firebaseUID, subscription.Tier)

	user, err := mus.GetUser(ctx, firebaseUID)
	if err != nil {
		return fmt.Errorf("get user for subscription update: %w", err)
	}

	// Update subscription
	user.Subscription = *subscription
	user.UpdatedAt = time.Now().UnixMilli()

	// Update usage tracking limits based on effective tier
	tokenCalc := NewTokenCalculator()
	user.UsageTracking.TokensLimit = tokenCalc.GetTokenLimitForTier(user.EffectiveTier(time.Now()))

	// Initialize billing period if not set (fix for 12/31/1969 issue)
	now := time.Now().UnixMilli()
	if user.UsageTracking.CurrentPeriodEnd <= 0 {
		log.Printf("[MultiUserStore] Initializing billing period for user: %s", firebaseUID)
		user.UsageTracking.CurrentPeriodStart = now
		user.UsageTracking.CurrentPeriodEnd = now + (30 * 24 * 60 * 60 * 1000) // 30 days from now
		user.UsageTracking.LastResetAt = now
	}

	return mus.UpdateUser(ctx, user)
}

func (mus *MultiUserStore) RecordTokenUsage(ctx context.Context, firebaseUID string, provider string, actualTokens int64, equivalentTokens int64) error {
	log.Printf("[MultiUserStore] Recording token usage for user: %s, provider: %s, tokens: %d", firebaseUID, provider, equivalentTokens)

	user, err := mus.GetUser(ctx, firebaseUID)
	if err != nil {
		return fmt.Errorf("get user for token usage recording: %w", err)
	}

	// Initialize provider usage if not exists
	if user.UsageTracking.ProviderUsage == nil {
		user.UsageTracking.ProviderUsage = make(map[string]ProviderUsage)
	}

	// Update provider usage
	tokenCalc := NewTokenCalculator()
	effectiveTier := user.EffectiveTier(time.Now())
	providerUsage := user.UsageTracking.ProviderUsage[provider]
	providerUsage.ActualTokensUsed += actualTokens
	providerUsage.EquivalentTokens += equivalentTokens
	providerUsage.RequestCount++
	providerUsage.ConversionRate = tokenCalc.GetConversionRate(provider, effectiveTier)
	user.UsageTracking.ProviderUsage[provider] = providerUsage

	// Update total usage
	user.UsageTracking.TokensUsed += equivalentTokens
	user.UpdatedAt = time.Now().UnixMilli()

	return mus.UpdateUser(ctx, user)
}

func (mus *MultiUserStore) CheckUsageLimits(ctx context.Context, firebaseUID string, estimatedTokens int64) (*UsageValidation, error) {
	log.Printf("[MultiUserStore] Checking usage limits for user: %s, estimated tokens: %d", firebaseUID, estimatedTokens)

	user, err := mus.GetUser(ctx, firebaseUID)
	if err != nil {
		log.Printf("[MultiUserStore] User not found during usage validation: %s, error: %v", firebaseUID, err)
		// For anonymous users, the GetUser method should have auto-created them
		// If we still get an error, it's likely a real issue, so we should still fail
		return nil, fmt.Errorf("get user for usage validation: %w", err)
	}

	now := time.Now()
	usageTracking := user.UsageTracking
	projectedUsage := usageTracking.TokensUsed + estimatedTokens
	tokenCalc := NewTokenCalculator()
	effectiveTier := user.EffectiveTier(now)
	tokensLimit := tokenCalc.GetTokenLimitForTier(effectiveTier)

	var usagePercentage float64
	if tokensLimit > 0 {
		usagePercentage = float64(usageTracking.TokensUsed) / float64(tokensLimit) * 100
	}

	// Check if user would exceed limits
	isAllowed := projectedUsage <= tokensLimit
	message := ""

	if !isAllowed {
		message = fmt.Sprintf("Request would exceed %s tier limit of %d tokens", effectiveTier, tokensLimit)
	} else if usagePercentage >= 90 {
		message = "Approaching usage limit - consider upgrading your plan"
	} else if usagePercentage >= 75 {
		message = "Usage is at 75% of limit"
	}

	remainingTokens := tokensLimit - usageTracking.TokensUsed
	if remainingTokens < 0 {
		remainingTokens = 0
	}

	return &UsageValidation{
		IsAllowed:       isAllowed,
		RemainingTokens: remainingTokens,
		TokensLimit:     tokensLimit,
		TokensUsed:      usageTracking.TokensUsed,
		UsagePercentage: usagePercentage,
		ResetDate:       usageTracking.CurrentPeriodEnd,
		Message:         message,
		FreeTrial:       user.FreeTrialActive(now),
	}, nil
}

func (mus *MultiUserStore) ResetUsageForBillingPeriod(ctx context.Context, firebaseUID string) error {
	log.Printf("[MultiUserStore] Resetting usage for billing period for user: %s", firebaseUID)

	user, err := mus.GetUser(ctx, firebaseUID)
	if err != nil {
		return fmt.Errorf("get user for usage reset: %w", err)
	}

	currentTime := time.Now()
	now := currentTime.UnixMilli()

	// Reset usage tracking
	user.UsageTracking.TokensUsed = 0
	user.UsageTracking.CurrentPeriodStart = now
	user.UsageTracking.CurrentPeriodEnd = now + (30 * 24 * 60 * 60 * 1000) // 30 days from now
	user.UsageTracking.LastResetAt = now
	user.UsageTracking.ProviderUsage = make(map[string]ProviderUsage)
	user.UsageTracking.DailyUsage = []DailyUsage{}
	user.UpdatedAt = now

	tokenCalc := NewTokenCalculator()
	user.UsageTracking.TokensLimit = tokenCalc.GetTokenLimitForTier(user.EffectiveTier(currentTime))

	log.Printf("[MultiUserStore] Reset usage for user: %s, new billing period: %d - %d",
		firebaseUID, user.UsageTracking.CurrentPeriodStart, user.UsageTracking.CurrentPeriodEnd)

	return mus.UpdateUser(ctx, user)
}

// Conversion method to maintain compatibility with existing User type
func (u *UserDocument) ToLegacyUser() *User {
	return &User{
		ID:            u.ID,
		Type:          u.Type,
		UserID:        u.UserID,
		ThreadID:      u.ThreadID,
		Email:         u.Email,
		DisplayName:   u.DisplayName,
		PhotoURL:      u.PhotoURL,
		CreatedAt:     u.CreatedAt,
		UpdatedAt:     u.UpdatedAt,
		LastLoginAt:   u.LastLoginAt,
		IsActive:      u.IsActive,
		Preferences:   u.Preferences,
		Subscription:  u.Subscription,
		UsageTracking: u.UsageTracking,
		FreeTrial:     u.FreeTrial,
	}
}
