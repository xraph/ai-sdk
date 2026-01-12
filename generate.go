package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/xraph/ai-sdk/internal/messages"
	"github.com/xraph/ai-sdk/internal/prompt"
	"github.com/xraph/ai-sdk/llm"
	logger "github.com/xraph/go-utils/log"
	"github.com/xraph/go-utils/metrics"
)

// TextGenerator provides a fluent interface for text generation.
// Renamed from GenerateBuilder for clarity.
//
// Example:
//
//	result, err := sdk.NewTextGenerator(ctx, llm, logger, metrics).
//	    WithPrompt("Write a haiku about {{.topic}}").
//	    WithVar("topic", "programming").
//	    Execute()
type TextGenerator struct {
	ctx      context.Context
	provider string
	model    string
	prompt   string
	vars     map[string]any

	// Generation parameters
	temperature *float64
	maxTokens   *int
	topP        *float64
	topK        *int
	stop        []string

	// Advanced options
	tools      []llm.Tool
	toolChoice string

	// System configuration
	systemPrompt string
	messages     []llm.ChatMessage

	// Execution options
	timeout  time.Duration
	cache    bool
	cacheTTL time.Duration

	// Callbacks
	onStart    func()
	onComplete func(Result)
	onError    func(error)

	// Internal
	llmManager LLMManager
	logger     logger.Logger
	metrics    metrics.Metrics
}

// GenerateBuilder is an alias for TextGenerator for backward compatibility.
// Deprecated: Use TextGenerator instead.
type GenerateBuilder = TextGenerator

// LLMManager interface for LLM operations.
type LLMManager interface {
	Chat(ctx context.Context, request llm.ChatRequest) (llm.ChatResponse, error)
	ChatStream(ctx context.Context, request llm.ChatRequest, handler func(llm.ChatStreamEvent) error) error
	SupportsStreaming(provider string) bool
}

// NewTextGenerator creates a new text generator.
func NewTextGenerator(ctx context.Context, llmManager LLMManager, logger logger.Logger, metrics metrics.Metrics) *TextGenerator {
	return &TextGenerator{
		ctx:        ctx,
		vars:       make(map[string]any),
		llmManager: llmManager,
		logger:     logger,
		metrics:    metrics,
		timeout:    30 * time.Second,
		cache:      false,
	}
}

// NewGenerateBuilder is an alias for NewTextGenerator for backward compatibility.
// Deprecated: Use NewTextGenerator instead.
var NewGenerateBuilder = NewTextGenerator

// WithProvider sets the LLM provider.
func (b *TextGenerator) WithProvider(provider string) *TextGenerator {
	b.provider = provider

	return b
}

// WithModel sets the model to use.
func (b *TextGenerator) WithModel(model string) *TextGenerator {
	b.model = model

	return b
}

// WithPrompt sets the prompt template.
func (b *TextGenerator) WithPrompt(prompt string) *TextGenerator {
	b.prompt = prompt

	return b
}

// WithVars sets template variables.
func (b *TextGenerator) WithVars(vars map[string]any) *TextGenerator {
	b.vars = vars

	return b
}

// WithVar sets a single template variable.
func (b *TextGenerator) WithVar(key string, value any) *TextGenerator {
	b.vars[key] = value

	return b
}

// WithSystemPrompt sets the system prompt.
func (b *TextGenerator) WithSystemPrompt(prompt string) *TextGenerator {
	b.systemPrompt = prompt

	return b
}

// WithTemperature sets the temperature (0.0-2.0).
func (b *TextGenerator) WithTemperature(temp float64) *TextGenerator {
	b.temperature = &temp

	return b
}

// WithMaxTokens sets the maximum tokens to generate.
func (b *TextGenerator) WithMaxTokens(tokens int) *TextGenerator {
	b.maxTokens = &tokens

	return b
}

// WithTopP sets the top-p sampling parameter.
func (b *TextGenerator) WithTopP(topP float64) *TextGenerator {
	b.topP = &topP

	return b
}

// WithTopK sets the top-k sampling parameter.
func (b *TextGenerator) WithTopK(topK int) *TextGenerator {
	b.topK = &topK

	return b
}

// WithStop sets stop sequences.
func (b *TextGenerator) WithStop(stop ...string) *TextGenerator {
	b.stop = stop

	return b
}

// WithTools adds tools for function calling.
func (b *TextGenerator) WithTools(tools ...llm.Tool) *TextGenerator {
	b.tools = append(b.tools, tools...)

	return b
}

// WithToolChoice sets the tool choice strategy.
func (b *TextGenerator) WithToolChoice(choice string) *TextGenerator {
	b.toolChoice = choice

	return b
}

// WithTimeout sets the request timeout.
func (b *TextGenerator) WithTimeout(timeout time.Duration) *TextGenerator {
	b.timeout = timeout

	return b
}

// WithCache enables caching with TTL.
func (b *TextGenerator) WithCache(ttl time.Duration) *TextGenerator {
	b.cache = true
	b.cacheTTL = ttl

	return b
}

// WithMessages sets the full conversation history.
func (b *TextGenerator) WithMessages(messages []llm.ChatMessage) *TextGenerator {
	b.messages = messages

	return b
}

// OnStart sets a callback for when generation starts.
func (b *TextGenerator) OnStart(fn func()) *TextGenerator {
	b.onStart = fn

	return b
}

