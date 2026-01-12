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
	rediscache "github.com/xraph/ai-sdk/integrations/caches/redis"
)

func TestRedisCacheIntegration(t *testing.T) {
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

	// Create cache store
	cache, err := rediscache.NewRedisCacheStore(rediscache.RedisCacheStoreConfig{
		Client: client,
		Prefix: "test:",
	})
	if err != nil {
		t.Fatalf("Failed to create Redis cache store: %v", err)
	}

	// Run standard tests
	TestCacheStoreOperations(t, cache)

	// Additional TTL test
	t.Run("TTL", func(t *testing.T) {
		err := cache.Set(ctx, "ttl-key", []byte("ttl-value"), 1*time.Second)
		if err != nil {
			t.Fatalf("Set with TTL failed: %v", err)
		}

		// Should exist immediately
		_, found, _ := cache.Get(ctx, "ttl-key")
		if !found {
			t.Error("Expected key to exist")
		}

		// Wait for expiration
		time.Sleep(2 * time.Second)

		// Should be expired
		_, found, _ = cache.Get(ctx, "ttl-key")
		if found {
			t.Error("Expected key to be expired")
		}
	})
}
