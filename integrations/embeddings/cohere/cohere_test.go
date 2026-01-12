package cohere

import (
	"context"
	"testing"
)

func TestNewCohereEmbeddings(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "missing API key",
			cfg: Config{
				Model: "embed-english-v3.0",
			},
			wantErr: true,
		},
		{
			name: "missing model",
			cfg: Config{
				APIKey: "test-key",
			},
			wantErr: true,
		},
		{
			name: "unsupported model",
			cfg: Config{
				APIKey: "test-key",
				Model:  "invalid-model",
			},
			wantErr: true,
		},
		{
			name: "valid config",
			cfg: Config{
				APIKey: "test-key",
				Model:  "embed-english-v3.0",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewCohereEmbeddings(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewCohereEmbeddings() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetModelDimensions(t *testing.T) {
	tests := []struct {
		model    string
		wantDims int
		wantErr  bool
	}{
		{"embed-english-v3.0", 1024, false},
		{"embed-multilingual-v3.0", 1024, false},
		{"embed-english-light-v3.0", 384, false},
		{"embed-multilingual-light-v3.0", 384, false},
		{"embed-english-v2.0", 4096, false},
		{"embed-english-light-v2.0", 1024, false},
		{"embed-multilingual-v2.0", 768, false},
		{"invalid-model", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			dims, err := getModelDimensions(tt.model)
			if (err != nil) != tt.wantErr {
				t.Errorf("getModelDimensions() error = %v, wantErr %v", err, tt.wantErr)
			}
			if dims != tt.wantDims {
				t.Errorf("getModelDimensions() = %d, want %d", dims, tt.wantDims)
			}
		})
	}
}

func TestCohereEmbeddings_Dimensions(t *testing.T) {
	embeddings, _ := NewCohereEmbeddings(Config{
		APIKey: "test-key",
		Model:  "embed-english-v3.0",
	})

	dims := embeddings.Dimensions()
	if dims != 1024 {
		t.Errorf("Dimensions() = %d, want 1024", dims)
	}
}

func TestCohereEmbeddings_EmbedEmpty(t *testing.T) {
	embeddings, _ := NewCohereEmbeddings(Config{
		APIKey: "test-key",
		Model:  "embed-english-v3.0",
	})

	vectors, err := embeddings.Embed(context.Background(), []string{})
	if err != nil {
		t.Errorf("Embed() with empty texts should not error, got: %v", err)
	}
	if len(vectors) != 0 {
		t.Errorf("Embed() with empty texts should return empty slice, got %d vectors", len(vectors))
	}
}

func TestCohereEmbeddings_Embed(t *testing.T) {
	t.Skip("requires valid Cohere API key - run integration tests instead")

	embeddings, err := NewCohereEmbeddings(Config{
		APIKey: "your-api-key",
		Model:  "embed-english-v3.0",
	})
	if err != nil {
		t.Fatal(err)
	}

	texts := []string{
		"The quick brown fox jumps over the lazy dog",
		"Machine learning is a subset of artificial intelligence",
	}

	vectors, err := embeddings.Embed(context.Background(), texts)
	if err != nil {
		t.Errorf("Embed() error = %v", err)
	}

	if len(vectors) != len(texts) {
		t.Errorf("expected %d vectors, got %d", len(texts), len(vectors))
	}

	for i, v := range vectors {
		if len(v.Values) != 1024 {
			t.Errorf("vector %d has %d dimensions, want 1024", i, len(v.Values))
		}
	}
}
