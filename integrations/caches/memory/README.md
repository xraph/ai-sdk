# Memory Cache Store

In-memory cache store implementation with LRU eviction for Forge AI SDK.

## ✅ Features

- ✅ Pure Go implementation (no external dependencies)
- ✅ LRU (Least Recently Used) eviction policy
- ✅ TTL (Time To Live) support with automatic cleanup
- ✅ Thread-safe with RWMutex
- ✅ Size limits with automatic eviction
- ✅ Zero configuration required
- ✅ Observability (logging & metrics)
- ✅ Perfect for testing and local development

## 🚀 Installation

```bash
go get github.com/xraph/ai-sdk/integrations/caches/memory
```

## 📖 Usage

### Basic Usage

```go
package main

import (
	"context"
	"log"

	"github.com/xraph/ai-sdk/integrations/caches/memory"
)

func main() {
	ctx := context.Background()

	// Create memory cache store
	cache := memory.NewMemoryCacheStore(memory.Config{
		MaxSize: 1000, // Maximum 1000 entries
	})
	defer cache.Close()

	// Set value
	err := cache.Set(ctx, "user:123", []byte(`{"name":"John"}`), 0)
	if err != nil {
		log.Fatal(err)
	}

	// Get value
	value, found, err := cache.Get(ctx, "user:123")
	if err != nil {
		log.Fatal(err)
	}
	if found {
		log.Printf("Found: %s\n", string(value))
	}

	// Delete value
	err = cache.Delete(ctx, "user:123")
	if err != nil {
		log.Fatal(err)
	}
}
```

### With TTL (Auto-Expiration)

```go
import "time"

// Cache with 5-minute TTL
cache := memory.NewMemoryCacheStore(memory.Config{
	MaxSize: 1000,
})

// Set with expiration
err := cache.Set(ctx, "session:abc", []byte("data"), 5*time.Minute)

// After 5 minutes, the entry will be automatically removed
```

### LRU Eviction

```go
// Small cache to demonstrate LRU
cache := memory.NewMemoryCacheStore(memory.Config{
	MaxSize: 3, // Only 3 entries max
})

// Add 4 entries - oldest will be evicted
cache.Set(ctx, "key1", []byte("val1"), 0)
cache.Set(ctx, "key2", []byte("val2"), 0)
cache.Set(ctx, "key3", []byte("val3"), 0)
cache.Set(ctx, "key4", []byte("val4"), 0) // key1 evicted

// key1 no longer exists
_, found, _ := cache.Get(ctx, "key1") // found = false
```

### For Testing

```go
func TestMyFeature(t *testing.T) {
	cache := memory.NewMemoryCacheStore(memory.Config{})
	defer cache.Close()

	// Use cache in tests...

	// Clear all entries
	cache.Clear(context.Background())

	// Check size
	if cache.Size() != 0 {
		t.Errorf("expected empty cache, got %d entries", cache.Size())
	}
}
```

## 🔧 Configuration

```go
type Config struct {
	MaxSize int           // Maximum number of entries (0 = unlimited)
	Logger  logger.Logger // Optional: Logger for debugging
	Metrics metrics.Metrics // Optional: Metrics for monitoring
}
```

## 📊 API

### Set

Sets a value in the cache with optional TTL.

```go
func (m *MemoryCacheStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
```

### Get

Retrieves a value from the cache.

```go
func (m *MemoryCacheStore) Get(ctx context.Context, key string) ([]byte, bool, error)
```

Returns: `value, found, error`

### Delete

Removes a value from the cache.

```go
func (m *MemoryCacheStore) Delete(ctx context.Context, key string) error
```

### Clear

Clears all entries from the cache.

```go
func (m *MemoryCacheStore) Clear(ctx context.Context) error
```

### Size

Returns the current number of entries.

```go
func (m *MemoryCacheStore) Size() int
```

### Close

Stops the cleanup goroutine and releases resources.

```go
func (m *MemoryCacheStore) Close() error
```

## 🧪 Testing

```bash
go test ./...
go test -race ./...  # Test for race conditions
go test -bench=. ./... # Run benchmarks
```

## 📈 Performance

