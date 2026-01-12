# Benchmarks

Comprehensive performance benchmarks for all Forge AI SDK integrations.

## Overview

This package provides benchmarks for:
- **Vector Stores**: Memory, pgvector, Qdrant, Pinecone, Weaviate, ChromaDB
- **State Stores**: Memory, PostgreSQL, Redis
- **Cache Stores**: Memory, Redis
- **Embeddings**: OpenAI, Cohere, HuggingFace, Ollama

## Running Benchmarks

### All Benchmarks

```bash
go test -bench=. -benchmem ./integrations/benchmarks/...
```

### Specific Category

```bash
# Vector stores
go test -bench=BenchmarkVectorStore -benchmem ./integrations/benchmarks/...

# State stores
go test -bench=BenchmarkStateStore -benchmem ./integrations/benchmarks/...

# Caches
go test -bench=BenchmarkCacheStore -benchmem ./integrations/benchmarks/...

# Embeddings (requires API keys)
go test -bench=BenchmarkEmbeddings -benchmem ./integrations/benchmarks/...
```

### With Custom Duration

```bash
# Run each benchmark for 10 seconds
go test -bench=. -benchtime=10s -benchmem ./integrations/benchmarks/...

# Run each benchmark 1000 times
go test -bench=. -benchtime=1000x -benchmem ./integrations/benchmarks/...
```

### Save Results

```bash
# Save to file
go test -bench=. -benchmem ./integrations/benchmarks/... | tee bench.txt

# Generate markdown report
go test -bench=. -benchmem ./integrations/benchmarks/... > bench.txt
# Then use report generator (see below)
```

## Benchmark Categories

### Vector Store Benchmarks

Tests performed:
- **Upsert**: Batch sizes of 10, 100, 1000 vectors (1536 dimensions)
- **Query**: Retrieve top 5, 10, 50 results
- **Query with Filter**: Metadata filtering performance
- **Delete**: Batch deletion of 10, 100 vectors

**Example output**:
```
BenchmarkVectorStore_Memory/Upsert/Batch10-8      50000    25341 ns/op    12345 B/op    123 allocs/op
BenchmarkVectorStore_Memory/Query/Limit5-8       100000    10234 ns/op     1234 B/op     12 allocs/op
```

### State Store Benchmarks

Tests performed:
- **Save**: Small and large state objects
- **Load**: Single state retrieval
- **List**: List all sessions for an agent
- **Delete**: State deletion
- **Concurrent**: Parallel read/write operations

**Example output**:
```
BenchmarkStateStore_Memory/Save-8                200000     5123 ns/op     2048 B/op     15 allocs/op
BenchmarkStateStore_Memory/Load-8                300000     3456 ns/op      512 B/op      8 allocs/op
```

### Cache Store Benchmarks

Tests performed:
- **Set**: With and without TTL, various value sizes
- **Get**: Hit and miss scenarios
- **Delete**: Entry deletion
- **Clear**: Full cache clear
- **Hit Ratio**: 80% hit rate test
- **Concurrent**: Parallel access patterns

**Example output**:
```
BenchmarkCacheStore_Memory/Get/Hit-8             500000     2341 ns/op      256 B/op      4 allocs/op
BenchmarkCacheStore_Memory/HitRatio/80Percent-8  100000    12345 ns/op         hit%:80.12
```

### Embeddings Benchmarks

Tests performed:
- **Embed/Single**: Single text embedding
- **Embed/Batch**: 3, 10, 100 texts at once
- **Comparison**: Multiple providers side-by-side

**Note**: Embeddings benchmarks make real API calls and incur costs. Run with caution:

```bash
export OPENAI_API_KEY="your-key"
export COHERE_API_KEY="your-key"
go test -bench=BenchmarkEmbeddings -benchtime=10x ./integrations/benchmarks/...
```

## Generating Reports

### Markdown Report

```bash
# Run benchmarks and save output
go test -bench=. -benchmem ./integrations/benchmarks/... > bench.txt

# Use the report generator (manual process - see report.go for parsing logic)
```

### Compare Results

```bash
# Save baseline
go test -bench=. -benchmem ./integrations/benchmarks/... > baseline.txt

# Make changes...

# Run again and compare
go test -bench=. -benchmem ./integrations/benchmarks/... > current.txt
benchstat baseline.txt current.txt
```

**Install benchstat**:
```bash
go install golang.org/x/perf/cmd/benchstat@latest
```

## Interpreting Results

### Key Metrics

