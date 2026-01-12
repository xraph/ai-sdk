package sdk

import (
	"context"
	"fmt"
	"time"

	logger "github.com/xraph/go-utils/log"
	"github.com/xraph/go-utils/metrics"
)

// PlanExecuteAgent is an agent that uses the Plan-Execute strategy.
// It decomposes tasks into plans, executes them, and verifies results.
type PlanExecuteAgent struct {
	*Agent
	strategy *PlanExecuteStrategy
}

// PlanExecuteAgentBuilder helps construct a PlanExecuteAgent.
type PlanExecuteAgentBuilder struct {
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

	// Plan-Execute specific configuration
	allowReplanning   bool
	verifySteps       bool
	maxReplanAttempts int
	timeout           time.Duration

	// Dependencies
	llmManager    LLMManager
	stateStore    StateStore
	planStore     PlanStore
	memoryManager *MemoryManager
	logger        logger.Logger
	metrics       metrics.Metrics

	// Optional components
	guardrails *GuardrailManager

	// Separate LLMs for different phases (optional)
	plannerLLM  LLMManager
	executorLLM LLMManager
	verifierLLM LLMManager
}

// NewPlanExecuteAgentBuilder creates a new builder for PlanExecuteAgent.
func NewPlanExecuteAgentBuilder(name string) *PlanExecuteAgentBuilder {
	return &PlanExecuteAgentBuilder{
		name:              name,
		maxIterations:     10,
		temperature:       0.7,
		allowReplanning:   true,
		verifySteps:       true,
		maxReplanAttempts: 3,
		timeout:           10 * time.Minute,
	}
}

// WithID sets the agent ID.
func (b *PlanExecuteAgentBuilder) WithID(id string) *PlanExecuteAgentBuilder {
	b.id = id
	return b
}

// WithDescription sets the agent description.
func (b *PlanExecuteAgentBuilder) WithDescription(desc string) *PlanExecuteAgentBuilder {
	b.description = desc
	return b
}

// WithModel sets the LLM model.
func (b *PlanExecuteAgentBuilder) WithModel(model string) *PlanExecuteAgentBuilder {
	b.model = model
	return b
}

// WithProvider sets the LLM provider.
func (b *PlanExecuteAgentBuilder) WithProvider(provider string) *PlanExecuteAgentBuilder {
	b.provider = provider
	return b
}

// WithSystemPrompt sets the system prompt.
func (b *PlanExecuteAgentBuilder) WithSystemPrompt(prompt string) *PlanExecuteAgentBuilder {
	b.systemPrompt = prompt
	return b
}

// WithTools sets the available tools.
func (b *PlanExecuteAgentBuilder) WithTools(tools ...Tool) *PlanExecuteAgentBuilder {
	b.tools = append(b.tools, tools...)
	return b
}

// WithMaxIterations sets the maximum plan steps.
func (b *PlanExecuteAgentBuilder) WithMaxIterations(max int) *PlanExecuteAgentBuilder {
	b.maxIterations = max
	return b
}

// WithTemperature sets the LLM temperature.
func (b *PlanExecuteAgentBuilder) WithTemperature(temp float64) *PlanExecuteAgentBuilder {
	b.temperature = temp
	return b
}

// WithAllowReplanning enables or disables replanning.
func (b *PlanExecuteAgentBuilder) WithAllowReplanning(allow bool) *PlanExecuteAgentBuilder {
	b.allowReplanning = allow
	return b
}

// WithVerifySteps enables or disables step verification.
func (b *PlanExecuteAgentBuilder) WithVerifySteps(verify bool) *PlanExecuteAgentBuilder {
	b.verifySteps = verify
	return b
}

// WithMaxReplanAttempts sets the maximum replanning attempts.
func (b *PlanExecuteAgentBuilder) WithMaxReplanAttempts(max int) *PlanExecuteAgentBuilder {
	b.maxReplanAttempts = max
	return b
}

// WithTimeout sets the execution timeout.
func (b *PlanExecuteAgentBuilder) WithTimeout(timeout time.Duration) *PlanExecuteAgentBuilder {
	b.timeout = timeout
	return b
}

// WithLLMManager sets the LLM manager for all phases.
func (b *PlanExecuteAgentBuilder) WithLLMManager(manager LLMManager) *PlanExecuteAgentBuilder {
	b.llmManager = manager
	return b
}

// WithPlannerLLM sets a dedicated LLM for planning phase.
func (b *PlanExecuteAgentBuilder) WithPlannerLLM(manager LLMManager) *PlanExecuteAgentBuilder {
	b.plannerLLM = manager
	return b
}

// WithExecutorLLM sets a dedicated LLM for execution phase.
func (b *PlanExecuteAgentBuilder) WithExecutorLLM(manager LLMManager) *PlanExecuteAgentBuilder {
	b.executorLLM = manager
	return b
}

// WithVerifierLLM sets a dedicated LLM for verification phase.
func (b *PlanExecuteAgentBuilder) WithVerifierLLM(manager LLMManager) *PlanExecuteAgentBuilder {
	b.verifierLLM = manager
	return b
}

// WithStateStore sets the state store.
func (b *PlanExecuteAgentBuilder) WithStateStore(store StateStore) *PlanExecuteAgentBuilder {
	b.stateStore = store
	return b
}

// WithPlanStore sets the plan store.
func (b *PlanExecuteAgentBuilder) WithPlanStore(store PlanStore) *PlanExecuteAgentBuilder {
	b.planStore = store
	return b
}

