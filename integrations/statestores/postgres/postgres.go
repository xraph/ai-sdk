package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	sdk "github.com/xraph/ai-sdk"
	logger "github.com/xraph/go-utils/log"
	"github.com/xraph/go-utils/metrics"
)

// PostgresStateStore implements the sdk.StateStore interface using PostgreSQL with JSONB storage.
type PostgresStateStore struct {
	pool    *pgxpool.Pool
	table   string
	logger  logger.Logger
	metrics metrics.Metrics
}

// Config provides configuration for the PostgresStateStore.
type Config struct {
	ConnString string // Required: PostgreSQL connection string
	TableName  string // Optional: Table name (default: "agent_states")
	Logger     logger.Logger
	Metrics    metrics.Metrics
}

// NewPostgresStateStore creates a new PostgreSQL-based state store.
func NewPostgresStateStore(ctx context.Context, cfg Config) (*PostgresStateStore, error) {
	if cfg.ConnString == "" {
		return nil, fmt.Errorf("connection string cannot be empty")
	}

	// Set defaults
	if cfg.TableName == "" {
		cfg.TableName = "agent_states"
	}

	// Create connection pool
	poolConfig, err := pgxpool.ParseConfig(cfg.ConnString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	store := &PostgresStateStore{
		pool:    pool,
		table:   cfg.TableName,
		logger:  cfg.Logger,
		metrics: cfg.Metrics,
	}

	// Initialize schema
	if err := store.initSchema(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	if store.logger != nil {
		store.logger.Info("postgres state store initialized",
			logger.String("table", cfg.TableName))
	}

	return store, nil
}

// initSchema creates the table and indexes if they don't exist.
func (p *PostgresStateStore) initSchema(ctx context.Context) error {
	createTableSQL := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			agent_id TEXT NOT NULL,
			session_id TEXT NOT NULL,
			state JSONB NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			PRIMARY KEY (agent_id, session_id)
		);
	`, p.table)

	if _, err := p.pool.Exec(ctx, createTableSQL); err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	// Create index on agent_id for efficient listing
	createIndexSQL := fmt.Sprintf(`
		CREATE INDEX IF NOT EXISTS idx_%s_agent_id 
		ON %s(agent_id);
	`, p.table, p.table)

	if _, err := p.pool.Exec(ctx, createIndexSQL); err != nil {
		p.logger.Warn("failed to create agent_id index, continuing", logger.Error(err))
	}

	// Create index on updated_at for time-based queries
	createTimeIndexSQL := fmt.Sprintf(`
		CREATE INDEX IF NOT EXISTS idx_%s_updated_at 
		ON %s(updated_at);
	`, p.table, p.table)

	if _, err := p.pool.Exec(ctx, createTimeIndexSQL); err != nil {
		p.logger.Warn("failed to create updated_at index, continuing", logger.Error(err))
	}

	if p.logger != nil {
		p.logger.Debug("schema initialized", logger.String("table", p.table))
	}

	return nil
}

// Save saves the agent state to PostgreSQL.
func (p *PostgresStateStore) Save(ctx context.Context, state *sdk.AgentState) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	if state.AgentID == "" {
		return fmt.Errorf("agent ID cannot be empty")
	}
	if state.SessionID == "" {
		return fmt.Errorf("session ID cannot be empty")
	}

	start := time.Now()
	defer func() {
		if p.metrics != nil {
			p.metrics.Histogram("forge.integrations.postgres.statestore.save_duration").Observe(time.Since(start).Seconds())
		}
	}()

	// Marshal state to JSON
	stateJSON, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Upsert with ON CONFLICT
	query := fmt.Sprintf(`
		INSERT INTO %s (agent_id, session_id, state, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		ON CONFLICT (agent_id, session_id) 
		DO UPDATE SET 
			state = EXCLUDED.state,
			updated_at = NOW()
	`, p.table)

	_, err = p.pool.Exec(ctx, query, state.AgentID, state.SessionID, stateJSON)
	if err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	if p.logger != nil {
		p.logger.Debug("saved state to postgres",
			logger.String("agent_id", state.AgentID),
			logger.String("session_id", state.SessionID),
			logger.Duration("duration", time.Since(start)))
	}

	if p.metrics != nil {
		p.metrics.Counter("forge.integrations.postgres.statestore.save").Inc()
	}

	return nil
}

// Load loads the agent state from PostgreSQL.
func (p *PostgresStateStore) Load(ctx context.Context, agentID, sessionID string) (*sdk.AgentState, error) {
	if agentID == "" {
		return nil, fmt.Errorf("agent ID cannot be empty")
	}
	if sessionID == "" {
		return nil, fmt.Errorf("session ID cannot be empty")
	}

	start := time.Now()
	defer func() {
		if p.metrics != nil {
			p.metrics.Histogram("forge.integrations.postgres.statestore.load_duration").Observe(time.Since(start).Seconds())
		}
	}()

	query := fmt.Sprintf(`
		SELECT state FROM %s
		WHERE agent_id = $1 AND session_id = $2
	`, p.table)

	var stateJSON []byte
	err := p.pool.QueryRow(ctx, query, agentID, sessionID).Scan(&stateJSON)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("state not found for agent %s, session %s", agentID, sessionID)
		}
		return nil, fmt.Errorf("failed to load state: %w", err)
	}

	// Unmarshal state
	var state sdk.AgentState
	if err := json.Unmarshal(stateJSON, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	if p.logger != nil {
		p.logger.Debug("loaded state from postgres",
			logger.String("agent_id", agentID),
			logger.String("session_id", sessionID),
			logger.Duration("duration", time.Since(start)))
	}

	if p.metrics != nil {
		p.metrics.Counter("forge.integrations.postgres.statestore.load").Inc()
	}

	return &state, nil
}

// Delete deletes the agent state from PostgreSQL.
func (p *PostgresStateStore) Delete(ctx context.Context, agentID, sessionID string) error {
	if agentID == "" {
		return fmt.Errorf("agent ID cannot be empty")
	}
	if sessionID == "" {
		return fmt.Errorf("session ID cannot be empty")
	}

	start := time.Now()

	query := fmt.Sprintf(`
		DELETE FROM %s
		WHERE agent_id = $1 AND session_id = $2
	`, p.table)

	_, err := p.pool.Exec(ctx, query, agentID, sessionID)
	if err != nil {
		return fmt.Errorf("failed to delete state: %w", err)
	}

	if p.logger != nil {
		p.logger.Debug("deleted state from postgres",
			logger.String("agent_id", agentID),
			logger.String("session_id", sessionID),
			logger.Duration("duration", time.Since(start)))
	}

	if p.metrics != nil {
		p.metrics.Counter("forge.integrations.postgres.statestore.delete").Inc()
	}

	return nil
}

// List lists all session IDs for an agent.
func (p *PostgresStateStore) List(ctx context.Context, agentID string) ([]string, error) {
	if agentID == "" {
		return nil, fmt.Errorf("agent ID cannot be empty")
	}

	start := time.Now()
	defer func() {
		if p.metrics != nil {
			p.metrics.Histogram("forge.integrations.postgres.statestore.list_duration").Observe(time.Since(start).Seconds())
		}
	}()

	query := fmt.Sprintf(`
		SELECT session_id FROM %s
		WHERE agent_id = $1
		ORDER BY updated_at DESC
	`, p.table)

	rows, err := p.pool.Query(ctx, query, agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}
	defer rows.Close()

	var sessions []string
	for rows.Next() {
		var sessionID string
		if err := rows.Scan(&sessionID); err != nil {
			return nil, fmt.Errorf("failed to scan session ID: %w", err)
		}
		sessions = append(sessions, sessionID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	if p.logger != nil {
		p.logger.Debug("listed sessions from postgres",
			logger.String("agent_id", agentID),
			logger.Int("count", len(sessions)),
			logger.Duration("duration", time.Since(start)))
	}

	if p.metrics != nil {
		p.metrics.Counter("forge.integrations.postgres.statestore.list").Inc()
		p.metrics.Histogram("forge.integrations.postgres.statestore.sessions").Observe(float64(len(sessions)))
	}

	return sessions, nil
}

// Close closes the PostgreSQL connection pool.
func (p *PostgresStateStore) Close() error {
	if p.pool != nil {
		p.pool.Close()
	}

	if p.logger != nil {
		p.logger.Info("postgres state store closed")
	}

	return nil
}

// HealthCheck verifies the database connection is healthy.
func (p *PostgresStateStore) HealthCheck(ctx context.Context) error {
	return p.pool.Ping(ctx)
}

// Count returns the total number of states in the store.
func (p *PostgresStateStore) Count(ctx context.Context) (int64, error) {
	query := fmt.Sprintf(`SELECT COUNT(*) FROM %s`, p.table)

	var count int64
	err := p.pool.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count states: %w", err)
	}

	return count, nil
}
