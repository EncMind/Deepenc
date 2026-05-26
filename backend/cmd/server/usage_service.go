package main

import (
	"context"
	"fmt"
	"log"
	"time"
)

// UsageService handles usage enforcement and model access control
type UsageService struct {
	userStore UserStoreInterface
	tokenCalc *TokenCalculator
}

// NewUsageService creates a new usage enforcement service
func NewUsageService(userStore UserStoreInterface) *UsageService {
	return &UsageService{
		userStore: userStore,
		tokenCalc: NewTokenCalculator(),
	}
}

// ValidateRequest checks if a user can make an API request with the specified model and estimated tokens
func (us *UsageService) ValidateRequest(ctx context.Context, firebaseUID string, model string, estimatedTokens int64) (*RequestValidation, error) {
	log.Printf("[UsageService] Validating request - User: %s, Model: %s, EstimatedTokens: %d", firebaseUID, model, estimatedTokens)

	// Get user to check tier and usage
	user, err := us.userStore.GetUser(ctx, firebaseUID)
	if err != nil {
		return nil, fmt.Errorf("get user for validation: %w", err)
	}

	validation := &RequestValidation{
		IsAllowed: false,
		Reason:    "",
		User:      user,
	}

	// Check if user is active
	if !user.IsActive {
		validation.Reason = "User account is deactivated"
		log.Printf("[UsageService] Request denied - inactive user: %s", firebaseUID)
		return validation, nil
	}

	now := time.Now()
	effectiveTier := user.EffectiveTier(now)

	// Check if model is allowed for user's tier
	if !us.tokenCalc.IsModelAllowedForTier(model, effectiveTier) {
		validation.Reason = fmt.Sprintf("Model '%s' not available for %s tier", model, effectiveTier)
		log.Printf("[UsageService] Request denied - model not allowed: %s for tier %s", model, effectiveTier)
		return validation, nil
	}

	// Get provider from model name
	provider := us.tokenCalc.getProviderFromModel(model)
	if provider == "" {
		validation.Reason = fmt.Sprintf("Unknown model provider for model: %s", model)
		log.Printf("[UsageService] Request denied - unknown provider for model: %s", model)
		return validation, nil
	}

	// Convert estimated tokens to GPT-equivalent
	conversionRate := us.tokenCalc.GetConversionRate(provider, effectiveTier)
	equivalentTokens := int64(float64(estimatedTokens) * conversionRate)

	// Check usage limits
	usageValidation, err := us.userStore.CheckUsageLimits(ctx, firebaseUID, equivalentTokens)
	if err != nil {
		return nil, fmt.Errorf("check usage limits: %w", err)
	}

	if !usageValidation.IsAllowed {
		validation.Reason = usageValidation.Message
		validation.UsageInfo = usageValidation
		log.Printf("[UsageService] Request denied - usage limits exceeded: %s", usageValidation.Message)
		return validation, nil
	}

	// All checks passed
	validation.IsAllowed = true
	validation.Reason = "Request approved"
	validation.UsageInfo = usageValidation
	validation.Provider = provider
	validation.ConversionRate = conversionRate
	validation.EstimatedEquivalentTokens = equivalentTokens

	log.Printf("[UsageService] Request approved - User: %s, Provider: %s, Rate: %.2f, EstimatedEquivalent: %d",
		firebaseUID, provider, conversionRate, equivalentTokens)

	return validation, nil
}

// RecordUsage records actual token usage after an API call completes
func (us *UsageService) RecordUsage(ctx context.Context, firebaseUID string, provider string, actualTokens int64) error {
	log.Printf("[UsageService] Recording actual usage - User: %s, Provider: %s, ActualTokens: %d", firebaseUID, provider, actualTokens)

	// Get user to determine conversion rate
	user, err := us.userStore.GetUser(ctx, firebaseUID)
	if err != nil {
		return fmt.Errorf("get user for usage recording: %w", err)
	}

	// Calculate equivalent tokens
	equivalentTokens := us.tokenCalc.CalculateEquivalentTokens(provider, actualTokens, user.EffectiveTier(time.Now()))

	// Record the usage
	return us.userStore.RecordTokenUsage(ctx, firebaseUID, provider, actualTokens, equivalentTokens)
}

