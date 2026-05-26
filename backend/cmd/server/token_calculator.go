package main

import (
	"log"
	"strings"
	"unicode/utf8"
)

// TokenCalculator handles token conversion rates based on user tiers
type TokenCalculator struct {
	freeConversionRates map[string]float64
	paidConversionRates map[string]float64
}

// NewTokenCalculator creates a new token calculator with tier-specific rates
func NewTokenCalculator() *TokenCalculator {
	return &TokenCalculator{
		// Free tier: all cheapest models have same cost (1:1 ratio)
		freeConversionRates: map[string]float64{
			"openai":    1.0, // GPT-4o-mini baseline
			"anthropic": 1.0, // Claude Haiku 3 - comparable cost
			"gemini":    1.0, // Gemini Flash-lite - comparable cost
		},
		// Paid tiers: full model access with different costs
		paidConversionRates: map[string]float64{
			"openai":    1.0,  // Baseline
			"anthropic": 7.5,  // 7.5x more expensive (premium models)
			"gemini":    1.25, // 1.25x more expensive (premium models)
		},
	}
}

// CalculateEquivalentTokens converts actual tokens to GPT-equivalent tokens based on user tier
func (tc *TokenCalculator) CalculateEquivalentTokens(provider string, actualTokens int64, userTier string) int64 {
	var rates map[string]float64

	// Use different conversion rates based on user tier
	if userTier == "free" {
		rates = tc.freeConversionRates
	} else {
		rates = tc.paidConversionRates
	}

	rate, exists := rates[provider]
	if !exists {
		log.Printf("[TokenCalculator] Unknown provider '%s', using default rate 1.0", provider)
		rate = 1.0 // Default to GPT rate
	}

	equivalentTokens := int64(float64(actualTokens) * rate)

	log.Printf("[TokenCalculator] Provider: %s, Tier: %s, Actual: %d, Rate: %.2f, Equivalent: %d",
		provider, userTier, actualTokens, rate, equivalentTokens)

	return equivalentTokens
}

// GetConversionRate returns the conversion rate for a provider and tier
func (tc *TokenCalculator) GetConversionRate(provider string, userTier string) float64 {
	var rates map[string]float64

	if userTier == "free" {
		rates = tc.freeConversionRates
	} else {
		rates = tc.paidConversionRates
	}

	rate, exists := rates[provider]
	if !exists {
		return 1.0 // Default to GPT rate
	}

	return rate
}

// GetTokenLimitForTier returns the token limit for a given tier
func (tc *TokenCalculator) GetTokenLimitForTier(tier string) int64 {
	limits := map[string]int64{
		"free":     100000,  // 100K tokens
		"plus":     550000,  // 550K GPT-equivalent tokens
		"pro":      1200000, // 1.2M GPT-equivalent tokens
		"pro_plus": 3000000, // 3M GPT-equivalent tokens
	}

	limit, exists := limits[tier]
	if !exists {
		log.Printf("[TokenCalculator] Unknown tier '%s', using free tier limit", tier)
		return limits["free"]
	}

	return limit
}

// GetAllowedModelsForTier returns the models allowed for a given tier
func (tc *TokenCalculator) GetAllowedModelsForTier(tier string) map[string][]string {
	switch tier {
	case "free":
		return map[string][]string{
			"openai":    {"gpt-4o-mini"},
			"anthropic": {"claude-3-haiku"},
			"gemini":    {"gemini-2.0-flash-lite"},
		}
    case "plus", "pro", "pro_plus":
		// Paid tiers get access to all models
		return map[string][]string{
            "openai":    {"gpt-4o-mini", "gpt-4o", "gpt-5.1", "gpt-5-mini"},
            "anthropic": {"claude-3-haiku", "claude-sonnet-4-5", "claude-opus-4-5"},
            "gemini":    {"gemini-2.0-flash-lite", "gemini-3-pro-preview", "gemini-2.5-flash"},
        }
	default:
		log.Printf("[TokenCalculator] Unknown tier '%s', using free tier models", tier)
		return tc.GetAllowedModelsForTier("free")
	}
}

