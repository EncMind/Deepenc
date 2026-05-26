package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
	"github.com/google/uuid"
)


type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	UserID   string        `json:"userId"`
	ThreadID string        `json:"threadId"`
}

type ModelsResponse struct {
	OpenAI           []string `json:"openai"`
	Anthropic        []string `json:"anthropic"`
	Gemini           []string `json:"gemini"`
	DefaultOpenAI    string   `json:"defaultOpenAI"`
	DefaultAnthropic string   `json:"defaultAnthropic"`
	DefaultGemini    string   `json:"defaultGemini"`
}

type ChosenModelsResponse struct {
	OpenAI    string `json:"openai"`
	Anthropic string `json:"anthropic"`
	Gemini    string `json:"gemini"`
}

type SetChosenModelsRequest struct {
	OpenAI    string `json:"openai"`
	Anthropic string `json:"anthropic"`
	Gemini    string `json:"gemini"`
}

type CreateThreadRequest struct {
	Title string `json:"title"`
}

type SendMessageRequest struct {
	Content string `json:"content"`
	Model   string `json:"model"`
}

type StreamWithContextRequest struct {
	ThreadID string        `json:"threadId"`
	UserID   string        `json:"userId"`
	Message  string        `json:"message"`
	Model    string        `json:"model"`
	Provider string        `json:"provider"`
	Context  ContextConfig `json:"context"`
	FreeTier bool          `json:"freeTier,omitempty"`
}

const persistenceTimeout = 30 * time.Second

var (
	cosmos               *CosmosStore
	multiCosmos          *MultiCosmosStore
	threadManager        *ThreadManager
	multiThreadManager   *MultiThreadManager
	contextBuilder       *ContextBuilder
	vectorStore          *VectorStore
	firebaseAuth         *FirebaseAuth
	userStore            *UserStore
	multiUserStore       *MultiUserStore
	anonymousUserStore   *AnonymousUserStore
	authMiddleware       *AuthMiddleware
	usageService         *UsageService
	stripeService        *StripeService
	encryptionService    *TEEEncryptionService
	contentParserService *ContentParserService
	useMultiContainer    bool
	rateLimiter          *RateLimiter
)

func createEncryptedMessage(id, threadID, userID, role, content, provider, model string, createdAt int64) (*EnhancedMessage, error) {
	var parsedContent *ParsedContent
	var hasCodeBlocks, hasMathBlocks, hasDiagrams bool
	var parsingProvider string
	var parsedAt int64

	// Parse assistant messages to extract metadata (code blocks, math, etc.)
	if role == "assistant" && contentParserService != nil && content != "" {
		parsed, err := contentParserService.ParseContent(content, provider)
		if err != nil {
			log.Printf("⚠️ Content parsing failed for %s: %v", provider, err)
		} else {
			parsedContent = parsed
			parsingProvider = provider
			parsedAt = time.Now().Unix()

			// Extract metadata flags from parsed blocks
			for _, block := range parsed.Blocks {
				switch block.Type {
				case "code":
					hasCodeBlocks = true
				case "math":
					hasMathBlocks = true
				case "diagram":
					hasDiagrams = true
				}
			}
		}
	}

	if encryptionService == nil {
		log.Printf("⚠️ Encryption service not available, saving as plaintext")
		return &EnhancedMessage{
			ID:              id,
			ThreadID:        threadID,
			UserID:          userID,
			Role:            role,
			Content:         content,
			Provider:        provider,
			Model:           model,
			CreatedAt:       createdAt,
			ParsedContent:   parsedContent,
			HasCodeBlocks:   hasCodeBlocks,
			HasMathBlocks:   hasMathBlocks,
			HasDiagrams:     hasDiagrams,
			ParsedAt:        parsedAt,
			ParsingProvider: parsingProvider,
		}, nil
	}

	encrypted, err := encryptionService.EncryptMessage(content)
	if err != nil {
		log.Printf("❌ Failed to encrypt message content: %v", err)
		return &EnhancedMessage{
			ID:              id,
			ThreadID:        threadID,
			UserID:          userID,
			Role:            role,
			Content:         content,
			Provider:        provider,
			Model:           model,
			CreatedAt:       createdAt,
			ParsedContent:   parsedContent,
			HasCodeBlocks:   hasCodeBlocks,
			HasMathBlocks:   hasMathBlocks,
			HasDiagrams:     hasDiagrams,
			ParsedAt:        parsedAt,
			ParsingProvider: parsingProvider,
		}, nil
	}

	return &EnhancedMessage{
		ID:        id,
		ThreadID:  threadID,
		UserID:    userID,
		Role:      role,
		Content:   "",
		Provider:  provider,
		Model:     model,
		CreatedAt: createdAt,

		EncryptedContent: encrypted.EncryptedContent,
		EncryptedAESKey:  encrypted.EncryptedAESKey,
		IV:               encrypted.IV,
		AuthTag:          encrypted.AuthTag,
		Algorithm:        encrypted.Algorithm,

		ParsedContent:   parsedContent,
		HasCodeBlocks:   hasCodeBlocks,
		HasMathBlocks:   hasMathBlocks,
		HasDiagrams:     hasDiagrams,
		ParsedAt:        parsedAt,
		ParsingProvider: parsingProvider,
	}, nil
}

