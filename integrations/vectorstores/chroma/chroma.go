package chroma

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	sdk "github.com/xraph/ai-sdk"
	logger "github.com/xraph/go-utils/log"
	"github.com/xraph/go-utils/metrics"
)

// ChromaVectorStore implements the sdk.VectorStore interface using ChromaDB's REST API.
type ChromaVectorStore struct {
	httpClient     *http.Client
	baseURL        string
	collectionName string
	logger         logger.Logger
	metrics        metrics.Metrics
}

// Config provides configuration for the ChromaVectorStore.
type Config struct {
	BaseURL        string        // Required: ChromaDB base URL (e.g., "http://localhost:8000")
	CollectionName string        // Required: Name of the collection
	APIKey         string        // Optional: API key for authentication
	Timeout        time.Duration // Optional: Request timeout (default: 30s)
	Logger         logger.Logger
	Metrics        metrics.Metrics
}

// NewChromaVectorStore creates a new ChromaDB-based vector store.
func NewChromaVectorStore(ctx context.Context, cfg Config) (*ChromaVectorStore, error) {
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("base URL is required")
	}
	if cfg.CollectionName == "" {
		return nil, fmt.Errorf("collection name is required")
	}

	// Set defaults
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	// Create HTTP client
	client := &http.Client{
		Timeout: cfg.Timeout,
	}

	store := &ChromaVectorStore{
		httpClient:     client,
		baseURL:        strings.TrimSuffix(cfg.BaseURL, "/"),
		collectionName: cfg.CollectionName,
		logger:         cfg.Logger,
		metrics:        cfg.Metrics,
	}

	// Ensure collection exists
	if err := store.ensureCollection(ctx); err != nil {
		return nil, fmt.Errorf("failed to ensure collection: %w", err)
	}

	if store.logger != nil {
		store.logger.Info("chroma store initialized",
			logger.String("collection", cfg.CollectionName),
			logger.String("base_url", cfg.BaseURL))
	}

	return store, nil
}

// ensureCollection creates the collection if it doesn't exist.
func (c *ChromaVectorStore) ensureCollection(ctx context.Context) error {
	// Try to get collection first
	url := fmt.Sprintf("%s/api/v1/collections/%s", c.baseURL, c.collectionName)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to get collection: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		if c.logger != nil {
			c.logger.Debug("collection already exists", logger.String("collection", c.collectionName))
		}
		return nil
	}

	// Collection doesn't exist, create it
	createURL := fmt.Sprintf("%s/api/v1/collections", c.baseURL)
	payload := map[string]interface{}{
		"name": c.collectionName,
		"metadata": map[string]string{
			"description": "Forge AI SDK vector store",
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal create request: %w", err)
	}

	req, err = http.NewRequestWithContext(ctx, "POST", createURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err = c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create collection: status %d, body: %s", resp.StatusCode, body)
	}

	if c.logger != nil {
		c.logger.Info("created collection", logger.String("collection", c.collectionName))
	}

	return nil
}

// Upsert adds or updates vectors in the store.
func (c *ChromaVectorStore) Upsert(ctx context.Context, vectors []sdk.Vector) error {
	if len(vectors) == 0 {
		return nil
	}

	start := time.Now()
	defer func() {
		if c.metrics != nil {
			c.metrics.Histogram("forge.integrations.chroma.upsert_duration").Observe(time.Since(start).Seconds())
		}
	}()

	// Prepare batch request
	ids := make([]string, len(vectors))
	embeddings := make([][]float64, len(vectors))
	metadatas := make([]map[string]any, len(vectors))

	for i, v := range vectors {
		if v.ID == "" {
			return fmt.Errorf("vector ID cannot be empty at index %d", i)
		}
		if len(v.Values) == 0 {
			return fmt.Errorf("vector values cannot be empty for ID %s", v.ID)
		}

		ids[i] = v.ID
		embeddings[i] = v.Values
		if v.Metadata != nil {
			metadatas[i] = v.Metadata
		} else {
			metadatas[i] = make(map[string]any)
		}
	}

	payload := map[string]interface{}{
		"ids":        ids,
		"embeddings": embeddings,
		"metadatas":  metadatas,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal upsert request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/collections/%s/add", c.baseURL, c.collectionName)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("upsert request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upsert failed: status %d, body: %s", resp.StatusCode, respBody)
	}

	if c.logger != nil {
		c.logger.Debug("upserted vectors to chroma",
			logger.Int("count", len(vectors)),
			logger.Duration("duration", time.Since(start)))
	}

	if c.metrics != nil {
		c.metrics.Counter("forge.integrations.chroma.upsert").Add(float64(len(vectors)))
	}

	return nil
}

