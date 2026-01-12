# Integration Plugins

Plugin wrappers for dynamic loading of integrations.

## 🎯 Overview

The plugins package provides wrappers that allow integrations to be loaded dynamically using the SDK's plugin system.

## 🚀 Usage

### As Direct Dependency

Most applications should use integrations directly:

```go
import (
    "github.com/xraph/ai-sdk/integrations/vectorstores/pinecone"
)

store, _ := pinecone.NewPineconeVectorStore(ctx, config)
```

### As Dynamic Plugin

For advanced use cases requiring runtime plugin loading:

```go
import (
    sdk "github.com/xraph/ai-sdk"
    "github.com/xraph/ai-sdk/integrations/plugins"
    "github.com/xraph/ai-sdk/integrations/vectorstores/pinecone"
)

// Create plugin system
pluginSystem := sdk.NewPluginSystem(logger, metrics)

// Create and wrap vector store
store, _ := pinecone.NewPineconeVectorStore(ctx, config)
plugin := plugins.NewVectorStorePlugin(plugins.VectorStorePluginConfig{
    Name:    "pinecone-vector-store",
    Version: "1.0.0",
    Store:   store,
})

// Register plugin
pluginSystem.RegisterPlugin(ctx, plugin)

// Execute operations via plugin
result, _ := pluginSystem.ExecutePlugin(ctx, "pinecone-vector-store", &plugins.QueryRequest{
    Vector: queryVector,
    Limit:  10,
})
```

## 🔧 Plugin Operations

### VectorStorePlugin

Supports three operations via Execute():

#### Upsert
```go
pluginSystem.ExecutePlugin(ctx, "vector-store", &plugins.UpsertRequest{
    Vectors: []sdk.Vector{...},
})
```

#### Query
```go
result, _ := pluginSystem.ExecutePlugin(ctx, "vector-store", &plugins.QueryRequest{
    Vector: queryVector,
    Limit:  10,
    Filter: map[string]any{"category": "docs"},
})
matches := result.([]sdk.VectorMatch)
```

#### Delete
```go
pluginSystem.ExecutePlugin(ctx, "vector-store", &plugins.DeleteRequest{
    IDs: []string{"id1", "id2"},
})
```

## 🏗️ Creating Custom Plugins

### Vector Store Plugin

```go
// 1. Create your vector store implementation
type MyVectorStore struct {
    // ...
}

func (m *MyVectorStore) Upsert(ctx context.Context, vectors []sdk.Vector) error {
    // Implementation
}

func (m *MyVectorStore) Query(ctx context.Context, vector []float64, limit int, filter map[string]any) ([]sdk.VectorMatch, error) {
    // Implementation
}

func (m *MyVectorStore) Delete(ctx context.Context, ids []string) error {
    // Implementation
}

// 2. Wrap as plugin
myStore := &MyVectorStore{}
plugin := plugins.NewVectorStorePlugin(plugins.VectorStorePluginConfig{
    Name:    "my-vector-store",
    Version: "1.0.0",
    Store:   myStore,
})

// 3. Register
pluginSystem.RegisterPlugin(ctx, plugin)
```

## 📦 Plugin System Benefits

1. **Runtime Loading**: Load plugins at runtime without recompilation
2. **Isolation**: Plugins run in isolated contexts
3. **Versioning**: Track plugin versions
4. **Hot Reloading**: Unload and reload plugins (with care)
5. **Extension Points**: Easy to extend with custom plugins

## ⚠️ When NOT to Use Plugins

- **Simple applications**: Direct imports are simpler
- **Type safety**: Direct use provides better type safety
- **Performance**: Plugins add small overhead
- **Debugging**: Harder to debug dynamically loaded code

## 🔒 Security Considerations

- Only load plugins from trusted sources
- Validate plugin signatures (if applicable)
- Run plugins with minimal privileges
- Monitor plugin resource usage

## 📚 Further Reading

- [Plugin System Documentation](../../docs/plugins.md)
- [SDK Architecture](../../docs/architecture.md)
- [Integration Guide](../README.md)

## 📝 License

MIT License - see [LICENSE](../../LICENSE) for details.

