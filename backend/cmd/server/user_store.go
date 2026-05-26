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

type User struct {
	ID          string          `json:"id"`
	Type        string          `json:"type"`
	UserID      string          `json:"userId"`   // Firebase UID
	ThreadID    string          `json:"threadId"` // Same as UserID for partition key compatibility
	Email       string          `json:"email"`
	DisplayName string          `json:"displayName"`
	PhotoURL    string          `json:"photoURL"`
	CreatedAt   int64           `json:"createdAt"`
	UpdatedAt   int64           `json:"updatedAt"`
	LastLoginAt int64           `json:"lastLoginAt"`
	IsActive    bool            `json:"isActive"`
	Preferences UserPreferences `json:"preferences"`

	// NEW: Subscription and usage tracking
	Subscription  Subscription    `json:"subscription"`
	UsageTracking UsageTracking   `json:"usageTracking"`
	FreeTrial     FreeTrialWindow `json:"freeTrial,omitempty"`
}

func (u *User) SyncFreeTrial(now time.Time) bool {
	if u == nil {
		return false
	}

	changed := false
	if u.FreeTrial.ensureWindow(now) {
		changed = true
	}

	effectiveTier := u.EffectiveTier(now)
	tokenCalc := NewTokenCalculator()
	desiredLimit := tokenCalc.GetTokenLimitForTier(effectiveTier)
	if u.UsageTracking.TokensLimit != desiredLimit {
		u.UsageTracking.TokensLimit = desiredLimit
		changed = true
	}

	return changed
}

