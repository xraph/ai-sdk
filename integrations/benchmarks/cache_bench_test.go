package benchmarks

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/xraph/ai-sdk/integrations/caches/memory"
)

// BenchmarkCacheStore_Memory benchmarks the in-memory cache store.
func BenchmarkCacheStore_Memory(b *testing.B) {
	store := memory.NewMemoryCacheStore(memory.Config{
		MaxSize: 10000,
	})
	defer func() { _ = store.Close() }()

	b.Run("Set", func(b *testing.B) {
		value := []byte("test-value")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = store.Set(context.Background(), fmt.Sprintf("key-%d", i), value, 0)
		}
	})

	b.Run("Set/LargeValue", func(b *testing.B) {
		largeValue := make([]byte, 10240) // 10KB
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = store.Set(context.Background(), fmt.Sprintf("key-%d", i), largeValue, 0)
		}
	})

	b.Run("Set/WithTTL", func(b *testing.B) {
		value := []byte("test-value")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = store.Set(context.Background(), fmt.Sprintf("key-%d", i), value, 5*time.Minute)
		}
	})

	// Prepopulate for get tests
	for i := 0; i < 1000; i++ {
		_ = store.Set(context.Background(), fmt.Sprintf("key-%d", i), []byte("value"), 0)
	}

	b.Run("Get/Hit", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _, _ = store.Get(context.Background(), fmt.Sprintf("key-%d", i%1000))
		}
	})

	b.Run("Get/Miss", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _, _ = store.Get(context.Background(), fmt.Sprintf("miss-key-%d", i))
		}
	})

	b.Run("Delete", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			key := fmt.Sprintf("delete-key-%d", i)
			_ = store.Set(context.Background(), key, []byte("value"), 0)
			b.StartTimer()
			_ = store.Delete(context.Background(), key)
		}
	})

	b.Run("Clear", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			// Add some data
			for j := 0; j < 100; j++ {
				_ = store.Set(context.Background(), fmt.Sprintf("key-%d", j), []byte("value"), 0)
			}
			b.StartTimer()
			_ = store.Clear(context.Background())
		}
	})
}

// BenchmarkCacheStore_HitRatio tests cache hit ratio under different scenarios.
func BenchmarkCacheStore_HitRatio(b *testing.B) {
	store := memory.NewMemoryCacheStore(memory.Config{
		MaxSize: 100, // Small cache to test LRU
	})
	defer func() { _ = store.Close() }()

	b.Run("HitRatio/80Percent", func(b *testing.B) {
		// Prepopulate
		for i := 0; i < 80; i++ {
			_ = store.Set(context.Background(), fmt.Sprintf("key-%d", i), []byte("value"), 0)
		}

		b.ResetTimer()
		hits := 0
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("key-%d", i%100) // 80% hit rate
			_, found, _ := store.Get(context.Background(), key)
			if found {
				hits++
			}
		}
		b.ReportMetric(float64(hits)/float64(b.N)*100, "hit%")
	})
}

// BenchmarkCacheStore_Concurrent tests concurrent cache access.
func BenchmarkCacheStore_Concurrent(b *testing.B) {
	store := memory.NewMemoryCacheStore(memory.Config{
		MaxSize: 10000,
	})
	defer func() { _ = store.Close() }()

	// Prepopulate
	for i := 0; i < 1000; i++ {
		_ = store.Set(context.Background(), fmt.Sprintf("key-%d", i), []byte("value"), 0)
	}

	b.Run("Get/Parallel", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				_, _, _ = store.Get(context.Background(), fmt.Sprintf("key-%d", i%1000))
				i++
			}
		})
	})

	b.Run("Set/Parallel", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				_ = store.Set(context.Background(), fmt.Sprintf("key-%d", i), []byte("value"), 0)
				i++
			}
		})
	})

	b.Run("Mixed/Parallel", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				if i%2 == 0 {
					_, _, _ = store.Get(context.Background(), fmt.Sprintf("key-%d", i%1000))
				} else {
					_ = store.Set(context.Background(), fmt.Sprintf("key-%d", i), []byte("value"), 0)
				}
				i++
			}
		})
	})
}
