# In-Memory Vector Store

Pure Go in-memory vector store implementation for testing and development.

## ⚠️ Important

**NOT for production use.** This implementation:
- Stores all vectors in memory (lost on restart)
- No persistence
- Limited scalability
- Best for testing, development, and prototypes

## ✅ Use Cases

- Unit testing
- Integration testing
- Local development
- Prototyping
- Demos and examples

## 🚀 Installation

```bash
go get github.com/xraph/ai-sdk/integrations/vectorstores/memory
```

## 📖 Usage

### Basic Example

```go
package main

import (
    "context"
    
    sdk "github.com/xraph/ai-sdk"
    "github.com/xraph/ai-sdk/integrations/vectorstores/memory"
)

func main() {
    ctx := context.Background()
    
    // Create store
    store := memory.NewMemoryVectorStore(memory.Config{})
    
    // Upsert vectors
    err := store.Upsert(ctx, []sdk.Vector{
        {
            ID:     "doc1",
            Values: []float64{0.1, 0.2, 0.3},
            Metadata: map[string]any{
                "title": "Introduction",
                "type":  "document",
            },
        },
        {
            ID:     "doc2",
            Values: []float64{0.4, 0.5, 0.6},
            Metadata: map[string]any{
                "title": "Advanced Topics",
                "type":  "document",
            },
        },
    })
    
    // Query by similarity
    results, err := store.Query(ctx, []float64{0.15, 0.25, 0.35}, 5, nil)
    for _, match := range results {
        fmt.Printf("ID: %s, Score: %.4f\n", match.ID, match.Score)
    }
    
    // Delete vectors
    err = store.Delete(ctx, []string{"doc1"})
    
    // Clear all
    err = store.Clear(ctx)
}
```

### With Filtering

```go
// Query with metadata filter
filter := map[string]any{
    "type": "document",
}

results, err := store.Query(ctx, queryVector, 10, filter)
```

### With Observability

```go
import (
    "github.com/xraph/go-utils/log"
    "github.com/xraph/go-utils/metrics"
)

logger := log.NewLogger(log.LevelDebug)
metricsProvider := metrics.NewPrometheusMetrics()

store := memory.NewMemoryVectorStore(memory.Config{
    Logger:  logger,
    Metrics: metricsProvider,
})
```

### Using with RAG

```go
import (
    sdk "github.com/xraph/ai-sdk"
    "github.com/xraph/ai-sdk/integrations/vectorstores/memory"
    "github.com/xraph/ai-sdk/integrations/embeddings/openai"
)

vectorStore := memory.NewMemoryVectorStore(memory.Config{})
embedder := openai.NewOpenAIEmbeddings(openai.Config{
    APIKey: os.Getenv("OPENAI_API_KEY"),
    Model:  "text-embedding-3-small",
})

rag := sdk.NewRAG(vectorStore, embedder, logger, metrics, nil)

// Index documents
rag.IndexDocument(ctx, sdk.Document{
    ID:      "doc1",
    Content: "AI is transforming software development...",
})

// Query
result, _ := rag.GenerateWithContext(ctx, "What is AI?", generator)
```

## 📊 Performance

### Characteristics

- **Upsert**: O(1) per vector
- **Query**: O(n) linear scan with cosine similarity
- **Delete**: O(1) per ID
- **Memory**: ~16 bytes per dimension per vector + metadata overhead

### Benchmarks

On Apple M1 Pro with 1536-dimensional vectors:

| Operation | Time | Throughput |
|-----------|------|------------|
| Upsert (single) | ~500ns | 2M ops/sec |
| Query (1K vectors) | ~5ms | 200 queries/sec |
| Delete (single) | ~100ns | 10M ops/sec |

## 🔧 API Reference

### NewMemoryVectorStore

```go
func NewMemoryVectorStore(cfg Config) *MemoryVectorStore
```

Creates a new in-memory vector store.

**Config**:
- `Logger`: Optional logger for debugging
- `Metrics`: Optional metrics provider

### Upsert

```go
func (m *MemoryVectorStore) Upsert(ctx context.Context, vectors []Vector) error
```

Adds or updates vectors. If a vector with the same ID exists, it's replaced.

**Errors**:
- Empty vector ID
- Empty vector values

### Query

```go
func (m *MemoryVectorStore) Query(ctx context.Context, vector []float64, limit int, filter map[string]any) ([]VectorMatch, error)
```

Performs cosine similarity search and returns top K matches.

**Parameters**:
- `vector`: Query vector (must match dimensions of stored vectors)
- `limit`: Maximum number of results
- `filter`: Optional metadata filter (exact match)

**Errors**:
- Empty query vector
- Non-positive limit

**Notes**:
- Vectors with mismatched dimensions are silently skipped
- Results sorted by similarity (descending)
- Score range: [-1, 1] where 1 is identical

### Delete

```go
func (m *MemoryVectorStore) Delete(ctx context.Context, ids []string) error
```

Removes vectors by ID. Non-existent IDs are silently ignored.

### Clear

```go
func (m *MemoryVectorStore) Clear(ctx context.Context) error
```

Removes all vectors from the store.

### Count

```go
func (m *MemoryVectorStore) Count() int
```

Returns the number of vectors currently in the store.

## 🔬 Testing

```bash
# Run tests
go test ./...

# With coverage
go test -cover ./...

# Run benchmarks
go test -bench=. ./...

# Race detection
go test -race ./...
```

## 📈 Metrics

When metrics are enabled, the following metrics are emitted:

| Metric | Type | Description |
|--------|------|-------------|
| `forge.integrations.memory.upsert` | Counter | Number of vectors upserted |
| `forge.integrations.memory.query` | Counter | Number of queries executed |
| `forge.integrations.memory.delete` | Counter | Number of vectors deleted |
| `forge.integrations.memory.total_vectors` | Gauge | Current vector count |
| `forge.integrations.memory.results` | Histogram | Query result counts |

## 🤔 When to Use

### ✅ Good For

- **Unit testing**: Fast, no external dependencies
- **Integration testing**: Predictable behavior
- **Local development**: Zero setup
- **Prototyping**: Quick iteration
- **CI/CD**: No infrastructure needed

### ❌ Not Good For

- **Production**: Data loss on restart
- **Large datasets**: Everything in memory
- **Distributed systems**: Single process only
- **High concurrency**: Global lock on queries

## 🔄 Migration to Production

When ready for production, migrate to a persistent vector store:

**PostgreSQL + pgvector**:
```go
import "github.com/xraph/ai-sdk/integrations/vectorstores/pgvector"

store := pgvector.NewPgVectorStore(pgvector.Config{
    ConnectionString: os.Getenv("DATABASE_URL"),
    TableName:        "vectors",
})
```

**Pinecone (managed)**:
```go
import "github.com/xraph/ai-sdk/integrations/vectorstores/pinecone"

store := pinecone.NewPineconeVectorStore(pinecone.Config{
    APIKey:    os.Getenv("PINECONE_API_KEY"),
    IndexName: "production-index",
})
```

All implement the same `VectorStore` interface - no code changes needed!

## 📝 License

MIT License - see [LICENSE](../../../LICENSE) for details.

