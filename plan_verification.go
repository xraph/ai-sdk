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

// PlanVerifier validates plans and step results.
type PlanVerifier struct {
	llmManager LLMManager
	provider   string
	model      string
	logger     logger.Logger
	metrics    metrics.Metrics

	// Configuration
	rules            []VerificationRule
	qualityThreshold float64
	structuralChecks bool
	semanticChecks   bool
	promptTemplate   string
}

// VerificationRule defines a validation rule.
type VerificationRule struct {
	Name        string                          `json:"name"`
	Description string                          `json:"description"`
	Weight      float64                         `json:"weight"` // 0-1, importance of this rule
	Validator   func(*Plan) *VerificationResult `json:"-"`
}

// PlanVerifierConfig configures the plan verifier.
type PlanVerifierConfig struct {
	LLMManager       LLMManager
	Provider         string
	Model            string
	Rules            []VerificationRule
	QualityThreshold float64
	StructuralChecks bool
	SemanticChecks   bool
	PromptTemplate   string
}

// NewPlanVerifier creates a new plan verifier.
func NewPlanVerifier(logger logger.Logger, metrics metrics.Metrics, config *PlanVerifierConfig) *PlanVerifier {
	if config == nil {
		config = &PlanVerifierConfig{}
	}

	// Set defaults
	if config.QualityThreshold == 0 {
		config.QualityThreshold = 0.7
	}

	if len(config.Rules) == 0 {
		config.Rules = getDefaultVerificationRules()
	}

	if config.PromptTemplate == "" {
		config.PromptTemplate = getDefaultVerificationPrompt()
	}

	return &PlanVerifier{
		llmManager:       config.LLMManager,
		provider:         config.Provider,
		model:            config.Model,
		logger:           logger,
		metrics:          metrics,
		rules:            config.Rules,
		qualityThreshold: config.QualityThreshold,
		structuralChecks: config.StructuralChecks || true,
		semanticChecks:   config.SemanticChecks || true,
		promptTemplate:   config.PromptTemplate,
	}
}

