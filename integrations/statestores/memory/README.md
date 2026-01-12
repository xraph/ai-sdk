# Memory State Store

In-memory state store implementation for Forge AI SDK. Perfect for testing, development, and single-instance deployments.

## ✅ Features

- ✅ Pure Go implementation (no external dependencies)
- ✅ Thread-safe with RWMutex
- ✅ Optional TTL support with automatic cleanup
- ✅ Session management per agent
- ✅ Zero configuration required
- ✅ Observability (logging & metrics)
- ✅ Perfect for unit tests and local development

## 🚀 Installation

```bash
go get github.com/xraph/ai-sdk/integrations/statestores/memory
```

## 📖 Usage

### Basic Usage

```go
package main

import (
	"context"
	"log"

	"github.com/xraph/ai-sdk/integrations/statestores/memory"
	sdk "github.com/xraph/ai-sdk"
)

func main() {
	ctx := context.Background()

	// Create memory state store
	store := memory.NewMemoryStateStore(memory.Config{})

	// Save agent state
	state := &sdk.AgentState{
		AgentID:   "agent-1",
		SessionID: "session-1",
		Context: map[string]any{
			"user": "john",
			"step": 1,
		},
	}

	if err := store.Save(ctx, state); err != nil {
		log.Fatal(err)
	}

	// Load agent state
	loadedState, err := store.Load(ctx, "agent-1", "session-1")
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Loaded state for agent %s\n", loadedState.AgentID)

	// List all sessions for an agent
	sessions, err := store.List(ctx, "agent-1")
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Agent has %d sessions\n", len(sessions))
}
```

### With TTL (Auto-Expiration)

```go
import "time"

// Create store with 1-hour TTL
store := memory.NewMemoryStateStore(memory.Config{
	TTL: 1 * time.Hour,
})

// States will automatically expire after 1 hour
_ = store.Save(ctx, state)

// After TTL, Load() will return an error
time.Sleep(61 * time.Minute)
_, err := store.Load(ctx, "agent-1", "session-1")
// err != nil (state expired)
```

### For Testing

```go
func TestMyFeature(t *testing.T) {
	store := memory.NewMemoryStateStore(memory.Config{})
	defer store.Clear() // Clean up after test

	// Run tests...

	// Check state count
	if store.Count() != expectedCount {
		t.Errorf("unexpected state count: %d", store.Count())
	}
}
```

## 🔧 Configuration

```go
type Config struct {
	TTL     time.Duration // Optional: TTL for states (0 = no expiration)
	Logger  logger.Logger // Optional: Logger for debugging
	Metrics metrics.Metrics // Optional: Metrics for monitoring
}
```

## 📊 API

### Save

Saves an agent state to memory.

```go
func (m *MemoryStateStore) Save(ctx context.Context, state *sdk.AgentState) error
```

### Load

Loads an agent state from memory.

```go
func (m *MemoryStateStore) Load(ctx context.Context, agentID, sessionID string) (*sdk.AgentState, error)
```

### Delete

Deletes an agent state.

```go
func (m *MemoryStateStore) Delete(ctx context.Context, agentID, sessionID string) error
```

### List

Lists all session IDs for an agent.

```go
func (m *MemoryStateStore) List(ctx context.Context, agentID string) ([]string, error)
```

### Clear

Clears all states (useful for testing).

```go
func (m *MemoryStateStore) Clear()
```

### Count

Returns the total number of states.

```go
func (m *MemoryStateStore) Count() int
```

## 🧪 Testing

```bash
go test ./...
go test -race ./...  # Test for race conditions
go test -bench=. ./... # Run benchmarks
```

## ⚠️ Production Considerations

**NOT RECOMMENDED for production distributed systems:**

- ❌ State is lost on restart (no persistence)
- ❌ No replication or high availability
- ❌ Limited to single instance memory
- ❌ No cross-process sharing

**Suitable for:**

- ✅ Unit and integration tests
- ✅ Local development
- ✅ Single-instance applications
- ✅ Prototyping and demos

For production, use:
- [Redis StateStore](../redis/) - Distributed, persistent
- [PostgreSQL StateStore](../postgres/) - Relational, ACID
- [DynamoDB StateStore](../dynamodb/) - Serverless, scalable

## 📝 License

MIT License - see [LICENSE](../../../LICENSE)

