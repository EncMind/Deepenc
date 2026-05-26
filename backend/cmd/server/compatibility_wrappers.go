package main

import (
	"context"
)

// UserStoreWrapper makes MultiUserStore compatible with the existing UserStore interface
type UserStoreWrapper struct {
	multiUserStore *MultiUserStore
}

func (usw *UserStoreWrapper) CreateUser(ctx context.Context, firebaseUID, email, displayName, photoURL string) (*User, error) {
	userDoc, err := usw.multiUserStore.CreateUser(ctx, firebaseUID, email, displayName, photoURL)
	if err != nil {
		return nil, err
	}
	return userDoc.ToLegacyUser(), nil
}

func (usw *UserStoreWrapper) GetUser(ctx context.Context, firebaseUID string) (*User, error) {
	userDoc, err := usw.multiUserStore.GetUser(ctx, firebaseUID)
	if err != nil {
		return nil, err
	}
	return userDoc.ToLegacyUser(), nil
}

func (usw *UserStoreWrapper) UpdateUser(ctx context.Context, user *User) error {
	userDoc := &UserDocument{
		ID:            user.ID,
		Type:          user.Type,
		UserID:        user.UserID,
		ThreadID:      user.ThreadID,
		Email:         user.Email,
		DisplayName:   user.DisplayName,
		PhotoURL:      user.PhotoURL,
		CreatedAt:     user.CreatedAt,
		UpdatedAt:     user.UpdatedAt,
		LastLoginAt:   user.LastLoginAt,
		IsActive:      user.IsActive,
		Preferences:   user.Preferences,
		Subscription:  user.Subscription,
		UsageTracking: user.UsageTracking,
		FreeTrial:     user.FreeTrial,
	}
	return usw.multiUserStore.UpdateUser(ctx, userDoc)
}

func (usw *UserStoreWrapper) UpdateLastLogin(ctx context.Context, firebaseUID string) error {
	return usw.multiUserStore.UpdateLastLogin(ctx, firebaseUID)
}

func (usw *UserStoreWrapper) DeactivateUser(ctx context.Context, firebaseUID string) error {
	return usw.multiUserStore.DeactivateUser(ctx, firebaseUID)
}

func (usw *UserStoreWrapper) GetUserPreferences(ctx context.Context, firebaseUID string) (*UserPreferences, error) {
	return usw.multiUserStore.GetUserPreferences(ctx, firebaseUID)
}

func (usw *UserStoreWrapper) UpdateUserPreferences(ctx context.Context, firebaseUID string, preferences *UserPreferences) error {
	return usw.multiUserStore.UpdateUserPreferences(ctx, firebaseUID, preferences)
}

// Subscription and usage management methods (placeholder implementations)
func (usw *UserStoreWrapper) UpdateSubscription(ctx context.Context, firebaseUID string, subscription *Subscription) error {
	// TODO: Implement in MultiUserStore when subscription fields are added to UserDocument
	return usw.multiUserStore.UpdateSubscription(ctx, firebaseUID, subscription)
}

func (usw *UserStoreWrapper) RecordTokenUsage(ctx context.Context, firebaseUID string, provider string, actualTokens int64, equivalentTokens int64) error {
	// TODO: Implement in MultiUserStore when usage tracking fields are added to UserDocument
	return usw.multiUserStore.RecordTokenUsage(ctx, firebaseUID, provider, actualTokens, equivalentTokens)
}

func (usw *UserStoreWrapper) CheckUsageLimits(ctx context.Context, firebaseUID string, estimatedTokens int64) (*UsageValidation, error) {
	// TODO: Implement in MultiUserStore when usage tracking fields are added to UserDocument
	return usw.multiUserStore.CheckUsageLimits(ctx, firebaseUID, estimatedTokens)
}

func (usw *UserStoreWrapper) ResetUsageForBillingPeriod(ctx context.Context, firebaseUID string) error {
	// TODO: Implement in MultiUserStore when usage tracking fields are added to UserDocument
	return usw.multiUserStore.ResetUsageForBillingPeriod(ctx, firebaseUID)
}

// ThreadManagerWrapper makes MultiThreadManager compatible with the existing ThreadManager interface
type ThreadManagerWrapper struct {
	multiThreadManager *MultiThreadManager
}

func (tmw *ThreadManagerWrapper) CreateThread(ctx context.Context, userID, title string) (*Thread, error) {
	return tmw.multiThreadManager.CreateThread(ctx, userID, title)
}

func (tmw *ThreadManagerWrapper) GetThread(ctx context.Context, threadID, userID string) (*Thread, error) {
	return tmw.multiThreadManager.GetThread(ctx, threadID, userID)
}

func (tmw *ThreadManagerWrapper) ListUserThreads(ctx context.Context, userID string, limit int) ([]*Thread, error) {
	return tmw.multiThreadManager.ListUserThreads(ctx, userID, limit)
}

func (tmw *ThreadManagerWrapper) GetThreadMessages(ctx context.Context, threadID, userID string, limit int) ([]*EnhancedMessage, error) {
	return tmw.multiThreadManager.GetThreadMessages(ctx, threadID, userID, limit)
}

func (tmw *ThreadManagerWrapper) SaveMessage(ctx context.Context, userID string, msg *EnhancedMessage) error {
	return tmw.multiThreadManager.SaveMessage(ctx, userID, msg)
}

func (tmw *ThreadManagerWrapper) SearchThreadMessages(ctx context.Context, threadID, userID, query string, limit int) ([]*EnhancedMessage, error) {
	return tmw.multiThreadManager.SearchThreadMessages(ctx, threadID, userID, query, limit)
}

func (tmw *ThreadManagerWrapper) SearchUserThreads(ctx context.Context, userID, query string, limit, offset int) ([]*Thread, error) {
	return tmw.multiThreadManager.SearchUserThreads(ctx, userID, query, limit, offset)
}

func (tmw *ThreadManagerWrapper) RenameThread(ctx context.Context, threadID, userID, newTitle string) error {
	return tmw.multiThreadManager.RenameThread(ctx, threadID, userID, newTitle)
}

func (tmw *ThreadManagerWrapper) DeleteThread(ctx context.Context, threadID, userID string) error {
	return tmw.multiThreadManager.DeleteThread(ctx, threadID, userID)
}
