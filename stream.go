package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/xraph/ai-sdk/internal/messages"
	"github.com/xraph/ai-sdk/internal/prompt"
	"github.com/xraph/ai-sdk/llm"
	logger "github.com/xraph/go-utils/log"
	"github.com/xraph/go-utils/metrics"
)

// ThinkingMarker defines a pair of start/end markers for detecting thinking blocks.
type ThinkingMarker struct {
	// Start is the marker that indicates the start of a thinking block
	Start string
	// End is the marker that indicates the end of a thinking block
	End string
}

// ThinkingMarkers is a list of marker pairs to check for thinking blocks.
// Multiple markers can be specified to support different model formats.
type ThinkingMarkers []ThinkingMarker

// Common thinking marker presets for various models.
var (
	// ThinkingMarkersDefault includes the most common thinking markers.
	ThinkingMarkersDefault = ThinkingMarkers{
		{Start: "<thinking>", End: "</thinking>"},
		{Start: "[REASONING]", End: "[/REASONING]"},
		{Start: "<seed:think>", End: "</seed:think>"},
	}

	// ThinkingMarkersSeedThink is for models that use <seed:think> format.
	ThinkingMarkersSeedThink = ThinkingMarkers{
		{Start: "<seed:think>", End: "</seed:think>"},
	}

	// ThinkingMarkersDeepSeek is for DeepSeek models.
	ThinkingMarkersDeepSeek = ThinkingMarkers{
		{Start: "<think>", End: "</think>"},
	}

	// ThinkingMarkersQwen is for Qwen models with reasoning.
	ThinkingMarkersQwen = ThinkingMarkers{
		{Start: "<|thinking|>", End: "<|/thinking|>"},
	}

	// ThinkingMarkersAll combines all known thinking markers.
	ThinkingMarkersAll = ThinkingMarkers{
		{Start: "<thinking>", End: "</thinking>"},
		{Start: "[REASONING]", End: "[/REASONING]"},
		{Start: "<seed:think>", End: "</seed:think>"},
		{Start: "<think>", End: "</think>"},
		{Start: "<|thinking|>", End: "<|/thinking|>"},
		{Start: "<reason>", End: "</reason>"},
		{Start: "<reasoning>", End: "</reasoning>"},
	}
)

// ContainsStart checks if the text contains any of the start markers.
func (tm ThinkingMarkers) ContainsStart(text string) bool {
	for _, marker := range tm {
		if strings.Contains(text, marker.Start) {
			return true
		}
	}

	return false
}

// ContainsEnd checks if the text contains any of the end markers.
func (tm ThinkingMarkers) ContainsEnd(text string) bool {
	for _, marker := range tm {
		if strings.Contains(text, marker.End) {
			return true
		}
	}

	return false
}

// CleanMarkers removes all thinking markers from the text.
func (tm ThinkingMarkers) CleanMarkers(text string) string {
	result := text
	for _, marker := range tm {
		result = strings.ReplaceAll(result, marker.Start, "")
		result = strings.ReplaceAll(result, marker.End, "")
	}

	return strings.TrimSpace(result)
}

// StreamingGenerator provides a fluent API for streaming text generation with
// enhanced features like reasoning steps, tool usage tracking, and token-level callbacks.
// Renamed from StreamBuilder for clarity.
//
// Example:
//
//	stream := sdk.NewStreamingGenerator(ctx, llm, logger, metrics).
//	    WithPrompt("Explain {{.topic}}").
//	    WithVar("topic", "quantum computing").
//	    OnToken(func(token string) { fmt.Print(token) }).
//	    OnReasoning(func(reasoning string) { log.Debug(reasoning) }).
//	    OnComplete(func(result StreamResponse) { log.Info("done") }).
//	    Execute()
type StreamingGenerator struct {
	ctx        context.Context
	llmManager LLMManager
	logger     logger.Logger
	metrics    metrics.Metrics

	// Model configuration
	provider string
	model    string

	// Prompt configuration
	prompt       string
	vars         map[string]any
	systemPrompt string
	messages     []llm.ChatMessage

	// LLM parameters
	temperature *float64
	maxTokens   *int
	topP        *float64
	topK        *int
	stop        []string

	// Tool configuration
	tools      []llm.Tool
	toolChoice string

	// Tool execution configuration
	toolRegistry      *ToolRegistry
	autoExecuteTools  bool          // Auto-execute tools and continue
	maxToolIterations int           // Max tool call iterations (prevent infinite loops)
	toolTimeout       time.Duration // Timeout per tool execution

	// Stream configuration
	includeReasoning bool
	bufferSize       int
	thinkingMarkers  ThinkingMarkers

	// Execution configuration
	timeout time.Duration

	// Legacy callbacks (still supported)
	onStart       func()
	onToken       func(token string)
	onReasoning   func(reasoning string)
	onToolCall    func(toolName string, args map[string]any)
	onComplete    func(StreamResponse)
	onError       func(error)
	onContentPart func(ContentPart)

	// Typed event callbacks (new spec-compliant callbacks)
	onThinkingStart func(executionID string)
	onThinkingDelta func(executionID string, delta string, index int64)
	onThinkingEnd   func(executionID string)
	onContentStart  func(executionID string)
	onContentDelta  func(executionID string, delta string, index int64)
	onContentEnd    func(executionID string)
	onToolUseStart  func(executionID, toolID, toolName string)
	onToolUseDelta  func(executionID, toolID string, delta string, index int64)
	onToolUseEnd    func(executionID, toolID string)
	onStreamEvent   func(event llm.ClientStreamEvent)

	// Tool execution callbacks
	onToolExecutionStart func(toolName string, args map[string]any)
	onToolExecutionEnd   func(toolName string, result *ToolExecutionResult)
	onToolResultStart    func(executionID, toolID, toolName string)
	onToolResultDelta    func(executionID, toolID string, delta string, index int64)
	onToolResultEnd      func(executionID, toolID string)

	// UI Part streaming callbacks
	onUIPartStart func(partID, partType string)
	onUIPartDelta func(partID, section string, data any)
	onUIPartEnd   func(partID string, part ContentPart)

	// UI Tool configuration
	uiToolRegistry      *UIToolRegistry
	enableUIRendering   bool // Auto-render UI tool results
	uiPartStreamManager *UIPartStreamManager

	// Presentation tools configuration
	includePresentationTools bool // Include built-in presentation tools
	enableUIOutputParsing    bool // Parse ui:type blocks from AI output
	uiOutputParser           *UIOutputParser

	// Structured response options
	parseStructured   bool
	responseParser    *ResponseParser
	artifactRegistry  *ArtifactRegistry
	citationManager   *CitationManager
	suggestionManager *SuggestionManager
}

