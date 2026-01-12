package sdk

import (
	"context"
	"time"
)

// ExecutionStrategy defines how an agent executes tasks.
// This allows different reasoning patterns (ReAct, Plan-Execute, etc.) to be plugged in.
type ExecutionStrategy interface {
	// Execute runs the strategy with the given agent and input
	Execute(ctx context.Context, agent *EnhancedAgent, input string) (*AgentExecution, error)

	// Name returns the strategy name
	Name() string

	// SupportsReplanning indicates if this strategy can replan on failures
	SupportsReplanning() bool
}

// ReasoningTrace captures the thought process during agent execution.
// This is the core data structure for ReAct-style reasoning.
type ReasoningTrace struct {
	// Step number in the reasoning sequence
	Step int `json:"step"`

	// Thought is what the agent is thinking about
	Thought string `json:"thought"`

	// Action is the tool/action the agent decided to take
	Action string `json:"action,omitempty"`

	// ActionInput contains the arguments for the action
	ActionInput map[string]any `json:"action_input,omitempty"`

	// Observation is the result/feedback from the action
	Observation string `json:"observation,omitempty"`

	// Reflection is self-assessment of the reasoning quality
	Reflection string `json:"reflection,omitempty"`

	// Confidence is a 0-1 score of how confident the agent is
	Confidence float64 `json:"confidence"`

	// Timestamp when this trace was created
	Timestamp time.Time `json:"timestamp"`

	// Metadata for additional context
	Metadata map[string]any `json:"metadata,omitempty"`
}

// ReflectionResult evaluates the quality of reasoning or plans.
type ReflectionResult struct {
	// Quality assessment: "good", "needs_improvement", "invalid"
	Quality string `json:"quality"`

	// Issues identified during reflection
	Issues []string `json:"issues,omitempty"`

	// Suggestions for improvement
	Suggestions []string `json:"suggestions,omitempty"`

	// ShouldReplan indicates if replanning is recommended
	ShouldReplan bool `json:"should_replan"`

	// Score is a 0-1 quality score
	Score float64 `json:"score"`

	// Reasoning explains the reflection assessment
	Reasoning string `json:"reasoning,omitempty"`
}

// ReflectionCriterion defines a criterion for evaluating reasoning quality.
type ReflectionCriterion struct {
	Name        string
	Description string
	Weight      float64 // 0-1, importance of this criterion
	Evaluator   func(context.Context, *ReasoningTrace) (float64, error)
}

// StrategyConfig provides common configuration for execution strategies.
type StrategyConfig struct {
	// MaxIterations limits the number of reasoning steps
	MaxIterations int

	// Timeout for the entire strategy execution
	Timeout time.Duration

	// EnableReflection turns on periodic self-reflection
	EnableReflection bool

	// ReflectionInterval is how often to reflect (every N steps)
	ReflectionInterval int

	// ConfidenceThreshold below which to trigger replanning
	ConfidenceThreshold float64

	// EnableMemory uses MemoryManager for context
	EnableMemory bool

	// Metadata for strategy-specific config
	Metadata map[string]any
}

