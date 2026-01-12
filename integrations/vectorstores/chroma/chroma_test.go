package chroma

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	sdk "github.com/xraph/ai-sdk"
)

func TestNewChromaVectorStore(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "missing base URL",
			cfg: Config{
				CollectionName: "test",
			},
			wantErr: true,
		},
		{
			name: "missing collection name",
			cfg: Config{
				BaseURL: "http://localhost:8000",
			},
			wantErr: true,
		},
		{
			name: "valid config",
			cfg: Config{
				BaseURL:        "http://localhost:8000",
				CollectionName: "test",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip actual creation for tests without valid Chroma instance
			if tt.name == "valid config" {
				t.Skip("requires running ChromaDB instance")
			}

			_, err := NewChromaVectorStore(context.Background(), tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewChromaVectorStore() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChromaVectorStore_Upsert(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/collections/test":
			// GET collection - return exists
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"name": "test"})
		case "/api/v1/collections/test/add":
			// POST upsert
			if r.Method != "POST" {
				t.Errorf("expected POST, got %s", r.Method)
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]bool{"success": true})
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	store, err := NewChromaVectorStore(context.Background(), Config{
		BaseURL:        server.URL,
		CollectionName: "test",
	})
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

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

	err = store.Upsert(context.Background(), vectors)
	if err != nil {
		t.Errorf("Upsert() error = %v", err)
	}
}

func TestChromaVectorStore_UpsertEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"name": "test"})
	}))
	defer server.Close()

	store, _ := NewChromaVectorStore(context.Background(), Config{
		BaseURL:        server.URL,
		CollectionName: "test",
	})

	err := store.Upsert(context.Background(), []sdk.Vector{})
	if err != nil {
		t.Errorf("Upsert() with empty vectors should not error, got: %v", err)
	}
}

func TestChromaVectorStore_UpsertInvalidVectors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"name": "test"})
	}))
	defer server.Close()

	store, _ := NewChromaVectorStore(context.Background(), Config{
		BaseURL:        server.URL,
		CollectionName: "test",
	})

	tests := []struct {
		name    string
		vectors []sdk.Vector
		wantErr bool
	}{
		{
			name: "empty ID",
			vectors: []sdk.Vector{
				{ID: "", Values: []float64{1, 2, 3}},
			},
			wantErr: true,
		},
		{
			name: "empty values",
			vectors: []sdk.Vector{
				{ID: "test", Values: []float64{}},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := store.Upsert(context.Background(), tt.vectors)
			if (err != nil) != tt.wantErr {
				t.Errorf("Upsert() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChromaVectorStore_Query(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/collections/test":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"name": "test"})
		case "/api/v1/collections/test/query":
			if r.Method != "POST" {
				t.Errorf("expected POST, got %s", r.Method)
			}
			w.WriteHeader(http.StatusOK)
			response := chromaQueryResponse{
				IDs:       [][]string{{"test-1", "test-2"}},
				Distances: [][]float64{{0.1, 0.2}},
				Metadatas: [][]map[string]any{
					{
						{"text": "doc 1"},
						{"text": "doc 2"},
					},
				},
			}
			json.NewEncoder(w).Encode(response)
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	store, _ := NewChromaVectorStore(context.Background(), Config{
		BaseURL:        server.URL,
		CollectionName: "test",
	})

	results, err := store.Query(context.Background(), []float64{0.1, 0.2, 0.3}, 5, nil)
	if err != nil {
		t.Errorf("Query() error = %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	if results[0].ID != "test-1" {
		t.Errorf("expected ID 'test-1', got '%s'", results[0].ID)
	}
}

func TestChromaVectorStore_QueryInvalid(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"name": "test"})
	}))
	defer server.Close()

	store, _ := NewChromaVectorStore(context.Background(), Config{
		BaseURL:        server.URL,
		CollectionName: "test",
	})

	tests := []struct {
		name    string
		vector  []float64
		limit   int
		wantErr bool
	}{
		{
			name:    "empty vector",
			vector:  []float64{},
			limit:   5,
			wantErr: true,
		},
		{
			name:    "zero limit",
			vector:  []float64{1, 2, 3},
			limit:   0,
			wantErr: true,
		},
		{
			name:    "negative limit",
			vector:  []float64{1, 2, 3},
			limit:   -1,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := store.Query(context.Background(), tt.vector, tt.limit, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("Query() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChromaVectorStore_Delete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/collections/test":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"name": "test"})
		case "/api/v1/collections/test/delete":
			if r.Method != "POST" {
				t.Errorf("expected POST, got %s", r.Method)
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]bool{"success": true})
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	store, _ := NewChromaVectorStore(context.Background(), Config{
		BaseURL:        server.URL,
		CollectionName: "test",
	})

	err := store.Delete(context.Background(), []string{"test-1", "test-2"})
	if err != nil {
		t.Errorf("Delete() error = %v", err)
	}
}

func TestChromaVectorStore_DeleteEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"name": "test"})
	}))
	defer server.Close()

	store, _ := NewChromaVectorStore(context.Background(), Config{
		BaseURL:        server.URL,
		CollectionName: "test",
	})

	err := store.Delete(context.Background(), []string{})
	if err != nil {
		t.Errorf("Delete() with empty IDs should not error, got: %v", err)
	}
}

func TestChromaVectorStore_Close(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"name": "test"})
	}))
	defer server.Close()

	store, _ := NewChromaVectorStore(context.Background(), Config{
		BaseURL:        server.URL,
		CollectionName: "test",
	})

	err := store.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

