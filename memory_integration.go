package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// StoreReasoningTrace stores a ReAct reasoning trace in episodic memory.
func StoreReasoningTrace(ctx context.Context, mm *MemoryManager, trace ReasoningTrace, agentID, executionID string) error {
	if mm == nil {
		return nil // No memory manager, skip storage
	}

	// Serialize trace to JSON for storage
	traceJSON, err := json.Marshal(trace)
	if err != nil {
		return fmt.Errorf("failed to serialize reasoning trace: %w", err)
	}

	// Create content string with key information
	content := fmt.Sprintf("Reasoning Step %d: %s | Action: %s | Observation: %s | Reflection: %s",
		trace.Step, trace.Thought, trace.Action, trace.Observation, trace.Reflection)

	// Store with metadata
	metadata := map[string]any{
		"type":         "reasoning_trace",
		"agent_id":     agentID,
		"execution_id": executionID,
		"step":         trace.Step,
		"action":       trace.Action,
		"confidence":   trace.Confidence,
		"timestamp":    trace.Timestamp,
		"trace_json":   string(traceJSON),
	}

	_, err = mm.Store(ctx, content, metadata, trace.Confidence)
	return err
}

// StorePlan stores a plan in episodic memory.
func StorePlan(ctx context.Context, mm *MemoryManager, plan *Plan) error {
	if mm == nil {
		return nil // No memory manager, skip storage
	}

	// Serialize plan to JSON
	planJSON, err := json.Marshal(plan)
	if err != nil {
		return fmt.Errorf("failed to serialize plan: %w", err)
	}

	// Create content string with plan summary
	content := fmt.Sprintf("Plan %s: %s | Steps: %d | Status: %s",
		plan.ID, plan.Goal, len(plan.Steps), plan.Status)

	// Calculate importance based on plan status and complexity
	importance := 0.5
	switch plan.Status {
	case PlanStatusCompleted:
		importance = 0.8
	case PlanStatusFailed:
		importance = 0.6 // Failed plans are important for learning
	}
	importance += float64(len(plan.Steps)) * 0.02 // More complex plans are more important

	if importance > 1.0 {
		importance = 1.0
	}

	// Store with metadata
	metadata := map[string]any{
		"type":       "plan",
		"agent_id":   plan.AgentID,
		"plan_id":    plan.ID,
		"goal":       plan.Goal,
		"status":     string(plan.Status),
		"steps":      len(plan.Steps),
		"version":    plan.Version,
		"created_at": plan.CreatedAt,
		"updated_at": plan.UpdatedAt,
		"plan_json":  string(planJSON),
	}

	_, err = mm.Store(ctx, content, metadata, importance)
	return err
}

// StorePlanStep stores a single plan step in memory.
func StorePlanStep(ctx context.Context, mm *MemoryManager, step *PlanStep, planID, agentID string) error {
	if mm == nil {
		return nil
	}

	// Create content string
	content := fmt.Sprintf("Plan Step %d: %s | Status: %s | Result: %v",
		step.Index, step.Description, step.Status, step.Result)

	// Calculate importance
	importance := 0.5
	switch step.Status {
	case PlanStepStatusCompleted:
		importance = 0.7
	case PlanStepStatusFailed:
		importance = 0.8 // Failed steps are important for learning
	}

	if step.Verification != nil {
		importance = (importance + step.Verification.Score) / 2
	}

	// Store with metadata
	metadata := map[string]any{
		"type":        "plan_step",
		"agent_id":    agentID,
		"plan_id":     planID,
		"step_id":     step.ID,
		"step_index":  step.Index,
		"description": step.Description,
		"status":      string(step.Status),
		"tools_used":  step.ToolsNeeded,
	}

	// Add duration if available
	if !step.StartTime.IsZero() && !step.EndTime.IsZero() {
		metadata["duration"] = step.EndTime.Sub(step.StartTime).Seconds()
	}

	if step.Error != "" {
		metadata["error"] = step.Error
	}

	_, err := mm.Store(ctx, content, metadata, importance)
	return err
}

// RecallSimilarTraces retrieves reasoning traces similar to the given query.
func RecallSimilarTraces(ctx context.Context, mm *MemoryManager, query string, limit int) ([]ReasoningTrace, error) {
	if mm == nil {
		return nil, nil
	}

	// Recall memories with type filter
	memories, err := mm.Recall(ctx, query, MemoryTierEpisodic, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to recall traces: %w", err)
	}

	traces := make([]ReasoningTrace, 0, len(memories))
	for _, mem := range memories {
		// Check if this is a reasoning trace
		if memType, ok := mem.Metadata["type"].(string); ok && memType == "reasoning_trace" {
			// Try to deserialize from stored JSON
			if traceJSON, ok := mem.Metadata["trace_json"].(string); ok {
				var trace ReasoningTrace
				if err := json.Unmarshal([]byte(traceJSON), &trace); err == nil {
					traces = append(traces, trace)
				}
			}
		}
	}

	return traces, nil
}

