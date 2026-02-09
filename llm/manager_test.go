package llm

import (
	"context"
	"testing"

	logger "github.com/xraph/go-utils/log"
	"github.com/xraph/go-utils/metrics"
)

// mockStreamingProvider is a mock provider that implements StreamingProvider interface
type mockStreamingProvider struct {
	name string
}

func (m *mockStreamingProvider) Name() string { return m.name }
func (m *mockStreamingProvider) Models() []string { return []string{"mock-model"} }
func (m *mockStreamingProvider) Chat(ctx context.Context, request ChatRequest) (ChatResponse, error) {
	return ChatResponse{}, nil
}
func (m *mockStreamingProvider) Complete(ctx context.Context, request CompletionRequest) (CompletionResponse, error) {
	return CompletionResponse{}, nil
}
func (m *mockStreamingProvider) Embed(ctx context.Context, request EmbeddingRequest) (EmbeddingResponse, error) {
	return EmbeddingResponse{}, nil
}
func (m *mockStreamingProvider) GetUsage() LLMUsage { return LLMUsage{} }
func (m *mockStreamingProvider) HealthCheck(ctx context.Context) error { return nil }
func (m *mockStreamingProvider) ChatStream(ctx context.Context, request ChatRequest, handler func(ChatStreamEvent) error) error {
	return nil
}

// mockBasicProvider is a mock provider that only implements LLMProvider interface (no streaming)
type mockBasicProvider struct {
	name string
}

func (m *mockBasicProvider) Name() string { return m.name }
func (m *mockBasicProvider) Models() []string { return []string{"mock-model"} }
func (m *mockBasicProvider) Chat(ctx context.Context, request ChatRequest) (ChatResponse, error) {
	return ChatResponse{}, nil
}
func (m *mockBasicProvider) Complete(ctx context.Context, request CompletionRequest) (CompletionResponse, error) {
	return CompletionResponse{}, nil
}
func (m *mockBasicProvider) Embed(ctx context.Context, request EmbeddingRequest) (EmbeddingResponse, error) {
	return EmbeddingResponse{}, nil
}
func (m *mockBasicProvider) GetUsage() LLMUsage       { return LLMUsage{} }
func (m *mockBasicProvider) HealthCheck(ctx context.Context) error { return nil }

func TestLLMManager_SupportsStreaming(t *testing.T) {
	tests := []struct {
		name             string
		providerName     string
		provider         LLMProvider
		expectedSupports bool
	}{
		{
			name:             "streaming provider",
			providerName:     "streaming-provider",
			provider:         &mockStreamingProvider{name: "streaming-provider"},
			expectedSupports: true,
		},
		{
			name:             "non-streaming provider",
			providerName:     "basic-provider",
			provider:         &mockBasicProvider{name: "basic-provider"},
			expectedSupports: false,
		},
		{
			name:             "non-existent provider",
			providerName:     "non-existent",
			provider:         nil,
			expectedSupports: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create manager
			manager, err := NewLLMManager(LLMManagerConfig{
				Logger:  logger.NewTestLogger(),
				Metrics: metrics.NewMockMetrics(),
			})
			if err != nil {
				t.Fatalf("Failed to create LLM manager: %v", err)
			}

			// Register provider if it exists
			if tt.provider != nil {
				if err := manager.RegisterProvider(tt.provider); err != nil {
					t.Fatalf("Failed to register provider: %v", err)
				}
			}

			// Test SupportsStreaming
			supports := manager.SupportsStreaming(tt.providerName)
			if supports != tt.expectedSupports {
				t.Errorf("SupportsStreaming(%q) = %v, want %v", tt.providerName, supports, tt.expectedSupports)
			}
		})
	}
}

func TestLLMManager_SupportsStreaming_ThreadSafety(t *testing.T) {
	manager, err := NewLLMManager(LLMManagerConfig{
		Logger:  logger.NewTestLogger(),
		Metrics: metrics.NewMockMetrics(),
	})
	if err != nil {
		t.Fatalf("Failed to create LLM manager: %v", err)
	}

	providerName := "streaming-test"

	// Register a streaming provider
	if err := manager.RegisterProvider(&mockStreamingProvider{name: providerName}); err != nil {
		t.Fatalf("Failed to register provider: %v", err)
	}

	// Test concurrent access
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				manager.SupportsStreaming(providerName)
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}
