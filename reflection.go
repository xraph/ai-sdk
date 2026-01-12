package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/xraph/ai-sdk/llm"
	logger "github.com/xraph/go-utils/log"
	"github.com/xraph/go-utils/metrics"
)

// ReflectionEngine evaluates the quality of reasoning and plans.
type ReflectionEngine struct {
	llmManager LLMManager
	provider   string
	model      string
	logger     logger.Logger
	metrics    metrics.Metrics

	// Configuration
	qualityThreshold float64
	criteria         []ReflectionCriterion
	promptTemplate   string
}

// ReflectionEngineConfig configures the reflection engine.
type ReflectionEngineConfig struct {
	LLMManager       LLMManager
	Provider         string
	Model            string
	QualityThreshold float64               // Minimum acceptable quality score
	Criteria         []ReflectionCriterion // Custom evaluation criteria
	PromptTemplate   string                // Custom prompt template
}

// NewReflectionEngine creates a new reflection engine.
func NewReflectionEngine(logger logger.Logger, metrics metrics.Metrics, config *ReflectionEngineConfig) *ReflectionEngine {
	if config == nil {
		config = &ReflectionEngineConfig{}
	}

	// Set defaults
	if config.QualityThreshold == 0 {
		config.QualityThreshold = 0.7
	}

	if len(config.Criteria) == 0 {
		config.Criteria = getDefaultCriteria()
	}

	if config.PromptTemplate == "" {
		config.PromptTemplate = getDefaultReflectionPrompt()
	}

	return &ReflectionEngine{
		llmManager:       config.LLMManager,
		provider:         config.Provider,
		model:            config.Model,
		logger:           logger,
		metrics:          metrics,
		qualityThreshold: config.QualityThreshold,
		criteria:         config.Criteria,
		promptTemplate:   config.PromptTemplate,
	}
}

