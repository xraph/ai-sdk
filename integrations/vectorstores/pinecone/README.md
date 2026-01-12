# Pinecone Vector Store

Pinecone managed vector database implementation using the official Go SDK.

## ✅ Features

- **Official Go SDK**: Uses `github.com/pinecone-io/go-pinecone`
- **Fully Managed**: No infrastructure to manage
- **Serverless**: Auto-scaling with pay-per-use
- **High Availability**: Built-in replication and failover
- **Advanced Filtering**: Metadata filtering with complex queries
- **Namespaces**: Multi-tenant data isolation
- **Global Distribution**: Deploy close to your users

## 🚀 Installation

```bash
# Install integration
go get github.com/xraph/ai-sdk/integrations/vectorstores/pinecone

# Sign up for Pinecone (free tier available)
# https://www.pinecone.io/
```

## 📖 Usage

### Basic Example

```go
package main

import (
    "context"
    
    sdk "github.com/xraph/ai-sdk"
    "github.com/xraph/ai-sdk/integrations/vectorstores/pinecone"
)

func main() {
    ctx := context.Background()
    
    // Create store
    store, err := pinecone.NewPineconeVectorStore(ctx, pinecone.Config{
        APIKey:    os.Getenv("PINECONE_API_KEY"),
        IndexName: "my-index",  // Must exist
        Namespace: "production", // Optional
    })
    if err != nil {
        panic(err)
    }
    defer store.Close()
    
    // Upsert vectors
    err = store.Upsert(ctx, []sdk.Vector{
        {
            ID:     "doc1",
            Values: []float64{0.1, 0.2, 0.3, /* ... */},
            Metadata: map[string]any{
                "title":    "Introduction",
                "category": "documentation",
            },
        },
    })
    
    // Query
    results, err := store.Query(ctx, queryVector, 10, nil)
    
    // Query with filter
    filter := map[string]any{
        "category": "documentation",
    }
    results, err = store.Query(ctx, queryVector, 10, filter)
    
    // Delete
    err = store.Delete(ctx, []string{"doc1"})
    
    // Get stats
    stats, err := store.Stats(ctx)
    fmt.Printf("Vectors: %d, Dimension: %d\n", 
        stats.TotalVectorCount, stats.Dimension)
}
```

### Creating an Index

Before using the store, create an index via Pinecone Console or API:

```bash
# Via Pinecone CLI
pinecone create-index \
  --name my-index \
  --dimension 1536 \
  --metric cosine \
  --cloud aws \
  --region us-east-1
```

Or programmatically:

```go
client, _ := pinecone.NewClient(pinecone.NewClientParams{
    ApiKey: os.Getenv("PINECONE_API_KEY"),
})

_, err := client.CreateServerlessIndex(ctx, &pinecone.CreateServerlessIndexRequest{
    Name:      "my-index",
    Dimension: 1536,
    Metric:    pinecone.Cosine,
    Cloud:     pinecone.Aws,
    Region:    "us-east-1",
})
```

### With Observability

```go
import (
    "github.com/xraph/go-utils/log"
    "github.com/xraph/go-utils/metrics"
)

logger := log.NewLogger(log.LevelDebug)
metricsProvider := metrics.NewPrometheusMetrics()

store, err := pinecone.NewPineconeVectorStore(ctx, pinecone.Config{
    APIKey:    os.Getenv("PINECONE_API_KEY"),
    IndexName: "production-index",
    Namespace: "prod",
    Logger:    logger,
    Metrics:   metricsProvider,
})
```

## 🔧 Configuration

### Config Options

```go
type Config struct {
    // Required
    APIKey    string  // Pinecone API key
    IndexName string  // Index name (must exist)
    
    // Optional
    Host      string        // Index host (auto-detected if empty)
    Namespace string        // Namespace for data isolation
    Timeout   time.Duration // Request timeout (default: 30s)
    
    // Observability
    Logger  logger.Logger
    Metrics metrics.Metrics
}
```

### Environment Variables

```bash
export PINECONE_API_KEY="your-api-key-here"
export PINECONE_INDEX_NAME="my-index"
```

## 📊 Performance

### Serverless Performance

