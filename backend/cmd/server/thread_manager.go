package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
	"github.com/google/uuid"
)

type Thread struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"` // "thread"
	UserID    string         `json:"userId"`
	ThreadID  string         `json:"threadId"` // = ID (partition key)
	Title     string         `json:"title"`
	CreatedAt int64          `json:"createdAt"`
	UpdatedAt int64          `json:"updatedAt"`
	Metadata  ThreadMetadata `json:"metadata"`
	Status    string         `json:"status"` // active | archived
}

type ThreadMetadata struct {
	MessageCount int    `json:"messageCount"`
	LastModel    string `json:"lastModel"`
}

type UserThreadList struct {
	ID       string   `json:"id"`
	Type     string   `json:"type"` // "user_threads"
	UserID   string   `json:"userId"`
	ThreadID string   `json:"threadId"` // = userId (partition)
	Threads  []string `json:"threads"`
}

type EnhancedMessage struct {
	ID        string `json:"id"`
	Type      string `json:"type"` // "message"
	ThreadID  string `json:"threadId"`
	UserID    string `json:"userId"`
	Role      string `json:"role"`
	Content   string `json:"content"` // Legacy: Plaintext content (for backward compatibility)
	Provider  string `json:"provider"`
	Model     string `json:"model"`
	CreatedAt int64  `json:"createdAt"`
	VectorID  string `json:"vectorId,omitempty"`

	// Encryption fields (new messages will use these)
	EncryptedContent string `json:"encryptedContent,omitempty"`
	EncryptedAESKey  string `json:"encryptedAESKey,omitempty"`
	IV               string `json:"iv,omitempty"`
	AuthTag          string `json:"authTag,omitempty"`
	Algorithm        string `json:"algorithm,omitempty"`

	// Content parsing fields (for AI assistant responses)
	ParsedContent   *ParsedContent `json:"parsedContent,omitempty"`
	HasCodeBlocks   bool           `json:"hasCodeBlocks,omitempty"`
	HasMathBlocks   bool           `json:"hasMathBlocks,omitempty"`
	HasDiagrams     bool           `json:"hasDiagrams,omitempty"`
	ParsedAt        int64          `json:"parsedAt,omitempty"`
	ParsingProvider string         `json:"parsingProvider,omitempty"`
}

type ThreadManager struct {
	cosmosStore       *CosmosStore
	vectorStore       *VectorStore
	encryptionService *TEEEncryptionService
}

func NewThreadManager(cosmos *CosmosStore, vectorStore *VectorStore, encSvc *TEEEncryptionService) (*ThreadManager, error) {
	log.Printf("[ThreadManager] Initialized using messages container (threads + messages)")
	return &ThreadManager{
		cosmosStore:       cosmos,
		vectorStore:       vectorStore,
		encryptionService: encSvc,
	}, nil
}

func (tm *ThreadManager) CreateThread(ctx context.Context, userID, title string) (*Thread, error) {
	threadID := "thread_" + uuid.NewString()
	now := time.Now().UnixMilli()
	thread := &Thread{
		ID:        threadID,
		Type:      "thread",
		UserID:    userID,
		ThreadID:  threadID,
		Title:     title,
		CreatedAt: now,
		UpdatedAt: now,
		Metadata:  ThreadMetadata{MessageCount: 0},
		Status:    "active",
	}

	body, err := json.Marshal(thread)
	if err != nil {
		return nil, fmt.Errorf("marshal thread: %w", err)
	}
	pk := azcosmos.NewPartitionKeyString(threadID)
	if _, err := tm.cosmosStore.container.CreateItem(ctx, pk, body, nil); err != nil {
		return nil, fmt.Errorf("create thread item: %w", err)
	}

	// Update user's list (best-effort)
	tm.addThreadToUserList(context.Background(), userID, threadID)
	return thread, nil
}