// StreamBuilder is an alias for StreamingGenerator for backward compatibility.
// Deprecated: Use StreamingGenerator instead.
type StreamBuilder = StreamingGenerator

// StreamResponse contains the complete result of a streaming operation.
// (Previously known as StreamResult)
type StreamResponse struct {
	// ExecutionID uniquely identifies this streaming session (for React key stability)
	ExecutionID string

	// Content is the full generated text
	Content string

	// ThinkingContent contains the extended thinking content (if available)
	ThinkingContent string

	// ReasoningSteps contains the thought process (if available) - legacy
	ReasoningSteps []string

	// ToolCalls contains any tools called during generation
	ToolCalls []ToolInvocation

	// Usage contains token usage information
	Usage *Usage

	// Metadata contains additional information
	Metadata map[string]any

	// Duration is the total time taken
	Duration time.Duration

	// StructuredResponse contains parsed content parts for frontend rendering
	StructuredResponse *StructuredResponse

	// Artifacts created during generation
	Artifacts []Artifact

	// Citations referenced in the response
	Citations []Citation

	// Suggestions for follow-up actions
	Suggestions []Suggestion

	// Model and provider used
	Model    string
	Provider string

	// Tool execution results
	ToolExecutions []ToolExecutionResult // All tool executions performed
	Iterations     int                   // Number of tool loop iterations

	// UI Parts rendered during generation
	UIParts          []ContentPart           // All UI parts rendered
	UIToolExecutions []UIToolExecutionResult // UI tool execution results

	// ParsedUIContent is the content with UI blocks removed (if UI parsing was enabled)
	ParsedUIContent string
}

// StreamResult is an alias for StreamResponse for backward compatibility.
// Deprecated: Use StreamResponse instead.
type StreamResult = StreamResponse

// ToolInvocation represents a function/tool call made by the LLM.
// Renamed from ToolCall for clarity (duplicated from llm.ToolInvocation for SDK-specific use).
type ToolInvocation struct {
	Name      string
	Arguments map[string]any
	Result    any
}

// NewStreamingGenerator creates a new streaming generator.
func NewStreamingGenerator(
	ctx context.Context,
	llmManager LLMManager,
	logger logger.Logger,
	metrics metrics.Metrics,
) *StreamingGenerator {
	return &StreamingGenerator{
		ctx:               ctx,
		llmManager:        llmManager,
		logger:            logger,
		metrics:           metrics,
		vars:              make(map[string]any),
		timeout:           60 * time.Second,
		bufferSize:        100,
		thinkingMarkers:   ThinkingMarkersDefault,
		maxToolIterations: 10,
		toolTimeout:       30 * time.Second,
	}
}

// NewStreamBuilder is an alias for NewStreamingGenerator for backward compatibility.
// Deprecated: Use NewStreamingGenerator instead.
var NewStreamBuilder = NewStreamingGenerator

// WithProvider sets the LLM provider.
func (b *StreamingGenerator) WithProvider(provider string) *StreamingGenerator {
	b.provider = provider

	return b
}

// WithModel sets the model to use.
func (b *StreamingGenerator) WithModel(model string) *StreamingGenerator {
	b.model = model

	return b
}

// WithPrompt sets the prompt template.
func (b *StreamingGenerator) WithPrompt(prompt string) *StreamingGenerator {
	b.prompt = prompt

	return b
}

// WithVars sets multiple template variables.
func (b *StreamingGenerator) WithVars(vars map[string]any) *StreamingGenerator {
	maps.Copy(b.vars, vars)

	return b
}

// WithVar sets a single template variable.
func (b *StreamingGenerator) WithVar(key string, value any) *StreamingGenerator {
	b.vars[key] = value

	return b
}

// WithSystemPrompt sets the system prompt.
func (b *StreamingGenerator) WithSystemPrompt(prompt string) *StreamingGenerator {
	b.systemPrompt = prompt

	return b
}

// WithMessages sets conversation history.
func (b *StreamingGenerator) WithMessages(messages []llm.ChatMessage) *StreamingGenerator {
	b.messages = messages

	return b
}

// WithTemperature sets the temperature parameter.
func (b *StreamingGenerator) WithTemperature(temp float64) *StreamingGenerator {
	b.temperature = &temp

	return b
}

// WithMaxTokens sets the maximum tokens to generate.
func (b *StreamingGenerator) WithMaxTokens(tokens int) *StreamingGenerator {
	b.maxTokens = &tokens

	return b
}

// WithTopP sets the top-p sampling parameter.
func (b *StreamingGenerator) WithTopP(topP float64) *StreamingGenerator {
	b.topP = &topP

	return b
}

// WithTopK sets the top-k sampling parameter.
func (b *StreamingGenerator) WithTopK(topK int) *StreamingGenerator {
	b.topK = &topK

	return b
}

// WithStop sets stop sequences.
func (b *StreamingGenerator) WithStop(sequences ...string) *StreamingGenerator {
	b.stop = sequences

	return b
}

// WithTools sets available tools/functions.
func (b *StreamingGenerator) WithTools(tools ...llm.Tool) *StreamingGenerator {
	b.tools = tools

	return b
}

// WithToolChoice sets tool selection strategy.
func (b *StreamingGenerator) WithToolChoice(choice string) *StreamingGenerator {
	b.toolChoice = choice

	return b
}

// WithToolRegistry sets the tool registry for automatic tool execution.
// When a tool registry is set and autoExecuteTools is enabled, tools will be
// automatically executed and their results fed back to the LLM.
func (b *StreamingGenerator) WithToolRegistry(registry *ToolRegistry) *StreamingGenerator {
	b.toolRegistry = registry

	return b
}

// WithAutoExecuteTools enables automatic tool execution and continuation.
// When enabled, the builder will execute tool calls using the tool registry
// and continue the conversation with the results.
func (b *StreamingGenerator) WithAutoExecuteTools(enabled bool) *StreamingGenerator {
	b.autoExecuteTools = enabled

	return b
}

// WithMaxToolIterations sets the maximum number of tool loop iterations.
// This prevents infinite loops when tools keep calling more tools.
// Default is 10.
func (b *StreamingGenerator) WithMaxToolIterations(max int) *StreamingGenerator {
	b.maxToolIterations = max

	return b
}

// WithToolTimeout sets the timeout for individual tool execution.
// Default is 30 seconds.
func (b *StreamingGenerator) WithToolTimeout(timeout time.Duration) *StreamingGenerator {
	b.toolTimeout = timeout

	return b
}

// OnToolExecutionStart registers a callback when tool execution begins.
func (b *StreamingGenerator) OnToolExecutionStart(fn func(toolName string, args map[string]any)) *StreamingGenerator {
	b.onToolExecutionStart = fn

	return b
}

