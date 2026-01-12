package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	sdk "github.com/xraph/ai-sdk"
	logger "github.com/xraph/go-utils/log"
	"github.com/xraph/go-utils/metrics"
)

// RedisStateStore implements StateStore using Redis.
type RedisStateStore struct {
	client  redis.UniversalClient
	prefix  string
	logger  logger.Logger
	metrics metrics.Metrics
}

// Config configures the Redis state store.
type Config struct {
	// Redis connection
	Addrs    []string // Redis addresses (single for standalone, multiple for cluster/sentinel)
	Password string   // Optional password
	DB       int      // Database number (standalone only)

	// Optional
	KeyPrefix    string        // Key prefix (default: "forge:state:")
	DialTimeout  time.Duration // Dial timeout (default: 5s)
	ReadTimeout  time.Duration // Read timeout (default: 3s)
	WriteTimeout time.Duration // Write timeout (default: 3s)
	PoolSize     int           // Connection pool size (default: 10)

	// Cluster/Sentinel
	MasterName    string   // Sentinel master name
	SentinelAddrs []string // Sentinel addresses

	// Observability
	Logger  logger.Logger
	Metrics metrics.Metrics
}

// NewRedisStateStore creates a new Redis-based state store.
func NewRedisStateStore(ctx context.Context, cfg Config) (*RedisStateStore, error) {
	if len(cfg.Addrs) == 0 {
		cfg.Addrs = []string{"localhost:6379"}
	}

	// Set defaults
	if cfg.KeyPrefix == "" {
		cfg.KeyPrefix = "forge:state:"
	}
	if cfg.DialTimeout == 0 {
		cfg.DialTimeout = 5 * time.Second
	}
	if cfg.ReadTimeout == 0 {
		cfg.ReadTimeout = 3 * time.Second
	}
	if cfg.WriteTimeout == 0 {
		cfg.WriteTimeout = 3 * time.Second
	}
	if cfg.PoolSize == 0 {
		cfg.PoolSize = 10
	}

	// Create Redis client
	var client redis.UniversalClient
	if len(cfg.SentinelAddrs) > 0 {
		// Sentinel mode
		client = redis.NewFailoverClient(&redis.FailoverOptions{
			MasterName:    cfg.MasterName,
			SentinelAddrs: cfg.SentinelAddrs,
			Password:      cfg.Password,
			DB:            cfg.DB,
			DialTimeout:   cfg.DialTimeout,
			ReadTimeout:   cfg.ReadTimeout,
			WriteTimeout:  cfg.WriteTimeout,
			PoolSize:      cfg.PoolSize,
		})
	} else if len(cfg.Addrs) > 1 {
		// Cluster mode
		client = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:        cfg.Addrs,
			Password:     cfg.Password,
			DialTimeout:  cfg.DialTimeout,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
			PoolSize:     cfg.PoolSize,
		})
	} else {
		// Standalone mode
		client = redis.NewClient(&redis.Options{
			Addr:         cfg.Addrs[0],
			Password:     cfg.Password,
			DB:           cfg.DB,
			DialTimeout:  cfg.DialTimeout,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
			PoolSize:     cfg.PoolSize,
		})
	}

	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	store := &RedisStateStore{
		client:  client,
		prefix:  cfg.KeyPrefix,
		logger:  cfg.Logger,
		metrics: cfg.Metrics,
	}

	if store.logger != nil {
		store.logger.Info("redis state store initialized",
			logger.Strings("addrs", cfg.Addrs))
	}

	return store, nil
}

// Save saves the agent state.
func (r *RedisStateStore) Save(ctx context.Context, state *sdk.AgentState) error {
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
		if r.metrics != nil {
			r.metrics.Histogram("forge.integrations.redis.state.save_duration").Observe(time.Since(start).Seconds())
		}
	}()

	// Serialize state
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Save to Redis
	key := r.stateKey(state.AgentID, state.SessionID)
	if err := r.client.Set(ctx, key, data, 0).Err(); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	// Add to session list
	listKey := r.sessionListKey(state.AgentID)
	if err := r.client.SAdd(ctx, listKey, state.SessionID).Err(); err != nil {
		return fmt.Errorf("failed to add to session list: %w", err)
	}

	if r.logger != nil {
		r.logger.Debug("saved state to redis",
			logger.String("agent_id", state.AgentID),
			logger.String("session_id", state.SessionID))
	}

	if r.metrics != nil {
		r.metrics.Counter("forge.integrations.redis.state.save").Inc()
	}

	return nil
}

