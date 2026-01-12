package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	sdk "github.com/xraph/ai-sdk"
	logger "github.com/xraph/go-utils/log"
	"github.com/xraph/go-utils/metrics"
)

// MemoryStateStore implements the sdk.StateStore interface using in-memory storage.
// Perfect for testing and local development. Not suitable for production distributed systems.
type MemoryStateStore struct {
	states   map[string]*stateEntry // key: "agentID:sessionID"
	sessions map[string][]string    // key: agentID, value: []sessionIDs
	mu       sync.RWMutex
	logger   logger.Logger
	metrics  metrics.Metrics
	ttl      time.Duration // Optional TTL for states
}

type stateEntry struct {
	state     *sdk.AgentState
	createdAt time.Time
	expiresAt *time.Time // nil means no expiration
}

// Config provides configuration for the MemoryStateStore.
type Config struct {
	TTL     time.Duration // Optional: TTL for states (0 = no expiration)
	Logger  logger.Logger
	Metrics metrics.Metrics
}

// NewMemoryStateStore creates a new in-memory state store.
func NewMemoryStateStore(cfg Config) *MemoryStateStore {
	store := &MemoryStateStore{
		states:   make(map[string]*stateEntry),
		sessions: make(map[string][]string),
		logger:   cfg.Logger,
		metrics:  cfg.Metrics,
		ttl:      cfg.TTL,
	}

	// Start cleanup goroutine if TTL is set
	if cfg.TTL > 0 {
		go store.cleanupExpired()
	}

	if store.logger != nil {
		store.logger.Info("memory state store initialized",
			logger.Duration("ttl", cfg.TTL))
	}

	return store
}

// Save saves the agent state to memory.
func (m *MemoryStateStore) Save(ctx context.Context, state *sdk.AgentState) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	if state.AgentID == "" {
		return fmt.Errorf("agent ID cannot be empty")
	}
	if state.SessionID == "" {
		return fmt.Errorf("session ID cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	key := m.key(state.AgentID, state.SessionID)

	// Create entry
	entry := &stateEntry{
		state:     state,
		createdAt: time.Now(),
	}

	if m.ttl > 0 {
		expiresAt := time.Now().Add(m.ttl)
		entry.expiresAt = &expiresAt
	}

	// Update states map
	m.states[key] = entry

	// Update sessions tracking
	if sessions, exists := m.sessions[state.AgentID]; exists {
		// Check if session already tracked
		found := false
		for _, sid := range sessions {
			if sid == state.SessionID {
				found = true
				break
			}
		}
		if !found {
			m.sessions[state.AgentID] = append(sessions, state.SessionID)
		}
	} else {
		m.sessions[state.AgentID] = []string{state.SessionID}
	}

	if m.logger != nil {
		m.logger.Debug("saved state to memory",
			logger.String("agent_id", state.AgentID),
			logger.String("session_id", state.SessionID))
	}

	if m.metrics != nil {
		m.metrics.Counter("forge.integrations.memory.statestore.save").Inc()
	}

	return nil
}

// Load loads the agent state from memory.
func (m *MemoryStateStore) Load(ctx context.Context, agentID, sessionID string) (*sdk.AgentState, error) {
	if agentID == "" {
		return nil, fmt.Errorf("agent ID cannot be empty")
	}
	if sessionID == "" {
		return nil, fmt.Errorf("session ID cannot be empty")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	key := m.key(agentID, sessionID)
	entry, exists := m.states[key]
	if !exists {
		return nil, fmt.Errorf("state not found for agent %s, session %s", agentID, sessionID)
	}

	// Check expiration
	if entry.expiresAt != nil && time.Now().After(*entry.expiresAt) {
		return nil, fmt.Errorf("state expired for agent %s, session %s", agentID, sessionID)
	}

	if m.logger != nil {
		m.logger.Debug("loaded state from memory",
			logger.String("agent_id", agentID),
			logger.String("session_id", sessionID))
	}

	if m.metrics != nil {
		m.metrics.Counter("forge.integrations.memory.statestore.load").Inc()
	}

	// Return a copy to prevent external modification
	stateCopy := *entry.state
	return &stateCopy, nil
}

// Delete deletes the agent state from memory.
func (m *MemoryStateStore) Delete(ctx context.Context, agentID, sessionID string) error {
	if agentID == "" {
		return fmt.Errorf("agent ID cannot be empty")
	}
	if sessionID == "" {
		return fmt.Errorf("session ID cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	key := m.key(agentID, sessionID)
	delete(m.states, key)

	// Update sessions tracking
	if sessions, exists := m.sessions[agentID]; exists {
		newSessions := []string{}
		for _, sid := range sessions {
			if sid != sessionID {
				newSessions = append(newSessions, sid)
			}
		}
		if len(newSessions) > 0 {
			m.sessions[agentID] = newSessions
		} else {
			delete(m.sessions, agentID)
		}
	}

	if m.logger != nil {
		m.logger.Debug("deleted state from memory",
			logger.String("agent_id", agentID),
			logger.String("session_id", sessionID))
	}

	if m.metrics != nil {
		m.metrics.Counter("forge.integrations.memory.statestore.delete").Inc()
	}

	return nil
}

// List lists all session IDs for an agent.
func (m *MemoryStateStore) List(ctx context.Context, agentID string) ([]string, error) {
	if agentID == "" {
		return nil, fmt.Errorf("agent ID cannot be empty")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions, exists := m.sessions[agentID]
	if !exists {
		return []string{}, nil
	}

	// Return a copy
	result := make([]string, len(sessions))
	copy(result, sessions)

	if m.logger != nil {
		m.logger.Debug("listed sessions from memory",
			logger.String("agent_id", agentID),
			logger.Int("count", len(result)))
	}

	if m.metrics != nil {
		m.metrics.Counter("forge.integrations.memory.statestore.list").Inc()
		m.metrics.Histogram("forge.integrations.memory.statestore.sessions").Observe(float64(len(result)))
	}

	return result, nil
}

// Clear removes all states (useful for testing).
func (m *MemoryStateStore) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.states = make(map[string]*stateEntry)
	m.sessions = make(map[string][]string)

	if m.logger != nil {
		m.logger.Debug("cleared all states from memory")
	}

	if m.metrics != nil {
		m.metrics.Counter("forge.integrations.memory.statestore.clear").Inc()
	}
}

// Count returns the total number of states in memory.
func (m *MemoryStateStore) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.states)
}

// Helper methods

func (m *MemoryStateStore) key(agentID, sessionID string) string {
	return fmt.Sprintf("%s:%s", agentID, sessionID)
}

func (m *MemoryStateStore) cleanupExpired() {
	ticker := time.NewTicker(m.ttl / 2) // Check twice per TTL period
	defer ticker.Stop()

	for range ticker.C {
		m.mu.Lock()
		now := time.Now()
		expired := []string{}

		for key, entry := range m.states {
			if entry.expiresAt != nil && now.After(*entry.expiresAt) {
				expired = append(expired, key)
			}
		}

		for _, key := range expired {
			delete(m.states, key)
		}

		if len(expired) > 0 && m.logger != nil {
			m.logger.Debug("cleaned up expired states",
				logger.Int("count", len(expired)))
		}

		m.mu.Unlock()
	}
}