// RecallSimilarPlans retrieves plans similar to the given query.
func RecallSimilarPlans(ctx context.Context, mm *MemoryManager, query string, limit int) ([]*Plan, error) {
	if mm == nil {
		return nil, nil
	}

	// Recall memories with type filter
	memories, err := mm.Recall(ctx, query, MemoryTierEpisodic, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to recall plans: %w", err)
	}

	plans := make([]*Plan, 0, len(memories))
	for _, mem := range memories {
		// Check if this is a plan
		if memType, ok := mem.Metadata["type"].(string); ok && memType == "plan" {
			// Try to deserialize from stored JSON
			if planJSON, ok := mem.Metadata["plan_json"].(string); ok {
				var plan Plan
				if err := json.Unmarshal([]byte(planJSON), &plan); err == nil {
					plans = append(plans, &plan)
				}
			}
		}
	}

	return plans, nil
}

// RecallRecentTraces retrieves the most recent reasoning traces.
func RecallRecentTraces(ctx context.Context, mm *MemoryManager, agentID string, limit int) ([]ReasoningTrace, error) {
	if mm == nil {
		return nil, nil
	}

	// Get all memories for this agent
	memories, err := mm.Recall(ctx, fmt.Sprintf("agent:%s reasoning", agentID), MemoryTierEpisodic, limit*2)
	if err != nil {
		return nil, fmt.Errorf("failed to recall recent traces: %w", err)
	}

	traces := make([]ReasoningTrace, 0, limit)
	for _, mem := range memories {
		if memType, ok := mem.Metadata["type"].(string); ok && memType == "reasoning_trace" {
			if traceJSON, ok := mem.Metadata["trace_json"].(string); ok {
				var trace ReasoningTrace
				if err := json.Unmarshal([]byte(traceJSON), &trace); err == nil {
					traces = append(traces, trace)
					if len(traces) >= limit {
						break
					}
				}
			}
		}
	}

	return traces, nil
}

// RecallRecentPlans retrieves the most recent plans for an agent.
func RecallRecentPlans(ctx context.Context, mm *MemoryManager, agentID string, limit int) ([]*Plan, error) {
	if mm == nil {
		return nil, nil
	}

	// Get all memories for this agent
	memories, err := mm.Recall(ctx, fmt.Sprintf("agent:%s plan", agentID), MemoryTierEpisodic, limit*2)
	if err != nil {
		return nil, fmt.Errorf("failed to recall recent plans: %w", err)
	}

	plans := make([]*Plan, 0, limit)
	for _, mem := range memories {
		if memType, ok := mem.Metadata["type"].(string); ok && memType == "plan" {
			if planJSON, ok := mem.Metadata["plan_json"].(string); ok {
				var plan Plan
				if err := json.Unmarshal([]byte(planJSON), &plan); err == nil {
					plans = append(plans, &plan)
					if len(plans) >= limit {
						break
					}
				}
			}
		}
	}

	return plans, nil
}

// RecallSuccessfulPlans retrieves completed plans similar to the query.
func RecallSuccessfulPlans(ctx context.Context, mm *MemoryManager, query string, limit int) ([]*Plan, error) {
	plans, err := RecallSimilarPlans(ctx, mm, query, limit*2)
	if err != nil {
		return nil, err
	}

	// Filter for successful plans
	successful := make([]*Plan, 0, limit)
	for _, plan := range plans {
		if plan.Status == PlanStatusCompleted {
			successful = append(successful, plan)
			if len(successful) >= limit {
				break
			}
		}
	}

	return successful, nil
}

// RecallFailedPlans retrieves failed plans to learn from mistakes.
func RecallFailedPlans(ctx context.Context, mm *MemoryManager, query string, limit int) ([]*Plan, error) {
	plans, err := RecallSimilarPlans(ctx, mm, query, limit*2)
	if err != nil {
		return nil, err
	}

	// Filter for failed plans
	failed := make([]*Plan, 0, limit)
	for _, plan := range plans {
		if plan.Status == PlanStatusFailed {
			failed = append(failed, plan)
			if len(failed) >= limit {
				break
			}
		}
	}

	return failed, nil
}

