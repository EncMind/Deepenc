package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"
)

type FirebaseAuth struct {
	client *auth.Client
}

type AuthClaims struct {
	UserID        string `json:"user_id"`
	Email         string `json:"email"`
	DisplayName   string `json:"name"`
	PhotoURL      string `json:"picture"`
	EmailVerified bool   `json:"email_verified"`
}

func NewFirebaseAuth(ctx context.Context) (*FirebaseAuth, error) {
	log.Printf("[Firebase] Initializing Firebase Auth")

	// Try to get credentials from environment or service account file
	var opt option.ClientOption
	var credSource string

	if serviceAccountPath := os.Getenv("FIREBASE_SERVICE_ACCOUNT_PATH"); serviceAccountPath != "" {
		log.Printf("[Firebase] Using service account from path: %s", serviceAccountPath)
		opt = option.WithCredentialsFile(serviceAccountPath)
		credSource = "file path"
	} else if serviceAccountJSON := os.Getenv("FIREBASE_SERVICE_ACCOUNT_JSON"); serviceAccountJSON != "" {
		log.Printf("[Firebase] Using service account from environment JSON (length: %d)", len(serviceAccountJSON))
		opt = option.WithCredentialsJSON([]byte(serviceAccountJSON))
		credSource = "environment JSON"
	} else {
		log.Printf("[Firebase] Using default service account file: firebase-service-account.json")
		opt = option.WithCredentialsFile("firebase-service-account.json")
		credSource = "default file"
	}

	projectID := os.Getenv("FIREBASE_PROJECT_ID")
	log.Printf("[Firebase] Project ID: %s", projectID)

	config := &firebase.Config{
		ProjectID: projectID,
	}

	log.Printf("[Firebase] Creating Firebase app with credentials from %s", credSource)
	app, err := firebase.NewApp(ctx, config, opt)
	if err != nil {
		log.Printf("[Firebase] Failed to initialize Firebase app: %v", err)
		return nil, fmt.Errorf("initialize firebase app: %w", err)
	}

	log.Printf("[Firebase] Getting Auth client")
	authClient, err := app.Auth(ctx)
	if err != nil {
		return nil, fmt.Errorf("initialize firebase auth: %w", err)
	}

	log.Println("✅ Firebase Auth initialized")
	return &FirebaseAuth{client: authClient}, nil
}

func (fa *FirebaseAuth) VerifyIDToken(ctx context.Context, idToken string) (*AuthClaims, error) {
	log.Printf("[Firebase] Verifying ID token (length: %d)", len(idToken))

	token, err := fa.client.VerifyIDToken(ctx, idToken)
	if err != nil {
		log.Printf("[Firebase] Token verification failed: %v", err)
		return nil, fmt.Errorf("verify token: %w", err)
	}

	email := getStringClaim(token.Claims, "email")
	displayName := getStringClaim(token.Claims, "name")
	emailVerified := getBoolClaim(token.Claims, "email_verified")

	// Check if user signed in with a federated provider (Google, etc.)
	signInProvider := getStringClaim(token.Claims, "firebase")
	if signInProviderMap, ok := token.Claims["firebase"].(map[string]interface{}); ok {
		if identities, ok := signInProviderMap["identities"].(map[string]interface{}); ok {
			// If user has google.com in their identities, they're a Google OAuth user
			if _, hasGoogle := identities["google.com"]; hasGoogle {
				log.Printf("[Firebase] User signed in with Google OAuth, marking email as verified")
				emailVerified = true
			}
		}
		signInProvider = getStringClaim(signInProviderMap, "sign_in_provider")
		log.Printf("[Firebase] Sign in provider: %s", signInProvider)
	}

    log.Printf("[Firebase] ✅ Token verified for user %s (verified: %v)", token.UID, emailVerified)

	claims := &AuthClaims{
		UserID:        token.UID,
		Email:         email,
		DisplayName:   displayName,
		PhotoURL:      getStringClaim(token.Claims, "picture"),
		EmailVerified: emailVerified,
	}

	return claims, nil
}

func (fa *FirebaseAuth) GetUser(ctx context.Context, uid string) (*auth.UserRecord, error) {
	user, err := fa.client.GetUser(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return user, nil
}

func (fa *FirebaseAuth) CreateUser(ctx context.Context, email, password, displayName string) (*auth.UserRecord, error) {
	params := (&auth.UserToCreate{}).
		Email(email).
		Password(password).
		DisplayName(displayName).
		EmailVerified(false)

	user, err := fa.client.CreateUser(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return user, nil
}

func (fa *FirebaseAuth) UpdateUser(ctx context.Context, uid string, displayName, photoURL *string) (*auth.UserRecord, error) {
	params := &auth.UserToUpdate{}

	if displayName != nil {
		params = params.DisplayName(*displayName)
	}
	if photoURL != nil {
		params = params.PhotoURL(*photoURL)
	}

	user, err := fa.client.UpdateUser(ctx, uid, params)
	if err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}

	return user, nil
}

func (fa *FirebaseAuth) DeleteUser(ctx context.Context, uid string) error {
	err := fa.client.DeleteUser(ctx, uid)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}

func (fa *FirebaseAuth) ExtractTokenFromHeader(authHeader string) string {
	if authHeader == "" {
		return ""
	}

	// Support both "Bearer <token>" and just "<token>"
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}

	return authHeader
}

// Helper functions
func getStringClaim(claims map[string]interface{}, key string) string {
	if val, ok := claims[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getBoolClaim(claims map[string]interface{}, key string) bool {
	if val, ok := claims[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}

func (fa *FirebaseAuth) HealthCheck(ctx context.Context) error {
	// Try to list users with limit 1 to check if Firebase is accessible
	iter := fa.client.Users(ctx, "")
	paging, err := iter.Next()
	if err != nil && err.Error() != "no more items in iterator" {
		return fmt.Errorf("firebase health check failed: %w", err)
	}

	// If we get here without error, Firebase is accessible
	_ = paging // Just to use the variable
	return nil
}