func (u *User) EffectiveTier(now time.Time) string {
	if u == nil {
		return "free"
	}

	// Paid subscriptions always win.
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

func (u *User) FreeTrialActive(now time.Time) bool {
	if u == nil {
		return false
	}

	// Ignore free trial if a paid plan exists.
	if u.Subscription.StripeSubscriptionID != "" {
		return false
	}

	return u.FreeTrial.Active(now)
}

type UserPreferences struct {
	DefaultModels        DefaultModels `json:"defaultModels"`
	Theme                string        `json:"theme"`
	Language             string        `json:"language"`
	NotificationsEnabled bool          `json:"notificationsEnabled"`
}

type DefaultModels struct {
	OpenAI    string `json:"openai"`
	Anthropic string `json:"anthropic"`
	Gemini    string `json:"gemini"`
}

// Subscription management types
type Subscription struct {
	Tier                 string `json:"tier"`   // "free", "plus", "pro", "pro_plus"
	Status               string `json:"status"` // "active", "canceled", "past_due"
	StripeCustomerID     string `json:"stripeCustomerId"`
	StripeSubscriptionID string `json:"stripeSubscriptionId"`
	CurrentPeriodStart   int64  `json:"currentPeriodStart"`
	CurrentPeriodEnd     int64  `json:"currentPeriodEnd"`
	CancelAtPeriodEnd    bool   `json:"cancelAtPeriodEnd"`
	CreatedAt            int64  `json:"createdAt"`
	UpdatedAt            int64  `json:"updatedAt"`
}

// Usage tracking types
type UsageTracking struct {
	// Current billing period usage
	CurrentPeriodStart int64 `json:"currentPeriodStart"`
	CurrentPeriodEnd   int64 `json:"currentPeriodEnd"`
	TokensUsed         int64 `json:"tokensUsed"`  // GPT-equivalent tokens
	TokensLimit        int64 `json:"tokensLimit"` // Based on tier

	// Per-provider breakdown
	ProviderUsage map[string]ProviderUsage `json:"providerUsage"`

	// Daily usage for analytics
	DailyUsage  []DailyUsage `json:"dailyUsage"`
	LastResetAt int64        `json:"lastResetAt"`
}

type ProviderUsage struct {
	ActualTokensUsed int64   `json:"actualTokensUsed"` // Real tokens consumed
	EquivalentTokens int64   `json:"equivalentTokens"` // GPT-equivalent after conversion
	ConversionRate   float64 `json:"conversionRate"`   // Multiplier used
	RequestCount     int64   `json:"requestCount"`     // Number of API calls
}

type DailyUsage struct {
	Date              string                   `json:"date"`       // "2025-01-15"
	TokensUsed        int64                    `json:"tokensUsed"` // GPT-equivalent
	ProviderBreakdown map[string]ProviderUsage `json:"providerBreakdown"`
}

// UsageValidation represents the result of usage limit checking
type UsageValidation struct {
	IsAllowed       bool    `json:"isAllowed"`
	RemainingTokens int64   `json:"remainingTokens"`
	TokensLimit     int64   `json:"tokensLimit"`
	TokensUsed      int64   `json:"tokensUsed"`
	UsagePercentage float64 `json:"usagePercentage"`
	ResetDate       int64   `json:"resetDate"`
	Message         string  `json:"message"`
	FreeTrial       bool    `json:"freeTrial,omitempty"`
}

type UserStore struct {
	container *azcosmos.ContainerClient
}

func NewUserStore(cosmosStore *CosmosStore) (*UserStore, error) {
	return &UserStore{
		container: cosmosStore.container,
	}, nil
}

func (us *UserStore) CreateUser(ctx context.Context, firebaseUID, email, displayName, photoURL string) (*User, error) {
	log.Printf("[UserStore] Creating user - Firebase UID: %s, Email: %s, DisplayName: %s", firebaseUID, email, displayName)

	now := time.Now().UnixMilli()
	userID := "user_" + uuid.NewString()

	user := &User{
		ID:          userID,
		Type:        "user",
		UserID:      firebaseUID,
		ThreadID:    firebaseUID, // Use same as UserID for partition key
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
                Anthropic: "claude-3-haiku", // Free tier default
                Gemini:    "gemini-2.0-flash-lite",   // Free tier default
            },
			Theme:                "light",
			Language:             "en",
			NotificationsEnabled: true,
		},

		// Initialize with free tier subscription
		Subscription: Subscription{
			Tier:               "free",
			Status:             "active",
			CurrentPeriodStart: now,
			CurrentPeriodEnd:   now + (30 * 24 * 60 * 60 * 1000), // 30 days from now
			CreatedAt:          now,
			UpdatedAt:          now,
		},

		// Initialize usage tracking for free tier
		UsageTracking: UsageTracking{
			CurrentPeriodStart: now,
			CurrentPeriodEnd:   now + (30 * 24 * 60 * 60 * 1000), // 30 days from now
			TokensUsed:         0,
			TokensLimit:        100000, // Free tier limit
			ProviderUsage: map[string]ProviderUsage{
				"openai":    {ConversionRate: 1.0},
				"anthropic": {ConversionRate: 1.0}, // Free tier: 1:1 ratio
				"gemini":    {ConversionRate: 1.0}, // Free tier: 1:1 ratio
			},
			DailyUsage:  []DailyUsage{},
			LastResetAt: now,
		},
	}

	user.SyncFreeTrial(time.UnixMilli(now))

	log.Printf("[UserStore] Generated user document ID: %s", userID)

	userBytes, err := json.Marshal(user)
	if err != nil {
		log.Printf("[UserStore] Failed to marshal user: %v", err)
		return nil, fmt.Errorf("marshal user: %w", err)
	}

	pk := azcosmos.NewPartitionKeyString(user.UserID)
	log.Printf("[UserStore] Using partition key: %s", user.UserID)

	log.Printf("[UserStore] Saving user to Cosmos DB")
	_, err = us.container.CreateItem(ctx, pk, userBytes, nil)
	if err != nil {
		log.Printf("[UserStore] Failed to create user in Cosmos DB: %v", err)
		return nil, fmt.Errorf("create user in cosmos: %w", err)
	}

	log.Printf("[UserStore] ✅ Successfully created user: %s (%s)", user.UserID, user.Email)
	return user, nil
}