| Operation | P50 Latency | P99 Latency |
|-----------|-------------|-------------|
| Upsert (single) | ~15ms | ~50ms |
| Upsert (batch 100) | ~100ms | ~300ms |
| Query (top 10) | ~20ms | ~80ms |
| Query (top 100) | ~30ms | ~120ms |
| Delete (batch) | ~30ms | ~100ms |

### Pod-based Performance (faster)

| Operation | P50 Latency | P99 Latency |
|-----------|-------------|-------------|
| Upsert (single) | ~5ms | ~20ms |
| Upsert (batch 100) | ~30ms | ~100ms |
| Query (top 10) | ~8ms | ~30ms |
| Query (top 100) | ~15ms | ~50ms |

## 🎯 Advanced Features

### Namespaces

Use namespaces for multi-tenancy:

```go
// Create separate stores per tenant
tenantStore, _ := pinecone.NewPineconeVectorStore(ctx, pinecone.Config{
    APIKey:    apiKey,
    IndexName: "shared-index",
    Namespace: fmt.Sprintf("tenant-%s", tenantID),
})
```

### Sparse-Dense Hybrid Search

```go
// Coming soon - hybrid sparse-dense vectors
// for combining semantic and keyword search
```

### Metadata Filtering

```go
// Complex filters
filter := map[string]any{
    "category": "documentation",
    "language": "en",
    "year": 2024,
}

results, err := store.Query(ctx, vector, 10, filter)
```

## 📈 Metrics

When metrics are enabled:

| Metric | Type | Description |
|--------|------|-------------|
| `forge.integrations.pinecone.upsert` | Counter | Vectors upserted |
| `forge.integrations.pinecone.query` | Counter | Queries executed |
| `forge.integrations.pinecone.delete` | Counter | Vectors deleted |
| `forge.integrations.pinecone.upsert_duration` | Histogram | Upsert latency (seconds) |
| `forge.integrations.pinecone.query_duration` | Histogram | Query latency (seconds) |
| `forge.integrations.pinecone.results` | Histogram | Results per query |

## 🧪 Testing

```bash
# Unit tests (no Pinecone account required)
go test -short ./...

# Integration tests (requires Pinecone API key)
export PINECONE_API_KEY="your-key"
go test ./...

# Benchmarks
go test -bench=. ./...
```

## 💰 Pricing

### Serverless (Pay-per-use)

- **Writes**: $0.002 per 1K write units
- **Reads**: $0.002 per 1K read units
- **Storage**: $0.25 per GB-month
- **Free tier**: 2M write units, 10M read units/month

### Pods (Reserved capacity)

- **s1.x1**: $70/month (100K 1536-dim vectors)
- **s1.x2**: $140/month (500K 1536-dim vectors)
- **p1.x1**: $185/month (1M 1536-dim vectors)

## 🐛 Troubleshooting

### API Key Invalid

```
Error: API key is invalid
```

**Solution**: Verify API key in Pinecone Console:
```bash
export PINECONE_API_KEY="your-correct-key"
```

### Index Not Found

```
Error: index not found
```

**Solution**: Create the index first via Console or API.

### Dimension Mismatch

```
Error: dimension mismatch
```

**Solution**: Ensure all vectors match the index dimension:
```go
// Index created with dimension 1536
// All vectors must have exactly 1536 dimensions
```

### Rate Limiting

```
Error: rate limit exceeded
```

**Solution**: Implement exponential backoff or upgrade plan.

## 🆚 Comparison

### Serverless vs Pods

| Feature | Serverless | Pods |
|---------|------------|------|
| **Cost Model** | Pay-per-use | Fixed monthly |
| **Scaling** | Auto-scale | Manual |
| **Latency** | 15-30ms | 5-15ms |
| **Best For** | Variable workloads | Consistent traffic |

## 📚 Resources

- [Pinecone Documentation](https://docs.pinecone.io/)
- [Go SDK](https://github.com/pinecone-io/go-pinecone)
- [Pinecone Console](https://app.pinecone.io/)
- [Pricing Calculator](https://www.pinecone.io/pricing/)

## 📝 License

MIT License - see [LICENSE](../../../LICENSE) for details.