func (tm *ThreadManager) addThreadToUserList(ctx context.Context, userID, threadID string) {
	userListID := "user_threads_" + userID
	pk := azcosmos.NewPartitionKeyString(userID)

	// Try read
	resp, err := tm.cosmosStore.container.ReadItem(ctx, pk, userListID, nil)
	var list UserThreadList
	if err != nil {
		list = UserThreadList{
			ID:       userListID,
			Type:     "user_threads",
			UserID:   userID,
			ThreadID: userID,
			Threads:  []string{threadID},
		}
		body, _ := json.Marshal(list)
		if _, err := tm.cosmosStore.container.CreateItem(ctx, pk, body, nil); err != nil {
			log.Printf("[ThreadManager] create user thread list: %v", err)
		}
		return
	}

	if err := json.Unmarshal(resp.Value, &list); err != nil {
		log.Printf("[ThreadManager] unmarshal user thread list: %v", err)
		return
	}
	// Prepend if missing
	found := false
	for _, t := range list.Threads {
		if t == threadID {
			found = true
			break
		}
	}
	if !found {
		list.Threads = append([]string{threadID}, list.Threads...)
	}
	body, _ := json.Marshal(list)
	if _, err := tm.cosmosStore.container.UpsertItem(ctx, pk, body, nil); err != nil {
		log.Printf("[ThreadManager] upsert user thread list: %v", err)
	}
}

func (tm *ThreadManager) removeThreadFromUserList(ctx context.Context, userID, threadID string) {
	userListID := "user_threads_" + userID
	pk := azcosmos.NewPartitionKeyString(userID)

	resp, err := tm.cosmosStore.container.ReadItem(ctx, pk, userListID, nil)
	if err != nil {
		log.Printf("[ThreadManager] removeThreadFromUserList read: %v", err)
		return
	}

	var list UserThreadList
	if err := json.Unmarshal(resp.Value, &list); err != nil {
		log.Printf("[ThreadManager] removeThreadFromUserList unmarshal: %v", err)
		return
	}

	filtered := make([]string, 0, len(list.Threads))
	for _, id := range list.Threads {
		if id != threadID {
			filtered = append(filtered, id)
		}
	}
	list.Threads = filtered

	body, err := json.Marshal(list)
	if err != nil {
		log.Printf("[ThreadManager] removeThreadFromUserList marshal: %v", err)
		return
	}
	if _, err := tm.cosmosStore.container.UpsertItem(ctx, pk, body, nil); err != nil {
		log.Printf("[ThreadManager] removeThreadFromUserList upsert: %v", err)
	}
}

func (tm *ThreadManager) GetThread(ctx context.Context, threadID, userID string) (*Thread, error) {
	pk := azcosmos.NewPartitionKeyString(threadID)
	resp, err := tm.cosmosStore.container.ReadItem(ctx, pk, threadID, nil)
	if err != nil {
		return nil, fmt.Errorf("read thread: %w", err)
	}
	var thread Thread
	if err := json.Unmarshal(resp.Value, &thread); err != nil {
		return nil, fmt.Errorf("unmarshal thread: %w", err)
	}
	if thread.Type != "thread" || thread.UserID != userID {
		return nil, fmt.Errorf("thread not found or access denied")
	}
	return &thread, nil
}

func (tm *ThreadManager) ListUserThreads(ctx context.Context, userID string, limit int) ([]*Thread, error) {
	userListID := "user_threads_" + userID
	pk := azcosmos.NewPartitionKeyString(userID)

	resp, err := tm.cosmosStore.container.ReadItem(ctx, pk, userListID, nil)
	if err != nil {
		log.Printf("[ThreadManager] no list for user %s (new user?)", userID)
		return []*Thread{}, nil
	}
	var list UserThreadList
	if err := json.Unmarshal(resp.Value, &list); err != nil {
		return nil, fmt.Errorf("unmarshal user list: %w", err)
	}

	threads := make([]*Thread, 0, min(limit, len(list.Threads)))
	for i, id := range list.Threads {
		if i >= limit {
			break
		}
		th, err := tm.GetThread(ctx, id, userID)
		if err != nil {
			log.Printf("[ThreadManager] get thread %s: %v", id, err)
			continue
		}
		threads = append(threads, th)
	}
	return threads, nil
}

