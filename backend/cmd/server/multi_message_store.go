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

// MessageDocument for the messages container
type MessageDocument struct {
	ID        string `json:"id"`
	Type      string `json:"type"`     // "message"
	ThreadID  string `json:"threadId"` // Partition key
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
}

type MultiMessageStore struct {
	messagesContainer *azcosmos.ContainerClient
	threadStore       *MultiThreadStore
	vectorStore       *VectorStore
	encryptionService *TEEEncryptionService
}

func NewMultiMessageStore(multiCosmosStore *MultiCosmosStore, threadStore *MultiThreadStore, vectorStore *VectorStore, encSvc *TEEEncryptionService) (*MultiMessageStore, error) {
	return &MultiMessageStore{
		messagesContainer: multiCosmosStore.GetMessagesContainer(),
		threadStore:       threadStore,
		vectorStore:       vectorStore,
		encryptionService: encSvc,
	}, nil
}

func (mms *MultiMessageStore) SaveMessage(ctx context.Context, userID string, msg *MessageDocument) error {
	if msg == nil {
		return fmt.Errorf("message is required")
	}

	if msg.ThreadID == "" {
		return fmt.Errorf("threadID is required")
	}

	if _, err := mms.threadStore.GetThread(ctx, msg.ThreadID, userID); err != nil {
		return err
	}

	if msg.UserID != "" && msg.UserID != userID {
		return fmt.Errorf("message user mismatch")
	}
	msg.UserID = userID

	if msg.ID == "" {
		msg.ID = "msg_" + uuid.NewString()
	}
	if msg.CreatedAt == 0 {
		msg.CreatedAt = time.Now().UnixMilli()
	}
	msg.Type = "message"

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	pk := azcosmos.NewPartitionKeyString(msg.ThreadID)
	_, err = mms.messagesContainer.CreateItem(ctx, pk, msgBytes, nil)
	if err != nil {
		return fmt.Errorf("save message: %w", err)
	}

	// Update thread metadata asynchronously
	go mms.updateThreadMetadata(msg)

	// Generate embedding asynchronously
	if mms.vectorStore != nil && msg.Role != "system" {
		go mms.generateAndStoreEmbedding(msg)
	}

	log.Printf("✅ Saved message in messages container: %s (thread=%s)", msg.ID, msg.ThreadID)
	return nil
}

func (mms *MultiMessageStore) updateThreadMetadata(msg *MessageDocument) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get current message count for the thread
	count, err := mms.GetThreadMessageCount(ctx, msg.ThreadID)
	if err != nil {
		log.Printf("[MultiMessageStore] get message count: %v", err)
		return
	}

	// Update thread metadata
	if err := mms.threadStore.UpdateThreadMetadata(ctx, msg.ThreadID, msg.UserID, count, msg.Model); err != nil {
		log.Printf("[MultiMessageStore] update thread metadata: %v", err)
	}
}

func (mms *MultiMessageStore) GetThreadMessages(ctx context.Context, threadID, userID string, limit int) ([]*MessageDocument, error) {
	log.Printf("[MultiMessageStore] GetThreadMessages: threadID=%s, limit=%d", threadID, limit)

	if _, err := mms.threadStore.GetThread(ctx, threadID, userID); err != nil {
		return nil, err
	}

	pk := azcosmos.NewPartitionKeyString(threadID)
	query := "SELECT * FROM c WHERE c.threadId = @threadId AND c.type = 'message' ORDER BY c.createdAt DESC"
	params := []azcosmos.QueryParameter{{Name: "@threadId", Value: threadID}}
	opts := &azcosmos.QueryOptions{QueryParameters: params, PageSizeHint: int32(limit)}

	pager := mms.messagesContainer.NewQueryItemsPager(query, pk, opts)

	var messages []*MessageDocument
	for pager.More() && len(messages) < limit {
		page, err := pager.NextPage(ctx)
		if err != nil {
			log.Printf("[MultiMessageStore] Query error for thread %s: %v", threadID, err)
			return nil, fmt.Errorf("query messages: %w", err)
		}

		for _, item := range page.Items {
			var msg MessageDocument
			if err := json.Unmarshal(item, &msg); err != nil {
				log.Printf("[MultiMessageStore] Failed to unmarshal message for thread %s: %v", threadID, err)
				continue
			}
			messages = append(messages, &msg)
			if len(messages) >= limit {
				break
			}
		}
	}

	// Decrypt messages before returning
	decryptFailures := 0
	for _, msg := range messages {
		if err := mms.decryptMessageContent(msg); err != nil {
			decryptFailures++
			log.Printf("❌ Failed to decrypt message %s for thread %s: %v", msg.ID, threadID, err)
			// Continue with other messages - don't fail the entire request
		}
	}

	if decryptFailures > 0 {
		log.Printf("[MultiMessageStore] Warning: %d/%d messages failed decryption for thread %s", decryptFailures, len(messages), threadID)
	}

	// Return in chronological order (oldest first)
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	log.Printf("[MultiMessageStore] Retrieved %d messages for thread %s", len(messages), threadID)
	return messages, nil
}