// WithMemoryManager sets the memory manager.
func (b *PlanExecuteAgentBuilder) WithMemoryManager(manager *MemoryManager) *PlanExecuteAgentBuilder {
	b.memoryManager = manager
	return b
}

// WithLogger sets the logger.
func (b *PlanExecuteAgentBuilder) WithLogger(logger logger.Logger) *PlanExecuteAgentBuilder {
	b.logger = logger
	return b
}

// WithMetrics sets the metrics collector.
func (b *PlanExecuteAgentBuilder) WithMetrics(metrics metrics.Metrics) *PlanExecuteAgentBuilder {
	b.metrics = metrics
	return b
}

// WithGuardrails sets the guardrail manager.
func (b *PlanExecuteAgentBuilder) WithGuardrails(guardrails *GuardrailManager) *PlanExecuteAgentBuilder {
	b.guardrails = guardrails
	return b
}

// Build creates the PlanExecuteAgent.
func (b *PlanExecuteAgentBuilder) Build() (*PlanExecuteAgent, error) {
	// Validate required fields
	if b.name == "" {
		return nil, fmt.Errorf("agent name is required")
	}
	if b.llmManager == nil {
		return nil, fmt.Errorf("LLM manager is required")
	}

	// Generate ID if not provided
	if b.id == "" {
		b.id = fmt.Sprintf("plan_execute_agent_%d", time.Now().UnixNano())
	}

	// Create base agent
	baseBuilder := NewAgentBuilder().
		WithID(b.id).
		WithName(b.name).
		WithDescription(b.description).
		WithModel(b.model).
		WithProvider(b.provider).
		WithSystemPrompt(b.systemPrompt).
		WithTools(b.tools...).
		WithLLMManager(b.llmManager).
		WithStateStore(b.stateStore).
		WithLogger(b.logger).
		WithMetrics(b.metrics).
		WithMaxIterations(b.maxIterations).
		WithTemperature(b.temperature)

	if b.guardrails != nil {
		baseBuilder.WithGuardrails(b.guardrails)
	}

	agent, err := baseBuilder.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build base agent: %w", err)
	}

	// Set default LLMs for phases if not specified
	plannerLLM := b.plannerLLM
	if plannerLLM == nil {
		plannerLLM = b.llmManager
	}

	executorLLM := b.executorLLM
	if executorLLM == nil {
		executorLLM = b.llmManager
	}

	verifierLLM := b.verifierLLM
	if verifierLLM == nil {
		verifierLLM = b.llmManager
	}

	// Create Plan-Execute strategy
	strategyConfig := &PlanExecuteStrategyConfig{
		Planner:           plannerLLM,
		Executor:          executorLLM,
		Verifier:          verifierLLM,
		PlanStore:         b.planStore,
		MemoryManager:     b.memoryManager,
		AllowReplanning:   b.allowReplanning,
		VerifySteps:       b.verifySteps,
		MaxReplanAttempts: b.maxReplanAttempts,
		Timeout:           b.timeout,
	}

	strategy := NewPlanExecuteStrategy(b.logger, b.metrics, strategyConfig)

	// Create PlanExecuteAgent
	planExecuteAgent := &PlanExecuteAgent{
		Agent:    agent,
		strategy: strategy,
	}

	return planExecuteAgent, nil
}

// Execute runs the agent using the Plan-Execute strategy.
func (a *PlanExecuteAgent) Execute(ctx context.Context, input string) (*AgentExecution, error) {
	if a.logger != nil {
		a.logger.Info("Executing PlanExecuteAgent",
			logger.String("agent_id", a.ID),
			logger.String("input", input),
		)
	}

	execution, err := a.strategy.Execute(ctx, a.Agent, input)
	if err != nil {
		if a.logger != nil {
			a.logger.Error("PlanExecuteAgent execution failed",
				logger.String("agent_id", a.ID),
				logger.String("error", err.Error()),
			)
		}
		return execution, err
	}

	if a.metrics != nil {
		a.metrics.Counter("forge.ai.sdk.plan_execute_agent.executions",
			metrics.WithLabel("agent_id", a.ID),
			metrics.WithLabel("status", string(execution.Status)),
		).Inc()
	}

	return execution, nil
}

// GetCurrentPlan returns the current plan being executed.
func (a *PlanExecuteAgent) GetCurrentPlan() *Plan {
	return a.strategy.GetCurrentPlan()
}

// GetPlanHistory returns all plans created during execution.
func (a *PlanExecuteAgent) GetPlanHistory() []*Plan {
	return a.strategy.GetPlanHistory()
}

// GetStrategy returns the underlying Plan-Execute strategy.
func (a *PlanExecuteAgent) GetStrategy() *PlanExecuteStrategy {
	return a.strategy
}

// AsTool converts the PlanExecuteAgent into a Tool that can be used by other agents.
func (a *PlanExecuteAgent) AsTool() Tool {
	return Tool{
		Name:        "call_" + a.Name,
		Description: a.getToolDescription(),
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"input": map[string]any{
					"type":        "string",
					"description": "The task for the " + a.Name + " agent to plan and execute",
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
				"output": execution.FinalOutput,
				"plan":   a.GetCurrentPlan(),
				"steps":  len(execution.Steps),
			}, nil
		},
	}
}

func (a *PlanExecuteAgent) getToolDescription() string {
	desc := a.Description
	if desc == "" {
		desc = fmt.Sprintf("Use plan-execute strategy to solve: %s", a.Name)
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