func (us *UserStore) GetUser(ctx context.Context, firebaseUID string) (*User, error) {
	log.Printf("[UserStore] Looking up user by Firebase UID: %s", firebaseUID)

	query := "SELECT * FROM c WHERE c.type = 'user' AND c.userId = @userId"
	opt := azcosmos.QueryOptions{
		QueryParameters: []azcosmos.QueryParameter{
			{Name: "@userId", Value: firebaseUID},
		},
	}

	log.Printf("[UserStore] Executing query with partition key: %s", firebaseUID)
	queryPager := us.container.NewQueryItemsPager(query, azcosmos.NewPartitionKeyString(firebaseUID), &opt)

	for queryPager.More() {
		queryResponse, err := queryPager.NextPage(ctx)
		if err != nil {
			log.Printf("[UserStore] Query failed for user %s: %v", firebaseUID, err)
			return nil, fmt.Errorf("query user: %w", err)
		}

		log.Printf("[UserStore] Query returned %d items", len(queryResponse.Items))
		for _, item := range queryResponse.Items {
			var user User
			if err := json.Unmarshal(item, &user); err != nil {
				log.Printf("[UserStore] Failed to unmarshal user item: %v", err)
				continue
			}
			if user.SyncFreeTrial(time.Now()) {
				if err := us.UpdateUser(ctx, &user); err != nil {
					log.Printf("[UserStore] Failed to persist free trial changes for %s: %v", user.UserID, err)
				}
			}
			log.Printf("[UserStore] ✅ Found user: %s (%s)", user.UserID, user.Email)
			return &user, nil
		}
	}

	log.Printf("[UserStore] ❌ User not found: %s", firebaseUID)
	return nil, fmt.Errorf("user not found")
}

func (us *UserStore) UpdateUser(ctx context.Context, user *User) error {
	user.UpdatedAt = time.Now().UnixMilli()

	userBytes, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("marshal user: %w", err)
	}

	pk := azcosmos.NewPartitionKeyString(user.UserID)

	_, err = us.container.ReplaceItem(ctx, pk, user.ID, userBytes, nil)
	if err != nil {
		return fmt.Errorf("update user in cosmos: %w", err)
	}

	return nil
}

func (us *UserStore) UpdateLastLogin(ctx context.Context, firebaseUID string) error {
	log.Printf("[UserStore] Updating last login for user: %s", firebaseUID)

	user, err := us.GetUser(ctx, firebaseUID)
	if err != nil {
		log.Printf("[UserStore] Failed to get user for last login update: %v", err)
		return err
	}

	oldLastLogin := user.LastLoginAt
	user.LastLoginAt = time.Now().UnixMilli()

	log.Printf("[UserStore] Updating last login from %d to %d", oldLastLogin, user.LastLoginAt)

	err = us.UpdateUser(ctx, user)
	if err != nil {
		log.Printf("[UserStore] Failed to update last login: %v", err)
	} else {
		log.Printf("[UserStore] ✅ Successfully updated last login for user: %s", firebaseUID)
	}

	return err
}

func (us *UserStore) DeactivateUser(ctx context.Context, firebaseUID string) error {
	user, err := us.GetUser(ctx, firebaseUID)
	if err != nil {
		return err
	}

	user.IsActive = false
	return us.UpdateUser(ctx, user)
}

func (us *UserStore) GetUserPreferences(ctx context.Context, firebaseUID string) (*UserPreferences, error) {
	user, err := us.GetUser(ctx, firebaseUID)
	if err != nil {
		return nil, err
	}

	return &user.Preferences, nil
}

func (us *UserStore) UpdateUserPreferences(ctx context.Context, firebaseUID string, preferences *UserPreferences) error {
	user, err := us.GetUser(ctx, firebaseUID)
	if err != nil {
		return err
	}

	user.Preferences = *preferences
	return us.UpdateUser(ctx, user)
}