func (mms *MultiMessageStore) GetThreadMessageCount(ctx context.Context, threadID string) (int, error) {
	pk := azcosmos.NewPartitionKeyString(threadID)
	query := "SELECT VALUE COUNT(1) FROM c WHERE c.threadId = @threadId AND c.type = 'message'"
	params := []azcosmos.QueryParameter{{Name: "@threadId", Value: threadID}}
	opts := &azcosmos.QueryOptions{QueryParameters: params}

	pager := mms.messagesContainer.NewQueryItemsPager(query, pk, opts)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return 0, fmt.Errorf("query message count: %w", err)
		}

		for _, item := range page.Items {
			var count int
			if err := json.Unmarshal(item, &count); err != nil {
				continue
			}
			return count, nil
		}
	}

	return 0, nil
}

func (mms *MultiMessageStore) SearchThreadMessages(ctx context.Context, threadID, userID, query string, limit int) ([]*MessageDocument, error) {
	if _, err := mms.threadStore.GetThread(ctx, threadID, userID); err != nil {
		return nil, err
	}

	if mms.vectorStore == nil {
		return mms.GetThreadMessages(ctx, threadID, userID, limit)
	}

	emb, err := mms.vectorStore.GenerateEmbedding(ctx, query)
	if err != nil {
		log.Printf("[MultiMessageStore] query embedding failed; fallback recent: %v", err)
		return mms.GetThreadMessages(ctx, threadID, userID, limit)
	}

	filter := map[string]any{"threadId": threadID}
	results, err := mms.vectorStore.SearchVectors(ctx, emb, filter, limit)
	if err != nil {
		log.Printf("[MultiMessageStore] vector search failed; fallback recent: %v", err)
		return mms.GetThreadMessages(ctx, threadID, userID, limit)
	}

	// Extract message IDs from vector search results
	messageIDs := make([]string, 0, len(results))
	for _, r := range results {
		messageIDs = append(messageIDs, r.MessageID)
	}

	// Batch fetch messages from Cosmos DB (more efficient than individual reads)
	fetchedMessages, err := mms.getMessagesBatch(ctx, threadID, messageIDs)
	if err != nil {
		log.Printf("[MultiMessageStore] batch fetch failed: %v", err)
		// Fallback to recent messages to avoid hard failure
		return mms.GetThreadMessages(ctx, threadID, userID, limit)
	}

	// Decrypt all messages and skip those that fail
	messages := make([]*MessageDocument, 0, len(fetchedMessages))
	for _, msg := range fetchedMessages {
		if err := mms.decryptMessageContent(msg); err != nil {
			log.Printf("[MultiMessageStore] decrypt message %s: %v (skipping)", msg.ID, err)
			continue // Skip messages that fail to decrypt
		}
		messages = append(messages, msg)
	}

	// Sort chronologically
	sort.Slice(messages, func(i, j int) bool {
		return messages[i].CreatedAt < messages[j].CreatedAt
	})

	return messages, nil
}

