# Integration Tests

Comprehensive integration tests for Forge AI SDK integrations using testcontainers-go.

## Overview

These tests verify that all integrations work correctly with real services running in Docker containers. Tests use `testcontainers-go` to automatically start, configure, and tear down service containers.

## Prerequisites

- Docker installed and running
- Go 1.21+
- Sufficient disk space for Docker images

## Running Tests

### All Integration Tests

```bash
go test -tags=integration ./...
```

### Specific Test Suite

```bash
# Vector stores only
go test -tags=integration -run TestPgVector ./...
go test -tags=integration -run TestQdrant ./...
go test -tags=integration -run TestChroma ./...

# State stores only
go test -tags=integration -run TestPostgresStateStore ./...
go test -tags=integration -run TestRedisStateStore ./...

# Caches only
go test -tags=integration -run TestRedisCache ./...
```

### With Verbose Output

```bash
go test -tags=integration -v ./...
```

### With Coverage

```bash
go test -tags=integration -cover -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Test Structure

### Vector Store Tests

Tests verify:
- **Upsert**: Adding/updating vectors
- **Query**: Similarity search
- **Query with Filter**: Metadata filtering
- **Delete**: Removing vectors

**Containers Used**:
- PostgreSQL + pgvector (`ankane/pgvector:latest`)
- Qdrant (`qdrant/qdrant:latest`)
- ChromaDB (`chromadb/chroma:latest`)

### State Store Tests

Tests verify:
- **Save**: Storing agent state
- **Load**: Retrieving agent state
- **List**: Listing sessions for an agent
- **Delete**: Removing agent state

**Containers Used**:
- PostgreSQL (`postgres:16-alpine`)
- Redis (`redis:7-alpine`)

### Cache Tests

Tests verify:
- **Set**: Storing cache entries
- **Get**: Retrieving cache entries
- **Delete**: Removing cache entries
- **Clear**: Clearing all entries
- **TTL**: Time-to-live expiration

**Containers Used**:
- Redis (`redis:7-alpine`)

## Docker Images

The tests will automatically pull these images if not present:

```bash
docker pull ankane/pgvector:latest
docker pull qdrant/qdrant:latest
docker pull chromadb/chroma:latest
docker pull postgres:16-alpine
docker pull redis:7-alpine
```

## Troubleshooting

### Docker Not Running

```bash
# Check Docker status
docker info

# Start Docker (macOS)
open -a Docker
```

### Permission Denied

```bash
# Add user to docker group (Linux)
sudo usermod -aG docker $USER
newgrp docker
```

### Container Start Timeout

Some containers may take longer to start on slower machines:

```go
// Increase timeout in tests
WaitingFor: wait.ForLog("...").WithStartupTimeout(120 * time.Second)
```

### Port Already in Use

Testcontainers automatically assigns random ports, but if you see port conflicts:

```bash
# Check for processes using ports
lsof -i :5432
lsof -i :6379
lsof -i :6334
lsof -i :8000

# Stop conflicting containers
docker ps
docker stop <container_id>
```

### Cleanup Failed Containers

```bash
# List all containers
docker ps -a

# Remove stopped containers
docker container prune

# Remove unused images
docker image prune
```

## CI/CD Integration

### GitHub Actions

```yaml
name: Integration Tests

on: [push, pull_request]

jobs:
  integration:
    runs-on: ubuntu-latest
    
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Run Integration Tests
        run: go test -tags=integration -v ./integrations/tests/integration/...
```

### GitLab CI

```yaml
integration_tests:
  image: golang:1.21
  services:
    - docker:dind
  variables:
    DOCKER_HOST: tcp://docker:2375
  script:
    - go test -tags=integration -v ./integrations/tests/integration/...
```

## Performance

**Typical Test Duration**:
- Vector store tests: ~30-60 seconds each
- State store tests: ~20-40 seconds each
- Cache tests: ~15-30 seconds each
- Total suite: ~5-10 minutes

**Resource Usage**:
- Memory: ~2-4 GB
- Disk: ~500 MB - 1 GB (Docker images)
- CPU: 2-4 cores recommended

## Best Practices

1. **Run Locally First**: Verify tests pass locally before CI
2. **Clean Up**: Containers are automatically cleaned up via `defer`
3. **Parallel Execution**: Tests can run in parallel with `-parallel` flag
4. **Isolation**: Each test uses unique table/collection names
5. **Timeouts**: Generous timeouts prevent flaky tests

## Writing New Integration Tests

### Template

```go
// +build integration

package integration

func TestMyIntegration(t *testing.T) {
	ctx := context.Background()

	// 1. Start container
	req := testcontainers.ContainerRequest{
		Image:        "my-service:latest",
		ExposedPorts: []string{"port/tcp"},
		WaitingFor:   wait.ForLog("ready"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer container.Terminate(ctx)

	// 2. Get connection details
	host, _ := container.Host(ctx)
	port, _ := container.MappedPort(ctx, "port")

	// 3. Create integration
	// ... your integration setup ...

	// 4. Run tests
	TestMyOperations(t, myIntegration)
}
```

## Additional Resources

- [testcontainers-go Documentation](https://golang.testcontainers.org/)
- [Docker Hub](https://hub.docker.com/)
- [Go Testing Package](https://pkg.go.dev/testing)

## Contributing

When adding new integrations:
1. Add integration test following the template
2. Add helper functions to `helpers.go` if needed
3. Update this README with new test details
4. Verify tests pass in CI

## License

MIT License - see [LICENSE](../../../LICENSE)