// Query performs vector similarity search.
func (c *ChromaVectorStore) Query(ctx context.Context, vector []float64, limit int, filter map[string]any) ([]sdk.VectorMatch, error) {
	if len(vector) == 0 {
		return nil, fmt.Errorf("query vector cannot be empty")
	}
	if limit <= 0 {
		return nil, fmt.Errorf("limit must be positive, got %d", limit)
	}

	start := time.Now()
	defer func() {
		if c.metrics != nil {
			c.metrics.Histogram("forge.integrations.chroma.query_duration").Observe(time.Since(start).Seconds())
		}
	}()

	payload := map[string]interface{}{
		"query_embeddings": [][]float64{vector},
		"n_results":        limit,
	}

	// Add filter if provided
	if len(filter) > 0 {
		payload["where"] = filter
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/collections/%s/query", c.baseURL, c.collectionName)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("query request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("query failed: status %d, body: %s", resp.StatusCode, respBody)
	}

	// Parse response
	var result chromaQueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to SDK format
	matches := []sdk.VectorMatch{}
	if len(result.IDs) > 0 && len(result.IDs[0]) > 0 {
		for i := 0; i < len(result.IDs[0]); i++ {
			match := sdk.VectorMatch{
				ID:       result.IDs[0][i],
				Metadata: make(map[string]any),
			}

			// Calculate similarity score from distance (Chroma returns distance, not similarity)
			if len(result.Distances) > 0 && len(result.Distances[0]) > i {
				// Convert L2 distance to similarity score (1 / (1 + distance))
				distance := result.Distances[0][i]
				match.Score = 1.0 / (1.0 + distance)
			}

			// Add metadata
			if len(result.Metadatas) > 0 && len(result.Metadatas[0]) > i && result.Metadatas[0][i] != nil {
				match.Metadata = result.Metadatas[0][i]
			}

			matches = append(matches, match)
		}
	}

	if c.logger != nil {
		c.logger.Debug("queried chroma",
			logger.Int("results", len(matches)),
			logger.Duration("duration", time.Since(start)))
	}

	if c.metrics != nil {
		c.metrics.Counter("forge.integrations.chroma.query").Inc()
		c.metrics.Histogram("forge.integrations.chroma.results").Observe(float64(len(matches)))
	}

	return matches, nil
}

// Delete removes vectors by ID.
func (c *ChromaVectorStore) Delete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	start := time.Now()

	payload := map[string]interface{}{
		"ids": ids,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal delete request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/collections/%s/delete", c.baseURL, c.collectionName)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("delete request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete failed: status %d, body: %s", resp.StatusCode, respBody)
	}

	if c.logger != nil {
		c.logger.Debug("deleted vectors from chroma",
			logger.Int("count", len(ids)),
			logger.Duration("duration", time.Since(start)))
	}

	if c.metrics != nil {
		c.metrics.Counter("forge.integrations.chroma.delete").Add(float64(len(ids)))
	}

	return nil
}

// Close closes any resources held by the store.
func (c *ChromaVectorStore) Close() error {
	if c.logger != nil {
		c.logger.Info("chroma store closed")
	}
	return nil
}

// chromaQueryResponse represents the response from ChromaDB query API.
type chromaQueryResponse struct {
	IDs       [][]string         `json:"ids"`
	Distances [][]float64        `json:"distances"`
	Metadatas [][]map[string]any `json:"metadatas"`
	Documents [][]string         `json:"documents,omitempty"`
}