// StoreReflection stores a reflection result in memory.
func StoreReflection(ctx context.Context, mm *MemoryManager, reflection ReflectionResult, agentID, executionID string, stepIndex int) error {
	if mm == nil {
		return nil
	}

	// Calculate importance from quality
	importance := 0.5
	switch reflection.Quality {
	case "good":
		importance = 0.7
	case "needs_improvement":
		importance = 0.8 // Important to learn from
	case "invalid":
		importance = 0.9 // Very important to avoid repeating
	}

	// Create content string
	content := fmt.Sprintf("Reflection on Step %d: Quality=%s | Issues: %v | Suggestions: %v",
		stepIndex, reflection.Quality, reflection.Issues, reflection.Suggestions)

	// Store with metadata
	metadata := map[string]any{
		"type":          "reflection",
		"agent_id":      agentID,
		"execution_id":  executionID,
		"step_index":    stepIndex,
		"quality":       reflection.Quality,
		"should_replan": reflection.ShouldReplan,
		"timestamp":     time.Now(),
	}

	_, err := mm.Store(ctx, content, metadata, importance)
	return err
}

// RecallReflections retrieves reflection results for analysis.
func RecallReflections(ctx context.Context, mm *MemoryManager, agentID string, limit int) ([]ReflectionResult, error) {
	if mm == nil {
		return nil, nil
	}

	memories, err := mm.Recall(ctx, fmt.Sprintf("agent:%s reflection", agentID), MemoryTierEpisodic, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to recall reflections: %w", err)
	}

	reflections := make([]ReflectionResult, 0, len(memories))
	for _, mem := range memories {
		if memType, ok := mem.Metadata["type"].(string); ok && memType == "reflection" {
			reflection := ReflectionResult{}

			if quality, ok := mem.Metadata["quality"].(string); ok {
				reflection.Quality = quality
			}
			if shouldReplan, ok := mem.Metadata["should_replan"].(bool); ok {
				reflection.ShouldReplan = shouldReplan
			}

			reflections = append(reflections, reflection)
		}
	}

	return reflections, nil
}

// CreateEpisodicMemory creates an episodic memory for an agent execution.
func CreateEpisodicMemory(ctx context.Context, mm *MemoryManager, execution *AgentExecution, title string) error {
	if mm == nil {
		return nil
	}

	// Collect all memory IDs from this execution
	memoryIDs := make([]string, 0)

	// Store each step as a memory and collect IDs
	for _, step := range execution.Steps {
		content := fmt.Sprintf("Step %d: %s -> %s", step.Index, step.Input, step.Output)
		metadata := map[string]any{
			"type":         "execution_step",
			"agent_id":     execution.AgentID,
			"execution_id": execution.ID,
			"step_index":   step.Index,
			"state":        string(step.State),
		}

		entry, err := mm.Store(ctx, content, metadata, 0.6)
		if err == nil && entry != nil {
			memoryIDs = append(memoryIDs, entry.ID)
		}
	}

	// Store episodic memory as a high-level memory entry
	episodeContent := fmt.Sprintf("Execution Episode: %s | Steps: %d | Status: %s | Output: %s",
		title, len(execution.Steps), execution.Status, execution.FinalOutput)

	episodeMetadata := map[string]any{
		"type":         "episode",
		"agent_id":     execution.AgentID,
		"execution_id": execution.ID,
		"title":        title,
		"status":       string(execution.Status),
		"step_count":   len(execution.Steps),
		"memory_ids":   memoryIDs,
		"timestamp":    execution.StartTime,
		"duration":     execution.EndTime.Sub(execution.StartTime).Seconds(),
	}

	_, err := mm.Store(ctx, episodeContent, episodeMetadata, 0.7)
	return err
}

// GetExecutionContext retrieves relevant context for an execution from memory.
func GetExecutionContext(ctx context.Context, mm *MemoryManager, query string, agentID string) (string, error) {
	if mm == nil {
		return "", nil
	}

	// Recall relevant memories
	memories, err := mm.Recall(ctx, query, MemoryTierEpisodic, 5)
	if err != nil {
		return "", err
	}

	if len(memories) == 0 {
		return "", nil
	}

	// Build context string
	var contextBuilder string
	contextBuilder += "Relevant past experiences:\n"
	for i, mem := range memories {
		contextBuilder += fmt.Sprintf("%d. %s (importance: %.2f)\n", i+1, mem.Content, mem.Importance)
	}

	return contextBuilder, nil
}
