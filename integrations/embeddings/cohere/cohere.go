package cohere

import (
	"context"
	"fmt"
	"time"

	coherego "github.com/cohere-ai/cohere-go/v2"
	cohereclient "github.com/cohere-ai/cohere-go/v2/client"
	sdk "github.com/xraph/ai-sdk"
	logger "github.com/xraph/go-utils/log"
	"github.com/xraph/go-utils/metrics"
)

// CohereEmbeddings implements the sdk.EmbeddingModel interface using Cohere's official Go SDK.
type CohereEmbeddings struct {
	client     *cohereclient.Client
	model      string
	dimensions int
	logger     logger.Logger
	metrics    metrics.Metrics
}

// Config provides configuration for CohereEmbeddings.
type Config struct {
	APIKey  string        // Required: Cohere API key
	Model   string        // Required: Model name (e.g., "embed-english-v3.0", "embed-multilingual-v3.0")
	Timeout time.Duration // Optional: Request timeout (default: 30s)
	Logger  logger.Logger
	Metrics metrics.Metrics
}

// NewCohereEmbeddings creates a new Cohere embeddings instance.
func NewCohereEmbeddings(cfg Config) (*CohereEmbeddings, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API key cannot be empty")
	}
	if cfg.Model == "" {
		return nil, fmt.Errorf("model name cannot be empty")
	}

	// Set defaults
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	// Create Cohere client
	client := cohereclient.NewClient(cohereclient.WithToken(cfg.APIKey))

	// Get dimensions for the model
	dims, err := getModelDimensions(cfg.Model)
	if err != nil {
		return nil, fmt.Errorf("unsupported model: %w", err)
	}

	embeddings := &CohereEmbeddings{
		client:     client,
		model:      cfg.Model,
		dimensions: dims,
		logger:     cfg.Logger,
		metrics:    cfg.Metrics,
	}

	if embeddings.logger != nil {
		embeddings.logger.Info("cohere embeddings initialized",
			logger.String("model", cfg.Model),
			logger.Int("dimensions", dims))
	}

	return embeddings, nil
}

// Embed generates embeddings for the given texts.
func (c *CohereEmbeddings) Embed(ctx context.Context, texts []string) ([]sdk.Vector, error) {
	if len(texts) == 0 {
		return []sdk.Vector{}, nil
	}

	start := time.Now()
	defer func() {
		if c.metrics != nil {
			c.metrics.Histogram("forge.integrations.cohere.embed_duration").Observe(time.Since(start).Seconds())
		}
	}()

	// Create embed request
	resp, err := c.client.Embed(ctx, &coherego.EmbedRequest{
		Texts:     texts,
		Model:     &c.model,
		InputType: coherego.EmbedInputTypeSearchDocument.Ptr(),
	})
	if err != nil {
		return nil, fmt.Errorf("cohere embed request failed: %w", err)
	}

	// Check response
	if len(resp.Embeddings) != len(texts) {
		return nil, fmt.Errorf("mismatch in embeddings count: expected %d, got %d", len(texts), len(resp.Embeddings))
	}

	// Convert to SDK format
	vectors := make([]sdk.Vector, len(resp.Embeddings))
	for i, embedding := range resp.Embeddings {
		vectors[i] = sdk.Vector{
			ID:     fmt.Sprintf("cohere-embedding-%d", i),
			Values: embedding,
			Metadata: map[string]any{
				"model": c.model,
				"text":  texts[i],
				"index": i,
			},
		}
	}

	if c.logger != nil {
		c.logger.Debug("generated cohere embeddings",
			logger.Int("count", len(vectors)),
			logger.Duration("duration", time.Since(start)))
	}

	if c.metrics != nil {
		c.metrics.Counter("forge.integrations.cohere.embed",
			metrics.WithLabel("model", c.model)).Add(float64(len(vectors)))

		// Track tokens if available
		if resp.Meta != nil && resp.Meta.BilledUnits != nil && resp.Meta.BilledUnits.InputTokens != nil {
			c.metrics.Counter("forge.integrations.cohere.tokens",
				metrics.WithLabel("model", c.model)).Add(float64(*resp.Meta.BilledUnits.InputTokens))
		}
	}

	return vectors, nil
}

// Dimensions returns the embedding dimension for the model.
func (c *CohereEmbeddings) Dimensions() int {
	return c.dimensions
}

// getModelDimensions returns the dimension for a given Cohere embedding model.
func getModelDimensions(model string) (int, error) {
	switch model {
	case "embed-english-v3.0":
		return 1024, nil
	case "embed-multilingual-v3.0":
		return 1024, nil
	case "embed-english-light-v3.0":
		return 384, nil
	case "embed-multilingual-light-v3.0":
		return 384, nil
	case "embed-english-v2.0":
		return 4096, nil
	case "embed-english-light-v2.0":
		return 1024, nil
	case "embed-multilingual-v2.0":
		return 768, nil
	default:
		return 0, fmt.Errorf("unknown model: %s", model)
	}
}