// OnToolExecutionEnd registers a callback when tool execution completes.
func (b *StreamingGenerator) OnToolExecutionEnd(fn func(toolName string, result *ToolExecutionResult)) *StreamingGenerator {
	b.onToolExecutionEnd = fn

	return b
}

// OnToolResultStart registers a callback for when tool result streaming starts.
func (b *StreamingGenerator) OnToolResultStart(fn func(executionID, toolID, toolName string)) *StreamingGenerator {
	b.onToolResultStart = fn

	return b
}

// OnToolResultDelta registers a callback for tool result content deltas.
func (b *StreamingGenerator) OnToolResultDelta(fn func(executionID, toolID string, delta string, index int64)) *StreamingGenerator {
	b.onToolResultDelta = fn

	return b
}

// OnToolResultEnd registers a callback for when tool result streaming ends.
func (b *StreamingGenerator) OnToolResultEnd(fn func(executionID, toolID string)) *StreamingGenerator {
	b.onToolResultEnd = fn

	return b
}

// OnUIPartStart registers a callback for when a UI part starts streaming.
func (b *StreamingGenerator) OnUIPartStart(fn func(partID, partType string)) *StreamingGenerator {
	b.onUIPartStart = fn

	return b
}

// OnUIPartDelta registers a callback for UI part section updates.
func (b *StreamingGenerator) OnUIPartDelta(fn func(partID, section string, data any)) *StreamingGenerator {
	b.onUIPartDelta = fn

	return b
}

// OnUIPartEnd registers a callback for when a UI part finishes streaming.
func (b *StreamingGenerator) OnUIPartEnd(fn func(partID string, part ContentPart)) *StreamingGenerator {
	b.onUIPartEnd = fn

	return b
}

// WithUIToolRegistry sets the UI tool registry for automatic UI rendering.
func (b *StreamingGenerator) WithUIToolRegistry(registry *UIToolRegistry) *StreamingGenerator {
	b.uiToolRegistry = registry

	return b
}

// WithUIToolRendering enables automatic UI rendering for tool results.
// When enabled, tool results from UI tools will be automatically rendered
// as streaming UI parts using the tool's RenderUI method.
func (b *StreamingGenerator) WithUIToolRendering(enabled bool) *StreamingGenerator {
	b.enableUIRendering = enabled

	return b
}

// WithPresentationTools includes built-in presentation tools that the AI can call.
// These include render_table, render_chart, render_metrics, render_timeline, etc.
// When enabled, the AI can format data using these tools for rich UI rendering.
//
// Example:
//
//	stream := sdk.NewStreamBuilder(ctx, llm, logger, metrics).
//	    WithPrompt("Show my sales data as a chart").
//	    WithPresentationTools().
//	    Stream()
func (b *StreamingGenerator) WithPresentationTools() *StreamingGenerator {
	b.includePresentationTools = true

	// Add presentation tools to the LLM tools list
	presentationSchemas := GetPresentationToolSchemas()
	b.tools = append(b.tools, presentationSchemas...)

	return b
}

// WithUIOutputParsing enables parsing of structured UI blocks from AI output.
// When enabled, the parser will detect and convert ui:type blocks in the response
// to ContentParts. Supports both fenced blocks (```ui:table {...}```) and
// inline tags (<ui:chart>{...}</ui:chart>).
//
// Example:
//
//	stream := sdk.NewStreamBuilder(ctx, llm, logger, metrics).
//	    WithPrompt("Show users as a table").
//	    WithUIOutputParsing(true).
//	    Stream()
func (b *StreamingGenerator) WithUIOutputParsing(enabled bool) *StreamingGenerator {
	b.enableUIOutputParsing = enabled

	if enabled && b.uiOutputParser == nil {
		b.uiOutputParser = NewUIOutputParser()
	}

	return b
}

// WithReasoning enables reasoning step extraction.
func (b *StreamingGenerator) WithReasoning(enabled bool) *StreamingGenerator {
	b.includeReasoning = enabled

	return b
}

// WithThinkingMarkers sets custom thinking/reasoning markers.
// Use this to support models with non-standard thinking block formats.
//
// Example:
//
//	builder.WithThinkingMarkers(sdk.ThinkingMarkersSeedThink)
//	// or custom markers:
//	builder.WithThinkingMarkers(sdk.ThinkingMarkers{
//	    {Start: "<my-think>", End: "</my-think>"},
//	})
func (b *StreamingGenerator) WithThinkingMarkers(markers ThinkingMarkers) *StreamingGenerator {
	b.thinkingMarkers = markers

	return b
}

// AddThinkingMarker adds an additional thinking marker pair.
// This allows extending the default markers without replacing them.
func (b *StreamingGenerator) AddThinkingMarker(start, end string) *StreamingGenerator {
	b.thinkingMarkers = append(b.thinkingMarkers, ThinkingMarker{Start: start, End: end})

	return b
}

// WithAllThinkingMarkers configures the builder to recognize all known thinking marker formats.
// This is useful when you want maximum compatibility across different models.
func (b *StreamingGenerator) WithAllThinkingMarkers() *StreamingGenerator {
	b.thinkingMarkers = ThinkingMarkersAll

	return b
}

// WithBufferSize sets the token buffer size.
func (b *StreamingGenerator) WithBufferSize(size int) *StreamingGenerator {
	b.bufferSize = size

	return b
}

// WithTimeout sets the execution timeout.
func (b *StreamingGenerator) WithTimeout(timeout time.Duration) *StreamingGenerator {
	b.timeout = timeout

	return b
}

// OnStart registers a callback to run before streaming starts.
func (b *StreamingGenerator) OnStart(fn func()) *StreamingGenerator {
	b.onStart = fn

	return b
}

// OnToken registers a callback for each generated token.
func (b *StreamingGenerator) OnToken(fn func(token string)) *StreamingGenerator {
	b.onToken = fn

	return b
}

// OnReasoning registers a callback for reasoning steps.
func (b *StreamingGenerator) OnReasoning(fn func(reasoning string)) *StreamingGenerator {
	b.onReasoning = fn

	return b
}

// OnToolCall registers a callback for tool invocations.
func (b *StreamingGenerator) OnToolCall(fn func(toolName string, args map[string]any)) *StreamingGenerator {
	b.onToolCall = fn

	return b
}

// OnComplete registers a callback to run after streaming completes.
func (b *StreamingGenerator) OnComplete(fn func(StreamResponse)) *StreamingGenerator {
	b.onComplete = fn

	return b
}

// OnError registers a callback to run on error.
func (b *StreamingGenerator) OnError(fn func(error)) *StreamingGenerator {
	b.onError = fn

	return b
}

