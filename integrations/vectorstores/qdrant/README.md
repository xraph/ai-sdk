# Qdrant Vector Store

Qdrant vector database implementation using the official Go SDK with gRPC.

## ✅ Features

- **Official Go SDK**: Uses `github.com/qdrant/go-client` (gRPC-based)
- **High Performance**: gRPC protocol for fast communication
- **Advanced Filtering**: Rich metadata filtering with complex conditions
- **Quantization**: Scalar and product quantization for memory efficiency
- **Hybrid Search**: Combine vector and full-text search
- **Docker Friendly**: Easy local development with Docker
- **Cloud & Self-Hosted**: Works with Qdrant Cloud or self-hosted

## 🚀 Installation

```bash
# Install integration
go get github.com/xraph/ai-sdk/integrations/vectorstores/qdrant

# Run Qdrant locally with Docker
docker run -p 6333:6333 -p 6334:6334 qdrant/qdrant
```

## 📖 Usage

### Basic Example

```go
package main

import (
    "context"
    
    sdk "github.com/xraph/ai-sdk"
    "github.com/xraph/ai-sdk/integrations/vectorstores/qdrant"
)

func main() {
    ctx := context.Background()
    
    // Create store (local)
    store, err := qdrant.NewQdrantVectorStore(ctx, qdrant.Config{
        Host:           "localhost:6334",  // gRPC port
        CollectionName: "my_vectors",
        VectorSize:     1536,              // dimensions
        Distance:       "cosine",          // cosine, euclidean, or dot
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
                "tags":     "ai,ml",
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
    
    // Count
    count, err := store.Count(ctx)
}
```

### Qdrant Cloud

```go
store, err := qdrant.NewQdrantVectorStore(ctx, qdrant.Config{
    Host:           "xyz-example.aws.cloud.qdrant.io:6334",
    CollectionName: "production_vectors",
    APIKey:         os.Getenv("QDRANT_API_KEY"),
    UseTLS:         true,
    VectorSize:     1536,
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

store, err := qdrant.NewQdrantVectorStore(ctx, qdrant.Config{
    Host:           "localhost:6334",
    CollectionName: "vectors",
    Logger:         logger,
    Metrics:        metricsProvider,
})
```

## 🔧 Configuration

### Config Options

```go
type Config struct {
    // Required
    Host           string  // Qdrant host with gRPC port (e.g., "localhost:6334")
    CollectionName string  // Collection name
    
    // Optional
    APIKey         string        // API key for Qdrant Cloud
    UseTLS         bool          // Use TLS (default: false for local)
    Timeout        time.Duration // gRPC timeout (default: 30s)
    VectorSize     uint64        // Vector dimensions
    Distance       string        // "cosine", "euclidean", "dot" (default: "cosine")
    
    // Observability
    Logger         logger.Logger
    Metrics        metrics.Metrics
}
```

### Distance Metrics

```go
// Cosine similarity (default) - best for normalized vectors
Config{Distance: "cosine"}

// Euclidean distance - best for absolute distances
Config{Distance: "euclidean"}

// Dot product - fastest, for pre-normalized vectors
Config{Distance: "dot"}
```

## 📊 Performance

### Benchmarks

On Qdrant v1.7 with 1M vectors (1536 dimensions) with HNSW:

| Operation | Latency | Throughput |
|-----------|---------|------------|
| Upsert (single) | ~2ms | 500 ops/sec |
| Upsert (batch 100) | ~20ms | 5K vectors/sec |
| Query (top 10) | ~3-5ms | 200-300 queries/sec |
| Query (top 100) | ~8-10ms | 100-150 queries/sec |
| Delete (batch) | ~5ms | 1K vectors/sec |

### Optimization Tips

1. **Batch Operations**: Batch upserts/deletes for better throughput
2. **gRPC vs REST**: Use gRPC port (6334) for 2-3x better performance
3. **Quantization**: Enable scalar quantization to reduce memory by 4x
4. **Payload Index**: Create payload indexes for frequently filtered fields
5. **HNSW Tuning**: Adjust `m` and `ef_construct` for recall/performance trade-off

## 🎯 Advanced Features

### Scalar Quantization

Enable quantization to reduce memory usage:

```bash
# Via Qdrant API
curl -X PUT http://localhost:6333/collections/my_vectors \
  -H 'Content-Type: application/json' \
  -d '{
    "optimizers_config": {
      "default_segment_number": 2
    },
    "quantization_config": {
      "scalar": {
        "type": "int8",
        "quantile": 0.99,
        "always_ram": true
      }
    }
  }'
```

### Payload Indexing

Create indexes for fast filtering:

