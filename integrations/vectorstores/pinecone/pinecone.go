package pinecone

import (
	"context"
	"fmt"
	"time"

	sdk "github.com/xraph/ai-sdk"
	logger "github.com/xraph/go-utils/log"
	"github.com/xraph/go-utils/metrics"

	"github.com/pinecone-io/go-pinecone/pinecone"
	"google.golang.org/protobuf/types/known/structpb"
)

// PineconeVectorStore implements VectorStore using Pinecone's official Go SDK.
type PineconeVectorStore struct {
	client    *pinecone.Client
	indexConn *pinecone.IndexConnection
	namespace string
	logger    logger.Logger
	metrics   metrics.Metrics
}

// Config configures the PineconeVectorStore.
type Config struct {
	// Required
	APIKey    string // Pinecone API key
	IndexName string // Index name (must exist)

	// Optional
	Host      string        // Index host (auto-detected if empty)
	Namespace string        // Namespace for data isolation
	Timeout   time.Duration // Request timeout (default: 30s)

	// Observability
	Logger  logger.Logger
	Metrics metrics.Metrics
}

// NewPineconeVectorStore creates a new Pinecone-based vector store.
func NewPineconeVectorStore(ctx context.Context, cfg Config) (*PineconeVectorStore, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}
	if cfg.IndexName == "" {
		return nil, fmt.Errorf("index name is required")
	}

	// Set defaults
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	// Create Pinecone client
	client, err := pinecone.NewClient(pinecone.NewClientParams{
		ApiKey: cfg.APIKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Pinecone client: %w", err)
	}

	// Connect to index
	host := cfg.Host
	if host == "" {
		// Describe index to get host
		idx, err := client.DescribeIndex(ctx, cfg.IndexName)
		if err != nil {
			return nil, fmt.Errorf("failed to describe index %s: %w", cfg.IndexName, err)
		}
		host = idx.Host
	}

	indexConn, err := client.Index(pinecone.NewIndexConnParams{
		Host:      host,
		Namespace: cfg.Namespace,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to index: %w", err)
	}

	store := &PineconeVectorStore{
		client:    client,
		indexConn: indexConn,
		namespace: cfg.Namespace,
		logger:    cfg.Logger,
		metrics:   cfg.Metrics,
	}

	if store.logger != nil {
		store.logger.Info("pinecone store initialized",
			logger.String("index", cfg.IndexName),
			logger.String("namespace", cfg.Namespace))
	}

	return store, nil
}

// Upsert adds or updates vectors in the store.
func (p *PineconeVectorStore) Upsert(ctx context.Context, vectors []sdk.Vector) error {
	if len(vectors) == 0 {
		return nil
	}

	start := time.Now()
	defer func() {
		if p.metrics != nil {
			p.metrics.Histogram("forge.integrations.pinecone.upsert_duration").Observe(time.Since(start).Seconds())
		}
	}()

	// Convert to Pinecone format
	pineconeVectors := make([]*pinecone.Vector, len(vectors))
	for i, v := range vectors {
		if v.ID == "" {
			return fmt.Errorf("vector ID cannot be empty")
		}
		if len(v.Values) == 0 {
			return fmt.Errorf("vector values cannot be empty for ID %s", v.ID)
		}

		// Convert metadata to structpb.Struct
		var metadata *pinecone.Metadata
		if v.Metadata != nil {
			metadataStruct, err := structpb.NewStruct(v.Metadata)
			if err != nil {
				return fmt.Errorf("failed to convert metadata for vector %s: %w", v.ID, err)
			}
			metadata = metadataStruct
		}

		pineconeVectors[i] = &pinecone.Vector{
			Id:       v.ID,
			Values:   toFloat32(v.Values),
			Metadata: metadata,
		}
	}

	// Upsert vectors
	_, err := p.indexConn.UpsertVectors(ctx, pineconeVectors)
	if err != nil {
		return fmt.Errorf("upsert failed: %w", err)
	}

	if p.logger != nil {
		p.logger.Debug("upserted vectors to pinecone",
			logger.Int("count", len(vectors)),
			logger.Duration("duration", time.Since(start)))
	}

	if p.metrics != nil {
		p.metrics.Counter("forge.integrations.pinecone.upsert").Add(float64(len(vectors)))
	}

	return nil
}

// Query performs vector similarity search.
func (p *PineconeVectorStore) Query(ctx context.Context, vector []float64, limit int, filter map[string]any) ([]sdk.VectorMatch, error) {
	if len(vector) == 0 {
		return nil, fmt.Errorf("query vector cannot be empty")
	}
	if limit <= 0 {
		return nil, fmt.Errorf("limit must be positive, got %d", limit)
	}

	start := time.Now()
	defer func() {
		if p.metrics != nil {
			p.metrics.Histogram("forge.integrations.pinecone.query_duration").Observe(time.Since(start).Seconds())
		}
	}()

	// Build query request
	queryReq := &pinecone.QueryByVectorValuesRequest{
		Vector:          toFloat32(vector),
		TopK:            uint32(limit),
		IncludeMetadata: true,
	}

	// Add filter if provided
	if len(filter) > 0 {
		pineconeFilter := convertToPineconeFilter(filter)
		queryReq.MetadataFilter = pineconeFilter
	}

	// Execute query
	response, err := p.indexConn.QueryByVectorValues(ctx, queryReq)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	// Convert results
	results := make([]sdk.VectorMatch, len(response.Matches))
	for i, match := range response.Matches {
		metadata := make(map[string]any)
		if match.Vector.Metadata != nil {
			// Convert structpb.Struct to map[string]any
			metadata = match.Vector.Metadata.AsMap()
		}

		results[i] = sdk.VectorMatch{
			ID:       match.Vector.Id,
			Score:    float64(match.Score),
			Metadata: metadata,
		}
	}

	if p.logger != nil {
		p.logger.Debug("queried pinecone",
			logger.Int("results", len(results)),
			logger.Duration("duration", time.Since(start)))
	}

	if p.metrics != nil {
		p.metrics.Counter("forge.integrations.pinecone.query").Inc()
		p.metrics.Histogram("forge.integrations.pinecone.results").Observe(float64(len(results)))
	}

	return results, nil
}

// Delete removes vectors by ID.
func (p *PineconeVectorStore) Delete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	start := time.Now()

	// Delete vectors
	err := p.indexConn.DeleteVectorsById(ctx, ids)
	if err != nil {
		return fmt.Errorf("delete failed: %w", err)
	}

	if p.logger != nil {
		p.logger.Debug("deleted vectors from pinecone",
			logger.Int("count", len(ids)),
			logger.Duration("duration", time.Since(start)))
	}

	if p.metrics != nil {
		p.metrics.Counter("forge.integrations.pinecone.delete").Add(float64(len(ids)))
	}

	return nil
}

