package memory

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"

	sdk "github.com/xraph/ai-sdk"
	logger "github.com/xraph/go-utils/log"
	"github.com/xraph/go-utils/metrics"
)

// MemoryVectorStore is an in-memory implementation of VectorStore for testing and development.
// It stores vectors in memory and performs cosine similarity search.
// NOT recommended for production use - data is lost on restart.
type MemoryVectorStore struct {
	vectors map[string]sdk.Vector
	mu      sync.RWMutex
	logger  logger.Logger
	metrics metrics.Metrics
}

// Config configures the MemoryVectorStore.
type Config struct {
	Logger  logger.Logger   // Optional: for debugging
	Metrics metrics.Metrics // Optional: for monitoring
}

// NewMemoryVectorStore creates a new in-memory vector store.
func NewMemoryVectorStore(cfg Config) *MemoryVectorStore {
	return &MemoryVectorStore{
		vectors: make(map[string]sdk.Vector),
		logger:  cfg.Logger,
		metrics: cfg.Metrics,
	}
}

// Upsert adds or updates vectors in the store.
func (m *MemoryVectorStore) Upsert(ctx context.Context, vectors []sdk.Vector) error {
	if len(vectors) == 0 {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, v := range vectors {
		if v.ID == "" {
			return fmt.Errorf("vector ID cannot be empty")
		}
		if len(v.Values) == 0 {
			return fmt.Errorf("vector values cannot be empty for ID %s", v.ID)
		}

		// Deep copy metadata to avoid mutation
		metadata := make(map[string]any)
		for k, val := range v.Metadata {
			metadata[k] = val
		}

		// Deep copy values
		values := make([]float64, len(v.Values))
		copy(values, v.Values)

		m.vectors[v.ID] = sdk.Vector{
			ID:       v.ID,
			Values:   values,
			Metadata: metadata,
		}
	}

	if m.logger != nil {
		m.logger.Debug("upserted vectors to memory store", logger.Int("count", len(vectors)))
	}

	if m.metrics != nil {
		m.metrics.Counter("forge.integrations.memory.upsert").Add(float64(len(vectors)))
		m.metrics.Gauge("forge.integrations.memory.total_vectors").Set(float64(len(m.vectors)))
	}

	return nil
}

// Query performs cosine similarity search and returns the top K matches.
func (m *MemoryVectorStore) Query(ctx context.Context, vector []float64, limit int, filter map[string]any) ([]sdk.VectorMatch, error) {
	if len(vector) == 0 {
		return nil, fmt.Errorf("query vector cannot be empty")
	}
	if limit <= 0 {
		return nil, fmt.Errorf("limit must be positive, got %d", limit)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.vectors) == 0 {
		return []sdk.VectorMatch{}, nil
	}

	// Calculate similarities
	type scoredMatch struct {
		id       string
		score    float64
		metadata map[string]any
	}

	matches := make([]scoredMatch, 0, len(m.vectors))

	for _, v := range m.vectors {
		// Check if vector passes filter
		if filter != nil && !matchesFilter(v.Metadata, filter) {
			continue
		}

		// Calculate cosine similarity
		score, err := cosineSimilarity(vector, v.Values)
		if err != nil {
			// Skip vectors with incompatible dimensions
			if m.logger != nil {
				m.logger.Warn("skipping vector with incompatible dimensions",
					logger.String("id", v.ID),
					logger.Int("expected", len(vector)),
					logger.Int("got", len(v.Values)))
			}
			continue
		}

		matches = append(matches, scoredMatch{
			id:       v.ID,
			score:    score,
			metadata: v.Metadata,
		})
	}

	// Sort by score descending
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].score > matches[j].score
	})

	// Limit results
	if limit > len(matches) {
		limit = len(matches)
	}
	matches = matches[:limit]

	// Convert to VectorMatch
	results := make([]sdk.VectorMatch, len(matches))
	for i, m := range matches {
		results[i] = sdk.VectorMatch{
			ID:       m.id,
			Score:    m.score,
			Metadata: m.metadata,
		}
	}

	if m.logger != nil {
		m.logger.Debug("queried memory store",
			logger.Int("total_vectors", len(m.vectors)),
			logger.Int("matches", len(results)))
	}

	if m.metrics != nil {
		m.metrics.Counter("forge.integrations.memory.query").Inc()
		m.metrics.Histogram("forge.integrations.memory.results").Observe(float64(len(results)))
	}

	return results, nil
}

// Delete removes vectors by ID.
func (m *MemoryVectorStore) Delete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	deleted := 0
	for _, id := range ids {
		if _, exists := m.vectors[id]; exists {
			delete(m.vectors, id)
			deleted++
		}
	}

	if m.logger != nil {
		m.logger.Debug("deleted vectors from memory store",
			logger.Int("requested", len(ids)),
			logger.Int("deleted", deleted))
	}

	if m.metrics != nil {
		m.metrics.Counter("forge.integrations.memory.delete").Add(float64(deleted))
		m.metrics.Gauge("forge.integrations.memory.total_vectors").Set(float64(len(m.vectors)))
	}

	return nil
}

// Clear removes all vectors from the store.
func (m *MemoryVectorStore) Clear(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	count := len(m.vectors)
	m.vectors = make(map[string]sdk.Vector)

	if m.logger != nil {
		m.logger.Debug("cleared memory store", logger.Int("count", count))
	}

	if m.metrics != nil {
		m.metrics.Gauge("forge.integrations.memory.total_vectors").Set(0)
	}

	return nil
}

// Count returns the number of vectors in the store.
func (m *MemoryVectorStore) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.vectors)
}

// cosineSimilarity computes the cosine similarity between two vectors.
// Returns a value between -1 and 1, where 1 means identical direction.
func cosineSimilarity(a, b []float64) (float64, error) {
	if len(a) != len(b) {
		return 0, fmt.Errorf("vector dimensions mismatch: %d != %d", len(a), len(b))
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	normA = math.Sqrt(normA)
	normB = math.Sqrt(normB)

	if normA == 0 || normB == 0 {
		return 0, fmt.Errorf("zero vector encountered")
	}

	return dotProduct / (normA * normB), nil
}

// matchesFilter checks if metadata matches all filter conditions.
// Supports exact match for strings and numbers.
func matchesFilter(metadata map[string]any, filter map[string]any) bool {
	for key, filterValue := range filter {
		metaValue, exists := metadata[key]
		if !exists {
			return false
		}

		// Simple equality check
		if metaValue != filterValue {
			return false
		}
	}
	return true
}
