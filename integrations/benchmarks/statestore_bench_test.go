package benchmarks

import (
	"context"
	"fmt"
	"testing"

	sdk "github.com/xraph/ai-sdk"
	"github.com/xraph/ai-sdk/integrations/statestores/memory"
)

// BenchmarkStateStore_Memory benchmarks the in-memory state store.
func BenchmarkStateStore_Memory(b *testing.B) {
	store := memory.NewMemoryStateStore(memory.Config{})

	b.Run("Save", func(b *testing.B) {
		state := &sdk.AgentState{
			AgentID:   "bench-agent",
			SessionID: "bench-session",
			Context: map[string]any{
				"key1": "value1",
				"key2": 123,
				"key3": true,
			},
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = store.Save(context.Background(), state)
		}
	})

	b.Run("Save/LargeState", func(b *testing.B) {
		largeContext := make(map[string]any)
		for i := 0; i < 1000; i++ {
			largeContext[fmt.Sprintf("key%d", i)] = fmt.Sprintf("value%d", i)
		}
		state := &sdk.AgentState{
			AgentID:   "bench-agent",
			SessionID: "bench-session",
			Context:   largeContext,
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = store.Save(context.Background(), state)
		}
	})

	// Prepopulate for load tests
	for i := 0; i < 100; i++ {
		state := &sdk.AgentState{
			AgentID:   "bench-agent",
			SessionID: fmt.Sprintf("session-%d", i),
			Context:   map[string]any{"index": i},
		}
		_ = store.Save(context.Background(), state)
	}

	b.Run("Load", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = store.Load(context.Background(), "bench-agent", "session-0")
		}
	})

	b.Run("List", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = store.List(context.Background(), "bench-agent")
		}
	})

	b.Run("Delete", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			sessionID := fmt.Sprintf("delete-session-%d", i)
			state := &sdk.AgentState{
				AgentID:   "bench-agent",
				SessionID: sessionID,
			}
			_ = store.Save(context.Background(), state)
			b.StartTimer()
			_ = store.Delete(context.Background(), "bench-agent", sessionID)
		}
	})
}

// BenchmarkStateStore_Concurrent tests concurrent access patterns.
func BenchmarkStateStore_Concurrent(b *testing.B) {
	store := memory.NewMemoryStateStore(memory.Config{})

	b.Run("Save/Parallel", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				state := &sdk.AgentState{
					AgentID:   fmt.Sprintf("agent-%d", i%10),
					SessionID: fmt.Sprintf("session-%d", i),
				}
				_ = store.Save(context.Background(), state)
				i++
			}
		})
	})

	b.Run("Load/Parallel", func(b *testing.B) {
		// Prepopulate
		for i := 0; i < 100; i++ {
			state := &sdk.AgentState{
				AgentID:   fmt.Sprintf("agent-%d", i%10),
				SessionID: fmt.Sprintf("session-%d", i),
			}
			_ = store.Save(context.Background(), state)
		}

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				_, _ = store.Load(context.Background(),
					fmt.Sprintf("agent-%d", i%10),
					fmt.Sprintf("session-%d", i%100))
				i++
			}
		})
	})
}