| Operation | Latency | Notes |
|-----------|---------|-------|
| Set       | ~2μs    | O(1) with occasional O(n) on eviction |
| Get       | ~1μs    | O(1) lookup |
| Delete    | ~1μs    | O(1) removal |
| Clear     | ~10μs   | O(n) |

*Benchmarks on MacBook Pro M1, 1000 entries in cache*

## 🔍 Advanced Usage

### Custom Max Size

```go
cache := memory.NewMemoryCacheStore(memory.Config{
	MaxSize: 10000, // 10k entries max
})
```

### No Size Limit

```go
cache := memory.NewMemoryCacheStore(memory.Config{
	MaxSize: 0, // Unlimited (be careful with memory!)
})
```

### With Observability

```go
import (
	"github.com/xraph/go-utils/log"
	"github.com/xraph/go-utils/metrics"
)

cache := memory.NewMemoryCacheStore(memory.Config{
	MaxSize: 1000,
	Logger:  log.NewLogger(),
	Metrics: metrics.NewMetrics(),
})
```

## ⚠️ Production Considerations

### Pros
- ✅ Extremely fast (in-memory)
- ✅ No external dependencies
- ✅ Perfect for testing
- ✅ LRU eviction prevents unbounded growth
- ✅ TTL support for auto-expiration

### Cons
- ❌ Data lost on restart (no persistence)
- ❌ No replication or high availability
- ❌ Limited to single instance memory
- ❌ No cross-process sharing

### Suitable For:
- ✅ Unit and integration tests
- ✅ Local development
- ✅ Single-instance applications
- ✅ Session caching
- ✅ Rate limiting
- ✅ Temporary data storage

### For Production, Consider:
- [Redis CacheStore](../redis/) - Distributed, persistent
- External cache services (Memcached, etc.)

## 💡 Use Cases

### Session Cache

```go
cache := memory.NewMemoryCacheStore(memory.Config{
	MaxSize: 10000,
})

// Store session with 30-minute TTL
sessionData := []byte(`{"user_id":123,"role":"admin"}`)
cache.Set(ctx, "session:"+sessionID, sessionData, 30*time.Minute)
```

### Rate Limiting

```go
func checkRateLimit(userID string) bool {
	key := fmt.Sprintf("ratelimit:%s", userID)
	
	// Check current count
	data, found, _ := cache.Get(ctx, key)
	count := 0
	if found {
		json.Unmarshal(data, &count)
	}
	
	if count >= 100 {
		return false // Rate limit exceeded
	}
	
	// Increment count
	count++
	data, _ = json.Marshal(count)
	cache.Set(ctx, key, data, 1*time.Minute)
	
	return true
}
```

### Computed Results Cache

```go
func getExpensiveData(id string) ([]byte, error) {
	key := fmt.Sprintf("data:%s", id)
	
	// Check cache first
	if data, found, _ := cache.Get(ctx, key); found {
		return data, nil
	}
	
	// Compute expensive result
	data := computeExpensiveResult(id)
	
	// Cache for 10 minutes
	cache.Set(ctx, key, data, 10*time.Minute)
	
	return data, nil
}
```

## 🔧 Troubleshooting

### Memory Usage Growing

**Issue**: Cache consuming too much memory

**Solutions**:
- Set appropriate `MaxSize` limit
- Use shorter TTLs
- Clear cache periodically
- Monitor `Size()` metric

### LRU Evicting Too Aggressively

**Issue**: Frequently accessed items being evicted

**Solutions**:
- Increase `MaxSize`
- Reduce TTL for less important items
- Use dedicated cache instances for different data types

### Concurrent Access Issues

**Issue**: Data race warnings

**Solution**: The cache is already thread-safe. If seeing issues, check if:
- Cache instance is being shared correctly
- Not storing pointers to mutable data

## 📚 Resources

- [LRU Cache Algorithm](https://en.wikipedia.org/wiki/Cache_replacement_policies#Least_recently_used_(LRU))
- [Go sync.RWMutex](https://pkg.go.dev/sync#RWMutex)
- [Time-based expiration](https://en.wikipedia.org/wiki/Cache_expiration)

## 🤝 Contributing

Contributions welcome! See the main [CONTRIBUTING.md](../../../CONTRIBUTING.md).

## 📝 License

MIT License - see [LICENSE](../../../LICENSE)