func decryptMessageContent(msg *EnhancedMessage) error {
	if encryptionService == nil {
		return nil
	}

	if msg.EncryptedContent == "" {
		return nil
	}

	encryptedMsg := &EncryptedMessage{
		EncryptedContent: msg.EncryptedContent,
		EncryptedAESKey:  msg.EncryptedAESKey,
		IV:               msg.IV,
		AuthTag:          msg.AuthTag,
		Algorithm:        msg.Algorithm,
	}

	decryptedContent, err := encryptionService.DecryptMessage(encryptedMsg)
	if err != nil {
		log.Printf("❌ Failed to decrypt message %s: %v", msg.ID, err)
		return fmt.Errorf("decryption failed: %w", err)
	}

	msg.Content = decryptedContent
	msg.EncryptedContent = ""
	msg.EncryptedAESKey = ""
	msg.IV = ""
	msg.AuthTag = ""
	msg.Algorithm = ""

	return nil
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Println("🚀 Starting Deepenc AI - Your private, secure and universal AI")

	configService := NewConfigService()
	log.Printf("✅ Configuration service initialized")

	rateLimiter = initRateLimiterFromEnv()

	useMultiContainer = strings.EqualFold(os.Getenv("USE_MULTI_CONTAINER"), "true")

	if useMultiContainer {
		log.Println("   Backend with Multi-Container Cosmos DB + Thread + RAG...")
	} else {
		log.Println("   Backend with Single-Container Cosmos DB + Thread + RAG...")
	}

	rootCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var err error

	if useMultiContainer {
		multiCosmos, err = NewMultiCosmosStore(rootCtx)
		if err != nil {
			log.Printf("❌ Multi-Container Cosmos DB initialization failed: %v", err)
			log.Printf("🔄 Falling back to single-container mode...")
			useMultiContainer = false
		} else {
			if err := multiCosmos.TestConnection(rootCtx); err != nil {
				log.Printf("⚠️ Multi-Container Cosmos test failed: %v", err)
				log.Printf("🔄 Falling back to single-container mode...")
				useMultiContainer = false
				multiCosmos = nil
			} else {
				log.Printf("✅ Multi-Container Cosmos test passed")
			}

			if useMultiContainer {
				if err := multiCosmos.Ping(rootCtx); err != nil {
					log.Printf("⚠️ Multi-Container Cosmos ping failed: %v", err)
				} else {
					log.Printf("✅ Multi-Container Cosmos ping ok")
				}
			}
		}
	}

	if !useMultiContainer {
		cosmos, err = NewCosmosStore(rootCtx)
		if err != nil {
			log.Fatalf("❌ Cosmos DB initialization failed: %v", err)
		}

		if err := cosmos.Ping(rootCtx); err != nil {
			log.Printf("⚠️ Cosmos ping failed: %v", err)
		} else {
			log.Printf("✅ Cosmos ping ok")
		}
	}

	vectorStore, err = NewVectorStore()
	if err != nil {
		log.Printf("⚠️ Vector store initialization failed: %v", err)
		log.Printf("   RAG features will be disabled")
		vectorStore = nil
	}

	firebaseAuth, err = NewFirebaseAuth(rootCtx)
	if err != nil {
		log.Printf("⚠️ Firebase Auth initialization failed: %v", err)
		log.Printf("   Authentication features will be disabled")
		firebaseAuth = nil
	}

	if useMultiContainer {
		multiUserStore, err = NewMultiUserStore(multiCosmos)
		if err != nil {
			log.Fatalf("❌ Multi User Store initialization failed: %v", err)
		}
		log.Println("✅ Multi User Store initialized")

		anonymousUserStore, err = NewAnonymousUserStore(multiCosmos)
		if err != nil {
			log.Fatalf("❌ Anonymous User Store initialization failed: %v", err)
		}
		log.Println("✅ Anonymous User Store initialized")
	} else {
		userStore, err = NewUserStore(cosmos)
		if err != nil {
			log.Fatalf("❌ User Store initialization failed: %v", err)
		}
		log.Println("✅ Legacy User Store initialized")
	}

	if useMultiContainer {
		usageService = NewUsageService(&UserStoreWrapper{multiUserStore})
	} else {
		usageService = NewUsageService(userStore)
	}
	log.Println("✅ Usage service initialized")

	// Initialize webhook event store for Stripe idempotency
	// Store webhook events in the users container using fixed partition "webhooks"
	var webhookContainer *azcosmos.ContainerClient
	var webhookPartitionField string
	if useMultiContainer {
		webhookContainer = multiCosmos.GetUsersContainer()
		webhookPartitionField = multiCosmos.GetUsersPartitionKey()
	} else {
		webhookContainer = cosmos.container
		webhookPartitionField = cosmos.partitionKey
	}

	webhookEventStore, err := NewWebhookEventStore(webhookContainer, webhookPartitionField)
	if err != nil {
		log.Fatalf("❌ Failed to initialize webhook event store: %v", err)
	}
	log.Println("✅ Webhook event store initialized")

	if useMultiContainer {
		stripeService = NewStripeService(&UserStoreWrapper{multiUserStore}, webhookEventStore)
	} else {
		stripeService = NewStripeService(userStore, webhookEventStore)
	}
	log.Println("✅ Stripe service initialized")

	if firebaseAuth != nil {
		if useMultiContainer {
			authMiddleware = NewAuthMiddleware(firebaseAuth, &UserStoreWrapper{multiUserStore})
		} else {
			authMiddleware = NewAuthMiddleware(firebaseAuth, userStore)
		}
		log.Println("✅ Authentication middleware initialized")
	}

	tokenCalculator := NewTokenCalculator()
	log.Println("✅ Token Calculator initialized")

	// Initialize encryption service before thread managers (they depend on it)
	encryptionService, err = NewTEEEncryptionService()
	if err != nil {
		log.Fatalf("❌ TEE Encryption Service initialization failed: %v", err)
	}
	log.Println("🔐 TEE Encryption Service initialized")

	if useMultiContainer {
		multiThreadManager, err = NewMultiThreadManager(multiCosmos, vectorStore, encryptionService)
		if err != nil {
			log.Fatalf("❌ Multi Thread Manager init failed: %v", err)
		}
		log.Println("✅ Multi Thread Manager initialized")
		contextBuilder = NewContextBuilder(&ThreadManagerWrapper{multiThreadManager}, vectorStore, tokenCalculator)
	} else {
		threadManager, err = NewThreadManager(cosmos, vectorStore, encryptionService)
		if err != nil {
			log.Fatalf("❌ Thread Manager init failed: %v", err)
		}
		log.Println("✅ Legacy Thread Manager initialized")
		contextBuilder = NewContextBuilder(threadManager, vectorStore, tokenCalculator)
	}

	initContentParserService()

	log.Println("✅ All components initialized")

	// Routes
	mux := http.NewServeMux()

	mux.HandleFunc("/api/healthz", handleHealthz)
	mux.HandleFunc("/api/health", handleHealthCheck)

	// Configuration endpoints (public for security)
	mux.HandleFunc("/api/config/firebase", configService.handleFirebaseConfig)
	mux.HandleFunc("/api/config/stripe", configService.handleStripeConfig)

	if authMiddleware != nil {
		mux.Handle("/api/models", authMiddleware.OptionalAuth(http.HandlerFunc(handleModels)))
	} else {
		mux.HandleFunc("/api/models", handleModels)
	}

	if authMiddleware != nil {
		mux.Handle("/api/chosen-models", authMiddleware.RequireAuth(http.HandlerFunc(handleChosenModels)))
	}

	if anonymousUserStore != nil {
		mux.HandleFunc("/api/anonymous-session", handleAnonymousSession)
	}

	if authMiddleware != nil {
		mux.Handle("/api/auth/register", authMiddleware.RequireAuth(http.HandlerFunc(authMiddleware.handleRegister)))
		mux.Handle("/api/auth/profile", authMiddleware.RequireAuth(http.HandlerFunc(authMiddleware.handleProfile)))
		mux.Handle("/api/auth/delete", authMiddleware.RequireAuth(http.HandlerFunc(authMiddleware.handleDeleteAccount)))
	}

	if authMiddleware != nil {
		mux.Handle("/api/threads", authMiddleware.OptionalAuth(http.HandlerFunc(handleThreads)))
		mux.Handle("/api/threads/", authMiddleware.OptionalAuth(http.HandlerFunc(handleThreadOperations)))
	} else {
		mux.HandleFunc("/api/threads", handleThreads)
		mux.HandleFunc("/api/threads/", handleThreadOperations)
	}
	mux.HandleFunc("/api/parse/content", handleParseContent)
	mux.HandleFunc("/api/parse/openai", handleParseOpenAI)
	mux.HandleFunc("/api/parse/claude", handleParseClaude)
	mux.HandleFunc("/api/parse/gemini", handleParseGemini)

	if authMiddleware != nil {
		mux.Handle("/api/stream/contextual", authMiddleware.OptionalAuth(http.HandlerFunc(handleStreamWithContext)))
	} else {
		mux.HandleFunc("/api/stream/contextual", handleStreamWithContext)
	}

	// Usage and subscription management (require auth)
	if authMiddleware != nil {
		mux.Handle("/api/usage/stats", authMiddleware.RequireAuth(http.HandlerFunc(handleUsageStats)))
		mux.Handle("/api/usage/analytics", authMiddleware.RequireAuth(http.HandlerFunc(handleUsageAnalytics)))
		mux.Handle("/api/usage/tiers", http.HandlerFunc(handleTierInfo))
		mux.Handle("/api/subscription", authMiddleware.RequireAuth(http.HandlerFunc(handleSubscriptionInfo)))
		mux.Handle("/api/subscription/status", authMiddleware.RequireAuth(http.HandlerFunc(handleSubscriptionStatus)))
		mux.Handle("/api/subscription/checkout", authMiddleware.RequireAuth(http.HandlerFunc(handleCreateCheckout)))
		mux.Handle("/api/subscription/portal", authMiddleware.RequireAuth(http.HandlerFunc(handleBillingPortal)))
		mux.Handle("/api/subscription/cancel", authMiddleware.RequireAuth(http.HandlerFunc(handleCancelSubscription)))
		mux.Handle("/api/subscription/update", authMiddleware.RequireAuth(http.HandlerFunc(handleSubscriptionUpdate)))
		mux.Handle("/api/subscription/renew", authMiddleware.RequireAuth(http.HandlerFunc(handleRenewSubscription)))
		mux.Handle("/api/usage/reset-billing", authMiddleware.RequireAuth(http.HandlerFunc(handleResetBillingPeriod)))
	} else {
		mux.HandleFunc("/api/usage/stats", handleUsageStats)
		mux.HandleFunc("/api/usage/analytics", handleUsageAnalytics)
		mux.HandleFunc("/api/usage/tiers", handleTierInfo)
		mux.HandleFunc("/api/subscription", handleSubscriptionInfo)
		mux.HandleFunc("/api/subscription/status", handleSubscriptionStatus)
		mux.HandleFunc("/api/subscription/checkout", handleCreateCheckout)
		mux.HandleFunc("/api/subscription/portal", handleBillingPortal)
		mux.HandleFunc("/api/subscription/cancel", handleCancelSubscription)
		mux.HandleFunc("/api/subscription/update", handleSubscriptionUpdate)
		mux.HandleFunc("/api/subscription/renew", handleRenewSubscription)
		mux.HandleFunc("/api/usage/reset-billing", handleResetBillingPeriod)
	}

	mux.HandleFunc("/api/webhooks/stripe", handleStripeWebhook)

	handler := corsMiddlewareWithConfig(configService, rateLimitMiddleware(requestSizeLimitMiddleware(requestLoggingMiddleware(mux))))

	addr := ":" + strings.TrimPrefix(envOr("PORT", "8080"), ":")

	log.Printf("🚀 Starting server with PID: %d", os.Getpid())
	testListener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Printf("❌ CRITICAL: Port %s is already in use", addr)
		log.Printf("❌ Another server instance may be running")
		log.Printf("❌ Error: %v", err)
		os.Exit(1)
	}
	testListener.Close() // Close the test listener immediately

	log.Printf("✅ Port %s is available", addr)
	log.Printf("✅ All components initialized successfully")
	log.Printf("✅ Server ready on %s (PID: %d)", addr, os.Getpid())
	log.Printf("📊 Health: http://localhost%s/api/health", addr)

	log.Printf("🎯 Starting HTTP server...")
	log.Fatal(http.ListenAndServe(addr, handler))
}
func getActiveThreadManager() ThreadManagerInterface {
	if useMultiContainer {
		return &ThreadManagerWrapper{multiThreadManager}
	}
	return threadManager
}
func requestLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := uuid.NewString()[:8] // Short ID for logging
		start := time.Now()

		ctx := context.WithValue(r.Context(), "requestID", requestID)
		r = r.WithContext(ctx)

		log.Printf("[REQ:%s] %s %s from %s", requestID, r.Method, r.URL.Path, r.RemoteAddr)
		wrappedWriter := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrappedWriter, r)

		// Log request completion
		duration := time.Since(start)
		log.Printf("[REQ:%s] Completed %d in %v", requestID, wrappedWriter.statusCode, duration)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Implement http.Flusher interface to support streaming
func (rw *responseWriter) Flush() {
	if flusher, ok := rw.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}
func corsMiddlewareWithConfig(configService *ConfigService, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Get CORS configuration from config service (includes production security)
		corsConfig := configService.GetCORSConfig()

		origin := r.Header.Get("Origin")
		if origin != "" {
			for _, allowedOrigin := range corsConfig.AllowedOrigins {
				if strings.TrimSpace(allowedOrigin) == origin {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					break
				}
			}
		}

		w.Header().Set("Access-Control-Allow-Headers", strings.Join(corsConfig.AllowedHeaders, ", "))
		w.Header().Set("Access-Control-Allow-Methods", strings.Join(corsConfig.AllowedMethods, ", "))
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Max-Age", strconv.Itoa(corsConfig.MaxAge))

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func rateLimitMiddleware(next http.Handler) http.Handler {
	if rateLimiter == nil {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !shouldRateLimitRequest(r) {
			next.ServeHTTP(w, r)
			return
		}

		clientID := getClientIdentifier(r)
		if rateLimiter.Allow(clientID) {
			next.ServeHTTP(w, r)
			return
		}

		log.Printf("[RateLimit] 429 Too Many Requests for %s %s from %s", r.Method, r.URL.Path, clientID)
		http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
	})
}

func shouldRateLimitRequest(r *http.Request) bool {
	path := r.URL.Path

	if strings.HasPrefix(path, "/api/anonymous-session") {
		return true
	}

	if path == "/api/stream/contextual" && strings.TrimSpace(r.Header.Get("Authorization")) == "" {
		return true
	}

	return false
}

func getClientIdentifier(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		for _, part := range parts {
			ip := strings.TrimSpace(part)
			if ip != "" {
				return ip
			}
		}
	}

	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err != nil || host == "" {
		return r.RemoteAddr
	}

	return host
}

func initRateLimiterFromEnv() *RateLimiter {
	requestsPerMinute := parseEnvInt("RATE_LIMIT_ANON_REQUESTS_PER_MINUTE", 60)
	burst := parseEnvInt("RATE_LIMIT_ANON_BURST", 5)
	ttlMinutes := parseEnvInt("RATE_LIMIT_ANON_TTL_MINUTES", 15)

	if requestsPerMinute <= 0 {
		log.Println("[RateLimit] Disabled (RATE_LIMIT_ANON_REQUESTS_PER_MINUTE <= 0)")
		return nil
	}

	ttl := time.Duration(ttlMinutes) * time.Minute
	log.Printf("[RateLimit] Enabled for anonymous endpoints at %d req/min (burst=%d, ttl=%v)", requestsPerMinute, burst, ttl)

	return NewRateLimiter(requestsPerMinute, burst, ttl)
}

func parseEnvInt(key string, defaultValue int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		log.Printf("[RateLimit] Invalid %s value '%s', using default %d", key, value, defaultValue)
		return defaultValue
	}

	return parsed
}

func requestSizeLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set request body size limit based on content type and endpoint
		var maxSize int64 = 1 << 20 // 1MB default

		// Larger limit for webhook endpoints (Stripe webhooks can be larger)
		if strings.HasPrefix(r.URL.Path, "/api/webhooks/") {
			maxSize = 2 << 20 // 2MB for webhooks
		}

		// Smaller limit for most API endpoints
		if strings.HasPrefix(r.URL.Path, "/api/") && r.Method == "POST" {
			maxSize = 512 << 10 // 512KB for API calls
		}

		r.Body = http.MaxBytesReader(w, r.Body, maxSize)
		next.ServeHTTP(w, r)
	})
}

func handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"services":  make(map[string]string),
	}
	services := health["services"].(map[string]string)

	if cosmos != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := cosmos.Ping(ctx); err != nil {
			services["cosmos"] = "unhealthy"
			health["status"] = "degraded"
		} else {
			services["cosmos"] = "healthy"
		}
	}

	if vectorStore != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := vectorStore.HealthCheck(ctx); err != nil {
			services["vectorStore"] = "unhealthy"
			health["status"] = "degraded"
		} else {
			services["vectorStore"] = "healthy"
		}
	} else {
		services["vectorStore"] = "not configured"
	}

	if firebaseAuth != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := firebaseAuth.HealthCheck(ctx); err != nil {
			services["firebaseAuth"] = "unhealthy"
			health["status"] = "degraded"
		} else {
			services["firebaseAuth"] = "healthy"
		}
	} else {
		services["firebaseAuth"] = "not configured"
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(health)
}

