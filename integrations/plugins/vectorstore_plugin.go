package plugins

import (
	"context"
	"fmt"

	sdk "github.com/xraph/ai-sdk"
)

// VectorStorePlugin wraps a VectorStore as a Plugin for dynamic loading.
type VectorStorePlugin struct {
	name    string
	version string
	store   sdk.VectorStore
}

// VectorStorePluginConfig configures a VectorStorePlugin.
type VectorStorePluginConfig struct {
	Name    string
	Version string
	Store   sdk.VectorStore
}

// NewVectorStorePlugin creates a new VectorStorePlugin.
func NewVectorStorePlugin(cfg VectorStorePluginConfig) *VectorStorePlugin {
	return &VectorStorePlugin{
		name:    cfg.Name,
		version: cfg.Version,
		store:   cfg.Store,
	}
}

// Name returns the plugin name.
func (v *VectorStorePlugin) Name() string {
	return v.name
}

// Version returns the plugin version.
func (v *VectorStorePlugin) Version() string {
	return v.version
}

// Initialize initializes the plugin.
func (v *VectorStorePlugin) Initialize(ctx context.Context) error {
	if v.store == nil {
		return fmt.Errorf("vector store cannot be nil")
	}
	return nil
}

// Execute executes vector store operations.
func (v *VectorStorePlugin) Execute(ctx context.Context, input any) (any, error) {
	if v.store == nil {
		return nil, fmt.Errorf("vector store not initialized")
	}

	// Type-switch on input to determine operation
	switch req := input.(type) {
	case *UpsertRequest:
		return nil, v.store.Upsert(ctx, req.Vectors)

	case *QueryRequest:
		return v.store.Query(ctx, req.Vector, req.Limit, req.Filter)

	case *DeleteRequest:
		return nil, v.store.Delete(ctx, req.IDs)

	default:
		return nil, fmt.Errorf("unsupported operation type: %T", input)
	}
}

// Shutdown shuts down the plugin.
func (v *VectorStorePlugin) Shutdown(ctx context.Context) error {
	// VectorStore interface doesn't have Close method by default
	// Implementations that need cleanup should implement Closer
	if closer, ok := v.store.(interface{ Close() error }); ok {
		return closer.Close()
	}
	return nil
}

// GetVectorStore returns the underlying VectorStore.
func (v *VectorStorePlugin) GetVectorStore() sdk.VectorStore {
	return v.store
}

// Request types for Execute operations

// UpsertRequest represents an upsert operation.
type UpsertRequest struct {
	Vectors []sdk.Vector
}

// QueryRequest represents a query operation.
type QueryRequest struct {
	Vector []float64
	Limit  int
	Filter map[string]any
}

// DeleteRequest represents a delete operation.
type DeleteRequest struct {
	IDs []string
}