// OnContentPart registers a callback for structured content parts.
// This is called when the response is parsed into structured parts (code blocks, tables, etc.).
func (b *StreamingGenerator) OnContentPart(fn func(ContentPart)) *StreamingGenerator {
	b.onContentPart = fn

	return b
}

// OnThinkingStart registers a callback for when thinking block starts.
func (b *StreamingGenerator) OnThinkingStart(fn func(executionID string)) *StreamingGenerator {
	b.onThinkingStart = fn

	return b
}

// OnThinkingDelta registers a callback for thinking content deltas.
func (b *StreamingGenerator) OnThinkingDelta(fn func(executionID string, delta string, index int64)) *StreamingGenerator {
	b.onThinkingDelta = fn

	return b
}

// OnThinkingEnd registers a callback for when thinking block ends.
func (b *StreamingGenerator) OnThinkingEnd(fn func(executionID string)) *StreamingGenerator {
	b.onThinkingEnd = fn

	return b
}

// OnContentStart registers a callback for when content block starts.
func (b *StreamingGenerator) OnContentStart(fn func(executionID string)) *StreamingGenerator {
	b.onContentStart = fn

	return b
}

// OnContentDelta registers a callback for content deltas.
func (b *StreamingGenerator) OnContentDelta(fn func(executionID string, delta string, index int64)) *StreamingGenerator {
	b.onContentDelta = fn

	return b
}

// OnContentEnd registers a callback for when content block ends.
func (b *StreamingGenerator) OnContentEnd(fn func(executionID string)) *StreamingGenerator {
	b.onContentEnd = fn

	return b
}

// OnToolUseStart registers a callback for when tool use starts.
func (b *StreamingGenerator) OnToolUseStart(fn func(executionID, toolID, toolName string)) *StreamingGenerator {
	b.onToolUseStart = fn

	return b
}

// OnToolUseDelta registers a callback for tool use argument deltas.
func (b *StreamingGenerator) OnToolUseDelta(fn func(executionID, toolID string, delta string, index int64)) *StreamingGenerator {
	b.onToolUseDelta = fn

	return b
}

// OnToolUseEnd registers a callback for when tool use ends.
func (b *StreamingGenerator) OnToolUseEnd(fn func(executionID, toolID string)) *StreamingGenerator {
	b.onToolUseEnd = fn

	return b
}

// OnStreamEvent registers a callback for all typed stream events.
// This provides access to the full ClientStreamEvent for custom handling.
func (b *StreamingGenerator) OnStreamEvent(fn func(event llm.ClientStreamEvent)) *StreamingGenerator {
	b.onStreamEvent = fn

	return b
}

// WithStructuredResponse enables parsing the response into structured content parts.
func (b *StreamingGenerator) WithStructuredResponse(enabled bool) *StreamingGenerator {
	b.parseStructured = enabled
	if enabled && b.responseParser == nil {
		b.responseParser = NewResponseParser()
	}

	return b
}

// WithResponseParser sets a custom response parser.
func (b *StreamingGenerator) WithResponseParser(parser *ResponseParser) *StreamingGenerator {
	b.responseParser = parser
	b.parseStructured = true

	return b
}

// WithArtifactRegistry sets the artifact registry for storing artifacts.
func (b *StreamingGenerator) WithArtifactRegistry(registry *ArtifactRegistry) *StreamingGenerator {
	b.artifactRegistry = registry

	return b
}

// WithCitationManager sets the citation manager for tracking citations.
func (b *StreamingGenerator) WithCitationManager(manager *CitationManager) *StreamingGenerator {
	b.citationManager = manager

	return b
}

// WithSuggestionManager sets the suggestion manager for generating follow-up suggestions.
func (b *StreamingGenerator) WithSuggestionManager(manager *SuggestionManager) *StreamingGenerator {
	b.suggestionManager = manager

	return b
}

