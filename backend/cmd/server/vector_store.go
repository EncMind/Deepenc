package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type VectorDocument struct {
	ID        string                 `json:"id"`
	ThreadID  string                 `json:"threadId"`
	MessageID string                 `json:"messageId"`
	UserID    string                 `json:"userId"`
	Content   string                 `json:"content"`
	Embedding []float32              `json:"embedding"`
	Metadata  map[string]interface{} `json:"metadata"`
}

type SearchResult struct {
	MessageID string  `json:"messageId"`
	ThreadID  string  `json:"threadId"`
	UserID    string  `json:"userId"`
	Role      string  `json:"role"`
	Content   string  `json:"content"`
	Timestamp int64   `json:"timestamp"`
	Score     float32 `json:"score"`
}
type VectorStore struct {
	qdrantURL      string
	openAIKey      string
	embeddingModel string
	httpClient     *http.Client
	scoreThreshold float32
	collectionName string
	embeddingDim   int
}

func NewVectorStore() (*VectorStore, error) {
	qdrantURL := os.Getenv("QDRANT_URL")
	if qdrantURL == "" {
		qdrantURL = "http://localhost:6333"
	}
	openAIKey := os.Getenv("OPENAI_API_KEY")
	if openAIKey == "" {
		log.Printf("[VectorStore] Warning: OPENAI_API_KEY not set; embeddings disabled")
	}
	model := os.Getenv("EMBEDDING_MODEL")
	if model == "" {
		model = "text-embedding-3-small"
	}
	dim := 1536
	if model == "text-embedding-3-large" {
		dim = 3072
	}
	th := float32(0.0)
	if v := stringsTrim(os.Getenv("QDRANT_SCORE_THRESHOLD")); v != "" {
		if f, err := strconv.ParseFloat(v, 32); err == nil {
			th = float32(f)
		}
	}

	vs := &VectorStore{
		qdrantURL:      qdrantURL,
		openAIKey:      openAIKey,
		embeddingModel: model,
		httpClient:     &http.Client{Timeout: 30 * time.Second},
		scoreThreshold: th,
		collectionName: "conversations",
		embeddingDim:   dim,
	}
	if err := vs.initializeCollection(context.Background()); err != nil {
		log.Printf("[VectorStore] Warning: init collection: %v", err)
	}
	log.Printf("[VectorStore] Qdrant=%s collection=%s dim=%d threshold=%.3f", qdrantURL, vs.collectionName, dim, th)
	return vs, nil
}

func (vs *VectorStore) initializeCollection(ctx context.Context) error {
	checkURL := fmt.Sprintf("%s/collections/%s", vs.qdrantURL, vs.collectionName)
	req, _ := http.NewRequestWithContext(ctx, "GET", checkURL, nil)
	resp, err := vs.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("check collection: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == 200 {
		return nil
	}

	createBody := map[string]interface{}{
		"vectors": map[string]interface{}{
			"size":     vs.embeddingDim,
			"distance": "Cosine",
		},
	}
	b, _ := json.Marshal(createBody)
	createURL := fmt.Sprintf("%s/collections/%s", vs.qdrantURL, vs.collectionName)
	req, _ = http.NewRequestWithContext(ctx, "PUT", createURL, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, err = vs.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("create collection: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("create collection failed: %s", string(body))
	}
	return nil
}

func (vs *VectorStore) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	if vs.openAIKey == "" {
		return nil, fmt.Errorf("OpenAI API key not configured")
	}
	reqBody := map[string]interface{}{
		"model": vs.embeddingModel,
		"input": text,
	}
	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+vs.openAIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := vs.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai error: %s", string(b))
	}
	var result struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if len(result.Data) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}
	return result.Data[0].Embedding, nil
}

func (vs *VectorStore) StoreVector(ctx context.Context, doc *VectorDocument) (string, error) {
	log.Printf("[Qdrant] Storing vector with ID: %s (embedding dim: %d)", doc.ID, len(doc.Embedding))

	point := map[string]interface{}{
		"id":     doc.ID,
		"vector": doc.Embedding,
		"payload": map[string]interface{}{
			"threadId":  doc.ThreadID,
			"messageId": doc.MessageID,
			"userId":    doc.UserID,
		},
	}
	for k, v := range doc.Metadata {
		point["payload"].(map[string]interface{})[k] = v
	}
	upsert := map[string]interface{}{"points": []interface{}{point}}
	body, _ := json.Marshal(upsert)

	url := fmt.Sprintf("%s/collections/%s/points", vs.qdrantURL, vs.collectionName)
	log.Printf("[Qdrant] Sending upsert request to: %s", url)

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(body))
	if err != nil {
		log.Printf("[Qdrant] Failed to create request: %v", err)
		return "", fmt.Errorf("qdrant create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := vs.httpClient.Do(req)
	if err != nil {
		log.Printf("[Qdrant] Request failed: %v", err)
		return "", fmt.Errorf("qdrant request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		log.Printf("[Qdrant] Error response (%d): %s", resp.StatusCode, string(b))
		return "", fmt.Errorf("qdrant error: %s", string(b))
	}

	log.Printf("[Qdrant] ✅ Successfully stored vector: %s", doc.ID)
	return doc.ID, nil
}

func (vs *VectorStore) SearchVectors(ctx context.Context, embedding []float32, filter map[string]interface{}, limit int) ([]*SearchResult, error) {
	search := map[string]interface{}{
		"vector":          embedding,
		"limit":           limit,
		"with_payload":    true,
		"with_vector":     false,
		"score_threshold": vs.scoreThreshold,
	}
	if len(filter) > 0 {
		must := make([]interface{}, 0, len(filter))
		for k, v := range filter {
			must = append(must, map[string]interface{}{
				"key":   k,
				"match": map[string]interface{}{"value": v},
			})
		}
		search["filter"] = map[string]interface{}{"must": must}
	}
	body, _ := json.Marshal(search)

	url := fmt.Sprintf("%s/collections/%s/points/search", vs.qdrantURL, vs.collectionName)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("qdrant search request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := vs.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("qdrant request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("qdrant error: %s", string(b))
	}

	var sr struct {
		Result []struct {
			ID      any                    `json:"id"`
			Score   float32                `json:"score"`
			Payload map[string]interface{} `json:"payload"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return nil, fmt.Errorf("decode search: %w", err)
	}

	results := make([]*SearchResult, 0, len(sr.Result))
	for _, r := range sr.Result {
		p := r.Payload
		results = append(results, &SearchResult{
			MessageID: asString(p["messageId"]),
			ThreadID:  asString(p["threadId"]),
			UserID:    asString(p["userId"]),
			Role:      asString(p["role"]),
			Content:   "",
			Timestamp: asInt64(p["timestamp"]),
			Score:     r.Score,
		})
	}
	return results, nil
}

func (vs *VectorStore) HealthCheck(ctx context.Context) error {
	url := fmt.Sprintf("%s/", vs.qdrantURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	resp, err := vs.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("qdrant request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("qdrant unhealthy: status %d", resp.StatusCode)
	}
	return nil
}

func stringsTrim(s string) string { return strings.TrimSpace(s) }

func asString(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}

func asInt64(v any) int64 {
	switch t := v.(type) {
	case float64:
		return int64(t)
	case int64:
		return t
	case int:
		return int64(t)
	case json.Number:
		x, _ := t.Int64()
		return x
	default:
		return 0
	}
}