func handleModels(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get user tier (default to free for unauthenticated users)
	userTier := "free"
	userID := GetUserIDFromContext(r.Context())
	if userID != "" && usageService != nil {
		if stats, err := usageService.GetUserUsageStats(r.Context(), userID); err == nil {
			userTier = stats.CurrentTier
		}
	}

	// Get tier-specific allowed models
	var allowedModels map[string][]string
	if usageService != nil {
		tokenCalc := NewTokenCalculator()
		allowedModels = tokenCalc.GetAllowedModelsForTier(userTier)
	} else {
		// Fallback to free tier models if service not available
		allowedModels = map[string][]string{
			"openai":    {"gpt-4o-mini"},
			"anthropic": {"claude-3-haiku"},
			"gemini":    {"gemini-2.0-flash-lite"},
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Fetch available models from APIs and filter against tier-allowed models
	openai := getAvailableOpenAIModels(ctx, allowedModels["openai"])
	anthropic := getAvailableAnthropicModels(ctx, allowedModels["anthropic"])
	gemini := getAvailableGeminiModels(ctx, allowedModels["gemini"])

	// Use tier-appropriate defaults
	var defaultOpenAI, defaultAnthropic, defaultGemini string
	if userTier == "free" {
		defaultOpenAI = getCheapestModel(openai, []string{"gpt-4o-mini"})
		defaultAnthropic = getCheapestModel(anthropic, []string{"claude-3-haiku"})
		defaultGemini = getCheapestModel(gemini, []string{"gemini-2.0-flash-lite"})
	} else {
		// For paid tiers, prefer better models but fall back to cheaper ones
		defaultOpenAI = getCheapestModel(openai, []string{"gpt-5.1", "gpt-5-mini", "gpt-4o"})
		defaultAnthropic = getCheapestModel(anthropic, []string{"claude-opus-4-5", "claude-sonnet-4-5", "claude-3-haiku"})
		defaultGemini = getCheapestModel(gemini, []string{"gemini-3-pro-preview", "gemini-2.5-flash", "gemini-2-5-flash-lite"})
	}

	resp := ModelsResponse{
		OpenAI:           openai,
		Anthropic:        anthropic,
		Gemini:           gemini,
		DefaultOpenAI:    defaultOpenAI,
		DefaultAnthropic: defaultAnthropic,
		DefaultGemini:    defaultGemini,
	}

	_ = json.NewEncoder(w).Encode(resp)
}

func getAvailableOpenAIModels(ctx context.Context, supported []string) []string {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Printf("OPENAI_API_KEY not set, using supported models as fallback")
		return supported
	}

	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.openai.com/v1/models", nil)
	if err != nil {
		log.Printf("Failed to create OpenAI models request: %v", err)
		return supported
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Failed to fetch OpenAI models: %v", err)
		return supported
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Printf("OpenAI models API returned status %d", resp.StatusCode)
		return supported
	}

	var apiResp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		log.Printf("Failed to decode OpenAI models response: %v", err)
		return supported
	}

	// Create a set of available models
	available := make(map[string]bool)
	for _, model := range apiResp.Data {
		available[model.ID] = true
	}

	// Filter our supported models to only include available ones
	var result []string
	for _, model := range supported {
		if available[model] {
			result = append(result, model)
		}
	}

	if len(result) == 0 {
		log.Printf("No supported OpenAI models found, using fallback")
		return supported
	}

	return result
}

func getAvailableAnthropicModels(ctx context.Context, supported []string) []string {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		log.Printf("ANTHROPIC_API_KEY not set, using supported models as fallback")
		return supported
	}

	// Anthropic doesn't have a public models API, so we'll just return our supported list
	// but we could test each model with a small request to see if it's available
	return supported
}

func getAvailableGeminiModels(ctx context.Context, supported []string) []string {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Printf("GEMINI_API_KEY not set, using supported models as fallback")
		return supported
	}

	// Test each model by making a small request to see if it's available
	var available []string
	for _, model := range supported {
		url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s?key=%s", model, apiKey)
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			continue
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			continue
		}
		resp.Body.Close()

		if resp.StatusCode == 200 {
			available = append(available, model)
		}
	}

	if len(available) == 0 {
		log.Printf("No available Gemini models found, using fallback")
		return supported
	}

	return available
}

