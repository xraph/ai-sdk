package qdrant

import (
	"context"
	"fmt"
	"time"

	sdk "github.com/xraph/ai-sdk"
	logger "github.com/xraph/go-utils/log"
	"github.com/xraph/go-utils/metrics"
	
	"github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// QdrantVectorStore implements VectorStore using Qdrant's official Go SDK.
type QdrantVectorStore struct {
	client         *qdrant.Client
	collectionName string
	logger         logger.Logger
	metrics        metrics.Metrics
}

// Config configures the QdrantVectorStore.
type Config struct {
	// Required
	Host           string // Qdrant host (e.g., "localhost:6334")
	CollectionName string // Collection name
	
	// Optional
	APIKey         string        // API key for Qdrant Cloud
	UseTLS         bool          // Use TLS (default: false for local, true for cloud)
	Timeout        time.Duration // gRPC timeout (default: 30s)
	VectorSize     uint64        // Vector dimensions (created on first insert if 0)
	Distance       string        // Distance metric: "cosine", "euclidean", "dot" (default: "cosine")
	
	// Observability
	Logger         logger.Logger
	Metrics        metrics.Metrics
}

// NewQdrantVectorStore creates a new Qdrant-based vector store.
func NewQdrantVectorStore(ctx context.Context, cfg Config) (*QdrantVectorStore, error) {
	if cfg.Host == "" {
		return nil, fmt.Errorf("host is required")
	}
	if cfg.CollectionName == "" {
		return nil, fmt.Errorf("collection name is required")
	}

	// Set defaults
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.Distance == "" {
		cfg.Distance = "cosine"
	}

	// Setup gRPC connection options
	var dialOpts []grpc.DialOption
	if cfg.UseTLS {
		// Use TLS credentials
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials())) // TODO: Add proper TLS
	} else {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	// Create Qdrant client
	client, err := qdrant.NewClient(&qdrant.Config{
		Host: cfg.Host,
		Port: 0, // Port is included in Host
		APIKey: cfg.APIKey,
		UseTLS: cfg.UseTLS,
		GrpcOptions: dialOpts,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Qdrant client: %w", err)
	}

	store := &QdrantVectorStore{
		client:         client,
		collectionName: cfg.CollectionName,
		logger:         cfg.Logger,
		metrics:        cfg.Metrics,
	}

	// Ensure collection exists
	if err := store.ensureCollection(ctx, cfg.VectorSize, cfg.Distance); err != nil {
		return nil, fmt.Errorf("failed to ensure collection: %w", err)
	}

	if store.logger != nil {
		store.logger.Info("qdrant store initialized",
			logger.String("collection", cfg.CollectionName),
			logger.String("host", cfg.Host))
	}

	return store, nil
}

// ensureCollection creates the collection if it doesn't exist.
func (q *QdrantVectorStore) ensureCollection(ctx context.Context, vectorSize uint64, distance string) error {
	// Check if collection exists
	exists, err := q.client.CollectionExists(ctx, q.collectionName)
	if err != nil {
		return fmt.Errorf("failed to check collection existence: %w", err)
	}

	if exists {
		if q.logger != nil {
			q.logger.Debug("collection already exists", logger.String("collection", q.collectionName))
		}
		return nil
	}

	// Only create if vector size is specified
	if vectorSize == 0 {
		if q.logger != nil {
			q.logger.Debug("collection does not exist, will be created on first insert")
		}
		return nil
	}

	// Map distance metric
	distanceMetric := qdrant.Distance_Cosine
	switch distance {
	case "cosine":
		distanceMetric = qdrant.Distance_Cosine
	case "euclidean":
		distanceMetric = qdrant.Distance_Euclid
	case "dot":
		distanceMetric = qdrant.Distance_Dot
	default:
		return fmt.Errorf("unsupported distance metric: %s", distance)
	}

	// Create collection
	err = q.client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: q.collectionName,
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     vectorSize,
			Distance: distanceMetric,
		}),
	})
	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	if q.logger != nil {
		q.logger.Info("created collection",
			logger.String("collection", q.collectionName),
			logger.Uint64("vector_size", vectorSize))
	}

	return nil
}

