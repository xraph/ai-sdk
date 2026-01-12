package sdk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/xraph/ai-sdk/llm"
	logger "github.com/xraph/go-utils/log"
	"github.com/xraph/go-utils/metrics"
)

// ReactStrategy implements the ReAct (Reasoning + Acting) pattern.
// It alternates between reasoning (thinking) and acting (tool use) until reaching a final answer.
type ReactStrategy struct {
	// Configuration
	maxIterations       int
	reflectionInterval  int // Reflect every N steps
	confidenceThreshold float64
	timeout             time.Duration

	// Dependencies
	memoryManager *MemoryManager
	logger        logger.Logger
	metrics       metrics.Metrics

	// State
	traces      []ReasoningTrace
	reflections []ReflectionResult

	// Prompt templates
	reasoningPrompt  string
	reflectionPrompt string
}

// ReactStrategyConfig configures the ReAct strategy.
type ReactStrategyConfig struct {
	MaxIterations       int
	ReflectionInterval  int
	ConfidenceThreshold float64
	Timeout             time.Duration
	MemoryManager       *MemoryManager
	ReasoningPrompt     string
	ReflectionPrompt    string
}

// NewReactStrategy creates a new ReAct strategy.
func NewReactStrategy(logger logger.Logger, metrics metrics.Metrics, config *ReactStrategyConfig) *ReactStrategy {
	if config == nil {
		config = &ReactStrategyConfig{}
	}

	// Set defaults
	if config.MaxIterations == 0 {
		config.MaxIterations = 10
	}
	if config.ReflectionInterval == 0 {
		config.ReflectionInterval = 3
	}
	if config.ConfidenceThreshold == 0 {
		config.ConfidenceThreshold = 0.7
	}
	if config.Timeout == 0 {
		config.Timeout = 5 * time.Minute
	}
	if config.ReasoningPrompt == "" {
		config.ReasoningPrompt = defaultReasoningPrompt
	}
	if config.ReflectionPrompt == "" {
		config.ReflectionPrompt = defaultReflectionPrompt
	}

	return &ReactStrategy{
		maxIterations:       config.MaxIterations,
		reflectionInterval:  config.ReflectionInterval,
		confidenceThreshold: config.ConfidenceThreshold,
		timeout:             config.Timeout,
		memoryManager:       config.MemoryManager,
		logger:              logger,
		metrics:             metrics,
		traces:              make([]ReasoningTrace, 0),
		reflections:         make([]ReflectionResult, 0),
		reasoningPrompt:     config.ReasoningPrompt,
		reflectionPrompt:    config.ReflectionPrompt,
	}
}