// IsModelAllowedForTier checks if a model is allowed for a given tier
func (tc *TokenCalculator) IsModelAllowedForTier(model string, tier string) bool {
	allowedModels := tc.GetAllowedModelsForTier(tier)

	// Determine provider from model name
	provider := tc.getProviderFromModel(model)
	if provider == "" {
		return false
	}

	models, exists := allowedModels[provider]
	if !exists {
		return false
	}

	for _, allowedModel := range models {
		if allowedModel == model {
			return true
		}
	}

	return false
}

// getProviderFromModel determines the provider based on model name
func (tc *TokenCalculator) getProviderFromModel(model string) string {
	if len(model) == 0 {
		return ""
	}

	// Check model prefixes to determine provider
	switch {
	case model[:3] == "gpt" || model[:4] == "gpt-":
		return "openai"
	case model[:6] == "claude":
		return "anthropic"
	case model[:6] == "gemini":
		return "gemini"
	default:
		log.Printf("[TokenCalculator] Unknown model format: %s", model)
		return ""
	}
}

// TierInfo contains information about a pricing tier
type TierInfo struct {
	ID            string              `json:"id"`
	Name          string              `json:"name"`
	Price         int                 `json:"price"` // Price in cents
	TokenLimit    int64               `json:"tokenLimit"`
	AllowedModels map[string][]string `json:"allowedModels"`
	Features      []string            `json:"features"`
}

// GetAllTierInfo returns information about all pricing tiers
func (tc *TokenCalculator) GetAllTierInfo() []TierInfo {
	return []TierInfo{
		{
			ID:            "free",
			Name:          "Free",
			Price:         0,
			TokenLimit:    100000,
			AllowedModels: tc.GetAllowedModelsForTier("free"),
			Features:      []string{"Basic support", "Cost-effective models"},
		},
		{
			ID:            "plus",
			Name:          "Plus",
			Price:         999, // $9.99
			TokenLimit:    550000,
			AllowedModels: tc.GetAllowedModelsForTier("plus"),
			Features:      []string{"Priority support", "All models", "Usage analytics"},
		},
		{
			ID:            "pro",
			Name:          "Pro",
			Price:         1999, // $19.99
			TokenLimit:    1200000,
			AllowedModels: tc.GetAllowedModelsForTier("pro"),
			Features:      []string{"Priority support", "All models", "Advanced analytics", "Early access"},
		},
		{
			ID:            "pro_plus",
			Name:          "Pro Plus",
			Price:         4999, // $49.99
			TokenLimit:    3000000,
			AllowedModels: tc.GetAllowedModelsForTier("pro_plus"),
			Features:      []string{"Premium support", "All models", "Detailed analytics", "API access", "Custom limits"},
		},
	}
}

// EstimateTokenCount provides a rough estimate of token count for a text
// This is used for pre-request validation - actual count comes from API response
func (tc *TokenCalculator) EstimateTokenCount(text string) int64 {
	// Rough estimation: ~4 characters per token for English text
	// This is conservative and will be replaced by actual API response
	return int64(len(text) / 4)
}

// GetModelContextWindow returns the maximum context window size for a model in tokens
func (tc *TokenCalculator) GetModelContextWindow(model string) int64 {
	contextWindows := map[string]int64{
		// OpenAI models
		"gpt-4o-mini":           128000, // 128K context
		"gpt-4o":                128000, // 128K context
		"gpt-5.1":               128000, // 128K context
		"gpt-5-mini":            128000, // 128K context

    // Anthropic models
    "claude-3-haiku":           200000, // 200K context
    "claude-sonnet-4-5":        200000, // 200K context
    "claude-opus-4-5":          200000, // 200K context

		// Gemini models
		"gemini-2.0-flash-lite": 1000000, // 1M context
		"gemini-3-pro-preview":  2000000, // 2M context
		"gemini-2.5-flash":      1000000, // 1M context
	}

	contextWindow, exists := contextWindows[model]
	if !exists {
		// Default to conservative 8K context for unknown models
		log.Printf("[TokenCalculator] Unknown model '%s', using default 8K context window", model)
		return 8000
	}

	return contextWindow
}

