package openai

import (
	"testing"

	"github.com/stretchr/testify/assert"
	sdk "github.com/xraph/ai-sdk"
)

func TestToFloat64(t *testing.T) {
	input := []float32{1.0, 2.5, 3.14159}
	output := toFloat64(input)

	assert.Len(t, output, len(input))
	for i := range input {
		assert.InDelta(t, float64(input[i]), output[i], 0.0001)
	}
}

func TestModelDefaults(t *testing.T) {
	tests := []struct {
		name       string
		model      string
		dimensions int
		expected   int
	}{
		{
			name:       "text-embedding-3-small",
			model:      ModelTextEmbedding3Small,
			dimensions: 0,
			expected:   1536,
		},
		{
			name:       "text-embedding-3-large",
			model:      ModelTextEmbedding3Large,
			dimensions: 0,
			expected:   3072,
		},
		{
			name:       "text-embedding-ada-002",
			model:      ModelTextEmbeddingAda002,
			dimensions: 0,
			expected:   1536,
		},
		{
			name:       "custom dimensions",
			model:      ModelTextEmbedding3Small,
			dimensions: 512,
			expected:   512,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			embeddings, err := NewOpenAIEmbeddings(Config{
				APIKey:     "test-key",
				Model:      tt.model,
				Dimensions: tt.dimensions,
			})
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, embeddings.Dimensions())
		})
	}
}

func TestOpenAIEmbeddings_ImplementsInterface(t *testing.T) {
	var _ sdk.EmbeddingModel = (*OpenAIEmbeddings)(nil)
}

func TestConfig_Validation(t *testing.T) {
	t.Run("missing API key", func(t *testing.T) {
		_, err := NewOpenAIEmbeddings(Config{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "API key is required")
	})
}