// VerifyStep validates a single plan step's output.
func (pv *PlanVerifier) VerifyStep(ctx context.Context, step *PlanStep, expectedOutcome string) (*VerificationResult, error) {
	if pv.logger != nil {
		pv.logger.Debug("Verifying step",
			logger.String("step_id", step.ID),
			logger.String("description", step.Description),
		)
	}

	// Build verification prompt
	prompt := pv.buildStepVerificationPrompt(step, expectedOutcome)

	// Call LLM for verification
	request := llm.ChatRequest{
		Provider: pv.provider,
		Model:    pv.model,
		Messages: []llm.ChatMessage{
			{
				Role:    "system",
				Content: "You are an expert at validating task execution results.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}
	temp := 0.3
	request.Temperature = &temp

	response, err := pv.llmManager.Chat(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("LLM verification call failed: %w", err)
	}

	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no verification response")
	}

	// Parse verification result
	result, err := pv.parseVerificationResult(response.Choices[0].Message.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse verification: %w", err)
	}

	if pv.metrics != nil {
		pv.metrics.Counter("forge.ai.sdk.verification.step_verifications").Inc()
		pv.metrics.Histogram("forge.ai.sdk.verification.step_score").Observe(result.Score)
	}

	return result, nil
}

// VerifyPlanStructure checks the structural integrity of a plan.
func (pv *PlanVerifier) VerifyPlanStructure(ctx context.Context, plan *Plan) (*VerificationResult, error) {
	if !pv.structuralChecks {
		return &VerificationResult{IsValid: true, Score: 1.0, Reasoning: "Structural checks disabled"}, nil
	}

	result := &VerificationResult{
		IsValid:     true,
		Score:       1.0,
		Issues:      make([]string, 0),
		Suggestions: make([]string, 0),
	}

	// Check for empty plan
	if len(plan.Steps) == 0 {
		result.IsValid = false
		result.Score = 0.0
		result.Issues = append(result.Issues, "Plan has no steps")
		return result, nil
	}

	// Check for circular dependencies
	if hasCycle := pv.checkCircularDependencies(plan); hasCycle {
		result.IsValid = false
		result.Score = 0.2
		result.Issues = append(result.Issues, "Plan contains circular dependencies")
	}

	// Check for invalid dependencies
	stepIDs := make(map[string]bool)
	for _, step := range plan.Steps {
		stepIDs[step.ID] = true
	}

	for _, step := range plan.Steps {
		for _, depID := range step.Dependencies {
			if !stepIDs[depID] {
				result.Issues = append(result.Issues, fmt.Sprintf("Step %s has invalid dependency: %s", step.ID, depID))
				result.Score -= 0.1
			}
		}
	}

	// Check for orphaned steps (steps that nothing depends on and don't produce final output)
	dependedOn := make(map[string]bool)
	for _, step := range plan.Steps {
		for _, depID := range step.Dependencies {
			dependedOn[depID] = true
		}
	}

	orphanCount := 0
	for _, step := range plan.Steps {
		if !dependedOn[step.ID] && step.Index < len(plan.Steps)-1 {
			orphanCount++
		}
	}

	if orphanCount > 0 {
		result.Suggestions = append(result.Suggestions, fmt.Sprintf("%d steps may be orphaned (not depended on)", orphanCount))
		result.Score -= float64(orphanCount) * 0.05
	}

	// Ensure score doesn't go below 0
	if result.Score < 0 {
		result.Score = 0
	}

	result.IsValid = result.Score >= pv.qualityThreshold

	if pv.metrics != nil {
		pv.metrics.Counter("forge.ai.sdk.verification.structure_checks").Inc()
	}

	return result, nil
}

// VerifyPlanQuality performs semantic validation of the plan.
func (pv *PlanVerifier) VerifyPlanQuality(ctx context.Context, plan *Plan, goal string) (*VerificationResult, error) {
	if !pv.semanticChecks {
		return &VerificationResult{IsValid: true, Score: 1.0, Reasoning: "Semantic checks disabled"}, nil
	}

	if pv.logger != nil {
		pv.logger.Debug("Verifying plan quality",
			logger.String("plan_id", plan.ID),
			logger.String("goal", goal),
		)
	}

	// Build verification prompt
	prompt := pv.buildPlanQualityPrompt(plan, goal)

	// Call LLM for semantic verification
	request := llm.ChatRequest{
		Provider: pv.provider,
		Model:    pv.model,
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
	temp := 0.3
	request.Temperature = &temp

	response, err := pv.llmManager.Chat(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("LLM quality verification failed: %w", err)
	}

	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no quality verification response")
	}

	// Parse result
	result, err := pv.parseVerificationResult(response.Choices[0].Message.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse quality verification: %w", err)
	}

	if pv.metrics != nil {
		pv.metrics.Counter("forge.ai.sdk.verification.quality_checks").Inc()
		pv.metrics.Histogram("forge.ai.sdk.verification.quality_score").Observe(result.Score)
	}

	return result, nil
}

// VerifyPlan performs comprehensive plan validation (structural + semantic).
func (pv *PlanVerifier) VerifyPlan(ctx context.Context, plan *Plan) (*VerificationResult, error) {
	// Run structural checks
	structResult, err := pv.VerifyPlanStructure(ctx, plan)
	if err != nil {
		return nil, fmt.Errorf("structural verification failed: %w", err)
	}

	// If structural checks fail badly, don't bother with semantic checks
	if !structResult.IsValid && structResult.Score < 0.3 {
		return structResult, nil
	}

	// Run semantic checks
	qualityResult, err := pv.VerifyPlanQuality(ctx, plan, plan.Goal)
	if err != nil {
		// If semantic check fails, return structural result
		if pv.logger != nil {
			pv.logger.Warn("Quality verification failed, using structural result", logger.Error(err))
		}
		return structResult, nil
	}

	// Combine results (weighted average)
	combinedResult := &VerificationResult{
		Score:       (structResult.Score*0.4 + qualityResult.Score*0.6),
		Issues:      append(structResult.Issues, qualityResult.Issues...),
		Suggestions: append(structResult.Suggestions, qualityResult.Suggestions...),
		Reasoning:   fmt.Sprintf("Structural: %s\nQuality: %s", structResult.Reasoning, qualityResult.Reasoning),
	}

	combinedResult.IsValid = combinedResult.Score >= pv.qualityThreshold

	return combinedResult, nil
}

// Helper methods

func (pv *PlanVerifier) checkCircularDependencies(plan *Plan) bool {
	// Build adjacency list
	graph := make(map[string][]string)
	for _, step := range plan.Steps {
		graph[step.ID] = step.Dependencies
	}

	// DFS to detect cycles
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var hasCycle func(string) bool
	hasCycle = func(nodeID string) bool {
		visited[nodeID] = true
		recStack[nodeID] = true

		for _, depID := range graph[nodeID] {
			if !visited[depID] {
				if hasCycle(depID) {
					return true
				}
			} else if recStack[depID] {
				return true
			}
		}

		recStack[nodeID] = false
		return false
	}

	for _, step := range plan.Steps {
		if !visited[step.ID] {
			if hasCycle(step.ID) {
				return true
			}
		}
	}

	return false
}

func (pv *PlanVerifier) buildStepVerificationPrompt(step *PlanStep, expectedOutcome string) string {
	var sb strings.Builder

	sb.WriteString(pv.promptTemplate)
	sb.WriteString("\n\n## Step to Verify\n\n")
	sb.WriteString(fmt.Sprintf("**Description:** %s\n", step.Description))
	sb.WriteString(fmt.Sprintf("**Status:** %s\n", step.Status))
	sb.WriteString(fmt.Sprintf("**Result:** %v\n", step.Result))

	if expectedOutcome != "" {
		sb.WriteString(fmt.Sprintf("\n**Expected Outcome:** %s\n", expectedOutcome))
	}

	if step.Error != "" {
		sb.WriteString(fmt.Sprintf("\n**Error:** %s\n", step.Error))
	}

	sb.WriteString("\n## Verification Criteria\n\n")
	sb.WriteString("1. Does the result match the expected outcome?\n")
	sb.WriteString("2. Is the result complete and usable?\n")
	sb.WriteString("3. Are there any errors or issues?\n")
	sb.WriteString("4. Is the result of sufficient quality?\n")

	sb.WriteString("\nProvide verification in JSON format with isValid, score (0-1), issues, and suggestions.\n")

	return sb.String()
}

func (pv *PlanVerifier) buildPlanQualityPrompt(plan *Plan, goal string) string {
	var sb strings.Builder

	sb.WriteString("Evaluate the quality and feasibility of this plan:\n\n")
	sb.WriteString(fmt.Sprintf("**Goal:** %s\n\n", goal))
	sb.WriteString("**Plan Steps:**\n\n")

	for i, step := range plan.Steps {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, step.Description))
		if len(step.ToolsNeeded) > 0 {
			sb.WriteString(fmt.Sprintf("   Tools: %v\n", step.ToolsNeeded))
		}
		if len(step.Dependencies) > 0 {
			sb.WriteString(fmt.Sprintf("   Depends on: %v\n", step.Dependencies))
		}
	}

	sb.WriteString("\n## Evaluation Criteria:\n\n")
	sb.WriteString("1. **Completeness:** Does the plan cover all necessary steps?\n")
	sb.WriteString("2. **Feasibility:** Are the steps realistic and achievable?\n")
	sb.WriteString("3. **Efficiency:** Is the plan optimally structured?\n")
	sb.WriteString("4. **Clarity:** Are step descriptions clear and actionable?\n")
	sb.WriteString("5. **Tool Usage:** Are tools appropriately assigned?\n")

	sb.WriteString("\nProvide evaluation in JSON format with isValid, score (0-1), issues, suggestions, and reasoning.\n")

	return sb.String()
}