func handleChosenModels(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID := GetUserIDFromContext(r.Context())

	switch r.Method {
	case http.MethodGet:
		// Get user's chosen models
		var userStoreToUse UserStoreInterface
		if useMultiContainer {
			userStoreToUse = &UserStoreWrapper{multiUserStore}
		} else {
			userStoreToUse = userStore
		}

		user, err := userStoreToUse.GetUser(r.Context(), userID)
		if err != nil {
			log.Printf("[ChosenModels] Failed to get user %s: %v", userID, err)
			http.Error(w, "Failed to get user", http.StatusInternalServerError)
			return
		}

		if user == nil {
			log.Printf("[ChosenModels] User %s not found", userID)
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		// Get available models to use as defaults
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		supportedOpenAI := []string{
			"gpt-5.1", "gpt-5-mini", "gpt-4o", "gpt-4o-mini",
		}
		supportedAnthropic := []string{
			"claude-opus-4-5", "claude-sonnet-4-5", "claude-3-haiku",
		}
		supportedGemini := []string{
			"gemini-3-pro-preview", "gemini-2.5-flash", "gemini-2.0-flash-lite",
		}

		openai := getAvailableOpenAIModels(ctx, supportedOpenAI)
		anthropic := getAvailableAnthropicModels(ctx, supportedAnthropic)
		gemini := getAvailableGeminiModels(ctx, supportedGemini)

		// Use user preferences if set, otherwise use cheapest defaults
		resp := ChosenModelsResponse{
			OpenAI:    getChosenOrDefault(user.Preferences.DefaultModels.OpenAI, openai, getCheapestModel(openai, []string{"gpt-4o-mini"})),
			Anthropic: getChosenOrDefault(user.Preferences.DefaultModels.Anthropic, anthropic, getCheapestModel(anthropic, []string{"claude-3-haiku"})),
			Gemini:    getChosenOrDefault(user.Preferences.DefaultModels.Gemini, gemini, getCheapestModel(gemini, []string{"gemini-2.0-flash-lite"})),
		}

		// Chosen models returned
		_ = json.NewEncoder(w).Encode(resp)

	case http.MethodPost:
		// Set user's chosen models
		var req SetChosenModelsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Printf("[ChosenModels] Failed to decode request: %v", err)
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		var userStoreToUse UserStoreInterface
		if useMultiContainer {
			userStoreToUse = &UserStoreWrapper{multiUserStore}
		} else {
			userStoreToUse = userStore
		}

		// Update user preferences
		newPrefs := &UserPreferences{
			DefaultModels: DefaultModels{
				OpenAI:    req.OpenAI,
				Anthropic: req.Anthropic,
				Gemini:    req.Gemini,
			},
		}

		err := userStoreToUse.UpdateUserPreferences(r.Context(), userID, newPrefs)
		if err != nil {
			log.Printf("[ChosenModels] Failed to update user %s preferences: %v", userID, err)
			http.Error(w, "Failed to update chosen models", http.StatusInternalServerError)
			return
		}

		// Chosen models updated
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "success"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func getChosenOrDefault(chosen string, available []string, defaultModel string) string {
	// If user has a chosen model and it's still available, use it
	if chosen != "" && contains(available, chosen) {
		return chosen
	}
	// Otherwise use the default (cheapest)
	return defaultModel
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func getCheapestModel(available []string, cheapestOptions []string) string {
	// Find the first cheapest option that's available
	for _, cheap := range cheapestOptions {
		for _, avail := range available {
			if cheap == avail {
				return cheap
			}
		}
	}
	// If no cheap options found, return the first available or a fallback
	if len(available) > 0 {
		return available[0]
	}
	return cheapestOptions[0] // fallback to first cheap option
}

func handleThreads(w http.ResponseWriter, r *http.Request) {
	identity, ok := requireIdentity(w, r)
	if !ok {
		return
	}
	userID := identity.UserID
	log.Printf("[Threads] %s request for user: %s (kind=%s)", r.Method, userID, identity.Kind)

	switch r.Method {
	case http.MethodPost:
		log.Printf("[Threads] Creating new thread for user: %s", userID)

		var req CreateThreadRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Printf("[Threads] Failed to decode create thread request: %v", err)
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.Title) == "" {
			req.Title = "New Conversation"
			log.Printf("[Threads] ⚠️  Using default title 'New Conversation' - frontend should provide title or let stream endpoint create thread")
		}

		log.Printf("[Threads] Creating thread (title omitted for privacy)")
		// Thread title received; not logging content
		ctx := r.Context()
		thread, err := getActiveThreadManager().CreateThread(ctx, userID, req.Title)
		if err != nil {
			log.Printf("[Threads] Failed to create thread for user %s: %v", userID, err)
			http.Error(w, "failed to create thread: "+err.Error(), http.StatusInternalServerError)
			return
		}

		log.Printf("[Threads] ✅ Successfully created thread: %s for user: %s", thread.ID, userID)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(thread)

	case http.MethodGet:
		log.Printf("[Threads] Listing threads for user: %s", userID)

		// Parse pagination and search parameters
		query := r.URL.Query()
		page := 1
		if p := query.Get("page"); p != "" {
			if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
				page = parsed
			}
		}

		limit := 20
		if l := query.Get("limit"); l != "" {
			if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
				limit = parsed
			}
		}

		search := strings.TrimSpace(query.Get("search"))

		ctx := r.Context()
		offset := (page - 1) * limit

		var threads []*Thread
		var err error

		if search != "" {
			// Thread search invoked
			threads, err = getActiveThreadManager().SearchUserThreads(ctx, userID, search, limit, offset)
		} else {
			log.Printf("[Threads] Listing threads for user %s (page %d, limit %d, offset %d)", userID, page, limit, offset)
			// Get enough threads to handle pagination (get all and slice client-side for MVP)
			allThreads, err := getActiveThreadManager().ListUserThreads(ctx, userID, 1000) // Get up to 1000 threads
			if err != nil {
				threads = []*Thread{}
			} else {
				// Apply offset and limit
				start := min(offset, len(allThreads))
				end := min(start+limit+1, len(allThreads)) // +1 to check if there are more
				threads = allThreads[start:end]
			}
		}

		if err != nil {
			log.Printf("[Threads] Failed to list threads for user %s: %v", userID, err)
			http.Error(w, "list threads: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Check if there are more threads
		hasMore := len(threads) > limit
		if hasMore {
			threads = threads[:limit] // Remove the extra thread
		}

		log.Printf("[Threads] ✅ Found %d threads for user: %s", len(threads), userID)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"threads": threads,
			"count":   len(threads),
			"page":    page,
			"limit":   limit,
			"hasMore": hasMore,
		})

	default:
		log.Printf("[Threads] Invalid method %s for user: %s", r.Method, userID)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleThreadOperations(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/threads/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
		log.Printf("[ThreadOps] Invalid path: %s", r.URL.Path)
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	threadID := parts[0]
	identity, ok := requireIdentity(w, r)
	if !ok {
		return
	}
	userID := identity.UserID

	log.Printf("[ThreadOps] %s operation on thread %s for user %s (kind=%s)", r.Method, threadID, userID, identity.Kind)

	if len(parts) == 1 {
		// Get single thread
		log.Printf("[ThreadOps] Getting thread details for: %s", threadID)

		if r.Method != http.MethodGet {
			log.Printf("[ThreadOps] Invalid method %s for getting thread %s", r.Method, threadID)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		ctx := r.Context()
		thread, err := getActiveThreadManager().GetThread(ctx, threadID, userID)
		if err != nil {
			log.Printf("[ThreadOps] Failed to get thread %s for user %s: %v", threadID, userID, err)
			http.Error(w, "Thread not found", http.StatusNotFound)
			return
		}

		log.Printf("[ThreadOps] ✅ Retrieved thread %s for user %s", threadID, userID)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(thread)
		return
	}

	switch parts[1] {
	case "messages":
		log.Printf("[ThreadOps] Messages operation: %s for thread %s", r.Method, threadID)

		switch r.Method {
		case http.MethodGet:
			log.Printf("💬 [ThreadOps] Getting messages for thread: %s", threadID)

			ctx := r.Context()
			msgs, err := getActiveThreadManager().GetThreadMessages(ctx, threadID, userID, 50)
			if err != nil {
				log.Printf("❌ [ThreadOps] Failed to get messages for thread %s: %v", threadID, err)
				http.Error(w, "get messages: "+err.Error(), http.StatusInternalServerError)
				return
			}

			// Messages are automatically decrypted by the thread manager
			log.Printf("📄 [ThreadOps] Retrieved %d messages for thread: %s", len(msgs), threadID)

			// Avoid logging message contents; keep only high-level stats
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"messages": msgs,
				"count":    len(msgs),
			})

		case http.MethodPost:
			log.Printf("[ThreadOps] Adding message to thread: %s", threadID)

			var req SendMessageRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				log.Printf("[ThreadOps] Failed to decode message request for thread %s: %v", threadID, err)
				http.Error(w, "Invalid request", http.StatusBadRequest)
				return
			}

			msgID := "msg_" + uuid.NewString()
			log.Printf("[ThreadOps] Creating message %s for thread %s (content length: %d)", msgID, threadID, len(req.Content))

			msg, err := createEncryptedMessage(
				msgID,
				threadID,
				userID,
				"user",
				req.Content,
				"", // No provider specified in this context
				req.Model,
				time.Now().UnixMilli(),
			)
			if err != nil {
				log.Printf("[ThreadOps] Failed to create encrypted message: %v", err)
				http.Error(w, "Failed to process message", http.StatusInternalServerError)
				return
			}
			ctx := r.Context()
			if err := getActiveThreadManager().SaveMessage(ctx, userID, msg); err != nil {
				log.Printf("[ThreadOps] Failed to save message %s to thread %s: %v", msgID, threadID, err)
				http.Error(w, "save message: "+err.Error(), http.StatusInternalServerError)
				return
			}

			log.Printf("[ThreadOps] ✅ Successfully saved message %s to thread %s", msgID, threadID)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(msg)

		default:
			log.Printf("[ThreadOps] Invalid method %s for messages in thread %s", r.Method, threadID)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}

	case "context":
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		query := r.URL.Query().Get("query")
		strategy := r.URL.Query().Get("strategy")
		if strategy == "" {
			strategy = "hybrid"
		}
		cfg := &ContextConfig{
			MaxMessages: 20,
			RAGCount:    10,
			RAGEnabled:  true,
			Strategy:    strategy,
		}
		ctx := r.Context()
		msgs, err := contextBuilder.BuildContext(ctx, threadID, userID, query, cfg)
		if err != nil {
			http.Error(w, "build context: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"context": msgs,
			"count":   len(msgs),
		})

	case "rename":
		if r.Method != "PUT" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Title string `json:"title"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		if strings.TrimSpace(req.Title) == "" {
			http.Error(w, "Title cannot be empty", http.StatusBadRequest)
			return
		}

		log.Printf("[ThreadOps] Renaming thread %s (title omitted for privacy)", threadID)

		// Update thread title (we'll need to implement this in the interface)
		ctx := r.Context()
		err := getActiveThreadManager().RenameThread(ctx, threadID, userID, req.Title)
		if err != nil {
			log.Printf("[ThreadOps] Failed to rename thread %s: %v", threadID, err)
			http.Error(w, "Failed to rename thread", http.StatusInternalServerError)
			return
		}

		log.Printf("[ThreadOps] ✅ Thread %s renamed successfully", threadID)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})

	case "delete":
		if r.Method != "DELETE" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		log.Printf("[ThreadOps] Deleting thread %s for user %s", threadID, userID)

		ctx := r.Context()
		err := getActiveThreadManager().DeleteThread(ctx, threadID, userID)
		if err != nil {
			log.Printf("[ThreadOps] Failed to delete thread %s: %v", threadID, err)
			http.Error(w, "Failed to delete thread", http.StatusInternalServerError)
			return
		}

		log.Printf("[ThreadOps] ✅ Thread %s deleted successfully", threadID)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})

	default:
		http.Error(w, "Invalid operation", http.StatusBadRequest)
	}
}

func handleStreamWithContext(w http.ResponseWriter, r *http.Request) {
	log.Printf("[Stream] Starting contextual stream request from %s", r.RemoteAddr)

	var req StreamWithContextRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[Stream] Failed to decode request body: %v", err)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	log.Printf("[Stream] Request details - Provider: %s, Model: %s, ThreadID: %s, UserID: %s, Message length: %d",
		req.Provider, req.Model, req.ThreadID, req.UserID, len(req.Message))

	if req.UserID == "" {
		if req.FreeTier {
			log.Printf("[Stream] ⚠️ Free tier request missing UserID - this should not happen")
			http.Error(w, "Free tier requests must include anonymous user ID", http.StatusBadRequest)
			return
		} else {
			req.UserID = GetUserIDFromContext(r.Context())
			log.Printf("[Stream] Extracted UserID from context: %s", req.UserID)
		}
	} else {
		log.Printf("[Stream] Using provided UserID: %s (FreeTier: %t)", req.UserID, req.FreeTier)
	}

	// Check if we need to create a new thread
	threadCreated := false
	if strings.TrimSpace(req.ThreadID) == "" {
		log.Printf("[Stream] No ThreadID provided, creating new thread for user: %s, provider: %s", req.UserID, req.Provider)
		// Avoid logging full user message

		// Generate thread title from user message and provider
		baseTitle := generateThreadTitle(req.Message)

		// Create provider-specific thread title to ensure separate conversations
		var threadTitle string
		switch req.Provider {
		case "openai":
			threadTitle = baseTitle + " (ChatGPT)"
		case "anthropic":
			threadTitle = baseTitle + " (Claude)"
		case "gemini":
			threadTitle = baseTitle + " (Gemini)"
		default:
			threadTitle = baseTitle + " (" + req.Provider + ")"
		}

		thread, err := getActiveThreadManager().CreateThread(r.Context(), req.UserID, threadTitle)
		if err != nil {
			log.Printf("[Stream] Failed to create new thread: %v", err)
			http.Error(w, "failed to create thread", http.StatusInternalServerError)
			return
		}
		req.ThreadID = thread.ID
		threadCreated = true
		log.Printf("[Stream] Created new thread for %s: %s", req.Provider, req.ThreadID)
	}

	// Always save user message for new threads to ensure each provider thread gets the complete conversation
	if threadCreated {
		// Persist user turn
		userMsgID := "msg_" + uuid.NewString()
		log.Printf("[Stream] Creating user message for new thread: %s", userMsgID)

		userMsg, err := createEncryptedMessage(
			userMsgID,
			req.ThreadID,
			req.UserID,
			"user",
			req.Message,
			req.Provider,
			req.Model,
			time.Now().UnixMilli(),
		)
		if err != nil {
			log.Printf("[Stream] Failed to create encrypted user message: %v", err)
			http.Error(w, "Failed to process message", http.StatusInternalServerError)
			return
		}

		log.Printf("[Stream] Saving user message to newly created thread: %s", req.ThreadID)
		if err := getActiveThreadManager().SaveMessage(r.Context(), req.UserID, userMsg); err != nil {
			log.Printf("[Stream] Failed to save user message: %v", err)
		} else {
			log.Printf("[Stream] Successfully saved user message to new thread: %s", userMsgID)
		}
	}

	// Build context
	log.Printf("[Stream] Building context for thread: %s", req.ThreadID)
	ctxMsgs, err := contextBuilder.BuildContext(r.Context(), req.ThreadID, req.UserID, req.Message, &req.Context)
	if err != nil {
		log.Printf("[Stream] Failed to build context: %v", err)
		ctxMsgs = nil
	} else {
		log.Printf("[Stream] Built context with %d messages", len(ctxMsgs))
	}

	allMsgs := contextBuilder.FormatContextForLLM(ctxMsgs, req.Message)
	log.Printf("[Stream] Provider: %s, Thread: %s, Context: %d messages", req.Provider, req.ThreadID, len(allMsgs))

	// Usage enforcement - validate request before API call
	if usageService != nil && !req.FreeTier {
		// Estimate token usage from input messages
		totalInputText := req.Message
		for _, msg := range ctxMsgs {
			totalInputText += " " + msg.Content
		}
		estimatedTokens := NewTokenCalculator().EstimateTokenCount(totalInputText)
		log.Printf("[Stream] Estimated input tokens: %d", estimatedTokens)

		validation, err := usageService.ValidateRequest(r.Context(), req.UserID, req.Model, estimatedTokens)
		if err != nil {
			log.Printf("[Stream] Usage validation error: %v", err)
			http.Error(w, "Usage validation failed", http.StatusInternalServerError)
			return
		}

		if !validation.IsAllowed {
			log.Printf("[Stream] Request blocked: %s", validation.Reason)
			http.Error(w, validation.Reason, http.StatusForbidden)
			return
		}

		log.Printf("[Stream] Usage validation passed - Provider: %s, ConversionRate: %.2f", validation.Provider, validation.ConversionRate)
	} else if req.FreeTier {
		log.Printf("[Stream] Free tier request - bypassing usage validation for model: %s", req.Model)
	}

	provider := strings.ToLower(req.Provider)
	log.Printf("[Stream] Routing to provider: %s", provider)

	switch provider {
	case "openai":
		log.Printf("[Stream] Calling OpenAI with model: %s", req.Model)
		streamOpenAIWithContext(w, r, req.ThreadID, req.UserID, req.Model, allMsgs)
	case "anthropic":
		log.Printf("[Stream] Calling Anthropic with model: %s", req.Model)
		streamAnthropicWithContext(w, r, req.ThreadID, req.UserID, req.Model, allMsgs)
	case "gemini":
		log.Printf("[Stream] Calling Gemini with model: %s", req.Model)
		streamGeminiWithContext(w, r, req.ThreadID, req.UserID, req.Model, allMsgs)
	default:
		log.Printf("[Stream] Invalid provider requested: %s", req.Provider)
		http.Error(w, "Invalid provider", http.StatusBadRequest)
	}

	log.Printf("[Stream] Completed stream request for thread: %s", req.ThreadID)
}

func setupStreamHeaders(w http.ResponseWriter) (http.Flusher, error) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("X-Accel-Buffering", "no")
	// Connection: keep-alive is forbidden in HTTP/2 and causes protocol violations
	fl, ok := w.(http.Flusher)
	if !ok {
		return nil, errors.New("streaming not supported")
	}
	return fl, nil
}

func getAnthropicMaxTokens(model string) int {
	switch model {
	case "claude-3-haiku":
		return 4096 // Claude Haiku 3 - max output 4,096 tokens
	case "claude-3-haiku-20240307":
		return 4096 // Claude Haiku 3 (versioned ID) - same limits
	case "claude-3-5-sonnet-20241022", "claude-3-5-sonnet-latest":
		return 8192 // Claude 3.5 Sonnet - max output 8,192 tokens
	case "claude-sonnet-4-5", "claude-sonnet-4-5-20250929":
		return 64000 // Claude Sonnet 4.5 - max output 64,000 tokens
	case "claude-opus-4-5", "claude-opus-4-5-20251101":
		return 32000 // Claude Opus 4.5 - max output 32,000 tokens
	default:
		return 4096 // Conservative default
	}
}

func getOpenAIMaxTokens(model string) int {
	switch {
	case strings.Contains(model, "gpt-5-mini"):
		return 128000 // GPT-5 Mini max output tokens
	case strings.Contains(model, "gpt-5.1"):
		return 128000 // GPT-5.1 max output tokens
	case strings.Contains(model, "gpt-4o"):
		return 16384 // GPT-4o and GPT-4o mini max output tokens
	default:
		return 16384 // Conservative default (GPT-4o level)
	}
}

func getGeminiMaxTokens(model string) int {
	switch {
	case strings.Contains(model, "gemini-3-pro-preview"):
		return 65536 // Gemini 3 Pro Preview max output tokens
	case strings.Contains(model, "gemini-2.5-flash"):
		return 65536 // Gemini 2.5 Flash max output tokens
	case strings.Contains(model, "gemini-2.0-flash-lite"):
		return 8192 // Gemini 2.0 Flash Lite max output tokens
	default:
		return 8192 // Conservative default
	}
}

// Get maximum input tokens for OpenAI models
func getOpenAIMaxInputTokens(model string) int {
	switch {
	case strings.Contains(model, "gpt-5.1") || strings.Contains(model, "gpt-5-mini"):
		return 272000 // GPT-5.1 and GPT-5 Mini max input tokens
	case strings.Contains(model, "gpt-4o"):
		return 111616 // GPT-4o and GPT-4o mini max input tokens
	default:
		return 111616 // Conservative default (GPT-4o level)
	}
}

// Get maximum input tokens for Anthropic models
func getAnthropicMaxInputTokens(model string) int {
	switch model {
	case "claude-3-haiku":
		return 195904 // Claude Haiku max input tokens
	case "claude-3-haiku-20240307":
		return 195904 // Claude Haiku (versioned ID) max input tokens
	case "claude-sonnet-4-5":
		return 136000 // Claude Sonnet 4.5 max input tokens
	case "claude-opus-4-5", "claude-opus-4-5-20251101":
		return 168000 // Claude Opus 4.5 max input tokens
	default:
		return 136000 // Conservative default
	}
}

// Get maximum input tokens for Gemini models
func getGeminiMaxInputTokens(model string) int {
	// All current Gemini models have the same input token limit
	return 1048576
}

func resolveAnthropicAPIModel(model string) (string, bool) {
	switch model {
	case "claude-3-haiku":
		return "claude-3-haiku-20240307", true
	case "claude-sonnet-4-5":
		return "claude-sonnet-4-5-20250929", true
	case "claude-opus-4-5":
		return "claude-opus-4-5-20251101", true
	default:
		return model, false
	}
}

// Estimate token count for messages (rough estimation: ~4 chars per token)
func estimateTokenCount(messages []ChatMessage) int {
	totalChars := 0
	for _, msg := range messages {
		totalChars += len(msg.Content) + len(msg.Role) + 10 // +10 for formatting overhead
	}
	return totalChars / 4 // Rough token estimation
}

// Validate input doesn't exceed model's input token limit
func validateInputTokens(messages []ChatMessage, model, provider string) error {
	estimatedTokens := estimateTokenCount(messages)
	var maxInputTokens int

	switch provider {
	case "openai":
		maxInputTokens = getOpenAIMaxInputTokens(model)
	case "anthropic":
		maxInputTokens = getAnthropicMaxInputTokens(model)
	case "gemini":
		maxInputTokens = getGeminiMaxInputTokens(model)
	default:
		return fmt.Errorf("Unknown provider: %s", provider)
	}

	if estimatedTokens > maxInputTokens {
		return fmt.Errorf("Input too long: estimated %d tokens exceeds maximum %d input tokens for model %s. Please reduce your input length or start a new conversation",
			estimatedTokens, maxInputTokens, model)
	}

	return nil
}

func convertToOpenAIMessages(in []ChatMessage) []map[string]string {
	out := make([]map[string]string, 0, len(in))
	for _, m := range in {
		role := m.Role
		if role != "user" && role != "assistant" && role != "system" {
			role = "user"
		}
		out = append(out, map[string]string{"role": role, "content": m.Content})
	}
	return out
}

// OpenAI
func streamOpenAIWithContext(w http.ResponseWriter, r *http.Request, threadID, userID, model string, messages []ChatMessage) {
	log.Printf("[OpenAI] Starting stream for user %s, thread %s with %d messages", userID, threadID, len(messages))

	// Validate input token count
	if err := validateInputTokens(messages, model, "openai"); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Set thread ID header for frontend tracking BEFORE setupStreamHeaders
	w.Header().Set("X-Thread-ID", threadID)
	fl, err := setupStreamHeaders(w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		http.Error(w, "OPENAI_API_KEY not set", http.StatusInternalServerError)
		return
	}

	// Create context with cancellation for request timeout and client disconnect detection
	timeout := 600 * time.Second
	if strings.Contains(model, "gpt-5.1") || strings.Contains(model, "gpt-5-mini") || strings.Contains(model, "claude-opus") {
		timeout = 3600 * time.Second
	}
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	openaiURL := "https://api.openai.com/v1/chat/completions"

	// Prepare messages with system message for markdown formatting
	openaiMessages := convertToOpenAIMessages(messages)

	// Add system message at the beginning if not already present
	hasSystemMsg := false
	for _, msg := range openaiMessages {
		if msg["role"] == "system" {
			hasSystemMsg = true
			break
		}
	}

	if !hasSystemMsg {
		systemMsg := map[string]string{
			"role":    "system",
			"content": "You are a helpful assistant. Always format code blocks using markdown syntax with triple backticks (```) followed by the language name. For example: ```python for Python code, ```javascript for JavaScript code, etc.",
		}
		openaiMessages = append([]map[string]string{systemMsg}, openaiMessages...)
	}

	body := map[string]any{
		"model":                 model,
		"stream":                true,
		"messages":              openaiMessages,
		"max_completion_tokens": getOpenAIMaxTokens(model),
	}
	bin, _ := json.Marshal(body)

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, openaiURL, bytes.NewReader(bin))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 300 * time.Second} // 5 minute timeout
	resp, err := client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			log.Printf("[OpenAI] Request cancelled or timed out: %v", ctx.Err())
			return // Don't send error response if context was cancelled
		}
		http.Error(w, "openai error: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		slurp, _ := io.ReadAll(resp.Body)
		http.Error(w, "openai error: "+string(slurp), resp.StatusCode)
		return
	}

	var assistantBuf bytes.Buffer
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	streamCompleted := false
	clientDisconnected := false
	chunkCount := 0
	totalTokensReceived := 0

	log.Printf("🚀 [OpenAI] Starting to process streaming response for thread %s", threadID)
	log.Printf("🚀 [OpenAI] Response status: %d", resp.StatusCode)

	appendToken := func(token string) bool {
		if token == "" {
			return false
		}
		chunkCount++
		totalTokensReceived += len(token)
		// Do not log token content; log only counts periodically
		if chunkCount <= 5 || chunkCount%100 == 0 {
			log.Printf("📦 [OpenAI] Chunk #%d: length=%d, total_buffer=%d",
				chunkCount, len(token), assistantBuf.Len())
		}
		assistantBuf.WriteString(token)
		if n, err := w.Write([]byte(token)); err != nil {
			log.Printf("❌ [OpenAI] Client disconnected during write: %v, bytes written: %d", err, n)
			clientDisconnected = true
			return true
		}
		fl.Flush()
		return false
	}

	for sc.Scan() {
		// Check if client disconnected or context cancelled
		select {
		case <-ctx.Done():
			log.Printf("[OpenAI] Context cancelled during streaming: %v", ctx.Err())
			clientDisconnected = true
			goto cleanup
		default:
		}

		line := sc.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" {
			continue
		}
		if data == "[DONE]" {
			log.Printf("[OpenAI] Received [DONE] signal, stream completed")
			streamCompleted = true
			break
		}
		var evt map[string]any
		if err := json.Unmarshal([]byte(data), &evt); err != nil {
			log.Printf("[OpenAI] JSON parse error for chunk (len=%d): %v", len(data), err)
			continue
		}
		choices, _ := evt["choices"].([]any)
		if len(choices) == 0 {
			continue
		}
		first, _ := choices[0].(map[string]any)
		delta, _ := first["delta"].(map[string]any)
		if delta == nil {
			continue
		}
		switch content := delta["content"].(type) {
		case string:
			if appendToken(content) {
				goto cleanup
			}
		case []any:
			for _, entry := range content {
				if part, ok := entry.(map[string]any); ok {
					if text, _ := part["text"].(string); text != "" {
						if appendToken(text) {
							goto cleanup
						}
					}
				}
			}
		case map[string]any:
			if text, _ := content["text"].(string); text != "" {
				if appendToken(text) {
					goto cleanup
				}
			}
		}
	}

	if err := sc.Err(); err != nil {
		log.Printf("[OpenAI] Stream scan error: %v", err)
	}

