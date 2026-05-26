package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
	"github.com/google/uuid"
)

// ThreadDocument for the threads container
type ThreadDocument struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`     // "thread"
	ThreadID  string         `json:"threadId"` // Partition key
	UserID    string         `json:"userId"`
	Title     string         `json:"title"`
	CreatedAt int64          `json:"createdAt"`
	UpdatedAt int64          `json:"updatedAt"`
	Metadata  ThreadMetadata `json:"metadata"`
	Status    string         `json:"status"` // active | archived
}

// UserThreadIndex for quick thread listing per user
type UserThreadIndex struct {
	ID       string   `json:"id"`
	Type     string   `json:"type"`     // "user_thread_index"
	UserID   string   `json:"userId"`   // Partition key
	ThreadID string   `json:"threadId"` // Same as UserID for partition
	Threads  []string `json:"threads"`  // Ordered list of thread IDs
}

type MultiThreadStore struct {
	threadsContainer *azcosmos.ContainerClient
	usersContainer   *azcosmos.ContainerClient // For user thread index
}

func NewMultiThreadStore(multiCosmosStore *MultiCosmosStore) (*MultiThreadStore, error) {
	return &MultiThreadStore{
		threadsContainer: multiCosmosStore.GetThreadsContainer(),
		usersContainer:   multiCosmosStore.GetUsersContainer(),
	}, nil
}

func (mts *MultiThreadStore) CreateThread(ctx context.Context, userID, title string) (*ThreadDocument, error) {
	threadID := "thread_" + uuid.NewString()
	now := time.Now().UnixMilli()

	thread := &ThreadDocument{
		ID:        threadID,
		Type:      "thread",
		ThreadID:  threadID,
		UserID:    userID,
		Title:     title,
		CreatedAt: now,
		UpdatedAt: now,
		Metadata:  ThreadMetadata{MessageCount: 0},
		Status:    "active",
	}

	threadBytes, err := json.Marshal(thread)
	if err != nil {
		return nil, fmt.Errorf("marshal thread: %w", err)
	}

	pk := azcosmos.NewPartitionKeyString(threadID)
	_, err = mts.threadsContainer.CreateItem(ctx, pk, threadBytes, nil)
	if err != nil {
		return nil, fmt.Errorf("create thread: %w", err)
	}

	// Update user's thread index
	go mts.addThreadToUserIndex(context.Background(), userID, threadID)

	log.Printf("✅ Created thread in threads container: %s (%s)", threadID, title)
	return thread, nil
}

func (mts *MultiThreadStore) addThreadToUserIndex(ctx context.Context, userID, threadID string) {
	indexID := "user_thread_index_" + userID
	pk := azcosmos.NewPartitionKeyString(userID)

	// Try to read existing index
	resp, err := mts.usersContainer.ReadItem(ctx, pk, indexID, nil)
	var index UserThreadIndex

	if err != nil {
		// Create new index
		index = UserThreadIndex{
			ID:       indexID,
			Type:     "user_thread_index",
			UserID:   userID,
			ThreadID: userID, // Same as UserID for partition
			Threads:  []string{threadID},
		}
	} else {
		// Update existing index
		if err := json.Unmarshal(resp.Value, &index); err != nil {
			log.Printf("[MultiThreadStore] unmarshal user thread index: %v", err)
			return
		}

		// Add thread if not already present (prepend for latest first)
		found := false
		for _, t := range index.Threads {
			if t == threadID {
				found = true
				break
			}
		}
		if !found {
			index.Threads = append([]string{threadID}, index.Threads...)
		}
	}

	indexBytes, _ := json.Marshal(index)
	if _, err := mts.usersContainer.UpsertItem(ctx, pk, indexBytes, nil); err != nil {
		log.Printf("[MultiThreadStore] upsert user thread index: %v", err)
	}
}

func (mts *MultiThreadStore) GetThread(ctx context.Context, threadID, userID string) (*ThreadDocument, error) {
	pk := azcosmos.NewPartitionKeyString(threadID)
	resp, err := mts.threadsContainer.ReadItem(ctx, pk, threadID, nil)
	if err != nil {
		return nil, fmt.Errorf("read thread: %w", err)
	}

	var thread ThreadDocument
	if err := json.Unmarshal(resp.Value, &thread); err != nil {
		return nil, fmt.Errorf("unmarshal thread: %w", err)
	}

	if thread.Type != "thread" || thread.UserID != userID {
		return nil, fmt.Errorf("thread not found or access denied")
	}

	return &thread, nil
}

func (mts *MultiThreadStore) ListUserThreads(ctx context.Context, userID string, limit int) ([]*ThreadDocument, error) {
	indexID := "user_thread_index_" + userID
	pk := azcosmos.NewPartitionKeyString(userID)

	resp, err := mts.usersContainer.ReadItem(ctx, pk, indexID, nil)
	if err != nil {
		log.Printf("[MultiThreadStore] no thread index for user %s (new user?)", userID)
		return []*ThreadDocument{}, nil
	}

	var index UserThreadIndex
	if err := json.Unmarshal(resp.Value, &index); err != nil {
		return nil, fmt.Errorf("unmarshal user thread index: %w", err)
	}

	threads := make([]*ThreadDocument, 0, min(limit, len(index.Threads)))
	for i, threadID := range index.Threads {
		if i >= limit {
			break
		}

		thread, err := mts.GetThread(ctx, threadID, userID)
		if err != nil {
			log.Printf("[MultiThreadStore] get thread %s: %v", threadID, err)
			continue
		}
		threads = append(threads, thread)
	}

	return threads, nil
}

func (mts *MultiThreadStore) SearchUserThreads(ctx context.Context, userID, query string, limit, offset int) ([]*ThreadDocument, error) {
	indexID := "user_thread_index_" + userID
	pk := azcosmos.NewPartitionKeyString(userID)

	resp, err := mts.usersContainer.ReadItem(ctx, pk, indexID, nil)
	if err != nil {
		log.Printf("[MultiThreadStore] no thread index for user %s (new user?)", userID)
		return []*ThreadDocument{}, nil
	}

	var index UserThreadIndex
	if err := json.Unmarshal(resp.Value, &index); err != nil {
		return nil, fmt.Errorf("unmarshal user thread index: %w", err)
	}

	// Search through threads by title
	var matchingThreads []*ThreadDocument
	queryLower := strings.ToLower(query)

	for _, threadID := range index.Threads {
		thread, err := mts.GetThread(ctx, threadID, userID)
		if err != nil {
			log.Printf("[MultiThreadStore] get thread %s: %v", threadID, err)
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
		return []*ThreadDocument{}, nil
	}

	end := start + limit
	if end > len(matchingThreads) {
		end = len(matchingThreads)
	}

	return matchingThreads[start:end], nil
}

func (mts *MultiThreadStore) UpdateThread(ctx context.Context, thread *ThreadDocument) error {
	thread.UpdatedAt = time.Now().UnixMilli()

	threadBytes, err := json.Marshal(thread)
	if err != nil {
		return fmt.Errorf("marshal thread: %w", err)
	}

	pk := azcosmos.NewPartitionKeyString(thread.ThreadID)
	_, err = mts.threadsContainer.ReplaceItem(ctx, pk, thread.ID, threadBytes, nil)
	if err != nil {
		return fmt.Errorf("update thread: %w", err)
	}

	return nil
}

func (mts *MultiThreadStore) UpdateThreadMetadata(ctx context.Context, threadID, userID string, messageCount int, lastModel string) error {
	thread, err := mts.GetThread(ctx, threadID, userID)
	if err != nil {
		return err
	}

	thread.Metadata.MessageCount = messageCount
	thread.Metadata.LastModel = lastModel

	return mts.UpdateThread(ctx, thread)
}

func (mts *MultiThreadStore) RenameThread(ctx context.Context, threadID, userID, newTitle string) error {
	thread, err := mts.GetThread(ctx, threadID, userID)
	if err != nil {
		return fmt.Errorf("get thread: %w", err)
	}

	thread.Title = newTitle
	thread.UpdatedAt = time.Now().UnixMilli()

	return mts.UpdateThread(ctx, thread)
}

func (mts *MultiThreadStore) DeleteThread(ctx context.Context, threadID, userID string) error {
	// First get the thread to ensure it exists and belongs to the user
	thread, err := mts.GetThread(ctx, threadID, userID)
	if err != nil {
		return fmt.Errorf("get thread: %w", err)
	}

	// Delete the thread document
	pk := azcosmos.NewPartitionKeyString(threadID)
	_, err = mts.threadsContainer.DeleteItem(ctx, pk, thread.ID, nil)
	if err != nil {
		return fmt.Errorf("delete thread: %w", err)
	}

	// Remove thread from user index
	err = mts.removeThreadFromUserIndex(ctx, userID, threadID)
	if err != nil {
		log.Printf("Warning: failed to remove thread from user index: %v", err)
	}

	return nil
}

func (mts *MultiThreadStore) removeThreadFromUserIndex(ctx context.Context, userID, threadID string) error {
	indexID := "user_threads_" + userID
	pk := azcosmos.NewPartitionKeyString(userID)

	// Try to get existing index
	response, err := mts.usersContainer.ReadItem(ctx, pk, indexID, nil)
	if err != nil {
		// Index doesn't exist, nothing to remove
		return nil
	}

	var index UserThreadIndex
	err = json.Unmarshal(response.Value, &index)
	if err != nil {
		return fmt.Errorf("unmarshal user thread index: %w", err)
	}

	// Remove thread from list
	var newThreads []string
	for _, tid := range index.Threads {
		if tid != threadID {
			newThreads = append(newThreads, tid)
		}
	}
	index.Threads = newThreads

	// Update the index
	indexBytes, err := json.Marshal(index)
	if err != nil {
		return fmt.Errorf("marshal user thread index: %w", err)
	}

	_, err = mts.usersContainer.ReplaceItem(ctx, pk, indexID, indexBytes, nil)
	if err != nil {
		return fmt.Errorf("update user thread index: %w", err)
	}

	return nil
}

func (mts *MultiThreadStore) ArchiveThread(ctx context.Context, threadID, userID string) error {
	thread, err := mts.GetThread(ctx, threadID, userID)
	if err != nil {
		return err
	}

	thread.Status = "archived"
	return mts.UpdateThread(ctx, thread)
}

// Conversion method to maintain compatibility with existing Thread type
func (t *ThreadDocument) ToLegacyThread() *Thread {
	return &Thread{
		ID:        t.ID,
		Type:      t.Type,
		UserID:    t.UserID,
		ThreadID:  t.ThreadID,
		Title:     t.Title,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
		Metadata:  t.Metadata,
		Status:    t.Status,
	}
}
