package main

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
)

type ContextConfig struct {
	MaxMessages    int    `json:"maxMessages"` // Deprecated: kept for backwards compatibility
	RAGCount       int    `json:"ragCount"`    // Deprecated: kept for backwards compatibility
	RAGEnabled     bool   `json:"ragEnabled"`
	Strategy       string `json:"strategy"`       // recent | rag | hybrid
	Model          string `json:"model"`          // Model name for token-aware processing
	TokenBudget    int64  `json:"tokenBudget"`    // Override default token budget
	UseTokenLimits bool   `json:"useTokenLimits"` // Use token-aware truncation instead of message count
}

type ContextBuilder struct {
	threadManager   ThreadManagerInterface
	vectorStore     *VectorStore
	tokenCalculator *TokenCalculator
}

func NewContextBuilder(tm ThreadManagerInterface, vs *VectorStore, tc *TokenCalculator) *ContextBuilder {
	return &ContextBuilder{
		threadManager:   tm,
		vectorStore:     vs,
		tokenCalculator: tc,
	}
}

func (cb *ContextBuilder) BuildContext(ctx context.Context, threadID, userID, currentQuery string, config *ContextConfig) ([]*EnhancedMessage, error) {
	if config == nil {
		config = cb.getDefaultConfig()
	}

	var contextMessages []*EnhancedMessage

	// Prefer token-aware building when TokenCalculator is available and model is specified
	// Fall back to UseTokenLimits flag for backwards compatibility
	if cb.tokenCalculator != nil && config.Model != "" {
		return cb.buildTokenAwareContext(ctx, threadID, userID, currentQuery, config)
	} else if config.UseTokenLimits && config.Model != "" && cb.tokenCalculator != nil {
		return cb.buildTokenAwareContext(ctx, threadID, userID, currentQuery, config)
	}

	// Fallback to original message-count based approach
	switch config.Strategy {
	case "recent":
		msgs, err := cb.threadManager.GetThreadMessages(ctx, threadID, userID, config.MaxMessages)
		if err != nil {
			log.Printf("[ContextBuilder] get recent: %v", err)
			return nil, err
		}
		contextMessages = msgs

	case "rag":
		if cb.vectorStore != nil && config.RAGEnabled && strings.TrimSpace(currentQuery) != "" {
			msgs, err := cb.threadManager.SearchThreadMessages(ctx, threadID, userID, currentQuery, config.RAGCount)
			if err != nil {
				log.Printf("[ContextBuilder] search similar: %v", err)
				msgs, _ = cb.threadManager.GetThreadMessages(ctx, threadID, userID, config.MaxMessages)
			}
			contextMessages = msgs
		} else {
			msgs, _ := cb.threadManager.GetThreadMessages(ctx, threadID, userID, config.MaxMessages)
			contextMessages = msgs
		}

	case "hybrid":
		fallthrough
	default:
		contextMessages = cb.buildHybridContext(ctx, threadID, userID, currentQuery, config)
	}

	log.Printf("[ContextBuilder] built context: thread=%s strategy=%s n=%d",
		threadID, config.Strategy, len(contextMessages))
	return contextMessages, nil
}

func (cb *ContextBuilder) buildHybridContext(ctx context.Context, threadID, userID, query string, config *ContextConfig) []*EnhancedMessage {
	recent, err := cb.threadManager.GetThreadMessages(ctx, threadID, userID, config.MaxMessages)
	if err != nil {
		log.Printf("[ContextBuilder] hybrid recent: %v", err)
		return nil
	}

	if cb.vectorStore == nil || !config.RAGEnabled || strings.TrimSpace(query) == "" {
		return recent
	}

	ragMessages, err := cb.threadManager.SearchThreadMessages(ctx, threadID, userID, query, config.RAGCount)
	if err != nil {
		log.Printf("[ContextBuilder] hybrid RAG: %v", err)
		return recent
	}

	// Deduplicate by ID
	m := make(map[string]*EnhancedMessage, len(recent)+len(ragMessages))
	for _, msg := range recent {
		m[msg.ID] = msg
	}
	for _, msg := range ragMessages {
		if _, ok := m[msg.ID]; !ok {
			m[msg.ID] = msg
		}
	}
	merged := make([]*EnhancedMessage, 0, len(m))
	for _, v := range m {
		merged = append(merged, v)
	}
	sort.Slice(merged, func(i, j int) bool { return merged[i].CreatedAt < merged[j].CreatedAt })
	return merged
}

func (cb *ContextBuilder) GetRelevantContext(ctx context.Context, threadID, userID, query string, k int) ([]*EnhancedMessage, error) {
	if cb.vectorStore == nil {
		return cb.threadManager.GetThreadMessages(ctx, threadID, userID, k)
	}
	return cb.threadManager.SearchThreadMessages(ctx, threadID, userID, query, k)
}