// AccurateTokenCount provides more accurate token counting based on model type
func (tc *TokenCalculator) AccurateTokenCount(text string, model string) int64 {
	provider := tc.getProviderFromModel(model)

	switch provider {
	case "openai":
		return tc.openaiTokenCount(text)
	case "anthropic":
		return tc.anthropicTokenCount(text)
	case "gemini":
		return tc.geminiTokenCount(text)
	default:
		return tc.EstimateTokenCount(text)
	}
}

// openaiTokenCount estimates tokens for OpenAI models (GPT tokenizer approximation)
func (tc *TokenCalculator) openaiTokenCount(text string) int64 {
	// GPT tokenizer approximation
	// More sophisticated than simple char/4, considers:
	// - Word boundaries
	// - Common tokens
	// - Unicode handling

	if len(text) == 0 {
		return 0
	}

	// Split on whitespace and punctuation
	words := strings.FieldsFunc(text, func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\n' || r == '.' || r == ',' || r == '!' || r == '?'
	})

	tokenCount := int64(0)
	for _, word := range words {
		wordLen := utf8.RuneCountInString(word)
		if wordLen == 0 {
			continue
		}

		// Estimate tokens per word based on length
		if wordLen <= 3 {
			tokenCount += 1 // Short words usually 1 token
		} else if wordLen <= 6 {
			tokenCount += 1 // Medium words usually 1-2 tokens, average 1.5
		} else if wordLen <= 10 {
			tokenCount += 2 // Longer words usually 2-3 tokens
		} else {
			tokenCount += int64(wordLen / 4) // Very long words, use char/4 approximation
		}
	}

	// Add small buffer for punctuation and special tokens
	return tokenCount + int64(len(words)/10)
}

// anthropicTokenCount estimates tokens for Anthropic models (Claude tokenizer approximation)
func (tc *TokenCalculator) anthropicTokenCount(text string) int64 {
	// Claude tokenizer is similar to GPT but slightly different
	// Generally similar ratios but may handle some cases differently
	baseCount := tc.openaiTokenCount(text)

	// Claude tokenizer tends to be slightly more efficient for longer texts
	if baseCount > 1000 {
		return int64(float64(baseCount) * 0.95) // 5% fewer tokens for long texts
	}

	return baseCount
}

// geminiTokenCount estimates tokens for Google Gemini models
func (tc *TokenCalculator) geminiTokenCount(text string) int64 {
	// Gemini tokenizer approximation
	// Generally more efficient than GPT tokenizer
	baseCount := tc.openaiTokenCount(text)

	// Gemini tends to use fewer tokens overall
	return int64(float64(baseCount) * 0.85) // ~15% fewer tokens than GPT estimate
}

// GetTokenBudgetAllocation returns recommended token allocation for context building
func (tc *TokenCalculator) GetTokenBudgetAllocation(model string) TokenBudget {
	contextWindow := tc.GetModelContextWindow(model)

	// Reserve space for system prompts and response generation
	availableForContext := int64(float64(contextWindow) * 0.75) // Use 75% for context

	return TokenBudget{
		Total:               contextWindow,
		AvailableForContext: availableForContext,
		RecentMessages:      int64(float64(availableForContext) * 0.70), // 70% for recent messages
		RAGResults:          int64(float64(availableForContext) * 0.20), // 20% for RAG results
		SystemPrompts:       int64(float64(availableForContext) * 0.10), // 10% for system prompts
	}
}

// TokenBudget represents the token allocation strategy for context building
type TokenBudget struct {
	Total               int64 `json:"total"`
	AvailableForContext int64 `json:"availableForContext"`
	RecentMessages      int64 `json:"recentMessages"`
	RAGResults          int64 `json:"ragResults"`
	SystemPrompts       int64 `json:"systemPrompts"`
}
