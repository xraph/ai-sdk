package pgvector

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdk "github.com/xraph/ai-sdk"
)

// Note: These are unit tests with mocked behavior.
// Integration tests using testcontainers are in integration_test.go

func TestVectorToString(t *testing.T) {
	tests := []struct {
		name     string
		input    []float64
		expected string
	}{
		{
			name:     "simple vector",
			input:    []float64{1.0, 2.0, 3.0},
			expected: "[1.000000,2.000000,3.000000]",
		},
		{
			name:     "negative values",
			input:    []float64{-1.5, 0.0, 1.5},
			expected: "[-1.500000,0.000000,1.500000]",
		},
		{
			name:     "single value",
			input:    []float64{42.0},
			expected: "[42.000000]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := vectorToString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildWhereClause(t *testing.T) {
	tests := []struct {
		name     string
		filter   map[string]any
		expected string
	}{
		{
			name:     "no filter",
			filter:   nil,
			expected: "",
		},
		{
			name:     "empty filter",
			filter:   map[string]any{},
			expected: "",
		},
		{
			name: "single condition",
			filter: map[string]any{
				"type": "document",
			},
			expected: "WHERE metadata->>'type' = 'document'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildWhereClause(tt.filter)
			if tt.expected == "" {
				assert.Empty(t, result)
			} else {
				assert.Contains(t, result, "WHERE")
			}
		})
	}
}

func TestConfig_Validation(t *testing.T) {
	ctx := context.Background()

	t.Run("missing connection string", func(t *testing.T) {
		_, err := NewPgVectorStore(ctx, Config{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "connection string is required")
	})
}

func TestMinMax(t *testing.T) {
	assert.Equal(t, 5, max(3, 5))
	assert.Equal(t, 5, max(5, 3))
	assert.Equal(t, 3, min(3, 5))
	assert.Equal(t, 3, min(5, 3))
}

// Mock test for interface compliance
func TestPgVectorStore_ImplementsInterface(t *testing.T) {
	var _ sdk.VectorStore = (*PgVectorStore)(nil)
}

// Benchmark vector string conversion
func BenchmarkVectorToString(b *testing.B) {
	vector := make([]float64, 1536) // typical embedding size
	for i := range vector {
		vector[i] = float64(i) / 1000.0
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = vectorToString(vector)
	}
}