cleanup:
	if clientDisconnected {
		log.Printf("[OpenAI] Stream terminated due to client disconnect for thread %s", threadID)
	}

	log.Printf("📊 [OpenAI] Stream statistics for thread %s:", threadID)
	log.Printf("📊 [OpenAI]   Total chunks processed: %d", chunkCount)
	log.Printf("📊 [OpenAI]   Total tokens received: %d", totalTokensReceived)
	log.Printf("📊 [OpenAI]   Final buffer length: %d", assistantBuf.Len())
	log.Printf("📊 [OpenAI]   Stream completed: %v", streamCompleted)
	log.Printf("📊 [OpenAI]   Client disconnected: %v", clientDisconnected)

	if streamCompleted {
		log.Printf("✅ [OpenAI] Stream completed successfully for thread %s", threadID)
	} else {
		log.Printf("⚠️ [OpenAI] Stream ended without [DONE] signal for thread %s", threadID)
	}

	if assistantBuf.Len() > 0 {
		rawResponse := assistantBuf.String()

		// Do not log or persist model response content

		msgID := "msg_" + uuid.NewString()
		log.Printf("💾 [OpenAI] Creating message with ID: %s", msgID)
		log.Printf("💾 [OpenAI] Message details: thread=%s, user=%s, provider=openai, model=%s", threadID, userID, model)

		assistantMsg, err := createEncryptedMessage(
			msgID,
			threadID,
			userID,
			"assistant",
			rawResponse,
			"openai",
			model,
			time.Now().UnixMilli(),
		)
		if err != nil {
			log.Printf("❌ [OpenAI] Failed to create encrypted assistant message: %v", err)
		} else {
			log.Printf("🔐 [OpenAI] Encrypted message created successfully, attempting to save...")
			saveCtx, saveCancel := context.WithTimeout(context.Background(), persistenceTimeout)
			if err := getActiveThreadManager().SaveMessage(saveCtx, userID, assistantMsg); err != nil {
				log.Printf("❌ [OpenAI] Failed to save assistant message to database: %v", err)
			} else {
				log.Printf("✅ [OpenAI] Successfully saved assistant message to database with ID: %s", msgID)
			}
			saveCancel()
		}

		// Record complete token usage for this API call (input + output)
		if usageService != nil {
			// Calculate output tokens
			outputTokens := NewTokenCalculator().EstimateTokenCount(assistantBuf.String())

			// Calculate input tokens (all messages sent to OpenAI)
			inputTokens := int64(0)
			tokenCalc := NewTokenCalculator()
			for _, msg := range messages {
				inputTokens += tokenCalc.AccurateTokenCount(msg.Content, model)
			}

			log.Printf("[Stream] Recording OpenAI usage: %d input + %d output = %d total tokens for user %s",
				inputTokens, outputTokens, inputTokens+outputTokens, userID)

			usageCtx, usageCancel := context.WithTimeout(context.Background(), persistenceTimeout)
			if err := usageService.RecordCompleteUsage(usageCtx, userID, "openai", inputTokens, outputTokens); err != nil {
				log.Printf("[Stream] Failed to record OpenAI usage: %v", err)
			}
			usageCancel()
		}
	}
}

