package qdrant

import (
	"testing"

	sdk "github.com/xraph/ai-sdk"
	"github.com/stretchr/testify/assert"
)

func TestToFloat32(t *testing.T) {
	input := []float64{1.0, 2.5, 3.14159}
	output := toFloat32(input)
	
	assert.Len(t, output, len(input))
	for i := range input {
		assert.InDelta(t, input[i], float64(output[i]), 0.0001)
	}
}

func TestConvertToQdrantValue(t *testing.T) {
	tests := []struct {
		name  string
		input any
	}{
		{"string", "test"},
		{"int", 42},
		{"int64", int64(42)},
		{"float64", 3.14},
		{"bool", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val := convertToQdrantValue(tt.input)
			assert.NotNil(t, val)
			assert.NotNil(t, val.Kind)
		})
	}
}

func TestQdrantVectorStore_ImplementsInterface(t *testing.T) {
	var _ sdk.VectorStore = (*QdrantVectorStore)(nil)
}

func BenchmarkToFloat32(b *testing.B) {
	vector := make([]float64, 1536)
	for i := range vector {
		vector[i] = float64(i) / 1000.0
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = toFloat32(vector)
	}
}

