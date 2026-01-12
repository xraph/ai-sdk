# Plan-Execute Agents

Plan-Execute agents break down complex tasks into structured plans, execute them systematically, and verify results. This pattern is ideal for multi-step projects where clear decomposition and progress tracking are important.

## Table of Contents
- [Overview](#overview)
- [How It Works](#how-it-works)
- [Creating a Plan-Execute Agent](#creating-a-plan-execute-agent)
- [Configuration Options](#configuration-options)
- [Planning Phase](#planning-phase)
- [Execution Phase](#execution-phase)
- [Verification Phase](#verification-phase)
- [Replanning](#replanning)
- [Examples](#examples)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

---

## Overview

### What is Plan-Execute?

Plan-Execute agents follow a three-phase approach:

```
1. PLAN: Decompose task into steps with dependencies
2. EXECUTE: Run steps (in parallel where possible)
3. VERIFY: Check quality and correctness
   └─> REPLAN: If needed, create improved plan
```

### When to Use Plan-Execute

✅ **Good for:**
- Complex multi-step projects
- Tasks with clear goals but unclear implementation
- When progress tracking is important
- Workflows requiring quality verification
- Tasks that benefit from parallel execution

❌ **Not ideal for:**
- Simple single-step tasks (use Basic Agent)
- Exploratory tasks with unknown goals (use ReAct)
- Time-critical operations
- Tasks requiring real-time adaptation

---

## How It Works

### Example Execution Flow

```
User Input: "Build a user authentication system"

PHASE 1: PLANNING
├─ LLM generates plan with 5 steps:
│  1. Design database schema
│  2. Implement user registration
│  3. Implement login with JWT
│  4. Add password reset functionality
│  5. Write integration tests
└─ Plan stored in PlanStore

PHASE 2: EXECUTION
├─ Step 1: Design database schema
│  ├─ Execute (using design tools)
│  ├─ Verify: Valid schema? ✓
│  └─ Store result
├─ Step 2 & 3: (Execute in parallel - no dependencies)
│  ├─ Step 2: User registration
│  └─ Step 3: Login with JWT
├─ Step 4: Password reset (depends on Step 2)
│  ├─ Execute
│  ├─ Verify: Failed! ✗
│  └─ Trigger replanning
├─ REPLAN: Create Plan v2
│  └─ Adjust Step 4, add error handling step
└─ Continue execution...

PHASE 3: FINAL VERIFICATION
├─ Verify complete plan
├─ Check all steps completed
└─ Return final output
```

### Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                    PlanExecuteAgent                           │
├──────────────────────────────────────────────────────────────┤
│  ┌────────────────────────────────────────────────────┐     │
│  │         PlanExecuteStrategy                        │     │
│  ├────────────────────────────────────────────────────┤     │
│  │  Phase 1: PLANNING                                 │     │
│  │    ├─> Planner LLM (create plan)                   │     │
│  │    └─> PlanVerifier (validate structure)           │     │
│  │                                                      │     │
│  │  Phase 2: EXECUTION (loop)                         │     │
│  │    ├─> Identify ready steps (check dependencies)   │     │
│  │    ├─> Execute steps in parallel                   │     │
│  │    ├─> Executor LLM (per step)                     │     │
│  │    ├─> Capture step results                        │     │
│  │    └─> Verify step (optional)                      │     │
│  │        └─> Verifier LLM                            │     │
│  │                                                      │     │
│  │  If failure && allow_replanning:                   │     │
│  │    ├─> ReflectionEngine (evaluate)                 │     │
│  │    ├─> ReplanEngine (create new plan)              │     │
│  │    └─> Retry with Plan v2                          │     │
│  │                                                      │     │
│  │  Phase 3: FINAL VERIFICATION                       │     │
│  │    ├─> Verify entire plan                          │     │
│  │    └─> Extract final output                        │     │
│  └────────────────────────────────────────────────────┘     │
│                                                               │
│  Components:                                                  │
│  - Agent (base functionality)                                │
│  - LLMManager (planning, execution, verification)           │
│  - PlanStore (plan persistence & history)                   │
│  - MemoryManager (learning from failures)                   │
│  - ReplanEngine (intelligent replanning)                    │
│  - PlanVerifier (quality checks)                            │
└──────────────────────────────────────────────────────────────┘
```

---

## Creating a Plan-Execute Agent

### Basic Example

```go
package main

import (
    "context"
    "fmt"
    "time"
    "github.com/xraph/ai-sdk"
)

func main() {
    // Setup dependencies
    llmManager := sdk.NewLLMManager(logger, metrics)
    planStore := sdk.NewInMemoryPlanStore()
    memoryManager := sdk.NewMemoryManager(agentID, embedder, vectorStore, logger, metrics, nil)
    
    // Define tools
    tools := []sdk.Tool{
        fileSystemTool,
        databaseTool,
        apiTool,
    }
    
    // Create Plan-Execute agent
    agent, err := sdk.NewPlanExecuteAgentBuilder("project_manager").
        WithModel("gpt-4").
        WithProvider("openai").
        WithSystemPrompt("You are an expert software architect.").
        WithTools(tools...).
        WithLLMManager(llmManager).
        WithPlanStore(planStore).
        WithMemoryManager(memoryManager).
        WithAllowReplanning(true).
        WithVerifySteps(true).
        WithMaxReplanAttempts(3).
        WithTimeout(30 * time.Minute).
        WithLogger(logger).
        WithMetrics(metrics).
        Build()
    
    if err != nil {
        panic(err)
    }
    
    // Execute
    execution, err := agent.Execute(context.Background(),
        "Build a REST API for user management with authentication")
    
    if err != nil {
        panic(err)
    }
    
    fmt.Println("Final Output:", execution.FinalOutput)
    
    // Inspect plan
    plan := agent.GetCurrentPlan()
    fmt.Printf("\nPlan: %s (Version: %d, Status: %s)\n", 
        plan.Goal, plan.Version, plan.Status)
    
    for _, step := range plan.Steps {
        fmt.Printf("\n%d. %s\n", step.Index+1, step.Description)
        fmt.Printf("   Status: %s\n", step.Status)
        fmt.Printf("   Tools: %v\n", step.ToolsNeeded)
        if step.Error != "" {
            fmt.Printf("   Error: %s\n", step.Error)
        }
        if step.Verification != nil {
            fmt.Printf("   Verification: %v (Score: %.2f)\n", 
                step.Verification.IsValid, step.Verification.Score)
        }
    }
    
    // Check if replanning occurred
    history := agent.GetPlanHistory()
    if len(history) > 1 {
        fmt.Printf("\nReplanned %d time(s)\n", len(history)-1)
    }
}
```

---

## Configuration Options

### Builder Methods

```go
agent, err := sdk.NewPlanExecuteAgentBuilder("agent_name").
    // Required
    WithLLMManager(llmManager).
    
    // Model configuration
    WithModel("gpt-4").                    // Default LLM for all phases
    WithProvider("openai").
    WithTemperature(0.7).
    
    // Separate LLMs for different phases (optional)
    WithPlannerLLM(plannerLLM).           // Dedicated planner
    WithExecutorLLM(executorLLM).         // Dedicated executor
    WithVerifierLLM(verifierLLM).         // Dedicated verifier
    
    // Agent identity
    WithID("custom_id").
    WithDescription("Agent purpose").
    WithSystemPrompt("You are...").
    
    // Tools
    WithTools(tool1, tool2).
    
    // Plan-Execute specific
    WithAllowReplanning(true).            // Enable automatic replanning
    WithVerifySteps(true).                 // Verify each step
    WithMaxReplanAttempts(3).              // Max replanning attempts
    WithTimeout(30 * time.Minute).         // Total execution timeout
    
    // Storage
    WithPlanStore(planStore).              // Plan persistence
    WithMemoryManager(memoryManager).      // Learning from failures
    WithStateStore(stateStore).            // State persistence
    
    // Observability
    WithLogger(logger).
    WithMetrics(metrics).
    WithGuardrails(guardrails).
    
    Build()
```

### Configuration Profiles

```go
// For critical production tasks
.WithAllowReplanning(true).
WithVerifySteps(true).
WithMaxReplanAttempts(3).
WithTimeout(60 * time.Minute).
WithPlannerLLM(gpt4).          // Use best model for planning
WithVerifierLLM(gpt4)          // Use best model for verification

// For faster, cost-effective execution
.WithAllowReplanning(false).
WithVerifySteps(false).
WithTimeout(10 * time.Minute).
WithModel("gpt-3.5-turbo")     // Cheaper model

// For experimental/development
.WithAllowReplanning(true).
WithVerifySteps(true).
WithMaxReplanAttempts(5).
WithTimeout(120 * time.Minute)
```

---

## Planning Phase

### Plan Structure

```go
type Plan struct {
    ID        string       // Unique plan ID
    AgentID   string       // Agent that created it
    Goal      string       // Overall goal
    Steps     []PlanStep   // Ordered steps
    Status    PlanStatus   // pending, running, completed, failed
    Version   int          // Plan version (increments on replan)
    CreatedAt time.Time
    UpdatedAt time.Time
    Metadata  map[string]any
}

type PlanStep struct {
    ID           string
    Index        int
    Description  string      // What to do
    ToolsNeeded  []string    // Required tools
    Dependencies []string    // Step IDs that must complete first
    Status       PlanStepStatus
    Result       any         // Step output
    Error        string      // If failed
    Verification *VerificationResult
}
```

### Planning Prompt

The planner LLM receives:

```
Task: Build a user authentication system

Available tools:
- file_write: Write files to disk
- db_query: Execute database queries
- api_call: Make HTTP API calls
- code_generate: Generate code

Create a detailed plan to accomplish this task:
1. Break down into clear, sequential steps
2. Identify which tools each step needs
3. Specify dependencies between steps
4. Consider error handling

Return the plan in JSON format:
{
  "steps": [
    {
      "description": "Design user table schema",
      "tools": ["db_query"],
      "dependencies": []
    },
    {
      "description": "Create user registration endpoint",
      "tools": ["code_generate", "file_write"],
      "dependencies": ["step_0"]
    },
    ...
  ]
}
```

### Plan Validation

Before execution, the plan is validated:

```go
// Structural validation
- No empty plans
- No circular dependencies
- All dependencies reference existing steps
- No orphaned steps (except final step)

// Semantic validation (if enabled)
- Steps are clear and actionable
- Tool assignments are appropriate
- Plan is complete for the goal
- Steps are in logical order
```

---

## Execution Phase

### Step Execution Order

Steps execute based on dependencies:

```
Plan:
  Step 1: Setup (no deps) ────┐
  Step 2: Build A (no deps) ──┤
  Step 3: Build B (deps: 1) ──┤
  Step 4: Test (deps: 2, 3) ──┤
  Step 5: Deploy (deps: 4) ────┘

Execution:
  Time 0: Step 1 & 2 start (parallel)
  Time 1: Step 1 completes
  Time 2: Step 2 completes, Step 3 starts
  Time 3: Step 3 completes, Step 4 starts
  Time 4: Step 4 completes, Step 5 starts
  Time 5: Step 5 completes
```

### Parallel Execution

```go
// Steps without dependencies run in parallel
// This is handled automatically by the strategy

executePlan(plan) {
    for {
        // Get steps ready to execute
        readySteps := plan.GetPendingSteps() // No deps or deps met
        
        if len(readySteps) == 0 {
            break // All done or waiting
        }
        
        // Execute in parallel
        var wg sync.WaitGroup
        for _, step := range readySteps {
            wg.Add(1)
            go func(s PlanStep) {
                defer wg.Done()
                executeStep(s)
            }(step)
        }
        wg.Wait()
    }
}
```

### Step Context

Each step receives context from previous steps:

```
Original Goal: Build authentication system

Current Step 3: Create login endpoint

Context from previous steps:
Step 1 (Setup database): Created users table with columns: id, email, password_hash
Step 2 (Generate models): Created User model with validation

Execute the current step based on the above context and original goal.
```

---

## Verification Phase

### Step Verification

If `WithVerifySteps(true)`, each step is verified:

```go
type VerificationResult struct {
    IsValid     bool
    Score       float64  // 0-1 quality score
    Issues      []string
    Suggestions []string
    Reasoning   string
}

// Verification checks:
// 1. Does result match step description?
// 2. Is result complete and usable?
// 3. Are there errors or issues?
// 4. Is quality sufficient?
```

Example verification:

```
Step: Create login endpoint
Result: [generated code for /api/login endpoint]

Verification:
  IsValid: true
  Score: 0.85
  Issues: []
  Suggestions: ["Consider adding rate limiting", "Add input sanitization"]
  Reasoning: "Endpoint correctly implements login with JWT. Code is well-structured.
             Minor improvements suggested for production readiness."
```

### Plan Verification

After all steps complete, the entire plan is verified:

```
Goal: Build authentication system

Completed Steps:
- Database schema ✓
- User registration ✓
- Login endpoint ✓
- Password reset ✓
- Integration tests ✓

Overall Verification:
  IsValid: true
  Score: 0.90
  Issues: []
  Suggestions: ["Add API documentation", "Consider OAuth2 support"]
  Reasoning: "All core authentication features implemented and tested.
             System is functional and ready for deployment."
```

---

## Replanning

### When Replanning Triggers

Replanning occurs when (if `WithAllowReplanning(true)`):

1. **Step Failure**: A step fails to execute
2. **Verification Failure**: Step verification score < threshold
3. **Multiple Failures**: > 50% of steps failed
4. **Explicit Flag**: Verification sets `ShouldReplan = true`
5. **Quality Issues**: Plan quality score < threshold

### Replanning Process

```
1. Analyze Failure
   ├─ Identify which steps failed
   ├─ Extract error messages
   └─ Run reflection on plan

2. Recall Past Experience
   ├─ Query memory for similar past plans
   ├─ Get successful approaches
   └─ Get failure patterns to avoid

3. Generate New Plan
   ├─ LLM creates improved plan
   ├─ Preserves completed steps
   ├─ Adjusts failed/pending steps
   └─ Adds error handling

4. Validate New Plan
   ├─ Structural validation
   ├─ Semantic validation
   └─ Store in plan history

5. Continue Execution
   └─ Execute Plan v2
```

### Example Replan

```
Original Plan v1:
1. Setup database ✓
2. Create API ✗ (Error: Missing authentication)
3. Add tests (pending)

Replan Reason: Step 2 failed - missing authentication middleware

New Plan v2:
1. Setup database ✓ (preserved)
2. Create authentication middleware (new)
3. Create API with auth (adjusted)
4. Add tests (preserved)

Result: Plan v2 executes successfully
```

### Replanning Configuration

```go
// Conservative (few replans, fail fast)
.WithAllowReplanning(true).
WithMaxReplanAttempts(1)

// Balanced (moderate replanning)
.WithAllowReplanning(true).
WithMaxReplanAttempts(3)

// Aggressive (many replans, maximize success)
.WithAllowReplanning(true).
WithMaxReplanAttempts(5)

// Disabled (no replanning)
.WithAllowReplanning(false)
```

---

## Examples

### Example 1: Software Project

```go
tools := []sdk.Tool{
    codeGeneratorTool,
    fileSystemTool,
    gitTool,
    testRunnerTool,
}

agent, _ := sdk.NewPlanExecuteAgentBuilder("dev_agent").
    WithModel("gpt-4").
    WithTools(tools...).
    WithLLMManager(llmManager).
    WithPlanStore(planStore).
    WithAllowReplanning(true).
    WithVerifySteps(true).
    Build()

execution, _ := agent.Execute(ctx, `
    Create a RESTful API for a blog system with:
    - Posts (CRUD operations)
    - Comments (nested under posts)
    - User authentication
    - Input validation
    - Unit tests
`)

// Agent creates and executes plan:
// 1. Design data models
// 2. Setup database migrations
// 3. Implement Posts API
// 4. Implement Comments API
// 5. Add authentication middleware
// 6. Add input validation
// 7. Write unit tests
// 8. Integration testing
```

### Example 2: Data Processing Pipeline

```go
tools := []sdk.Tool{
    dataFetchTool,
    dataCleanTool,
    dataTransformTool,
    dataAnalysisTool,
    dataVisualizationTool,
}

agent, _ := sdk.NewPlanExecuteAgentBuilder("data_engineer").
    WithTools(tools...).
    WithTimeout(60 * time.Minute).
    Build()

execution, _ := agent.Execute(ctx, `
    Build a data pipeline that:
    1. Fetches sales data from API
    2. Cleans and validates data
    3. Aggregates by region and product
    4. Calculates trends
    5. Generates visualizations
    6. Exports report
`)

// Parallel execution where possible:
// Step 1: Fetch → Step 2: Clean → Step 3a: Aggregate by region
//                                  Step 3b: Aggregate by product
//                                  ↓
//                                  Step 4: Calculate trends
//                                  ↓
//                                  Step 5: Visualize
//                                  ↓
//                                  Step 6: Export
```

### Example 3: Research Project

```go
agent, _ := sdk.NewPlanExecuteAgentBuilder("researcher").
    WithTools(searchTool, summarizerTool, analysisTool).
    WithAllowReplanning(true).
    WithMaxReplanAttempts(2).
    Build()

execution, _ := agent.Execute(ctx, `
    Research the impact of AI on healthcare:
    - Literature review (last 5 years)
    - Current applications
    - Challenges and limitations
    - Future outlook
    - Comprehensive report with citations
`)

// Plan includes:
// 1. Search academic databases
// 2. Summarize key papers
// 3. Search for current applications
// 4. Analyze challenges
// 5. Synthesize future predictions
// 6. Compile report with citations
```

---

## Best Practices

### 1. Tool Design for Plans

```go
// ✅ Good: Granular, composable tools
tools := []Tool{
    {Name: "read_file", ...},
    {Name: "write_file", ...},
    {Name: "run_test", ...},
}

// ❌ Bad: Monolithic tools
tool := Tool{Name: "do_everything", ...}
```

### 2. Plan Granularity

```go
// ✅ Good: Clear, actionable steps
"Create user registration endpoint with email validation"

// ❌ Bad: Too vague
"Setup backend"

// ❌ Bad: Too granular
"Type the word 'function'"
```

### 3. Verification Strategy

```go
// For critical systems
.WithVerifySteps(true).
WithVerifierLLM(gpt4) // Use best model

// For development/testing
.WithVerifySteps(false) // Skip for speed

// For selective verification
// (implement custom logic to verify only critical steps)
```

### 4. Timeout Management

```go
// Set appropriate timeouts
.WithTimeout(30 * time.Minute) // Overall plan timeout

// Also set per-step timeouts in tools
Tool{
    Name: "api_call",
    Timeout: 30 * time.Second, // Per-step timeout
}
```

### 5. Memory Usage

```go
// Enable memory for learning
.WithMemoryManager(mm)

// After failures, query memory
failedPlans, _ := sdk.RecallFailedPlans(ctx, mm, plan.Goal, 3)
// Avoid past mistakes

// After success, store for future
_ = sdk.StorePlan(ctx, mm, successfulPlan)
```

---

## Troubleshooting

### Problem: Plans are Too Vague

**Symptoms**: Steps like "Setup system", "Do task"

**Solutions**:
```go
// 1. Improve system prompt
WithSystemPrompt(`You are a detailed planner. Each step should be:
- Specific and actionable
- Clearly describe what to do
- Specify which tools to use
Example: "Create user table with columns: id, email, password_hash using db_query tool"`)

// 2. Provide examples in prompt
// (modify strategy to include example plans)

// 3. Use better planning model
WithPlannerLLM(betterModel)
```

### Problem: Excessive Replanning

**Symptoms**: Many replan attempts, high costs

**Solutions**:
```go
// 1. Reduce max attempts
WithMaxReplanAttempts(2)

// 2. Disable step verification
WithVerifySteps(false)

// 3. Improve tool quality
// Ensure tools provide good results

// 4. Better error messages
// Tools should return actionable error info
```

### Problem: Steps Fail Due to Dependencies

**Symptoms**: "Tool not found", "Missing input"

**Solutions**:
```go
// 1. Validate plan before execution
// (automatic in strategy)

// 2. Improve planning prompt
// Emphasize dependency specification

// 3. Better tool descriptions
Tool{
    Name: "create_api",
    Description: "Creates API endpoint. Requires: database schema from db_setup tool",
}
```

### Problem: Long Execution Times

**Symptoms**: Timeouts, slow execution

**Solutions**:
```go
// 1. Disable verification
WithVerifySteps(false)

// 2. Optimize tools
// Ensure tools execute quickly

// 3. Reduce plan complexity
// Use smaller scope or break into sub-plans

// 4. Increase timeout
WithTimeout(60 * time.Minute)
```

---

## Performance Tuning

### Metrics to Track

```go
metrics.Histogram("plan_execute.plan_size").Observe(float64(len(plan.Steps)))
metrics.Histogram("plan_execute.replans").Observe(float64(replanCount))
metrics.Histogram("plan_execute.verification_score").Observe(verificationScore)
metrics.Histogram("plan_execute.execution_time").Observe(duration.Seconds())
metrics.Counter("plan_execute.step_failures").Inc()
```

### Optimization Checklist

- [ ] Tools are granular and focused
- [ ] Plan verification enabled only if needed
- [ ] Step verification selective
- [ ] Appropriate timeouts set
- [ ] Memory enabled for learning
- [ ] Replanning limits appropriate
- [ ] Error handling in tools
- [ ] Parallel execution opportunities identified
- [ ] Metrics collection enabled

---

## Next Steps

- [ReAct Agents](./REACT_AGENTS.md) - For exploratory tasks
- [Reflection & Replanning](./REFLECTION_AND_REPLANNING.md) - Deep dive into self-improvement
- [Agent Patterns Overview](./AGENT_PATTERNS.md) - Compare all patterns
- [Tool System](./TOOLS.md) - Creating effective tools
- [Memory System](./MEMORY.md) - Learning from experience