// Stream executes the streaming generation.
func (b *StreamingGenerator) Stream() (*StreamResponse, error) {
	startTime := time.Now()

	// Call onStart callback
	if b.onStart != nil {
		b.onStart()
	}

	// Create timeout context
	ctx, cancel := context.WithTimeout(b.ctx, b.timeout)
	defer cancel()

	// Render prompt with variables
	renderedPrompt, err := b.renderPrompt()
	if err != nil {
		if b.onError != nil {
			b.onError(err)
		}

		if b.metrics != nil {
			b.metrics.Counter("forge.ai.sdk.stream.errors", metrics.WithLabel("error", "prompt_render")).Inc()
		}

		return nil, fmt.Errorf("prompt rendering failed: %w", err)
	}

	// Build messages
	messages := b.buildMessages(renderedPrompt)

	// Log execution
	if b.logger != nil {
		b.logger.Debug("Executing streaming generation",
			logger.String("provider", b.provider),
			logger.String("model", b.model),
			logger.Bool("include_reasoning", b.includeReasoning),
		)
	}

	// Build LLM request
	request := llm.ChatRequest{
		Provider: b.provider,
		Model:    b.model,
		Messages: messages,
		Stream:   true,
	}

	if b.temperature != nil {
		request.Temperature = b.temperature
	}

	if b.maxTokens != nil {
		request.MaxTokens = b.maxTokens
	}

	if b.topP != nil {
		request.TopP = b.topP
	}

	if b.topK != nil {
		request.TopK = b.topK
	}

	if len(b.stop) > 0 {
		request.Stop = b.stop
	}

	if len(b.tools) > 0 {
		request.Tools = b.tools
		if b.toolChoice != "" {
			request.ToolChoice = b.toolChoice
		}
	}

	// Create typed stream handler for spec-compliant event processing
	streamHandler := llm.NewClientStreamHandler(llm.ClientStreamHandlerConfig{
		Model:    b.model,
		Provider: b.provider,
		Context:  ctx,
		OnEvent:  b.handleTypedStreamEvent,
	})

	// Initialize UI part stream manager if UI rendering is enabled
	if b.enableUIRendering {
		b.uiPartStreamManager = NewUIPartStreamManager(
			b.handleTypedStreamEvent,
			b.logger,
			b.metrics,
		)
	}

	// Create result accumulator
	result := &StreamResponse{
		ExecutionID:      streamHandler.GetExecutionID(),
		ReasoningSteps:   make([]string, 0),
		UIParts:          make([]ContentPart, 0),
		UIToolExecutions: make([]UIToolExecutionResult, 0),
		ToolCalls:        make([]ToolInvocation, 0),
		ToolExecutions:   make([]ToolExecutionResult, 0),
		Metadata:         make(map[string]any),
		Model:            b.model,
		Provider:         b.provider,
	}

	var (
		fullContent      strings.Builder
		thinkingContent  strings.Builder
		currentReasoning strings.Builder
		currentToolArgs  strings.Builder
	)

	inReasoningBlock := false
	currentToolID := ""
	currentToolName := ""
	iteration := 0

	// Define stream handler that uses the typed event system
	handler := func(event llm.ChatStreamEvent) error {
		if event.Error != "" {
			return fmt.Errorf("stream error: %s", event.Error)
		}

		// Process through typed stream handler for spec-compliant events
		if err := streamHandler.HandleChatStreamEvent(event); err != nil {
			return err
		}

		// Handle block-level events (Anthropic-style with BlockType/BlockState)
		if event.BlockType != "" {
			return b.handleBlockEvent(event, result, &fullContent, &thinkingContent, &currentToolArgs, &currentToolID, &currentToolName)
		}

		// Handle content tokens from choices (fallback for providers without block-level events)
		var token string

		if len(event.Choices) > 0 {
			choice := event.Choices[0]

			// Streaming content is in Delta
			if choice.Delta != nil && choice.Delta.Content != "" {
				token = choice.Delta.Content
			} else if choice.Message.Content != "" {
				// Fallback for non-streaming responses
				token = choice.Message.Content
			}

			if token != "" {
				// Check for reasoning markers using configured thinking markers
				if b.includeReasoning {
					if b.thinkingMarkers.ContainsStart(token) {
						inReasoningBlock = true

						currentReasoning.Reset()

						// Fire thinking start callback
						if b.onThinkingStart != nil {
							b.onThinkingStart(result.ExecutionID)
						}
					}

					if inReasoningBlock {
						currentReasoning.WriteString(token)

						// Fire thinking delta callback
						if b.onThinkingDelta != nil {
							b.onThinkingDelta(result.ExecutionID, token, streamHandler.GetIndex())
						}

						if b.thinkingMarkers.ContainsEnd(token) {
							inReasoningBlock = false
							reasoning := currentReasoning.String()

							// Clean up markers using configured markers
							reasoning = b.thinkingMarkers.CleanMarkers(reasoning)

							if reasoning != "" {
								result.ReasoningSteps = append(result.ReasoningSteps, reasoning)
								thinkingContent.WriteString(reasoning)

								if b.onReasoning != nil {
									b.onReasoning(reasoning)
								}
							}

							// Fire thinking end callback
							if b.onThinkingEnd != nil {
								b.onThinkingEnd(result.ExecutionID)
							}

							currentReasoning.Reset()
						}
					} else {
						fullContent.WriteString(token)

						// Fire content delta callback
						if b.onContentDelta != nil {
							b.onContentDelta(result.ExecutionID, token, streamHandler.GetIndex())
						}

						if b.onToken != nil {
							b.onToken(token)
						}
					}
				} else {
					fullContent.WriteString(token)

					// Fire content delta callback
					if b.onContentDelta != nil {
						b.onContentDelta(result.ExecutionID, token, streamHandler.GetIndex())
					}

					if b.onToken != nil {
						b.onToken(token)
					}
				}
			}
		}

		// Handle tool calls from choices
		if len(event.Choices) > 0 {
			choice := event.Choices[0]

			var toolCalls []llm.ToolCall

			// Streaming tool calls are in Delta
			if choice.Delta != nil && len(choice.Delta.ToolCalls) > 0 {
				toolCalls = choice.Delta.ToolCalls
			} else if len(choice.Message.ToolCalls) > 0 {
				// Fallback for non-streaming responses
				toolCalls = choice.Message.ToolCalls
			}

			for _, tc := range toolCalls {
				// Fire tool use callbacks
				if tc.ID != "" && tc.ID != currentToolID {
					// End previous tool if any
					if currentToolID != "" && b.onToolUseEnd != nil {
						b.onToolUseEnd(result.ExecutionID, currentToolID)
					}

					currentToolID = tc.ID
					if tc.Function != nil {
						currentToolName = tc.Function.Name
					}

					if b.onToolUseStart != nil {
						b.onToolUseStart(result.ExecutionID, tc.ID, currentToolName)
					}
				}

				if tc.Function != nil && tc.Function.Arguments != "" {
					if b.onToolUseDelta != nil {
						b.onToolUseDelta(result.ExecutionID, tc.ID, tc.Function.Arguments, streamHandler.GetIndex())
					}
				}

				toolCall := ToolInvocation{
					Name:      tc.Function.Name,
					Arguments: make(map[string]any),
				}

				if tc.Function.Arguments != "" {
					toolCall.Arguments["raw"] = tc.Function.Arguments
				}

				result.ToolCalls = append(result.ToolCalls, toolCall)

				if b.onToolCall != nil {
					b.onToolCall(toolCall.Name, toolCall.Arguments)
				}
			}
		}

		// Handle usage information
		if event.Usage != nil {
			result.Usage = &Usage{
				InputTokens:  int(event.Usage.InputTokens),
				OutputTokens: int(event.Usage.OutputTokens),
			}
		}

		return nil
	}

	// Agentic tool loop - iterate until no tool calls or max iterations reached
	for iteration < b.maxToolIterations {
		iteration++

		// Track tool calls for this iteration
		iterationToolCalls := make([]ToolInvocation, 0)

		// Use native streaming
		err = b.llmManager.ChatStream(ctx, request, handler)
		if err != nil {
			if b.onError != nil {
				b.onError(err)
			}

			if b.metrics != nil {
				b.metrics.Counter("forge.ai.sdk.stream.errors", metrics.WithLabel("error", "llm_stream")).Inc()
			}

			return nil, fmt.Errorf("LLM streaming request failed: %w", err)
		}

		// Collect tool calls from this iteration (for streaming path)
		if len(iterationToolCalls) == 0 {
			// Check if streaming handler accumulated tool calls
			// Tool calls from streaming are already in result.ToolCalls
			// We need to track which ones are new for this iteration
			for _, tc := range result.ToolCalls {
				// Check if this is from the current iteration by checking if already processed
				if _, processed := tc.Arguments["_processed"]; !processed {
					iterationToolCalls = append(iterationToolCalls, tc)
				}
			}
		}

		// Check if we should continue with tool execution
		if !b.autoExecuteTools || b.toolRegistry == nil || len(iterationToolCalls) == 0 {
			// No auto-execute or no tool calls, exit the loop
			break
		}

		// Log tool execution iteration
		if b.logger != nil {
			b.logger.Debug("Executing tools in agentic loop",
				logger.Int("iteration", iteration),
				logger.Int("tool_count", len(iterationToolCalls)),
			)
		}

		// Execute tool calls
		execResults := b.executeToolCalls(ctx, result.ExecutionID, iterationToolCalls, streamHandler)
		result.ToolExecutions = append(result.ToolExecutions, execResults...)

		// Mark tool calls as processed
		for i := range result.ToolCalls {
			if result.ToolCalls[i].Arguments == nil {
				result.ToolCalls[i].Arguments = make(map[string]any)
			}

			result.ToolCalls[i].Arguments["_processed"] = true
		}

		// Build tool result messages and add them to the request
		toolResultMessages := b.buildToolResultMessages(iterationToolCalls, execResults)
		request.Messages = append(request.Messages, toolResultMessages...)

		// Reset iteration tracking
		currentToolID = ""
		currentToolName = ""
	}

	// Track iterations in result
	result.Iterations = iteration

	// Finalize result
	result.Content = fullContent.String()
	result.ThinkingContent = thinkingContent.String()
	result.Duration = time.Since(startTime)

	// Parse into structured response if enabled
	if b.parseStructured && b.responseParser != nil {
		result.StructuredResponse = b.parseToStructuredResponse(result)
	}

	// Parse UI blocks from content if enabled
	if b.enableUIOutputParsing && b.uiOutputParser != nil && result.Content != "" {
		parseResult := b.uiOutputParser.Parse(result.Content)

		// If UI blocks were found, add them to structured response
		if len(parseResult.UIBlocks) > 0 {
			// Stream UI block events if handlers are registered
			for _, block := range parseResult.UIBlocks {
				if b.onUIPartStart != nil {
					b.onUIPartStart(fmt.Sprintf("parsed_%s", block.Type), string(block.Type))
				}

				if b.onUIPartEnd != nil && block.Part != nil {
					b.onUIPartEnd(fmt.Sprintf("parsed_%s", block.Type), block.Part)
				}
			}

			// If we have a structured response, add the parsed parts
			if result.StructuredResponse != nil {
				result.StructuredResponse.Parts = append(result.StructuredResponse.Parts, parseResult.Parts...)
			} else {
				// Create a new structured response with parsed UI blocks
				result.StructuredResponse = ConvertToStructuredResponse(parseResult)
			}

			// Update content to clean version (UI blocks removed)
			result.ParsedUIContent = parseResult.CleanContent
		}
	}

	// Generate suggestions if manager is configured
	if b.suggestionManager != nil {
		result.Suggestions = b.generateSuggestions(result)
	}

	// Log completion
	if b.logger != nil {
		totalTokens := 0
		if result.Usage != nil {
			totalTokens = result.Usage.InputTokens + result.Usage.OutputTokens
		}

		b.logger.Info("Streaming generation completed",
			logger.String("execution_id", result.ExecutionID),
			logger.Int("tokens", totalTokens),
			logger.Duration("duration", result.Duration),
			logger.Int("reasoning_steps", len(result.ReasoningSteps)),
			logger.Int("suggestions", len(result.Suggestions)),
			logger.Int("artifacts", len(result.Artifacts)),
			logger.Int("citations", len(result.Citations)),
			logger.Bool("structured_response", result.StructuredResponse != nil),
			logger.Int("content_length", len(result.Content)),
			logger.Int("thinking_length", len(result.ThinkingContent)),
			logger.Int("tool_calls", len(result.ToolCalls)),
		)
	}

	if b.metrics != nil {
		b.metrics.Counter("forge.ai.sdk.stream.success").Inc()
		b.metrics.Histogram("forge.ai.sdk.stream.duration").Observe(result.Duration.Seconds())

		if result.Usage != nil {
			totalTokens := result.Usage.InputTokens + result.Usage.OutputTokens
			b.metrics.Histogram("forge.ai.sdk.stream.tokens").Observe(float64(totalTokens))
		}
	}

	if b.onComplete != nil {
		b.onComplete(*result)
	}

	return result, nil
}

