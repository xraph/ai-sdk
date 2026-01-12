package pinecone

import (
	"testing"

	"github.com/stretchr/testify/assert"
	sdk "github.com/xraph/ai-sdk"
)

func TestToFloat32(t *testing.T) {
	input := []float64{1.0, 2.5, 3.14159}
	output := toFloat32(input)

	assert.Len(t, output, len(input))
	for i := range input {
		assert.InDelta(t, input[i], float64(output[i]), 0.0001)
	}
}

func TestConvertToPineconeFilter(t *testing.T) {
	filter := map[string]any{
		"category": "test",
		"count":    float64(42), // structpb converts numbers to float64
	}

	result := convertToPineconeFilter(filter)
	assert.NotNil(t, result)

	// Convert back to map to verify
	resultMap := result.AsMap()
	assert.Len(t, resultMap, 2)
	assert.Equal(t, "test", resultMap["category"])
	assert.Equal(t, float64(42), resultMap["count"])
}

func TestPineconeVectorStore_ImplementsInterface(t *testing.T) {
	var _ sdk.VectorStore = (*PineconeVectorStore)(nil)
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