- **ns/op**: Nanoseconds per operation (lower is better)
- **B/op**: Bytes allocated per operation (lower is better)
- **allocs/op**: Number of allocations per operation (lower is better)
- **MB/s**: Throughput in megabytes per second (higher is better)

### Relative Performance

```
Operation A: 1000 ns/op
Operation B: 2000 ns/op
```
→ Operation A is 2x faster than Operation B

### What to Optimize

**Critical path optimization**:
1. High frequency operations (query, get, load)
2. Operations with high allocation counts
3. Operations in hot paths

**Less critical**:
- Infrequent operations (initialization, cleanup)
- Admin operations (clear, list all)

## Example Comparison Tables

### Vector Store Comparison (Query Performance)

| Store | ns/op (Top 10) | Relative Speed |
|-------|---------------|----------------|
| Memory | 10,234 | 1.00x (baseline) |
| Qdrant | 15,678 | 1.53x slower |
| pgvector | 18,901 | 1.85x slower |
| ChromaDB | 22,345 | 2.18x slower |

*Lower is better. Times are for local containers.*

### State Store Comparison (Save Performance)

| Store | ns/op | Relative Speed |
|-------|-------|----------------|
| Memory | 5,123 | 1.00x (baseline) |
| Redis | 8,456 | 1.65x slower |
| PostgreSQL | 12,789 | 2.50x slower |

*Lower is better. Times are for local services.*

### Cache Store Comparison (Get Hit Performance)

| Store | ns/op | Relative Speed |
|-------|-------|----------------|
| Memory | 2,341 | 1.00x (baseline) |
| Redis | 6,789 | 2.90x slower |

*Lower is better. Times are for local services.*

## Best Practices

### 1. Run Multiple Times

```bash
# Run 10 times and take average
for i in {1..10}; do
  go test -bench=. -benchmem ./integrations/benchmarks/... >> results.txt
done
```

### 2. Control Environment

- Close unnecessary applications
- Disable power saving modes
- Use consistent hardware
- Run on same machine for comparisons

### 3. Warm Up

Many benchmarks automatically warm up, but for critical measurements:

```bash
# Run twice, discard first run
go test -bench=. -benchtime=1x ./integrations/benchmarks/... > /dev/null
go test -bench=. -benchmem ./integrations/benchmarks/...
```

### 4. Profile Bottlenecks

```bash
# CPU profile
go test -bench=. -cpuprofile=cpu.prof ./integrations/benchmarks/...
go tool pprof cpu.prof

# Memory profile
go test -bench=. -memprofile=mem.prof ./integrations/benchmarks/...
go tool pprof mem.prof
```

## CI/CD Integration

### GitHub Actions

```yaml
name: Benchmarks

on:
  push:
    branches: [main]
  pull_request:

jobs:
  benchmark:
    runs-on: ubuntu-latest
    
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Run Benchmarks
        run: |
          go test -bench=. -benchmem ./integrations/benchmarks/... | tee bench.txt
          
      - name: Upload Results
        uses: actions/upload-artifact@v3
        with:
          name: benchmark-results
          path: bench.txt
```

### Compare PR vs Main

```yaml
      - name: Benchmark Comparison
        run: |
          git fetch origin main
          git checkout main
          go test -bench=. -benchmem ./integrations/benchmarks/... > main.txt
          git checkout -
          go test -bench=. -benchmem ./integrations/benchmarks/... > pr.txt
          benchstat main.txt pr.txt
```

## Troubleshooting

### Unstable Results

**Issue**: Results vary significantly between runs

**Solutions**:
- Increase `-benchtime` (e.g., `10s` or `1000x`)
- Check for background processes
- Use `benchstat` to compare multiple runs
- Run on dedicated hardware

### Memory Issues

**Issue**: Out of memory errors

**Solutions**:
- Reduce batch sizes in benchmarks
- Increase available memory
- Run benchmarks individually

### API Rate Limits

**Issue**: Embedding benchmarks fail with rate limits

**Solutions**:
- Reduce `-benchtime`
- Add delays between requests
- Use higher tier API keys
- Mock API responses for pure performance testing

## Additional Resources

- [Go Benchmark Guide](https://go.dev/doc/diagnostics#profiling)
- [benchstat Tool](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat)
- [Performance Best Practices](https://github.com/golang/go/wiki/Performance)

## Contributing

When adding new benchmarks:
1. Follow naming convention: `Benchmark<Category>_<Integration>`
2. Include memory benchmarking (`-benchmem`)
3. Test various batch sizes/scenarios
4. Document any special requirements (API keys, services)
5. Update this README

## License

MIT License - see [LICENSE](../../LICENSE)

