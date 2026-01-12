package sdk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/xraph/ai-sdk/llm"
	logger "github.com/xraph/go-utils/log"
	"github.com/xraph/go-utils/metrics"
)

// PlanExecuteStrategy implements the Plan-Execute pattern.
// It decomposes tasks into plans, executes them, and verifies results.
type PlanExecuteStrategy struct {
	// LLM managers for different phases
	planner  LLMManager
	executor LLMManager
	verifier LLMManager

	// Storage
	planStore PlanStore

	// Configuration
	allowReplanning   bool
	verifySteps       bool
	maxReplanAttempts int
	timeout           time.Duration

	// Dependencies
	memoryManager *MemoryManager
	logger        logger.Logger
	metrics       metrics.Metrics

	// State
	currentPlan *Plan
	planHistory []*Plan
}

// PlanExecuteStrategyConfig configures the Plan-Execute strategy.
type PlanExecuteStrategyConfig struct {
	Planner           LLMManager
	Executor          LLMManager
	Verifier          LLMManager
	PlanStore         PlanStore
	MemoryManager     *MemoryManager
	AllowReplanning   bool
	VerifySteps       bool
	MaxReplanAttempts int
	Timeout           time.Duration
}

// NewPlanExecuteStrategy creates a new Plan-Execute strategy.
func NewPlanExecuteStrategy(logger logger.Logger, metrics metrics.Metrics, config *PlanExecuteStrategyConfig) *PlanExecuteStrategy {
	if config == nil {
		config = &PlanExecuteStrategyConfig{}
	}

	// Set defaults
	if config.MaxReplanAttempts == 0 {
		config.MaxReplanAttempts = 3
	}
	if config.Timeout == 0 {
		config.Timeout = 10 * time.Minute
	}

	// Use same LLM for all phases if not specified
	if config.Executor == nil {
		config.Executor = config.Planner
	}
	if config.Verifier == nil {
		config.Verifier = config.Planner
	}

	return &PlanExecuteStrategy{
		planner:           config.Planner,
		executor:          config.Executor,
		verifier:          config.Verifier,
		planStore:         config.PlanStore,
		memoryManager:     config.MemoryManager,
		allowReplanning:   config.AllowReplanning,
		verifySteps:       config.VerifySteps,
		maxReplanAttempts: config.MaxReplanAttempts,
		timeout:           config.Timeout,
		logger:            logger,
		metrics:           metrics,
		planHistory:       make([]*Plan, 0),
	}
}

// Execute runs the Plan-Execute strategy.
func (s *PlanExecuteStrategy) Execute(ctx context.Context, agent *EnhancedAgent, input string) (*AgentExecution, error) {
	startTime := time.Now()

	// Create execution context with timeout
	execCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	execution := &AgentExecution{
		ID:        generateExecutionID(),
		AgentID:   agent.ID,
		StartTime: startTime,
		Status:    ExecutionStatusRunning,
		Steps:     make([]*AgentStep, 0),
		Metadata:  make(map[string]any),
	}

	if s.logger != nil {
		s.logger.Info("Starting Plan-Execute strategy",
			logger.String("agent_id", agent.ID),
			logger.String("execution_id", execution.ID),
		)
	}

	// Phase 1: Create initial plan
	plan, err := s.createPlan(execCtx, agent, input)
	if err != nil {
		execution.Status = ExecutionStatusFailed
		execution.Error = fmt.Sprintf("planning failed: %s", err.Error())
		return execution, fmt.Errorf("failed to create plan: %w", err)
	}

	s.currentPlan = plan
	s.planHistory = append(s.planHistory, plan)
	execution.Metadata["plan_id"] = plan.ID
	execution.Metadata["plan"] = plan

	// Save plan if store available
	if s.planStore != nil {
		if err := s.planStore.Save(execCtx, plan); err != nil {
			if s.logger != nil {
				s.logger.Warn("Failed to save plan", logger.String("error", err.Error()))
			}
		}
	}

	// Phase 2: Execute plan with optional replanning
	replanAttempts := 0
	for replanAttempts <= s.maxReplanAttempts {
		// Check for cancellation
		select {
		case <-execCtx.Done():
			execution.Status = ExecutionStatusCancelled
			execution.Error = "execution timeout"
			return execution, execCtx.Err()
		default:
		}

		// Execute the plan
		err := s.executePlan(execCtx, agent, plan, execution)
		if err == nil {
			// Plan completed successfully
			break
		}

		// Check if we should replan
		if !s.allowReplanning || replanAttempts >= s.maxReplanAttempts {
			execution.Status = ExecutionStatusFailed
			execution.Error = fmt.Sprintf("plan execution failed: %s", err.Error())
			return execution, err
		}

		// Replan
		if s.logger != nil {
			s.logger.Info("Replanning due to failure",
				logger.String("error", err.Error()),
				logger.Int("attempt", replanAttempts+1),
			)
		}

		newPlan, replanErr := s.replan(execCtx, agent, plan, err)
		if replanErr != nil {
			execution.Status = ExecutionStatusFailed
			execution.Error = fmt.Sprintf("replanning failed: %s", replanErr.Error())
			return execution, replanErr
		}

		plan = newPlan
		s.currentPlan = plan
		s.planHistory = append(s.planHistory, plan)
		replanAttempts++

		if s.planStore != nil {
			_ = s.planStore.Save(execCtx, plan)
		}
	}

	// Phase 3: Final verification
	if s.verifySteps {
		verification, err := s.verifyPlan(execCtx, agent, plan)
		if err == nil {
			execution.Metadata["verification"] = verification
		}
	}

	// Build final output
	execution.Status = ExecutionStatusCompleted
	execution.FinalOutput = s.buildFinalOutput(plan)
	execution.EndTime = time.Now()
	execution.Metadata["replan_attempts"] = replanAttempts
	execution.Metadata["plan_history"] = s.planHistory

	if s.logger != nil {
		s.logger.Info("Plan-Execute strategy completed",
			logger.String("execution_id", execution.ID),
			logger.Int("steps", len(plan.Steps)),
			logger.Int("replans", replanAttempts),
		)
	}

	if s.metrics != nil {
		s.metrics.Counter("forge.ai.sdk.plan_execute.executions",
			metrics.WithLabel("status", string(execution.Status)),
		).Inc()
		s.metrics.Histogram("forge.ai.sdk.plan_execute.steps").Observe(float64(len(plan.Steps)))
		s.metrics.Histogram("forge.ai.sdk.plan_execute.replans").Observe(float64(replanAttempts))
	}

	return execution, nil
}

