package openai

import (
	"context"
	"fmt"
	"time"

	"github.com/sashabaranov/go-openai"
	sdk "github.com/xraph/ai-sdk"
	logger "github.com/xraph/go-utils/log"
	"github.com/xraph/go-utils/metrics"
)

// OpenAIEmbeddings implements EmbeddingModel using OpenAI's embeddings API.
type OpenAIEmbeddings struct {
	client     *openai.Client
	model      string
	dimensions int
	logger     logger.Logger
	metrics    metrics.Metrics
}

// Config configures the OpenAI embeddings.
type Config struct {
	// Required
	APIKey string // OpenAI API key
	
	// Optional
	Model       string // Model name (default: "text-embedding-3-small")
	Dimensions  int    // Embedding dimensions (model-specific)
	BaseURL     string // Custom API base URL
	OrgID       string // Organization ID
	
	// Observability
	Logger  logger.Logger
	Metrics metrics.Metrics
}

// Model constants
const (
	ModelTextEmbedding3Small = "text-embedding-3-small" // 1536 dimensions, $0.02/1M tokens
	ModelTextEmbedding3Large = "text-embedding-3-large" // 3072 dimensions, $0.13/1M tokens
	ModelTextEmbeddingAda002 = "text-embedding-ada-002" // 1536 dimensions, $0.10/1M tokens
)

// NewOpenAIEmbeddings creates a new OpenAI embeddings instance.
func NewOpenAIEmbeddings(cfg Config) (*OpenAIEmbeddings, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	// Set defaults
	if cfg.Model == "" {
		cfg.Model = ModelTextEmbedding3Small
	}

	// Determine dimensions based on model
	if cfg.Dimensions == 0 {
		switch cfg.Model {
		case ModelTextEmbedding3Small, ModelTextEmbeddingAda002:
			cfg.Dimensions = 1536
		case ModelTextEmbedding3Large:
			cfg.Dimensions = 3072
		default:
			// Default for unknown models
			cfg.Dimensions = 1536
		}
	}

	// Create OpenAI client config
	clientCfg := openai.DefaultConfig(cfg.APIKey)
	if cfg.BaseURL != "" {
		clientCfg.BaseURL = cfg.BaseURL
	}
	if cfg.OrgID != "" {
		clientCfg.OrgID = cfg.OrgID
	}

	client := openai.NewClientWithConfig(clientCfg)

	embeddings := &OpenAIEmbeddings{
		client:     client,
		model:      cfg.Model,
		dimensions: cfg.Dimensions,
		logger:     cfg.Logger,
		metrics:    cfg.Metrics,
	}

	if embeddings.logger != nil {
		embeddings.logger.Info("openai embeddings initialized",
			logger.String("model", cfg.Model),
			logger.Int("dimensions", cfg.Dimensions))
	}

	return embeddings, nil
}

// Embed generates embeddings for the given texts.
func (o *OpenAIEmbeddings) Embed(ctx context.Context, texts []string) ([]sdk.Vector, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("texts cannot be empty")
	}

	start := time.Now()
	defer func() {
		if o.metrics != nil {
			o.metrics.Histogram("forge.integrations.openai.embed_duration").Observe(time.Since(start).Seconds())
		}
	}()

	// OpenAI supports batching up to 2048 texts
	const maxBatchSize = 2048
	if len(texts) > maxBatchSize {
		return o.embedBatches(ctx, texts, maxBatchSize)
	}

	// Create embedding request
	req := openai.EmbeddingRequest{
		Input: texts,
		Model: openai.EmbeddingModel(o.model),
	}

	// Add dimensions for models that support it (text-embedding-3-*)
	if o.model == ModelTextEmbedding3Small || o.model == ModelTextEmbedding3Large {
		req.Dimensions = o.dimensions
	}

	// Call OpenAI API
	resp, err := o.client.CreateEmbeddings(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create embeddings: %w", err)
	}

	// Convert to SDK format
	vectors := make([]sdk.Vector, len(resp.Data))
	for i, data := range resp.Data {
		vectors[i] = sdk.Vector{
			ID:     fmt.Sprintf("embed-%d", data.Index),
			Values: toFloat64(data.Embedding),
		}
	}

	if o.logger != nil {
		o.logger.Debug("generated embeddings",
			logger.Int("count", len(texts)),
			logger.Int("tokens", resp.Usage.TotalTokens),
			logger.Duration("duration", time.Since(start)))
	}

	if o.metrics != nil {
		o.metrics.Counter("forge.integrations.openai.embed").Add(float64(len(texts)))
		o.metrics.Counter("forge.integrations.openai.tokens").Add(float64(resp.Usage.TotalTokens))
	}

	return vectors, nil
}

// embedBatches handles large batches by splitting them.
func (o *OpenAIEmbeddings) embedBatches(ctx context.Context, texts []string, batchSize int) ([]sdk.Vector, error) {
	var allVectors []sdk.Vector

	for i := 0; i < len(texts); i += batchSize {
		end := i + batchSize
		if end > len(texts) {
			end = len(texts)
		}

		batch := texts[i:end]
		vectors, err := o.Embed(ctx, batch)
		if err != nil {
			return nil, fmt.Errorf("failed to embed batch %d-%d: %w", i, end, err)
		}

		allVectors = append(allVectors, vectors...)
	}

	return allVectors, nil
}

// Dimensions returns the embedding dimension.
func (o *OpenAIEmbeddings) Dimensions() int {
	return o.dimensions
}

// Model returns the model name.
func (o *OpenAIEmbeddings) Model() string {
	return o.model
}

// toFloat64 converts float32 slice to float64 slice.
func toFloat64(values []float32) []float64 {
	result := make([]float64, len(values))
	for i, v := range values {
		result[i] = float64(v)
	}
	return result
}

