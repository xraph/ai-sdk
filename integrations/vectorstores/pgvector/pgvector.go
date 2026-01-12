package pgvector

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	sdk "github.com/xraph/ai-sdk"
	logger "github.com/xraph/go-utils/log"
	"github.com/xraph/go-utils/metrics"
)

// PgVectorStore implements VectorStore using PostgreSQL with pgvector extension.
type PgVectorStore struct {
	pool      *pgxpool.Pool
	tableName string
	logger    logger.Logger
	metrics   metrics.Metrics
}

// Config configures the PgVectorStore.
type Config struct {
	ConnectionString string        // Required: PostgreSQL connection string
	TableName        string        // Optional: defaults to "vectors"
	Dimensions       int           // Optional: vector dimensions (validated on first insert)
	IndexType        string        // Optional: "hnsw" or "ivfflat", defaults to "hnsw"
	MaxConns         int           // Optional: max connections, defaults to 25
	MinConns         int           // Optional: min connections, defaults to 5
	ConnectTimeout   time.Duration // Optional: connect timeout, defaults to 30s
	Logger           logger.Logger
	Metrics          metrics.Metrics
}

// NewPgVectorStore creates a new pgvector-based vector store.
func NewPgVectorStore(ctx context.Context, cfg Config) (*PgVectorStore, error) {
	if cfg.ConnectionString == "" {
		return nil, fmt.Errorf("connection string is required")
	}

	// Set defaults
	if cfg.TableName == "" {
		cfg.TableName = "vectors"
	}
	if cfg.MaxConns == 0 {
		cfg.MaxConns = 25
	}
	if cfg.MinConns == 0 {
		cfg.MinConns = 5
	}
	if cfg.ConnectTimeout == 0 {
		cfg.ConnectTimeout = 30 * time.Second
	}

	// Parse pool config
	poolConfig, err := pgxpool.ParseConfig(cfg.ConnectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

	// Safe conversion since MaxConns and MinConns are reasonable connection pool sizes
	poolConfig.MaxConns = int32(cfg.MaxConns)   // #nosec G115 - connection pool size is always reasonable
	poolConfig.MinConns = int32(cfg.MinConns)   // #nosec G115 - connection pool size is always reasonable
	poolConfig.MaxConnLifetime = 1 * time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute

	// Create connection pool
	connectCtx, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(connectCtx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	store := &PgVectorStore{
		pool:      pool,
		tableName: cfg.TableName,
		logger:    cfg.Logger,
		metrics:   cfg.Metrics,
	}

	// Verify pgvector extension and create table
	if err := store.initialize(ctx, cfg.Dimensions, cfg.IndexType); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to initialize: %w", err)
	}

	if store.logger != nil {
		store.logger.Info("pgvector store initialized",
			logger.String("table", cfg.TableName),
			logger.Int("max_conns", cfg.MaxConns))
	}

	return store, nil
}

// initialize ensures the pgvector extension exists and creates the table.
func (p *PgVectorStore) initialize(ctx context.Context, dimensions int, indexType string) error {
	conn, err := p.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire connection: %w", err)
	}
	defer conn.Release()

	// Enable pgvector extension
	_, err = conn.Exec(ctx, "CREATE EXTENSION IF NOT EXISTS vector")
	if err != nil {
		return fmt.Errorf("failed to create vector extension: %w", err)
	}

	// Create table with vector column (dimension set on first insert if not specified)
	dimClause := ""
	if dimensions > 0 {
		dimClause = fmt.Sprintf("(%d)", dimensions)
	}

	createTableSQL := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id TEXT PRIMARY KEY,
			embedding vector%s NOT NULL,
			metadata JSONB,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		)
	`, pgx.Identifier{p.tableName}.Sanitize(), dimClause)

	_, err = conn.Exec(ctx, createTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	// Create index if specified
	if indexType != "" {
		if err := p.createIndex(ctx, conn, indexType, dimensions); err != nil {
			// Log warning but don't fail - index can be created later
			if p.logger != nil {
				p.logger.Warn("failed to create index", logger.String("error", err.Error()))
			}
		}
	}

	return nil
}

// createIndex creates a vector index (HNSW or IVFFlat).
func (p *PgVectorStore) createIndex(ctx context.Context, conn *pgxpool.Conn, indexType string, dimensions int) error {
	indexName := fmt.Sprintf("%s_embedding_idx", p.tableName)

	var indexSQL string
	switch strings.ToLower(indexType) {
	case "hnsw":
		// HNSW index - better recall, slower build
		indexSQL = fmt.Sprintf(`
			CREATE INDEX IF NOT EXISTS %s ON %s 
			USING hnsw (embedding vector_cosine_ops)
		`, indexName, pgx.Identifier{p.tableName}.Sanitize())
	case "ivfflat":
		// IVFFlat index - faster build, lower recall
		lists := 100 // Default, can be configured
		if dimensions > 0 {
			// Rule of thumb: lists = rows / 1000, capped
			lists = max(10, min(dimensions/10, 1000))
		}
		indexSQL = fmt.Sprintf(`
			CREATE INDEX IF NOT EXISTS %s ON %s 
			USING ivfflat (embedding vector_cosine_ops) WITH (lists = %d)
		`, indexName, pgx.Identifier{p.tableName}.Sanitize(), lists)
	default:
		return fmt.Errorf("unsupported index type: %s (use 'hnsw' or 'ivfflat')", indexType)
	}

	_, err := conn.Exec(ctx, indexSQL)
	if err != nil {
		return fmt.Errorf("failed to create %s index: %w", indexType, err)
	}

	if p.logger != nil {
		p.logger.Info("created vector index",
			logger.String("type", indexType),
			logger.String("name", indexName))
	}

	return nil
}

// Upsert adds or updates vectors in the store.
func (p *PgVectorStore) Upsert(ctx context.Context, vectors []sdk.Vector) error {
	if len(vectors) == 0 {
		return nil
	}

	start := time.Now()
	defer func() {
		if p.metrics != nil {
			p.metrics.Histogram("forge.integrations.pgvector.upsert_duration").Observe(time.Since(start).Seconds())
		}
	}()

	// Use batch insert for efficiency
	batch := &pgx.Batch{}
	upsertSQL := fmt.Sprintf(`
		INSERT INTO %s (id, embedding, metadata, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (id) DO UPDATE SET
			embedding = EXCLUDED.embedding,
			metadata = EXCLUDED.metadata,
			updated_at = NOW()
	`, pgx.Identifier{p.tableName}.Sanitize())

	for _, v := range vectors {
		if v.ID == "" {
			return fmt.Errorf("vector ID cannot be empty")
		}
		if len(v.Values) == 0 {
			return fmt.Errorf("vector values cannot be empty for ID %s", v.ID)
		}

		// Convert to pgvector format
		embedding := vectorToString(v.Values)
		batch.Queue(upsertSQL, v.ID, embedding, v.Metadata)
	}

	// Execute batch
	br := p.pool.SendBatch(ctx, batch)
	defer br.Close()

	// Check all results
	for i := 0; i < len(vectors); i++ {
		_, err := br.Exec()
		if err != nil {
			return fmt.Errorf("failed to upsert vector %s: %w", vectors[i].ID, err)
		}
	}

	if p.logger != nil {
		p.logger.Debug("upserted vectors to pgvector",
			logger.Int("count", len(vectors)),
			logger.Duration("duration", time.Since(start)))
	}

	if p.metrics != nil {
		p.metrics.Counter("forge.integrations.pgvector.upsert").Add(float64(len(vectors)))
	}

	return nil
}

// Query performs vector similarity search using cosine distance.
func (p *PgVectorStore) Query(ctx context.Context, vector []float64, limit int, filter map[string]any) ([]sdk.VectorMatch, error) {
	if len(vector) == 0 {
		return nil, fmt.Errorf("query vector cannot be empty")
	}
	if limit <= 0 {
		return nil, fmt.Errorf("limit must be positive, got %d", limit)
	}

	start := time.Now()
	defer func() {
		if p.metrics != nil {
			p.metrics.Histogram("forge.integrations.pgvector.query_duration").Observe(time.Since(start).Seconds())
		}
	}()

	// Build query with optional filter
	embedding := vectorToString(vector)
	querySQL := fmt.Sprintf(`
		SELECT id, 1 - (embedding <=> $1::vector) AS score, metadata
		FROM %s
		%s
		ORDER BY embedding <=> $1::vector
		LIMIT $2
	`, pgx.Identifier{p.tableName}.Sanitize(), buildWhereClause(filter))

	args := []any{embedding, limit}

	// Execute query
	rows, err := p.pool.Query(ctx, querySQL, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	// Collect results
	var results []sdk.VectorMatch
	for rows.Next() {
		var match sdk.VectorMatch
		if err := rows.Scan(&match.ID, &match.Score, &match.Metadata); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		results = append(results, match)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	if p.logger != nil {
		p.logger.Debug("queried pgvector",
			logger.Int("results", len(results)),
			logger.Duration("duration", time.Since(start)))
	}

	if p.metrics != nil {
		p.metrics.Counter("forge.integrations.pgvector.query").Inc()
		p.metrics.Histogram("forge.integrations.pgvector.results").Observe(float64(len(results)))
	}

	return results, nil
}

// Delete removes vectors by ID.
func (p *PgVectorStore) Delete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	start := time.Now()

	deleteSQL := fmt.Sprintf(`
		DELETE FROM %s WHERE id = ANY($1)
	`, pgx.Identifier{p.tableName}.Sanitize())

	result, err := p.pool.Exec(ctx, deleteSQL, ids)
	if err != nil {
		return fmt.Errorf("delete failed: %w", err)
	}

	deleted := result.RowsAffected()

	if p.logger != nil {
		p.logger.Debug("deleted vectors from pgvector",
			logger.Int("requested", len(ids)),
			logger.Int64("deleted", deleted),
			logger.Duration("duration", time.Since(start)))
	}

	if p.metrics != nil {
		p.metrics.Counter("forge.integrations.pgvector.delete").Add(float64(deleted))
	}

	return nil
}

// Close closes the connection pool.
func (p *PgVectorStore) Close() {
	p.pool.Close()
	if p.logger != nil {
		p.logger.Info("pgvector store closed")
	}
}

// Count returns the number of vectors in the store.
func (p *PgVectorStore) Count(ctx context.Context) (int64, error) {
	var count int64
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM %s", pgx.Identifier{p.tableName}.Sanitize())
	err := p.pool.QueryRow(ctx, countSQL).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count failed: %w", err)
	}
	return count, nil
}

// vectorToString converts a float64 slice to pgvector string format.
func vectorToString(values []float64) string {
	parts := make([]string, len(values))
	for i, v := range values {
		parts[i] = fmt.Sprintf("%f", v)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

// buildWhereClause builds a WHERE clause for metadata filtering.
func buildWhereClause(filter map[string]any) string {
	if len(filter) == 0 {
		return ""
	}

	// Simple JSON containment for now
	// For production, consider more sophisticated filtering
	var conditions []string
	for key, value := range filter {
		conditions = append(conditions, fmt.Sprintf("metadata->>'%s' = '%v'", key, value))
	}

	return "WHERE " + strings.Join(conditions, " AND ")
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
