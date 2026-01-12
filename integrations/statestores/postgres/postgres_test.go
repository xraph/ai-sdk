package postgres

import (
	"context"
	"testing"

	sdk "github.com/xraph/ai-sdk"
)

func TestNewPostgresStateStore(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "missing connection string",
			cfg: Config{
				TableName: "test_states",
			},
			wantErr: true,
		},
		{
			name: "valid config with defaults",
			cfg: Config{
				ConnString: "postgres://user:pass@localhost:5432/testdb",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip actual creation for tests without valid PostgreSQL instance
			if tt.name == "valid config with defaults" {
				t.Skip("requires running PostgreSQL instance")
			}

			_, err := NewPostgresStateStore(context.Background(), tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewPostgresStateStore() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPostgresStateStore_Save(t *testing.T) {
	t.Skip("requires running PostgreSQL instance - run integration tests instead")

	ctx := context.Background()
	store, err := NewPostgresStateStore(ctx, Config{
		ConnString: "postgres://postgres:postgres@localhost:5432/testdb",
		TableName:  "test_states",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	state := &sdk.AgentState{
		AgentID:   "agent-1",
		SessionID: "session-1",
		Context:   map[string]any{"key": "value"},
	}

	err = store.Save(ctx, state)
	if err != nil {
		t.Errorf("Save() error = %v", err)
	}
}

func TestPostgresStateStore_SaveInvalid(t *testing.T) {
	t.Skip("requires running PostgreSQL instance - run integration tests instead")

	ctx := context.Background()
	store, err := NewPostgresStateStore(ctx, Config{
		ConnString: "postgres://postgres:postgres@localhost:5432/testdb",
		TableName:  "test_states",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	tests := []struct {
		name    string
		state   *sdk.AgentState
		wantErr bool
	}{
		{
			name:    "nil state",
			state:   nil,
			wantErr: true,
		},
		{
			name: "empty agent ID",
			state: &sdk.AgentState{
				AgentID:   "",
				SessionID: "session-1",
			},
			wantErr: true,
		},
		{
			name: "empty session ID",
			state: &sdk.AgentState{
				AgentID:   "agent-1",
				SessionID: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := store.Save(ctx, tt.state)
			if (err != nil) != tt.wantErr {
				t.Errorf("Save() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPostgresStateStore_Load(t *testing.T) {
	t.Skip("requires running PostgreSQL instance - run integration tests instead")

	ctx := context.Background()
	store, _ := NewPostgresStateStore(ctx, Config{
		ConnString: "postgres://postgres:postgres@localhost:5432/testdb",
		TableName:  "test_states",
	})
	defer store.Close()

	// Save first
	originalState := &sdk.AgentState{
		AgentID:   "agent-1",
		SessionID: "session-1",
		Context:   map[string]any{"key": "value"},
	}
	_ = store.Save(ctx, originalState)

	// Load
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

func TestPostgresStateStore_LoadNotFound(t *testing.T) {
	t.Skip("requires running PostgreSQL instance - run integration tests instead")

	ctx := context.Background()
	store, _ := NewPostgresStateStore(ctx, Config{
		ConnString: "postgres://postgres:postgres@localhost:5432/testdb",
		TableName:  "test_states",
	})
	defer store.Close()

	_, err := store.Load(ctx, "non-existent-agent", "non-existent-session")
	if err == nil {
		t.Error("Load() should return error for non-existent state")
	}
}

func TestPostgresStateStore_Delete(t *testing.T) {
	t.Skip("requires running PostgreSQL instance - run integration tests instead")

	ctx := context.Background()
	store, _ := NewPostgresStateStore(ctx, Config{
		ConnString: "postgres://postgres:postgres@localhost:5432/testdb",
		TableName:  "test_states",
	})
	defer store.Close()

	// Save first
	state := &sdk.AgentState{
		AgentID:   "agent-1",
		SessionID: "session-1",
	}
	_ = store.Save(ctx, state)

	// Delete
	err := store.Delete(ctx, "agent-1", "session-1")
	if err != nil {
		t.Errorf("Delete() error = %v", err)
	}

	// Verify deleted
	_, err = store.Load(ctx, "agent-1", "session-1")
	if err == nil {
		t.Error("Load() should fail after Delete()")
	}
}

func TestPostgresStateStore_List(t *testing.T) {
	t.Skip("requires running PostgreSQL instance - run integration tests instead")

	ctx := context.Background()
	store, _ := NewPostgresStateStore(ctx, Config{
		ConnString: "postgres://postgres:postgres@localhost:5432/testdb",
		TableName:  "test_states",
	})
	defer store.Close()

	// Save multiple sessions
	sessions := []string{"session-1", "session-2", "session-3"}
	for _, sid := range sessions {
		state := &sdk.AgentState{
			AgentID:   "agent-1",
			SessionID: sid,
		}
		_ = store.Save(ctx, state)
	}

	// List
	result, err := store.List(ctx, "agent-1")
	if err != nil {
		t.Errorf("List() error = %v", err)
	}

	if len(result) != 3 {
		t.Errorf("expected 3 sessions, got %d", len(result))
	}
}

func TestPostgresStateStore_HealthCheck(t *testing.T) {
	t.Skip("requires running PostgreSQL instance - run integration tests instead")

	ctx := context.Background()
	store, _ := NewPostgresStateStore(ctx, Config{
		ConnString: "postgres://postgres:postgres@localhost:5432/testdb",
		TableName:  "test_states",
	})
	defer store.Close()

	err := store.HealthCheck(ctx)
	if err != nil {
		t.Errorf("HealthCheck() error = %v", err)
	}
}

func TestPostgresStateStore_Count(t *testing.T) {
	t.Skip("requires running PostgreSQL instance - run integration tests instead")

	ctx := context.Background()
	store, _ := NewPostgresStateStore(ctx, Config{
		ConnString: "postgres://postgres:postgres@localhost:5432/testdb",
		TableName:  "test_states",
	})
	defer store.Close()

	count, err := store.Count(ctx)
	if err != nil {
		t.Errorf("Count() error = %v", err)
	}

	if count < 0 {
		t.Errorf("Count() returned negative value: %d", count)
	}
}
