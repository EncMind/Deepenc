package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
)

// FirebaseClientConfig contains the client-side Firebase configuration
type FirebaseClientConfig struct {
	APIKey            string `json:"apiKey"`
	AuthDomain        string `json:"authDomain"`
	ProjectID         string `json:"projectId"`
	StorageBucket     string `json:"storageBucket"`
	MessagingSenderID string `json:"messagingSenderId"`
	AppID             string `json:"appId"`
	MeasurementID     string `json:"measurementId"`
}

// ConfigService manages application configuration
type ConfigService struct {
	firebaseConfig       FirebaseClientConfig
	corsConfig           CORSConfig
	stripePublishableKey string
}

// CORSConfig holds CORS configuration
type CORSConfig struct {
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
	MaxAge         int
}

// NewConfigService creates a new configuration service
func NewConfigService() *ConfigService {
	return &ConfigService{
		firebaseConfig:       loadFirebaseConfig(),
		corsConfig:           loadCORSConfig(),
		stripePublishableKey: loadStripePublishableKey(),
	}
}

// loadStripePublishableKey loads Stripe publishable key from environment
func loadStripePublishableKey() string {
	key := os.Getenv("STRIPE_PUBLISHABLE_KEY")
	if key == "" {
		log.Println("⚠️  STRIPE_PUBLISHABLE_KEY not set - Stripe functionality will be unavailable")
	}
	return key
}

// loadFirebaseConfig loads Firebase configuration from environment variables
func loadFirebaseConfig() FirebaseClientConfig {
	// Load all Firebase configuration from environment variables
	config := FirebaseClientConfig{
		APIKey:            os.Getenv("FIREBASE_API_KEY"),
		AuthDomain:        os.Getenv("FIREBASE_AUTH_DOMAIN"),
		ProjectID:         os.Getenv("FIREBASE_PROJECT_ID"),
		StorageBucket:     os.Getenv("FIREBASE_STORAGE_BUCKET"),
		MessagingSenderID: os.Getenv("FIREBASE_MESSAGING_SENDER_ID"),
		AppID:             os.Getenv("FIREBASE_APP_ID"),
		MeasurementID:     os.Getenv("FIREBASE_MEASUREMENT_ID"),
	}

	// Validate required fields (frontend needs these for Firebase SDK initialization)
	var missing []string
	if config.APIKey == "" {
		missing = append(missing, "FIREBASE_API_KEY")
	}
	if config.AuthDomain == "" {
		missing = append(missing, "FIREBASE_AUTH_DOMAIN")
	}
	if config.ProjectID == "" {
		missing = append(missing, "FIREBASE_PROJECT_ID")
	}
	if config.StorageBucket == "" {
		missing = append(missing, "FIREBASE_STORAGE_BUCKET")
	}
	if config.MessagingSenderID == "" {
		missing = append(missing, "FIREBASE_MESSAGING_SENDER_ID")
	}
	if config.AppID == "" {
		missing = append(missing, "FIREBASE_APP_ID")
	}

	if len(missing) > 0 {
		log.Fatalf("❌ Firebase configuration error: Missing required environment variables: %v\n"+
			"Frontend requires these for Firebase SDK initialization.\n"+
			"Please set all Firebase variables in your .env file", missing)
	}

	log.Printf("✅ Firebase configuration loaded successfully (Project: %s)", config.ProjectID)
	return config
}

// loadCORSConfig loads CORS configuration from environment variables
func loadCORSConfig() CORSConfig {
	allowedOrigins := envOr("ALLOWED_ORIGINS", "http://localhost:4200,https://localhost:4200")

	origins := strings.Split(allowedOrigins, ",")
	if os.Getenv("ENVIRONMENT") == "production" {
		origins = filterLocalhostOrigins(origins)
	}

	return CORSConfig{
		AllowedOrigins: origins,
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Authorization", "Content-Type", "X-Requested-With"},
		MaxAge:         3600,
	}
}

// filterLocalhostOrigins removes localhost origins for production security
func filterLocalhostOrigins(origins []string) []string {
	var filtered []string
	for _, origin := range origins {
		origin = strings.TrimSpace(origin)
		if !strings.Contains(origin, "localhost") && !strings.Contains(origin, "127.0.0.1") {
			filtered = append(filtered, origin)
		} else {
			log.Printf("Filtered out localhost origin in production: %s", origin)
		}
	}

	if len(filtered) == 0 {
		log.Fatal("No valid origins configured for production. Please set ALLOWED_ORIGINS environment variable.")
	}

	return filtered
}

// GetFirebaseConfig returns the safe Firebase configuration for clients
func (cs *ConfigService) GetFirebaseConfig() FirebaseClientConfig {
	return cs.firebaseConfig
}

// GetCORSConfig returns the CORS configuration
func (cs *ConfigService) GetCORSConfig() CORSConfig {
	return cs.corsConfig
}

// IsFirebaseConfigValid validates that the Firebase configuration is complete
func (cs *ConfigService) IsFirebaseConfigValid() bool {
	config := cs.firebaseConfig
	return config.ProjectID != "" &&
		   config.AuthDomain != "" &&
		   config.StorageBucket != "" &&
		   config.MessagingSenderID != "" &&
		   config.AppID != ""
}

// handleFirebaseConfig serves the safe Firebase configuration to clients
func (cs *ConfigService) handleFirebaseConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	if err := json.NewEncoder(w).Encode(cs.firebaseConfig); err != nil {
		log.Printf("Error encoding Firebase config: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("Served Firebase configuration to client")
}

// handleStripeConfig serves the Stripe publishable key to clients
func (cs *ConfigService) handleStripeConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	response := map[string]string{
		"publishableKey": cs.stripePublishableKey,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding Stripe config: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("Served Stripe configuration to client")
}

