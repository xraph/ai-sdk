package memory

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdk "github.com/xraph/ai-sdk"
)

func TestMemoryVectorStore_Upsert(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryVectorStore(Config{})

	vectors := []sdk.Vector{
		{
			ID:     "vec1",
			Values: []float64{1.0, 0.0, 0.0},
			Metadata: map[string]any{
				"category": "test",
			},
		},
		{
			ID:     "vec2",
			Values: []float64{0.0, 1.0, 0.0},
			Metadata: map[string]any{
				"category": "test",
			},
		},
	}

	err := store.Upsert(ctx, vectors)
	require.NoError(t, err)
	assert.Equal(t, 2, store.Count())
}

func TestMemoryVectorStore_UpsertUpdate(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryVectorStore(Config{})

	// Insert
	err := store.Upsert(ctx, []sdk.Vector{
		{ID: "vec1", Values: []float64{1.0, 0.0}},
	})
	require.NoError(t, err)

	// Update same ID
	err = store.Upsert(ctx, []sdk.Vector{
		{ID: "vec1", Values: []float64{0.0, 1.0}},
	})
	require.NoError(t, err)

	assert.Equal(t, 1, store.Count())

	// Verify updated value
	results, err := store.Query(ctx, []float64{0.0, 1.0}, 1, nil)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "vec1", results[0].ID)
	assert.InDelta(t, 1.0, results[0].Score, 0.001)
}