// EvaluateStep evaluates the quality of a single reasoning or execution step.
func (re *ReflectionEngine) EvaluateStep(ctx context.Context, step *AgentStep, history []*AgentStep) (*ReflectionResult, error) {
	if re.logger != nil {
		re.logger.Debug("Evaluating step",
			logger.String("step_id", step.ID),
			logger.Int("step_index", step.Index),
		)
	}

	// Build evaluation prompt
	prompt := re.buildStepEvaluationPrompt(step, history)

	// Call LLM for evaluation
	request := llm.ChatRequest{
		Provider: re.provider,
		Model:    re.model,
		Messages: []llm.ChatMessage{
			{
				Role:    "system",
				Content: "You are an expert at evaluating reasoning quality and identifying issues.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}
	temp := 0.3
	request.Temperature = &temp // Lower temperature for more consistent evaluation

	response, err := re.llmManager.Chat(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("LLM evaluation call failed: %w", err)
	}

	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no evaluation response from LLM")
	}

	// Parse reflection result
	result, err := re.parseReflectionResult(response.Choices[0].Message.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse reflection result: %w", err)
	}

	if re.metrics != nil {
		re.metrics.Counter("forge.ai.sdk.reflection.evaluations").Inc()
		re.metrics.Histogram("forge.ai.sdk.reflection.quality_score").Observe(result.Score)
	}

	return result, nil
}

// EvaluateTrace evaluates a reasoning trace.
func (re *ReflectionEngine) EvaluateTrace(ctx context.Context, trace ReasoningTrace) (*ReflectionResult, error) {
	if re.logger != nil {
		re.logger.Debug("Evaluating reasoning trace",
			logger.Int("step", trace.Step),
		)
	}

	// Build evaluation prompt
	prompt := re.buildTraceEvaluationPrompt(trace)

	// Call LLM
	request := llm.ChatRequest{
		Provider: re.provider,
		Model:    re.model,
		Messages: []llm.ChatMessage{
			{
				Role:    "system",
				Content: "You are an expert at evaluating reasoning quality.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}
	tempTrace := 0.3
	request.Temperature = &tempTrace

	response, err := re.llmManager.Chat(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("LLM evaluation call failed: %w", err)
	}

	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no evaluation response")
	}

	// Parse result
	result, err := re.parseReflectionResult(response.Choices[0].Message.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse reflection: %w", err)
	}

	return result, nil
}

// EvaluatePlan evaluates the quality of an entire plan.
func (re *ReflectionEngine) EvaluatePlan(ctx context.Context, plan *Plan) (*ReflectionResult, error) {
	if re.logger != nil {
		re.logger.Debug("Evaluating plan",
			logger.String("plan_id", plan.ID),
			logger.Int("steps", len(plan.Steps)),
		)
	}

	// Build evaluation prompt
	prompt := re.buildPlanEvaluationPrompt(plan)

	// Call LLM
	request := llm.ChatRequest{
		Provider: re.provider,
		Model:    re.model,
		Messages: []llm.ChatMessage{
			{
				Role:    "system",
				Content: "You are an expert at evaluating plan quality and feasibility.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}
	tempPlan := 0.3
	request.Temperature = &tempPlan

	response, err := re.llmManager.Chat(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("LLM evaluation call failed: %w", err)
	}

	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no evaluation response")
	}

	// Parse result
	result, err := re.parseReflectionResult(response.Choices[0].Message.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse reflection: %w", err)
	}

	// Check if plan should be replanned
	result.ShouldReplan = result.Score < re.qualityThreshold || result.Quality == "invalid"

	if re.metrics != nil {
		re.metrics.Counter("forge.ai.sdk.reflection.plan_evaluations").Inc()
		if result.ShouldReplan {
			re.metrics.Counter("forge.ai.sdk.reflection.replan_triggered").Inc()
		}
	}

	return result, nil
}

// ShouldReplan determines if replanning is needed based on reflection.
func (re *ReflectionEngine) ShouldReplan(result *ReflectionResult) bool {
	if result == nil {
		return false
	}

	// Replan if explicitly flagged
	if result.ShouldReplan {
		return true
	}

	// Replan if quality is below threshold
	if result.Score < re.qualityThreshold {
		return true
	}

	// Replan if quality is invalid
	if result.Quality == "invalid" {
		return true
	}

	return false
}

// Helper methods

func (re *ReflectionEngine) buildStepEvaluationPrompt(step *AgentStep, history []*AgentStep) string {
	var sb strings.Builder

	sb.WriteString(re.promptTemplate)
	sb.WriteString("\n\n## Current Step to Evaluate\n\n")
	sb.WriteString(fmt.Sprintf("**Input:** %s\n", step.Input))
	sb.WriteString(fmt.Sprintf("**Output:** %s\n", step.Output))

	if step.Reasoning != "" {
		sb.WriteString(fmt.Sprintf("**Reasoning:** %s\n", step.Reasoning))
	}

	if len(step.ToolCalls) > 0 {
		sb.WriteString("\n**Tool Calls:**\n")
		for _, tc := range step.ToolCalls {
			sb.WriteString(fmt.Sprintf("- %s: %v\n", tc.Name, tc.Arguments))
		}
	}

	if len(step.ToolResults) > 0 {
		sb.WriteString("\n**Tool Results:**\n")
		for _, tr := range step.ToolResults {
			sb.WriteString(fmt.Sprintf("- %s: %v\n", tr.Name, tr.Result))
			if tr.Error != "" {
				sb.WriteString(fmt.Sprintf("  Error: %s\n", tr.Error))
			}
		}
	}

	if step.Error != "" {
		sb.WriteString(fmt.Sprintf("\n**Error:** %s\n", step.Error))
	}

	// Add recent history for context
	if len(history) > 0 {
		sb.WriteString("\n## Recent History (for context)\n\n")
		start := len(history) - 3
		if start < 0 {
			start = 0
		}
		for i := start; i < len(history); i++ {
			h := history[i]
			sb.WriteString(fmt.Sprintf("Step %d: %s → %s\n", h.Index, h.Input, h.Output))
		}
	}

	sb.WriteString("\n## Evaluation Criteria\n\n")
	for _, criterion := range re.criteria {
		sb.WriteString(fmt.Sprintf("- **%s** (weight: %.2f): %s\n", criterion.Name, criterion.Weight, criterion.Description))
	}

	sb.WriteString("\n## Required Output Format\n\n")
	sb.WriteString("Provide your evaluation in JSON format:\n")
	sb.WriteString("```json\n")
	sb.WriteString("{\n")
	sb.WriteString("  \"quality\": \"good\" | \"needs_improvement\" | \"invalid\",\n")
	sb.WriteString("  \"score\": 0.0-1.0,\n")
	sb.WriteString("  \"issues\": [\"issue1\", \"issue2\"],\n")
	sb.WriteString("  \"suggestions\": [\"suggestion1\", \"suggestion2\"],\n")
	sb.WriteString("  \"reasoning\": \"explanation of the evaluation\"\n")
	sb.WriteString("}\n")
	sb.WriteString("```\n")

	return sb.String()
}

func (re *ReflectionEngine) buildTraceEvaluationPrompt(trace ReasoningTrace) string {
	var sb strings.Builder

	sb.WriteString("Evaluate the quality of this reasoning trace:\n\n")
	sb.WriteString(fmt.Sprintf("**Thought:** %s\n", trace.Thought))
	sb.WriteString(fmt.Sprintf("**Action:** %s\n", trace.Action))
	sb.WriteString(fmt.Sprintf("**Observation:** %s\n", trace.Observation))

	if trace.Reflection != "" {
		sb.WriteString(fmt.Sprintf("**Self-Reflection:** %s\n", trace.Reflection))
	}

	sb.WriteString(fmt.Sprintf("\n**Confidence:** %.2f\n", trace.Confidence))

	sb.WriteString("\nEvaluate:\n")
	sb.WriteString("1. Is the thought process logical and clear?\n")
	sb.WriteString("2. Is the chosen action appropriate for the thought?\n")
	sb.WriteString("3. Does the observation provide useful information?\n")
	sb.WriteString("4. Is the confidence level reasonable?\n")

	sb.WriteString("\nProvide evaluation in JSON format with quality, score, issues, and suggestions.\n")

	return sb.String()
}

func (re *ReflectionEngine) buildPlanEvaluationPrompt(plan *Plan) string {
	var sb strings.Builder

	sb.WriteString("Evaluate the quality of this plan:\n\n")
	sb.WriteString(fmt.Sprintf("**Goal:** %s\n\n", plan.Goal))
	sb.WriteString(fmt.Sprintf("**Status:** %s\n", plan.Status))
	sb.WriteString(fmt.Sprintf("**Steps:** %d\n\n", len(plan.Steps)))

	sb.WriteString("## Plan Steps:\n\n")
	for i, step := range plan.Steps {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, step.Description))
		if len(step.ToolsNeeded) > 0 {
			sb.WriteString(fmt.Sprintf("   Tools: %v\n", step.ToolsNeeded))
		}
		if len(step.Dependencies) > 0 {
			sb.WriteString(fmt.Sprintf("   Dependencies: %v\n", step.Dependencies))
		}
		sb.WriteString(fmt.Sprintf("   Status: %s\n", step.Status))
		if step.Result != nil {
			sb.WriteString(fmt.Sprintf("   Result: %v\n", step.Result))
		}
		if step.Error != "" {
			sb.WriteString(fmt.Sprintf("   Error: %s\n", step.Error))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("## Evaluation Criteria:\n\n")
	sb.WriteString("1. **Completeness:** Does the plan cover all necessary steps to achieve the goal?\n")
	sb.WriteString("2. **Feasibility:** Are the steps realistic and achievable?\n")
	sb.WriteString("3. **Dependencies:** Are dependencies correctly identified and ordered?\n")
	sb.WriteString("4. **Tool Usage:** Are the right tools assigned to each step?\n")
	sb.WriteString("5. **Error Handling:** Are there issues that need addressing?\n")

	sb.WriteString("\nProvide evaluation in JSON format with quality, score, issues, suggestions, and shouldReplan flag.\n")

	return sb.String()
}

func (re *ReflectionEngine) parseReflectionResult(content string) (*ReflectionResult, error) {
	// Try to extract JSON from response
	jsonStart := strings.Index(content, "{")
	jsonEnd := strings.LastIndex(content, "}")

	if jsonStart == -1 || jsonEnd == -1 {
		// Fallback: parse from text
		return re.parseReflectionFromText(content), nil
	}

	jsonStr := content[jsonStart : jsonEnd+1]

	var result ReflectionResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		// Fallback to text parsing
		return re.parseReflectionFromText(content), nil
	}

	// Calculate score if not provided
	if result.Score == 0 {
		result.Score = re.calculateScoreFromQuality(result.Quality)
	}

	return &result, nil
}

func (re *ReflectionEngine) parseReflectionFromText(content string) *ReflectionResult {
	result := &ReflectionResult{
		Quality:     "needs_improvement",
		Score:       0.5,
		Issues:      make([]string, 0),
		Suggestions: make([]string, 0),
		Reasoning:   content,
	}

	lower := strings.ToLower(content)

	// Determine quality from keywords
	if strings.Contains(lower, "excellent") || strings.Contains(lower, "perfect") || strings.Contains(lower, "outstanding") {
		result.Quality = "good"
		result.Score = 0.9
	} else if strings.Contains(lower, "good") || strings.Contains(lower, "satisfactory") || strings.Contains(lower, "adequate") {
		result.Quality = "good"
		result.Score = 0.75
	} else if strings.Contains(lower, "invalid") || strings.Contains(lower, "incorrect") || strings.Contains(lower, "wrong") {
		result.Quality = "invalid"
		result.Score = 0.3
	}

	// Extract issues
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(line), "issue:") || strings.HasPrefix(strings.ToLower(line), "- issue:") {
			issue := strings.TrimSpace(strings.TrimPrefix(strings.ToLower(line), "issue:"))
			issue = strings.TrimSpace(strings.TrimPrefix(issue, "- "))
			if issue != "" {
				result.Issues = append(result.Issues, issue)
			}
		}
		if strings.HasPrefix(strings.ToLower(line), "suggestion:") || strings.HasPrefix(strings.ToLower(line), "- suggestion:") {
			suggestion := strings.TrimSpace(strings.TrimPrefix(strings.ToLower(line), "suggestion:"))
			suggestion = strings.TrimSpace(strings.TrimPrefix(suggestion, "- "))
			if suggestion != "" {
				result.Suggestions = append(result.Suggestions, suggestion)
			}
		}
	}

	// Determine if replanning is needed
	result.ShouldReplan = result.Quality == "invalid" || len(result.Issues) > 3

	return result
}

func (re *ReflectionEngine) calculateScoreFromQuality(quality string) float64 {
	switch quality {
	case "good":
		return 0.8
	case "needs_improvement":
		return 0.6
	case "invalid":
		return 0.3
	default:
		return 0.5
	}
}

// Default configuration

func getDefaultCriteria() []ReflectionCriterion {
	return []ReflectionCriterion{
		{
			Name:        "Logical Coherence",
			Description: "Is the reasoning logically sound and consistent?",
			Weight:      0.3,
		},
		{
			Name:        "Action Appropriateness",
			Description: "Is the chosen action suitable for the situation?",
			Weight:      0.25,
		},
		{
			Name:        "Completeness",
			Description: "Does the step fully address the required task?",
			Weight:      0.2,
		},
		{
			Name:        "Efficiency",
			Description: "Is the approach efficient and not overly complex?",
			Weight:      0.15,
		},
		{
			Name:        "Error Handling",
			Description: "Are errors properly identified and handled?",
			Weight:      0.1,
		},
	}
}

func getDefaultReflectionPrompt() string {
	return `You are evaluating the quality of an agent's reasoning or execution step.
Your role is to identify strengths, weaknesses, and provide constructive feedback.

Focus on:
- Logical coherence and consistency
- Appropriateness of actions taken
- Completeness of the solution
- Efficiency and simplicity
- Error handling and recovery`
}