// Anthropic
func streamAnthropicWithContext(w http.ResponseWriter, r *http.Request, threadID, userID, model string, messages []ChatMessage) {
	log.Printf("[Anthropic] Starting stream for user %s, thread %s with %d messages", userID, threadID, len(messages))

	// Validate input token count
	if err := validateInputTokens(messages, model, "anthropic"); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Set thread ID header for frontend tracking BEFORE setupStreamHeaders
	w.Header().Set("X-Thread-ID", threadID)
	fl, err := setupStreamHeaders(w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		http.Error(w, "ANTHROPIC_API_KEY not set", http.StatusInternalServerError)
		return
	}

	// Create context with cancellation for request timeout and client disconnect detection
	timeout := 600 * time.Second
	if strings.Contains(model, "gpt-5.1") || strings.Contains(model, "gpt-5-mini") || strings.Contains(model, "claude-opus") {
		timeout = 3600 * time.Second
	}
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	type aContent struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	type aMsg struct {
		Role    string     `json:"role"`
		Content []aContent `json:"content"`
	}
	msgs := make([]aMsg, 0, len(messages))
	for _, m := range messages {
		role := m.Role
		if role != "user" && role != "assistant" {
			role = "user"
		}
		msgs = append(msgs, aMsg{Role: role, Content: []aContent{{Type: "text", Text: m.Content}}})
	}

	apiModel, mapped := resolveAnthropicAPIModel(model)
	if mapped {
		log.Printf("[Anthropic] Resolved model alias %q to API model %q", model, apiModel)
	}

	anthURL := "https://api.anthropic.com/v1/messages"
	body := map[string]any{
		"model":       apiModel,
		"messages":    msgs,
		"max_tokens":  getAnthropicMaxTokens(apiModel),
		"stream":      true,
		"temperature": 0.7,
	}
	bin, _ := json.Marshal(body)

	up, _ := http.NewRequestWithContext(ctx, http.MethodPost, anthURL, bytes.NewReader(bin))
	up.Header.Set("Content-Type", "application/json")
	up.Header.Set("x-api-key", apiKey)
	up.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 300 * time.Second} // 5 minute timeout
	resp, err := client.Do(up)
	if err != nil {
		if ctx.Err() != nil {
			log.Printf("[Anthropic] Request cancelled or timed out: %v", ctx.Err())
			return // Don't send error response if context was cancelled
		}
		http.Error(w, "anthropic error: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		slurp, _ := io.ReadAll(resp.Body)
		http.Error(w, "anthropic error: "+string(slurp), resp.StatusCode)
		return
	}

	var assistantBuf bytes.Buffer
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	streamCompleted := false
	clientDisconnected := false
	chunkCount := 0
	totalTokensReceived := 0

	log.Printf("[Anthropic] Starting to process streaming response")

	appendToken := func(token string) bool {
		if token == "" {
			return false
		}
		chunkCount++
		totalTokensReceived += len(token)
		if chunkCount <= 5 || chunkCount%100 == 0 {
			// Log only metadata; never log model output content
			log.Printf("📦 [Anthropic] Chunk #%d: length=%d", chunkCount, len(token))
		}
		assistantBuf.WriteString(token)
		if n, err := w.Write([]byte(token)); err != nil {
			log.Printf("[Anthropic] Client disconnected during write: %v, bytes written: %d", err, n)
			clientDisconnected = true
			return true
		}
		fl.Flush()
		return false
	}

	for sc.Scan() {
		// Check if client disconnected or context cancelled
		select {
		case <-ctx.Done():
			log.Printf("[Anthropic] Context cancelled during streaming: %v", ctx.Err())
			clientDisconnected = true
			goto cleanup
		default:
		}

		line := sc.Text()
		// Avoid logging raw content; only length metadata is safe to log
		if len(line) > 1000 {
			log.Printf("🔍 [Anthropic] Processing long line: length=%d", len(line))
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" {
			continue
		}
		var evt map[string]any
		if err := json.Unmarshal([]byte(data), &evt); err != nil {
			log.Printf("[Anthropic] JSON parse error for chunk (len=%d): %v", len(data), err)
			continue
		}
		t, _ := evt["type"].(string)

		// Do not log event payloads to avoid exposing model output
		if t != "content_block_delta" && t != "ping" {
			log.Printf("🔍 [Anthropic] Received event type: %s", t)
		}

		if t == "content_block_delta" {
			if delta, ok := evt["delta"].(map[string]any); ok {
				switch content := delta["text"].(type) {
				case string:
					if appendToken(content) {
						goto cleanup
					}
				case []any:
					for _, entry := range content {
						if part, ok := entry.(map[string]any); ok {
							if text, _ := part["text"].(string); text != "" {
								if appendToken(text) {
									goto cleanup
								}
							}
						}
					}
				case map[string]any:
					if text, _ := content["text"].(string); text != "" {
						if appendToken(text) {
							goto cleanup
						}
					}
				}
			}
		}
		if t == "error" {
			log.Printf("[Anthropic] Received error event: %v", evt)

			// Extract error details
			if errMap, ok := evt["error"].(map[string]any); ok {
				errorType, _ := errMap["type"].(string)
				errorMsg, _ := errMap["message"].(string)

				// Append error message to the response
				errorText := fmt.Sprintf("\n\n❌ **Error from Claude API**: %s\n\nThis is a temporary issue with Anthropic's service. Please try again in a moment.", errorMsg)
				if errorType == "overloaded_error" {
					errorText = "\n\n❌ **Claude API is currently overloaded**\n\nAnthropic's servers are experiencing high load. Please try again in a few moments. Your partial response has been saved."
				}
				appendToken(errorText)
			}
			break
		}
		if t == "message_stop" {
			log.Printf("[Anthropic] Received message_stop signal, stream completed")
			streamCompleted = true
			break
		}
	}

	if err := sc.Err(); err != nil {
		log.Printf("❌ [Anthropic] Stream scan error: %v", err)
	} else {
		log.Printf("ℹ️  [Anthropic] Scanner stopped without error (normal EOF or break)")
	}

cleanup:
	if clientDisconnected {
		log.Printf("[Anthropic] Stream terminated due to client disconnect for thread %s", threadID)
	}

	if streamCompleted {
		log.Printf("[Anthropic] Stream completed successfully for thread %s", threadID)
	} else {
		log.Printf("[Anthropic] Stream ended without message_stop signal for thread %s", threadID)
	}

	log.Printf("📊 [Anthropic]   Total chunks processed: %d", chunkCount)
	log.Printf("📊 [Anthropic]   Total tokens received: %d", totalTokensReceived)

	if assistantBuf.Len() > 0 {
		rawResponse := assistantBuf.String()

		// Do not log or persist model response content; keep only length
		log.Printf("🔍 [DEBUG] Raw Claude response length: %d", len(rawResponse))

		assistantMsg, err := createEncryptedMessage(
			"msg_"+uuid.NewString(),
			threadID,
			userID,
			"assistant",
			rawResponse,
			"anthropic",
			model,
			time.Now().UnixMilli(),
		)
		if err != nil {
			log.Printf("❌ Failed to create encrypted assistant message: %v", err)
		} else {
			saveCtx, saveCancel := context.WithTimeout(context.Background(), persistenceTimeout)
			if err := getActiveThreadManager().SaveMessage(saveCtx, userID, assistantMsg); err != nil {
				log.Printf("❌ Failed to save Anthropic assistant message: %v", err)
			} else {
				log.Printf("✅ Successfully saved Anthropic assistant message")
			}
			saveCancel()
		}

		// Record complete token usage for this API call (input + output)
		if usageService != nil {
			// Calculate output tokens
			outputTokens := NewTokenCalculator().EstimateTokenCount(assistantBuf.String())

			// Calculate input tokens (all messages sent to Anthropic)
			inputTokens := int64(0)
			tokenCalc := NewTokenCalculator()
			for _, msg := range messages {
				inputTokens += tokenCalc.AccurateTokenCount(msg.Content, model)
			}

			log.Printf("[Stream] Recording Anthropic usage: %d input + %d output = %d total tokens for user %s",
				inputTokens, outputTokens, inputTokens+outputTokens, userID)

			usageCtx, usageCancel := context.WithTimeout(context.Background(), persistenceTimeout)
			if err := usageService.RecordCompleteUsage(usageCtx, userID, "anthropic", inputTokens, outputTokens); err != nil {
				log.Printf("[Stream] Failed to record Anthropic usage: %v", err)
			}
			usageCancel()
		}
	}
}