func (pv *PlanVerifier) parseVerificationResult(content string) (*VerificationResult, error) {
	// Try to extract JSON from response
	jsonStart := strings.Index(content, "{")
	jsonEnd := strings.LastIndex(content, "}")

	if jsonStart == -1 || jsonEnd == -1 {
		// Fallback: parse from text
		return pv.parseVerificationFromText(content), nil
	}

	jsonStr := content[jsonStart : jsonEnd+1]

	var result VerificationResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		// Fallback to text parsing
		return pv.parseVerificationFromText(content), nil
	}

	// Ensure score is set
	if result.Score == 0 && result.IsValid {
		result.Score = 0.8
	} else if result.Score == 0 && !result.IsValid {
		result.Score = 0.3
	}

	return &result, nil
}

func (pv *PlanVerifier) parseVerificationFromText(content string) *VerificationResult {
	result := &VerificationResult{
		IsValid:     true,
		Score:       0.7,
		Issues:      make([]string, 0),
		Suggestions: make([]string, 0),
		Reasoning:   content,
	}

	lower := strings.ToLower(content)

	// Determine validity from keywords
	if strings.Contains(lower, "invalid") || strings.Contains(lower, "fail") || strings.Contains(lower, "incorrect") {
		result.IsValid = false
		result.Score = 0.4
	} else if strings.Contains(lower, "excellent") || strings.Contains(lower, "perfect") {
		result.Score = 0.9
	}

	// Extract issues and suggestions
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(line), "issue:") {
			issue := strings.TrimSpace(strings.TrimPrefix(strings.ToLower(line), "issue:"))
			if issue != "" {
				result.Issues = append(result.Issues, issue)
			}
		}
		if strings.HasPrefix(strings.ToLower(line), "suggestion:") {
			suggestion := strings.TrimSpace(strings.TrimPrefix(strings.ToLower(line), "suggestion:"))
			if suggestion != "" {
				result.Suggestions = append(result.Suggestions, suggestion)
			}
		}
	}

	return result
}