// UpdateSubscription updates a user's subscription information
func (us *UserStore) UpdateSubscription(ctx context.Context, firebaseUID string, subscription *Subscription) error {
	log.Printf("[UserStore] Updating subscription for user: %s, tier: %s", firebaseUID, subscription.Tier)

	user, err := us.GetUser(ctx, firebaseUID)
	if err != nil {
		return fmt.Errorf("get user for subscription update: %w", err)
	}

	subscription.UpdatedAt = time.Now().UnixMilli()
	user.Subscription = *subscription

	// Update token limits and conversion rates based on new tier
	tokenCalc := NewTokenCalculator()
	user.UsageTracking.TokensLimit = tokenCalc.GetTokenLimitForTier(user.EffectiveTier(time.Now()))

	// Update conversion rates for all providers
	for provider := range user.UsageTracking.ProviderUsage {
		newRate := tokenCalc.GetConversionRate(provider, user.EffectiveTier(time.Now()))
		providerUsage := user.UsageTracking.ProviderUsage[provider]
		providerUsage.ConversionRate = newRate
		user.UsageTracking.ProviderUsage[provider] = providerUsage
	}

	log.Printf("[UserStore] Updated subscription - Tier: %s, TokensLimit: %d", subscription.Tier, user.UsageTracking.TokensLimit)
	return us.UpdateUser(ctx, user)
}

// RecordTokenUsage records token usage for a user after an API call
func (us *UserStore) RecordTokenUsage(ctx context.Context, firebaseUID string, provider string, actualTokens int64, equivalentTokens int64) error {
	log.Printf("[UserStore] Recording usage for user %s: %s provider, %d actual tokens, %d equivalent tokens",
		firebaseUID, provider, actualTokens, equivalentTokens)

	user, err := us.GetUser(ctx, firebaseUID)
	if err != nil {
		return fmt.Errorf("get user for usage recording: %w", err)
	}

	now := time.Now()
	today := now.Format("2006-01-02")
	effectiveTier := user.EffectiveTier(now)
	tokenCalc := NewTokenCalculator()

	// Update total usage
	user.UsageTracking.TokensUsed += equivalentTokens

	// Update provider usage
	if user.UsageTracking.ProviderUsage == nil {
		user.UsageTracking.ProviderUsage = make(map[string]ProviderUsage)
	}

	providerUsage := user.UsageTracking.ProviderUsage[provider]
	providerUsage.ActualTokensUsed += actualTokens
	providerUsage.EquivalentTokens += equivalentTokens
	providerUsage.RequestCount++
	providerUsage.ConversionRate = tokenCalc.GetConversionRate(provider, effectiveTier)
	user.UsageTracking.ProviderUsage[provider] = providerUsage

	// Update daily usage
	var todayUsage *DailyUsage
	for i := range user.UsageTracking.DailyUsage {
		if user.UsageTracking.DailyUsage[i].Date == today {
			todayUsage = &user.UsageTracking.DailyUsage[i]
			break
		}
	}

	if todayUsage == nil {
		// Create new daily usage entry
		newDailyUsage := DailyUsage{
			Date:              today,
			TokensUsed:        0,
			ProviderBreakdown: make(map[string]ProviderUsage),
		}
		user.UsageTracking.DailyUsage = append(user.UsageTracking.DailyUsage, newDailyUsage)
		todayUsage = &user.UsageTracking.DailyUsage[len(user.UsageTracking.DailyUsage)-1]
	}

	todayUsage.TokensUsed += equivalentTokens
	if todayUsage.ProviderBreakdown == nil {
		todayUsage.ProviderBreakdown = make(map[string]ProviderUsage)
	}

	dailyProviderUsage := todayUsage.ProviderBreakdown[provider]
	dailyProviderUsage.ActualTokensUsed += actualTokens
	dailyProviderUsage.EquivalentTokens += equivalentTokens
	dailyProviderUsage.RequestCount++
	dailyProviderUsage.ConversionRate = providerUsage.ConversionRate
	todayUsage.ProviderBreakdown[provider] = dailyProviderUsage

	// Keep only last 30 days of daily usage
	if len(user.UsageTracking.DailyUsage) > 30 {
		user.UsageTracking.DailyUsage = user.UsageTracking.DailyUsage[len(user.UsageTracking.DailyUsage)-30:]
	}

	var percent float64
	if user.UsageTracking.TokensLimit > 0 {
		percent = float64(user.UsageTracking.TokensUsed) / float64(user.UsageTracking.TokensLimit) * 100
	}

	log.Printf("[UserStore] Updated usage - Total: %d/%d tokens (%.1f%%)",
		user.UsageTracking.TokensUsed, user.UsageTracking.TokensLimit, percent)

	return us.UpdateUser(ctx, user)
}

