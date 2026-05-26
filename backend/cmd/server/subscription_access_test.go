package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCancellationMidPeriodWithTokens(t *testing.T) {
	ctx := context.Background()
	mockStore := NewMockUserStore()

	// Setup: User with pro subscription, 7M tokens remaining
	user := &User{
		ID:     "test_user_123",
		UserID: "test_user_123",
		Subscription: Subscription{
			Tier:               "pro",
			Status:             "active",
			CurrentPeriodStart: time.Now().Add(-15 * 24 * time.Hour).Unix(), // 15 days ago
			CurrentPeriodEnd:   time.Now().Add(13 * 24 * time.Hour).Unix(),  // 13 days from now
		},
		UsageTracking: UsageTracking{
			TokensUsed:  3000000, // 3M used
			TokensLimit: 10000000, // 10M limit
		},
	}
	mockStore.users[user.ID] = user

	// Action: Cancel subscription (mid-period)
	err := cancelSubscriptionMidPeriod(ctx, mockStore, user.ID)
	assert.NoError(t, err)

	// Verify: Status updated but tier remains pro
	updatedUser, _ := mockStore.GetUser(ctx, user.ID)
	assert.Equal(t, "canceled", updatedUser.Subscription.Status)
	assert.Equal(t, "pro", updatedUser.Subscription.Tier)
	assert.True(t, updatedUser.Subscription.CancelAtPeriodEnd)

	// Verify: Can still use remaining tokens
	validation, err := mockStore.CheckUsageLimits(ctx, user.ID, 1000000)
	assert.NoError(t, err)
	assert.True(t, validation.IsAllowed, "User should have access with 7M tokens remaining")
	assert.Equal(t, int64(7000000), validation.RemainingTokens)

	// Simulate: User exhausts remaining tokens
	err = mockStore.RecordTokenUsage(ctx, user.ID, "openai", 7000000, 7000000)
	assert.NoError(t, err)

	// Update tokens used in mock
	updatedUser.UsageTracking.TokensUsed = 10000000
	mockStore.users[user.ID] = updatedUser

	// Verify: Access blocked when tokens depleted
	validation, err = mockStore.CheckUsageLimits(ctx, user.ID, 1)
	assert.NoError(t, err)

	// Simulate: Automatic downgrade when tokens reach 0
	err = handleTokenDepletion(ctx, mockStore, user.ID)
	assert.NoError(t, err)

	// Verify: User downgraded to free
	finalUser, _ := mockStore.GetUser(ctx, user.ID)
	assert.Equal(t, "free", finalUser.Subscription.Tier)
}

func TestCancellationWith0Tokens(t *testing.T) {
	ctx := context.Background()
	mockStore := NewMockUserStore()

	// Setup: User with all tokens used
	user := &User{
		ID:     "test_user_456",
		UserID: "test_user_456",
		Subscription: Subscription{
			Tier:   "pro",
			Status: "active",
		},
		UsageTracking: UsageTracking{
			TokensUsed:  10000000,
			TokensLimit: 10000000,
		},
	}
	mockStore.users[user.ID] = user

	// Action: Cancel with 0 tokens
	err := cancelSubscriptionImmediate(ctx, mockStore, user.ID)
	assert.NoError(t, err)

	// Verify: Immediate downgrade
	updatedUser, _ := mockStore.GetUser(ctx, user.ID)
	assert.Equal(t, "free", updatedUser.Subscription.Tier)
	assert.Equal(t, "canceled", updatedUser.Subscription.Status)

	// Verify: Cannot use service
	validation, _ := mockStore.CheckUsageLimits(ctx, user.ID, 1)
	assert.False(t, validation.IsAllowed)
}

func TestCancellationAtPeriodEnd(t *testing.T) {
	ctx := context.Background()
	mockStore := NewMockUserStore()

	// Setup: User at end of period with tokens remaining
	user := &User{
		ID:     "test_user_789",
		UserID: "test_user_789",
		Subscription: Subscription{
			Tier:               "pro",
			Status:             "active",
			CurrentPeriodStart: time.Now().Add(-30 * 24 * time.Hour).Unix(),
			CurrentPeriodEnd:   time.Now().Unix(), // Period ending now
		},
		UsageTracking: UsageTracking{
			TokensUsed:  5000000,  // 5M used
			TokensLimit: 10000000, // 5M remaining (forfeited)
		},
	}
	mockStore.users[user.ID] = user

	// Action: Cancel at period end
	err := cancelSubscriptionImmediate(ctx, mockStore, user.ID)
	assert.NoError(t, err)

	// Verify: Immediate downgrade to free
	updatedUser, _ := mockStore.GetUser(ctx, user.ID)
	assert.Equal(t, "free", updatedUser.Subscription.Tier)
	assert.Equal(t, "canceled", updatedUser.Subscription.Status)
	assert.Equal(t, int64(1000000), updatedUser.UsageTracking.TokensLimit, "Should have free tier limit")

	// Verify: Cannot use remaining 5M tokens (forfeited)
	validation, _ := mockStore.CheckUsageLimits(ctx, user.ID, 2000000)
	assert.False(t, validation.IsAllowed, "Should not allow request exceeding free tier limit")
}