// Upsert adds or updates vectors in the store.
func (q *QdrantVectorStore) Upsert(ctx context.Context, vectors []sdk.Vector) error {
	if len(vectors) == 0 {
		return nil
	}

	start := time.Now()
	defer func() {
		if q.metrics != nil {
			q.metrics.Histogram("forge.integrations.qdrant.upsert_duration").Observe(time.Since(start).Seconds())
		}
	}()

	// Convert to Qdrant format
	points := make([]*qdrant.PointStruct, len(vectors))
	for i, v := range vectors {
		if v.ID == "" {
			return fmt.Errorf("vector ID cannot be empty")
		}
		if len(v.Values) == 0 {
			return fmt.Errorf("vector values cannot be empty for ID %s", v.ID)
		}

		// Convert metadata to Qdrant payload
		payload := make(map[string]*qdrant.Value)
		for key, val := range v.Metadata {
			payload[key] = convertToQdrantValue(val)
		}

		points[i] = &qdrant.PointStruct{
			Id: &qdrant.PointId{
				PointIdOptions: &qdrant.PointId_Uuid{Uuid: v.ID},
			},
			Vectors: &qdrant.Vectors{
				VectorsOptions: &qdrant.Vectors_Vector{
					Vector: &qdrant.Vector{Data: toFloat32(v.Values)},
				},
			},
			Payload: payload,
		}
	}

	// Upsert points
	_, err := q.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: q.collectionName,
		Points:         points,
		Wait:           boolPtr(true), // Wait for operation to complete
	})
	if err != nil {
		return fmt.Errorf("upsert failed: %w", err)
	}

	if q.logger != nil {
		q.logger.Debug("upserted vectors to qdrant",
			logger.Int("count", len(vectors)),
			logger.Duration("duration", time.Since(start)))
	}

	if q.metrics != nil {
		q.metrics.Counter("forge.integrations.qdrant.upsert").Add(float64(len(vectors)))
	}

	return nil
}

// Query performs vector similarity search.
func (q *QdrantVectorStore) Query(ctx context.Context, vector []float64, limit int, filter map[string]any) ([]sdk.VectorMatch, error) {
	if len(vector) == 0 {
		return nil, fmt.Errorf("query vector cannot be empty")
	}
	if limit <= 0 {
		return nil, fmt.Errorf("limit must be positive, got %d", limit)
	}

	start := time.Now()
	defer func() {
		if q.metrics != nil {
			q.metrics.Histogram("forge.integrations.qdrant.query_duration").Observe(time.Since(start).Seconds())
		}
	}()

	// Build filter condition
	var filterCondition *qdrant.Filter
	if len(filter) > 0 {
		filterCondition = buildQdrantFilter(filter)
	}

	// Search
	searchResult, err := q.client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: q.collectionName,
		Query: &qdrant.Query{
			QueryVariant: &qdrant.Query_Nearest{
				Nearest: &qdrant.VectorInput{
					VectorInputVariant: &qdrant.VectorInput_Dense{
						Dense: &qdrant.DenseVector{Data: toFloat32(vector)},
					},
				},
			},
		},
		Filter:     filterCondition,
		Limit:      uint64Ptr(uint64(limit)),
		WithPayload: &qdrant.WithPayloadSelector{
			SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: true},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	// Convert results
	results := make([]sdk.VectorMatch, len(searchResult))
	for i, point := range searchResult {
		// Extract ID
		var id string
		if uuid := point.Id.GetUuid(); uuid != "" {
			id = uuid
		} else if num := point.Id.GetNum(); num != 0 {
			id = fmt.Sprintf("%d", num)
		}

		// Extract metadata
		metadata := make(map[string]any)
		if point.Payload != nil {
			metadata = convertFromQdrantPayload(point.Payload)
		}

		results[i] = sdk.VectorMatch{
			ID:       id,
			Score:    float64(point.Score),
			Metadata: metadata,
		}
	}

	if q.logger != nil {
		q.logger.Debug("queried qdrant",
			logger.Int("results", len(results)),
			logger.Duration("duration", time.Since(start)))
	}

	if q.metrics != nil {
		q.metrics.Counter("forge.integrations.qdrant.query").Inc()
		q.metrics.Histogram("forge.integrations.qdrant.results").Observe(float64(len(results)))
	}

	return results, nil
}