// Gemini
func streamGeminiWithContext(w http.ResponseWriter, r *http.Request, threadID, userID, model string, messages []ChatMessage) {
	log.Printf("[Gemini] Starting stream for user %s, thread %s with %d messages", userID, threadID, len(messages))

	// Validate input token count
	if err := validateInputTokens(messages, model, "gemini"); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Set thread ID header for frontend tracking BEFORE setupStreamHeaders
	w.Header().Set("X-Thread-ID", threadID)
	fl, err := setupStreamHeaders(w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		http.Error(w, "GEMINI_API_KEY not set", http.StatusInternalServerError)
		return
	}

	// Create context with cancellation for request timeout and client disconnect detection
	timeout := 300 * time.Second
	if strings.Contains(model, "gemini-3-pro-preview") {
		timeout = 3600 * time.Second
	}
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	// Convert all messages to Gemini format
	var geminiContents []map[string]any
	log.Printf("[Gemini] Converting %d messages to Gemini format", len(messages))
	for i, msg := range messages {
		// Do not log message content
		role := msg.Role
		if role == "assistant" {
			role = "model" // Gemini uses "model" instead of "assistant"
		}
		if role == "user" || role == "model" {
			geminiContents = append(geminiContents, map[string]any{
				"role": role,
				"parts": []map[string]any{
					{"text": msg.Content},
				},
			})
			log.Printf("[Gemini] Added message %d as role=%s", i, role)
		} else {
			log.Printf("[Gemini] Skipped message %d with role=%s", i, msg.Role)
		}
	}
	log.Printf("[Gemini] Final geminiContents has %d messages", len(geminiContents))

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, apiKey)
	payload := map[string]any{
		"contents": geminiContents,
		"generationConfig": map[string]any{
			"maxOutputTokens": getGeminiMaxTokens(model),
		},
	}
	bin, _ := json.Marshal(payload)

	up, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bin))
	up.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 300 * time.Second} // 5 minute timeout
	resp, err := client.Do(up)
	if err != nil {
		if ctx.Err() != nil {
			log.Printf("[Gemini] Request cancelled or timed out: %v", ctx.Err())
			return // Don't send error response if context was cancelled
		}
		http.Error(w, "gemini error: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		slurp, _ := io.ReadAll(resp.Body)
		http.Error(w, "gemini error: "+string(slurp), resp.StatusCode)
		return
	}

	var g struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&g); err != nil {
		log.Printf("[Gemini] JSON decode error: %v", err)
		http.Error(w, "gemini decode error", http.StatusBadGateway)
		return
	}

	var assistantBuf bytes.Buffer
	clientDisconnected := false

	log.Printf("[Gemini] Starting to process response")

	for _, c := range g.Candidates {
		for _, p := range c.Content.Parts {
			if p.Text != "" {
				assistantBuf.WriteString(p.Text)
				if n, err := w.Write([]byte(p.Text)); err != nil {
					log.Printf("[Gemini] Client disconnected during write: %v, bytes written: %d", err, n)
					clientDisconnected = true
					goto cleanup
				}
				fl.Flush()
			}
		}
	}

	log.Printf("[Gemini] Response completed successfully for thread %s", threadID)

cleanup:
	if clientDisconnected {
		log.Printf("[Gemini] Response terminated due to client disconnect for thread %s", threadID)
		return
	}

	if assistantBuf.Len() > 0 {
		rawResponse := assistantBuf.String()

		// Do not log or persist model response content; keep only length
		log.Printf("🔍 [DEBUG] Raw Gemini response length: %d", len(rawResponse))

		assistantMsg, err := createEncryptedMessage(
			"msg_"+uuid.NewString(),
			threadID,
			userID,
			"assistant",
			rawResponse,
			"gemini",
			model,
			time.Now().UnixMilli(),
		)
		if err != nil {
			log.Printf("❌ Failed to create encrypted assistant message: %v", err)
		} else {
			if err := getActiveThreadManager().SaveMessage(context.Background(), userID, assistantMsg); err != nil {
				log.Printf("❌ Failed to save Gemini assistant message: %v", err)
			} else {
				log.Printf("✅ Successfully saved Gemini assistant message")
			}
		}

		// Record complete token usage for this API call (input + output)
		if usageService != nil {
			// Calculate output tokens
			outputTokens := NewTokenCalculator().EstimateTokenCount(assistantBuf.String())

			// Calculate input tokens (all messages sent to Gemini)
			inputTokens := int64(0)
			tokenCalc := NewTokenCalculator()
			for _, msg := range messages {
				inputTokens += tokenCalc.AccurateTokenCount(msg.Content, model)
			}

			log.Printf("[Stream] Recording Gemini usage: %d input + %d output = %d total tokens for user %s",
				inputTokens, outputTokens, inputTokens+outputTokens, userID)

			if err := usageService.RecordCompleteUsage(context.Background(), userID, "gemini", inputTokens, outputTokens); err != nil {
				log.Printf("[Stream] Failed to record Gemini usage: %v", err)
			}
		}
	}
}

// generateThreadTitle creates a meaningful title from the user's message
func generateThreadTitle(message string) string {
	// Clean and limit the message
	message = strings.TrimSpace(message)
	if message == "" {
		return "New Conversation"
	}

	// Remove common question words and punctuation for cleaner titles
	message = strings.ReplaceAll(message, "?", "")
	message = strings.ReplaceAll(message, "!", "")
	message = strings.ReplaceAll(message, ".", "")

	// Split into words and take first meaningful ones
	words := strings.Fields(message)
	if len(words) == 0 {
		return "New Conversation"
	}

	// Skip common starting words
	skipWords := map[string]bool{
		"can": true, "could": true, "would": true, "will": true,
		"how": true, "what": true, "where": true, "when": true, "why": true,
		"please": true, "help": true, "i": true, "me": true,
		"do": true, "does": true, "is": true, "are": true,
	}

	var titleWords []string
	for _, word := range words {
		lowerWord := strings.ToLower(word)
		if !skipWords[lowerWord] || len(titleWords) == 0 {
			titleWords = append(titleWords, word)
		}
		// Limit to 6 words for reasonable title length
		if len(titleWords) >= 6 {
			break
		}
	}

	if len(titleWords) == 0 {
		return "New Conversation"
	}

	title := strings.Join(titleWords, " ")

	// Limit total length to 50 characters
	if len(title) > 50 {
		title = title[:47] + "..."
	}

	return title
}

