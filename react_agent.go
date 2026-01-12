package sdk

import (
	"context"
	"fmt"
	"time"

	logger "github.com/xraph/go-utils/log"
	"github.com/xraph/go-utils/metrics"
)

// ReactAgent is an agent that uses the ReAct (Reasoning + Acting) strategy.
// It alternates between reasoning and acting until reaching a final answer.
type ReactAgent struct {
	*EnhancedAgent
	strategy        *ReactStrategy
	reasoningPrompt string
}

// ReactAgentBuilder helps construct a ReactAgent.
type ReactAgentBuilder struct {
	// Base agent configuration
	id          string
	name        string
	description string
	model       string
	provider    string

	// Execution configuration
	systemPrompt  string
	tools         []Tool
	maxIterations int
	temperature   float64

	// ReAct-specific configuration
	reflectionInterval  int
	confidenceThreshold float64
	reasoningPrompt     string

	// Dependencies
	llmManager    LLMManager
	stateStore    StateStore
	memoryManager *MemoryManager
	logger        logger.Logger
	metrics       metrics.Metrics

	// Optional components
	guardrails *GuardrailManager
}

// NewReactAgentBuilder creates a new builder for ReactAgent.
func NewReactAgentBuilder(name string) *ReactAgentBuilder {
	return &ReactAgentBuilder{
		name:                name,
		maxIterations:       10,
		temperature:         0.7,
		reflectionInterval:  3,
		confidenceThreshold: 0.7,
	}
}

// WithID sets the agent ID.
func (b *ReactAgentBuilder) WithID(id string) *ReactAgentBuilder {
	b.id = id
	return b
}

// WithDescription sets the agent description.
func (b *ReactAgentBuilder) WithDescription(desc string) *ReactAgentBuilder {
	b.description = desc
	return b
}

// WithModel sets the LLM model.
func (b *ReactAgentBuilder) WithModel(model string) *ReactAgentBuilder {
	b.model = model
	return b
}

// WithProvider sets the LLM provider.
func (b *ReactAgentBuilder) WithProvider(provider string) *ReactAgentBuilder {
	b.provider = provider
	return b
}

// WithSystemPrompt sets the system prompt.
func (b *ReactAgentBuilder) WithSystemPrompt(prompt string) *ReactAgentBuilder {
	b.systemPrompt = prompt
	return b
}

// WithTools sets the available tools.
func (b *ReactAgentBuilder) WithTools(tools ...Tool) *ReactAgentBuilder {
	b.tools = append(b.tools, tools...)
	return b
}

// WithMaxIterations sets the maximum reasoning steps.
func (b *ReactAgentBuilder) WithMaxIterations(max int) *ReactAgentBuilder {
	b.maxIterations = max
	return b
}

// WithTemperature sets the LLM temperature.
func (b *ReactAgentBuilder) WithTemperature(temp float64) *ReactAgentBuilder {
	b.temperature = temp
	return b
}

// WithReflectionInterval sets how often to self-reflect.
func (b *ReactAgentBuilder) WithReflectionInterval(interval int) *ReactAgentBuilder {
	b.reflectionInterval = interval
	return b
}

// WithConfidenceThreshold sets the minimum confidence level.
func (b *ReactAgentBuilder) WithConfidenceThreshold(threshold float64) *ReactAgentBuilder {
	b.confidenceThreshold = threshold
	return b
}

// WithReasoningPrompt sets a custom reasoning prompt template.
func (b *ReactAgentBuilder) WithReasoningPrompt(prompt string) *ReactAgentBuilder {
	b.reasoningPrompt = prompt
	return b
}

// WithLLMManager sets the LLM manager.
func (b *ReactAgentBuilder) WithLLMManager(manager LLMManager) *ReactAgentBuilder {
	b.llmManager = manager
	return b
}

// WithStateStore sets the state store.
func (b *ReactAgentBuilder) WithStateStore(store StateStore) *ReactAgentBuilder {
	b.stateStore = store
	return b
}

// WithMemoryManager sets the memory manager.
func (b *ReactAgentBuilder) WithMemoryManager(manager *MemoryManager) *ReactAgentBuilder {
	b.memoryManager = manager
	return b
}

// WithLogger sets the logger.
func (b *ReactAgentBuilder) WithLogger(logger logger.Logger) *ReactAgentBuilder {
	b.logger = logger
	return b
}

// WithMetrics sets the metrics collector.
func (b *ReactAgentBuilder) WithMetrics(metrics metrics.Metrics) *ReactAgentBuilder {
	b.metrics = metrics
	return b
}

// WithGuardrails sets the guardrail manager.
func (b *ReactAgentBuilder) WithGuardrails(guardrails *GuardrailManager) *ReactAgentBuilder {
	b.guardrails = guardrails
	return b
}

