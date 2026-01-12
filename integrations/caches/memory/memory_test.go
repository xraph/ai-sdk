package memory

import (
	"context"
	"testing"
	"time"
)

func TestNewMemoryCacheStore(t *testing.T) {
	store := NewMemoryCacheStore(Config{})
	if store == nil {
		t.Fatal("expected non-nil store")
	}
	defer store.Close()

	if store.Size() != 0 {
		t.Errorf("new store should be empty, got %d entries", store.Size())
	}
}

func TestMemoryCacheStore_SetGet(t *testing.T) {
	store := NewMemoryCacheStore(Config{})
	defer store.Close()
	ctx := context.Background()

	// Set value
	err := store.Set(ctx, "test-key", []byte("test-value"), 0)
	if err != nil {
		t.Errorf("Set() error = %v", err)
	}

	// Get value
	value, found, err := store.Get(ctx, "test-key")
	if err != nil {
		t.Errorf("Get() error = %v", err)
	}
	if !found {
		t.Error("Get() expected key to be found")
	}
	if string(value) != "test-value" {
		t.Errorf("Get() = %s, want test-value", string(value))
	}
}

func TestMemoryCacheStore_GetNonExistent(t *testing.T) {
	store := NewMemoryCacheStore(Config{})
	defer store.Close()
	ctx := context.Background()

	_, found, err := store.Get(ctx, "non-existent")
	if err != nil {
		t.Errorf("Get() error = %v", err)
	}
	if found {
		t.Error("Get() expected key not to be found")
	}
}

func TestMemoryCacheStore_Delete(t *testing.T) {
	store := NewMemoryCacheStore(Config{})
	defer store.Close()
	ctx := context.Background()

	// Set then delete
	_ = store.Set(ctx, "test-key", []byte("test-value"), 0)
	err := store.Delete(ctx, "test-key")
	if err != nil {
		t.Errorf("Delete() error = %v", err)
	}

	// Verify deleted
	_, found, _ := store.Get(ctx, "test-key")
	if found {
		t.Error("Get() after Delete() should not find key")
	}
}

func TestMemoryCacheStore_Clear(t *testing.T) {
	store := NewMemoryCacheStore(Config{})
	defer store.Close()
	ctx := context.Background()

	// Set multiple keys
	_ = store.Set(ctx, "key1", []byte("val1"), 0)
	_ = store.Set(ctx, "key2", []byte("val2"), 0)
	_ = store.Set(ctx, "key3", []byte("val3"), 0)

	if store.Size() != 3 {
		t.Errorf("expected 3 entries, got %d", store.Size())
	}

	// Clear all
	err := store.Clear(ctx)
	if err != nil {
		t.Errorf("Clear() error = %v", err)
	}

	if store.Size() != 0 {
		t.Errorf("expected 0 entries after clear, got %d", store.Size())
	}
}

func TestMemoryCacheStore_TTL(t *testing.T) {
	store := NewMemoryCacheStore(Config{})
	defer store.Close()
	ctx := context.Background()

	// Set with short TTL
	err := store.Set(ctx, "ttl-key", []byte("ttl-value"), 100*time.Millisecond)
	if err != nil {
		t.Errorf("Set() error = %v", err)
	}

	// Should exist immediately
	_, found, _ := store.Get(ctx, "ttl-key")
	if !found {
		t.Error("Get() expected key to exist before TTL")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should be expired
	_, found, _ = store.Get(ctx, "ttl-key")
	if found {
		t.Error("Get() expected key to be expired after TTL")
	}
}

func TestMemoryCacheStore_LRUEviction(t *testing.T) {
	store := NewMemoryCacheStore(Config{
		MaxSize: 3, // Small cache to test eviction
	})
	defer store.Close()
	ctx := context.Background()

	// Fill cache
	_ = store.Set(ctx, "key1", []byte("val1"), 0)
	_ = store.Set(ctx, "key2", []byte("val2"), 0)
	_ = store.Set(ctx, "key3", []byte("val3"), 0)

	// Add one more - should evict LRU
	_ = store.Set(ctx, "key4", []byte("val4"), 0)

	// key1 should be evicted (least recently used)
	_, found, _ := store.Get(ctx, "key1")
	if found {
		t.Error("Get() expected key1 to be evicted")
	}

	// Others should exist
	_, found2, _ := store.Get(ctx, "key2")
	_, found3, _ := store.Get(ctx, "key3")
	_, found4, _ := store.Get(ctx, "key4")

	if !found2 || !found3 || !found4 {
		t.Error("Get() expected key2, key3, key4 to exist")
	}
}

func TestMemoryCacheStore_Update(t *testing.T) {
	store := NewMemoryCacheStore(Config{})
	defer store.Close()
	ctx := context.Background()

	// Set initial value
	_ = store.Set(ctx, "key", []byte("value1"), 0)

	// Update value
	_ = store.Set(ctx, "key", []byte("value2"), 0)

	// Get updated value
	value, found, _ := store.Get(ctx, "key")
	if !found {
		t.Error("Get() expected key to exist")
	}
	if string(value) != "value2" {
		t.Errorf("Get() = %s, want value2", string(value))
	}
}

func TestMemoryCacheStore_Concurrent(t *testing.T) {
	store := NewMemoryCacheStore(Config{
		MaxSize: 100,
	})
	defer store.Close()
	ctx := context.Background()

	// Concurrent writes
	const goroutines = 10
	done := make(chan bool, goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			for j := 0; j < 10; j++ {
				key := string(rune('a' + id))
				_ = store.Set(ctx, key, []byte("value"), 0)
				_, _, _ = store.Get(ctx, key)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < goroutines; i++ {
		<-done
	}

	// Should have some entries
	if store.Size() == 0 {
		t.Error("expected non-zero size after concurrent operations")
	}
}

func TestMemoryCacheStore_Size(t *testing.T) {
	store := NewMemoryCacheStore(Config{})
	defer store.Close()
	ctx := context.Background()

	if store.Size() != 0 {
		t.Errorf("Size() = %d, want 0", store.Size())
	}

	_ = store.Set(ctx, "key1", []byte("val1"), 0)
	_ = store.Set(ctx, "key2", []byte("val2"), 0)

	if store.Size() != 2 {
		t.Errorf("Size() = %d, want 2", store.Size())
	}

	_ = store.Delete(ctx, "key1")

	if store.Size() != 1 {
		t.Errorf("Size() = %d, want 1", store.Size())
	}
}

