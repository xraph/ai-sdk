package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/xraph/ai-sdk/llm"
	logger "github.com/xraph/go-utils/log"
	"github.com/xraph/go-utils/metrics"
)

// ReplanEngine manages intelligent replanning for failed or low-quality plans.
type ReplanEngine struct {
	llmManager    LLMManager
	provider      string
	model         string
	logger        logger.Logger
	metrics       metrics.Metrics
	memoryManager *MemoryManager

	// Configuration
	triggers          []ReplanTrigger
	learningEnabled   bool
	maxReplanAttempts int
	promptTemplate    string

	// Learning storage
	failurePatterns map[string]int // Track common failure patterns
}

// ReplanTrigger defines conditions that trigger replanning.
type ReplanTrigger struct {
	Name        string                              `json:"name"`
	Description string                              `json:"description"`
	Condition   func(*Plan, *ReflectionResult) bool `json:"-"`
	Priority    int                                 `json:"priority"` // Higher priority triggers first
}

// ReplanEngineConfig configures the replanning engine.
type ReplanEngineConfig struct {
	LLMManager        LLMManager
	Provider          string
	Model             string
	MemoryManager     *MemoryManager
	Triggers          []ReplanTrigger
	LearningEnabled   bool
	MaxReplanAttempts int
	PromptTemplate    string
}

// NewReplanEngine creates a new replanning engine.
func NewReplanEngine(logger logger.Logger, metrics metrics.Metrics, config *ReplanEngineConfig) *ReplanEngine {
	if config == nil {
		config = &ReplanEngineConfig{}
	}

	// Set defaults
	if config.MaxReplanAttempts == 0 {
		config.MaxReplanAttempts = 3
	}

	if len(config.Triggers) == 0 {
		config.Triggers = getDefaultReplanTriggers()
	}

	if config.PromptTemplate == "" {
		config.PromptTemplate = getDefaultReplanPrompt()
	}

	return &ReplanEngine{
		llmManager:        config.LLMManager,
		provider:          config.Provider,
		model:             config.Model,
		logger:            logger,
		metrics:           metrics,
		memoryManager:     config.MemoryManager,
		triggers:          config.Triggers,
		learningEnabled:   config.LearningEnabled,
		maxReplanAttempts: config.MaxReplanAttempts,
		promptTemplate:    config.PromptTemplate,
		failurePatterns:   make(map[string]int),
	}
}

// ShouldReplan determines if replanning is needed based on triggers.
func (re *ReplanEngine) ShouldReplan(ctx context.Context, plan *Plan, reflection *ReflectionResult) (bool, string) {
	if plan == nil {
		return false, ""
	}

	// Check each trigger in priority order
	for _, trigger := range re.triggers {
		if trigger.Condition(plan, reflection) {
			if re.logger != nil {
				re.logger.Info("Replan trigger activated",
					logger.String("trigger", trigger.Name),
					logger.String("plan_id", plan.ID),
				)
			}

			if re.metrics != nil {
				re.metrics.Counter("forge.ai.sdk.replan.triggers",
					metrics.WithLabel("trigger", trigger.Name),
				).Inc()
			}

			return true, trigger.Name
		}
	}

	return false, ""
}

