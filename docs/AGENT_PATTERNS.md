# Agent Patterns in Forge AI SDK

This guide provides an overview of the agent patterns available in Forge AI SDK and when to use each one.

## Overview

Forge AI SDK provides three main agent patterns, each optimized for different types of tasks:

| Pattern | Best For | Key Features | Complexity |
|---------|----------|--------------|------------|
| **Basic Agent** | Simple tasks, conversation | Stateful, memory, tool calls | Low |
| **ReAct Agent** | Research, exploration, iterative reasoning | Thought → Action → Observation loop | Medium |
| **Plan-Execute Agent** | Complex multi-step tasks, project management | Planning → Execution → Verification | High |

---

## 1. Basic Agent

### When to Use
- Simple conversational interfaces
- Single-step tool execution
- Straightforward Q&A systems
- Basic task automation

### Characteristics
- Single execution loop
- Stateful with memory
- Tool calling support
- Conversation history

### Example
```go
agent, _ := sdk.NewAgentBuilder().
    WithName("assistant").
    WithModel("gpt-4").
    WithProvider("openai").
    WithTools(tools...).
    WithLLMManager(llmManager).
    WithStateStore(stateStore).
    Build()

result, _ := agent.Execute(ctx, "What's the weather in NYC?")
```

### Pros
- ✅ Simple and straightforward
- ✅ Low latency
- ✅ Predictable behavior
- ✅ Easy to debug

### Cons
- ❌ Limited reasoning capability
- ❌ No self-reflection
- ❌ Single-shot execution

---

## 2. ReAct Agent (Reasoning + Acting)

### When to Use
- Research and information gathering
- Tasks requiring iterative exploration
- Problems with unclear solution paths
- When you need transparent reasoning

### Characteristics
- Iterative thought → action → observation loop
- Self-reflection at configurable intervals
- Confidence tracking per step
- Memory integration for learning

### Example
```go
agent, _ := sdk.NewReactAgentBuilder("researcher").
    WithModel("gpt-4").
    WithTools(searchTool, calculatorTool).
    WithLLMManager(llmManager).
    WithMemoryManager(memoryManager).
    WithMaxIterations(10).
    WithReflectionInterval(3).
    Build()

execution, _ := agent.Execute(ctx, "Research quantum computing trends")

// Access reasoning traces
for _, trace := range agent.GetTraces() {
    fmt.Printf("Step %d: %s → %s → %s\n", 
        trace.Step, trace.Thought, trace.Action, trace.Observation)
}
```

### Pros
- ✅ Transparent reasoning process
- ✅ Self-correcting through reflection
- ✅ Adaptable to new information
- ✅ Learns from experience via memory
- ✅ Confidence-based decision making

### Cons
- ❌ Higher token usage (multiple LLM calls)
- ❌ Slower than basic agents
- ❌ May get stuck in loops without proper configuration
- ❌ Requires careful prompt engineering

### Configuration Tips
```go
// For exploratory research
.WithMaxIterations(15).
WithReflectionInterval(5).
WithConfidenceThreshold(0.6)

// For focused tasks
.WithMaxIterations(5).
WithReflectionInterval(2).
WithConfidenceThreshold(0.8)
```

---

## 3. Plan-Execute Agent

### When to Use
- Complex multi-step projects
- Tasks with clear goals but unclear steps
- When you need progress tracking
- Workflows requiring verification

### Characteristics
- Upfront planning phase
- Parallel step execution (where possible)
- Automatic verification
- Intelligent replanning on failures
- Plan versioning and history

### Example
```go
agent, _ := sdk.NewPlanExecuteAgentBuilder("project_manager").
    WithModel("gpt-4").
    WithTools(fileTool, apiTool, dbTool).
    WithLLMManager(llmManager).
    WithPlanStore(planStore).
    WithMemoryManager(memoryManager).
    WithAllowReplanning(true).
    WithVerifySteps(true).
    WithMaxReplanAttempts(3).
    Build()

execution, _ := agent.Execute(ctx, "Build authentication system")

// Access plan
plan := agent.GetCurrentPlan()
for _, step := range plan.Steps {
    fmt.Printf("%s: %s (Status: %s)\n", 
        step.ID, step.Description, step.Status)
}
```

### Pros
- ✅ Structured approach to complex tasks
- ✅ Clear progress tracking
- ✅ Automatic replanning on failures
- ✅ Verification of outputs
- ✅ Parallel execution of independent steps
- ✅ Plan history for learning

