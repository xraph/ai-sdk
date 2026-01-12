package memory

import (
	"context"
	"fmt"
	"testing"
	"time"

	sdk "github.com/xraph/ai-sdk"
)

func TestNewMemoryStateStore(t *testing.T) {
	store := NewMemoryStateStore(Config{})
	if store == nil {
		t.Fatal("expected non-nil store")
	}

	if store.Count() != 0 {
		t.Errorf("new store should be empty, got %d states", store.Count())
	}
}

func TestMemoryStateStore_Save(t *testing.T) {
	store := NewMemoryStateStore(Config{})
	ctx := context.Background()

	state := &sdk.AgentState{
		AgentID:   "agent-1",
		SessionID: "session-1",
		Context:   map[string]any{"key": "value"},
	}

	err := store.Save(ctx, state)
	if err != nil {
		t.Errorf("Save() error = %v", err)
	}

	if store.Count() != 1 {
		t.Errorf("expected 1 state, got %d", store.Count())
	}
}

func TestMemoryStateStore_SaveNil(t *testing.T) {
	store := NewMemoryStateStore(Config{})
	ctx := context.Background()

	err := store.Save(ctx, nil)
	if err == nil {
		t.Error("Save(nil) should return error")
	}
}

func TestMemoryStateStore_SaveEmptyIDs(t *testing.T) {
	store := NewMemoryStateStore(Config{})
	ctx := context.Background()

	tests := []struct {
		name  string
		state *sdk.AgentState
	}{
		{
			name: "empty agent ID",
			state: &sdk.AgentState{
				AgentID:   "",
				SessionID: "session-1",
			},
		},
		{
			name: "empty session ID",
			state: &sdk.AgentState{
				AgentID:   "agent-1",
				SessionID: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := store.Save(ctx, tt.state)
			if err == nil {
				t.Error("Save() should return error for empty IDs")
			}
		})
	}
}

func TestMemoryStateStore_Load(t *testing.T) {
	store := NewMemoryStateStore(Config{})
	ctx := context.Background()

	originalState := &sdk.AgentState{
		AgentID:   "agent-1",
		SessionID: "session-1",
		Context:   map[string]any{"key": "value"},
	}

	_ = store.Save(ctx, originalState)

	loadedState, err := store.Load(ctx, "agent-1", "session-1")
	if err != nil {
		t.Errorf("Load() error = %v", err)
	}

	if loadedState.AgentID != originalState.AgentID {
		t.Errorf("AgentID = %v, want %v", loadedState.AgentID, originalState.AgentID)
	}

	if loadedState.SessionID != originalState.SessionID {
		t.Errorf("SessionID = %v, want %v", loadedState.SessionID, originalState.SessionID)
	}
}

func TestMemoryStateStore_LoadNotFound(t *testing.T) {
	store := NewMemoryStateStore(Config{})
	ctx := context.Background()

	_, err := store.Load(ctx, "agent-1", "session-1")
	if err == nil {
		t.Error("Load() should return error for non-existent state")
	}
}

func TestMemoryStateStore_Delete(t *testing.T) {
	store := NewMemoryStateStore(Config{})
	ctx := context.Background()

	state := &sdk.AgentState{
		AgentID:   "agent-1",
		SessionID: "session-1",
	}

	_ = store.Save(ctx, state)

	err := store.Delete(ctx, "agent-1", "session-1")
	if err != nil {
		t.Errorf("Delete() error = %v", err)
	}

	if store.Count() != 0 {
		t.Errorf("expected 0 states after delete, got %d", store.Count())
	}

	// Verify state is actually deleted
	_, err = store.Load(ctx, "agent-1", "session-1")
	if err == nil {
		t.Error("Load() should fail after Delete()")
	}
}

func TestMemoryStateStore_List(t *testing.T) {
	store := NewMemoryStateStore(Config{})
	ctx := context.Background()

	// Save multiple sessions for one agent
	sessions := []string{"session-1", "session-2", "session-3"}
	for _, sid := range sessions {
		state := &sdk.AgentState{
			AgentID:   "agent-1",
			SessionID: sid,
		}
		_ = store.Save(ctx, state)
	}

	// Save session for different agent
	_ = store.Save(ctx, &sdk.AgentState{
		AgentID:   "agent-2",
		SessionID: "session-1",
	})

	result, err := store.List(ctx, "agent-1")
	if err != nil {
		t.Errorf("List() error = %v", err)
	}

	if len(result) != 3 {
		t.Errorf("expected 3 sessions, got %d", len(result))
	}

	// Verify all sessions are present
	sessionMap := make(map[string]bool)
	for _, sid := range result {
		sessionMap[sid] = true
	}

	for _, sid := range sessions {
		if !sessionMap[sid] {
			t.Errorf("missing session %s in List() result", sid)
		}
	}
}

func TestMemoryStateStore_ListEmpty(t *testing.T) {
	store := NewMemoryStateStore(Config{})
	ctx := context.Background()

	result, err := store.List(ctx, "non-existent-agent")
	if err != nil {
		t.Errorf("List() error = %v", err)
	}

	if len(result) != 0 {
		t.Errorf("expected empty list, got %d sessions", len(result))
	}
}

func TestMemoryStateStore_Clear(t *testing.T) {
	store := NewMemoryStateStore(Config{})
	ctx := context.Background()

	// Add some states
	for i := 0; i < 5; i++ {
		state := &sdk.AgentState{
			AgentID:   "agent-1",
			SessionID: fmt.Sprintf("session-%d", i),
		}
		_ = store.Save(ctx, state)
	}

	if store.Count() != 5 {
		t.Errorf("expected 5 states before clear, got %d", store.Count())
	}

	store.Clear()

	if store.Count() != 0 {
		t.Errorf("expected 0 states after clear, got %d", store.Count())
	}
}

func TestMemoryStateStore_TTL(t *testing.T) {
	store := NewMemoryStateStore(Config{
		TTL: 100 * time.Millisecond,
	})
	ctx := context.Background()

	state := &sdk.AgentState{
		AgentID:   "agent-1",
		SessionID: "session-1",
	}

	_ = store.Save(ctx, state)

	// Should load immediately
	_, err := store.Load(ctx, "agent-1", "session-1")
	if err != nil {
		t.Errorf("Load() should succeed before TTL, error = %v", err)
	}

	// Wait for TTL to expire
	time.Sleep(150 * time.Millisecond)

	// Should fail to load after TTL
	_, err = store.Load(ctx, "agent-1", "session-1")
	if err == nil {
		t.Error("Load() should fail after TTL")
	}
}

func TestMemoryStateStore_Concurrent(t *testing.T) {
	store := NewMemoryStateStore(Config{})
	ctx := context.Background()

	// Test concurrent writes
	const goroutines = 10
	done := make(chan bool, goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			state := &sdk.AgentState{
				AgentID:   "agent-1",
				SessionID: fmt.Sprintf("session-%d", id),
			}
			_ = store.Save(ctx, state)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < goroutines; i++ {
		<-done
	}

	if store.Count() != goroutines {
		t.Errorf("expected %d states, got %d", goroutines, store.Count())
	}

	// Test concurrent reads
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			_, _ = store.Load(ctx, "agent-1", fmt.Sprintf("session-%d", id))
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < goroutines; i++ {
		<-done
	}
}

