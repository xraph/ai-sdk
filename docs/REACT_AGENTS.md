# ReAct Agents: Reasoning + Acting

ReAct (Reasoning + Acting) is an agent pattern that combines iterative reasoning with action execution. The agent alternates between thinking about what to do next, executing actions (like calling tools), and observing the results.

## Table of Contents
- [Overview](#overview)
- [How It Works](#how-it-works)
- [Creating a ReAct Agent](#creating-a-react-agent)
- [Configuration Options](#configuration-options)
- [Reasoning Loop](#reasoning-loop)
- [Self-Reflection](#self-reflection)
- [Memory Integration](#memory-integration)
- [Examples](#examples)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

---

## Overview

### What is ReAct?

ReAct agents follow a simple but powerful loop:

```
1. Thought: What should I do next?
2. Action: Execute a tool or provide an answer
3. Observation: What was the result?
4. Reflection: How well did that work? (periodic)
... repeat until final answer
```

### When to Use ReAct

✅ **Good for:**
- Research and information gathering
- Multi-step problem solving
- Tasks with uncertain solution paths
- When reasoning transparency is important
- Exploratory data analysis

❌ **Not ideal for:**
- Simple single-step tasks
- Highly structured workflows (use Plan-Execute instead)
- Time-critical operations (use Basic Agent)
- Tasks requiring strict ordering (use Plan-Execute)

---

## How It Works

### The ReAct Loop

```
User Input: "What's the population of the capital of France?"

Step 1 - Thought:
  "I need to first identify the capital of France, then find its population."

Step 2 - Action:
  Tool: search, Query: "capital of France"

Step 3 - Observation:
  "Paris is the capital of France"

Step 4 - Thought:
  "Now I need to find the population of Paris"

Step 5 - Action:
  Tool: search, Query: "population of Paris"

Step 6 - Observation:
  "Paris has a population of approximately 2.1 million"

Step 7 - Thought:
  "I have the answer"

Step 8 - Final Answer:
  "The capital of France is Paris, which has a population of approximately 2.1 million people."
```

### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                       ReactAgent                             │
├─────────────────────────────────────────────────────────────┤
│  ┌────────────────────────────────────────────────────┐    │
│  │            ReactStrategy                            │    │
│  ├────────────────────────────────────────────────────┤    │
│  │  Loop (maxIterations):                             │    │
│  │    1. Generate Thought (LLM)                       │    │
│  │    2. Parse Action or Final Answer                 │    │
│  │    3. Execute Tool (if action)                     │    │
│  │    4. Capture Observation                          │    │
│  │    5. Reflect (if interval reached)                │    │
│  │       └─> ReflectionEngine                         │    │
│  │    6. Check confidence                             │    │
│  │    7. Store in Memory                              │    │
│  └────────────────────────────────────────────────────┘    │
│                                                              │
│  Components:                                                 │
│  - Agent (base functionality)                               │
│  - LLMManager (for reasoning)                               │
│  - MemoryManager (for learning)                             │
│  - Tools (for actions)                                      │
│  - ReflectionEngine (for quality checks)                    │
└─────────────────────────────────────────────────────────────┘
```

---

## Creating a ReAct Agent

### Basic Example

```go
package main

import (
    "context"
    "fmt"
    "github.com/xraph/ai-sdk"
)

func main() {
    // Setup dependencies
    llmManager := sdk.NewLLMManager(logger, metrics)
    memoryManager := sdk.NewMemoryManager(agentID, embedder, vectorStore, logger, metrics, nil)
    
    // Define tools
    searchTool := sdk.Tool{
        Name:        "search",
        Description: "Search the web for information",
        Parameters: map[string]any{
            "type": "object",
            "properties": map[string]any{
                "query": map[string]any{
                    "type":        "string",
                    "description": "Search query",
                },
            },
            "required": []string{"query"},
        },
        Handler: func(ctx context.Context, params map[string]any) (any, error) {
            query := params["query"].(string)
            return performWebSearch(query), nil
        },
    }
    
    // Create ReAct agent
    agent, err := sdk.NewReactAgentBuilder("research_assistant").
        WithModel("gpt-4").
        WithProvider("openai").
        WithSystemPrompt("You are a helpful research assistant.").
        WithTools(searchTool).
        WithLLMManager(llmManager).
        WithMemoryManager(memoryManager).
        WithMaxIterations(10).
        WithReflectionInterval(3).
        WithConfidenceThreshold(0.7).
        WithLogger(logger).
        WithMetrics(metrics).
        Build()
    
    if err != nil {
        panic(err)
    }
    
    // Execute
    execution, err := agent.Execute(context.Background(), 
        "What are the latest developments in quantum computing?")
    
    if err != nil {
        panic(err)
    }
    
    fmt.Println("Final Answer:", execution.FinalOutput)
    
    // Inspect reasoning
    traces := agent.GetTraces()
    for _, trace := range traces {
        fmt.Printf("\nStep %d (Confidence: %.2f):\n", trace.Step, trace.Confidence)
        fmt.Printf("  Thought: %s\n", trace.Thought)
        fmt.Printf("  Action: %s\n", trace.Action)
        fmt.Printf("  Observation: %s\n", trace.Observation)
        if trace.Reflection != "" {
            fmt.Printf("  Reflection: %s\n", trace.Reflection)
        }
    }
}
```

---

## Configuration Options

### Builder Methods

```go
agent, err := sdk.NewReactAgentBuilder("agent_name").
    // Required
    WithLLMManager(llmManager).
    
    // Model configuration
    WithModel("gpt-4").              // LLM model to use
    WithProvider("openai").          // Provider (openai, anthropic, etc.)
    WithTemperature(0.7).            // Creativity (0.0-1.0)
    
    // Agent identity
    WithID("custom_id").             // Custom agent ID
    WithDescription("Agent purpose"). // Description
    WithSystemPrompt("You are...").  // System prompt
    
    // Tools
    WithTools(tool1, tool2).         // Available tools
    
    // Execution control
    WithMaxIterations(10).           // Max reasoning steps
    WithConfidenceThreshold(0.7).    // Min confidence to continue
    
    // Reflection
    WithReflectionInterval(3).       // Reflect every N steps (0 = disabled)
    
    // Custom prompts
    WithReasoningPrompt(template).   // Custom reasoning template
    
    // Dependencies
    WithMemoryManager(mm).           // For learning from experience
    WithStateStore(store).           // For state persistence
    WithGuardrails(guardrails).      // Safety checks
    
    // Observability
    WithLogger(logger).              // Structured logging
    WithMetrics(metrics).            // Metrics collection
    
    Build()
```

### Configuration Profiles

```go
// For exploratory research (high tolerance, deep exploration)
.WithMaxIterations(15).
WithReflectionInterval(5).
WithConfidenceThreshold(0.6).
WithTemperature(0.8)

// For focused tasks (strict, fast)
.WithMaxIterations(5).
WithReflectionInterval(2).
WithConfidenceThreshold(0.85).
WithTemperature(0.3)

// For production (balanced)
.WithMaxIterations(10).
WithReflectionInterval(3).
WithConfidenceThreshold(0.75).
WithTemperature(0.7)
```

---

## Reasoning Loop

### Prompt Format

The ReAct agent uses a specific prompt format to guide the LLM:

```
You are an AI assistant that follows the ReAct pattern: Thought, Action, Observation.
You have access to the following tools:
- search: Search the web (Parameters: {query: string})
- calculator: Perform calculations (Parameters: {expression: string})

Use this format:
Thought: Your reasoning about what to do next
Action: tool_name
Action Input: {"param": "value"}
Observation: The result of the action
... (this Thought/Action/Observation can repeat multiple times)
Thought: I have the final answer
Final Answer: The complete answer to the original question

User Input: What is 15% of 240?
Thought:
```

### Parsing Logic

The strategy parses LLM output to extract:

1. **Thought**: Everything before "Action:" or "Final Answer:"
2. **Action**: Tool name after "Action:"
3. **Action Input**: JSON after "Action Input:"
4. **Final Answer**: Content after "Final Answer:"

Example parsing:
```go
output := `Thought: I need to calculate 15% of 240
Action: calculator
Action Input: {"expression": "0.15 * 240"}`

// Parsed as:
// thought = "I need to calculate 15% of 240"
// action = "calculator"
// actionInput = {"expression": "0.15 * 240"}
```

---

## Self-Reflection

### How It Works

Every N steps (configured via `WithReflectionInterval`), the agent evaluates its own reasoning:

```go
// Reflection evaluates:
// 1. Is the reasoning logical?
// 2. Is the action appropriate?
// 3. Is progress being made?
// 4. Should strategy change?

type ReflectionResult struct {
    Quality       string   // "good", "needs_improvement", "invalid"
    Score         float64  // 0.0-1.0
    Issues        []string // Identified problems
    Suggestions   []string // How to improve
    ShouldReplan  bool     // Major strategy change needed
}
```

### Reflection Example

```
Step 3 Reflection:
  Quality: needs_improvement
  Score: 0.65
  Issues:
    - Search query too broad, results not specific
    - Action doesn't address the core question
  Suggestions:
    - Refine search query to be more specific
    - Focus on numerical data rather than general info
  Should Replan: false
```

### Using Reflections

```go
reflections := agent.GetReflections()
for _, r := range reflections {
    if r.Score < 0.7 {
        log.Printf("Low quality reasoning detected: %s", r.Issues)
        // Take corrective action
    }
}
```

---

## Memory Integration

### Storing Reasoning Traces

ReAct agents automatically store reasoning traces in memory for future learning:

```go
// Automatic storage (handled by strategy)
trace := ReasoningTrace{
    Step:        1,
    Thought:     "I need to search for...",
    Action:      "search",
    Observation: "Found results...",
    Confidence:  0.85,
}
// Stored as episodic memory with importance based on confidence
```

### Recalling Past Experience

```go
// Agent can recall similar past reasoning
traces, err := sdk.RecallSimilarTraces(
    ctx,
    memoryManager,
    "quantum computing research",
    5, // top 5 similar traces
)

// Use recalled traces to inform current reasoning
for _, trace := range traces {
    fmt.Printf("Past approach: %s → %s\n", trace.Thought, trace.Action)
}
```

### Learning from Failures

```go
// Failed executions are stored with high importance
// Agent learns what NOT to do

// Query for common failure patterns
failedTraces := queryMemoryByMetadata(map[string]any{
    "type":   "reasoning_trace",
    "status": "failed",
})
```

---

## Examples

### Example 1: Multi-Step Research

```go
agent, _ := sdk.NewReactAgentBuilder("researcher").
    WithModel("gpt-4").
    WithTools(searchTool, summarizerTool).
    WithLLMManager(llmManager).
    WithMaxIterations(12).
    Build()

execution, _ := agent.Execute(ctx, `
    Research the following and provide a summary:
    1. Current quantum computing capabilities
    2. Major players in the field
    3. Expected timeline for practical applications
`)

// Agent will:
// 1. Search for quantum computing capabilities
// 2. Summarize findings
// 3. Search for companies/researchers
// 4. Search for timeline predictions
// 5. Synthesize all information
// 6. Provide final summary
```

### Example 2: Data Analysis

```go
analysisTool := sdk.Tool{
    Name: "analyze_csv",
    Description: "Analyze CSV data",
    Handler: func(ctx context.Context, params map[string]any) (any, error) {
        // Perform analysis
        return statistics, nil
    },
}

agent, _ := sdk.NewReactAgentBuilder("analyst").
    WithTools(analysisTool, plotTool, searchTool).
    WithMaxIterations(8).
    Build()

execution, _ := agent.Execute(ctx, 
    "Analyze sales data and identify trends. Explain any anomalies.")

// Agent iteratively:
// 1. Analyzes data
// 2. Identifies patterns
// 3. Searches for context on anomalies
// 4. Generates visualizations
// 5. Provides insights
```

### Example 3: Agent as Tool

```go
// Create specialized ReAct agents
researchAgent, _ := NewReactAgent("researcher", ...)
calculatorAgent, _ := NewReactAgent("calculator", ...)

// Convert to tools
tools := []Tool{
    researchAgent.AsTool(),
    calculatorAgent.AsTool(),
}

// Create coordinator agent
coordinator, _ := sdk.NewAgentBuilder().
    WithTools(tools...).
    Build()

// Coordinator delegates to specialized agents
result, _ := coordinator.Execute(ctx, 
    "Research the GDP of top 5 countries and calculate their average")
```

---

## Best Practices

### 1. Tool Design

```go
// ✅ Good: Focused, single-purpose tools
searchTool := Tool{Name: "search", Description: "Search the web"}
calculateTool := Tool{Name: "calculate", Description: "Math calculations"}

// ❌ Bad: Overly broad tools
doEverythingTool := Tool{Name: "do_stuff", Description: "Does many things"}
```

### 2. System Prompts

```go
// ✅ Good: Clear identity and constraints
WithSystemPrompt(`You are a research assistant specializing in technology.
Be thorough but concise. Always cite sources for facts.
If uncertain, say so explicitly.`)

// ❌ Bad: Vague or contradictory
WithSystemPrompt("You are smart. Be creative but precise.")
```

### 3. Iteration Limits

```go
// ✅ Good: Appropriate for task complexity
// Simple task
WithMaxIterations(5)

// Complex research
WithMaxIterations(15)

// ❌ Bad: Too high (wasted tokens) or too low (incomplete)
WithMaxIterations(50) // Usually unnecessary
WithMaxIterations(2)  // Too restrictive for ReAct
```

### 4. Confidence Thresholds

```go
// ✅ Good: Match to task criticality
// High-stakes decision
WithConfidenceThreshold(0.9)

// Exploratory research
WithConfidenceThreshold(0.6)

// ❌ Bad: Mismatched to task
WithConfidenceThreshold(0.99) // Too strict, may never complete
WithConfidenceThreshold(0.1)  // Too lenient, low quality
```

### 5. Error Handling

```go
execution, err := agent.Execute(ctx, input)
if err != nil {
    // Check execution status
    if execution != nil {
        switch execution.Status {
        case sdk.ExecutionStatusFailed:
            // Handle failure
            log.Error("Agent failed", "error", execution.Error)
        case sdk.ExecutionStatusCancelled:
            // Handle cancellation
        }
        
        // Still access partial results
        traces := agent.GetTraces()
        fmt.Printf("Completed %d steps before failure\n", len(traces))
    }
}
```

---

## Troubleshooting

### Problem: Agent Gets Stuck in Loops

**Symptoms**: Same action repeated multiple times

**Solutions**:
```go
// 1. Enable reflection
WithReflectionInterval(2) // Catch loops early

// 2. Lower max iterations
WithMaxIterations(8) // Limit potential loops

// 3. Improve tool descriptions
Tool{
    Name: "search",
    Description: "Search ONCE for specific information. Don't repeat same query.",
}

// 4. Add to system prompt
WithSystemPrompt(`...
If you find yourself repeating actions, try a different approach.`)
```

### Problem: Low Quality Outputs

**Symptoms**: Reflections show low scores

**Solutions**:
```go
// 1. Increase reflection frequency
WithReflectionInterval(2) // More frequent checks

// 2. Raise confidence threshold
WithConfidenceThreshold(0.8) // Stricter quality bar

// 3. Improve tool quality
// Ensure tools return rich, useful information

// 4. Use memory for learning
WithMemoryManager(mm) // Learn from past attempts
```

### Problem: High Token Usage

**Symptoms**: Expensive executions

**Solutions**:
```go
// 1. Reduce max iterations
WithMaxIterations(6)

// 2. Disable or reduce reflection
WithReflectionInterval(0) // Disable
// or
WithReflectionInterval(5) // Less frequent

// 3. Use smaller model for reflection
// (if using custom ReflectionEngine)

// 4. Improve tool efficiency
// Return concise, relevant information
```

### Problem: Timeouts

**Symptoms**: Execution doesn't complete

**Solutions**:
```go
// 1. Add context timeout
ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
defer cancel()

// 2. Reduce max iterations
WithMaxIterations(5)

// 3. Optimize tool performance
// Ensure tools respond quickly

// 4. Use streaming for long operations
// (if tool supports it)
```

---

## Performance Tuning

### Metrics to Monitor

```go
// Track these metrics for optimization
metrics.Histogram("react.iterations").Observe(float64(len(traces)))
metrics.Histogram("react.confidence").Observe(avgConfidence)
metrics.Counter("react.reflection.triggered").Inc()
metrics.Histogram("react.execution_time").Observe(duration.Seconds())
```

### Optimization Checklist

- [ ] Appropriate max iterations for task type
- [ ] Reflection interval tuned (or disabled for simple tasks)
- [ ] Tools return concise, relevant information
- [ ] System prompt is clear and focused
- [ ] Confidence threshold matches task criticality
- [ ] Memory enabled for repeated similar tasks
- [ ] Timeouts set appropriately
- [ ] Error handling implemented
- [ ] Metrics collection enabled

---

## Next Steps

- [Plan-Execute Agents](./PLAN_EXECUTE_AGENTS.md) - For structured multi-step tasks
- [Reflection & Replanning](./REFLECTION_AND_REPLANNING.md) - Advanced self-improvement
- [Agent Patterns Overview](./AGENT_PATTERNS.md) - Compare all patterns
- [Tool System](./TOOLS.md) - Creating effective tools
- [Memory System](./MEMORY.md) - Learning from experience