func (tm *ThreadManager) SearchUserThreads(ctx context.Context, userID, query string, limit, offset int) ([]*Thread, error) {
	userListID := "user_threads_" + userID
	pk := azcosmos.NewPartitionKeyString(userID)

	resp, err := tm.cosmosStore.container.ReadItem(ctx, pk, userListID, nil)
	if err != nil {
		log.Printf("[ThreadManager] no list for user %s (new user?)", userID)
		return []*Thread{}, nil
	}

	var list UserThreadList
	if err := json.Unmarshal(resp.Value, &list); err != nil {
		return nil, fmt.Errorf("unmarshal user list: %w", err)
	}

	// Search through threads by title
	var matchingThreads []*Thread
	queryLower := strings.ToLower(query)

	for _, threadID := range list.Threads {
		thread, err := tm.GetThread(ctx, threadID, userID)
		if err != nil {
			log.Printf("[ThreadManager] get thread %s: %v", threadID, err)
			continue
		}

		// Check if title contains the search query (case insensitive)
		if strings.Contains(strings.ToLower(thread.Title), queryLower) {
			matchingThreads = append(matchingThreads, thread)
		}
	}

	// Apply pagination
	start := offset
	if start > len(matchingThreads) {
		return []*Thread{}, nil
	}

	end := start + limit
	if end > len(matchingThreads) {
		end = len(matchingThreads)
	}

	return matchingThreads[start:end], nil
}

func (tm *ThreadManager) GetThreadMessages(ctx context.Context, threadID, userID string, limit int) ([]*EnhancedMessage, error) {
	if _, err := tm.GetThread(ctx, threadID, userID); err != nil {
		return nil, err
	}

	pk := azcosmos.NewPartitionKeyString(threadID)
	query := "SELECT * FROM c WHERE c.threadId = @threadId AND (c.type = 'message' OR IS_NULL(c.type)) ORDER BY c.createdAt DESC"
	params := []azcosmos.QueryParameter{{Name: "@threadId", Value: threadID}}
	opts := &azcosmos.QueryOptions{QueryParameters: params, PageSizeHint: int32(limit)}

	pager := tm.cosmosStore.container.NewQueryItemsPager(query, pk, opts)

	var messages []*EnhancedMessage
	for pager.More() && len(messages) < limit {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("query messages: %w", err)
		}
		for _, it := range page.Items {
			var doc map[string]any
			if err := json.Unmarshal(it, &doc); err != nil {
				continue
			}
			if t, ok := doc["type"].(string); ok && t == "thread" {
				continue
			}
			var m EnhancedMessage
			if err := json.Unmarshal(it, &m); err != nil {
				continue
			}
			if m.Type == "" {
				m.Type = "message"
			}
			messages = append(messages, &m)
			if len(messages) >= limit {
				break
			}
		}
	}

	// Decrypt messages before returning
	for _, msg := range messages {
		if err := tm.decryptMessageContent(msg); err != nil {
			log.Printf("❌ Failed to decrypt message %s: %v", msg.ID, err)
			// Continue with other messages - don't fail the entire request
		}
	}

	// ASC
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}
	return messages, nil
}

func (tm *ThreadManager) SaveMessage(ctx context.Context, userID string, msg *EnhancedMessage) error {
	if msg == nil {
		return fmt.Errorf("message is required")
	}

	if msg.ThreadID == "" {
		return fmt.Errorf("threadID is required")
	}

	if _, err := tm.GetThread(ctx, msg.ThreadID, userID); err != nil {
		return err
	}

	if msg.UserID != "" && msg.UserID != userID {
		return fmt.Errorf("message user mismatch")
	}
	msg.UserID = userID
	log.Printf("[ThreadManager] Saving message to thread: %s", msg.ThreadID)

	if msg.ID == "" {
		msg.ID = "msg_" + uuid.NewString()
		log.Printf("[ThreadManager] Generated new message ID: %s", msg.ID)
	} else {
		log.Printf("[ThreadManager] Using existing message ID: %s", msg.ID)
	}

	if msg.CreatedAt == 0 {
		msg.CreatedAt = time.Now().UnixMilli()
		log.Printf("[ThreadManager] Set creation timestamp: %d", msg.CreatedAt)
	}

	msg.Type = "message"
	log.Printf("[ThreadManager] Message details - Role: %s, Provider: %s, Model: %s, Content length: %d",
		msg.Role, msg.Provider, msg.Model, len(msg.Content))

	// Save message
	log.Printf("[ThreadManager] Marshaling message for Cosmos DB")
	body, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[ThreadManager] Failed to marshal message: %v", err)
		return fmt.Errorf("marshal message: %w", err)
	}

	pk := azcosmos.NewPartitionKeyString(msg.ThreadID)
	log.Printf("[ThreadManager] Saving message to Cosmos DB with partition key: %s", msg.ThreadID)
	if _, err := tm.cosmosStore.container.CreateItem(ctx, pk, body, nil); err != nil {
		log.Printf("[ThreadManager] Failed to save message to Cosmos DB: %v", err)
		return fmt.Errorf("save message: %w", err)
	}

	log.Printf("[ThreadManager] Successfully saved message to Cosmos DB: %s", msg.ID)

	// Update thread doc (best-effort; simple read-modify-replace)
	go func() {
		uCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		thread, err := tm.GetThread(uCtx, msg.ThreadID, msg.UserID)
		if err != nil {
			log.Printf("[ThreadManager] update thread get: %v", err)
			return
		}
		thread.Metadata.MessageCount++
		thread.Metadata.LastModel = msg.Model
		thread.UpdatedAt = time.Now().UnixMilli()

		tb, _ := json.Marshal(thread)
		tpk := azcosmos.NewPartitionKeyString(msg.ThreadID)
		if _, err := tm.cosmosStore.container.ReplaceItem(uCtx, tpk, thread.ID, tb, nil); err != nil {
			log.Printf("[ThreadManager] update thread replace: %v", err)
		}
	}()

	// Generate embedding async
	if tm.vectorStore != nil && msg.Role != "system" {
		go tm.generateAndStoreEmbedding(msg)
	}

	log.Printf("[ThreadManager] saved message: %s (thread=%s)", msg.ID, msg.ThreadID)
	return nil
}