// Execute runs the ReAct reasoning loop.
func (s *ReactStrategy) Execute(ctx context.Context, agent *Agent, input string) (*AgentExecution, error) {
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

	// Clear previous traces
	s.traces = make([]ReasoningTrace, 0)
	s.reflections = make([]ReflectionResult, 0)

	if s.logger != nil {
		s.logger.Info("Starting ReAct strategy execution",
			logger.String("agent_id", agent.ID),
			logger.String("execution_id", execution.ID),
		)
	}

	// Recall relevant memories if available
	var memoryContext string
	if s.memoryManager != nil {
		memories, err := s.memoryManager.Recall(execCtx, input, MemoryTierLongTerm, 5)
		if err == nil && len(memories) > 0 {
			memoryContext = s.formatMemories(memories)
		}
	}

	// Main ReAct loop
	currentInput := input
	for iteration := 0; iteration < s.maxIterations; iteration++ {
		// Check for cancellation
		select {
		case <-execCtx.Done():
			execution.Status = ExecutionStatusCancelled
			execution.Error = "execution timeout"
			return execution, execCtx.Err()
		default:
		}

		// Step 1: Generate thought/reasoning
		trace, err := s.think(execCtx, agent, currentInput, memoryContext, iteration)
		if err != nil {
			execution.Status = ExecutionStatusFailed
			execution.Error = err.Error()
			return execution, fmt.Errorf("thinking failed at iteration %d: %w", iteration, err)
		}

		s.traces = append(s.traces, trace)

		// Create agent step for this iteration
		step := &AgentStep{
			Index:       iteration,
			ID:          fmt.Sprintf("%s_step_%d", execution.ID, iteration),
			AgentID:     agent.ID,
			ExecutionID: execution.ID,
			Input:       currentInput,
			StartTime:   time.Now(),
			State:       StepStateRunning,
			Metadata:    make(map[string]any),
		}
		step.Metadata["reasoning_trace"] = trace

		// Check if this is a final answer
		if s.isFinalAnswer(trace) {
			step.Output = trace.Observation
			step.State = StepStateCompleted
			step.EndTime = time.Now()
			step.Duration = step.EndTime.Sub(step.StartTime)
			execution.Steps = append(execution.Steps, step)

			execution.Status = ExecutionStatusCompleted
			execution.FinalOutput = trace.Observation
			execution.EndTime = time.Now()

			if s.logger != nil {
				s.logger.Info("ReAct strategy completed with final answer",
					logger.String("execution_id", execution.ID),
					logger.Int("iterations", iteration+1),
				)
			}

			return execution, nil
		}

		// Step 2: Execute action if specified
		if trace.Action != "" {
			observation, err := s.act(execCtx, agent, trace)
			if err != nil {
				step.Error = err.Error()
				step.State = StepStateFailed
				trace.Observation = fmt.Sprintf("Error: %s", err.Error())
				trace.Confidence = 0.3 // Low confidence on error
			} else {
				trace.Observation = observation
				step.ToolResults = []StepToolResult{
					{
						Name:     trace.Action,
						Result:   observation,
						Duration: time.Since(step.StartTime),
					},
				}
			}

			// Update trace with observation
			s.traces[len(s.traces)-1] = trace
		}

		step.Output = trace.Observation
		step.State = StepStateCompleted
		step.EndTime = time.Now()
		step.Duration = step.EndTime.Sub(step.StartTime)
		execution.Steps = append(execution.Steps, step)

		// Step 3: Periodic self-reflection
		if s.reflectionInterval > 0 && (iteration+1)%s.reflectionInterval == 0 {
			reflection, err := s.reflect(execCtx, agent, s.traces)
			if err == nil {
				s.reflections = append(s.reflections, reflection)

				// Check if we should stop based on reflection
				if reflection.ShouldReplan || reflection.Score < 0.5 {
					if s.logger != nil {
						s.logger.Warn("Reflection suggests stopping",
							logger.String("quality", reflection.Quality),
							logger.Float64("score", reflection.Score),
						)
					}
				}
			}
		}

		// Step 4: Check confidence threshold
		if trace.Confidence < s.confidenceThreshold {
			if s.logger != nil {
				s.logger.Warn("Confidence below threshold",
					logger.Float64("confidence", trace.Confidence),
					logger.Float64("threshold", s.confidenceThreshold),
				)
			}
		}

		// Prepare next iteration input
		currentInput = s.buildNextInput(trace)
	}

	// Max iterations reached without final answer
	execution.Status = ExecutionStatusCompleted
	execution.FinalOutput = "Max iterations reached without definitive answer"
	if len(s.traces) > 0 {
		lastTrace := s.traces[len(s.traces)-1]
		if lastTrace.Observation != "" {
			execution.FinalOutput = lastTrace.Observation
		}
	}
	execution.EndTime = time.Now()

	if s.logger != nil {
		s.logger.Warn("ReAct strategy completed at max iterations",
			logger.String("execution_id", execution.ID),
			logger.Int("iterations", s.maxIterations),
		)
	}

	return execution, nil
}

// think generates reasoning about what to do next.
func (s *ReactStrategy) think(
	ctx context.Context,
	agent *Agent,
	input string,
	memoryContext string,
	iteration int,
) (ReasoningTrace, error) {
	trace := ReasoningTrace{
		Step:      iteration,
		Timestamp: time.Now(),
		Metadata:  make(map[string]any),
	}

	// Build prompt with context
	prompt := s.buildReasoningPrompt(input, memoryContext, s.traces)

	// Call LLM for reasoning
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

	if agent.temperature != 0 {
		request.Temperature = &agent.temperature
	}

	// Add tools if available
	if len(agent.tools) > 0 {
		llmTools := make([]llm.Tool, len(agent.tools))
		for i, tool := range agent.tools {
			llmTools[i] = llm.Tool{
				Type: "function",
				Function: &llm.FunctionDefinition{
					Name:        tool.Name,
					Description: tool.Description,
					Parameters:  tool.Parameters,
				},
			}
		}
		request.Tools = llmTools
	}

	response, err := agent.llmManager.Chat(ctx, request)
	if err != nil {
		return trace, fmt.Errorf("LLM call failed: %w", err)
	}

	if len(response.Choices) == 0 {
		return trace, errors.New("no response from LLM")
	}

	// Parse response
	content := response.Choices[0].Message.Content
	trace.Thought, trace.Action, trace.ActionInput, trace.Confidence = s.parseReasoningResponse(content)

	// Handle tool calls from structured tool calling
	if len(response.Choices[0].Message.ToolCalls) > 0 {
		toolCall := response.Choices[0].Message.ToolCalls[0]
		trace.Action = toolCall.Function.Name

		// Parse arguments
		var args map[string]any
		if toolCall.Function.Arguments != "" {
			_ = json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
			trace.ActionInput = args
		}
	}

	return trace, nil
}