// getMessage retrieves a single message document from the messages container for the given thread/message ID pair.
func (mms *MultiMessageStore) getMessage(ctx context.Context, threadID, messageID string) (*MessageDocument, error) {
	pk := azcosmos.NewPartitionKeyString(threadID)
	resp, err := mms.messagesContainer.ReadItem(ctx, pk, messageID, nil)
	if err != nil {
		return nil, fmt.Errorf("read message: %w", err)
	}

	var msg MessageDocument
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
func (mms *MultiMessageStore) getMessagesBatch(ctx context.Context, threadID string, messageIDs []string) ([]*MessageDocument, error) {
	if len(messageIDs) == 0 {
		return []*MessageDocument{}, nil
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

	pager := mms.messagesContainer.NewQueryItemsPager(query, pk, opts)

	var messages []*MessageDocument
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("query messages batch: %w", err)
		}
		for _, item := range page.Items {
			var msg MessageDocument
			if err := json.Unmarshal(item, &msg); err != nil {
				log.Printf("[MultiMessageStore] unmarshal message in batch: %v", err)
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

func (mms *MultiMessageStore) generateAndStoreEmbedding(msg *MessageDocument) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Use plaintext content for embedding; decrypt on-the-fly if stored content is encrypted
	plaintext := msg.Content
	if plaintext == "" && msg.EncryptedContent != "" && mms.encryptionService != nil {
		dec, err := mms.encryptionService.DecryptMessage(&EncryptedMessage{
			EncryptedContent: msg.EncryptedContent,
			EncryptedAESKey:  msg.EncryptedAESKey,
			IV:               msg.IV,
			AuthTag:          msg.AuthTag,
			Algorithm:        msg.Algorithm,
		})
		if err != nil {
			log.Printf("[MultiMessageStore] Failed to decrypt content for embedding %s: %v", msg.ID, err)
		} else {
			plaintext = dec
		}
	}

	if strings.TrimSpace(plaintext) == "" {
		log.Printf("[MultiMessageStore] Skipping embedding for message %s: empty content", msg.ID)
		return
	}

	emb, err := mms.vectorStore.GenerateEmbedding(ctx, plaintext)
	if err != nil {
		log.Printf("[MultiMessageStore] embedding: %v", err)
		return
	}

	vecID, err := mms.vectorStore.StoreVector(ctx, &VectorDocument{
		ID:        msg.ID,
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
	})
	if err != nil {
		log.Printf("[MultiMessageStore] store vector: %v", err)
		return
	}

	// Update message with vector ID
	msg.VectorID = vecID
	msgBytes, _ := json.Marshal(msg)
	pk := azcosmos.NewPartitionKeyString(msg.ThreadID)
	if _, err := mms.messagesContainer.ReplaceItem(ctx, pk, msg.ID, msgBytes, nil); err != nil {
		log.Printf("[MultiMessageStore] update message with vector: %v", err)
	}
}

// Conversion method to maintain compatibility with existing EnhancedMessage type
func (m *MessageDocument) ToLegacyMessage() *EnhancedMessage {
	return &EnhancedMessage{
		ID:        m.ID,
		Type:      m.Type,
		ThreadID:  m.ThreadID,
		UserID:    m.UserID,
		Role:      m.Role,
		Content:   m.Content,
		Provider:  m.Provider,
		Model:     m.Model,
		CreatedAt: m.CreatedAt,
		VectorID:  m.VectorID,
	}
}

func (m *EnhancedMessage) ToMessageDocument() *MessageDocument {
	return &MessageDocument{
		ID:        m.ID,
		Type:      m.Type,
		ThreadID:  m.ThreadID,
		UserID:    m.UserID,
		Role:      m.Role,
		Content:   m.Content,
		Provider:  m.Provider,
		Model:     m.Model,
		CreatedAt: m.CreatedAt,
		VectorID:  m.VectorID,

		// Copy encryption fields
		EncryptedContent: m.EncryptedContent,
		EncryptedAESKey:  m.EncryptedAESKey,
		IV:               m.IV,
		AuthTag:          m.AuthTag,
		Algorithm:        m.Algorithm,
	}
}

// decryptMessageContent decrypts the content of a MessageDocument if it's encrypted
func (mms *MultiMessageStore) decryptMessageContent(msg *MessageDocument) error {
	if mms.encryptionService == nil {
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

	decryptedContent, err := mms.encryptionService.DecryptMessage(encryptedMsg)
	if err != nil {
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
