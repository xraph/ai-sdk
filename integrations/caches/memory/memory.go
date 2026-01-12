package memory

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	logger "github.com/xraph/go-utils/log"
	"github.com/xraph/go-utils/metrics"
)

// MemoryCacheStore implements a simple in-memory cache with LRU eviction and TTL support.
// Perfect for testing and local development. Not suitable for distributed production systems.
type MemoryCacheStore struct {
	entries  map[string]*cacheEntry
	lruList  *lruList
	maxSize  int
	mu       sync.RWMutex
	logger   logger.Logger
	metrics  metrics.Metrics
	stopChan chan struct{}
}

type cacheEntry struct {
	key       string
	value     []byte
	expiresAt *time.Time
	lruNode   *lruNode
}

// Config provides configuration for the MemoryCacheStore.
type Config struct {
	MaxSize int // Maximum number of entries (0 = unlimited)
	Logger  logger.Logger
	Metrics metrics.Metrics
}

// NewMemoryCacheStore creates a new in-memory cache store.
func NewMemoryCacheStore(cfg Config) *MemoryCacheStore {
	store := &MemoryCacheStore{
		entries:  make(map[string]*cacheEntry),
		lruList:  newLRUList(),
		maxSize:  cfg.MaxSize,
		logger:   cfg.Logger,
		metrics:  cfg.Metrics,
		stopChan: make(chan struct{}),
	}

	// Start cleanup goroutine
	go store.cleanupExpired()

	if store.logger != nil {
		store.logger.Info("memory cache store initialized",
			logger.Int("max_size", cfg.MaxSize))
	}

	return store
}

// Get retrieves a value from the cache.
func (m *MemoryCacheStore) Get(ctx context.Context, key string) ([]byte, bool, error) {
	m.mu.RLock()
	entry, exists := m.entries[key]
	m.mu.RUnlock()

	if !exists {
		if m.metrics != nil {
			m.metrics.Counter("forge.integrations.memory.cache.get",
				metrics.WithLabel("hit", "false")).Inc()
		}
		return nil, false, nil
	}

	// Check expiration
	if entry.expiresAt != nil && time.Now().After(*entry.expiresAt) {
		m.mu.Lock()
		m.remove(entry)
		m.mu.Unlock()

		if m.metrics != nil {
			m.metrics.Counter("forge.integrations.memory.cache.get",
				metrics.WithLabel("hit", "false")).Inc()
		}
		return nil, false, nil
	}

	// Update LRU
	m.mu.Lock()
	m.lruList.moveToFront(entry.lruNode)
	m.mu.Unlock()

	if m.logger != nil {
		m.logger.Debug("cache hit", logger.String("key", key))
	}

	if m.metrics != nil {
		m.metrics.Counter("forge.integrations.memory.cache.get",
			metrics.WithLabel("hit", "true")).Inc()
	}

	// Return a copy
	valueCopy := make([]byte, len(entry.value))
	copy(valueCopy, entry.value)
	return valueCopy, true, nil
}

// Set sets a value in the cache with an optional TTL.
func (m *MemoryCacheStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if entry exists
	if existing, exists := m.entries[key]; exists {
		// Update existing entry
		existing.value = value
		if ttl > 0 {
			expiresAt := time.Now().Add(ttl)
			existing.expiresAt = &expiresAt
		} else {
			existing.expiresAt = nil
		}
		m.lruList.moveToFront(existing.lruNode)
	} else {
		// Create new entry
		entry := &cacheEntry{
			key:   key,
			value: value,
		}

		if ttl > 0 {
			expiresAt := time.Now().Add(ttl)
			entry.expiresAt = &expiresAt
		}

		// Add to LRU list
		node := m.lruList.pushFront(key)
		entry.lruNode = node

		m.entries[key] = entry

		// Evict if over max size
		if m.maxSize > 0 && len(m.entries) > m.maxSize {
			m.evictLRU()
		}
	}

	if m.logger != nil {
		m.logger.Debug("cache set", logger.String("key", key), logger.Duration("ttl", ttl))
	}

	if m.metrics != nil {
		m.metrics.Counter("forge.integrations.memory.cache.set").Inc()
	}

	return nil
}

// Delete removes a value from the cache.
func (m *MemoryCacheStore) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry, exists := m.entries[key]
	if !exists {
		return nil
	}

	m.remove(entry)

	if m.logger != nil {
		m.logger.Debug("cache delete", logger.String("key", key))
	}

	if m.metrics != nil {
		m.metrics.Counter("forge.integrations.memory.cache.delete").Inc()
	}

	return nil
}

// Clear clears all entries from the cache.
func (m *MemoryCacheStore) Clear(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.entries = make(map[string]*cacheEntry)
	m.lruList = newLRUList()

	if m.logger != nil {
		m.logger.Debug("cache cleared")
	}

	if m.metrics != nil {
		m.metrics.Counter("forge.integrations.memory.cache.clear").Inc()
	}

	return nil
}

// Size returns the current number of entries in the cache.
func (m *MemoryCacheStore) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.entries)
}

// Close stops the cleanup goroutine.
func (m *MemoryCacheStore) Close() error {
	close(m.stopChan)
	return nil
}

// Helper methods

func (m *MemoryCacheStore) remove(entry *cacheEntry) {
	delete(m.entries, entry.key)
	m.lruList.remove(entry.lruNode)
}

func (m *MemoryCacheStore) evictLRU() {
	if m.lruList.tail == nil {
		return
	}

	key := m.lruList.tail.key
	entry, exists := m.entries[key]
	if !exists {
		return
	}

	m.remove(entry)

	if m.logger != nil {
		m.logger.Debug("evicted LRU entry", logger.String("key", key))
	}

	if m.metrics != nil {
		m.metrics.Counter("forge.integrations.memory.cache.evictions").Inc()
	}
}

func (m *MemoryCacheStore) cleanupExpired() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.mu.Lock()
			now := time.Now()
			expired := []*cacheEntry{}

			for _, entry := range m.entries {
				if entry.expiresAt != nil && now.After(*entry.expiresAt) {
					expired = append(expired, entry)
				}
			}

			for _, entry := range expired {
				m.remove(entry)
			}

			if len(expired) > 0 && m.logger != nil {
				m.logger.Debug("cleaned up expired entries",
					logger.Int("count", len(expired)))
			}

			m.mu.Unlock()

		case <-m.stopChan:
			return
		}
	}
}

// LRU list implementation

type lruNode struct {
	key  string
	prev *lruNode
	next *lruNode
}

type lruList struct {
	head *lruNode
	tail *lruNode
}

func newLRUList() *lruList {
	return &lruList{}
}

func (l *lruList) pushFront(key string) *lruNode {
	node := &lruNode{key: key}

	if l.head == nil {
		l.head = node
		l.tail = node
	} else {
		node.next = l.head
		l.head.prev = node
		l.head = node
	}

	return node
}

func (l *lruList) moveToFront(node *lruNode) {
	if node == l.head {
		return
	}

	// Remove from current position
	if node.prev != nil {
		node.prev.next = node.next
	}
	if node.next != nil {
		node.next.prev = node.prev
	}
	if node == l.tail {
		l.tail = node.prev
	}

	// Move to front
	node.prev = nil
	node.next = l.head
	l.head.prev = node
	l.head = node
}

func (l *lruList) remove(node *lruNode) {
	if node.prev != nil {
		node.prev.next = node.next
	} else {
		l.head = node.next
	}

	if node.next != nil {
		node.next.prev = node.prev
	} else {
		l.tail = node.prev
	}
}
