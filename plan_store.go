package sdk

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	// ErrPlanNotFound is returned when a plan doesn't exist
	ErrPlanNotFound = errors.New("plan not found")

	// ErrPlanExists is returned when trying to create a plan with existing ID
	ErrPlanExists = errors.New("plan already exists")
)

// PlanStore provides persistence for plans.
// Implementations can use memory, files, databases, etc.
type PlanStore interface {
	// Save persists a plan
	Save(ctx context.Context, plan *Plan) error

	// Load retrieves a plan by ID
	Load(ctx context.Context, planID string) (*Plan, error)

	// Delete removes a plan
	Delete(ctx context.Context, planID string) error

	// List returns all plans for an agent
	List(ctx context.Context, agentID string) ([]*Plan, error)

	// ListByStatus returns plans with specific status
	ListByStatus(ctx context.Context, agentID string, status PlanStatus) ([]*Plan, error)
}

// InMemoryPlanStore is a simple in-memory implementation.
// Useful for testing and development.
type InMemoryPlanStore struct {
	plans map[string]*Plan
	mu    sync.RWMutex
}

// NewInMemoryPlanStore creates a new in-memory plan store.
func NewInMemoryPlanStore() *InMemoryPlanStore {
	return &InMemoryPlanStore{
		plans: make(map[string]*Plan),
	}
}

