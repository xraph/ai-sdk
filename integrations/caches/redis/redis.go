package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	logger "github.com/xraph/go-utils/log"
	"github.com/xraph/go-utils/metrics"
)

// RedisCacheStore implements CacheStore using Redis.
type RedisCacheStore struct {
	client  redis.UniversalClient
	prefix  string
	logger  logger.Logger
	metrics metrics.Metrics
}

// Config configures the Redis cache store.
type Config struct {
	// Redis connection
	Addrs    []string // Redis addresses
	Password string   // Optional password
	DB       int      // Database number (standalone only)

	// Optional
	KeyPrefix    string        // Key prefix (default: "forge:cache:")
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

// NewRedisCacheStore creates a new Redis-based cache store.
func NewRedisCacheStore(ctx context.Context, cfg Config) (*RedisCacheStore, error) {
	if len(cfg.Addrs) == 0 {
		cfg.Addrs = []string{"localhost:6379"}
	}

	// Set defaults
	if cfg.KeyPrefix == "" {
		cfg.KeyPrefix = "forge:cache:"
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

	// Create Redis client (same logic as state store)
	var client redis.UniversalClient
	if len(cfg.SentinelAddrs) > 0 {
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
		client = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:        cfg.Addrs,
			Password:     cfg.Password,
			DialTimeout:  cfg.DialTimeout,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
			PoolSize:     cfg.PoolSize,
		})
	} else {
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

	store := &RedisCacheStore{
		client:  client,
		prefix:  cfg.KeyPrefix,
		logger:  cfg.Logger,
		metrics: cfg.Metrics,
	}

	if store.logger != nil {
		store.logger.Info("redis cache store initialized",
			logger.Strings("addrs", cfg.Addrs))
	}

	return store, nil
}

// Get retrieves a value from the cache.
func (r *RedisCacheStore) Get(ctx context.Context, key string) ([]byte, bool, error) {
	if key == "" {
		return nil, false, fmt.Errorf("key cannot be empty")
	}

	start := time.Now()
	defer func() {
		if r.metrics != nil {
			r.metrics.Histogram("forge.integrations.redis.cache.get_duration").Observe(time.Since(start).Seconds())
		}
	}()

	fullKey := r.prefix + key
	data, err := r.client.Get(ctx, fullKey).Bytes()
	if err == redis.Nil {
		if r.metrics != nil {
			r.metrics.Counter("forge.integrations.redis.cache.miss").Inc()
		}
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("failed to get from cache: %w", err)
	}

	if r.logger != nil {
		r.logger.Debug("cache hit", logger.String("key", key))
	}

	if r.metrics != nil {
		r.metrics.Counter("forge.integrations.redis.cache.hit").Inc()
		r.metrics.Counter("forge.integrations.redis.cache.get").Inc()
	}

	return data, true, nil
}

// Set stores a value in the cache with TTL.
func (r *RedisCacheStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}

	start := time.Now()
	defer func() {
		if r.metrics != nil {
			r.metrics.Histogram("forge.integrations.redis.cache.set_duration").Observe(time.Since(start).Seconds())
		}
	}()

	fullKey := r.prefix + key
	if err := r.client.Set(ctx, fullKey, value, ttl).Err(); err != nil {
		return fmt.Errorf("failed to set cache: %w", err)
	}

	if r.logger != nil {
		r.logger.Debug("cache set",
			logger.String("key", key),
			logger.Duration("ttl", ttl))
	}

	if r.metrics != nil {
		r.metrics.Counter("forge.integrations.redis.cache.set").Inc()
	}

	return nil
}

// Delete removes a value from the cache.
func (r *RedisCacheStore) Delete(ctx context.Context, key string) error {
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}

	start := time.Now()

	fullKey := r.prefix + key
	if err := r.client.Del(ctx, fullKey).Err(); err != nil {
		return fmt.Errorf("failed to delete from cache: %w", err)
	}

	if r.logger != nil {
		r.logger.Debug("cache delete",
			logger.String("key", key),
			logger.Duration("duration", time.Since(start)))
	}

	if r.metrics != nil {
		r.metrics.Counter("forge.integrations.redis.cache.delete").Inc()
	}

	return nil
}

// Clear removes all values from the cache with the prefix.
func (r *RedisCacheStore) Clear(ctx context.Context) error {
	start := time.Now()

	// Scan and delete all keys with prefix
	var cursor uint64
	var deletedCount int64
	pattern := r.prefix + "*"

	for {
		keys, nextCursor, err := r.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return fmt.Errorf("failed to scan keys: %w", err)
		}

		if len(keys) > 0 {
			deleted, err := r.client.Del(ctx, keys...).Result()
			if err != nil {
				return fmt.Errorf("failed to delete keys: %w", err)
			}
			deletedCount += deleted
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	if r.logger != nil {
		r.logger.Info("cache cleared",
			logger.Int64("deleted", deletedCount),
			logger.Duration("duration", time.Since(start)))
	}

	if r.metrics != nil {
		r.metrics.Counter("forge.integrations.redis.cache.clear").Inc()
	}

	return nil
}

// Close closes the Redis connection.
func (r *RedisCacheStore) Close() error {
	if err := r.client.Close(); err != nil {
		return fmt.Errorf("failed to close Redis client: %w", err)
	}

	if r.logger != nil {
		r.logger.Info("redis cache store closed")
	}

	return nil
}

// Stats returns cache statistics.
func (r *RedisCacheStore) Stats(ctx context.Context) (*Stats, error) {
	info, err := r.client.Info(ctx, "stats").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	// Parse info string (simplified)
	stats := &Stats{
		RawInfo: info,
	}

	// Count keys with prefix
	var cursor uint64
	pattern := r.prefix + "*"
	for {
		keys, nextCursor, err := r.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to scan keys: %w", err)
		}

		stats.KeyCount += int64(len(keys))

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return stats, nil
}

// Stats represents cache statistics.
type Stats struct {
	KeyCount int64
	RawInfo  string
}