// executeToolCalls executes a batch of tool calls using the tool registry and returns the results.
// It emits tool result events for each tool execution.
func (b *StreamingGenerator) executeToolCalls(
	ctx context.Context,
	executionID string,
	toolCalls []ToolInvocation,
	streamHandler *llm.ClientStreamHandler,
) []ToolExecutionResult {
	results := make([]ToolExecutionResult, 0, len(toolCalls))

	for _, tc := range toolCalls {
		if b.toolRegistry == nil {
			continue
		}

		// Parse arguments from raw JSON if present
		args := tc.Arguments
		if rawArgs, ok := args["raw"].(string); ok && rawArgs != "" {
			var parsedArgs map[string]any
			if err := json.Unmarshal([]byte(rawArgs), &parsedArgs); err == nil {
				args = parsedArgs
			}
		}

		// Generate tool ID if not present
		toolID := tc.Name
		if id, ok := tc.Arguments["id"].(string); ok && id != "" {
			toolID = id
		}

		// Fire tool execution start callback
		if b.onToolExecutionStart != nil {
			b.onToolExecutionStart(tc.Name, args)
		}

		// Emit tool result start event
		if b.onToolResultStart != nil {
			b.onToolResultStart(executionID, toolID, tc.Name)
		}

		if b.onStreamEvent != nil {
			b.onStreamEvent(llm.NewToolResultStartEvent(executionID, toolID, tc.Name))
		}

		// Create timeout context for this tool
		toolCtx, cancel := context.WithTimeout(ctx, b.toolTimeout)

		var execResult *ToolExecutionResult

		// Check if this is a presentation tool
		if b.includePresentationTools && IsPresentationTool(tc.Name) {
			// Execute presentation tool with UI streaming
			onEvent := func(event llm.ClientStreamEvent) error {
				if b.onStreamEvent != nil {
					b.onStreamEvent(event)
				}

				return nil
			}

			uiResult, err := ExecutePresentationTool(toolCtx, tc.Name, args, onEvent, executionID)
			if err != nil {
				execResult = &ToolExecutionResult{
					ToolName:  tc.Name,
					Success:   false,
					Error:     err,
					Timestamp: time.Now(),
					Metadata:  make(map[string]any),
				}
			} else {
				execResult = &ToolExecutionResult{
					ToolName:  tc.Name,
					Success:   uiResult.Success,
					Result:    uiResult.Result,
					Duration:  uiResult.Duration,
					Timestamp: uiResult.Timestamp,
					Metadata:  uiResult.Metadata,
				}
				if uiResult.Error != nil {
					execResult.Error = uiResult.Error
					execResult.Success = false
				}
			}
		} else {
			// Execute regular tool through registry
			execResult, _ = b.toolRegistry.ExecuteTool(toolCtx, tc.Name, "", args)
		}

		cancel()

		if execResult == nil {
			execResult = &ToolExecutionResult{
				ToolName:  tc.Name,
				Timestamp: time.Now(),
				Metadata:  make(map[string]any),
			}
		}

		// Handle execution errors (for regular tools)
		if execResult.Error != nil {
			execResult.Success = false
		} else if !execResult.Success {
			// Success wasn't explicitly set - default to true if no error
			execResult.Success = true
		}

		// Convert result to string for streaming
		var resultContent string
		if execResult.Error != nil {
			resultContent = "Error: " + execResult.Error.Error()
		} else {
			// Try to JSON encode the result
			resultBytes, err := json.Marshal(execResult.Result)
			if err != nil {
				resultContent = fmt.Sprintf("%v", execResult.Result)
			} else {
				resultContent = string(resultBytes)
			}
		}

		// Emit tool result delta event with the result content
		if b.onToolResultDelta != nil {
			b.onToolResultDelta(executionID, toolID, resultContent, streamHandler.GetIndex())
		}

		if b.onStreamEvent != nil {
			b.onStreamEvent(llm.NewToolResultDeltaEvent(executionID, toolID, resultContent, streamHandler.GetIndex()))
		}

		// Emit tool result end event
		if b.onToolResultEnd != nil {
			b.onToolResultEnd(executionID, toolID)
		}

		if b.onStreamEvent != nil {
			b.onStreamEvent(llm.NewToolResultEndEvent(executionID, toolID))
		}

		// Fire tool execution end callback
		if b.onToolExecutionEnd != nil {
			b.onToolExecutionEnd(tc.Name, execResult)
		}

		results = append(results, *execResult)
	}

	return results
}