// OnComplete sets a callback for when generation completes.
func (b *TextGenerator) OnComplete(fn func(Result)) *TextGenerator {
	b.onComplete = fn

	return b
}

// OnError sets a callback for when an error occurs.
func (b *TextGenerator) OnError(fn func(error)) *TextGenerator {
	b.onError = fn

	return b
}

// Execute performs the generation.
func (b *TextGenerator) Execute() (*Result, error) {
	startTime := time.Now()

	// Trigger start callback
	if b.onStart != nil {
		b.onStart()
	}

	// Apply timeout
	ctx := b.ctx
	if b.timeout > 0 {
		var cancel context.CancelFunc

		ctx, cancel = context.WithTimeout(ctx, b.timeout)
		defer cancel()
	}

	// Render prompt template
	renderedPrompt, err := b.renderPrompt()
	if err != nil {
		if b.onError != nil {
			b.onError(err)
		}

		return nil, fmt.Errorf("failed to render prompt: %w", err)
	}

	// Build messages
	messages := b.buildMessages(renderedPrompt)

	// Create request
	request := llm.ChatRequest{
		Provider:    b.provider,
		Model:       b.model,
		Messages:    messages,
		Temperature: b.temperature,
		MaxTokens:   b.maxTokens,
		TopP:        b.topP,
		TopK:        b.topK,
		Stop:        b.stop,
		Tools:       b.tools,
		ToolChoice:  b.toolChoice,
	}

	// Log request
	if b.logger != nil {
		b.logger.Debug("executing generation",
			logger.String("provider", b.provider),
			logger.String("model", b.model),
			logger.Int("prompt_length", len(renderedPrompt)),
		)
	}

	// Execute request
	response, err := b.llmManager.Chat(ctx, request)
	if err != nil {
		if b.onError != nil {
			b.onError(err)
		}

		if b.metrics != nil {
			b.metrics.Counter("forge.ai.sdk.generate.errors",
				metrics.WithLabel("provider", b.provider),
				metrics.WithLabel("model", b.model),
			).Inc()
		}

		return nil, fmt.Errorf("failed to generate: %w", err)
	}

	// Build result
	result := &Result{
		Metadata:     make(map[string]any),
		FinishReason: "unknown",
	}

	if len(response.Choices) > 0 {
		choice := response.Choices[0]
		result.Content = choice.Message.Content
		result.FinishReason = choice.FinishReason

		// Extract tool calls if present
		if len(choice.Message.ToolCalls) > 0 {
			result.ToolCalls = make([]ToolCallResult, 0, len(choice.Message.ToolCalls))
			for _, tc := range choice.Message.ToolCalls {
				// Parse arguments from JSON string to map
				var args map[string]any
				if tc.Function != nil && tc.Function.Arguments != "" {
					if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
						// If parsing fails, store as string
						args = map[string]any{"raw": tc.Function.Arguments}
					}
				} else {
					args = make(map[string]any)
				}

				result.ToolCalls = append(result.ToolCalls, ToolCallResult{
					Name:      tc.Function.Name,
					Arguments: args,
				})
			}
		}
	}

	if response.Usage != nil {
		result.Usage = &Usage{
			Provider:     b.provider,
			Model:        b.model,
			InputTokens:  int(response.Usage.InputTokens),
			OutputTokens: int(response.Usage.OutputTokens),
			Cost:         response.Usage.Cost,
			Timestamp:    time.Now(),
		}
	}

	result.Metadata["duration"] = time.Since(startTime)
	result.Metadata["response_id"] = response.ID

	// Trigger complete callback
	if b.onComplete != nil {
		b.onComplete(*result)
	}

	// Record metrics
	if b.metrics != nil {
		b.metrics.Counter("forge.ai.sdk.generate.success",
			metrics.WithLabel("provider", b.provider),
			metrics.WithLabel("model", b.model),
		).Inc()

		b.metrics.Histogram("forge.ai.sdk.generate.duration",
			metrics.WithLabel("provider", b.provider),
			metrics.WithLabel("model", b.model),
		).Observe(time.Since(startTime).Seconds())

		if result.Usage != nil {
			b.metrics.Counter("forge.ai.sdk.generate.tokens",
				metrics.WithLabel("provider", b.provider),
				metrics.WithLabel("model", b.model),
				metrics.WithLabel("type", "input"),
			).Add(float64(result.Usage.InputTokens))

			b.metrics.Counter("forge.ai.sdk.generate.tokens",
				metrics.WithLabel("provider", b.provider),
				metrics.WithLabel("model", b.model),
				metrics.WithLabel("type", "output"),
			).Add(float64(result.Usage.OutputTokens))
		}
	}

	return result, nil
}

// renderPrompt renders the prompt template with variables.
func (b *TextGenerator) renderPrompt() (string, error) {
	return prompt.Render(b.prompt, b.vars)
}

// buildMessages builds the chat messages.
func (b *TextGenerator) buildMessages(userPrompt string) []llm.ChatMessage {
	return messages.Build(b.systemPrompt, b.messages, userPrompt)
}

// String returns the generated content (convenience method).
func (r *Result) String() string {
	return r.Content
}

// JSON marshals the result to JSON.
func (r *Result) JSON() ([]byte, error) {
	return json.Marshal(r)
}
