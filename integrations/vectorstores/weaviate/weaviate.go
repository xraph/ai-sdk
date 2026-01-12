//go:build weaviate
// +build weaviate

package weaviate

import (
	"context"
	"fmt"
	"time"

	"github.com/weaviate/weaviate-go-client/v4/weaviate"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/auth"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/filters"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/graphql"
	"github.com/weaviate/weaviate/entities/models"
	"github.com/weaviate/weaviate/entities/replication"

	sdk "github.com/xraph/ai-sdk"
	logger "github.com/xraph/go-utils/log"
	"github.com/xraph/go-utils/metrics"
)

// WeaviateVectorStore implements VectorStore using Weaviate's official v4 Go SDK.
type WeaviateVectorStore struct {
	client    *weaviate.Client
	className string
	logger    logger.Logger
	metrics   metrics.Metrics
}

// Config configures the WeaviateVectorStore.
type Config struct {
	// Required
	Host      string // Weaviate host (e.g., "localhost:8080")
	ClassName string // Class name for vectors

	// Optional
	Scheme       string            // http or https (default: http)
	APIKey       string            // API key for authentication
	Headers      map[string]string // Additional headers
	Timeout      time.Duration     // Request timeout (default: 30s)
	VectorConfig *VectorConfig     // Vector configuration

	// Observability
	Logger  logger.Logger
	Metrics metrics.Metrics
}

// VectorConfig configures vector settings.
type VectorConfig struct {
	Dimensions int    // Vector dimensions
	Distance   string // Distance metric: cosine, dot, l2-squared (default: cosine)
}

