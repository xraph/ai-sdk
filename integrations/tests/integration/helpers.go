//go:build integration
// +build integration

package integration

import (
	"context"
	"fmt"
	"math/rand"
	"testing"

	sdk "github.com/xraph/ai-sdk"
)

// GenerateTestVectors creates random vectors for testing.
func GenerateTestVectors(count, dimensions int) []sdk.Vector {
	vectors := make([]sdk.Vector, count)
	for i := 0; i < count; i++ {
		values := make([]float64, dimensions)
		for j := 0; j < dimensions; j++ {
			values[j] = rand.Float64()
		}
		vectors[i] = sdk.Vector{
			ID:     fmt.Sprintf("test-vec-%d", i),
			Values: values,
			Metadata: map[string]any{
				"index": i,
				"test":  true,
			},
		}
	}
	return vectors
}

// GenerateTestVector creates a single random vector.
func GenerateTestVector(dimensions int) []float64 {
	values := make([]float64, dimensions)
	for i := 0; i < dimensions; i++ {
		values[i] = rand.Float64()
	}
	return values
}

// TestVectorStoreOperations runs a standard test suite for vector stores.
func TestVectorStoreOperations(t *testing.T, store sdk.VectorStore) {
	ctx := context.Background()

	// Test Upsert
	t.Run("Upsert", func(t *testing.T) {
		vectors := GenerateTestVectors(10, 3)
		err := store.Upsert(ctx, vectors)
		if err != nil {
			t.Fatalf("Upsert failed: %v", err)
		}
	})

	// Test Query
	t.Run("Query", func(t *testing.T) {
		queryVec := GenerateTestVector(3)
		results, err := store.Query(ctx, queryVec, 5, nil)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
		if len(results) == 0 {
			t.Error("Expected at least one result from query")
		}
		if len(results) > 5 {
			t.Errorf("Expected max 5 results, got %d", len(results))
		}
	})

	// Test Query with filter
	t.Run("QueryWithFilter", func(t *testing.T) {
		queryVec := GenerateTestVector(3)
		filter := map[string]any{"test": true}
		results, err := store.Query(ctx, queryVec, 5, filter)
		if err != nil {
			t.Fatalf("Query with filter failed: %v", err)
		}
		// Verify filter was applied (implementation-specific)
		for _, result := range results {
			if testVal, ok := result.Metadata["test"]; !ok || testVal != true {
				t.Error("Filter not applied correctly")
			}
		}
	})

	// Test Delete
	t.Run("Delete", func(t *testing.T) {
		ids := []string{"test-vec-0", "test-vec-1"}
		err := store.Delete(ctx, ids)
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}
	})
}

// TestStateStoreOperations runs a standard test suite for state stores.
func TestStateStoreOperations(t *testing.T, store sdk.StateStore) {
	ctx := context.Background()

	// Test Save
	t.Run("Save", func(t *testing.T) {
		state := &sdk.AgentState{
			AgentID:   "test-agent",
			SessionID: "test-session",
			Context: map[string]any{
				"key": "value",
			},
		}
		err := store.Save(ctx, state)
		if err != nil {
			t.Fatalf("Save failed: %v", err)
		}
	})

	// Test Load
	t.Run("Load", func(t *testing.T) {
		state, err := store.Load(ctx, "test-agent", "test-session")
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}
		if state.AgentID != "test-agent" {
			t.Errorf("Expected AgentID 'test-agent', got '%s'", state.AgentID)
		}
		if state.SessionID != "test-session" {
			t.Errorf("Expected SessionID 'test-session', got '%s'", state.SessionID)
		}
	})

	// Test List
	t.Run("List", func(t *testing.T) {
		// Save another session
		state2 := &sdk.AgentState{
			AgentID:   "test-agent",
			SessionID: "test-session-2",
		}
		_ = store.Save(ctx, state2)

		sessions, err := store.List(ctx, "test-agent")
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if len(sessions) < 1 {
			t.Error("Expected at least one session")
		}
	})

	// Test Delete
	t.Run("Delete", func(t *testing.T) {
		err := store.Delete(ctx, "test-agent", "test-session")
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		// Verify deletion
		_, err = store.Load(ctx, "test-agent", "test-session")
		if err == nil {
			t.Error("Expected error loading deleted state")
		}
	})
}

// TestCacheStoreOperations runs a standard test suite for cache stores.
func TestCacheStoreOperations(t *testing.T, store sdk.CacheStore) {
	ctx := context.Background()

	// Test Set
	t.Run("Set", func(t *testing.T) {
		err := store.Set(ctx, "test-key", []byte("test-value"), 0)
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}
	})

	// Test Get
	t.Run("Get", func(t *testing.T) {
		value, found, err := store.Get(ctx, "test-key")
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if !found {
			t.Error("Expected key to be found")
		}
		if string(value) != "test-value" {
			t.Errorf("Expected 'test-value', got '%s'", string(value))
		}
	})

	// Test Get non-existent
	t.Run("GetNonExistent", func(t *testing.T) {
		_, found, err := store.Get(ctx, "non-existent-key")
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if found {
			t.Error("Expected key not to be found")
		}
	})

	// Test Delete
	t.Run("Delete", func(t *testing.T) {
		err := store.Delete(ctx, "test-key")
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		// Verify deletion
		_, found, _ := store.Get(ctx, "test-key")
		if found {
			t.Error("Expected key to be deleted")
		}
	})

	// Test Clear
	t.Run("Clear", func(t *testing.T) {
		// Set some keys
		_ = store.Set(ctx, "key1", []byte("val1"), 0)
		_ = store.Set(ctx, "key2", []byte("val2"), 0)

		err := store.Clear(ctx)
		if err != nil {
			t.Fatalf("Clear failed: %v", err)
		}

		// Verify all cleared
		_, found1, _ := store.Get(ctx, "key1")
		_, found2, _ := store.Get(ctx, "key2")
		if found1 || found2 {
			t.Error("Expected all keys to be cleared")
		}
	})
}

