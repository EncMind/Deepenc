package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
)

type AuthContext struct {
	UserID      string
	Email       string
	DisplayName string
	PhotoURL    string
	IsVerified  bool
}

type contextKey string

const (
	AuthContextKey contextKey = "auth_context"
)

type AuthMiddleware struct {
	firebaseAuth *FirebaseAuth
	userStore    UserStoreInterface
}

func NewAuthMiddleware(firebaseAuth *FirebaseAuth, userStore UserStoreInterface) *AuthMiddleware {
	return &AuthMiddleware{
		firebaseAuth: firebaseAuth,
		userStore:    userStore,
	}
}

func (am *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[AuthMiddleware] RequireAuth: %s %s", r.Method, r.URL.Path)

		authContext := am.authenticate(r)
		if authContext == nil {
			log.Printf("[AuthMiddleware] RequireAuth: Authentication failed for %s %s", r.Method, r.URL.Path)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		log.Printf("[AuthMiddleware] RequireAuth: Success for user %s on %s %s", authContext.UserID, r.Method, r.URL.Path)

		// Add auth context to request
		ctx := context.WithValue(r.Context(), AuthContextKey, authContext)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (am *AuthMiddleware) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[AuthMiddleware] OptionalAuth: %s %s", r.Method, r.URL.Path)

		authContext := am.authenticate(r)
		if authContext != nil {
			log.Printf("[AuthMiddleware] OptionalAuth: Authenticated user %s for %s %s", authContext.UserID, r.Method, r.URL.Path)
		} else {
			log.Printf("[AuthMiddleware] OptionalAuth: Anonymous request for %s %s", r.Method, r.URL.Path)
		}

		// Add auth context to request (can be nil)
		ctx := context.WithValue(r.Context(), AuthContextKey, authContext)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (am *AuthMiddleware) authenticate(r *http.Request) *AuthContext {
	log.Printf("[Auth] Authenticating request: %s %s", r.Method, r.URL.Path)

	// Try Authorization header first
	authHeader := r.Header.Get("Authorization")
	token := am.firebaseAuth.ExtractTokenFromHeader(authHeader)

	// Fallback to X-Auth-Token header
	if token == "" {
		token = r.Header.Get("X-Auth-Token")
		if token != "" {
			log.Printf("[Auth] Using X-Auth-Token header")
		}
	} else {
		log.Printf("[Auth] Using Authorization header")
	}

	if token == "" {
		log.Printf("[Auth] No token found in headers")
		return nil
	}

	log.Printf("[Auth] Verifying token (length: %d)", len(token))
	claims, err := am.firebaseAuth.VerifyIDToken(r.Context(), token)
	if err != nil {
		log.Printf("[Auth] Token verification failed: %v", err)
		return nil
	}

	log.Printf("[Auth] Token verified for user: %s (email: %s, verified: %v)", claims.UserID, claims.Email, claims.EmailVerified)

	// Update last login time
	go func() {
		ctx := context.Background()
		log.Printf("[Auth] Updating last login for user: %s", claims.UserID)
		if err := am.userStore.UpdateLastLogin(ctx, claims.UserID); err != nil {
			log.Printf("[Auth] Failed to update last login for user %s: %v", claims.UserID, err)
		} else {
			log.Printf("[Auth] Successfully updated last login for user: %s", claims.UserID)
		}
	}()

	authCtx := &AuthContext{
		UserID:      claims.UserID,
		Email:       claims.Email,
		DisplayName: claims.DisplayName,
		PhotoURL:    claims.PhotoURL,
		IsVerified:  claims.EmailVerified,
	}

	log.Printf("[Auth] Created auth context for user: %s", claims.UserID)
	return authCtx
}

func GetAuthContext(ctx context.Context) *AuthContext {
	if authCtx, ok := ctx.Value(AuthContextKey).(*AuthContext); ok {
		return authCtx
	}
	return nil
}

func GetUserIDFromContext(ctx context.Context) string {
	if authCtx := GetAuthContext(ctx); authCtx != nil {
		return authCtx.UserID
	}
	return ""
}

// Auth handlers
func (am *AuthMiddleware) handleRegister(w http.ResponseWriter, r *http.Request) {
	log.Printf("[Register] Starting registration process")

	if r.Method != http.MethodPost {
		log.Printf("[Register] Invalid method: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get authenticated context (user already created in Firebase)
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		log.Printf("[Register] No auth context found")
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	log.Printf("[Register] Auth context found for user: %s (verified: %v)", authCtx.UserID, authCtx.IsVerified)

	// Require email verification before creating account
	if !authCtx.IsVerified {
		log.Printf("[Register] Email not verified for user: %s", authCtx.UserID)
		http.Error(w, "Email verification required", http.StatusForbidden)
		return
	}

	var req struct {
		Email       string `json:"email"`
		DisplayName string `json:"displayName"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[Register] Failed to decode request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

    log.Printf("[Register] Registration request received (DisplayName provided: %t)", req.DisplayName != "")

	if req.Email == "" {
		log.Printf("[Register] Empty email in request")
		http.Error(w, "Email is required", http.StatusBadRequest)
		return
	}

	// Create user profile in Cosmos DB (Firebase user already exists)
	log.Printf("[Register] Creating user profile in Cosmos DB for user: %s", authCtx.UserID)
	user, err := am.userStore.CreateUser(r.Context(), authCtx.UserID, req.Email, req.DisplayName, authCtx.PhotoURL)
	if err != nil {
		log.Printf("[Register] Failed to create user in Cosmos DB: %v", err)
		http.Error(w, "Failed to create user profile", http.StatusInternalServerError)
		return
	}

	log.Printf("[Register] Successfully created user profile: %s", user.UserID)

	response := map[string]interface{}{
		"message": "User profile created successfully",
		"user": map[string]interface{}{
			"id":          user.UserID,
			"email":       user.Email,
			"displayName": user.DisplayName,
			"photoURL":    user.PhotoURL,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("[Register] Failed to encode response: %v", err)
	} else {
		log.Printf("[Register] Registration completed successfully for user: %s", authCtx.UserID)
	}
}

func (am *AuthMiddleware) handleProfile(w http.ResponseWriter, r *http.Request) {
	log.Printf("[Profile] Handling profile request: %s", r.Method)

	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		log.Printf("[Profile] No auth context found")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	log.Printf("[Profile] Auth context found for user: %s", authCtx.UserID)

	switch r.Method {
	case http.MethodGet:
		log.Printf("[Profile] Getting user profile for: %s", authCtx.UserID)
		user, err := am.userStore.GetUser(r.Context(), authCtx.UserID)
		if err != nil {
			log.Printf("[Profile] User not found in database: %s, error: %v", authCtx.UserID, err)
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		// Auto-fix: Update email from auth context if it's empty in the database
		needsUpdate := false
		if user.Email == "" && authCtx.Email != "" {
			log.Printf("[Profile] Auto-fixing empty email for user: %s (setting to %s)", authCtx.UserID, authCtx.Email)
			user.Email = authCtx.Email
			needsUpdate = true
		}

		// Auto-fix: Update displayName from auth context if it's empty or "Anonymous User"
		if (user.DisplayName == "" || user.DisplayName == "Anonymous User") && authCtx.DisplayName != "" {
			log.Printf("[Profile] Auto-fixing displayName for user: %s (setting to %s)", authCtx.UserID, authCtx.DisplayName)
			user.DisplayName = authCtx.DisplayName
			needsUpdate = true
		}

		// If we made any fixes, persist them to the database
		if needsUpdate {
			if err := am.userStore.UpdateUser(r.Context(), user); err != nil {
				log.Printf("[Profile] Failed to persist auto-fixes for user %s: %v", authCtx.UserID, err)
			} else {
				log.Printf("[Profile] Successfully persisted auto-fixes for user: %s", authCtx.UserID)
			}
		}

		log.Printf("[Profile] Successfully retrieved user profile: %s", user.UserID)
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(user); err != nil {
			log.Printf("[Profile] Failed to encode user profile: %v", err)
		}

	case http.MethodPut:
		var req struct {
			DisplayName string          `json:"displayName"`
			PhotoURL    string          `json:"photoURL"`
			Preferences UserPreferences `json:"preferences"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Update user in Cosmos DB
		user, err := am.userStore.GetUser(r.Context(), authCtx.UserID)
		if err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		if req.DisplayName != "" {
			user.DisplayName = req.DisplayName
		}
		if req.PhotoURL != "" {
			user.PhotoURL = req.PhotoURL
		}
		user.Preferences = req.Preferences

		if err := am.userStore.UpdateUser(r.Context(), user); err != nil {
			http.Error(w, "Failed to update user", http.StatusInternalServerError)
			return
		}

		// Update Firebase user
		if req.DisplayName != "" || req.PhotoURL != "" {
			var displayName, photoURL *string
			if req.DisplayName != "" {
				displayName = &req.DisplayName
			}
			if req.PhotoURL != "" {
				photoURL = &req.PhotoURL
			}

			go func() {
				if _, err := am.firebaseAuth.UpdateUser(context.Background(), authCtx.UserID, displayName, photoURL); err != nil {
					log.Printf("Failed to update Firebase user: %v", err)
				}
			}()
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (am *AuthMiddleware) handleDeleteAccount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Deactivate user in Cosmos DB
	if err := am.userStore.DeactivateUser(r.Context(), authCtx.UserID); err != nil {
		log.Printf("Failed to deactivate user in Cosmos DB: %v", err)
	}

	// Delete user from Firebase
	if err := am.firebaseAuth.DeleteUser(r.Context(), authCtx.UserID); err != nil {
		log.Printf("Failed to delete Firebase user: %v", err)
		http.Error(w, "Failed to delete account", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Account deleted successfully",
	})
}