func TestSuccessfulRenewal(t *testing.T) {
	ctx := context.Background()
	mockStore := NewMockUserStore()

	// Setup: User near renewal with 8M tokens used
	user := &User{
		ID:     "test_user_pro",
		UserID: "test_user_pro",
		Subscription: Subscription{
			Tier:               "pro",
			Status:             "active",
			CurrentPeriodStart: time.Now().Add(-30 * 24 * time.Hour).Unix(),
			CurrentPeriodEnd:   time.Now().Unix(), // Ending now
		},
		UsageTracking: UsageTracking{
			TokensUsed:  8000000,
			TokensLimit: 10000000,
		},
	}
	mockStore.users[user.ID] = user

	// Action: Process renewal
	err := processRenewal(ctx, mockStore, user.ID)
	assert.NoError(t, err)

	// Verify: Tier unchanged
	renewedUser, _ := mockStore.GetUser(ctx, user.ID)
	assert.Equal(t, "pro", renewedUser.Subscription.Tier)
	assert.Equal(t, "active", renewedUser.Subscription.Status)

	// Verify: Usage reset
	assert.Equal(t, int64(0), renewedUser.UsageTracking.TokensUsed)
	assert.Equal(t, int64(10000000), renewedUser.UsageTracking.TokensLimit)

	// Verify: Period dates updated
	assert.Greater(t, renewedUser.Subscription.CurrentPeriodEnd, time.Now().Unix())

	// Verify: Can use full quota
	validation, _ := mockStore.CheckUsageLimits(ctx, user.ID, 9000000)
	assert.True(t, validation.IsAllowed)
	assert.Equal(t, int64(10000000), validation.RemainingTokens)
}

func TestFailedRenewalGracePeriod(t *testing.T) {
	ctx := context.Background()
	mockStore := NewMockUserStore()

	// Setup
	user := &User{
		ID:     "test_user_fail",
		UserID: "test_user_fail",
		Subscription: Subscription{
			Tier:   "pro",
			Status: "active",
		},
		UsageTracking: UsageTracking{
			TokensUsed:  5000000,
			TokensLimit: 10000000,
		},
	}
	mockStore.users[user.ID] = user

	// Action: Payment fails
	err := processPaymentFailure(ctx, mockStore, user.ID)
	assert.NoError(t, err)

	// Verify: Status past_due but tier still pro
	failedUser, _ := mockStore.GetUser(ctx, user.ID)
	assert.Equal(t, "past_due", failedUser.Subscription.Status)
	assert.Equal(t, "pro", failedUser.Subscription.Tier)

	// Verify: Still has access during grace period
	validation, _ := mockStore.CheckUsageLimits(ctx, user.ID, 1000000)
	assert.True(t, validation.IsAllowed, "User should have access during grace period")

	// Simulate: Grace period expires
	err = expireGracePeriod(ctx, mockStore, user.ID)
	assert.NoError(t, err)

	// Verify: Downgraded to free
	expiredUser, _ := mockStore.GetUser(ctx, user.ID)
	assert.Equal(t, "free", expiredUser.Subscription.Tier)
	assert.Equal(t, "canceled", expiredUser.Subscription.Status)
}

func TestUsageQuotaResetOnRenewal(t *testing.T) {
	ctx := context.Background()
	mockStore := NewMockUserStore()

	// Setup: User with high usage
	user := &User{
		ID:     "test_user_quota_reset",
		UserID: "test_user_quota_reset",
		Subscription: Subscription{
			Tier:               "pro",
			Status:             "active",
			CurrentPeriodStart: time.Now().Add(-30 * 24 * time.Hour).Unix(),
			CurrentPeriodEnd:   time.Now().Unix(),
		},
		UsageTracking: UsageTracking{
			TokensUsed:  9500000,  // 9.5M used
			TokensLimit: 10000000, // Only 500K remaining
		},
	}
	mockStore.users[user.ID] = user

	// Verify: Only 500K tokens remaining before renewal
	validation, _ := mockStore.CheckUsageLimits(ctx, user.ID, 1000000)
	assert.False(t, validation.IsAllowed, "Should not allow 1M request with only 500K remaining")
	assert.Equal(t, int64(500000), validation.RemainingTokens)

	// Action: Trigger renewal
	err := processRenewal(ctx, mockStore, user.ID)
	assert.NoError(t, err)

	// Verify: Usage reset to 0
	renewedUser, _ := mockStore.GetUser(ctx, user.ID)
	assert.Equal(t, int64(0), renewedUser.UsageTracking.TokensUsed)
	assert.Equal(t, int64(10000000), renewedUser.UsageTracking.TokensLimit)

	// Verify: Can use full quota
	validation, _ = mockStore.CheckUsageLimits(ctx, user.ID, 9000000)
	assert.True(t, validation.IsAllowed)
	assert.Equal(t, int64(10000000), validation.RemainingTokens)
}