// Close closes the Pinecone client.
func (p *PineconeVectorStore) Close() error {
	// Pinecone client doesn't require explicit closing
	if p.logger != nil {
		p.logger.Info("pinecone store closed")
	}
	return nil
}

// Stats returns index statistics.
func (p *PineconeVectorStore) Stats(ctx context.Context) (*IndexStats, error) {
	stats, err := p.indexConn.DescribeIndexStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	result := &IndexStats{
		TotalVectorCount: int64(stats.TotalVectorCount),
		Dimension:        int(stats.Dimension),
	}

	if stats.Namespaces != nil {
		result.Namespaces = make(map[string]int64)
		for ns, nsStats := range stats.Namespaces {
			result.Namespaces[ns] = int64(nsStats.VectorCount)
		}
	}

	return result, nil
}

// IndexStats represents Pinecone index statistics.
type IndexStats struct {
	TotalVectorCount int64
	Dimension        int
	Namespaces       map[string]int64
}

// Helper functions

func toFloat32(values []float64) []float32 {
	result := make([]float32, len(values))
	for i, v := range values {
		result[i] = float32(v)
	}
	return result
}

func convertToPineconeFilter(filter map[string]any) *pinecone.MetadataFilter {
	// Convert map to structpb.Struct
	metadataFilter, err := structpb.NewStruct(filter)
	if err != nil {
		// Return nil if conversion fails
		return nil
	}
	return metadataFilter
}
