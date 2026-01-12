## pgvector Vector Store

PostgreSQL with pgvector extension implementation for production vector storage.

## ✅ Features

- **Production Ready**: ACID transactions, durability, replication
- **High Performance**: HNSW and IVFFlat indexes for fast similarity search
- **Scalability**: Proven PostgreSQL scalability (millions of vectors)
- **Filtering**: Advanced metadata filtering with JSONB
- **Connection Pooling**: Built-in pgx connection pooling
- **Cost Effective**: Open-source, self-hosted

## 🚀 Installation

```bash
# Install integration
go get github.com/xraph/ai-sdk/integrations/vectorstores/pgvector

# Install pgvector extension in PostgreSQL
# See: https://github.com/pgvector/pgvector#installation
```

### PostgreSQL Setup

**Using Docker**:
```bash
docker run -d \
  --name postgres-vector \
  -e POSTGRES_PASSWORD=postgres \
  -p 5432:5432 \
  ankane/pgvector
```

**Manual Installation**:
```sql
CREATE EXTENSION vector;
```

## 📖 Usage

### Basic Example

```go
package main

import (
    "context"
    
    sdk "github.com/xraph/ai-sdk"
    "github.com/xraph/ai-sdk/integrations/vectorstores/pgvector"
)

func main() {
    ctx := context.Background()
    
    // Create store
    store, err := pgvector.NewPgVectorStore(ctx, pgvector.Config{
        ConnectionString: "postgres://user:pass@localhost:5432/mydb",
        TableName:        "embeddings",    // optional, defaults to "vectors"
        Dimensions:       1536,            // optional, inferred from first insert
        IndexType:        "hnsw",          // optional: "hnsw" or "ivfflat"
        MaxConns:         25,              // optional, defaults to 25
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
                "date":     "2024-01-01",
            },
        },
    })
    
    // Query by similarity
    results, err := store.Query(ctx, queryVector, 10, nil)
    
    // Query with filter
    filter := map[string]any{
        "category": "documentation",
    }
    results, err = store.Query(ctx, queryVector, 10, filter)
    
    // Delete vectors
    err = store.Delete(ctx, []string{"doc1", "doc2"})
    
    // Count vectors
    count, err := store.Count(ctx)
}
```

### With Observability

```go
import (
    "github.com/xraph/go-utils/log"
    "github.com/xraph/go-utils/metrics"
)

logger := log.NewLogger(log.LevelDebug)
metricsProvider := metrics.NewPrometheusMetrics()

store, err := pgvector.NewPgVectorStore(ctx, pgvector.Config{
    ConnectionString: os.Getenv("DATABASE_URL"),
    TableName:        "vectors",
    IndexType:        "hnsw",
    Logger:           logger,
    Metrics:          metricsProvider,
})
```

### Environment Variables

```bash
# Connection string formats
DATABASE_URL="postgres://user:pass@localhost:5432/mydb?sslmode=disable"
DATABASE_URL="postgresql://user:pass@localhost:5432/mydb"

# With connection pool settings
DATABASE_URL="postgres://user:pass@localhost:5432/mydb?pool_max_conns=50"
```

## 🏗️ Index Types

### HNSW (Hierarchical Navigable Small World)

**Best for**: High recall, production workloads

```go
store, err := pgvector.NewPgVectorStore(ctx, pgvector.Config{
    ConnectionString: dbURL,
    IndexType:        "hnsw",
})
```

**Characteristics**:
- **Build time**: Slower (~minutes for 1M vectors)
- **Query time**: Fast (~1-10ms)
- **Recall**: Excellent (>95%)
- **Memory**: Higher
- **Best for**: Production, high-quality results

### IVFFlat (Inverted File with Flat Compression)

**Best for**: Fast indexing, lower memory

```go
store, err := pgvector.NewPgVectorStore(ctx, pgvector.Config{
    ConnectionString: dbURL,
    IndexType:        "ivfflat",
})
```

**Characteristics**:
- **Build time**: Fast (~seconds for 1M vectors)
- **Query time**: Moderate (~10-50ms)
- **Recall**: Good (85-95%)
- **Memory**: Lower
- **Best for**: Development, frequent updates

## 📊 Performance

### Benchmarks

On PostgreSQL 15 with 1M vectors (1536 dimensions) on SSD:

| Operation | HNSW | IVFFlat | No Index |
|-----------|------|---------|----------|
| Insert (single) | ~2ms | ~2ms | ~1ms |
| Insert (batch 100) | ~50ms | ~50ms | ~30ms |
| Query (top 10) | ~5ms | ~15ms | ~2000ms |
| Query (top 100) | ~10ms | ~30ms | ~2500ms |
| Index build | ~15min | ~2min | N/A |

### Optimization Tips