func TestGracePeriodRecovery(t *testing.T) {
	ctx := context.Background()
	mockStore := NewMockUserStore()

	// Setup: User in past_due status
	user := &User{
		ID:     "test_user_recovery",
		UserID: "test_user_recovery",
		Subscription: Subscription{
			Tier:   "pro",
			Status: "past_due",
		},
		UsageTracking: UsageTracking{
			TokensUsed:  3000000,
			TokensLimit: 10000000,
		},
	}
	mockStore.users[user.ID] = user

	// Verify: User still has pro access during grace period
	validation, _ := mockStore.CheckUsageLimits(ctx, user.ID, 1000000)
	assert.True(t, validation.IsAllowed)

	// Action: User updates payment method and payment succeeds
	err := processPaymentRecovery(ctx, mockStore, user.ID)
	assert.NoError(t, err)

	// Verify: Back to active status
	recoveredUser, _ := mockStore.GetUser(ctx, user.ID)
	assert.Equal(t, "active", recoveredUser.Subscription.Status)
	assert.Equal(t, "pro", recoveredUser.Subscription.Tier)

	// Verify: Can continue using service
	validation, _ = mockStore.CheckUsageLimits(ctx, user.ID, 5000000)
	assert.True(t, validation.IsAllowed)
}

// Helper functions for subscription management

func cancelSubscriptionMidPeriod(ctx context.Context, store *MockUserStore, userID string) error {
	user, _ := store.GetUser(ctx, userID)
	user.Subscription.Status = "canceled"
	user.Subscription.CancelAtPeriodEnd = true
	return store.UpdateUser(ctx, user)
}

func cancelSubscriptionImmediate(ctx context.Context, store *MockUserStore, userID string) error {
	user, _ := store.GetUser(ctx, userID)
	user.Subscription.Status = "canceled"
	user.Subscription.Tier = "free"
	user.UsageTracking.TokensLimit = 1000000
	return store.UpdateUser(ctx, user)
}

func handleTokenDepletion(ctx context.Context, store *MockUserStore, userID string) error {
	user, _ := store.GetUser(ctx, userID)
	if user.UsageTracking.TokensUsed >= user.UsageTracking.TokensLimit {
		user.Subscription.Tier = "free"
		user.UsageTracking.TokensLimit = 1000000
		return store.UpdateUser(ctx, user)
	}
	return nil
}

func processRenewal(ctx context.Context, store *MockUserStore, userID string) error {
	user, _ := store.GetUser(ctx, userID)

	// Update period dates
	user.Subscription.CurrentPeriodStart = time.Now().Unix()
	user.Subscription.CurrentPeriodEnd = time.Now().Add(30 * 24 * time.Hour).Unix()

	// Reset usage
	user.UsageTracking.TokensUsed = 0
	user.UsageTracking.CurrentPeriodStart = user.Subscription.CurrentPeriodStart

	return store.UpdateUser(ctx, user)
}

func processPaymentFailure(ctx context.Context, store *MockUserStore, userID string) error {
	user, _ := store.GetUser(ctx, userID)
	user.Subscription.Status = "past_due"
	// Tier remains "pro" during grace period
	return store.UpdateUser(ctx, user)
}

func expireGracePeriod(ctx context.Context, store *MockUserStore, userID string) error {
	user, _ := store.GetUser(ctx, userID)
	user.Subscription.Status = "canceled"
	user.Subscription.Tier = "free"
	user.UsageTracking.TokensLimit = 1000000
	return store.UpdateUser(ctx, user)
}

func processPaymentRecovery(ctx context.Context, store *MockUserStore, userID string) error {
	user, _ := store.GetUser(ctx, userID)
	user.Subscription.Status = "active"
	// Tier remains "pro"
	return store.UpdateUser(ctx, user)
}