// RecordCompleteUsage records full token usage including input, context, and output tokens
func (us *UsageService) RecordCompleteUsage(ctx context.Context, firebaseUID string, provider string, inputTokens, outputTokens int64) error {
	totalTokens := inputTokens + outputTokens
	log.Printf("[UsageService] Recording complete usage - User: %s, Provider: %s, InputTokens: %d, OutputTokens: %d, TotalTokens: %d",
		firebaseUID, provider, inputTokens, outputTokens, totalTokens)

	// Get user to determine conversion rate
	user, err := us.userStore.GetUser(ctx, firebaseUID)
	if err != nil {
		return fmt.Errorf("get user for usage recording: %w", err)
	}

	// Calculate equivalent tokens for total usage
	equivalentTokens := us.tokenCalc.CalculateEquivalentTokens(provider, totalTokens, user.EffectiveTier(time.Now()))

	// Record the usage
	return us.userStore.RecordTokenUsage(ctx, firebaseUID, provider, totalTokens, equivalentTokens)
}

// GetUserUsageStats returns current usage statistics for a user
func (us *UsageService) GetUserUsageStats(ctx context.Context, firebaseUID string) (*UserUsageStats, error) {
	user, err := us.userStore.GetUser(ctx, firebaseUID)
	if err != nil {
		return nil, fmt.Errorf("get user for usage stats: %w", err)
	}

	effectiveTier := user.EffectiveTier(time.Now())
	tokensLimit := us.tokenCalc.GetTokenLimitForTier(effectiveTier)
	remainingTokens := tokensLimit - user.UsageTracking.TokensUsed
	if remainingTokens < 0 {
		remainingTokens = 0
	}

	var usagePercentage float64
	if tokensLimit > 0 {
		usagePercentage = float64(user.UsageTracking.TokensUsed) / float64(tokensLimit) * 100
	}

	stats := &UserUsageStats{
		TokensUsed:       user.UsageTracking.TokensUsed,
		TokensLimit:      tokensLimit,
		RemainingTokens:  remainingTokens,
		UsagePercentage:  usagePercentage,
		CurrentTier:      effectiveTier,
		FreeTrial:        user.FreeTrialActive(time.Now()),
		BillingPeriodEnd: user.UsageTracking.CurrentPeriodEnd,
		ProviderUsage:    user.UsageTracking.ProviderUsage,
		AllowedModels:    us.tokenCalc.GetAllowedModelsForTier(effectiveTier),
	}

	return stats, nil
}

// GetTierInfo returns information about all available tiers
func (us *UsageService) GetTierInfo() []TierInfo {
	return us.tokenCalc.GetAllTierInfo()
}

// RequestValidation contains the result of request validation
type RequestValidation struct {
	IsAllowed                 bool             `json:"isAllowed"`
	Reason                    string           `json:"reason"`
	Provider                  string           `json:"provider,omitempty"`
	ConversionRate            float64          `json:"conversionRate,omitempty"`
	EstimatedEquivalentTokens int64            `json:"estimatedEquivalentTokens,omitempty"`
	UsageInfo                 *UsageValidation `json:"usageInfo,omitempty"`
	User                      *User            `json:"user,omitempty"`
}

// UserUsageStats contains comprehensive usage statistics for a user
type UserUsageStats struct {
	TokensUsed       int64                    `json:"tokensUsed"`
	TokensLimit      int64                    `json:"tokensLimit"`
	RemainingTokens  int64                    `json:"remainingTokens"`
	UsagePercentage  float64                  `json:"usagePercentage"`
	CurrentTier      string                   `json:"currentTier"`
	FreeTrial        bool                     `json:"freeTrial"`
	BillingPeriodEnd int64                    `json:"billingPeriodEnd"`
	ProviderUsage    map[string]ProviderUsage `json:"providerUsage"`
	AllowedModels    map[string][]string      `json:"allowedModels"`
}