// CheckUsageLimits validates if a user can make an API call with estimated token usage
func (us *UserStore) CheckUsageLimits(ctx context.Context, firebaseUID string, estimatedTokens int64) (*UsageValidation, error) {
	user, err := us.GetUser(ctx, firebaseUID)
	if err != nil {
		return nil, fmt.Errorf("get user for usage check: %w", err)
	}

	// Check if billing period needs reset
	now := time.Now().UnixMilli()
	if now > user.UsageTracking.CurrentPeriodEnd {
		log.Printf("[UserStore] Billing period expired for user %s, resetting usage", firebaseUID)
		if err := us.ResetUsageForBillingPeriod(ctx, firebaseUID); err != nil {
			log.Printf("[UserStore] Failed to reset billing period: %v", err)
		}
		// Re-fetch user after reset
		user, err = us.GetUser(ctx, firebaseUID)
		if err != nil {
			return nil, fmt.Errorf("get user after reset: %w", err)
		}
	}

	tokensUsed := user.UsageTracking.TokensUsed
	effectiveTier := user.EffectiveTier(time.Now())
	tokenCalc := NewTokenCalculator()
	tokensLimit := tokenCalc.GetTokenLimitForTier(effectiveTier)
	remainingTokens := tokensLimit - tokensUsed
	var usagePercentage float64
	if tokensLimit > 0 {
		usagePercentage = float64(tokensUsed) / float64(tokensLimit) * 100
	}

	validation := &UsageValidation{
		IsAllowed:       remainingTokens >= estimatedTokens,
		RemainingTokens: remainingTokens,
		TokensLimit:     tokensLimit,
		TokensUsed:      tokensUsed,
		UsagePercentage: usagePercentage,
		ResetDate:       user.UsageTracking.CurrentPeriodEnd,
		FreeTrial:       user.FreeTrialActive(time.Now()),
	}

	if !validation.IsAllowed {
		validation.Message = fmt.Sprintf("Usage limit exceeded. %d tokens used of %d limit. Estimated request needs %d tokens.",
			tokensUsed, tokensLimit, estimatedTokens)
		log.Printf("[UserStore] Usage limit check failed for user %s: %s", firebaseUID, validation.Message)
	} else {
		validation.Message = "Usage within limits"
		log.Printf("[UserStore] Usage check passed for user %s: %d/%d tokens used (%.1f%%)",
			firebaseUID, tokensUsed, tokensLimit, usagePercentage)
	}

	return validation, nil
}

// ResetUsageForBillingPeriod resets usage counters for a new billing period
func (us *UserStore) ResetUsageForBillingPeriod(ctx context.Context, firebaseUID string) error {
	log.Printf("[UserStore] Resetting usage for new billing period - user: %s", firebaseUID)

	user, err := us.GetUser(ctx, firebaseUID)
	if err != nil {
		return fmt.Errorf("get user for usage reset: %w", err)
	}

	now := time.Now().UnixMilli()

	// Reset usage counters
	user.UsageTracking.TokensUsed = 0
	user.UsageTracking.CurrentPeriodStart = now
	user.UsageTracking.CurrentPeriodEnd = now + (30 * 24 * 60 * 60 * 1000) // 30 days from now
	user.UsageTracking.LastResetAt = now

	// Reset provider usage
	for provider, providerUsage := range user.UsageTracking.ProviderUsage {
		providerUsage.ActualTokensUsed = 0
		providerUsage.EquivalentTokens = 0
		providerUsage.RequestCount = 0
		// Keep conversion rate
		user.UsageTracking.ProviderUsage[provider] = providerUsage
	}

	// Archive current daily usage and start fresh
	user.UsageTracking.DailyUsage = []DailyUsage{}

	tokenCalc := NewTokenCalculator()
	user.UsageTracking.TokensLimit = tokenCalc.GetTokenLimitForTier(user.EffectiveTier(time.Now()))

	log.Printf("[UserStore] Reset usage period: %d to %d", user.UsageTracking.CurrentPeriodStart, user.UsageTracking.CurrentPeriodEnd)
	return us.UpdateUser(ctx, user)
}