// Delete removes vectors by ID.
func (q *QdrantVectorStore) Delete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	start := time.Now()

	// Convert IDs to Qdrant format
	pointIDs := make([]*qdrant.PointId, len(ids))
	for i, id := range ids {
		pointIDs[i] = &qdrant.PointId{
			PointIdOptions: &qdrant.PointId_Uuid{Uuid: id},
		}
	}

	// Delete points
	_, err := q.client.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: q.collectionName,
		Points: &qdrant.PointsSelector{
			PointsSelectorOneOf: &qdrant.PointsSelector_Points{
				Points: &qdrant.PointsIdsList{Ids: pointIDs},
			},
		},
		Wait: boolPtr(true),
	})
	if err != nil {
		return fmt.Errorf("delete failed: %w", err)
	}

	if q.logger != nil {
		q.logger.Debug("deleted vectors from qdrant",
			logger.Int("count", len(ids)),
			logger.Duration("duration", time.Since(start)))
	}

	if q.metrics != nil {
		q.metrics.Counter("forge.integrations.qdrant.delete").Add(float64(len(ids)))
	}

	return nil
}

// Close closes the Qdrant client connection.
func (q *QdrantVectorStore) Close() error {
	if err := q.client.Close(); err != nil {
		return fmt.Errorf("failed to close client: %w", err)
	}

	if q.logger != nil {
		q.logger.Info("qdrant store closed")
	}

	return nil
}

// Count returns the number of vectors in the collection.
func (q *QdrantVectorStore) Count(ctx context.Context) (int64, error) {
	info, err := q.client.GetCollection(ctx, q.collectionName)
	if err != nil {
		return 0, fmt.Errorf("failed to get collection info: %w", err)
	}

	return int64(info.GetPointsCount()), nil
}

// Helper functions

func toFloat32(values []float64) []float32 {
	result := make([]float32, len(values))
	for i, v := range values {
		result[i] = float32(v)
	}
	return result
}

func convertToQdrantValue(val any) *qdrant.Value {
	switch v := val.(type) {
	case string:
		return &qdrant.Value{Kind: &qdrant.Value_StringValue{StringValue: v}}
	case int:
		return &qdrant.Value{Kind: &qdrant.Value_IntegerValue{IntegerValue: int64(v)}}
	case int64:
		return &qdrant.Value{Kind: &qdrant.Value_IntegerValue{IntegerValue: v}}
	case float64:
		return &qdrant.Value{Kind: &qdrant.Value_DoubleValue{DoubleValue: v}}
	case bool:
		return &qdrant.Value{Kind: &qdrant.Value_BoolValue{BoolValue: v}}
	default:
		// Convert to string as fallback
		return &qdrant.Value{Kind: &qdrant.Value_StringValue{StringValue: fmt.Sprintf("%v", v)}}
	}
}

func convertFromQdrantPayload(payload map[string]*qdrant.Value) map[string]any {
	result := make(map[string]any)
	for key, val := range payload {
		switch v := val.Kind.(type) {
		case *qdrant.Value_StringValue:
			result[key] = v.StringValue
		case *qdrant.Value_IntegerValue:
			result[key] = v.IntegerValue
		case *qdrant.Value_DoubleValue:
			result[key] = v.DoubleValue
		case *qdrant.Value_BoolValue:
			result[key] = v.BoolValue
		}
	}
	return result
}

func buildQdrantFilter(filter map[string]any) *qdrant.Filter {
	var conditions []*qdrant.Condition
	for key, val := range filter {
		conditions = append(conditions, &qdrant.Condition{
			ConditionOneOf: &qdrant.Condition_Field{
				Field: &qdrant.FieldCondition{
					Key: key,
					Match: &qdrant.Match{
						MatchValue: &qdrant.Match_Keyword{
							Keyword: fmt.Sprintf("%v", val),
						},
					},
				},
			},
		})
	}

	return &qdrant.Filter{
		Must: conditions,
	}
}

func boolPtr(b bool) *bool {
	return &b
}

func uint64Ptr(u uint64) *uint64 {
	return &u
}

