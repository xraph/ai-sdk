# Cohere Embeddings

Official Cohere embeddings implementation for Forge AI SDK using Cohere Go SDK v2.

## ✅ Features

- ✅ Official Cohere Go SDK v2
- ✅ Support for all Cohere embedding models
- ✅ Multilingual embeddings
- ✅ Batch processing
- ✅ Token usage tracking
- ✅ Production-ready error handling
- ✅ Observability (logging & metrics)

## 🚀 Installation

```bash
go get github.com/xraph/ai-sdk/integrations/embeddings/cohere
go get github.com/cohere-ai/cohere-go/v2
```

## 📖 Usage

### Basic Usage

```go
package main

import (
	"context"
	"log"

	"github.com/xraph/ai-sdk/integrations/embeddings/cohere"
	sdk "github.com/xraph/ai-sdk"
)

func main() {
	ctx := context.Background()

	// Create Cohere embeddings
	embedder, err := cohere.NewCohereEmbeddings(cohere.Config{
		APIKey: "your-cohere-api-key",
		Model:  "embed-english-v3.0",
	})
	if err != nil {
		log.Fatal(err)
	}

	// Generate embeddings
	texts := []string{
		"The quick brown fox jumps over the lazy dog",
		"Machine learning is transforming technology",
	}

	vectors, err := embedder.Embed(ctx, texts)
	if err != nil {
		log.Fatal(err)
	}

	for i, v := range vectors {
		log.Printf("Text %d: %d dimensions\n", i, len(v.Values))
	}
}
```

### With Environment Variable

```go
import "os"

embedder, err := cohere.NewCohereEmbeddings(cohere.Config{
	APIKey: os.Getenv("COHERE_API_KEY"),
	Model:  "embed-multilingual-v3.0",
})
```

## 🔧 Configuration

```go
type Config struct {
	APIKey  string        // Required: Cohere API key
	Model   string        // Required: Model name
	Timeout time.Duration // Optional: Request timeout (default: 30s)
	Logger  logger.Logger // Optional: Logger for debugging
	Metrics metrics.Metrics // Optional: Metrics for monitoring
}
```

## 🤖 Supported Models

### V3 Models (Recommended)

| Model | Dimensions | Languages | Use Case |
|-------|------------|-----------|----------|
| `embed-english-v3.0` | 1024 | English | High quality English embeddings |
| `embed-multilingual-v3.0` | 1024 | 100+ | Multilingual support |
| `embed-english-light-v3.0` | 384 | English | Faster, smaller embeddings |
| `embed-multilingual-light-v3.0` | 384 | 100+ | Faster multilingual |

### V2 Models (Legacy)

| Model | Dimensions | Languages |
|-------|------------|-----------|
| `embed-english-v2.0` | 4096 | English |
| `embed-english-light-v2.0` | 1024 | English |
| `embed-multilingual-v2.0` | 768 | 100+ |

**Recommendation**: Use V3 models for best performance and quality.

## 📊 Performance

| Model | Latency (p50) | Cost per 1M tokens | Dimensions |
|-------|---------------|-------------------|------------|
| embed-english-v3.0 | ~200ms | $0.10 | 1024 |
| embed-multilingual-v3.0 | ~200ms | $0.10 | 1024 |
| embed-english-light-v3.0 | ~100ms | $0.10 | 384 |
| embed-multilingual-light-v3.0 | ~100ms | $0.10 | 384 |

*Benchmarks with batch size 10, measured via Cohere API*

## 🌍 Multilingual Support

```go
embedder, _ := cohere.NewCohereEmbeddings(cohere.Config{
	APIKey: apiKey,
	Model:  "embed-multilingual-v3.0",
})

texts := []string{
	"Hello, world!",           // English
	"Bonjour le monde!",       // French
	"Hola mundo!",             // Spanish
	"你好世界！",               // Chinese
	"مرحبا بالعالم!",         // Arabic
}

vectors, _ := embedder.Embed(ctx, texts)
// All vectors are in same embedding space for cross-lingual similarity
```

## 💰 Cost Optimization

### Batch Processing

```go
// More efficient than individual requests
largeTextSet := make([]string, 100)
vectors, _ := embedder.Embed(ctx, largeTextSet) // Single API call
```

### Choose Right Model

```go
// For high accuracy
embedder, _ := cohere.NewCohereEmbeddings(cohere.Config{
	Model: "embed-english-v3.0", // 1024 dims, best quality
})

// For speed and lower cost
embedder, _ := cohere.NewCohereEmbeddings(cohere.Config{
	Model: "embed-english-light-v3.0", // 384 dims, faster
})
```

