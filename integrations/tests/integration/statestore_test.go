//go:build integration
// +build integration

package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/xraph/ai-sdk/integrations/statestores/postgres"
	"github.com/xraph/ai-sdk/integrations/statestores/redis"
)

func TestPostgresStateStoreIntegration(t *testing.T) {
	ctx := context.Background()

	// Start PostgreSQL
	req := testcontainers.ContainerRequest{
		Image:        "postgres:16-alpine",
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
	store, err := postgres.NewPostgresStateStore(ctx, postgres.Config{
		ConnString: connString,
		TableName:  "test_states",
	})
	if err != nil {
		t.Fatalf("Failed to create PostgreSQL state store: %v", err)
	}
	defer store.Close()

	// Run standard tests
	TestStateStoreOperations(t, store)
}

func TestRedisStateStoreIntegration(t *testing.T) {
	ctx := context.Background()

	// Start Redis
	req := testcontainers.ContainerRequest{
		Image:        "redis:7-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForLog("Ready to accept connections").WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("Failed to start Redis container: %v", err)
	}
	defer container.Terminate(ctx)

	// Get connection details
	host, err := container.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}
	port, err := container.MappedPort(ctx, "6379")
	if err != nil {
		t.Fatal(err)
	}

	// Create Redis client
	client := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", host, port.Port()),
	})

	// Create store
	store, err := redis.NewRedisStateStore(redis.RedisStateStoreConfig{
		Client: client,
		Prefix: "test:",
	})
	if err != nil {
		t.Fatalf("Failed to create Redis state store: %v", err)
	}

	// Run standard tests
	TestStateStoreOperations(t, store)
}