// createPlan generates a plan for the given task.
func (s *PlanExecuteStrategy) createPlan(ctx context.Context, agent *EnhancedAgent, task string) (*Plan, error) {
	// Build planning prompt
	prompt := s.buildPlanningPrompt(task, agent.tools)

	// Call LLM for planning
	request := llm.ChatRequest{
		Provider: agent.Provider,
		Model:    agent.Model,
		Messages: []llm.ChatMessage{
			{
				Role:    "system",
				Content: "You are an expert at breaking down complex tasks into clear, executable steps.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	response, err := s.planner.Chat(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("planning LLM call failed: %w", err)
	}

	if len(response.Choices) == 0 {
		return nil, errors.New("no planning response from LLM")
	}

	// Parse plan from response
	plan, err := s.parsePlan(response.Choices[0].Message.Content, agent.ID, task)
	if err != nil {
		return nil, fmt.Errorf("failed to parse plan: %w", err)
	}

	if s.logger != nil {
		s.logger.Info("Plan created",
			logger.String("plan_id", plan.ID),
			logger.Int("steps", len(plan.Steps)),
		)
	}

	return plan, nil
}

// executePlan executes all steps in the plan.
func (s *PlanExecuteStrategy) executePlan(ctx context.Context, agent *EnhancedAgent, plan *Plan, execution *AgentExecution) error {
	plan.Status = PlanStatusInProgress

	// Track completed steps for dependency resolution
	completedSteps := make(map[string]bool)

	// Execute steps respecting dependencies
	for {
		// Get pending steps that can execute
		pendingSteps := plan.GetPendingSteps()
		if len(pendingSteps) == 0 {
			// Check if all steps are completed
			allCompleted := true
			for _, step := range plan.Steps {
				if step.Status != PlanStepStatusCompleted {
					allCompleted = false
					break
				}
			}

			if allCompleted {
				plan.Status = PlanStatusCompleted
				return nil
			}

			// Some steps failed or skipped
			plan.Status = PlanStatusFailed
			return errors.New("plan execution incomplete")
		}

		// Execute pending steps in parallel
		var wg sync.WaitGroup
		errors := make(chan error, len(pendingSteps))

		for _, step := range pendingSteps {
			wg.Add(1)

			go func(st PlanStep) {
				defer wg.Done()

				// Find the actual step in the plan (not the copy)
				var stepPtr *PlanStep
				for i := range plan.Steps {
					if plan.Steps[i].ID == st.ID {
						stepPtr = &plan.Steps[i]
						break
					}
				}

				if stepPtr == nil {
					errors <- fmt.Errorf("step %s not found", st.ID)
					return
				}

				err := s.executeStep(ctx, agent, stepPtr, plan, execution)
				if err != nil {
					errors <- fmt.Errorf("step %s failed: %w", st.ID, err)
					return
				}

				completedSteps[st.ID] = true
			}(step)
		}

		wg.Wait()
		close(errors)

		// Check for errors
		for err := range errors {
			if err != nil {
				return err
			}
		}
	}
}

// executeStep executes a single plan step.
func (s *PlanExecuteStrategy) executeStep(
	ctx context.Context,
	agent *EnhancedAgent,
	step *PlanStep,
	plan *Plan,
	execution *AgentExecution,
) error {
	step.Status = PlanStepStatusRunning
	step.StartTime = time.Now()

	if s.logger != nil {
		s.logger.Info("Executing plan step",
			logger.String("step_id", step.ID),
			logger.String("description", step.Description),
		)
	}

	// Build step execution prompt
	prompt := s.buildStepPrompt(step, plan)

	// Call LLM to execute step
	request := llm.ChatRequest{
		Provider: agent.Provider,
		Model:    agent.Model,
		Messages: []llm.ChatMessage{
			{
				Role:    "system",
				Content: agent.systemPrompt,
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	// Add tools if step needs them
	if len(step.ToolsNeeded) > 0 {
		llmTools := make([]llm.Tool, 0)
		for _, toolName := range step.ToolsNeeded {
			for _, tool := range agent.tools {
				if tool.Name == toolName {
					llmTools = append(llmTools, llm.Tool{
						Type: "function",
						Function: &llm.FunctionDefinition{
							Name:        tool.Name,
							Description: tool.Description,
							Parameters:  tool.Parameters,
						},
					})
					break
				}
			}
		}
		request.Tools = llmTools
	}

	response, err := s.executor.Chat(ctx, request)
	if err != nil {
		step.MarkFailed(err)
		return err
	}

	if len(response.Choices) == 0 {
		err := errors.New("no response from executor")
		step.MarkFailed(err)
		return err
	}

	// Handle tool calls if any
	if len(response.Choices[0].Message.ToolCalls) > 0 {
		result, err := s.executeToolCalls(ctx, agent, response.Choices[0].Message.ToolCalls)
		if err != nil {
			step.MarkFailed(err)
			return err
		}
		step.MarkCompleted(result)
	} else {
		step.MarkCompleted(response.Choices[0].Message.Content)
	}

	// Create agent step for tracking
	agentStep := &AgentStep{
		Index:       step.Index,
		ID:          step.ID,
		AgentID:     agent.ID,
		ExecutionID: execution.ID,
		Input:       step.Description,
		Output:      fmt.Sprintf("%v", step.Result),
		StartTime:   step.StartTime,
		EndTime:     step.EndTime,
		Duration:    step.EndTime.Sub(step.StartTime),
		State:       StepStateCompleted,
		Metadata:    step.Metadata,
	}
	execution.Steps = append(execution.Steps, agentStep)

	// Verify step if enabled
	if s.verifySteps {
		verification, err := s.verifyStep(ctx, agent, step)
		if err == nil {
			step.Verification = verification

			if !verification.IsValid && s.allowReplanning {
				if s.logger != nil {
					s.logger.Warn("Step verification failed",
						logger.String("step_id", step.ID),
						logger.Float64("score", verification.Score),
					)
				}
			}
		}
	}

	if s.metrics != nil {
		s.metrics.Counter("forge.ai.sdk.plan_execute.steps_executed").Inc()
		s.metrics.Histogram("forge.ai.sdk.plan_execute.step_duration").Observe(step.EndTime.Sub(step.StartTime).Seconds())
	}

	return nil
}

// executeToolCalls executes tool calls from LLM response.
func (s *PlanExecuteStrategy) executeToolCalls(ctx context.Context, agent *EnhancedAgent, toolCalls []llm.ToolCall) (any, error) {
	results := make([]any, len(toolCalls))

	for i, tc := range toolCalls {
		// Find tool
		var tool *Tool
		for j := range agent.tools {
			if agent.tools[j].Name == tc.Function.Name {
				tool = &agent.tools[j]
				break
			}
		}

		if tool == nil {
			return nil, fmt.Errorf("tool not found: %s", tc.Function.Name)
		}

		// Parse arguments
		var args map[string]any
		if tc.Function.Arguments != "" {
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
				return nil, fmt.Errorf("failed to parse tool arguments: %w", err)
			}
		}

		// Execute tool
		result, err := tool.Handler(ctx, args)
		if err != nil {
			return nil, fmt.Errorf("tool %s failed: %w", tc.Function.Name, err)
		}

		results[i] = result
	}

	if len(results) == 1 {
		return results[0], nil
	}
	return results, nil
}

// verifyStep validates a step's output.
func (s *PlanExecuteStrategy) verifyStep(ctx context.Context, agent *EnhancedAgent, step *PlanStep) (*VerificationResult, error) {
	prompt := fmt.Sprintf(`Verify the quality of this step execution:

Step: %s
Result: %v

Evaluate:
1. Does the result accomplish the step's goal?
2. Is the output complete and correct?
3. Are there any issues or concerns?

Provide a score (0-1) and explain your assessment.`, step.Description, step.Result)

	request := llm.ChatRequest{
		Provider: agent.Provider,
		Model:    agent.Model,
		Messages: []llm.ChatMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	response, err := s.verifier.Chat(ctx, request)
	if err != nil {
		return nil, err
	}

	if len(response.Choices) == 0 {
		return nil, errors.New("no verification response")
	}

	// Parse verification
	content := response.Choices[0].Message.Content
	verification := &VerificationResult{
		Reasoning: content,
		Timestamp: time.Now(),
	}

	// Extract score and validity
	verification.Score = s.extractVerificationScore(content)
	verification.IsValid = verification.Score >= 0.7
	verification.Issues = s.extractVerificationIssues(content)
	verification.Suggestions = s.extractVerificationSuggestions(content)

	return verification, nil
}

// verifyPlan validates the entire plan execution.
func (s *PlanExecuteStrategy) verifyPlan(ctx context.Context, agent *EnhancedAgent, plan *Plan) (*VerificationResult, error) {
	// Build summary of plan execution
	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("Goal: %s\n\n", plan.Goal))
	summary.WriteString("Steps executed:\n")

	for _, step := range plan.Steps {
		summary.WriteString(fmt.Sprintf("- %s: %v\n", step.Description, step.Result))
	}

	prompt := fmt.Sprintf(`Verify if this plan successfully accomplished its goal:

%s

Evaluate:
1. Were all necessary steps completed?
2. Does the overall result satisfy the goal?
3. Are there any gaps or issues?

Provide an overall assessment and score (0-1).`, summary.String())

	request := llm.ChatRequest{
		Provider: agent.Provider,
		Model:    agent.Model,
		Messages: []llm.ChatMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	response, err := s.verifier.Chat(ctx, request)
	if err != nil {
		return nil, err
	}

	if len(response.Choices) == 0 {
		return nil, errors.New("no verification response")
	}

	content := response.Choices[0].Message.Content
	verification := &VerificationResult{
		Reasoning: content,
		Timestamp: time.Now(),
	}

	verification.Score = s.extractVerificationScore(content)
	verification.IsValid = verification.Score >= 0.7

	return verification, nil
}

// replan creates a new plan based on previous failure.
func (s *PlanExecuteStrategy) replan(ctx context.Context, agent *EnhancedAgent, failedPlan *Plan, failureReason error) (*Plan, error) {
	if s.logger != nil {
		s.logger.Info("Creating replan",
			logger.String("original_plan", failedPlan.ID),
			logger.String("reason", failureReason.Error()),
		)
	}

	// Build replanning prompt with context
	prompt := s.buildReplanningPrompt(failedPlan, failureReason)

	request := llm.ChatRequest{
		Provider: agent.Provider,
		Model:    agent.Model,
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

	response, err := s.planner.Chat(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("replanning LLM call failed: %w", err)
	}

	if len(response.Choices) == 0 {
		return nil, errors.New("no replanning response")
	}

	// Parse new plan
	newPlan, err := s.parsePlan(response.Choices[0].Message.Content, agent.ID, failedPlan.Goal)
	if err != nil {
		return nil, fmt.Errorf("failed to parse replan: %w", err)
	}

	newPlan.ParentPlanID = failedPlan.ID
	newPlan.Version = failedPlan.Version + 1

	return newPlan, nil
}

// Helper methods

func (s *PlanExecuteStrategy) buildPlanningPrompt(task string, tools []Tool) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Task: %s\n\n", task))
	sb.WriteString("Available tools:\n")

	for _, tool := range tools {
		sb.WriteString(fmt.Sprintf("- %s: %s\n", tool.Name, tool.Description))
	}

	sb.WriteString(`
Create a detailed plan to accomplish this task:
1. Break down into clear, sequential steps
2. Identify which tools each step needs
3. Specify dependencies between steps
4. Consider error handling

Return the plan in JSON format:
{
  "steps": [
    {
      "description": "...",
      "tools": ["tool1", "tool2"],
      "dependencies": []
    }
  ]
}`)

	return sb.String()
}

func (s *PlanExecuteStrategy) buildStepPrompt(step *PlanStep, plan *Plan) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Execute this step: %s\n\n", step.Description))
	sb.WriteString(fmt.Sprintf("Overall goal: %s\n\n", plan.Goal))

	// Add context from completed steps
	sb.WriteString("Previous results:\n")
	for _, prevStep := range plan.Steps {
		if prevStep.Status == PlanStepStatusCompleted && prevStep.Index < step.Index {
			sb.WriteString(fmt.Sprintf("- %s: %v\n", prevStep.Description, prevStep.Result))
		}
	}

	sb.WriteString("\nProvide a clear, complete result for this step.")

	return sb.String()
}

func (s *PlanExecuteStrategy) buildReplanningPrompt(failedPlan *Plan, failureReason error) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Original goal: %s\n\n", failedPlan.Goal))
	sb.WriteString(fmt.Sprintf("Failure reason: %s\n\n", failureReason.Error()))
	sb.WriteString("Steps attempted:\n")

	for _, step := range failedPlan.Steps {
		status := "✓"
		if step.Status == PlanStepStatusFailed {
			status = "✗"
		}
		sb.WriteString(fmt.Sprintf("%s %s\n", status, step.Description))
		if step.Error != "" {
			sb.WriteString(fmt.Sprintf("  Error: %s\n", step.Error))
		}
	}

	sb.WriteString("\nCreate a revised plan that addresses the failure. Use the same JSON format.")

	return sb.String()
}

func (s *PlanExecuteStrategy) parsePlan(content string, agentID string, goal string) (*Plan, error) {
	// Try to extract JSON from response
	jsonStart := strings.Index(content, "{")
	jsonEnd := strings.LastIndex(content, "}")

	if jsonStart == -1 || jsonEnd == -1 {
		return nil, errors.New("no JSON found in planning response")
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

func (s *PlanExecuteStrategy) buildFinalOutput(plan *Plan) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Goal: %s\n\n", plan.Goal))
	sb.WriteString("Results:\n")

	for _, step := range plan.Steps {
		if step.Status == PlanStepStatusCompleted {
			sb.WriteString(fmt.Sprintf("- %s: %v\n", step.Description, step.Result))
		}
	}

	return sb.String()
}

func (s *PlanExecuteStrategy) extractVerificationScore(content string) float64 {
	// Simple extraction - look for score patterns
	var score float64
	if strings.Contains(content, "score:") {
		_, _ = fmt.Sscanf(content, "score: %f", &score) // Best effort parsing
	}

	// Default based on keywords
	if score == 0 {
		lower := strings.ToLower(content)
		if strings.Contains(lower, "excellent") || strings.Contains(lower, "perfect") {
			score = 0.9
		} else if strings.Contains(lower, "good") || strings.Contains(lower, "satisfactory") {
			score = 0.8
		} else if strings.Contains(lower, "adequate") || strings.Contains(lower, "acceptable") {
			score = 0.7
		} else if strings.Contains(lower, "poor") || strings.Contains(lower, "inadequate") {
			score = 0.5
		} else {
			score = 0.75 // Default middle score
		}
	}

	return score
}

func (s *PlanExecuteStrategy) extractVerificationIssues(content string) []string {
	var issues []string
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(line), "issue:") {
			issues = append(issues, strings.TrimSpace(strings.TrimPrefix(strings.ToLower(line), "issue:")))
		}
	}

	return issues
}

func (s *PlanExecuteStrategy) extractVerificationSuggestions(content string) []string {
	var suggestions []string
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(line), "suggestion:") {
			suggestions = append(suggestions, strings.TrimSpace(strings.TrimPrefix(strings.ToLower(line), "suggestion:")))
		}
	}

	return suggestions
}

// Name returns the strategy name.
func (s *PlanExecuteStrategy) Name() string {
	return "Plan-Execute"
}

// SupportsReplanning indicates if this strategy can replan.
func (s *PlanExecuteStrategy) SupportsReplanning() bool {
	return s.allowReplanning
}

// GetCurrentPlan returns the current plan being executed.
func (s *PlanExecuteStrategy) GetCurrentPlan() *Plan {
	return s.currentPlan
}

// GetPlanHistory returns all plans created during execution.
func (s *PlanExecuteStrategy) GetPlanHistory() []*Plan {
	return s.planHistory
}