// Replan creates a new plan based on the failed plan and learnings.
func (re *ReplanEngine) Replan(ctx context.Context, originalPlan *Plan, failureContext string, agentID string, tools []Tool) (*Plan, error) {
	if re.logger != nil {
		re.logger.Info("Creating replan",
			logger.String("original_plan", originalPlan.ID),
			logger.String("context", failureContext),
		)
	}

	// Learn from failure if enabled
	if re.learningEnabled {
		re.learnFromFailure(ctx, originalPlan, failureContext)
	}

	// Recall similar past plans for learning
	var pastLearnings string
	if re.memoryManager != nil {
		successfulPlans, _ := RecallSuccessfulPlans(ctx, re.memoryManager, originalPlan.Goal, 3)
		failedPlans, _ := RecallFailedPlans(ctx, re.memoryManager, originalPlan.Goal, 2)

		if len(successfulPlans) > 0 || len(failedPlans) > 0 {
			pastLearnings = re.buildLearningsContext(successfulPlans, failedPlans)
		}
	}

	// Build replanning prompt
	prompt := re.buildReplanPrompt(originalPlan, failureContext, pastLearnings, tools)

	// Call LLM for replanning
	request := llm.ChatRequest{
		Provider: re.provider,
		Model:    re.model,
		Messages: []llm.ChatMessage{
			{
				Role:    "system",
				Content: "You are an expert at revising plans based on failures and learnings.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}
	temp := 0.7 // Higher temperature for creative replanning
	request.Temperature = &temp

	response, err := re.llmManager.Chat(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("LLM replanning call failed: %w", err)
	}

	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no replanning response from LLM")
	}

	// Parse new plan
	newPlan, err := re.parsePlan(response.Choices[0].Message.Content, agentID, originalPlan.Goal)
	if err != nil {
		return nil, fmt.Errorf("failed to parse replan: %w", err)
	}

	// Set replan metadata
	newPlan.ParentPlanID = originalPlan.ID
	newPlan.Version = originalPlan.Version + 1
	newPlan.Metadata["replan_reason"] = failureContext
	newPlan.Metadata["replan_timestamp"] = time.Now()

	// Store replan in memory if available
	if re.memoryManager != nil {
		_ = StorePlan(ctx, re.memoryManager, newPlan)
	}

	if re.metrics != nil {
		re.metrics.Counter("forge.ai.sdk.replan.replans_created").Inc()
		re.metrics.Histogram("forge.ai.sdk.replan.version").Observe(float64(newPlan.Version))
	}

	return newPlan, nil
}

// learnFromFailure tracks failure patterns for future improvement.
func (re *ReplanEngine) learnFromFailure(ctx context.Context, plan *Plan, failureContext string) {
	// Extract failure pattern
	pattern := re.extractFailurePattern(plan, failureContext)
	if pattern != "" {
		re.failurePatterns[pattern]++

		if re.logger != nil {
			re.logger.Debug("Learned failure pattern",
				logger.String("pattern", pattern),
				logger.Int("occurrences", re.failurePatterns[pattern]),
			)
		}
	}

	// Store in memory for long-term learning
	if re.memoryManager != nil {
		learningContent := fmt.Sprintf("Plan failure pattern: %s | Context: %s", pattern, failureContext)
		metadata := map[string]any{
			"type":            "failure_pattern",
			"pattern":         pattern,
			"plan_id":         plan.ID,
			"goal":            plan.Goal,
			"failure_context": failureContext,
		}
		_, _ = re.memoryManager.Store(ctx, learningContent, metadata, 0.8)
	}
}

// extractFailurePattern identifies the type of failure.
func (re *ReplanEngine) extractFailurePattern(plan *Plan, failureContext string) string {
	lower := strings.ToLower(failureContext)

	// Common failure patterns
	if strings.Contains(lower, "dependency") || strings.Contains(lower, "circular") {
		return "dependency_issue"
	}
	if strings.Contains(lower, "tool") || strings.Contains(lower, "missing") {
		return "tool_availability"
	}
	if strings.Contains(lower, "timeout") || strings.Contains(lower, "time") {
		return "timeout"
	}
	if strings.Contains(lower, "incomplete") || strings.Contains(lower, "missing step") {
		return "incomplete_plan"
	}
	if strings.Contains(lower, "invalid") || strings.Contains(lower, "incorrect") {
		return "invalid_logic"
	}

	// Check step statuses
	failedCount := 0
	for _, step := range plan.Steps {
		if step.Status == PlanStepStatusFailed {
			failedCount++
		}
	}

	if failedCount > len(plan.Steps)/2 {
		return "multiple_step_failures"
	}

	return "general_failure"
}

// buildLearningsContext creates a context string from past plans.
func (re *ReplanEngine) buildLearningsContext(successful, failed []*Plan) string {
	var sb strings.Builder

	if len(successful) > 0 {
		sb.WriteString("\n## Successful Past Plans (for reference):\n\n")
		for i, plan := range successful {
			sb.WriteString(fmt.Sprintf("%d. Goal: %s\n", i+1, plan.Goal))
			sb.WriteString("   Key steps:\n")
			for j, step := range plan.Steps {
				if j < 3 { // Show first 3 steps
					sb.WriteString(fmt.Sprintf("   - %s\n", step.Description))
				}
			}
			sb.WriteString("\n")
		}
	}

	if len(failed) > 0 {
		sb.WriteString("\n## Failed Past Plans (to avoid):\n\n")
		for i, plan := range failed {
			sb.WriteString(fmt.Sprintf("%d. Goal: %s\n", i+1, plan.Goal))
			sb.WriteString("   What went wrong:\n")
			for _, step := range plan.Steps {
				if step.Status == PlanStepStatusFailed && step.Error != "" {
					sb.WriteString(fmt.Sprintf("   - %s: %s\n", step.Description, step.Error))
				}
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// buildReplanPrompt constructs the prompt for replanning.
func (re *ReplanEngine) buildReplanPrompt(originalPlan *Plan, failureContext, pastLearnings string, tools []Tool) string {
	var sb strings.Builder

	sb.WriteString(re.promptTemplate)
	sb.WriteString("\n\n## Original Plan\n\n")
	sb.WriteString(fmt.Sprintf("**Goal:** %s\n", originalPlan.Goal))
	sb.WriteString(fmt.Sprintf("**Version:** %d\n", originalPlan.Version))
	sb.WriteString(fmt.Sprintf("**Status:** %s\n\n", originalPlan.Status))

	sb.WriteString("### Steps Attempted:\n\n")
	for i, step := range originalPlan.Steps {
		status := "✓"
		switch step.Status {
		case PlanStepStatusFailed:
			status = "✗"
		case PlanStepStatusPending:
			status = "○"
		}
		sb.WriteString(fmt.Sprintf("%s %d. %s\n", status, i+1, step.Description))
		if step.Status == PlanStepStatusCompleted && step.Result != nil {
			sb.WriteString(fmt.Sprintf("   Result: %v\n", step.Result))
		}
		if step.Error != "" {
			sb.WriteString(fmt.Sprintf("   Error: %s\n", step.Error))
		}
	}

	sb.WriteString(fmt.Sprintf("\n## Failure Context\n\n%s\n", failureContext))

	if pastLearnings != "" {
		sb.WriteString(pastLearnings)
	}

	// Add common failure patterns if learned
	if len(re.failurePatterns) > 0 {
		sb.WriteString("\n## Known Failure Patterns (to avoid):\n\n")
		for pattern, count := range re.failurePatterns {
			if count > 1 {
				sb.WriteString(fmt.Sprintf("- %s (occurred %d times)\n", pattern, count))
			}
		}
	}

	sb.WriteString("\n## Available Tools:\n\n")
	for _, tool := range tools {
		sb.WriteString(fmt.Sprintf("- **%s**: %s\n", tool.Name, tool.Description))
	}

	sb.WriteString("\n## Instructions\n\n")
	sb.WriteString("Create a revised plan that:\n")
	sb.WriteString("1. Addresses the failure causes\n")
	sb.WriteString("2. Preserves successful steps where possible\n")
	sb.WriteString("3. Adds error handling and validation\n")
	sb.WriteString("4. Considers alternative approaches\n")
	sb.WriteString("5. Learns from past failures\n\n")

	sb.WriteString("Return the plan in JSON format:\n")
	sb.WriteString("```json\n")
	sb.WriteString("{\n")
	sb.WriteString("  \"steps\": [\n")
	sb.WriteString("    {\n")
	sb.WriteString("      \"description\": \"...\",\n")
	sb.WriteString("      \"tools\": [\"tool1\", \"tool2\"],\n")
	sb.WriteString("      \"dependencies\": []\n")
	sb.WriteString("    }\n")
	sb.WriteString("  ]\n")
	sb.WriteString("}\n")
	sb.WriteString("```\n")

	return sb.String()
}

// parsePlan parses a plan from LLM response.
func (re *ReplanEngine) parsePlan(content string, agentID string, goal string) (*Plan, error) {
	// Try to extract JSON from response
	jsonStart := strings.Index(content, "{")
	jsonEnd := strings.LastIndex(content, "}")

	if jsonStart == -1 || jsonEnd == -1 {
		return nil, fmt.Errorf("no JSON found in replanning response")
	}

	jsonStr := content[jsonStart : jsonEnd+1]

	var planData struct {
		Steps []struct {
			Description  string   `json:"description"`
			Tools        []string `json:"tools"`
			Dependencies []string `json:"dependencies"`
		} `json:"steps"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &planData); err != nil {
		return nil, fmt.Errorf("failed to parse plan JSON: %w", err)
	}

	// Create plan
	plan := &Plan{
		ID:        generatePlanID(),
		AgentID:   agentID,
		Goal:      goal,
		Status:    PlanStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Version:   1,
		Metadata:  make(map[string]any),
	}

	// Create steps
	plan.Steps = make([]PlanStep, len(planData.Steps))
	for i, stepData := range planData.Steps {
		plan.Steps[i] = PlanStep{
			ID:           generateStepID(plan.ID, i),
			Index:        i,
			Description:  stepData.Description,
			ToolsNeeded:  stepData.Tools,
			Dependencies: stepData.Dependencies,
			Status:       PlanStepStatusPending,
			MaxRetries:   3,
			Metadata:     make(map[string]any),
		}
	}

	return plan, nil
}

// GetFailurePatterns returns learned failure patterns.
func (re *ReplanEngine) GetFailurePatterns() map[string]int {
	return re.failurePatterns
}

// Default configuration

func getDefaultReplanTriggers() []ReplanTrigger {
	return []ReplanTrigger{
		{
			Name:        "InvalidPlan",
			Description: "Plan is marked as invalid by reflection",
			Priority:    100,
			Condition: func(plan *Plan, reflection *ReflectionResult) bool {
				return reflection != nil && reflection.Quality == "invalid"
			},
		},
		{
			Name:        "LowQualityScore",
			Description: "Plan quality score is below threshold",
			Priority:    90,
			Condition: func(plan *Plan, reflection *ReflectionResult) bool {
				return reflection != nil && reflection.Score < 0.5
			},
		},
		{
			Name:        "MultipleStepFailures",
			Description: "More than half of steps have failed",
			Priority:    80,
			Condition: func(plan *Plan, reflection *ReflectionResult) bool {
				if plan == nil {
					return false
				}
				failedCount := 0
				for _, step := range plan.Steps {
					if step.Status == PlanStepStatusFailed {
						failedCount++
					}
				}
				return failedCount > len(plan.Steps)/2
			},
		},
		{
			Name:        "CriticalStepFailure",
			Description: "A critical step with no alternatives has failed",
			Priority:    85,
			Condition: func(plan *Plan, reflection *ReflectionResult) bool {
				if plan == nil {
					return false
				}
				for _, step := range plan.Steps {
					if step.Status == PlanStepStatusFailed && len(step.Dependencies) == 0 {
						// First step or independent step failed
						return true
					}
				}
				return false
			},
		},
		{
			Name:        "ExplicitReplanFlag",
			Description: "Reflection explicitly recommends replanning",
			Priority:    95,
			Condition: func(plan *Plan, reflection *ReflectionResult) bool {
				return reflection != nil && reflection.ShouldReplan
			},
		},
	}
}

func getDefaultReplanPrompt() string {
	return `You are an expert at revising plans based on failures and learnings.
Your goal is to create an improved plan that addresses the issues in the original plan
while preserving what worked and learning from past experiences.

Focus on:
- Root cause analysis of failures
- Alternative approaches and strategies
- Better error handling and validation
- Clearer step definitions and dependencies
- Learning from successful patterns`
}