// NewWeaviateVectorStore creates a new Weaviate-based vector store.
func NewWeaviateVectorStore(ctx context.Context, cfg Config) (*WeaviateVectorStore, error) {
	if cfg.Host == "" {
		return nil, fmt.Errorf("host is required")
	}
	if cfg.ClassName == "" {
		return nil, fmt.Errorf("class name is required")
	}

	// Set defaults
	if cfg.Scheme == "" {
		cfg.Scheme = "http"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	// Build client config
	clientConfig := weaviate.Config{
		Host:   cfg.Host,
		Scheme: cfg.Scheme,
	}

	// Add authentication if API key provided
	if cfg.APIKey != "" {
		clientConfig.AuthConfig = auth.ApiKey{Value: cfg.APIKey}
	}

	// Add headers
	if cfg.Headers != nil {
		clientConfig.Headers = cfg.Headers
	}

	// Create client
	client, err := weaviate.NewClient(clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Weaviate client: %w", err)
	}

	store := &WeaviateVectorStore{
		client:    client,
		className: cfg.ClassName,
		logger:    cfg.Logger,
		metrics:   cfg.Metrics,
	}

	// Ensure class exists
	if err := store.ensureClass(ctx, cfg.VectorConfig); err != nil {
		return nil, fmt.Errorf("failed to ensure class: %w", err)
	}

	if store.logger != nil {
		store.logger.Info("weaviate store initialized",
			logger.String("class", cfg.ClassName),
			logger.String("host", cfg.Host))
	}

	return store, nil
}

// ensureClass creates the class if it doesn't exist.
func (w *WeaviateVectorStore) ensureClass(ctx context.Context, vectorCfg *VectorConfig) error {
	// Check if class exists
	exists, err := w.client.Schema().ClassExistenceChecker().
		WithClassName(w.className).
		Do(ctx)
	if err != nil {
		return fmt.Errorf("failed to check class existence: %w", err)
	}

	if exists {
		if w.logger != nil {
			w.logger.Debug("class already exists", logger.String("class", w.className))
		}
		return nil
	}

	// Create class schema using raw JSON
	classSchema := map[string]interface{}{
		"class":       w.className,
		"description": "Vector store class for Forge AI SDK",
		"vectorizer":  "none", // We provide our own vectors
	}

	// Add vector configuration if provided
	if vectorCfg != nil {
		classSchema["vectorIndexConfig"] = map[string]interface{}{
			"distance": vectorCfg.Distance,
		}
	}

	// Create class
	err = w.client.Schema().ClassCreator().
		WithClass(classSchema).
		Do(ctx)
	if err != nil {
		return fmt.Errorf("failed to create class: %w", err)
	}

	if w.logger != nil {
		w.logger.Info("created class", logger.String("class", w.className))
	}

	return nil
}

// Upsert adds or updates vectors in the store.
func (w *WeaviateVectorStore) Upsert(ctx context.Context, vectors []sdk.Vector) error {
	if len(vectors) == 0 {
		return nil
	}

	start := time.Now()
	defer func() {
		if w.metrics != nil {
			w.metrics.Histogram("forge.integrations.weaviate.upsert_duration").Observe(time.Since(start).Seconds())
		}
	}()

	// Batch create objects
	batch := w.client.Batch().ObjectsBatcher()

	for _, v := range vectors {
		if v.ID == "" {
			return fmt.Errorf("vector ID cannot be empty")
		}
		if len(v.Values) == 0 {
			return fmt.Errorf("vector values cannot be empty for ID %s", v.ID)
		}

		// Create object as map
		obj := map[string]interface{}{
			"class":  w.className,
			"id":     v.ID,
			"vector": toFloat32(v.Values),
		}

		// Add properties (metadata)
		if v.Metadata != nil {
			obj["properties"] = v.Metadata
		}

		batch = batch.WithObjects(obj)
	}

	// Set consistency level
	batch = batch.WithConsistencyLevel(replication.ConsistencyLevel.QUORUM)

	// Execute batch
	result, err := batch.Do(ctx)
	if err != nil {
		return fmt.Errorf("batch upsert failed: %w", err)
	}

	// Check for errors
	if len(result) > 0 {
		for _, res := range result {
			if res.Result != nil && res.Result.Errors != nil {
				return fmt.Errorf("batch upsert error: %v", res.Result.Errors)
			}
		}
	}

	if w.logger != nil {
		w.logger.Debug("upserted vectors to weaviate",
			logger.Int("count", len(vectors)),
			logger.Duration("duration", time.Since(start)))
	}

	if w.metrics != nil {
		w.metrics.Counter("forge.integrations.weaviate.upsert").Add(float64(len(vectors)))
	}

	return nil
}

// Query performs vector similarity search.
func (w *WeaviateVectorStore) Query(ctx context.Context, vector []float64, limit int, filter map[string]any) ([]sdk.VectorMatch, error) {
	if len(vector) == 0 {
		return nil, fmt.Errorf("query vector cannot be empty")
	}
	if limit <= 0 {
		return nil, fmt.Errorf("limit must be positive, got %d", limit)
	}

	start := time.Now()
	defer func() {
		if w.metrics != nil {
			w.metrics.Histogram("forge.integrations.weaviate.query_duration").Observe(time.Since(start).Seconds())
		}
	}()

	// Build GraphQL query
	query := w.client.GraphQL().Get().
		WithClassName(w.className).
		WithNearVector(&graphql.NearVectorArgumentBuilder{}.
			WithVector(toFloat32(vector)).
			Build()).
		WithLimit(limit).
		WithFields(graphql.Field{Name: "_additional", Fields: []graphql.Field{
			{Name: "id"},
			{Name: "distance"},
		}})

	// Add filter if provided
	if len(filter) > 0 {
		whereFilter := buildWeaviateFilter(filter)
		if whereFilter != nil {
			query = query.WithWhere(whereFilter)
		}
	}

	// Execute query
	result, err := query.Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	// Parse results
	matches := []sdk.VectorMatch{}
	if result.Data != nil {
		if classData, ok := result.Data["Get"].(map[string]interface{}); ok {
			if objects, ok := classData[w.className].([]interface{}); ok {
				for _, obj := range objects {
					if objMap, ok := obj.(map[string]interface{}); ok {
						match := sdk.VectorMatch{
							Metadata: make(map[string]any),
						}

						// Extract ID and distance
						if additional, ok := objMap["_additional"].(map[string]interface{}); ok {
							if id, ok := additional["id"].(string); ok {
								match.ID = id
							}
							if distance, ok := additional["distance"].(float64); ok {
								// Convert distance to similarity score (1 - distance for cosine)
								match.Score = 1.0 - distance
							}
						}

						// Extract metadata
						for key, val := range objMap {
							if key != "_additional" {
								match.Metadata[key] = val
							}
						}

						matches = append(matches, match)
					}
				}
			}
		}
	}

	if w.logger != nil {
		w.logger.Debug("queried weaviate",
			logger.Int("results", len(matches)),
			logger.Duration("duration", time.Since(start)))
	}

	if w.metrics != nil {
		w.metrics.Counter("forge.integrations.weaviate.query").Inc()
		w.metrics.Histogram("forge.integrations.weaviate.results").Observe(float64(len(matches)))
	}

	return matches, nil
}

// Delete removes vectors by ID.
func (w *WeaviateVectorStore) Delete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	start := time.Now()

	// Delete objects one by one (batch delete not directly supported in v4)
	var lastErr error
	deleted := 0

	for _, id := range ids {
		err := w.client.Data().Deleter().
			WithClassName(w.className).
			WithID(id).
			WithConsistencyLevel(replication.ConsistencyLevel.QUORUM).
			Do(ctx)
		if err != nil {
			lastErr = err
			if w.logger != nil {
				w.logger.Warn("failed to delete object",
					logger.String("id", id),
					logger.Error(err))
			}
		} else {
			deleted++
		}
	}

	if w.logger != nil {
		w.logger.Debug("deleted vectors from weaviate",
			logger.Int("requested", len(ids)),
			logger.Int("deleted", deleted),
			logger.Duration("duration", time.Since(start)))
	}

	if w.metrics != nil {
		w.metrics.Counter("forge.integrations.weaviate.delete").Add(float64(deleted))
	}

	return lastErr
}