// Load loads the agent state.
func (r *RedisStateStore) Load(ctx context.Context, agentID, sessionID string) (*sdk.AgentState, error) {
	if agentID == "" {
		return nil, fmt.Errorf("agent ID cannot be empty")
	}
	if sessionID == "" {
		return nil, fmt.Errorf("session ID cannot be empty")
	}

	start := time.Now()
	defer func() {
		if r.metrics != nil {
			r.metrics.Histogram("forge.integrations.redis.state.load_duration").Observe(time.Since(start).Seconds())
		}
	}()

	key := r.stateKey(agentID, sessionID)
	data, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, fmt.Errorf("state not found for agent %s, session %s", agentID, sessionID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load state: %w", err)
	}

	var state sdk.AgentState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	if r.logger != nil {
		r.logger.Debug("loaded state from redis",
			logger.String("agent_id", agentID),
			logger.String("session_id", sessionID))
	}

	if r.metrics != nil {
		r.metrics.Counter("forge.integrations.redis.state.load").Inc()
	}

	return &state, nil
}

// Delete deletes the agent state.
func (r *RedisStateStore) Delete(ctx context.Context, agentID, sessionID string) error {
	if agentID == "" {
		return fmt.Errorf("agent ID cannot be empty")
	}
	if sessionID == "" {
		return fmt.Errorf("session ID cannot be empty")
	}

	start := time.Now()

	// Delete state
	key := r.stateKey(agentID, sessionID)
	if err := r.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete state: %w", err)
	}

	// Remove from session list
	listKey := r.sessionListKey(agentID)
	if err := r.client.SRem(ctx, listKey, sessionID).Err(); err != nil {
		return fmt.Errorf("failed to remove from session list: %w", err)
	}

	if r.logger != nil {
		r.logger.Debug("deleted state from redis",
			logger.String("agent_id", agentID),
			logger.String("session_id", sessionID),
			logger.Duration("duration", time.Since(start)))
	}

	if r.metrics != nil {
		r.metrics.Counter("forge.integrations.redis.state.delete").Inc()
	}

	return nil
}

// List lists all sessions for an agent.
func (r *RedisStateStore) List(ctx context.Context, agentID string) ([]string, error) {
	if agentID == "" {
		return nil, fmt.Errorf("agent ID cannot be empty")
	}

	start := time.Now()
	defer func() {
		if r.metrics != nil {
			r.metrics.Histogram("forge.integrations.redis.state.list_duration").Observe(time.Since(start).Seconds())
		}
	}()

	listKey := r.sessionListKey(agentID)
	sessions, err := r.client.SMembers(ctx, listKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	if r.logger != nil {
		r.logger.Debug("listed sessions from redis",
			logger.String("agent_id", agentID),
			logger.Int("count", len(sessions)))
	}

	if r.metrics != nil {
		r.metrics.Counter("forge.integrations.redis.state.list").Inc()
	}

	return sessions, nil
}

// Close closes the Redis connection.
func (r *RedisStateStore) Close() error {
	if err := r.client.Close(); err != nil {
		return fmt.Errorf("failed to close Redis client: %w", err)
	}

	if r.logger != nil {
		r.logger.Info("redis state store closed")
	}

	return nil
}

// Helper methods

func (r *RedisStateStore) stateKey(agentID, sessionID string) string {
	return fmt.Sprintf("%s%s:%s", r.prefix, agentID, sessionID)
}

func (r *RedisStateStore) sessionListKey(agentID string) string {
	return fmt.Sprintf("%ssessions:%s", r.prefix, agentID)
}

// Pattern returns the key pattern for all states.
func (r *RedisStateStore) Pattern() string {
	return r.prefix + "*"
}

// Count returns the number of stored states (expensive operation).
func (r *RedisStateStore) Count(ctx context.Context) (int64, error) {
	var cursor uint64
	var count int64
	pattern := r.Pattern()

	for {
		keys, nextCursor, err := r.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return 0, fmt.Errorf("failed to scan keys: %w", err)
		}

		// Filter out session lists
		for _, key := range keys {
			if !strings.Contains(key, "sessions:") {
				count++
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return count, nil
}
