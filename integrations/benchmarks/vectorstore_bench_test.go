package benchmarks

import (
	"context"
	"fmt"
	"math/rand"
	"testing"

	sdk "github.com/xraph/ai-sdk"
	"github.com/xraph/ai-sdk/integrations/vectorstores/memory"
)

// generateBenchVectors creates test vectors for benchmarking.
func generateBenchVectors(count, dimensions int) []sdk.Vector {
	vectors := make([]sdk.Vector, count)
	for i := 0; i < count; i++ {
		values := make([]float64, dimensions)
		for j := 0; j < dimensions; j++ {
			values[j] = rand.Float64()
		}
		vectors[i] = sdk.Vector{
			ID:     fmt.Sprintf("bench-vec-%d", i),
			Values: values,
			Metadata: map[string]any{
				"index": i,
				"batch": i / 100,
			},
		}
	}
	return vectors
}

// BenchmarkVectorStore_Memory benchmarks the in-memory vector store.
func BenchmarkVectorStore_Memory(b *testing.B) {
	store, _ := memory.NewInMemoryVectorStore(memory.InMemoryConfig{
		Dimensions: 1536,
	})

	b.Run("Upsert/Batch10", func(b *testing.B) {
		vectors := generateBenchVectors(10, 1536)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = store.Upsert(context.Background(), vectors)
		}
	})

	b.Run("Upsert/Batch100", func(b *testing.B) {
		vectors := generateBenchVectors(100, 1536)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = store.Upsert(context.Background(), vectors)
		}
	})

	b.Run("Upsert/Batch1000", func(b *testing.B) {
		vectors := generateBenchVectors(1000, 1536)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = store.Upsert(context.Background(), vectors)
		}
	})

	// Prepopulate for query tests
	vectors := generateBenchVectors(10000, 1536)
	_ = store.Upsert(context.Background(), vectors)

	b.Run("Query/Limit5", func(b *testing.B) {
		queryVec := generateBenchVectors(1, 1536)[0].Values
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = store.Query(context.Background(), queryVec, 5, nil)
		}
	})

	b.Run("Query/Limit10", func(b *testing.B) {
		queryVec := generateBenchVectors(1, 1536)[0].Values
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = store.Query(context.Background(), queryVec, 10, nil)
		}
	})

	b.Run("Query/Limit50", func(b *testing.B) {
		queryVec := generateBenchVectors(1, 1536)[0].Values
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = store.Query(context.Background(), queryVec, 50, nil)
		}
	})

	b.Run("Query/WithFilter", func(b *testing.B) {
		queryVec := generateBenchVectors(1, 1536)[0].Values
		filter := map[string]any{"batch": 0}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = store.Query(context.Background(), queryVec, 10, filter)
		}
	})

	b.Run("Delete/Batch10", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			ids := make([]string, 10)
			for j := 0; j < 10; j++ {
				ids[j] = fmt.Sprintf("bench-vec-%d", i*10+j)
			}
			b.StartTimer()
			_ = store.Delete(context.Background(), ids)
		}
	})

	b.Run("Delete/Batch100", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			ids := make([]string, 100)
			for j := 0; j < 100; j++ {
				ids[j] = fmt.Sprintf("bench-vec-%d", i*100+j)
			}
			b.StartTimer()
			_ = store.Delete(context.Background(), ids)
		}
	})
}

// Note: Additional vector store benchmarks (pgvector, qdrant, pinecone, etc.)
// would follow the same pattern. They are commented out as they require
// running services. Use integration tests with -bench flag for those.

