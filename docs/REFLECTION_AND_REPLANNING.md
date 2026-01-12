# Reflection and Replanning

Advanced self-improvement capabilities for ReAct and Plan-Execute agents through quality evaluation, learning from failures, and intelligent replanning.

## Table of Contents
- [Overview](#overview)
- [Reflection System](#reflection-system)
- [Replanning Engine](#replanning-engine)
- [Plan Verification](#plan-verification)
- [Memory Integration](#memory-integration)
- [Configuration](#configuration)
- [Examples](#examples)
- [Best Practices](#best-practices)

---

## Overview

### What are Reflection and Replanning?

These are self-improvement mechanisms that allow agents to:

1. **Reflect**: Evaluate the quality of their own reasoning and plans
2. **Learn**: Store and recall past experiences to avoid mistakes
3. **Replan**: Create improved plans based on failures and learnings

### Architecture

```
┌────────────────────────────────────────────────────────────┐
│                  Self-Improvement System                    │
├────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────────┐    ┌─────────────────┐              │
│  │ ReflectionEngine │    │  ReplanEngine   │              │
│  │                  │    │                 │              │
│  │ - EvaluateStep   │───▶│ - ShouldReplan  │              │
│  │ - EvaluateTrace  │    │ - Replan        │              │
│  │ - EvaluatePlan   │    │ - LearnPattern  │              │
│  └──────────────────┘    └─────────────────┘              │
│          │                        │                         │
│          └──────┬─────────────────┘                         │
│                 ▼                                           │
│       ┌─────────────────┐                                  │
│       │  PlanVerifier   │                                  │
│       │                 │                                  │
│       │ - VerifyStep    │                                  │
│       │ - VerifyStructure│                                 │
│       │ - VerifyQuality │                                  │
│       └─────────────────┘                                  │
│                 │                                           │
│                 ▼                                           │
│       ┌─────────────────┐                                  │
│       │ MemoryManager   │                                  │
│       │                 │                                  │
│       │ - Store traces  │                                  │
│       │ - Store plans   │                                  │
│       │ - Recall similar│                                  │
│       └─────────────────┘                                  │
│                                                              │
└────────────────────────────────────────────────────────────┘
```

---

## Reflection System

### Purpose

The Reflection Engine evaluates the quality of agent reasoning and plans to:
- Detect logical errors
- Assess action appropriateness
- Measure completeness
- Identify improvement opportunities

### ReflectionEngine

```go
type ReflectionEngine struct {
    llmManager       LLMManager
    qualityThreshold float64
    criteria         []ReflectionCriterion
}

type ReflectionCriterion struct {
    Name        string  // e.g., "Logical Coherence"
    Description string
    Weight      float64 // 0-1, importance
}

type ReflectionResult struct {
    Quality       string   // "good", "needs_improvement", "invalid"
    Score         float64  // 0-1 quality score
    Issues        []string // Identified problems
    Suggestions   []string // How to improve
    ShouldReplan  bool     // Trigger replanning
    Reasoning     string   // Detailed explanation
}
```

### Creating a Reflection Engine

```go
reflectionEngine := sdk.NewReflectionEngine(logger, metrics, &sdk.ReflectionEngineConfig{
    LLMManager:       llmManager,
    Provider:         "openai",
    Model:            "gpt-4",
    QualityThreshold: 0.7, // Minimum acceptable score
    Criteria: []sdk.ReflectionCriterion{
        {
            Name:        "Logical Coherence",
            Description: "Is the reasoning logically sound?",
            Weight:      0.3,
        },
        {
            Name:        "Action Appropriateness",
            Description: "Is the chosen action suitable?",
            Weight:      0.25,
        },
        {
            Name:        "Completeness",
            Description: "Is the solution complete?",
            Weight:      0.2,
        },
        {
            Name:        "Efficiency",
            Description: "Is the approach efficient?",
            Weight:      0.15,
        },
        {
            Name:        "Error Handling",
            Description: "Are errors properly handled?",
            Weight:      0.1,
        },
    },
})
```

### Evaluating Steps

```go
// Evaluate a single reasoning step
result, err := reflectionEngine.EvaluateStep(ctx, step, history)

if result.Quality == "invalid" {
    log.Error("Invalid reasoning detected", "issues", result.Issues)
    // Take corrective action
}

if result.Score < 0.7 {
    log.Warn("Low quality reasoning", 
        "score", result.Score,
        "suggestions", result.Suggestions)
}
```

### Evaluating Plans

```go
// Evaluate entire plan
result, err := reflectionEngine.EvaluatePlan(ctx, plan)

if reflectionEngine.ShouldReplan(result) {
    // Trigger replanning
    log.Info("Replanning triggered", 
        "reason", result.Quality,
        "score", result.Score)
}
```

### Example Reflection

```
Input Step:
  Thought: "I should search for quantum computing"
  Action: search
  Observation: "Found 1000 results about quantum computing"

Reflection Result:
  Quality: needs_improvement
  Score: 0.65
  Issues:
    - Search query too broad, results not actionable
    - No clear next step identified
    - Observation doesn't help answer original question
  Suggestions:
    - Narrow search to specific aspect (e.g., "recent quantum computing breakthroughs")
    - Define what information is needed from results
    - Plan follow-up actions
  ShouldReplan: false
  Reasoning: "The agent's approach is on the right track but needs refinement.
             The search query is too broad to yield useful results. A more
             specific query would lead to better outcomes."
```

### Default Criteria

The system comes with sensible defaults:

```go
Default Criteria:
1. Logical Coherence (30%) - Is reasoning sound?
2. Action Appropriateness (25%) - Is action suitable?
3. Completeness (20%) - Is solution complete?
4. Efficiency (15%) - Is approach efficient?
5. Error Handling (10%) - Are errors handled?
```

---

## Replanning Engine

### Purpose

The Replanning Engine creates improved plans when:
- Original plan fails
- Quality is insufficient
- Better approach is identified

### ReplanEngine

```go
type ReplanEngine struct {
    llmManager      LLMManager
    memoryManager   *MemoryManager
    triggers        []ReplanTrigger
    learningEnabled bool
    failurePatterns map[string]int // Track patterns
}

type ReplanTrigger struct {
    Name        string
    Description string
    Condition   func(*Plan, *ReflectionResult) bool
    Priority    int // Higher priority triggers first
}
```

### Creating a Replan Engine

```go
replanEngine := sdk.NewReplanEngine(logger, metrics, &sdk.ReplanEngineConfig{
    LLMManager:        llmManager,
    Provider:          "openai",
    Model:             "gpt-4",
    MemoryManager:     memoryManager,
    LearningEnabled:   true, // Track failure patterns
    MaxReplanAttempts: 3,
    Triggers: []sdk.ReplanTrigger{
        {
            Name:     "InvalidPlan",
            Priority: 100,
            Condition: func(plan *Plan, refl *ReflectionResult) bool {
                return refl != nil && refl.Quality == "invalid"
            },
        },
        {
            Name:     "LowQuality",
            Priority: 90,
            Condition: func(plan *Plan, refl *ReflectionResult) bool {
                return refl != nil && refl.Score < 0.5
            },
        },
        {
            Name:     "MultipleFailures",
            Priority: 80,
            Condition: func(plan *Plan, refl *ReflectionResult) bool {
                failedCount := 0
                for _, step := range plan.Steps {
                    if step.Status == sdk.PlanStepStatusFailed {
                        failedCount++
                    }
                }
                return failedCount > len(plan.Steps)/2
            },
        },
    },
})
```

### Checking if Replanning Needed

```go
shouldReplan, trigger := replanEngine.ShouldReplan(ctx, plan, reflection)

if shouldReplan {
    log.Info("Replanning triggered",
        "trigger", trigger,
        "plan_id", plan.ID)
    
    // Create new plan
    newPlan, err := replanEngine.Replan(ctx, plan, trigger, agentID, tools)
    if err != nil {
        // Handle replan failure
    }
}
```

### Replanning Process

```
1. ANALYZE FAILURE
   ├─ Examine failed steps
   ├─ Extract error messages
   ├─ Identify failure pattern
   └─ Run reflection on plan

2. RECALL EXPERIENCE
   ├─ Query memory for similar goals
   ├─ Get successful past plans
   ├─ Get failed plans to avoid
   └─ Review failure patterns

3. GENERATE NEW PLAN
   ├─ Build context with learnings
   ├─ LLM creates improved plan
   ├─ Preserve completed steps
   ├─ Adjust failed/pending steps
   └─ Add error handling

4. VALIDATE NEW PLAN
   ├─ Structural validation
   ├─ Quality check
   └─ Store in history

5. CONTINUE EXECUTION
   └─ Execute Plan v2
```

### Example Replan

```
Original Plan v1:
Goal: Build authentication system
Steps:
  1. Create database schema ✓
  2. Build API endpoints ✗ (Missing middleware)
  3. Add tests ○ (pending)

Failure Context: "Step 2 failed - authentication middleware not implemented"

Memory Recall:
  - Similar past plan: "Auth system v2" (successful)
    Key: Added middleware before API implementation
  - Failure pattern: "Missing dependency" (occurred 3 times)
    Common cause: Steps out of order

New Plan v2:
Goal: Build authentication system
Steps:
  1. Create database schema ✓ (preserved)
  2. Implement authentication middleware (new)
  3. Build API endpoints with auth (adjusted)
  4. Add integration tests (adjusted)
  5. Add error handling (new)

Changes:
  - Added middleware step before API
  - Modified API step to use middleware
  - Enhanced testing to cover auth flow
  - Added explicit error handling step
```

### Learning from Failures

```go
// Automatic learning
replanEngine.LearnFromFailure(ctx, failedPlan, failureContext)

// Query learned patterns
patterns := replanEngine.GetFailurePatterns()
for pattern, count := range patterns {
    fmt.Printf("Pattern: %s (occurred %d times)\n", pattern, count)
}

// Common patterns tracked:
// - "dependency_issue"
// - "tool_availability"
// - "timeout"
// - "incomplete_plan"
// - "invalid_logic"
// - "multiple_step_failures"
```

---

## Plan Verification

### Purpose

The Plan Verifier validates plans at two levels:
1. **Structural**: Dependencies, cycles, completeness
2. **Semantic**: Quality, feasibility, clarity

### PlanVerifier

```go
type PlanVerifier struct {
    llmManager       LLMManager
    rules            []VerificationRule
    qualityThreshold float64
    structuralChecks bool
    semanticChecks   bool
}

type VerificationResult struct {
    IsValid     bool
    Score       float64  // 0-1 quality score
    Issues      []string
    Suggestions []string
    Reasoning   string
}
```

### Creating a Verifier

```go
verifier := sdk.NewPlanVerifier(logger, metrics, &sdk.PlanVerifierConfig{
    LLMManager:       llmManager,
    Provider:         "openai",
    Model:            "gpt-4",
    QualityThreshold: 0.7,
    StructuralChecks: true, // Check dependencies, cycles
    SemanticChecks:   true, // Check quality, feasibility
})
```

### Structural Verification

```go
// Checks:
// - No empty plans
// - No circular dependencies
// - Valid dependency references
// - No orphaned steps

result, err := verifier.VerifyPlanStructure(ctx, plan)

if !result.IsValid {
    fmt.Printf("Structural issues: %v\n", result.Issues)
    // Example issues:
    // - "Step 3 has invalid dependency: step_99"
    // - "Circular dependency: Step 2 → Step 4 → Step 2"
}
```

### Semantic Verification

```go
// Checks:
// - Steps are clear and actionable
// - Tools appropriately assigned
// - Plan is complete for goal
// - Steps in logical order

result, err := verifier.VerifyPlanQuality(ctx, plan, plan.Goal)

if result.Score < 0.7 {
    fmt.Printf("Quality concerns: %v\n", result.Suggestions)
    // Example suggestions:
    // - "Add error handling step after API creation"
    // - "Include data validation before processing"
}
```

### Step Verification

```go
// Verify individual step output
result, err := verifier.VerifyStep(ctx, step, expectedOutcome)

if !result.IsValid {
    fmt.Printf("Step %s verification failed\n", step.ID)
    fmt.Printf("Score: %.2f, Issues: %v\n", result.Score, result.Issues)
    
    // Decide whether to replan
    if result.Score < 0.3 {
        // Critical failure, trigger replan
    }
}
```

---

## Memory Integration

### Storing Traces and Plans

```go
// Store reasoning trace (automatic in ReAct)
err := sdk.StoreReasoningTrace(ctx, memoryManager, trace, agentID, executionID)

// Store plan (automatic in Plan-Execute)
err := sdk.StorePlan(ctx, memoryManager, plan)

// Store step
err := sdk.StorePlanStep(ctx, memoryManager, step, planID, agentID)

// Store reflection
err := sdk.StoreReflection(ctx, memoryManager, reflection, agentID, executionID, stepIndex)
```

### Recalling Past Experience

```go
// Recall similar traces
traces, err := sdk.RecallSimilarTraces(ctx, memoryManager, "quantum computing research", 5)
for _, trace := range traces {
    fmt.Printf("Past: %s → %s\n", trace.Thought, trace.Action)
}

// Recall similar plans
plans, err := sdk.RecallSimilarPlans(ctx, memoryManager, "build auth system", 3)
for _, plan := range plans {
    fmt.Printf("Plan: %s (%d steps, status: %s)\n", 
        plan.Goal, len(plan.Steps), plan.Status)
}

// Recall successful plans only
successful, err := sdk.RecallSuccessfulPlans(ctx, memoryManager, "authentication", 5)

// Recall failed plans to avoid mistakes
failed, err := sdk.RecallFailedPlans(ctx, memoryManager, "authentication", 3)
```

### Learning Patterns

```go
// Get execution context from memory
context, err := sdk.GetExecutionContext(ctx, memoryManager, 
    "user authentication implementation", agentID)

// Context includes:
// - Similar past executions
// - Common success patterns
// - Known failure patterns
// - Relevant tools and approaches
```

---

## Configuration

### Integrated Configuration

```go
// For ReAct Agent with reflection
reactAgent, _ := sdk.NewReactAgentBuilder("agent").
    WithLLMManager(llmManager).
    WithMemoryManager(memoryManager).
    WithReflectionInterval(3). // Reflect every 3 steps
    WithConfidenceThreshold(0.7).
    Build()

// For Plan-Execute Agent with verification and replanning
planAgent, _ := sdk.NewPlanExecuteAgentBuilder("agent").
    WithLLMManager(llmManager).
    WithPlanStore(planStore).
    WithMemoryManager(memoryManager).
    WithAllowReplanning(true).
    WithVerifySteps(true).
    WithMaxReplanAttempts(3).
    Build()
```

### Standalone Usage

```go
// Use reflection engine independently
reflectionEngine := sdk.NewReflectionEngine(...)
result, _ := reflectionEngine.EvaluateStep(ctx, step, history)

// Use replan engine independently
replanEngine := sdk.NewReplanEngine(...)
if shouldReplan, _ := replanEngine.ShouldReplan(ctx, plan, reflection); shouldReplan {
    newPlan, _ := replanEngine.Replan(ctx, plan, context, agentID, tools)
}

// Use plan verifier independently
verifier := sdk.NewPlanVerifier(...)
result, _ := verifier.VerifyPlan(ctx, plan)
```

---

## Examples

### Example 1: ReAct with Custom Reflection

```go
// Create reflection engine with custom criteria
reflectionEngine := sdk.NewReflectionEngine(logger, metrics, &sdk.ReflectionEngineConfig{
    LLMManager: llmManager,
    Criteria: []sdk.ReflectionCriterion{
        {Name: "Accuracy", Weight: 0.4},
        {Name: "Relevance", Weight: 0.3},
        {Name: "Completeness", Weight: 0.3},
    },
    QualityThreshold: 0.75,
})

// Create ReAct agent
agent, _ := sdk.NewReactAgentBuilder("researcher").
    WithLLMManager(llmManager).
    WithMemoryManager(memoryManager).
    WithReflectionInterval(2).
    Build()

// Execute
execution, _ := agent.Execute(ctx, "Research topic")

// Manual reflection on specific steps
for _, step := range execution.Steps {
    if step.Index%2 == 0 { // Every other step
        result, _ := reflectionEngine.EvaluateStep(ctx, step, execution.Steps[:step.Index])
        
        if result.Score < 0.7 {
            log.Warn("Quality issue detected", 
                "step", step.Index,
                "score", result.Score,
                "issues", result.Issues)
        }
    }
}
```

### Example 2: Plan-Execute with Learning

```go
// Setup memory for learning
memoryManager := sdk.NewMemoryManager(...)

// Create plan-execute agent with replanning
agent, _ := sdk.NewPlanExecuteAgentBuilder("builder").
    WithLLMManager(llmManager).
    WithPlanStore(planStore).
    WithMemoryManager(memoryManager).
    WithAllowReplanning(true).
    WithMaxReplanAttempts(3).
    Build()

// First execution (may fail and replan)
execution1, _ := agent.Execute(ctx, "Build feature X")

// Memory now contains:
// - Plan v1 (original)
// - Plan v2 (if replanned)
// - Failure patterns
// - Success strategies

// Second execution benefits from learning
execution2, _ := agent.Execute(ctx, "Build similar feature Y")
// Agent recalls successful approach from feature X
// Avoids known pitfalls
```

### Example 3: Custom Replan Triggers

```go
// Create replan engine with custom triggers
replanEngine := sdk.NewReplanEngine(logger, metrics, &sdk.ReplanEngineConfig{
    LLMManager: llmManager,
    Triggers: []sdk.ReplanTrigger{
        {
            Name:     "HighCostDetected",
            Priority: 95,
            Condition: func(plan *Plan, refl *ReflectionResult) bool {
                // Custom logic: replan if cost is too high
                estimatedCost := calculatePlanCost(plan)
                return estimatedCost > 10.0
            },
        },
        {
            Name:     "TimeoutRisk",
            Priority: 85,
            Condition: func(plan *Plan, refl *ReflectionResult) bool {
                // Custom logic: replan if likely to timeout
                estimatedTime := calculatePlanTime(plan)
                return estimatedTime > 30*time.Minute
            },
        },
    },
})
```

---

## Best Practices

### 1. Reflection Frequency

```go
// High-stakes tasks: Frequent reflection
.WithReflectionInterval(2) // Every 2 steps

// Exploratory tasks: Moderate reflection
.WithReflectionInterval(5) // Every 5 steps

// Simple tasks: Infrequent or disabled
.WithReflectionInterval(0) // Disabled
```

### 2. Quality Thresholds

```go
// Critical systems: High threshold
qualityThreshold: 0.85

// Production: Balanced threshold
qualityThreshold: 0.70

// Development: Lower threshold
qualityThreshold: 0.60
```

### 3. Replanning Limits

```go
// Conservative: Few replans
.WithMaxReplanAttempts(1)

// Balanced: Moderate replanning
.WithMaxReplanAttempts(3)

// Aggressive: Many replans (higher cost)
.WithMaxReplanAttempts(5)
```

### 4. Memory Management

```go
// Enable memory for all agents that benefit from learning
.WithMemoryManager(memoryManager)

// Periodically clean old, low-importance memories
memoryManager.ConsolidateMemories(ctx)

// Query memory before execution for context
context, _ := sdk.GetExecutionContext(ctx, memoryManager, task, agentID)
```

### 5. Verification Strategy

```go
// Verify everything (critical systems)
.WithVerifySteps(true).
WithStructuralChecks(true).
WithSemanticChecks(true)

// Selective verification (balanced)
.WithVerifySteps(true).   // Verify steps
WithSemanticChecks(false)  // Skip semantic checks

// Fast execution (development)
.WithVerifySteps(false)    // No verification
```

---

## Performance Considerations

### Token Usage

Reflection and verification increase token usage:

```
Without reflection/verification:
- Planning: 1,000 tokens
- Execution: 5,000 tokens
- Total: 6,000 tokens

With reflection/verification:
- Planning: 1,000 tokens
- Execution: 5,000 tokens
- Reflection: 2,000 tokens (periodic)
- Verification: 1,500 tokens (per step)
- Replanning: 2,000 tokens (if triggered)
- Total: 9,500-13,500 tokens (58-125% increase)
```

### Optimization Strategies

```go
// 1. Use smaller models for reflection
reflectionConfig := &sdk.ReflectionEngineConfig{
    Model: "gpt-3.5-turbo", // Cheaper model
}

// 2. Reduce reflection frequency
.WithReflectionInterval(5) // Less frequent

// 3. Selective verification
// Only verify critical steps

// 4. Disable for non-critical paths
if isProduction {
    .WithVerifySteps(true)
} else {
    .WithVerifySteps(false)
}
```

---

## Next Steps

- [ReAct Agents Guide](./REACT_AGENTS.md)
- [Plan-Execute Agents Guide](./PLAN_EXECUTE_AGENTS.md)
- [Agent Patterns Overview](./AGENT_PATTERNS.md)
- [Memory System Documentation](./MEMORY.md)
- [Tool System Documentation](./TOOLS.md)