// decryptMessageContent decrypts the content of an EnhancedMessage if it's encrypted
// This method accesses the global encryptionService
func (tm *ThreadManager) decryptMessageContent(msg *EnhancedMessage) error {
	// Use the injected encryption service
	if tm.encryptionService == nil {
		// If encryption service is not available, leave message as is
		return nil
	}

	// Check if message has encrypted content
	if msg.EncryptedContent == "" {
		// This is a legacy plaintext message, no decryption needed
		return nil
	}

	// Message is encrypted, decrypt it
	encryptedMsg := &EncryptedMessage{
		EncryptedContent: msg.EncryptedContent,
		EncryptedAESKey:  msg.EncryptedAESKey,
		IV:               msg.IV,
		AuthTag:          msg.AuthTag,
		Algorithm:        msg.Algorithm,
	}

	decryptedContent, err := tm.encryptionService.DecryptMessage(encryptedMsg)
	if err != nil {
		return fmt.Errorf("decryption failed: %w", err)
	}

	// Update the message with decrypted content
	msg.Content = decryptedContent
	// Clear encrypted fields for the response (keep them in storage)
	msg.EncryptedContent = ""
	msg.EncryptedAESKey = ""
	msg.IV = ""
	msg.AuthTag = ""
	msg.Algorithm = ""

	return nil
}

// getMessage retrieves a single message document from Cosmos DB for the given thread/message ID pair.
func (tm *ThreadManager) getMessage(ctx context.Context, threadID, messageID string) (*EnhancedMessage, error) {
	pk := azcosmos.NewPartitionKeyString(threadID)
	resp, err := tm.cosmosStore.container.ReadItem(ctx, pk, messageID, nil)
	if err != nil {
		return nil, fmt.Errorf("read message: %w", err)
	}

	var msg EnhancedMessage
	if err := json.Unmarshal(resp.Value, &msg); err != nil {
		return nil, fmt.Errorf("unmarshal message: %w", err)
	}

	if msg.Type == "" {
		msg.Type = "message"
	}
	if msg.ThreadID == "" {
		msg.ThreadID = threadID
	}

	return &msg, nil
}

