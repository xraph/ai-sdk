package weaviate

import (
	"context"
	"testing"

	sdk "github.com/xraph/ai-sdk"
)

func TestNewWeaviateVectorStore(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "missing host",
			cfg: Config{
				ClassName: "TestClass",
			},
			wantErr: true,
		},
		{
			name: "missing class name",
			cfg: Config{
				Host: "localhost:8080",
			},
			wantErr: true,
		},
		{
			name: "valid config with defaults",
			cfg: Config{
				Host:      "localhost:8080",
				ClassName: "TestClass",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip actual creation for tests without valid Weaviate instance
			if tt.name == "valid config with defaults" {
				t.Skip("requires running Weaviate instance")
			}

			_, err := NewWeaviateVectorStore(context.Background(), tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewWeaviateVectorStore() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWeaviateVectorStore_Upsert(t *testing.T) {
	t.Skip("requires running Weaviate instance - run integration tests instead")

	ctx := context.Background()
	store, _ := NewWeaviateVectorStore(ctx, Config{
		Host:      "localhost:8080",
		ClassName: "TestVectors",
	})

	vectors := []sdk.Vector{
		{
			ID:     "test-1",
			Values: []float64{0.1, 0.2, 0.3},
			Metadata: map[string]any{
				"text": "test document 1",
			},
		},
		{
			ID:     "test-2",
			Values: []float64{0.4, 0.5, 0.6},
			Metadata: map[string]any{
				"text": "test document 2",
			},
		},
	}

	err := store.Upsert(ctx, vectors)
	if err != nil {
		t.Errorf("Upsert() error = %v", err)
	}
}

func TestWeaviateVectorStore_Query(t *testing.T) {
	t.Skip("requires running Weaviate instance - run integration tests instead")

	ctx := context.Background()
	store, _ := NewWeaviateVectorStore(ctx, Config{
		Host:      "localhost:8080",
		ClassName: "TestVectors",
	})

	results, err := store.Query(ctx, []float64{0.1, 0.2, 0.3}, 5, nil)
	if err != nil {
		t.Errorf("Query() error = %v", err)
	}

	if len(results) > 5 {
		t.Errorf("Query() returned more results than limit: got %d, want ≤5", len(results))
	}
}

func TestWeaviateVectorStore_Delete(t *testing.T) {
	t.Skip("requires running Weaviate instance - run integration tests instead")

	ctx := context.Background()
	store, _ := NewWeaviateVectorStore(ctx, Config{
		Host:      "localhost:8080",
		ClassName: "TestVectors",
	})

	err := store.Delete(ctx, []string{"test-1", "test-2"})
	if err != nil {
		t.Errorf("Delete() error = %v", err)
	}
}

func TestToFloat32(t *testing.T) {
	input := []float64{1.1, 2.2, 3.3}
	result := toFloat32(input)

	if len(result) != len(input) {
		t.Errorf("toFloat32() length = %d, want %d", len(result), len(input))
	}

	for i, v := range result {
		expected := float32(input[i])
		if v != expected {
			t.Errorf("toFloat32()[%d] = %v, want %v", i, v, expected)
		}
	}
}
