# Redis State Store

Redis-based StateStore implementation for agent state persistence using the official Go Redis client.

## ✅ Features

- **Production Ready**: Battle-tested Redis for state storage
- **High Availability**: Supports Sentinel and Cluster modes
- **Fast**: Sub-millisecond reads/writes
- **Scalable**: Horizontal scaling with Redis Cluster
- **Session Management**: Built-in session listing per agent
- **Connection Pooling**: Efficient connection management

## 🚀 Installation

```bash
# Install integration
go get github.com/xraph/ai-sdk/integrations/statestores/redis

# Run Redis locally
docker run -d -p 6379:6379 redis:latest
```

## 📖 Usage

### Basic Example

```go
package main

import (
    "context"
    
    sdk "github.com/xraph/ai-sdk"
    "github.com/xraph/ai-sdk/integrations/statestores/redis"
)

func main() {
    ctx := context.Background()
    
    // Create store
    store, err := redis.NewRedisStateStore(ctx, redis.Config{
        Addrs: []string{"localhost:6379"},
    })
    if err != nil {
        panic(err)
    }
    defer store.Close()
    
    // Save state
    state := &sdk.AgentState{
        AgentID:   "agent-1",
        SessionID: "session-123",
        Messages:  []sdk.Message{...},
        Context:   map[string]any{"user_id": "user-456"},
    }
    err = store.Save(ctx, state)
    
    // Load state
    loaded, err := store.Load(ctx, "agent-1", "session-123")
    
    // List sessions
    sessions, err := store.List(ctx, "agent-1")
    
    // Delete state
    err = store.Delete(ctx, "agent-1", "session-123")
}
```

### Redis Cluster

```go
store, err := redis.NewRedisStateStore(ctx, redis.Config{
    Addrs: []string{
        "redis-node1:6379",
        "redis-node2:6379",
        "redis-node3:6379",
    },
    Password: os.Getenv("REDIS_PASSWORD"),
})
```

### Redis Sentinel

```go
store, err := redis.NewRedisStateStore(ctx, redis.Config{
    MasterName: "mymaster",
    SentinelAddrs: []string{
        "sentinel1:26379",
        "sentinel2:26379",
        "sentinel3:26379",
    },
    Password: os.Getenv("REDIS_PASSWORD"),
})
```

## 🔧 Configuration

### Config Options

```go
type Config struct {
    // Required
    Addrs    []string  // Redis addresses
    
    // Optional
    Password       string        // Redis password
    DB             int           // Database number (default: 0)
    KeyPrefix      string        // Key prefix (default: "forge:state:")
    DialTimeout    time.Duration // Dial timeout (default: 5s)
    ReadTimeout    time.Duration // Read timeout (default: 3s)
    WriteTimeout   time.Duration // Write timeout (default: 3s)
    PoolSize       int           // Connection pool size (default: 10)
    
    // Cluster/Sentinel
    MasterName     string   // Sentinel master name
    SentinelAddrs  []string // Sentinel addresses
    
    // Observability
    Logger  logger.Logger
    Metrics metrics.Metrics
}
```

## 📊 Performance

### Benchmarks

On Redis 7 with localhost connection:

| Operation | Latency | Throughput |
|-----------|---------|------------|
| Save | <1ms | 10K ops/sec |
| Load | <1ms | 15K ops/sec |
| Delete | <1ms | 12K ops/sec |
| List | 1-5ms | 5K ops/sec |

## 📈 Metrics

When metrics are enabled:

| Metric | Type | Description |
|--------|------|-------------|
| `forge.integrations.redis.state.save` | Counter | States saved |
| `forge.integrations.redis.state.load` | Counter | States loaded |
| `forge.integrations.redis.state.delete` | Counter | States deleted |
| `forge.integrations.redis.state.list` | Counter | Session lists retrieved |
| `forge.integrations.redis.state.save_duration` | Histogram | Save latency (seconds) |
| `forge.integrations.redis.state.load_duration` | Histogram | Load latency (seconds) |

## 🧪 Testing

```bash
# Unit tests
go test ./...

# With Redis (Docker)
docker run -d -p 6379:6379 redis:latest
go test ./...
```

## 📝 License

MIT License - see [LICENSE](../../../LICENSE) for details.