// buildToolResultMessages creates tool result messages from execution results.
func (b *StreamingGenerator) buildToolResultMessages(toolCalls []ToolInvocation, execResults []ToolExecutionResult) []llm.ChatMessage {
	messages := make([]llm.ChatMessage, 0, len(execResults)+1)

	// First, add an assistant message with the tool calls
	if len(toolCalls) > 0 {
		llmToolCalls := make([]llm.ToolCall, 0, len(toolCalls))
		for i, tc := range toolCalls {
			var toolID string
			if id, ok := tc.Arguments["id"].(string); ok && id != "" {
				toolID = id
			} else {
				toolID = fmt.Sprintf("%s_%d", tc.Name, i)
			}

			// Get raw arguments
			rawArgs := ""
			if raw, ok := tc.Arguments["raw"].(string); ok {
				rawArgs = raw
			} else {
				argsBytes, _ := json.Marshal(tc.Arguments)
				rawArgs = string(argsBytes)
			}

			llmToolCalls = append(llmToolCalls, llm.ToolCall{
				ID:   toolID,
				Type: "function",
				Function: &llm.FunctionCall{
					Name:      tc.Name,
					Arguments: rawArgs,
				},
			})
		}

		messages = append(messages, llm.ChatMessage{
			Role:      "assistant",
			ToolCalls: llmToolCalls,
		})
	}

	// Then, add tool result messages for each execution
	for i, result := range execResults {
		toolID := result.ToolName

		if i < len(toolCalls) {
			if id, ok := toolCalls[i].Arguments["id"].(string); ok && id != "" {
				toolID = id
			} else {
				toolID = fmt.Sprintf("%s_%d", result.ToolName, i)
			}
		}

		var content string
		if result.Error != nil {
			content = "Error: " + result.Error.Error()
		} else {
			// Try to JSON encode the result
			resultBytes, err := json.Marshal(result.Result)
			if err != nil {
				content = fmt.Sprintf("%v", result.Result)
			} else {
				content = string(resultBytes)
			}
		}

		messages = append(messages, llm.ChatMessage{
			Role:       "tool",
			Content:    content,
			ToolCallID: toolID,
			Name:       result.ToolName,
		})
	}

	return messages
}

// handleTypedStreamEvent handles typed ClientStreamEvents and dispatches to callbacks.
func (b *StreamingGenerator) handleTypedStreamEvent(event llm.ClientStreamEvent) error {
	// Fire the generic stream event callback
	if b.onStreamEvent != nil {
		b.onStreamEvent(event)
	}

	// Dispatch to specific typed callbacks
	switch event.Type {
	case llm.EventThinkingStart:
		if b.onThinkingStart != nil {
			b.onThinkingStart(event.ExecutionID)
		}
	case llm.EventThinkingDelta:
		if b.onThinkingDelta != nil {
			b.onThinkingDelta(event.ExecutionID, event.Delta, event.Index)
		}
	case llm.EventThinkingEnd:
		if b.onThinkingEnd != nil {
			b.onThinkingEnd(event.ExecutionID)
		}
	case llm.EventContentStart:
		if b.onContentStart != nil {
			b.onContentStart(event.ExecutionID)
		}
	case llm.EventContentDelta:
		if b.onContentDelta != nil {
			b.onContentDelta(event.ExecutionID, event.Delta, event.Index)
		}
	case llm.EventContentEnd:
		if b.onContentEnd != nil {
			b.onContentEnd(event.ExecutionID)
		}
	case llm.EventToolUseStart:
		if b.onToolUseStart != nil {
			b.onToolUseStart(event.ExecutionID, event.ToolID, event.ToolName)
		}
	case llm.EventToolUseDelta:
		if b.onToolUseDelta != nil {
			b.onToolUseDelta(event.ExecutionID, event.ToolID, event.Delta, event.Index)
		}
	case llm.EventToolUseEnd:
		if b.onToolUseEnd != nil {
			b.onToolUseEnd(event.ExecutionID, event.ToolID)
		}
	case llm.EventUIPartStart:
		if b.onUIPartStart != nil {
			b.onUIPartStart(event.PartID, event.PartType)
		}
	case llm.EventUIPartDelta:
		if b.onUIPartDelta != nil {
			b.onUIPartDelta(event.PartID, event.Section, event.PartData)
		}
	case llm.EventUIPartEnd:
		if b.onUIPartEnd != nil {
			// Try to build final part if we have a stream manager
			var part ContentPart

			if b.uiPartStreamManager != nil {
				if streamer, ok := b.uiPartStreamManager.GetStreamer(event.PartID); ok {
					part, _ = streamer.BuildFinalPart()
				}
			}

			b.onUIPartEnd(event.PartID, part)
		}
	}

	return nil
}

