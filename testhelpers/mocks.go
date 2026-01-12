package testhelpers

import (
	"context"

	"github.com/xraph/ai-sdk/llm"
	logger "github.com/xraph/go-utils/log"
	"github.com/xraph/go-utils/metrics"
)

// MockLLMManager for testing.
type MockLLMManager struct {
	ChatFunc       func(ctx context.Context, request llm.ChatRequest) (llm.ChatResponse, error)
	ChatStreamFunc func(ctx context.Context, request llm.ChatRequest, handler func(llm.ChatStreamEvent) error) error
}

func (m *MockLLMManager) Chat(ctx context.Context, request llm.ChatRequest) (llm.ChatResponse, error) {
	if m.ChatFunc != nil {
		return m.ChatFunc(ctx, request)
	}

	return llm.ChatResponse{}, nil
}

func (m *MockLLMManager) ChatStream(ctx context.Context, request llm.ChatRequest, handler func(llm.ChatStreamEvent) error) error {
	if m.ChatStreamFunc != nil {
		return m.ChatStreamFunc(ctx, request, handler)
	}

	return nil
}

func (m *MockLLMManager) SupportsStreaming(provider string) bool {
	return true
}

// NewMockLLM returns a new mock LLM manager for testing.
func NewMockLLM() *MockLLMManager {
	return &MockLLMManager{}
}

// NewMockLogger returns a new mock logger for testing.
func NewMockLogger() logger.Logger {
	return logger.NewTestLogger()
}

// NewMockMetricsInstance returns a mock metrics instance for testing.
func NewMockMetricsInstance() metrics.Metrics {
	return metrics.NewMockMetrics()
}

// NewMockMetrics returns a mock metrics instance for testing.
// Alias for NewMockMetricsInstance for convenience.
func NewMockMetrics() metrics.Metrics {
	return metrics.NewMockMetrics()
}