// Close closes the Weaviate client connection.
func (w *WeaviateVectorStore) Close() error {
	// Weaviate v4 client doesn't require explicit closing
	if w.logger != nil {
		w.logger.Info("weaviate store closed")
	}
	return nil
}

// Count returns the number of objects in the class.
func (w *WeaviateVectorStore) Count(ctx context.Context) (int64, error) {
	result, err := w.client.GraphQL().Aggregate().
		WithClassName(w.className).
		WithFields(graphql.Field{Name: "meta", Fields: []graphql.Field{
			{Name: "count"},
		}}).
		Do(ctx)
	if err != nil {
		return 0, fmt.Errorf("count failed: %w", err)
	}

	// Parse count
	if result.Data != nil {
		if aggData, ok := result.Data["Aggregate"].(map[string]interface{}); ok {
			if classAgg, ok := aggData[w.className].([]interface{}); ok && len(classAgg) > 0 {
				if meta, ok := classAgg[0].(map[string]interface{})["meta"].(map[string]interface{}); ok {
					if count, ok := meta["count"].(float64); ok {
						return int64(count), nil
					}
				}
			}
		}
	}

	return 0, nil
}

// Helper functions

func toFloat32(values []float64) []float32 {
	result := make([]float32, len(values))
	for i, v := range values {
		result[i] = float32(v)
	}
	return result
}

func buildWeaviateFilter(filter map[string]any) *filters.WhereBuilder {
	if len(filter) == 0 {
		return nil
	}

	// Build simple equality filters
	var conditions []*filters.WhereBuilder
	for key, val := range filter {
		condition := filters.Where().
			WithPath([]string{key}).
			WithOperator(filters.Equal).
			WithValueString(fmt.Sprintf("%v", val))
		conditions = append(conditions, condition)
	}

	// Combine with AND
	if len(conditions) == 1 {
		return conditions[0]
	}

	result := conditions[0]
	for i := 1; i < len(conditions); i++ {
		result = result.WithOperands([]*filters.WhereBuilder{conditions[i]})
		result = result.WithOperator(filters.And)
	}

	return result
}