// Save persists a plan in memory.
func (s *InMemoryPlanStore) Save(ctx context.Context, plan *Plan) error {
	if plan == nil {
		return errors.New("plan cannot be nil")
	}

	if plan.ID == "" {
		return errors.New("plan ID is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Update timestamp
	plan.UpdatedAt = time.Now()

	// Clone to prevent external modifications
	s.plans[plan.ID] = plan.Clone()

	return nil
}

// Load retrieves a plan from memory.
func (s *InMemoryPlanStore) Load(ctx context.Context, planID string) (*Plan, error) {
	if planID == "" {
		return nil, errors.New("plan ID is required")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	plan, exists := s.plans[planID]
	if !exists {
		return nil, ErrPlanNotFound
	}

	// Return a clone to prevent external modifications
	return plan.Clone(), nil
}

// Delete removes a plan from memory.
func (s *InMemoryPlanStore) Delete(ctx context.Context, planID string) error {
	if planID == "" {
		return errors.New("plan ID is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.plans[planID]; !exists {
		return ErrPlanNotFound
	}

	delete(s.plans, planID)
	return nil
}

// List returns all plans for an agent.
func (s *InMemoryPlanStore) List(ctx context.Context, agentID string) ([]*Plan, error) {
	if agentID == "" {
		return nil, errors.New("agent ID is required")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	var plans []*Plan
	for _, plan := range s.plans {
		if plan.AgentID == agentID {
			plans = append(plans, plan.Clone())
		}
	}

	return plans, nil
}

// ListByStatus returns plans with specific status.
func (s *InMemoryPlanStore) ListByStatus(ctx context.Context, agentID string, status PlanStatus) ([]*Plan, error) {
	if agentID == "" {
		return nil, errors.New("agent ID is required")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	var plans []*Plan
	for _, plan := range s.plans {
		if plan.AgentID == agentID && plan.Status == status {
			plans = append(plans, plan.Clone())
		}
	}

	return plans, nil
}

// Clear removes all plans (useful for testing).
func (s *InMemoryPlanStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.plans = make(map[string]*Plan)
}

// Count returns the total number of plans stored.
func (s *InMemoryPlanStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.plans)
}

// PlanStoreWrapper wraps a PlanStore with additional functionality.
type PlanStoreWrapper struct {
	store     PlanStore
	onSave    func(*Plan)
	onLoad    func(*Plan)
	onDelete  func(string)
	cacheSize int
	cache     map[string]*cachedPlan
	cacheMu   sync.RWMutex
}

type cachedPlan struct {
	plan      *Plan
	timestamp time.Time
}

// NewPlanStoreWrapper creates a wrapped plan store with caching.
func NewPlanStoreWrapper(store PlanStore, cacheSize int) *PlanStoreWrapper {
	if cacheSize <= 0 {
		cacheSize = 100 // default cache size
	}

	return &PlanStoreWrapper{
		store:     store,
		cacheSize: cacheSize,
		cache:     make(map[string]*cachedPlan),
	}
}

// WithCallbacks adds callbacks for store operations.
func (w *PlanStoreWrapper) WithCallbacks(
	onSave func(*Plan),
	onLoad func(*Plan),
	onDelete func(string),
) *PlanStoreWrapper {
	w.onSave = onSave
	w.onLoad = onLoad
	w.onDelete = onDelete
	return w
}

// Save persists a plan and updates cache.
func (w *PlanStoreWrapper) Save(ctx context.Context, plan *Plan) error {
	if err := w.store.Save(ctx, plan); err != nil {
		return err
	}

	// Update cache
	w.cacheMu.Lock()
	w.cache[plan.ID] = &cachedPlan{
		plan:      plan.Clone(),
		timestamp: time.Now(),
	}
	w.evictOldCache()
	w.cacheMu.Unlock()

	if w.onSave != nil {
		w.onSave(plan)
	}

	return nil
}

// Load retrieves a plan, checking cache first.
func (w *PlanStoreWrapper) Load(ctx context.Context, planID string) (*Plan, error) {
	// Check cache first
	w.cacheMu.RLock()
	if cached, exists := w.cache[planID]; exists {
		w.cacheMu.RUnlock()
		if w.onLoad != nil {
			w.onLoad(cached.plan)
		}
		return cached.plan.Clone(), nil
	}
	w.cacheMu.RUnlock()

	// Load from store
	plan, err := w.store.Load(ctx, planID)
	if err != nil {
		return nil, err
	}

	// Update cache
	w.cacheMu.Lock()
	w.cache[planID] = &cachedPlan{
		plan:      plan.Clone(),
		timestamp: time.Now(),
	}
	w.cacheMu.Unlock()

	if w.onLoad != nil {
		w.onLoad(plan)
	}

	return plan, nil
}

// Delete removes a plan from store and cache.
func (w *PlanStoreWrapper) Delete(ctx context.Context, planID string) error {
	if err := w.store.Delete(ctx, planID); err != nil {
		return err
	}

	// Remove from cache
	w.cacheMu.Lock()
	delete(w.cache, planID)
	w.cacheMu.Unlock()

	if w.onDelete != nil {
		w.onDelete(planID)
	}

	return nil
}

// List returns all plans for an agent.
func (w *PlanStoreWrapper) List(ctx context.Context, agentID string) ([]*Plan, error) {
	return w.store.List(ctx, agentID)
}

// ListByStatus returns plans with specific status.
func (w *PlanStoreWrapper) ListByStatus(ctx context.Context, agentID string, status PlanStatus) ([]*Plan, error) {
	return w.store.ListByStatus(ctx, agentID, status)
}

// evictOldCache removes oldest entries when cache is full.
func (w *PlanStoreWrapper) evictOldCache() {
	if len(w.cache) <= w.cacheSize {
		return
	}

	// Find oldest entry
	var oldestID string
	var oldestTime time.Time
	first := true

	for id, cached := range w.cache {
		if first || cached.timestamp.Before(oldestTime) {
			oldestID = id
			oldestTime = cached.timestamp
			first = false
		}
	}

	delete(w.cache, oldestID)
}

// ClearCache clears the cache.
func (w *PlanStoreWrapper) ClearCache() {
	w.cacheMu.Lock()
	defer w.cacheMu.Unlock()

	w.cache = make(map[string]*cachedPlan)
}

// generatePlanID generates a unique plan ID.
func generatePlanID() string {
	return fmt.Sprintf("plan_%d", time.Now().UnixNano())
}

// generateStepID generates a unique step ID.
func generateStepID(planID string, index int) string {
	return fmt.Sprintf("%s_step_%d", planID, index)
}