// handleBlockEvent handles block-level events from providers like Anthropic.
func (b *StreamingGenerator) handleBlockEvent(
	event llm.ChatStreamEvent,
	result *StreamResponse,
	fullContent *strings.Builder,
	thinkingContent *strings.Builder,
	currentToolArgs *strings.Builder,
	currentToolID *string,
	currentToolName *string,
) error {
	blockType := event.BlockType
	blockState := event.BlockState

	// Get content from choices
	var content string
	if len(event.Choices) > 0 && event.Choices[0].Delta != nil {
		content = event.Choices[0].Delta.Content
	}

	switch blockType {
	case string(llm.BlockTypeThinking):
		switch blockState {
		case string(llm.BlockStateStart):
			if b.onThinkingStart != nil {
				b.onThinkingStart(result.ExecutionID)
			}
		case string(llm.BlockStateDelta):
			if content != "" {
				thinkingContent.WriteString(content)

				if b.onThinkingDelta != nil {
					b.onThinkingDelta(result.ExecutionID, content, int64(event.BlockIndex))
				}

				if b.onReasoning != nil {
					b.onReasoning(content)
				}
			}
		case string(llm.BlockStateStop):
			if b.onThinkingEnd != nil {
				b.onThinkingEnd(result.ExecutionID)
			}

			if thinkingContent.Len() > 0 {
				result.ReasoningSteps = append(result.ReasoningSteps, thinkingContent.String())
			}
		}

	case string(llm.BlockTypeText):
		switch blockState {
		case string(llm.BlockStateStart):
			if b.onContentStart != nil {
				b.onContentStart(result.ExecutionID)
			}
		case string(llm.BlockStateDelta):
			if content != "" {
				fullContent.WriteString(content)

				if b.onContentDelta != nil {
					b.onContentDelta(result.ExecutionID, content, int64(event.BlockIndex))
				}

				if b.onToken != nil {
					b.onToken(content)
				}
			}
		case string(llm.BlockStateStop):
			if b.onContentEnd != nil {
				b.onContentEnd(result.ExecutionID)
			}
		}

	case string(llm.BlockTypeToolUse):
		switch blockState {
		case string(llm.BlockStateStart):
			// Get tool info from choices
			if len(event.Choices) > 0 && event.Choices[0].Delta != nil && len(event.Choices[0].Delta.ToolCalls) > 0 {
				tc := event.Choices[0].Delta.ToolCalls[0]

				*currentToolID = tc.ID
				if tc.Function != nil {
					*currentToolName = tc.Function.Name
				}
			}

			if b.onToolUseStart != nil {
				b.onToolUseStart(result.ExecutionID, *currentToolID, *currentToolName)
			}
		case string(llm.BlockStateDelta):
			// Get args from choices
			var args string

			if len(event.Choices) > 0 && event.Choices[0].Delta != nil && len(event.Choices[0].Delta.ToolCalls) > 0 {
				tc := event.Choices[0].Delta.ToolCalls[0]
				if tc.Function != nil {
					args = tc.Function.Arguments
				}
			}

			if args != "" {
				currentToolArgs.WriteString(args)

				if b.onToolUseDelta != nil {
					b.onToolUseDelta(result.ExecutionID, *currentToolID, args, int64(event.BlockIndex))
				}
			}
		case string(llm.BlockStateStop):
			if b.onToolUseEnd != nil {
				b.onToolUseEnd(result.ExecutionID, *currentToolID)
			}
			// Store completed tool call
			if *currentToolID != "" {
				toolCall := ToolInvocation{
					Name: *currentToolName,
					Arguments: map[string]any{
						"raw": currentToolArgs.String(),
					},
				}

				result.ToolCalls = append(result.ToolCalls, toolCall)
				if b.onToolCall != nil {
					b.onToolCall(*currentToolName, toolCall.Arguments)
				}
			}
			// Reset
			*currentToolID = ""
			*currentToolName = ""

			currentToolArgs.Reset()
		}
	}

	return nil
}

// renderPrompt renders the prompt template with variables.
func (b *StreamingGenerator) renderPrompt() (string, error) {
	return prompt.Render(b.prompt, b.vars)
}

// buildMessages constructs the message array for the LLM request.
func (b *StreamingGenerator) buildMessages(userPrompt string) []llm.ChatMessage {
	return messages.BuildWithHistoryFirst(b.messages, b.systemPrompt, userPrompt)
}

// parseToStructuredResponse parses the result into a structured response.
func (b *StreamingGenerator) parseToStructuredResponse(result *StreamResponse) *StructuredResponse {
	builder := NewResponseBuilder().
		WithMetadata(b.model, b.provider, result.Duration)

	if result.Usage != nil {
		builder.WithTokenUsage(result.Usage.InputTokens, result.Usage.OutputTokens)
	}

	// Parse content into parts
	parts := b.responseParser.Parse(result.Content)
	for _, part := range parts {
		builder.AddPart(part)

		// Callback for each content part
		if b.onContentPart != nil {
			b.onContentPart(part)
		}

		// Extract artifacts from code blocks
		if b.artifactRegistry != nil {
			if codePart, ok := part.(*CodePart); ok {
				artifact := NewCodeArtifact(
					fmt.Sprintf("code_%d", time.Now().UnixNano()),
					codePart.Language,
					codePart.Code,
				)
				if err := b.artifactRegistry.Create(artifact); err == nil {
					result.Artifacts = append(result.Artifacts, *artifact)
					builder.AddArtifact(*artifact)
				}
			}
		}
	}

	// Add reasoning as thinking parts
	for _, reasoning := range result.ReasoningSteps {
		builder.AddThinking(reasoning)
	}

	// Add citations if available
	if b.citationManager != nil {
		for _, citation := range b.citationManager.GetCitations() {
			builder.AddCitation(citation)
		}
	}

	return builder.Build()
}

// generateSuggestions generates follow-up suggestions based on the result.
func (b *StreamingGenerator) generateSuggestions(result *StreamResponse) []Suggestion {
	input := SuggestionInput{
		Content: result.Content,
		Query:   b.prompt,
		Context: result.Metadata,
	}

	// Extract topics from content (simple extraction)
	input.Topics = extractTopics(result.Content)

	return b.suggestionManager.GenerateSuggestions(b.ctx, input)
}

// extractTopics extracts potential topics from content.
// This is a simple implementation - production would use NLP.
func extractTopics(content string) []string {
	topics := make([]string, 0)

	// Look for capitalized phrases that might be topics
	words := strings.Fields(content)
	for i, word := range words {
		// Skip common articles and short words
		if len(word) < 4 {
			continue
		}

		// Check for capitalized words that aren't at sentence start
		if i > 0 && len(word) > 0 {
			firstChar := word[0]
			if firstChar >= 'A' && firstChar <= 'Z' {
				// Clean up punctuation
				clean := strings.Trim(word, ".,!?;:\"'()[]")
				if len(clean) > 3 && !slices.Contains(topics, clean) {
					topics = append(topics, clean)
				}
			}
		}

		// Limit topics
		if len(topics) >= 5 {
			break
		}
	}

	return topics
}