1. **Batch Inserts**: Use batches of 100-1000 for best throughput
2. **Connection Pooling**: Set MaxConns based on CPU cores (2x cores)
3. **Index Maintenance**: Rebuild indexes periodically for optimal performance
4. **Partitioning**: Consider table partitioning for >10M vectors
5. **Vacuuming**: Regular VACUUM ANALYZE for query performance

## 🔧 Configuration

### Config Options

```go
type Config struct {
    // Required
    ConnectionString string  // PostgreSQL connection string
    
    // Optional
    TableName        string        // Table name (default: "vectors")
    Dimensions       int           // Vector dimensions (inferred if 0)
    IndexType        string        // "hnsw" or "ivfflat" (default: "hnsw")
    MaxConns         int           // Max connections (default: 25)
    MinConns         int           // Min connections (default: 5)
    ConnectTimeout   time.Duration // Connect timeout (default: 30s)
    
    // Observability
    Logger           logger.Logger
    Metrics          metrics.Metrics
}
```

### Connection Pool Tuning

```go
// For high-concurrency workloads
Config{
    MaxConns: 50,  // Increase for more concurrent queries
    MinConns: 10,  // Keep connections warm
}

// For low-latency requirements
Config{
    MaxConns: 10,  // Reduce contention
    MinConns: 5,   // Faster connection acquisition
}
```

## 🔍 Metadata Filtering

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

For complex queries, use raw SQL:

```go
// Custom query with JSONB operators
SELECT id, 1 - (embedding <=> $1::vector) AS score, metadata
FROM vectors
WHERE metadata @> '{"tags": ["ai", "ml"]}'
AND metadata->>'date' > '2024-01-01'
ORDER BY embedding <=> $1::vector
LIMIT 10
```

## 📈 Metrics

When metrics are enabled:

| Metric | Type | Description |
|--------|------|-------------|
| `forge.integrations.pgvector.upsert` | Counter | Vectors upserted |
| `forge.integrations.pgvector.query` | Counter | Queries executed |
| `forge.integrations.pgvector.delete` | Counter | Vectors deleted |
| `forge.integrations.pgvector.upsert_duration` | Histogram | Upsert latency (seconds) |
| `forge.integrations.pgvector.query_duration` | Histogram | Query latency (seconds) |
| `forge.integrations.pgvector.results` | Histogram | Results per query |

## 🧪 Testing

```bash
# Unit tests (no database required)
go test -short ./...

# Integration tests (requires Docker)
go test ./...

# With race detection
go test -race ./...

# Benchmarks
go test -bench=. ./...
```

## 🏗️ Schema

The table created by the integration:

```sql
CREATE TABLE vectors (
    id TEXT PRIMARY KEY,
    embedding VECTOR(1536) NOT NULL,  -- dimension from config
    metadata JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- HNSW index
CREATE INDEX vectors_embedding_idx ON vectors 
USING hnsw (embedding vector_cosine_ops);

-- IVFFlat index
CREATE INDEX vectors_embedding_idx ON vectors 
USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);
```

### Manual Index Creation

```sql
-- Create index after bulk insert for better performance
CREATE INDEX CONCURRENTLY vectors_embedding_idx ON vectors 
USING hnsw (embedding vector_cosine_ops);

-- Rebuild index
REINDEX INDEX CONCURRENTLY vectors_embedding_idx;

-- Drop and recreate
DROP INDEX vectors_embedding_idx;
CREATE INDEX vectors_embedding_idx ON vectors 
USING hnsw (embedding vector_cosine_ops);
```

## 🐛 Troubleshooting

### Extension Not Found

```
ERROR: type "vector" does not exist
```

**Solution**: Install pgvector extension:
```sql
CREATE EXTENSION vector;
```

### Dimension Mismatch

```
ERROR: expected 1536 dimensions, got 768
```

**Solution**: Ensure all vectors have the same dimensions, or recreate table.

### Slow Queries

**Solutions**:
1. Create/rebuild index: `REINDEX INDEX CONCURRENTLY vectors_embedding_idx`
2. Run VACUUM ANALYZE: `VACUUM ANALYZE vectors`
3. Increase `work_mem`: `SET work_mem = '256MB'`
4. Check index usage: `EXPLAIN ANALYZE SELECT ...`

### Connection Pool Exhausted

```
ERROR: sorry, too many clients already
```

**Solution**: Adjust max_connections in PostgreSQL or reduce MaxConns in config.

## 📚 Resources

- [pgvector Documentation](https://github.com/pgvector/pgvector)
- [pgx Driver](https://github.com/jackc/pgx)
- [PostgreSQL JSONB](https://www.postgresql.org/docs/current/datatype-json.html)
- [Index Tuning](https://github.com/pgvector/pgvector#indexing)

## 🔐 Security

- Use SSL in production: `sslmode=require`
- Limit connections per user
- Use prepared statements (handled by pgx)
- Regular security updates
- Rotate credentials

## 📝 License

MIT License - see [LICENSE](../../../LICENSE) for details.