// Default configuration

func getDefaultVerificationRules() []VerificationRule {
	return []VerificationRule{
		{
			Name:        "NoEmptySteps",
			Description: "Plan must have at least one step",
			Weight:      1.0,
			Validator: func(plan *Plan) *VerificationResult {
				if len(plan.Steps) == 0 {
					return &VerificationResult{
						IsValid: false,
						Score:   0.0,
						Issues:  []string{"Plan has no steps"},
					}
				}
				return &VerificationResult{IsValid: true, Score: 1.0}
			},
		},
		{
			Name:        "ValidDependencies",
			Description: "All step dependencies must reference existing steps",
			Weight:      0.9,
			Validator: func(plan *Plan) *VerificationResult {
				stepIDs := make(map[string]bool)
				for _, step := range plan.Steps {
					stepIDs[step.ID] = true
				}

				issues := make([]string, 0)
				for _, step := range plan.Steps {
					for _, depID := range step.Dependencies {
						if !stepIDs[depID] {
							issues = append(issues, fmt.Sprintf("Invalid dependency: %s", depID))
						}
					}
				}

				if len(issues) > 0 {
					return &VerificationResult{
						IsValid: false,
						Score:   0.3,
						Issues:  issues,
					}
				}
				return &VerificationResult{IsValid: true, Score: 1.0}
			},
		},
	}
}

func getDefaultVerificationPrompt() string {
	return `You are validating the quality and correctness of a plan or step execution.
Your role is to identify issues, assess quality, and provide constructive feedback.

Focus on:
- Correctness and completeness
- Feasibility and practicality
- Clarity and actionability
- Potential issues or risks
- Opportunities for improvement`
}