// act executes the specified action/tool.
func (s *ReactStrategy) act(ctx context.Context, agent *Agent, trace ReasoningTrace) (string, error) {
	// Find the tool
	var tool *Tool
	for i := range agent.tools {
		if agent.tools[i].Name == trace.Action {
			tool = &agent.tools[i]
			break
		}
	}

	if tool == nil {
		return "", fmt.Errorf("tool not found: %s", trace.Action)
	}

	// Execute tool
	if tool.Handler == nil {
		return "", fmt.Errorf("tool %s has no handler", trace.Action)
	}

	result, err := tool.Handler(ctx, trace.ActionInput)
	if err != nil {
		return "", fmt.Errorf("tool execution failed: %w", err)
	}

	// Convert result to string
	observation := fmt.Sprintf("%v", result)
	if resultStr, ok := result.(string); ok {
		observation = resultStr
	} else if resultBytes, err := json.Marshal(result); err == nil {
		observation = string(resultBytes)
	}

	if s.metrics != nil {
		s.metrics.Counter("forge.ai.sdk.react.tool_calls",
			metrics.WithLabel("tool", trace.Action),
		).Inc()
	}

	return observation, nil
}

// reflect performs self-assessment of reasoning quality.
func (s *ReactStrategy) reflect(
	ctx context.Context,
	agent *Agent,
	traces []ReasoningTrace,
) (ReflectionResult, error) {
	reflection := ReflectionResult{
		Issues:      make([]string, 0),
		Suggestions: make([]string, 0),
	}

	// Build reflection prompt
	prompt := s.buildReflectionPrompt(traces)

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

	response, err := agent.llmManager.Chat(ctx, request)
	if err != nil {
		return reflection, err
	}

	if len(response.Choices) == 0 {
		return reflection, errors.New("no reflection response")
	}

	// Parse reflection
	content := response.Choices[0].Message.Content
	reflection.Quality = s.extractQuality(content)
	reflection.Score = s.extractScore(content)
	reflection.Reasoning = content
	reflection.Issues = s.extractIssues(content)
	reflection.Suggestions = s.extractSuggestions(content)
	reflection.ShouldReplan = strings.Contains(strings.ToLower(content), "replan") ||
		strings.Contains(strings.ToLower(content), "start over")

	return reflection, nil
}

// Helper methods

func (s *ReactStrategy) buildReasoningPrompt(input string, memoryContext string, previousTraces []ReasoningTrace) string {
	var sb strings.Builder

	// Add memory context if available
	if memoryContext != "" {
		sb.WriteString("Relevant past experiences:\n")
		sb.WriteString(memoryContext)
		sb.WriteString("\n\n")
	}

	// Add task
	sb.WriteString(fmt.Sprintf(s.reasoningPrompt, input))

	// Add previous traces
	if len(previousTraces) > 0 {
		sb.WriteString("\n\nPrevious steps:\n")
		for _, trace := range previousTraces {
			sb.WriteString(fmt.Sprintf("Step %d:\n", trace.Step))
			sb.WriteString(fmt.Sprintf("  Thought: %s\n", trace.Thought))
			if trace.Action != "" {
				sb.WriteString(fmt.Sprintf("  Action: %s\n", trace.Action))
			}
			if trace.Observation != "" {
				sb.WriteString(fmt.Sprintf("  Observation: %s\n", trace.Observation))
			}
			sb.WriteString("\n")
		}
	}

	sb.WriteString("\nWhat is your next thought and action?")

	return sb.String()
}

func (s *ReactStrategy) buildReflectionPrompt(traces []ReasoningTrace) string {
	var sb strings.Builder
	sb.WriteString(s.reflectionPrompt)
	sb.WriteString("\n\nReasoning steps to evaluate:\n")

	for _, trace := range traces {
		sb.WriteString(fmt.Sprintf("Step %d: %s\n", trace.Step, trace.Thought))
		if trace.Action != "" {
			sb.WriteString(fmt.Sprintf("  Action: %s\n", trace.Action))
		}
		if trace.Observation != "" {
			sb.WriteString(fmt.Sprintf("  Observation: %s\n", trace.Observation))
		}
	}

	return sb.String()
}