## 🔗 Use with Forge AI SDK

### RAG System

```go
import (
	"github.com/xraph/ai-sdk"
	"github.com/xraph/ai-sdk/integrations/embeddings/cohere"
	"github.com/xraph/ai-sdk/integrations/vectorstores/pinecone"
)

// Create embeddings
embedder, _ := cohere.NewCohereEmbeddings(cohere.Config{
	APIKey: os.Getenv("COHERE_API_KEY"),
	Model:  "embed-english-v3.0",
})

// Create vector store
store, _ := pinecone.NewPineconeVectorStore(pinecone.Config{
	APIKey:    os.Getenv("PINECONE_API_KEY"),
	IndexName: "documents",
})

// Create RAG system
rag := sdk.NewRAG(sdk.RAGConfig{
	VectorStore: store,
	Embedder:    embedder,
	TopK:        5,
})
```

## 🧪 Testing

```bash
# Unit tests
go test ./...

# Integration tests (requires API key)
export COHERE_API_KEY="your-api-key"
go test -tags=integration ./...
```

## 📈 Advanced Usage

### Custom Timeout

```go
embedder, _ := cohere.NewCohereEmbeddings(cohere.Config{
	APIKey:  apiKey,
	Model:   "embed-english-v3.0",
	Timeout: 60 * time.Second,
})
```

### Error Handling

```go
vectors, err := embedder.Embed(ctx, texts)
if err != nil {
	// Handle specific errors
	if strings.Contains(err.Error(), "invalid_api_key") {
		log.Fatal("Invalid Cohere API key")
	}
	if strings.Contains(err.Error(), "rate_limit") {
		// Implement retry with backoff
		time.Sleep(time.Second)
		vectors, err = embedder.Embed(ctx, texts)
	}
}
```

## ⚠️ Production Considerations

### Pros
- ✅ High-quality embeddings
- ✅ Excellent multilingual support
- ✅ Official Go SDK
- ✅ Competitive pricing
- ✅ Good documentation
- ✅ Fast inference

### Cons
- ⚠️ Requires API key
- ⚠️ External API dependency
- ⚠️ Rate limits apply
- ⚠️ Network latency

### Best Practices

**Rate Limiting**:
```go
import "golang.org/x/time/rate"

limiter := rate.NewLimiter(rate.Limit(10), 1) // 10 req/sec

limiter.Wait(ctx)
vectors, err := embedder.Embed(ctx, texts)
```

**Caching**:
```go
// Cache embeddings for frequently used texts
cache := make(map[string][]float64)

func getEmbedding(text string) ([]float64, error) {
	if cached, ok := cache[text]; ok {
		return cached, nil
	}
	vectors, err := embedder.Embed(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	cache[text] = vectors[0].Values
	return vectors[0].Values, nil
}
```

**Retry Logic**:
```go
import "github.com/cenkalti/backoff/v4"

b := backoff.NewExponentialBackOff()
var vectors []sdk.Vector

err := backoff.Retry(func() error {
	var retryErr error
	vectors, retryErr = embedder.Embed(ctx, texts)
	return retryErr
}, b)
```

## 🔧 Troubleshooting

### Invalid API Key

```go
// Verify API key
embedder, err := cohere.NewCohereEmbeddings(cohere.Config{
	APIKey: os.Getenv("COHERE_API_KEY"),
	Model:  "embed-english-v3.0",
})
if err != nil {
	log.Fatal("Check COHERE_API_KEY environment variable")
}
```

### Rate Limit Errors

- Free tier: 100 requests/minute
- Production tier: Higher limits
- Implement exponential backoff
- Batch requests when possible

### Timeout Errors

```go
embedder, _ := cohere.NewCohereEmbeddings(cohere.Config{
	Timeout: 120 * time.Second, // Increase for large batches
})
```

## 📚 Resources

- [Cohere Documentation](https://docs.cohere.com/)
- [Cohere Go SDK](https://github.com/cohere-ai/cohere-go)
- [Embed API Reference](https://docs.cohere.com/reference/embed)
- [Pricing](https://cohere.com/pricing)
- [Supported Languages](https://docs.cohere.com/docs/supported-languages)

## 🤝 Contributing

Contributions welcome! See the main [CONTRIBUTING.md](../../../CONTRIBUTING.md).

## 📝 License

MIT License - see [LICENSE](../../../LICENSE)

