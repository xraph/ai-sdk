package plugins

import (
	"context"
	"testing"

	sdk "github.com/xraph/ai-sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockVectorStore for testing
type MockVectorStore struct {
	upsertCalled bool
	queryCalled  bool
	deleteCalled bool
}

func (m *MockVectorStore) Upsert(ctx context.Context, vectors []sdk.Vector) error {
	m.upsertCalled = true
	return nil
}

func (m *MockVectorStore) Query(ctx context.Context, vector []float64, limit int, filter map[string]any) ([]sdk.VectorMatch, error) {
	m.queryCalled = true
	return []sdk.VectorMatch{
		{ID: "test1", Score: 0.9},
		{ID: "test2", Score: 0.8},
	}, nil
}

func (m *MockVectorStore) Delete(ctx context.Context, ids []string) error {
	m.deleteCalled = true
	return nil
}

func TestVectorStorePlugin_BasicOperations(t *testing.T) {
	ctx := context.Background()
	mockStore := &MockVectorStore{}

	plugin := NewVectorStorePlugin(VectorStorePluginConfig{
		Name:    "test-vector-store",
		Version: "1.0.0",
		Store:   mockStore,
	})

	// Test initialization
	err := plugin.Initialize(ctx)
	require.NoError(t, err)

	// Test metadata
	assert.Equal(t, "test-vector-store", plugin.Name())
	assert.Equal(t, "1.0.0", plugin.Version())

	// Test upsert
	_, err = plugin.Execute(ctx, &UpsertRequest{
		Vectors: []sdk.Vector{
			{ID: "vec1", Values: []float64{1.0, 2.0}},
		},
	})
	require.NoError(t, err)
	assert.True(t, mockStore.upsertCalled)

	// Test query
	result, err := plugin.Execute(ctx, &QueryRequest{
		Vector: []float64{1.0, 2.0},
		Limit:  10,
	})
	require.NoError(t, err)
	matches, ok := result.([]sdk.VectorMatch)
	require.True(t, ok)
	assert.Len(t, matches, 2)
	assert.True(t, mockStore.queryCalled)

	// Test delete
	_, err = plugin.Execute(ctx, &DeleteRequest{
		IDs: []string{"vec1"},
	})
	require.NoError(t, err)
	assert.True(t, mockStore.deleteCalled)

	// Test shutdown
	err = plugin.Shutdown(ctx)
	require.NoError(t, err)
}

func TestVectorStorePlugin_GetVectorStore(t *testing.T) {
	mockStore := &MockVectorStore{}
	plugin := NewVectorStorePlugin(VectorStorePluginConfig{
		Name:    "test",
		Version: "1.0.0",
		Store:   mockStore,
	})

	store := plugin.GetVectorStore()
	assert.Equal(t, mockStore, store)
}

func TestVectorStorePlugin_NilStore(t *testing.T) {
	ctx := context.Background()
	plugin := NewVectorStorePlugin(VectorStorePluginConfig{
		Name:    "test",
		Version: "1.0.0",
		Store:   nil,
	})

	err := plugin.Initialize(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "vector store cannot be nil")
}

func TestVectorStorePlugin_UnsupportedOperation(t *testing.T) {
	ctx := context.Background()
	mockStore := &MockVectorStore{}
	plugin := NewVectorStorePlugin(VectorStorePluginConfig{
		Name:    "test",
		Version: "1.0.0",
		Store:   mockStore,
	})

	err := plugin.Initialize(ctx)
	require.NoError(t, err)

	// Try to execute with unsupported input type
	_, err = plugin.Execute(ctx, "invalid input")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported operation type")
}

func TestVectorStorePlugin_ImplementsInterface(t *testing.T) {
	var _ sdk.Plugin = (*VectorStorePlugin)(nil)
}