func (s *ReactStrategy) parseReasoningResponse(content string) (thought, action string, actionInput map[string]any, confidence float64) {
	confidence = 0.8 // Default confidence

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(strings.ToLower(line), "thought:") {
			thought = strings.TrimSpace(strings.TrimPrefix(strings.ToLower(line), "thought:"))
		} else if strings.HasPrefix(strings.ToLower(line), "action:") {
			action = strings.TrimSpace(strings.TrimPrefix(strings.ToLower(line), "action:"))
		} else if strings.HasPrefix(strings.ToLower(line), "action input:") {
			inputStr := strings.TrimSpace(strings.TrimPrefix(strings.ToLower(line), "action input:"))
			_ = json.Unmarshal([]byte(inputStr), &actionInput)
		} else if strings.HasPrefix(strings.ToLower(line), "confidence:") {
			confStr := strings.TrimSpace(strings.TrimPrefix(strings.ToLower(line), "confidence:"))
			_, _ = fmt.Sscanf(confStr, "%f", &confidence) // Best effort parsing
		}
	}

	// If no structured format, treat entire content as thought
	if thought == "" {
		thought = content
	}

	return
}

func (s *ReactStrategy) isFinalAnswer(trace ReasoningTrace) bool {
	lowerThought := strings.ToLower(trace.Thought)
	lowerObs := strings.ToLower(trace.Observation)

	finalMarkers := []string{
		"final answer:",
		"the answer is",
		"in conclusion",
		"therefore,",
		"to summarize",
	}

	for _, marker := range finalMarkers {
		if strings.Contains(lowerThought, marker) || strings.Contains(lowerObs, marker) {
			return true
		}
	}

	// No action means final answer
	return trace.Action == ""
}

func (s *ReactStrategy) buildNextInput(trace ReasoningTrace) string {
	if trace.Observation != "" {
		return trace.Observation
	}
	return trace.Thought
}

func (s *ReactStrategy) formatMemories(memories []*MemoryEntry) string {
	var sb strings.Builder
	for i, mem := range memories {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, mem.Content))
	}
	return sb.String()
}

func (s *ReactStrategy) extractQuality(content string) string {
	lower := strings.ToLower(content)
	if strings.Contains(lower, "good") || strings.Contains(lower, "excellent") {
		return "good"
	}
	if strings.Contains(lower, "invalid") || strings.Contains(lower, "incorrect") {
		return "invalid"
	}
	return "needs_improvement"
}

func (s *ReactStrategy) extractScore(content string) float64 {
	// Try to find a numeric score
	var score float64
	if strings.Contains(content, "score:") {
		_, _ = fmt.Sscanf(content, "score: %f", &score) // Best effort parsing
	}
	// Default based on quality
	if score == 0 {
		quality := s.extractQuality(content)
		switch quality {
		case "good":
			score = 0.8
		case "needs_improvement":
			score = 0.6
		case "invalid":
			score = 0.3
		}
	}
	return score
}

func (s *ReactStrategy) extractIssues(content string) []string {
	// Simple extraction - look for lines starting with "Issue:" or "-"
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

func (s *ReactStrategy) extractSuggestions(content string) []string {
	// Simple extraction
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
func (s *ReactStrategy) Name() string {
	return "ReAct"
}

// SupportsReplanning indicates if this strategy can replan.
func (s *ReactStrategy) SupportsReplanning() bool {
	return false // ReAct doesn't do explicit replanning
}

// GetTraces returns the reasoning traces.
func (s *ReactStrategy) GetTraces() []ReasoningTrace {
	return s.traces
}

// GetReflections returns the reflection results.
func (s *ReactStrategy) GetReflections() []ReflectionResult {
	return s.reflections
}

// Default prompts
const defaultReasoningPrompt = `You are solving: %s

Use this format:
Thought: [Your reasoning about what to do next]
Action: [Tool to use, or leave blank if you have the final answer]
Action Input: [JSON arguments for the tool]
Confidence: [Your confidence level 0-1]

OR if you have the final answer:
Thought: [Your reasoning about the solution]
Final Answer: [Your complete answer]

Be concise and logical in your reasoning.`

const defaultReflectionPrompt = `Evaluate the quality of the following reasoning steps.

Consider:
1. Logical consistency
2. Tool choice appropriateness
3. Progress toward the goal
4. Any circular reasoning or mistakes

Provide:
- Quality assessment (good/needs_improvement/invalid)
- Specific issues found
- Suggestions for improvement
- Whether to replan from scratch`