func (cb *ContextBuilder) FormatContextForLLM(messages []*EnhancedMessage, currentQuery string) []ChatMessage {
	out := make([]ChatMessage, 0, len(messages)+1)
	for _, msg := range messages {
		out = append(out, ChatMessage{Role: msg.Role, Content: msg.Content})
	}
	// Avoid duplicating the just-saved user turn
	if strings.TrimSpace(currentQuery) != "" {
		if len(out) == 0 || out[len(out)-1].Role != "user" || out[len(out)-1].Content != currentQuery {
			out = append(out, ChatMessage{Role: "user", Content: currentQuery})
		}
	}
	return out
}

func (cb *ContextBuilder) getDefaultConfig() *ContextConfig {
	return &ContextConfig{
		MaxMessages:    20, // Deprecated: kept for backwards compatibility
		RAGCount:       10, // Deprecated: kept for backwards compatibility
		RAGEnabled:     true,
		Strategy:       "hybrid",
		UseTokenLimits: true, // Default to token-aware processing (recommended)
	}
}

func (cb *ContextBuilder) BuildPromptWithContext(context []*EnhancedMessage, currentQuery string) string {
	var b strings.Builder
	if len(context) > 0 {
		b.WriteString("Based on the following conversation history:\n\n")
		for _, m := range context {
			switch m.Role {
			case "user":
				b.WriteString(fmt.Sprintf("User: %s\n\n", m.Content))
			case "assistant":
				b.WriteString(fmt.Sprintf("Assistant: %s\n\n", m.Content))
			}
		}
		b.WriteString("\nCurrent question: ")
	}
	b.WriteString(currentQuery)
	return b.String()
}

// buildTokenAwareContext builds context using token limits instead of message counts
func (cb *ContextBuilder) buildTokenAwareContext(ctx context.Context, threadID, userID, currentQuery string, config *ContextConfig) ([]*EnhancedMessage, error) {
	// Get token budget for the model
	var budget TokenBudget
	if config.TokenBudget > 0 {
		// Use custom budget if specified
		budget = TokenBudget{
			Total:               config.TokenBudget,
			AvailableForContext: config.TokenBudget,
			RecentMessages:      int64(float64(config.TokenBudget) * 0.70),
			RAGResults:          int64(float64(config.TokenBudget) * 0.20),
			SystemPrompts:       int64(float64(config.TokenBudget) * 0.10),
		}
	} else {
		// Use model-specific budget
		budget = cb.tokenCalculator.GetTokenBudgetAllocation(config.Model)
	}

	log.Printf("[ContextBuilder] Token budget: total=%d, recent=%d, rag=%d",
		budget.AvailableForContext, budget.RecentMessages, budget.RAGResults)

	switch config.Strategy {
	case "recent":
		return cb.buildTokenAwareRecent(ctx, threadID, userID, budget, config)
	case "rag":
		return cb.buildTokenAwareRAG(ctx, threadID, userID, currentQuery, budget, config)
	case "hybrid":
		fallthrough
	default:
		return cb.buildTokenAwareHybrid(ctx, threadID, userID, currentQuery, budget, config)
	}
}

// buildTokenAwareRecent builds context using only recent messages within token budget
func (cb *ContextBuilder) buildTokenAwareRecent(ctx context.Context, threadID, userID string, budget TokenBudget, config *ContextConfig) ([]*EnhancedMessage, error) {
	// Get a large batch of recent messages to work with
	allMessages, err := cb.threadManager.GetThreadMessages(ctx, threadID, userID, 200)
	if err != nil {
		log.Printf("[ContextBuilder] token-aware recent: %v", err)
		return nil, err
	}

	// Truncate to fit token budget
	return cb.truncateMessagesToTokenBudget(allMessages, budget.RecentMessages, config.Model), nil
}

// buildTokenAwareRAG builds context using only RAG results within token budget
func (cb *ContextBuilder) buildTokenAwareRAG(ctx context.Context, threadID, userID, query string, budget TokenBudget, config *ContextConfig) ([]*EnhancedMessage, error) {
	if cb.vectorStore == nil || !config.RAGEnabled || strings.TrimSpace(query) == "" {
		// Fallback to recent messages
		return cb.buildTokenAwareRecent(ctx, threadID, userID, budget, config)
	}

	// Get more RAG results than we need, then truncate
	ragMessages, err := cb.threadManager.SearchThreadMessages(ctx, threadID, userID, query, 50)
	if err != nil {
		log.Printf("[ContextBuilder] token-aware RAG: %v", err)
		return cb.buildTokenAwareRecent(ctx, threadID, userID, budget, config)
	}

	// Truncate RAG results to fit budget
	return cb.truncateMessagesToTokenBudget(ragMessages, budget.RAGResults, config.Model), nil
}