// getMessagesBatch retrieves multiple messages from Cosmos DB in a single query
func (tm *ThreadManager) getMessagesBatch(ctx context.Context, threadID string, messageIDs []string) ([]*EnhancedMessage, error) {
	if len(messageIDs) == 0 {
		return []*EnhancedMessage{}, nil
	}

	pk := azcosmos.NewPartitionKeyString(threadID)

	// Build IN query for batch fetch
	placeholders := make([]string, len(messageIDs))
	params := make([]azcosmos.QueryParameter, len(messageIDs))
	for i, id := range messageIDs {
		placeholders[i] = fmt.Sprintf("@id%d", i)
		params[i] = azcosmos.QueryParameter{Name: placeholders[i], Value: id}
	}

	query := fmt.Sprintf("SELECT * FROM c WHERE c.id IN (%s)", strings.Join(placeholders, ", "))
	opts := &azcosmos.QueryOptions{QueryParameters: params}

	pager := tm.cosmosStore.container.NewQueryItemsPager(query, pk, opts)

	var messages []*EnhancedMessage
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("query messages batch: %w", err)
		}
		for _, item := range page.Items {
			var msg EnhancedMessage
			if err := json.Unmarshal(item, &msg); err != nil {
				log.Printf("[ThreadManager] unmarshal message in batch: %v", err)
				continue
			}
			if msg.Type == "" {
				msg.Type = "message"
			}
			if msg.ThreadID == "" {
				msg.ThreadID = threadID
			}
			messages = append(messages, &msg)
		}
	}

	return messages, nil
}

func (tm *ThreadManager) generateAndStoreEmbedding(msg *EnhancedMessage) {
	log.Printf("[Vector] Starting embedding generation for message: %s (thread: %s)", msg.ID, msg.ThreadID)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Use plaintext content for embedding; decrypt on-the-fly if stored content is encrypted
	plaintext := msg.Content
	if plaintext == "" && msg.EncryptedContent != "" && tm.encryptionService != nil {
		dec, err := tm.encryptionService.DecryptMessage(&EncryptedMessage{
			EncryptedContent: msg.EncryptedContent,
			EncryptedAESKey:  msg.EncryptedAESKey,
			IV:               msg.IV,
			AuthTag:          msg.AuthTag,
			Algorithm:        msg.Algorithm,
		})
		if err != nil {
			log.Printf("[Vector] Failed to decrypt content for embedding %s: %v", msg.ID, err)
		} else {
			plaintext = dec
		}
	}

	if strings.TrimSpace(plaintext) == "" {
		log.Printf("[Vector] Skipping embedding for message %s: empty content", msg.ID)
		return
	}

	log.Printf("[Vector] Generating embedding for content (length: %d)", len(plaintext))
	emb, err := tm.vectorStore.GenerateEmbedding(ctx, plaintext)
	if err != nil {
		log.Printf("[Vector] Failed to generate embedding for message %s: %v", msg.ID, err)
		return
	}

	log.Printf("[Vector] Successfully generated embedding (dim: %d)", len(emb))

	// Extract UUID from msg.ID (remove "msg_" prefix for Qdrant)
	vectorID := msg.ID
	if strings.HasPrefix(vectorID, "msg_") {
		vectorID = vectorID[4:] // Remove "msg_" prefix
		log.Printf("[Vector] Converted vector ID from %s to %s for Qdrant", msg.ID, vectorID)
	}

	vectorDoc := &VectorDocument{
		ID:        vectorID,
		ThreadID:  msg.ThreadID,
		MessageID: msg.ID,
		UserID:    msg.UserID,
		Content:   "",
		Embedding: emb,
		Metadata: map[string]any{
			"role":      msg.Role,
			"timestamp": msg.CreatedAt,
			"provider":  msg.Provider,
			"model":     msg.Model,
		},
	}

	log.Printf("[Vector] Storing vector document with ID: %s", vectorID)
	vecID, err := tm.vectorStore.StoreVector(ctx, vectorDoc)
	if err != nil {
		log.Printf("[Vector] Failed to store vector for message %s: %v", msg.ID, err)
		return
	}

	log.Printf("[Vector] Successfully stored vector with ID: %s", vecID)
	msg.VectorID = vecID

	// Update message with vector ID
	log.Printf("[Vector] Updating message %s with vector ID", msg.ID)
	b, _ := json.Marshal(msg)
	pk := azcosmos.NewPartitionKeyString(msg.ThreadID)
	if _, err := tm.cosmosStore.container.ReplaceItem(ctx, pk, msg.ID, b, nil); err != nil {
		log.Printf("[Vector] Failed to update message %s with vector ID: %v", msg.ID, err)
	} else {
		log.Printf("[Vector] Successfully updated message %s with vector ID", msg.ID)
	}
}

