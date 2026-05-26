package main

import "context"

// UserStoreInterface defines the interface for user storage operations
type UserStoreInterface interface {
	CreateUser(ctx context.Context, firebaseUID, email, displayName, photoURL string) (*User, error)
	GetUser(ctx context.Context, firebaseUID string) (*User, error)
	UpdateUser(ctx context.Context, user *User) error
	UpdateLastLogin(ctx context.Context, firebaseUID string) error
	DeactivateUser(ctx context.Context, firebaseUID string) error
	GetUserPreferences(ctx context.Context, firebaseUID string) (*UserPreferences, error)
	UpdateUserPreferences(ctx context.Context, firebaseUID string, preferences *UserPreferences) error

	// Subscription and usage management
	UpdateSubscription(ctx context.Context, firebaseUID string, subscription *Subscription) error
	RecordTokenUsage(ctx context.Context, firebaseUID string, provider string, actualTokens int64, equivalentTokens int64) error
	CheckUsageLimits(ctx context.Context, firebaseUID string, estimatedTokens int64) (*UsageValidation, error)
	ResetUsageForBillingPeriod(ctx context.Context, firebaseUID string) error
}

// ThreadManagerInterface defines the interface for thread management operations
type ThreadManagerInterface interface {
	CreateThread(ctx context.Context, userID, title string) (*Thread, error)
	GetThread(ctx context.Context, threadID, userID string) (*Thread, error)
	ListUserThreads(ctx context.Context, userID string, limit int) ([]*Thread, error)
	SearchUserThreads(ctx context.Context, userID, query string, limit, offset int) ([]*Thread, error)
	RenameThread(ctx context.Context, threadID, userID, newTitle string) error
	DeleteThread(ctx context.Context, threadID, userID string) error
	GetThreadMessages(ctx context.Context, threadID, userID string, limit int) ([]*EnhancedMessage, error)
	SaveMessage(ctx context.Context, userID string, msg *EnhancedMessage) error
	SearchThreadMessages(ctx context.Context, threadID, userID, query string, limit int) ([]*EnhancedMessage, error)
}