// buildTokenAwareHybrid builds context using both recent and RAG messages within token budget
func (cb *ContextBuilder) buildTokenAwareHybrid(ctx context.Context, threadID, userID, query string, budget TokenBudget, config *ContextConfig) ([]*EnhancedMessage, error) {
	// Get recent messages
	recent, err := cb.threadManager.GetThreadMessages(ctx, threadID, userID, 200)
	if err != nil {
		log.Printf("[ContextBuilder] token-aware hybrid recent: %v", err)
		return nil, err
	}

	var ragMessages []*EnhancedMessage
	if cb.vectorStore != nil && config.RAGEnabled && strings.TrimSpace(query) != "" {
		ragMessages, err = cb.threadManager.SearchThreadMessages(ctx, threadID, userID, query, 50)
		if err != nil {
			log.Printf("[ContextBuilder] token-aware hybrid RAG: %v", err)
			ragMessages = nil
		}
	}

	// Truncate each set to their respective budgets
	recentTruncated := cb.truncateMessagesToTokenBudget(recent, budget.RecentMessages, config.Model)
	ragTruncated := cb.truncateMessagesToTokenBudget(ragMessages, budget.RAGResults, config.Model)

	// Merge and deduplicate
	merged := cb.mergeAndDeduplicateMessages(recentTruncated, ragTruncated)

	log.Printf("[ContextBuilder] token-aware hybrid: recent=%d rag=%d merged=%d",
		len(recentTruncated), len(ragTruncated), len(merged))

	return merged, nil
}

// truncateMessagesToTokenBudget truncates messages to fit within token budget
func (cb *ContextBuilder) truncateMessagesToTokenBudget(messages []*EnhancedMessage, tokenBudget int64, model string) []*EnhancedMessage {
	if len(messages) == 0 || tokenBudget <= 0 {
		return []*EnhancedMessage{}
	}

	result := make([]*EnhancedMessage, 0, len(messages))
	currentTokens := int64(0)

	// Process messages in reverse order (most recent first) to prioritize recent content
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		msgTokens := cb.tokenCalculator.AccurateTokenCount(msg.Content, model)

		if currentTokens+msgTokens <= tokenBudget {
			result = append([]*EnhancedMessage{msg}, result...) // Prepend to maintain chronological order
			currentTokens += msgTokens
		} else {
			// Try to include a truncated version of this message if it's important
			if len(result) == 0 && msgTokens > tokenBudget {
				// If this is the only message and it's too long, truncate it
				truncatedContent := cb.truncateContentToTokens(msg.Content, tokenBudget, model)
				if truncatedContent != "" {
					truncatedMsg := &EnhancedMessage{
						ID:        msg.ID,
						Content:   truncatedContent,
						Role:      msg.Role,
						CreatedAt: msg.CreatedAt,
					}
					result = append([]*EnhancedMessage{truncatedMsg}, result...)
				}
			}
			break
		}
	}

	log.Printf("[ContextBuilder] truncated: %d messages -> %d messages (%d tokens)",
		len(messages), len(result), currentTokens)

	return result
}

// truncateContentToTokens truncates content to fit within token limit
func (cb *ContextBuilder) truncateContentToTokens(content string, tokenLimit int64, model string) string {
	if tokenLimit <= 0 {
		return ""
	}

	currentTokens := cb.tokenCalculator.AccurateTokenCount(content, model)
	if currentTokens <= tokenLimit {
		return content
	}

	// Binary search to find the right truncation point
	words := strings.Fields(content)
	if len(words) == 0 {
		return ""
	}

	left, right := 0, len(words)
	bestContent := ""

	for left <= right {
		mid := (left + right) / 2
		truncated := strings.Join(words[:mid], " ")
		tokens := cb.tokenCalculator.AccurateTokenCount(truncated, model)

		if tokens <= tokenLimit {
			bestContent = truncated
			left = mid + 1
		} else {
			right = mid - 1
		}
	}

	if bestContent != "" {
		bestContent += "... [truncated]"
	}

	return bestContent
}

// mergeAndDeduplicateMessages merges two message slices and removes duplicates
func (cb *ContextBuilder) mergeAndDeduplicateMessages(recent, rag []*EnhancedMessage) []*EnhancedMessage {
	seen := make(map[string]bool)
	var result []*EnhancedMessage

	// Add all messages, deduplicating by ID
	allMessages := append([]*EnhancedMessage{}, recent...)
	allMessages = append(allMessages, rag...)

	for _, msg := range allMessages {
		if !seen[msg.ID] {
			seen[msg.ID] = true
			result = append(result, msg)
		}
	}

	// Sort by creation time to maintain chronological order
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt < result[j].CreatedAt
	})

	return result
}

// CalculateInputTokens calculates the total input tokens for a request (context + user message)
func (cb *ContextBuilder) CalculateInputTokens(contextMessages []*EnhancedMessage, userMessage string, model string) int64 {
	if cb.tokenCalculator == nil {
		// Fallback to simple estimation when no token calculator available
		totalContent := userMessage
		for _, msg := range contextMessages {
			totalContent += msg.Content
		}
		// Use basic estimation: ~4 characters per token
		return int64(len(totalContent) / 4)
	}

	totalTokens := int64(0)

	// Count tokens in context messages
	for _, msg := range contextMessages {
		totalTokens += cb.tokenCalculator.AccurateTokenCount(msg.Content, model)
	}

	// Count tokens in user message
	totalTokens += cb.tokenCalculator.AccurateTokenCount(userMessage, model)

	log.Printf("[ContextBuilder] Input token calculation: context=%d messages, user=%d chars, total=%d tokens",
		len(contextMessages), len(userMessage), totalTokens)

	return totalTokens
}
