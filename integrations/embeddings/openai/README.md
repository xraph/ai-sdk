# OpenAI Embeddings

OpenAI embeddings implementation using the official Go SDK.

## ✅ Features

- **Latest Models**: Support for text-embedding-3-small/large
- **Flexible Dimensions**: Configure dimensions for text-embedding-3-* models
- **Batch Processing**: Automatic batching for large inputs
- **Production Ready**: Uses official `go-openai` SDK
- **Cost Effective**: Choose model based on quality/cost needs

## 🚀 Installation

```bash
# Install integration
go get github.com/xraph/ai-sdk/integrations/embeddings/openai
```

## 📖 Usage

### Basic Example

```go
package main

import (
    "context"
    
    sdk "github.com/xraph/ai-sdk"
    "github.com/xraph/ai-sdk/integrations/embeddings/openai"
    "github.com/xraph/ai-sdk/integrations/vectorstores/pinecone"
)

func main() {
    ctx := context.Background()
    
    // Create embeddings
    embedder, err := openai.NewOpenAIEmbeddings(openai.Config{
        APIKey: os.Getenv("OPENAI_API_KEY"),
        Model:  openai.ModelTextEmbedding3Small,
    })
    if err != nil {
        panic(err)
    }
    
    // Generate embeddings
    texts := []string{
        "The quick brown fox jumps over the lazy dog",
        "AI is transforming software development",
    }
    
    vectors, err := embedder.Embed(ctx, texts)
    fmt.Printf("Generated %d embeddings\n", len(vectors))
    fmt.Printf("Dimensions: %d\n", embedder.Dimensions())
    
    // Use with RAG
    vectorStore, _ := pinecone.NewPineconeVectorStore(ctx, pinecone.Config{
        APIKey:    os.Getenv("PINECONE_API_KEY"),
        IndexName: "my-index",
    })
    
    rag := sdk.NewRAG(vectorStore, embedder, logger, metrics, nil)
}
```

### Model Selection

```go
// Most cost-effective (recommended for most use cases)
embedder, _ := openai.NewOpenAIEmbeddings(openai.Config{
    APIKey: apiKey,
    Model:  openai.ModelTextEmbedding3Small,  // 1536 dims, $0.02/1M tokens
})

// Higher quality
embedder, _ := openai.NewOpenAIEmbeddings(openai.Config{
    APIKey: apiKey,
    Model:  openai.ModelTextEmbedding3Large,  // 3072 dims, $0.13/1M tokens
})

// Legacy (still supported)
embedder, _ := openai.NewOpenAIEmbeddings(openai.Config{
    APIKey: apiKey,
    Model:  openai.ModelTextEmbeddingAda002,  // 1536 dims, $0.10/1M tokens
})
```

### Custom Dimensions

text-embedding-3-* models support custom dimensions (trade quality for storage):

```go
embedder, _ := openai.NewOpenAIEmbeddings(openai.Config{
    APIKey:     apiKey,
    Model:      openai.ModelTextEmbedding3Large,
    Dimensions: 1024,  // Reduce from 3072 to 1024
})
```

## 🔧 Configuration

### Config Options

```go
type Config struct {
    // Required
    APIKey string  // OpenAI API key
    
    // Optional
    Model       string  // Model name (default: "text-embedding-3-small")
    Dimensions  int     // Custom dimensions (text-embedding-3-* only)
    BaseURL     string  // Custom API base URL
    OrgID       string  // Organization ID
    
    // Observability
    Logger  logger.Logger
    Metrics metrics.Metrics
}
```

## 📊 Model Comparison

| Model | Dimensions | Cost/1M tokens | Quality | Use Case |
|-------|------------|----------------|---------|----------|
| text-embedding-3-small | 1536 | $0.02 | Good | Most applications |
| text-embedding-3-large | 3072 | $0.13 | Excellent | High-quality search |
| text-embedding-ada-002 | 1536 | $0.10 | Good | Legacy applications |

## 📈 Metrics

When metrics are enabled:

| Metric | Type | Description |
|--------|------|-------------|
| `forge.integrations.openai.embed` | Counter | Embeddings generated |
| `forge.integrations.openai.tokens` | Counter | Tokens consumed |
| `forge.integrations.openai.embed_duration` | Histogram | Embedding latency (seconds) |

## 💰 Cost Optimization

### Tips

1. **Use text-embedding-3-small**: 5-10x cheaper than large
2. **Reduce dimensions**: Trade quality for cost/storage
3. **Batch requests**: Reduce API calls
4. **Cache embeddings**: Avoid re-embedding same text

### Example Cost Calculation

```go
// 1M tokens at various configs:
// text-embedding-3-small (1536 dims): $0.02
// text-embedding-3-large (3072 dims): $0.13
// text-embedding-3-large (1024 dims): $0.13 (same cost, less storage)
```

## 🧪 Testing

```bash
# Unit tests (no API key required)
go test -short ./...

# Integration tests (requires API key)
export OPENAI_API_KEY="your-key"
go test ./...
```

## 🐛 Troubleshooting

### Invalid API Key

```
Error: invalid API key
```

**Solution**: Verify API key:
```bash
export OPENAI_API_KEY="sk-..."
```

### Rate Limiting

```
Error: rate limit exceeded
```

**Solution**: Implement exponential backoff or reduce batch size.

### Token Limit

```
Error: maximum context length exceeded
```

**Solution**: Split large texts into smaller chunks before embedding.

## 📝 License

MIT License - see [LICENSE](../../../LICENSE) for details.

