# Integration Examples

Comprehensive examples demonstrating how to use Forge AI SDK integrations.

## 📂 Examples

### Vector Stores

**File**: `vectorstores/main.go`

Demonstrates all vector store implementations:
- In-memory (for testing)
- pgvector (PostgreSQL)
- Qdrant
- Pinecone

```bash
cd vectorstores
go run main.go
```

### Complete RAG System

**File**: `complete_rag/main.go`

Full RAG implementation using:
- OpenAI embeddings
- Pinecone vector store
- Document indexing
- Semantic search
- Context retrieval

```bash
# Set environment variables
export OPENAI_API_KEY="your-key"
export PINECONE_API_KEY="your-key"
export PINECONE_INDEX="your-index"

cd complete_rag
go run main.go
```

## 🚀 Quick Start

### 1. In-Memory Vector Store (No Setup Required)

```bash
cd vectorstores
go run main.go
```

### 2. With Docker Services

Start required services:

```bash
# PostgreSQL with pgvector
docker run -d -p 5432:5432 \
  -e POSTGRES_PASSWORD=postgres \
  ankane/pgvector

# Qdrant
docker run -d -p 6333:6333 -p 6334:6334 \
  qdrant/qdrant

# Redis
docker run -d -p 6379:6379 \
  redis:latest
```

### 3. With Cloud Services

Set environment variables:

```bash
export OPENAI_API_KEY="sk-..."
export PINECONE_API_KEY="..."
export PINECONE_INDEX="your-index"
```

## 📚 What You'll Learn

- Setting up vector stores
- Generating embeddings
- Indexing documents
- Performing similarity search
- Building RAG systems
- Filtering and metadata
- Performance optimization

## 🔧 Configuration

Each example can be configured via:
- Environment variables
- Command-line flags (where applicable)
- Config files

See individual example READMEs for details.

## 🐛 Troubleshooting

### Connection Errors

Ensure services are running:

```bash
# Check PostgreSQL
psql -h localhost -U postgres -c "SELECT 1"

# Check Qdrant
curl http://localhost:6333/

# Check Redis
redis-cli ping
```

### API Key Issues

Verify environment variables:

```bash
echo $OPENAI_API_KEY
echo $PINECONE_API_KEY
```

## 📖 Next Steps

1. Start with in-memory examples
2. Try Docker-based services
3. Experiment with cloud services
4. Build your own RAG application

## 🤝 Contributing

Found an issue or have an example idea? Open an issue or PR!

## 📝 License

MIT License - see [LICENSE](../../LICENSE) for details.