func (tm *ThreadManager) SearchThreadMessages(ctx context.Context, threadID, userID, query string, limit int) ([]*EnhancedMessage, error) {
	if _, err := tm.GetThread(ctx, threadID, userID); err != nil {
		return nil, err
	}

	if tm.vectorStore == nil {
		return tm.GetThreadMessages(ctx, threadID, userID, limit)
	}
	emb, err := tm.vectorStore.GenerateEmbedding(ctx, query)
	if err != nil {
		log.Printf("[ThreadManager] query embedding failed; fallback recent: %v", err)
		return tm.GetThreadMessages(ctx, threadID, userID, limit)
	}
	filter := map[string]any{"threadId": threadID}

	results, err := tm.vectorStore.SearchVectors(ctx, emb, filter, limit)
	if err != nil {
		log.Printf("[ThreadManager] vector search failed; fallback recent: %v", err)
		return tm.GetThreadMessages(ctx, threadID, userID, limit)
	}

	// Extract message IDs from vector search results
	messageIDs := make([]string, 0, len(results))
	for _, r := range results {
		messageIDs = append(messageIDs, r.MessageID)
	}

	// Batch fetch messages from Cosmos DB (more efficient than individual reads)
	messages, err := tm.getMessagesBatch(ctx, threadID, messageIDs)
	if err != nil {
		log.Printf("[ThreadManager] batch fetch failed: %v", err)
		// Fallback to recent messages to avoid hard failure
		return tm.GetThreadMessages(ctx, threadID, userID, limit)
	}

	// Decrypt all messages and skip those that fail
	msgs := make([]*EnhancedMessage, 0, len(messages))
	for _, msg := range messages {
		if err := tm.decryptMessageContent(msg); err != nil {
			log.Printf("[ThreadManager] decrypt message %s: %v (skipping)", msg.ID, err)
			continue // Skip messages that fail to decrypt
		}
		msgs = append(msgs, msg)
	}

	// Keep chronological order
	sort.Slice(msgs, func(i, j int) bool { return msgs[i].CreatedAt < msgs[j].CreatedAt })
	return msgs, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (tm *ThreadManager) RenameThread(ctx context.Context, threadID, userID, newTitle string) error {
	thread, err := tm.GetThread(ctx, threadID, userID)
	if err != nil {
		return fmt.Errorf("get thread: %w", err)
	}

	thread.Title = newTitle
	thread.UpdatedAt = time.Now().UnixMilli()

	body, err := json.Marshal(thread)
	if err != nil {
		return fmt.Errorf("marshal thread: %w", err)
	}

	pk := azcosmos.NewPartitionKeyString(threadID)
	_, err = tm.cosmosStore.container.ReplaceItem(ctx, pk, thread.ID, body, nil)
	if err != nil {
		return fmt.Errorf("update thread: %w", err)
	}

	return nil
}

func (tm *ThreadManager) DeleteThread(ctx context.Context, threadID, userID string) error {
	thread, err := tm.GetThread(ctx, threadID, userID)
	if err != nil {
		return fmt.Errorf("get thread: %w", err)
	}

	pk := azcosmos.NewPartitionKeyString(threadID)

	// Delete all items within the partition (messages + thread)
	query := "SELECT c.id FROM c"
	pager := tm.cosmosStore.container.NewQueryItemsPager(query, pk, nil)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("list partition items: %w", err)
		}
		for _, item := range page.Items {
			var doc map[string]any
			if err := json.Unmarshal(item, &doc); err != nil {
				log.Printf("[ThreadManager] delete thread %s: unmarshal item: %v", threadID, err)
				continue
			}
			id, _ := doc["id"].(string)
			if id == "" {
				continue
			}
			if _, err := tm.cosmosStore.container.DeleteItem(ctx, pk, id, nil); err != nil {
				log.Printf("[ThreadManager] delete item %s from thread %s: %v", id, threadID, err)
			}
		}
	}

	// Remove thread from user's thread list (best effort)
	if ctx.Err() == nil {
		tm.removeThreadFromUserList(ctx, userID, thread.ID)
	} else {
		// Fall back to background context if original one was cancelled
		go tm.removeThreadFromUserList(context.Background(), userID, thread.ID)
	}

	// Ensure the thread document is gone (ignore if already deleted)
	if _, err := tm.cosmosStore.container.DeleteItem(ctx, pk, thread.ID, nil); err != nil && !isNotFound(err) {
		return fmt.Errorf("delete thread: %w", err)
	}

	return nil
}
