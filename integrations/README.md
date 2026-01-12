# Forge AI SDK - Integrations

Official integrations for the Forge AI SDK providing plug-and-play implementations for popular vector stores, state stores, embedding models, and caching solutions.

## 🎯 Philosophy

All integrations follow these principles:

1. **SDK-First**: Use official Go SDKs over REST APIs whenever available
2. **Thin Adapters**: Minimal wrapper over official clients
3. **Interface Compliance**: Implement standard SDK interfaces
4. **Zero Config Defaults**: Sensible defaults, optional configuration
5. **Production Ready**: Full test coverage, observability, error handling

## 📦 Available Integrations

### Vector Stores

| Integration | Status | Package | SDK Type | Notes |
|------------|--------|---------|----------|-------|
| [Memory](vectorstores/memory/) | ✅ | Pure Go | Native | For testing/development |
| [pgvector](vectorstores/pgvector/) | ✅ | `github.com/jackc/pgx/v5` | Native Driver | PostgreSQL + vector extension |
| [Qdrant](vectorstores/qdrant/) | ✅ | `github.com/qdrant/go-client` | Official | gRPC-based, Docker-friendly |
| [Pinecone](vectorstores/pinecone/) | ✅ | `github.com/pinecone-io/go-pinecone` | Official | Serverless & pod-based |
| [Weaviate](vectorstores/weaviate/) | 🚧 | `github.com/weaviate/weaviate-go-client/v4` | Official | Hybrid search |
| [Chroma](vectorstores/chroma/) | 🚧 | Custom HTTP | REST | No official Go SDK |

### State Stores

| Integration | Status | Package | SDK Type |
|------------|--------|---------|----------|
| [Memory](statestores/memory/) | ✅ | Pure Go | Native |
| [Redis](statestores/redis/) | ✅ | `github.com/redis/go-redis/v9` | Official |
| [PostgreSQL](statestores/postgres/) | 🚧 | `github.com/jackc/pgx/v5` | Native Driver |
| [DynamoDB](statestores/dynamodb/) | 🚧 | `github.com/aws/aws-sdk-go-v2` | Official |

### Cache Stores

| Integration | Status | Package | SDK Type |
|------------|--------|---------|----------|
| [Memory](caches/memory/) | ✅ | Pure Go | Native |
| [Redis](caches/redis/) | ✅ | `github.com/redis/go-redis/v9` | Official |
| [Memcached](caches/memcached/) | 🚧 | `github.com/bradfitz/gomemcache` | Community |

### Embedding Models

| Integration | Status | Package | SDK Type |
|------------|--------|---------|----------|
| [OpenAI](embeddings/openai/) | ✅ | `github.com/sashabaranov/go-openai` | Community |
| [Ollama](embeddings/ollama/) | ✅ | Internal | Native |
| [Cohere](embeddings/cohere/) | 🚧 | `github.com/cohere-ai/cohere-go/v2` | Official |
| [HuggingFace](embeddings/huggingface/) | 🚧 | Custom HTTP | REST |

Legend: ✅ Complete | 🚧 In Progress | ⏳ Planned

## 🚀 Quick Start

### Installation

Install only the integrations you need:

```bash
# Vector store
go get github.com/xraph/ai-sdk/integrations/vectorstores/pinecone

# Embeddings
go get github.com/xraph/ai-sdk/integrations/embeddings/openai

# State store
go get github.com/xraph/ai-sdk/integrations/statestores/redis
```

### Basic Usage

```go
package main

import (
    "context"
    
    sdk "github.com/xraph/ai-sdk"
    "github.com/xraph/ai-sdk/integrations/vectorstores/pinecone"
    "github.com/xraph/ai-sdk/integrations/embeddings/openai"
)

func main() {
    ctx := context.Background()
    
    // Setup vector store
    vectorStore, _ := pinecone.NewPineconeVectorStore(pinecone.Config{
        APIKey:    os.Getenv("PINECONE_API_KEY"),
        IndexName: "my-index",
    })
    
    // Setup embeddings
    embedder, _ := openai.NewOpenAIEmbeddings(openai.Config{
        APIKey: os.Getenv("OPENAI_API_KEY"),
        Model:  "text-embedding-3-small",
    })
    
    // Use with RAG
    rag := sdk.NewRAG(vectorStore, embedder, logger, metrics, nil)
    
    // Index documents
    rag.IndexDocument(ctx, sdk.Document{
        ID:      "doc1",
        Content: "AI is transforming software development...",
    })
    
    // Retrieve and generate
    result, _ := rag.GenerateWithContext(ctx, "What is AI?", generator)
}
```

