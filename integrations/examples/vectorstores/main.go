package main

import (
	"context"
	"fmt"
	"log"

	sdk "github.com/xraph/ai-sdk"
	"github.com/xraph/ai-sdk/integrations/vectorstores/memory"
	"github.com/xraph/ai-sdk/integrations/vectorstores/pgvector"
	"github.com/xraph/ai-sdk/integrations/vectorstores/pinecone"
	"github.com/xraph/ai-sdk/integrations/vectorstores/qdrant"
)

// Build with: go run main.go
// Or from integrations/: go run examples/vectorstores/main.go

func main() {
	ctx := context.Background()

	// Example vectors
	vectors := []sdk.Vector{
		{
			ID:     "doc1",
			Values: []float64{0.1, 0.2, 0.3, 0.4, 0.5},
			Metadata: map[string]any{
				"title":    "Introduction to AI",
				"category": "documentation",
			},
		},
		{
			ID:     "doc2",
			Values: []float64{0.5, 0.4, 0.3, 0.2, 0.1},
			Metadata: map[string]any{
				"title":    "Advanced Topics",
				"category": "documentation",
			},
		},
	}

	// 1. In-Memory Vector Store (for testing)
	fmt.Println("=== In-Memory Vector Store ===")
	demoMemoryStore(ctx, vectors)

	// 2. pgvector (PostgreSQL)
	// Uncomment if PostgreSQL with pgvector is available
	// fmt.Println("\n=== pgvector (PostgreSQL) ===")
	// demoPgVector(ctx, vectors)

	// 3. Qdrant
	// Uncomment if Qdrant is running
	// fmt.Println("\n=== Qdrant ===")
	// demoQdrant(ctx, vectors)

	// 4. Pinecone
	// Uncomment if Pinecone API key is available
	// fmt.Println("\n=== Pinecone ===")
	// demoPinecone(ctx, vectors)
}

func demoMemoryStore(ctx context.Context, vectors []sdk.Vector) {
	// Create memory store
	store := memory.NewMemoryVectorStore(memory.Config{})

	// Upsert vectors
	if err := store.Upsert(ctx, vectors); err != nil {
		log.Fatalf("Failed to upsert: %v", err)
	}
	fmt.Printf("Upserted %d vectors\n", len(vectors))

	// Query
	queryVector := []float64{0.15, 0.25, 0.35, 0.45, 0.55}
	results, err := store.Query(ctx, queryVector, 10, nil)
	if err != nil {
		log.Fatalf("Failed to query: %v", err)
	}

	fmt.Printf("Found %d results:\n", len(results))
	for i, result := range results {
		fmt.Printf("  %d. ID: %s, Score: %.4f\n", i+1, result.ID, result.Score)
		if title, ok := result.Metadata["title"]; ok {
			fmt.Printf("     Title: %s\n", title)
		}
	}

	// Query with filter
	filter := map[string]any{"category": "documentation"}
	filteredResults, err := store.Query(ctx, queryVector, 10, filter)
	if err != nil {
		log.Fatalf("Failed to query with filter: %v", err)
	}
	fmt.Printf("Filtered results: %d\n", len(filteredResults))

	// Delete
	if err := store.Delete(ctx, []string{"doc1"}); err != nil {
		log.Fatalf("Failed to delete: %v", err)
	}
	fmt.Printf("Deleted doc1, remaining: %d\n", store.Count())
}

func demoPgVector(ctx context.Context, vectors []sdk.Vector) {
	// Create pgvector store
	store, err := pgvector.NewPgVectorStore(ctx, pgvector.Config{
		ConnectionString: "postgres://user:pass@localhost:5432/mydb",
		TableName:        "vectors",
		Dimensions:       5,
		IndexType:        "hnsw",
	})
	if err != nil {
		log.Fatalf("Failed to create pgvector store: %v", err)
	}
	defer store.Close()

	// Upsert and query...
	if err := store.Upsert(ctx, vectors); err != nil {
		log.Fatalf("Failed to upsert: %v", err)
	}

	queryVector := []float64{0.15, 0.25, 0.35, 0.45, 0.55}
	results, err := store.Query(ctx, queryVector, 10, nil)
	if err != nil {
		log.Fatalf("Failed to query: %v", err)
	}

	fmt.Printf("Found %d results from pgvector\n", len(results))
	count, _ := store.Count(ctx)
	fmt.Printf("Total vectors in store: %d\n", count)
}

func demoQdrant(ctx context.Context, vectors []sdk.Vector) {
	// Create Qdrant store
	store, err := qdrant.NewQdrantVectorStore(ctx, qdrant.Config{
		Host:           "localhost:6334",
		CollectionName: "example_vectors",
		VectorSize:     5,
		Distance:       "cosine",
	})
	if err != nil {
		log.Fatalf("Failed to create Qdrant store: %v", err)
	}
	defer store.Close()

	// Upsert and query...
	if err := store.Upsert(ctx, vectors); err != nil {
		log.Fatalf("Failed to upsert: %v", err)
	}

	queryVector := []float64{0.15, 0.25, 0.35, 0.45, 0.55}
	results, err := store.Query(ctx, queryVector, 10, nil)
	if err != nil {
		log.Fatalf("Failed to query: %v", err)
	}

	fmt.Printf("Found %d results from Qdrant\n", len(results))
	count, _ := store.Count(ctx)
	fmt.Printf("Total vectors in collection: %d\n", count)
}

func demoPinecone(ctx context.Context, vectors []sdk.Vector) {
	// Create Pinecone store
	store, err := pinecone.NewPineconeVectorStore(ctx, pinecone.Config{
		APIKey:    "your-pinecone-api-key",
		IndexName: "example-index",
		Namespace: "example",
	})
	if err != nil {
		log.Fatalf("Failed to create Pinecone store: %v", err)
	}
	defer store.Close()

	// Upsert and query...
	if err := store.Upsert(ctx, vectors); err != nil {
		log.Fatalf("Failed to upsert: %v", err)
	}

	queryVector := []float64{0.15, 0.25, 0.35, 0.45, 0.55}
	results, err := store.Query(ctx, queryVector, 10, nil)
	if err != nil {
		log.Fatalf("Failed to query: %v", err)
	}

	fmt.Printf("Found %d results from Pinecone\n", len(results))
	stats, _ := store.Stats(ctx)
	fmt.Printf("Total vectors in index: %d\n", stats.TotalVectorCount)
}