// Build creates the ReactAgent.
func (b *ReactAgentBuilder) Build() (*ReactAgent, error) {
	// Validate required fields
	if b.name == "" {
		return nil, fmt.Errorf("agent name is required")
	}
	if b.llmManager == nil {
		return nil, fmt.Errorf("LLM manager is required")
	}

	// Generate ID if not provided
	if b.id == "" {
		b.id = fmt.Sprintf("react_agent_%d", time.Now().UnixNano())
	}

	// Create base enhanced agent
	baseBuilder := NewEnhancedAgentBuilder(b.name).
		WithID(b.id).
		WithDescription(b.description).
		WithModel(b.model).
		WithProvider(b.provider).
		WithSystemPrompt(b.systemPrompt).
		WithTools(b.tools...).
		WithLLMManager(b.llmManager).
		WithLogger(b.logger).
		WithMetrics(b.metrics).
		WithMaxIterations(b.maxIterations).
		WithTemperature(b.temperature)

	if b.guardrails != nil {
		baseBuilder.WithGuardrails(b.guardrails)
	}

	if b.stateStore != nil {
		// Enhanced agent doesn't have state store, so we skip it for now
	}

	enhancedAgent, err := baseBuilder.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build base agent: %w", err)
	}

	// Create ReAct strategy
	strategyConfig := &ReactStrategyConfig{
		MaxIterations:       b.maxIterations,
		ReflectionInterval:  b.reflectionInterval,
		ConfidenceThreshold: b.confidenceThreshold,
		MemoryManager:       b.memoryManager,
		ReasoningPrompt:     b.reasoningPrompt,
	}

	strategy := NewReactStrategy(b.logger, b.metrics, strategyConfig)

	// Create ReactAgent
	reactAgent := &ReactAgent{
		EnhancedAgent:   enhancedAgent,
		strategy:        strategy,
		reasoningPrompt: b.reasoningPrompt,
	}

	return reactAgent, nil
}

// Execute runs the agent using the ReAct strategy.
func (a *ReactAgent) Execute(ctx context.Context, input string) (*AgentExecution, error) {
	if a.logger != nil {
		a.logger.Info("Executing ReactAgent",
			logger.String("agent_id", a.ID),
			logger.String("input", input),
		)
	}

	execution, err := a.strategy.Execute(ctx, a.EnhancedAgent, input)
	if err != nil {
		if a.logger != nil {
			a.logger.Error("ReactAgent execution failed",
				logger.String("agent_id", a.ID),
				logger.String("error", err.Error()),
			)
		}
		return execution, err
	}

	if a.metrics != nil {
		a.metrics.Counter("forge.ai.sdk.react_agent.executions",
			metrics.WithLabel("agent_id", a.ID),
			metrics.WithLabel("status", string(execution.Status)),
		).Inc()
	}

	return execution, nil
}

// GetTraces returns the reasoning traces from the last execution.
func (a *ReactAgent) GetTraces() []ReasoningTrace {
	return a.strategy.GetTraces()
}

// GetReflections returns the reflection results from the last execution.
func (a *ReactAgent) GetReflections() []ReflectionResult {
	return a.strategy.GetReflections()
}

// GetStrategy returns the underlying ReAct strategy.
func (a *ReactAgent) GetStrategy() *ReactStrategy {
	return a.strategy
}

// SetReasoningPrompt updates the reasoning prompt template.
func (a *ReactAgent) SetReasoningPrompt(prompt string) {
	a.reasoningPrompt = prompt
	// Update strategy's prompt if needed
}

// AsTool converts the ReactAgent into a Tool that can be used by other agents.
func (a *ReactAgent) AsTool() Tool {
	return Tool{
		Name:        "call_" + a.Name,
		Description: a.getToolDescription(),
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"input": map[string]any{
					"type":        "string",
					"description": "The question or task for the " + a.Name + " agent",
				},
			},
			"required": []string{"input"},
		},
		Handler: func(ctx context.Context, params map[string]any) (any, error) {
			input, ok := params["input"].(string)
			if !ok {
				return nil, fmt.Errorf("input parameter is required and must be a string")
			}

			execution, err := a.Execute(ctx, input)
			if err != nil {
				return nil, err
			}

			return map[string]any{
				"output":     execution.FinalOutput,
				"iterations": len(execution.Steps),
				"traces":     a.GetTraces(),
			}, nil
		},
	}
}

func (a *ReactAgent) getToolDescription() string {
	desc := a.Description
	if desc == "" {
		desc = fmt.Sprintf("Use ReAct reasoning to solve: %s", a.Name)
	}

	if len(a.tools) > 0 {
		toolNames := make([]string, len(a.tools))
		for i, tool := range a.tools {
			toolNames[i] = tool.Name
		}
		desc += fmt.Sprintf(" Available tools: %v", toolNames)
	}

	return desc
}
