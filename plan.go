package sdk

import (
	"encoding/json"
	"fmt"
	"time"
)

// Plan represents a decomposed task with multiple steps.
// Used by Plan-Execute strategy to break down complex tasks.
type Plan struct {
	// ID is the unique identifier for this plan
	ID string `json:"id"`

	// AgentID is the agent that owns this plan
	AgentID string `json:"agent_id"`

	// Goal is the high-level objective
	Goal string `json:"goal"`

	// Steps are the individual actions to accomplish the goal
	Steps []PlanStep `json:"steps"`

	// Status tracks overall plan execution status
	Status PlanStatus `json:"status"`

	// CreatedAt is when the plan was created
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when the plan was last modified
	UpdatedAt time.Time `json:"updated_at"`

	// Metadata for additional plan context
	Metadata map[string]any `json:"metadata,omitempty"`

	// Version for tracking plan revisions
	Version int `json:"version"`

	// ParentPlanID if this is a replan
	ParentPlanID string `json:"parent_plan_id,omitempty"`
}

// PlanStep represents a single step in a plan.
type PlanStep struct {
	// ID is the unique identifier for this step
	ID string `json:"id"`

	// Index is the step number (0-based)
	Index int `json:"index"`

	// Description explains what this step does
	Description string `json:"description"`

	// ToolsNeeded lists the tools required for this step
	ToolsNeeded []string `json:"tools_needed,omitempty"`

	// Dependencies are step IDs that must complete first
	Dependencies []string `json:"dependencies,omitempty"`

	// Status tracks step execution status
	Status PlanStepStatus `json:"status"`

	// Result stores the step's output
	Result any `json:"result,omitempty"`

	// Error stores failure information
	Error string `json:"error,omitempty"`

	// Retries counts how many times this step has been retried
	Retries int `json:"retries"`

	// MaxRetries limits retry attempts
	MaxRetries int `json:"max_retries"`

	// Verification holds quality assessment
	Verification *VerificationResult `json:"verification,omitempty"`

	// StartTime when step execution began
	StartTime time.Time `json:"start_time,omitempty"`

	// EndTime when step execution completed
	EndTime time.Time `json:"end_time,omitempty"`

	// Metadata for additional step context
	Metadata map[string]any `json:"metadata,omitempty"`
}

// PlanStatus represents the overall status of a plan.
type PlanStatus string

const (
	PlanStatusPending    PlanStatus = "pending"
	PlanStatusInProgress PlanStatus = "in_progress"
	PlanStatusCompleted  PlanStatus = "completed"
	PlanStatusFailed     PlanStatus = "failed"
	PlanStatusCancelled  PlanStatus = "cancelled"
)

// PlanStepStatus represents the status of a single step.
type PlanStepStatus string

const (
	PlanStepStatusPending   PlanStepStatus = "pending"
	PlanStepStatusRunning   PlanStepStatus = "running"
	PlanStepStatusCompleted PlanStepStatus = "completed"
	PlanStepStatusFailed    PlanStepStatus = "failed"
	PlanStepStatusSkipped   PlanStepStatus = "skipped"
)

// VerificationResult validates the quality of a plan or step.
type VerificationResult struct {
	// IsValid indicates if the output meets requirements
	IsValid bool `json:"is_valid"`

	// Score is a 0-1 quality assessment
	Score float64 `json:"score"`

	// Issues lists problems found
	Issues []string `json:"issues,omitempty"`

	// Suggestions for improvement
	Suggestions []string `json:"suggestions,omitempty"`

	// Reasoning explains the verification assessment
	Reasoning string `json:"reasoning,omitempty"`

	// Timestamp when verification was performed
	Timestamp time.Time `json:"timestamp"`
}

// CanExecute checks if a step's dependencies are satisfied.
func (s *PlanStep) CanExecute(completedSteps map[string]bool) bool {
	if s.Status != PlanStepStatusPending {
		return false
	}

	for _, depID := range s.Dependencies {
		if !completedSteps[depID] {
			return false
		}
	}

	return true
}

// MarkCompleted marks the step as completed with result.
func (s *PlanStep) MarkCompleted(result any) {
	s.Status = PlanStepStatusCompleted
	s.Result = result
	s.EndTime = time.Now()
}

// MarkFailed marks the step as failed with error.
func (s *PlanStep) MarkFailed(err error) {
	s.Status = PlanStepStatusFailed
	if err != nil {
		s.Error = err.Error()
	}
	s.EndTime = time.Now()
}

// ShouldRetry checks if the step can be retried.
func (s *PlanStep) ShouldRetry() bool {
	return s.Status == PlanStepStatusFailed && s.Retries < s.MaxRetries
}

// Progress returns the completion percentage (0-100).
func (p *Plan) Progress() float64 {
	if len(p.Steps) == 0 {
		return 0
	}

	completed := 0
	for _, step := range p.Steps {
		if step.Status == PlanStepStatusCompleted {
			completed++
		}
	}

	return float64(completed) / float64(len(p.Steps)) * 100
}

// GetPendingSteps returns steps that are ready to execute.
func (p *Plan) GetPendingSteps() []PlanStep {
	completedSteps := make(map[string]bool)
	for _, step := range p.Steps {
		if step.Status == PlanStepStatusCompleted {
			completedSteps[step.ID] = true
		}
	}

	var pending []PlanStep
	for _, step := range p.Steps {
		if step.CanExecute(completedSteps) {
			pending = append(pending, step)
		}
	}

	return pending
}

// ToJSON serializes the plan to JSON.
func (p *Plan) ToJSON() ([]byte, error) {
	return json.Marshal(p)
}

// PlanFromJSON deserializes a plan from JSON.
func PlanFromJSON(data []byte) (*Plan, error) {
	var plan Plan
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal plan: %w", err)
	}
	return &plan, nil
}

// Clone creates a deep copy of the plan.
func (p *Plan) Clone() *Plan {
	clone := &Plan{
		ID:           p.ID,
		AgentID:      p.AgentID,
		Goal:         p.Goal,
		Steps:        make([]PlanStep, len(p.Steps)),
		Status:       p.Status,
		CreatedAt:    p.CreatedAt,
		UpdatedAt:    p.UpdatedAt,
		Version:      p.Version,
		ParentPlanID: p.ParentPlanID,
	}

	// Deep copy steps
	for i, step := range p.Steps {
		clone.Steps[i] = PlanStep{
			ID:           step.ID,
			Index:        step.Index,
			Description:  step.Description,
			ToolsNeeded:  append([]string{}, step.ToolsNeeded...),
			Dependencies: append([]string{}, step.Dependencies...),
			Status:       step.Status,
			Result:       step.Result,
			Error:        step.Error,
			Retries:      step.Retries,
			MaxRetries:   step.MaxRetries,
			StartTime:    step.StartTime,
			EndTime:      step.EndTime,
		}

		if step.Verification != nil {
			clone.Steps[i].Verification = &VerificationResult{
				IsValid:     step.Verification.IsValid,
				Score:       step.Verification.Score,
				Issues:      append([]string{}, step.Verification.Issues...),
				Suggestions: append([]string{}, step.Verification.Suggestions...),
				Reasoning:   step.Verification.Reasoning,
				Timestamp:   step.Verification.Timestamp,
			}
		}

		if step.Metadata != nil {
			clone.Steps[i].Metadata = make(map[string]any)
			for k, v := range step.Metadata {
				clone.Steps[i].Metadata[k] = v
			}
		}
	}

	if p.Metadata != nil {
		clone.Metadata = make(map[string]any)
		for k, v := range p.Metadata {
			clone.Metadata[k] = v
		}
	}

	return clone
}