// handleUsageStats returns current usage statistics for the authenticated user
func handleUsageStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := GetUserIDFromContext(r.Context())
	if userID == "" {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	if usageService == nil {
		http.Error(w, "Usage service not available", http.StatusInternalServerError)
		return
	}

	stats, err := usageService.GetUserUsageStats(r.Context(), userID)
	if err != nil {
		log.Printf("[Usage] Failed to get usage stats for user %s: %v", userID, err)
		http.Error(w, "Failed to get usage stats", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// handleUsageAnalytics returns detailed usage analytics for the authenticated user
func handleUsageAnalytics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := GetUserIDFromContext(r.Context())
	if userID == "" {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	if usageService == nil {
		http.Error(w, "Usage service not available", http.StatusInternalServerError)
		return
	}

	// Get user data for detailed analytics
	var userStoreToUse UserStoreInterface
	if useMultiContainer {
		userStoreToUse = &UserStoreWrapper{multiUserStore}
	} else {
		userStoreToUse = userStore
	}

	user, err := userStoreToUse.GetUser(r.Context(), userID)
	if err != nil {
		log.Printf("[Analytics] Failed to get user %s: %v", userID, err)
		http.Error(w, "Failed to get user data", http.StatusInternalServerError)
		return
	}

	now := time.Now()
	effectiveTier := user.EffectiveTier(now)
	displayTier := effectiveTier
	freeTrialActive := user.FreeTrialActive(now)
	if freeTrialActive {
		displayTier = "free_trial"
	}

	analytics := map[string]interface{}{
		"currentPeriod": map[string]interface{}{
			"start":       user.UsageTracking.CurrentPeriodStart,
			"end":         user.UsageTracking.CurrentPeriodEnd,
			"tokensUsed":  user.UsageTracking.TokensUsed,
			"tokensLimit": user.UsageTracking.TokensLimit,
		},
		"dailyUsage":         user.UsageTracking.DailyUsage,
		"providerUsage":      user.UsageTracking.ProviderUsage,
		"tier":               displayTier,
		"subscriptionStatus": user.Subscription.Status,
		"conversionRates":    getConversionRatesForTier(effectiveTier),
	}

	analytics["freeTrial"] = map[string]interface{}{
		"active":  freeTrialActive,
		"startAt": user.FreeTrial.StartAt,
		"endAt":   user.FreeTrial.EndAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analytics)
}

// getConversionRatesForTier returns the conversion rates for a given tier
func getConversionRatesForTier(tier string) map[string]float64 {
	calc := NewTokenCalculator()
	return map[string]float64{
		"openai":    calc.GetConversionRate("openai", tier),
		"anthropic": calc.GetConversionRate("anthropic", tier),
		"gemini":    calc.GetConversionRate("gemini", tier),
	}
}

// handleSubscriptionInfo returns current subscription information (matching /api/subscription endpoint from design)
func handleSubscriptionInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := GetUserIDFromContext(r.Context())
	if userID == "" {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	if usageService == nil {
		http.Error(w, "Usage service not available", http.StatusInternalServerError)
		return
	}

	// Get user stats which include subscription info
	stats, err := usageService.GetUserUsageStats(r.Context(), userID)
	if err != nil {
		log.Printf("[Subscription] Failed to get user stats for %s: %v", userID, err)
		http.Error(w, "Failed to get subscription info", http.StatusInternalServerError)
		return
	}

	// Get user for subscription details
	var userStoreToUse UserStoreInterface
	if useMultiContainer {
		userStoreToUse = &UserStoreWrapper{multiUserStore}
	} else {
		userStoreToUse = userStore
	}

	user, err := userStoreToUse.GetUser(r.Context(), userID)
	if err != nil {
		log.Printf("[Subscription] Failed to get user %s: %v", userID, err)
		http.Error(w, "Failed to get user data", http.StatusInternalServerError)
		return
	}

	// Build response matching design document format
	displayTier := stats.CurrentTier
	if stats.FreeTrial {
		displayTier = "free_trial"
	}

	response := map[string]interface{}{
		"tier":               displayTier,
		"status":             user.Subscription.Status,
		"currentPeriodStart": user.Subscription.CurrentPeriodStart,
		"currentPeriodEnd":   user.Subscription.CurrentPeriodEnd,
		"usage": map[string]interface{}{
			"tokensUsed":        stats.TokensUsed,
			"tokensLimit":       stats.TokensLimit,
			"usagePercentage":   stats.UsagePercentage,
			"providerBreakdown": stats.ProviderUsage,
			"freeTrial":         stats.FreeTrial,
		},
	}

	response["freeTrial"] = map[string]interface{}{
		"active":  stats.FreeTrial,
		"startAt": user.FreeTrial.StartAt,
		"endAt":   user.FreeTrial.EndAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleTierInfo returns information about all pricing tiers (public endpoint)
func handleTierInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if usageService == nil {
		http.Error(w, "Usage service not available", http.StatusInternalServerError)
		return
	}

	tiers := usageService.GetTierInfo()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tiers": tiers,
	})
}

// handleSubscriptionStatus returns current subscription status for the authenticated user
func handleSubscriptionStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := GetUserIDFromContext(r.Context())
	if userID == "" {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	if stripeService == nil {
		http.Error(w, "Stripe service not available", http.StatusInternalServerError)
		return
	}

	status, err := stripeService.GetUserSubscriptionStatus(r.Context(), userID)
	if err != nil {
		log.Printf("[Subscription] Failed to get subscription status for user %s: %v", userID, err)
		http.Error(w, "Failed to get subscription status", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// handleResetBillingPeriod resets the billing period for the authenticated user (temporary fix)
func handleResetBillingPeriod(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := GetUserIDFromContext(r.Context())
	if userID == "" {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	log.Printf("[BillingReset] Resetting billing period for user: %s", userID)

	var userStoreToUse UserStoreInterface
	if useMultiContainer {
		userStoreToUse = &UserStoreWrapper{multiUserStore}
	} else {
		userStoreToUse = userStore
	}

	err := userStoreToUse.ResetUsageForBillingPeriod(r.Context(), userID)
	if err != nil {
		log.Printf("[BillingReset] Failed to reset billing period for user %s: %v", userID, err)
		http.Error(w, "Failed to reset billing period", http.StatusInternalServerError)
		return
	}

	log.Printf("[BillingReset] Successfully reset billing period for user: %s", userID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Billing period reset successfully",
	})
}

// handleCreateCheckout creates a Stripe checkout session for subscription
func handleCreateCheckout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := GetUserIDFromContext(r.Context())
	if userID == "" {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	var req struct {
		Tier string `json:"tier"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate tier
	validTiers := []string{"plus", "pro", "pro_plus"}
	isValid := false
	for _, tier := range validTiers {
		if req.Tier == tier {
			isValid = true
			break
		}
	}

	if !isValid {
		http.Error(w, "Invalid tier", http.StatusBadRequest)
		return
	}

	if stripeService == nil {
		http.Error(w, "Stripe service not available", http.StatusInternalServerError)
		return
	}

	// Get user details from database
	user, err := multiUserStore.GetUser(r.Context(), userID)
	if err != nil {
		log.Printf("[Subscription] Failed to get user: %v", err)
		http.Error(w, "Failed to get user information", http.StatusInternalServerError)
		return
	}

	// Create or reuse Stripe customer
	var customerID string
	if user.Subscription.StripeCustomerID != "" {
		// Reuse existing customer
		customerID = user.Subscription.StripeCustomerID
		log.Printf("[Subscription] Reusing existing Stripe customer: %s for user: %s", customerID, userID)
	} else {
		// Create new customer
		email := user.Email
		if email == "" {
			email = userID + "@deepenc.temp" // Fallback email
		}
		displayName := user.DisplayName
		if displayName == "" {
			displayName = "DeepEnc User"
		}

		customer, err := stripeService.CreateCustomer(r.Context(), userID, email, displayName)
		if err != nil {
			log.Printf("[Subscription] Failed to create Stripe customer: %v", err)
			http.Error(w, "Failed to create customer", http.StatusInternalServerError)
			return
		}
		customerID = customer.ID
		log.Printf("[Subscription] Created new Stripe customer: %s for user: %s", customerID, userID)
	}

	// Create checkout session
	session, err := stripeService.CreateCheckoutSession(r.Context(), userID, req.Tier, customerID)
	if err != nil {
		log.Printf("[Subscription] Failed to create checkout session: %v", err)
		http.Error(w, "Failed to create checkout session", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"sessionId": session.ID,
		"url":       session.URL,
	})
}

// handleBillingPortal creates a billing portal session for subscription management
func handleBillingPortal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := GetUserIDFromContext(r.Context())
	if userID == "" {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	if stripeService == nil {
		http.Error(w, "Stripe service not available", http.StatusInternalServerError)
		return
	}

	// Get user subscription to find customer ID
	user, err := multiUserStore.GetUser(r.Context(), userID)
	if err != nil {
		log.Printf("[Subscription] Failed to get user %s: %v", userID, err)
		http.Error(w, "Failed to get user information", http.StatusInternalServerError)
		return
	}

	if user.Subscription.StripeCustomerID == "" {
		log.Printf("[Subscription] User %s has no Stripe customer ID", userID)
		http.Error(w, "No billing history available - no active subscription", http.StatusBadRequest)
		return
	}

	customerID := user.Subscription.StripeCustomerID
	log.Printf("[Subscription] Creating billing portal session for customer: %s", customerID)

	session, err := stripeService.CreateBillingPortalSession(r.Context(), customerID)
	if err != nil {
		log.Printf("[Subscription] Failed to create billing portal session: %v", err)
		http.Error(w, "Failed to create billing portal session", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"url": session.URL,
	})
}

// handleRenewSubscription manually renews the user's current subscription
func handleRenewSubscription(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := GetUserIDFromContext(r.Context())
	if userID == "" {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	if stripeService == nil {
		http.Error(w, "Stripe service not available", http.StatusInternalServerError)
		return
	}

	log.Printf("[Subscription] Manual renewal requested by user: %s", userID)

	// Call the renewal service
	err := stripeService.RenewCurrentSubscription(r.Context(), userID)
	if err != nil {
		log.Printf("[Subscription] Failed to renew subscription for user %s: %v", userID, err)
		http.Error(w, fmt.Sprintf("Failed to renew subscription: %v", err), http.StatusBadRequest)
		return
	}

	// Get updated subscription info
	user, err := multiUserStore.GetUser(r.Context(), userID)
	if err != nil {
		log.Printf("[Subscription] Failed to get updated user info: %v", err)
		http.Error(w, "Renewal succeeded but failed to get updated info", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Subscription renewed successfully. Your billing period has been reset and you have full token quota available.",
		"subscription": map[string]interface{}{
			"tier":                 user.Subscription.Tier,
			"status":               user.Subscription.Status,
			"current_period_start": user.Subscription.CurrentPeriodStart,
			"current_period_end":   user.Subscription.CurrentPeriodEnd,
		},
		"usage": map[string]interface{}{
			"tokens_used":  user.UsageTracking.TokensUsed,
			"tokens_limit": user.UsageTracking.TokensLimit,
		},
	})
}

// handleCancelSubscription cancels a user's subscription
func handleCancelSubscription(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := GetUserIDFromContext(r.Context())
	if userID == "" {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	if stripeService == nil {
		http.Error(w, "Stripe service not available", http.StatusInternalServerError)
		return
	}

	log.Printf("[CancelSubscription] Processing cancellation request for user: %s", userID)

	err := stripeService.CancelSubscription(r.Context(), userID)
	if err != nil {
		log.Printf("[CancelSubscription] Failed to cancel subscription for user %s: %v", userID, err)
		http.Error(w, fmt.Sprintf("Failed to cancel subscription: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("[CancelSubscription] ✅ Successfully canceled subscription for user: %s", userID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Subscription has been scheduled for cancellation at the end of the current billing period",
	})
}

// handleStripeWebhook processes Stripe webhook events
func handleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	log.Printf("[Webhook] 🔔 Received webhook request: %s %s", r.Method, r.URL.Path)
	// Avoid verbose header/user-agent logging by default

	if r.Method != http.MethodPost {
		log.Printf("[Webhook] ❌ Invalid method: %s (expected POST)", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	payload, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[Webhook] ❌ Failed to read webhook payload: %v", err)
		http.Error(w, "Failed to read payload", http.StatusBadRequest)
		return
	}

	log.Printf("[Webhook] ✅ Successfully read payload: %d bytes", len(payload))

	signature := r.Header.Get("Stripe-Signature")
	if signature == "" {
		log.Printf("[Webhook] ❌ Missing Stripe signature header")
		http.Error(w, "Missing signature", http.StatusBadRequest)
		return
	}

	signaturePreview := signature
	if len(signaturePreview) > 50 {
		signaturePreview = signaturePreview[:50] + "..."
	}
	log.Printf("[Webhook] ✅ Stripe signature present (preview suppressed for security)")

	if stripeService == nil {
		log.Printf("[Webhook] ❌ Stripe service not available for webhook")
		http.Error(w, "Stripe service not available", http.StatusInternalServerError)
		return
	}

	log.Printf("[Webhook] 🔄 Processing webhook with Stripe service...")
	if err := stripeService.HandleWebhook(payload, signature); err != nil {
		log.Printf("[Webhook] ❌ Failed to process webhook: %v", err)
		http.Error(w, "Webhook processing failed", http.StatusBadRequest)
		return
	}

	log.Printf("[Webhook] ✅ Webhook processed successfully")
	w.WriteHeader(http.StatusOK)
}

// handleSubscriptionUpdate updates a user's subscription (legacy endpoint - now redirects to checkout)
func handleSubscriptionUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := GetUserIDFromContext(r.Context())
	if userID == "" {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	var req struct {
		Tier string `json:"tier"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("[Subscription] Legacy update request for user %s to tier %s - redirecting to checkout", userID, req.Tier)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":  "Use /api/subscription/checkout endpoint for new subscriptions",
		"redirect": "/api/subscription/checkout",
		"tier":     req.Tier,
	})
}

// handleAnonymousSession handles GET and POST for anonymous session data
func handleAnonymousSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get anonymousUserId from query parameter or header
	anonymousUserID := r.URL.Query().Get("anonymousUserId")
	if anonymousUserID == "" {
		anonymousUserID = r.Header.Get("X-Anonymous-User-ID")
	}

	if anonymousUserID == "" {
		http.Error(w, "anonymousUserId is required", http.StatusBadRequest)
		return
	}

	log.Printf("[handleAnonymousSession] %s request for user: %s", r.Method, anonymousUserID)

	switch r.Method {
	case http.MethodGet:
		// Get existing session data
		user, err := anonymousUserStore.GetOrCreateAnonymousUser(ctx, anonymousUserID)
		if err != nil {
			log.Printf("[handleAnonymousSession] Error getting anonymous user: %v", err)
			http.Error(w, "Failed to get session", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user.SessionData)

	case http.MethodPost:
		// Update session data
		var sessionData AnonymousSessionData
		if err := json.NewDecoder(r.Body).Decode(&sessionData); err != nil {
			log.Printf("[handleAnonymousSession] Error decoding session data: %v", err)
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Ensure the anonymousUserId matches
		sessionData.AnonymousUserID = anonymousUserID

		// Get or create user
		user, err := anonymousUserStore.GetOrCreateAnonymousUser(ctx, anonymousUserID)
		if err != nil {
			log.Printf("[handleAnonymousSession] Error getting anonymous user: %v", err)
			http.Error(w, "Failed to get session", http.StatusInternalServerError)
			return
		}

		// Update session data
		user.SessionData = sessionData
		if err := anonymousUserStore.UpdateAnonymousUser(ctx, user); err != nil {
			log.Printf("[handleAnonymousSession] Error updating anonymous user: %v", err)
			http.Error(w, "Failed to update session", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

type ParseRequest struct {
	Content  string `json:"content"`
	Provider string `json:"provider,omitempty"`
}

// handleParseContent handles universal content parsing
func handleParseContent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ParseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Content == "" {
		http.Error(w, "Content is required", http.StatusBadRequest)
		return
	}

	// Use universal strategy if no provider specified
	provider := req.Provider
	if provider == "" {
		provider = "universal"
	}

	parsed, err := contentParserService.ParseContent(req.Content, provider)
	if err != nil {
		log.Printf("❌ Content parsing failed: %v", err)
		http.Error(w, "Parsing failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(parsed)
}

// handleParseOpenAI handles OpenAI-specific content parsing
func handleParseOpenAI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ParseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Content == "" {
		http.Error(w, "Content is required", http.StatusBadRequest)
		return
	}

	parsed, err := contentParserService.ParseContent(req.Content, "openai")
	if err != nil {
		log.Printf("❌ OpenAI content parsing failed: %v", err)
		http.Error(w, "Parsing failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(parsed)
}

// handleParseClaude handles Claude-specific content parsing
func handleParseClaude(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ParseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Content == "" {
		http.Error(w, "Content is required", http.StatusBadRequest)
		return
	}

	parsed, err := contentParserService.ParseContent(req.Content, "claude")
	if err != nil {
		log.Printf("❌ Claude content parsing failed: %v", err)
		http.Error(w, "Parsing failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(parsed)
}

// handleParseGemini handles Gemini-specific content parsing
func handleParseGemini(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ParseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Content == "" {
		http.Error(w, "Content is required", http.StatusBadRequest)
		return
	}

	parsed, err := contentParserService.ParseContent(req.Content, "gemini")
	if err != nil {
		log.Printf("❌ Gemini content parsing failed: %v", err)
		http.Error(w, "Parsing failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(parsed)
}