## 📚 Comparison Matrix

### Vector Stores

| Feature | Memory | pgvector | Qdrant | Pinecone | Weaviate |
|---------|--------|----------|--------|----------|----------|
| **Cost** | Free | Self-hosted | Self-hosted/Cloud | Managed | Self-hosted/Cloud |
| **Setup** | Instant | PostgreSQL | Docker | API Key | Docker/Cloud |
| **Performance** | In-memory | Excellent | Excellent | Excellent | Good |
| **Scalability** | Limited | Good | Excellent | Excellent | Good |
| **Filtering** | Basic | Good | Excellent | Excellent | Excellent |
| **Indexing** | None | HNSW, IVFFlat | HNSW | Proprietary | HNSW |
| **Best For** | Testing | Production | Production | Prod/Scale | Hybrid Search |

### Embedding Models

| Provider | Model | Dimensions | Cost/1M tokens | Latency |
|----------|-------|------------|----------------|---------|
| OpenAI | text-embedding-3-small | 1536 | $0.02 | ~50ms |
| OpenAI | text-embedding-3-large | 3072 | $0.13 | ~100ms |
| Cohere | embed-english-v3.0 | 1024 | $0.10 | ~80ms |
| Ollama | nomic-embed-text | 768 | Free | ~200ms |

## 🏗️ Architecture

```
┌─────────────────────────────────────┐
│     Forge AI SDK (Core)             │
│  ┌─────────────────────────────┐   │
│  │  Interfaces                  │   │
│  │  - VectorStore               │   │
│  │  - StateStore                │   │
│  │  - CacheStore                │   │
│  │  - EmbeddingModel            │   │
│  └─────────────────────────────┘   │
└─────────────────────────────────────┘
              ▲
              │ implements
              │
┌─────────────────────────────────────┐
│     Integrations Module             │
│  ┌────────────┬──────────┬────────┐ │
│  │ Vector     │ State    │ Cache  │ │
│  │ Stores     │ Stores   │ Stores │ │
│  ├────────────┼──────────┼────────┤ │
│  │ Embeddings │ Plugins  │  MCP   │ │
│  └────────────┴──────────┴────────┘ │
└─────────────────────────────────────┘
```

## 🧪 Testing

Each integration includes:

- **Unit tests**: Mock external APIs
- **Integration tests**: Docker containers via testcontainers-go
- **Benchmarks**: Performance comparisons

Run tests:

```bash
# Unit tests only
go test ./... -short

# Include integration tests (requires Docker)
go test ./...

# Run benchmarks
go test -bench=. ./...
```

## 🔧 Development

### Adding a New Integration

1. Create directory: `integrations/{category}/{name}/`
2. Implement interface (e.g., `VectorStore`)
3. Add tests (unit + integration)
4. Create README with examples
5. Update main integrations README

### SDK Interface Compliance

All integrations must implement their respective interfaces:

**VectorStore**:
```go
Upsert(ctx context.Context, vectors []sdk.Vector) error
Query(ctx context.Context, vector []float64, limit int, filter map[string]any) ([]sdk.VectorMatch, error)
Delete(ctx context.Context, ids []string) error
```

**StateStore**:
```go
Save(ctx context.Context, state *sdk.AgentState) error
Load(ctx context.Context, agentID, sessionID string) (*sdk.AgentState, error)
Delete(ctx context.Context, agentID, sessionID string) error
List(ctx context.Context, agentID string) ([]string, error)
```

**CacheStore**:
```go
Get(ctx context.Context, key string) ([]byte, bool, error)
Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
Delete(ctx context.Context, key string) error
Clear(ctx context.Context) error
```

**EmbeddingModel**:
```go
Embed(ctx context.Context, texts []string) ([]sdk.Vector, error)
Dimensions() int
```

## 📖 Documentation

- [Vector Store Guide](docs/vector-stores.md)
- [State Store Guide](docs/state-stores.md)
- [Embedding Models Guide](docs/embeddings.md)
- [Migration Guide](docs/migration.md)

## 🤝 Contributing

See [CONTRIBUTING.md](../CONTRIBUTING.md) for guidelines.

## 📝 License

MIT License - see [LICENSE](../LICENSE) for details.

---

**Built with ❤️ by the Forge Team**

