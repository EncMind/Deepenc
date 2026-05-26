package main

import (
	"context"
	"fmt"
	"log"
)

// MultiThreadManager manages threads and messages using separate containers
type MultiThreadManager struct {
	multiCosmosStore *MultiCosmosStore
	threadStore      *MultiThreadStore
	messageStore     *MultiMessageStore
	vectorStore      *VectorStore
}

func NewMultiThreadManager(multiCosmosStore *MultiCosmosStore, vectorStore *VectorStore, encSvc *TEEEncryptionService) (*MultiThreadManager, error) {
	threadStore, err := NewMultiThreadStore(multiCosmosStore)
	if err != nil {
		return nil, err
	}

	messageStore, err := NewMultiMessageStore(multiCosmosStore, threadStore, vectorStore, encSvc)
	if err != nil {
		return nil, err
	}

	log.Printf("[MultiThreadManager] Initialized with separate containers for threads and messages")
	return &MultiThreadManager{
		multiCosmosStore: multiCosmosStore,
		threadStore:      threadStore,
		messageStore:     messageStore,
		vectorStore:      vectorStore,
	}, nil
}

// Thread operations
func (mtm *MultiThreadManager) CreateThread(ctx context.Context, userID, title string) (*Thread, error) {
	threadDoc, err := mtm.threadStore.CreateThread(ctx, userID, title)
	if err != nil {
		return nil, err
	}
	return threadDoc.ToLegacyThread(), nil
}

func (mtm *MultiThreadManager) GetThread(ctx context.Context, threadID, userID string) (*Thread, error) {
	threadDoc, err := mtm.threadStore.GetThread(ctx, threadID, userID)
	if err != nil {
		return nil, err
	}
	return threadDoc.ToLegacyThread(), nil
}

func (mtm *MultiThreadManager) ListUserThreads(ctx context.Context, userID string, limit int) ([]*Thread, error) {
	threadDocs, err := mtm.threadStore.ListUserThreads(ctx, userID, limit)
	if err != nil {
		return nil, err
	}

	threads := make([]*Thread, 0, len(threadDocs))
	for _, doc := range threadDocs {
		threads = append(threads, doc.ToLegacyThread())
	}
	return threads, nil
}

func (mtm *MultiThreadManager) SearchUserThreads(ctx context.Context, userID, query string, limit, offset int) ([]*Thread, error) {
	threadDocs, err := mtm.threadStore.SearchUserThreads(ctx, userID, query, limit, offset)
	if err != nil {
		return nil, err
	}

	threads := make([]*Thread, 0, len(threadDocs))
	for _, doc := range threadDocs {
		threads = append(threads, doc.ToLegacyThread())
	}
	return threads, nil
}

// Message operations
func (mtm *MultiThreadManager) SaveMessage(ctx context.Context, userID string, msg *EnhancedMessage) error {
	if msg == nil {
		return fmt.Errorf("message is required")
	}

	if _, err := mtm.threadStore.GetThread(ctx, msg.ThreadID, userID); err != nil {
		return err
	}

	if msg.UserID != "" && msg.UserID != userID {
		return fmt.Errorf("message user mismatch")
	}
	msg.UserID = userID

	msgDoc := msg.ToMessageDocument()
	return mtm.messageStore.SaveMessage(ctx, userID, msgDoc)
}

func (mtm *MultiThreadManager) GetThreadMessages(ctx context.Context, threadID, userID string, limit int) ([]*EnhancedMessage, error) {
	if _, err := mtm.threadStore.GetThread(ctx, threadID, userID); err != nil {
		return nil, err
	}

	msgDocs, err := mtm.messageStore.GetThreadMessages(ctx, threadID, userID, limit)
	if err != nil {
		return nil, err
	}

	messages := make([]*EnhancedMessage, 0, len(msgDocs))
	for _, doc := range msgDocs {
		messages = append(messages, doc.ToLegacyMessage())
	}
	return messages, nil
}

func (mtm *MultiThreadManager) SearchThreadMessages(ctx context.Context, threadID, userID, query string, limit int) ([]*EnhancedMessage, error) {
	if _, err := mtm.threadStore.GetThread(ctx, threadID, userID); err != nil {
		return nil, err
	}

	msgDocs, err := mtm.messageStore.SearchThreadMessages(ctx, threadID, userID, query, limit)
	if err != nil {
		return nil, err
	}

	messages := make([]*EnhancedMessage, 0, len(msgDocs))
	for _, doc := range msgDocs {
		messages = append(messages, doc.ToLegacyMessage())
	}
	return messages, nil
}

// Archive thread
func (mtm *MultiThreadManager) ArchiveThread(ctx context.Context, threadID, userID string) error {
	return mtm.threadStore.ArchiveThread(ctx, threadID, userID)
}

// Get message count for a thread
func (mtm *MultiThreadManager) GetThreadMessageCount(ctx context.Context, threadID string) (int, error) {
	return mtm.messageStore.GetThreadMessageCount(ctx, threadID)
}

func (mtm *MultiThreadManager) RenameThread(ctx context.Context, threadID, userID, newTitle string) error {
	return mtm.threadStore.RenameThread(ctx, threadID, userID, newTitle)
}

func (mtm *MultiThreadManager) DeleteThread(ctx context.Context, threadID, userID string) error {
	return mtm.threadStore.DeleteThread(ctx, threadID, userID)
}