```bash
curl -X PUT http://localhost:6333/collections/my_vectors/index \
  -H 'Content-Type: application/json' \
  -d '{
    "field_name": "category",
    "field_schema": "keyword"
  }'
```

### Custom HNSW Parameters

```bash
curl -X PUT http://localhost:6333/collections/my_vectors \
  -H 'Content-Type: application/json' \
  -d '{
    "vectors": {
      "size": 1536,
      "distance": "Cosine"
    },
    "hnsw_config": {
      "m": 16,
      "ef_construct": 100
    }
  }'
```

## 🔍 Filtering

### Simple Filters

```go
// Exact match
filter := map[string]any{
    "category": "documentation",
    "language": "en",
}

results, err := store.Query(ctx, vector, 10, filter)
```

### Advanced Filtering

For complex filters, use Qdrant's native filter API directly:

```go
// Range query
filter := &qdrant.Filter{
    Must: []*qdrant.Condition{
        {
            ConditionOneOf: &qdrant.Condition_Field{
                Field: &qdrant.FieldCondition{
                    Key: "price",
                    Range: &qdrant.Range{
                        Gte: floatPtr(10.0),
                        Lte: floatPtr(100.0),
                    },
                },
            },
        },
    },
}
```

## 📈 Metrics

When metrics are enabled:

| Metric | Type | Description |
|--------|------|-------------|
| `forge.integrations.qdrant.upsert` | Counter | Vectors upserted |
| `forge.integrations.qdrant.query` | Counter | Queries executed |
| `forge.integrations.qdrant.delete` | Counter | Vectors deleted |
| `forge.integrations.qdrant.upsert_duration` | Histogram | Upsert latency (seconds) |
| `forge.integrations.qdrant.query_duration` | Histogram | Query latency (seconds) |
| `forge.integrations.qdrant.results` | Histogram | Results per query |

## 🧪 Testing

```bash
# Unit tests (no Qdrant required)
go test -short ./...

# Integration tests (requires Docker)
docker run -d -p 6333:6333 -p 6334:6334 qdrant/qdrant
go test ./...

# With race detection
go test -race ./...

# Benchmarks
go test -bench=. ./...
```

## 🐳 Docker Setup

### Basic Setup

```bash
docker run -p 6333:6333 -p 6334:6334 \
  -v $(pwd)/qdrant_storage:/qdrant/storage \
  qdrant/qdrant
```

### With Persistence

```bash
docker run -d \
  --name qdrant \
  -p 6333:6333 \
  -p 6334:6334 \
  -v $(pwd)/qdrant_storage:/qdrant/storage:z \
  qdrant/qdrant
```

### Docker Compose

```yaml
version: '3.8'
services:
  qdrant:
    image: qdrant/qdrant:latest
    ports:
      - "6333:6333"  # REST API
      - "6334:6334"  # gRPC
    volumes:
      - ./qdrant_storage:/qdrant/storage:z
```

## 🐛 Troubleshooting

### Connection Refused

```
Error: connection refused
```

**Solution**: Ensure Qdrant is running and use gRPC port (6334):
```bash
docker ps | grep qdrant
# Use localhost:6334 not localhost:6333
```

### Collection Not Found

```
Error: collection not found
```

**Solution**: Specify VectorSize in config to auto-create:
```go
Config{
    Host:       "localhost:6334",
    VectorSize: 1536,  // Auto-creates collection
}
```

### TLS Errors (Cloud)

```
Error: TLS handshake failed
```

**Solution**: Enable TLS for Qdrant Cloud:
```go
Config{
    Host:   "xyz.cloud.qdrant.io:6334",
    UseTLS: true,
    APIKey: os.Getenv("QDRANT_API_KEY"),
}
```

### Dimension Mismatch

```
Error: wrong vector dimension
```

**Solution**: Ensure all vectors have the same dimensions as the collection.

## 🆚 Comparison

### Qdrant vs pgvector

| Feature | Qdrant | pgvector |
|---------|--------|----------|
| **Setup** | Docker | PostgreSQL + extension |
| **Protocol** | gRPC | SQL |
| **Filtering** | Advanced (nested, range) | Basic (JSONB) |
| **Quantization** | Built-in | Manual |
| **Horizontal Scaling** | Yes | Limited |
| **ACID** | No | Yes |
| **Best For** | Pure vector search | Existing PostgreSQL apps |

## 📚 Resources

- [Qdrant Documentation](https://qdrant.tech/documentation/)
- [Go Client SDK](https://github.com/qdrant/go-client)
- [Qdrant Cloud](https://cloud.qdrant.io/)
- [Performance Tuning](https://qdrant.tech/documentation/guides/optimize/)

## 📝 License

MIT License - see [LICENSE](../../../LICENSE) for details.