### Cons
- ❌ Highest token usage (planning + execution + verification)
- ❌ Slowest of the three patterns
- ❌ Overhead for simple tasks
- ❌ Requires good tool design

### Configuration Tips
```go
// For critical tasks with verification
.WithAllowReplanning(true).
WithVerifySteps(true).
WithMaxReplanAttempts(3).
WithTimeout(30 * time.Minute)

// For faster execution
.WithAllowReplanning(false).
WithVerifySteps(false).
WithTimeout(5 * time.Minute)
```

---

## Decision Matrix

### Choose Basic Agent When:
- Task is single-step or conversational
- Low latency is critical
- Token cost is a concern
- Behavior needs to be predictable

### Choose ReAct Agent When:
- Solution path is unclear
- Multiple information sources needed
- Reasoning transparency is important
- Task benefits from iterative exploration

### Choose Plan-Execute Agent When:
- Task has multiple discrete steps
- Dependencies between steps exist
- Progress tracking is needed
- Failures should trigger intelligent recovery

---

## Combining Patterns

You can combine patterns for more sophisticated systems:

### Example: Multi-Agent System
```go
// Research agent for information gathering
researcher := NewReactAgent("researcher", ...)

// Planning agent for complex tasks
planner := NewPlanExecuteAgent("planner", ...)

// Coordinator uses basic agent pattern
coordinator := NewAgent("coordinator", ...)

// Convert agents to tools
coordinatorTools := []Tool{
    researcher.AsTool(),
    planner.AsTool(),
}

coordinator.WithTools(coordinatorTools...)
```

### Example: Hierarchical Planning
```go
// High-level planner
mainPlanner := NewPlanExecuteAgent("main", ...)

// Each step executed by ReAct agent for flexibility
stepExecutor := NewReactAgent("executor", ...)

// Configure planner to use executor for each step
// (implementation would involve custom execution logic)
```

---

## Performance Comparison

Based on typical tasks:

| Pattern | Avg Latency | Token Usage | Success Rate | Cost (Relative) |
|---------|-------------|-------------|--------------|-----------------|
| Basic Agent | 2-5s | 1x | 85% | 1x |
| ReAct Agent | 10-30s | 5-10x | 90% | 5-10x |
| Plan-Execute Agent | 20-60s | 8-15x | 95% | 8-15x |

*Note: Values vary based on task complexity, model, and configuration*

---

## Best Practices

### 1. Start Simple
Begin with the simplest pattern that meets your needs. Upgrade only when necessary.

### 2. Monitor Performance
Track token usage, latency, and success rates to optimize pattern choice.

```go
// Use metrics to track pattern performance
metrics.Counter("agent.pattern", 
    metrics.WithLabel("type", "react"),
    metrics.WithLabel("status", "success"),
).Inc()
```

### 3. Configure Appropriately
Each pattern has knobs to tune - adjust based on your specific use case.

### 4. Use Memory Wisely
All patterns benefit from memory, but ReAct and Plan-Execute patterns particularly benefit from episodic memory of past attempts.

```go
memoryManager := sdk.NewMemoryManager(agentID, ..., &sdk.MemoryManagerOptions{
    WorkingCapacity: 20,
    ShortTermTTL:    24 * time.Hour,
})
```

### 5. Handle Failures Gracefully
```go
execution, err := agent.Execute(ctx, input)
if err != nil {
    // Check if it's a transient vs permanent failure
    // Consider fallback patterns
}
```

---

## Advanced Features

### Reflection System
Available for ReAct and Plan-Execute patterns:
- Quality evaluation of reasoning/plans
- Automatic issue detection
- Suggestion generation
- See [REFLECTION_AND_REPLANNING.md](./REFLECTION_AND_REPLANNING.md)

### Memory Integration
All patterns support memory, but with different semantics:
- **Basic Agent**: Conversation history
- **ReAct Agent**: Episodic reasoning traces
- **Plan-Execute Agent**: Plan history and failure patterns

### Tool System
All patterns use the same tool system:
```go
registry := sdk.NewToolRegistry(logger, metrics)
registry.RegisterFunc("search", "Search the web", searchFunc)
```

---

## Next Steps

- [ReAct Agents Detailed Guide](./REACT_AGENTS.md)
- [Plan-Execute Agents Detailed Guide](./PLAN_EXECUTE_AGENTS.md)
- [Reflection & Replanning](./REFLECTION_AND_REPLANNING.md)
- [Tool System Documentation](./TOOLS.md)
- [Memory System Documentation](./MEMORY.md)

