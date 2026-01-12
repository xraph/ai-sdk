//go:build integration
// +build integration

package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/xraph/ai-sdk/integrations/vectorstores/chroma"
	"github.com/xraph/ai-sdk/integrations/vectorstores/pgvector"
	"github.com/xraph/ai-sdk/integrations/vectorstores/qdrant"
)

func TestPgVectorIntegration(t *testing.T) {
	ctx := context.Background()

	// Start PostgreSQL with pgvector
	req := testcontainers.ContainerRequest{
		Image:        "ankane/pgvector:latest",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       "testdb",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("Failed to start PostgreSQL container: %v", err)
	}
	defer container.Terminate(ctx)

	// Get connection details
	host, err := container.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}
	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatal(err)
	}

	connString := fmt.Sprintf("postgres://postgres:test@%s:%s/testdb?sslmode=disable", host, port.Port())

	// Create store
	store, err := pgvector.NewPgVectorStore(ctx, pgvector.PgVectorConfig{
		ConnString: connString,
		TableName:  "test_vectors",
		Dimensions: 3,
	})
	if err != nil {
		t.Fatalf("Failed to create pgvector store: %v", err)
	}
	defer store.Close()

	// Run standard tests
	TestVectorStoreOperations(t, store)
}

func TestQdrantIntegration(t *testing.T) {
	ctx := context.Background()

	// Start Qdrant
	req := testcontainers.ContainerRequest{
		Image:        "qdrant/qdrant:latest",
		ExposedPorts: []string{"6334/tcp"},
		WaitingFor:   wait.ForLog("Qdrant gRPC listening").WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("Failed to start Qdrant container: %v", err)
	}
	defer container.Terminate(ctx)

	// Get connection details
	host, err := container.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}
	port, err := container.MappedPort(ctx, "6334")
	if err != nil {
		t.Fatal(err)
	}

	// Create store
	store, err := qdrant.NewQdrantVectorStore(ctx, qdrant.QdrantConfig{
		Host:           fmt.Sprintf("%s:%s", host, port.Port()),
		CollectionName: "test_collection",
		Dimensions:     3,
	})
	if err != nil {
		t.Fatalf("Failed to create Qdrant store: %v", err)
	}

	// Run standard tests
	TestVectorStoreOperations(t, store)
}

func TestChromaIntegration(t *testing.T) {
	ctx := context.Background()

	// Start ChromaDB
	req := testcontainers.ContainerRequest{
		Image:        "chromadb/chroma:latest",
		ExposedPorts: []string{"8000/tcp"},
		WaitingFor:   wait.ForHTTP("/api/v1/heartbeat").WithPort("8000").WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("Failed to start ChromaDB container: %v", err)
	}
	defer container.Terminate(ctx)

	// Get connection details
	host, err := container.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}
	port, err := container.MappedPort(ctx, "8000")
	if err != nil {
		t.Fatal(err)
	}

	baseURL := fmt.Sprintf("http://%s:%s", host, port.Port())

	// Create store
	store, err := chroma.NewChromaVectorStore(ctx, chroma.Config{
		BaseURL:        baseURL,
		CollectionName: "test_collection",
	})
	if err != nil {
		t.Fatalf("Failed to create ChromaDB store: %v", err)
	}
	defer store.Close()

	// Run standard tests
	TestVectorStoreOperations(t, store)
}