func TestMemoryVectorStore_UpsertValidation(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryVectorStore(Config{})

	tests := []struct {
		name    string
		vectors []sdk.Vector
		wantErr bool
	}{
		{
			name:    "empty ID",
			vectors: []sdk.Vector{{ID: "", Values: []float64{1.0}}},
			wantErr: true,
		},
		{
			name:    "empty values",
			vectors: []sdk.Vector{{ID: "vec1", Values: []float64{}}},
			wantErr: true,
		},
		{
			name:    "nil values",
			vectors: []sdk.Vector{{ID: "vec1", Values: nil}},
			wantErr: true,
		},
		{
			name:    "empty list",
			vectors: []sdk.Vector{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := store.Upsert(ctx, tt.vectors)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMemoryVectorStore_Query(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryVectorStore(Config{})

	// Insert test vectors
	vectors := []sdk.Vector{
		{ID: "vec1", Values: []float64{1.0, 0.0, 0.0}},
		{ID: "vec2", Values: []float64{0.9, 0.1, 0.0}},
		{ID: "vec3", Values: []float64{0.0, 1.0, 0.0}},
		{ID: "vec4", Values: []float64{0.0, 0.0, 1.0}},
	}
	err := store.Upsert(ctx, vectors)
	require.NoError(t, err)

	// Query for similar to vec1
	query := []float64{1.0, 0.0, 0.0}
	results, err := store.Query(ctx, query, 2, nil)
	require.NoError(t, err)
	assert.Len(t, results, 2)

	// First result should be vec1 (exact match)
	assert.Equal(t, "vec1", results[0].ID)
	assert.InDelta(t, 1.0, results[0].Score, 0.001)

	// Second result should be vec2 (most similar)
	assert.Equal(t, "vec2", results[1].ID)
	assert.Greater(t, results[1].Score, 0.9)
}

func TestMemoryVectorStore_QueryWithFilter(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryVectorStore(Config{})

	// Insert vectors with metadata
	vectors := []sdk.Vector{
		{
			ID:       "vec1",
			Values:   []float64{1.0, 0.0},
			Metadata: map[string]any{"category": "A"},
		},
		{
			ID:       "vec2",
			Values:   []float64{0.9, 0.1},
			Metadata: map[string]any{"category": "B"},
		},
		{
			ID:       "vec3",
			Values:   []float64{0.8, 0.2},
			Metadata: map[string]any{"category": "A"},
		},
	}
	err := store.Upsert(ctx, vectors)
	require.NoError(t, err)

	// Query with filter
	query := []float64{1.0, 0.0}
	filter := map[string]any{"category": "A"}
	results, err := store.Query(ctx, query, 10, filter)
	require.NoError(t, err)

	// Should only return vectors with category A
	assert.Len(t, results, 2)
	for _, r := range results {
		assert.Contains(t, []string{"vec1", "vec3"}, r.ID)
	}
}

func TestMemoryVectorStore_QueryValidation(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryVectorStore(Config{})

	tests := []struct {
		name    string
		vector  []float64
		limit   int
		wantErr bool
	}{
		{
			name:    "empty vector",
			vector:  []float64{},
			limit:   1,
			wantErr: true,
		},
		{
			name:    "nil vector",
			vector:  nil,
			limit:   1,
			wantErr: true,
		},
		{
			name:    "zero limit",
			vector:  []float64{1.0},
			limit:   0,
			wantErr: true,
		},
		{
			name:    "negative limit",
			vector:  []float64{1.0},
			limit:   -1,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := store.Query(ctx, tt.vector, tt.limit, nil)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMemoryVectorStore_QueryEmpty(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryVectorStore(Config{})

	// Query empty store
	results, err := store.Query(ctx, []float64{1.0, 0.0}, 10, nil)
	require.NoError(t, err)
	assert.Len(t, results, 0)
}

func TestMemoryVectorStore_QueryDimensionMismatch(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryVectorStore(Config{})

	// Insert 3D vectors
	err := store.Upsert(ctx, []sdk.Vector{
		{ID: "vec1", Values: []float64{1.0, 0.0, 0.0}},
		{ID: "vec2", Values: []float64{0.0, 1.0, 0.0}},
	})
	require.NoError(t, err)

	// Query with 2D vector - should skip mismatched vectors
	results, err := store.Query(ctx, []float64{1.0, 0.0}, 10, nil)
	require.NoError(t, err)
	assert.Len(t, results, 0)
}

func TestMemoryVectorStore_Delete(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryVectorStore(Config{})

	// Insert vectors
	vectors := []sdk.Vector{
		{ID: "vec1", Values: []float64{1.0, 0.0}},
		{ID: "vec2", Values: []float64{0.0, 1.0}},
		{ID: "vec3", Values: []float64{0.5, 0.5}},
	}
	err := store.Upsert(ctx, vectors)
	require.NoError(t, err)

	// Delete some vectors
	err = store.Delete(ctx, []string{"vec1", "vec3"})
	require.NoError(t, err)
	assert.Equal(t, 1, store.Count())

	// Verify only vec2 remains
	results, err := store.Query(ctx, []float64{0.0, 1.0}, 10, nil)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "vec2", results[0].ID)
}

func TestMemoryVectorStore_DeleteNonexistent(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryVectorStore(Config{})

	// Delete from empty store
	err := store.Delete(ctx, []string{"vec1", "vec2"})
	require.NoError(t, err)
	assert.Equal(t, 0, store.Count())
}

func TestMemoryVectorStore_DeleteEmpty(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryVectorStore(Config{})

	// Delete empty list
	err := store.Delete(ctx, []string{})
	require.NoError(t, err)
}

func TestMemoryVectorStore_Clear(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryVectorStore(Config{})

	// Insert vectors
	vectors := []sdk.Vector{
		{ID: "vec1", Values: []float64{1.0, 0.0}},
		{ID: "vec2", Values: []float64{0.0, 1.0}},
	}
	err := store.Upsert(ctx, vectors)
	require.NoError(t, err)
	assert.Equal(t, 2, store.Count())

	// Clear
	err = store.Clear(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, store.Count())
}

func TestMemoryVectorStore_CosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a        []float64
		b        []float64
		expected float64
		wantErr  bool
	}{
		{
			name:     "identical vectors",
			a:        []float64{1.0, 0.0, 0.0},
			b:        []float64{1.0, 0.0, 0.0},
			expected: 1.0,
		},
		{
			name:     "orthogonal vectors",
			a:        []float64{1.0, 0.0},
			b:        []float64{0.0, 1.0},
			expected: 0.0,
		},
		{
			name:     "opposite vectors",
			a:        []float64{1.0, 0.0},
			b:        []float64{-1.0, 0.0},
			expected: -1.0,
		},
		{
			name:    "dimension mismatch",
			a:       []float64{1.0, 0.0},
			b:       []float64{1.0, 0.0, 0.0},
			wantErr: true,
		},
		{
			name:    "zero vector",
			a:       []float64{0.0, 0.0},
			b:       []float64{1.0, 0.0},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := cosineSimilarity(tt.a, tt.b)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.InDelta(t, tt.expected, result, 0.001)
			}
		})
	}
}

func TestMemoryVectorStore_Concurrent(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryVectorStore(Config{})

	// Concurrent writes
	const numGoroutines = 10
	const vectorsPerGoroutine = 100

	done := make(chan bool, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			vectors := make([]sdk.Vector, vectorsPerGoroutine)
			for j := 0; j < vectorsPerGoroutine; j++ {
				vectors[j] = sdk.Vector{
					ID:     fmt.Sprintf("vec-%d-%d", id, j),
					Values: []float64{float64(id), float64(j)},
				}
			}
			_ = store.Upsert(ctx, vectors)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify count
	assert.Equal(t, numGoroutines*vectorsPerGoroutine, store.Count())

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		go func() {
			_, _ = store.Query(ctx, []float64{1.0, 1.0}, 10, nil)
			done <- true
		}()
	}

	// Wait for all reads
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}

func BenchmarkMemoryVectorStore_Upsert(b *testing.B) {
	ctx := context.Background()
	store := NewMemoryVectorStore(Config{})

	vector := sdk.Vector{
		ID:     "vec1",
		Values: make([]float64, 1536), // typical embedding size
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vector.ID = fmt.Sprintf("vec%d", i)
		_ = store.Upsert(ctx, []sdk.Vector{vector})
	}
}

func BenchmarkMemoryVectorStore_Query(b *testing.B) {
	ctx := context.Background()
	store := NewMemoryVectorStore(Config{})

	// Insert 1000 vectors
	vectors := make([]sdk.Vector, 1000)
	for i := range vectors {
		vectors[i] = sdk.Vector{
			ID:     fmt.Sprintf("vec%d", i),
			Values: make([]float64, 1536),
		}
		for j := range vectors[i].Values {
			vectors[i].Values[j] = float64(i+j) / 1000.0
		}
	}
	_ = store.Upsert(ctx, vectors)

	query := make([]float64, 1536)
	for i := range query {
		query[i] = float64(i) / 1000.0
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.Query(ctx, query, 10, nil)
	}
}
