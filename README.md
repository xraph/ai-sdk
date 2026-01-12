# 🚀 Forge AI SDK

> **Enterprise-Grade AI SDK for Go** - Zero-Cost Guardrails, Native Concurrency, Production-First

A production-ready, type-safe AI SDK for Go with advanced features like multi-tier memory, RAG, workflow orchestration, cost management, and enterprise guardrails. Built for high-throughput microservices with zero external dependencies.

> ⚠️ **Status**: Alpha - API may change. Test coverage 81%+. Suitable for evaluation and non-critical production use.

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![Test Coverage](https://img.shields.io/badge/coverage-81%25-brightgreen)](./tests)
[![Integrations](https://img.shields.io/badge/integrations-17-blue)](./integrations)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

---

## ✨ Features

### 🎯 **Core Capabilities**

| Feature | Description | Status |
|---------|-------------|--------|
| **Fluent API** | Type-safe builders with method chaining | ✅ |
| **Structured Output** | Go generics + JSON schema validation | ✅ |
| **Enhanced Streaming** | Token-by-token with reasoning steps | ✅ |
| **RAG Support** | Full pipeline with semantic search & reranking | ✅ |
| **Multi-Tier Memory** | Working → Short → Long → Episodic | ✅ |
| **Tool System** | Auto-discovery from Go functions | ✅ |
| **Workflow Engine** | DAG-based orchestration | ✅ |
| **Prompt Templates** | Versioning + A/B testing | ✅ |
| **Safety Guardrails** | PII, injection, toxicity detection | ✅ |
| **Cost Management** | Budget tracking + optimization | ✅ |
| **Resilience** | Circuit breakers, retries, rate limiting | ✅ |
| **Observability** | Tracing, profiling, debugging | ✅ |

---

## 🚦 Quick Start

### Installation

```bash
go get github.com/xraph/ai-sdk
```

### Basic Text Generation

```go
package main

import (
	"context"
	"fmt"
	
	"github.com/xraph/ai-sdk"
)

func main() {
	// Create LLM manager
	llmManager := sdk.NewLLMManager(logger, metrics)
	
	// Generate text
	result, err := sdk.NewGenerateBuilder(context.Background(), llmManager, logger, metrics).
		WithProvider("openai").
		WithModel("gpt-4").
		WithPrompt("Explain quantum computing in simple terms").
		WithMaxTokens(500).
		WithTemperature(0.7).
		Execute()
	
	if err != nil {
		panic(err)
	}
	
	fmt.Println(result.Content)
}
```

---

## 📚 Examples

### 1. Structured Output with Type Safety

```go
type Person struct {
	Name    string   `json:"name" description:"Full name"`
	Age     int      `json:"age" description:"Age in years"`
	Hobbies []string `json:"hobbies" description:"List of hobbies"`
}

person, err := sdk.NewGenerateObjectBuilder[Person](ctx, llmManager, logger, metrics).
	WithProvider("openai").
	WithModel("gpt-4").
	WithPrompt("Extract person info: John Doe, 30, loves reading and hiking").
	WithValidator(func(p *Person) error {
		if p.Age < 0 || p.Age > 150 {
			return fmt.Errorf("invalid age: %d", p.Age)
		}
		return nil
	}).
	Execute()

fmt.Printf("%+v\n", person)
// Output: &{Name:John Doe Age:30 Hobbies:[reading hiking]}
```

### 2. Enhanced Streaming with Reasoning

```go
stream := sdk.NewStreamBuilder(ctx, llmManager, logger, metrics).
	WithProvider("anthropic").
	WithModel("claude-3-opus").
	WithPrompt("Solve: What is 15% of 240?").
	WithReasoning(true).
	OnToken(func(token string) {
		fmt.Print(token)
	}).
	OnReasoning(func(step string) {
		fmt.Printf("[Thinking: %s]\n", step)
	}).
	OnComplete(func(result *sdk.Result) {
		fmt.Printf("\n✅ Complete! Cost: $%.4f\n", result.Usage.Cost)
	})

result, err := stream.Stream()
```

**Output:**
```
[Thinking: Need to calculate 15% of 240]
[Thinking: 15% = 0.15, so 0.15 * 240]
The answer is 36.
✅ Complete! Cost: $0.0120
```

### 3. RAG (Retrieval Augmented Generation)

```go
// Setup RAG
embeddingModel := &MyEmbeddingModel{}
vectorStore := &MyVectorStore{}

rag := sdk.NewRAG(embeddingModel, vectorStore, logger, metrics, &sdk.RAGOptions{
	ChunkSize:    500,
	ChunkOverlap: 50,
	TopK:         5,
})

// Index documents
rag.IndexDocument(ctx, "doc1", "Quantum computers use qubits that can be 0 and 1 simultaneously...")
rag.IndexDocument(ctx, "doc2", "Machine learning models require training data...")

// Retrieve and generate
generator := sdk.NewGenerateBuilder(ctx, llmManager, logger, metrics).
	WithProvider("openai").
	WithModel("gpt-4")

result, err := rag.GenerateWithContext(ctx, generator, "How do quantum computers work?", 5)
```

### 4. Stateful Agents with Memory

```go
// Create agent
agent, err := sdk.NewAgent("assistant", "AI Assistant", llmManager, stateStore, logger, metrics, &sdk.AgentOptions{
	SystemPrompt: "You are a helpful assistant with memory of past conversations.",
	MaxIterations: 10,
	Temperature: 0.7,
})

// First conversation
result1, err := agent.Execute(ctx, "My name is Alice and I love hiking.")
// Agent: "Nice to meet you, Alice! Hiking is a wonderful activity..."

// Second conversation (agent remembers)
result2, err := agent.Execute(ctx, "What do you know about me?")
// Agent: "You're Alice, and you mentioned you love hiking!"

// Save state
agent.SaveState(ctx)
```

### 4a. ReAct Agents (Reasoning + Acting)

```go
// Create ReAct agent with iterative reasoning
reactAgent, err := sdk.NewReactAgentBuilder("research_assistant").
	WithModel("gpt-4").
	WithProvider("openai").
	WithSystemPrompt("You are a research assistant that thinks step-by-step.").
	WithTools(searchTool, calculatorTool, summarizerTool).
	WithLLMManager(llmManager).
	WithMemoryManager(memoryManager).
	WithMaxIterations(10).
	WithReflectionInterval(3).    // Self-reflect every 3 steps
	WithConfidenceThreshold(0.7). // Minimum confidence level
	WithLogger(logger).
	WithMetrics(metrics).
	Build()

// Execute with ReAct loop: Thought → Action → Observation → Reflection
execution, err := reactAgent.Execute(ctx, "What are the latest breakthroughs in quantum computing and their potential applications?")

// Get reasoning traces
traces := reactAgent.GetTraces()
for _, trace := range traces {
	fmt.Printf("Step %d:\n", trace.Step)
	fmt.Printf("  Thought: %s\n", trace.Thought)
	fmt.Printf("  Action: %s\n", trace.Action)
	fmt.Printf("  Observation: %s\n", trace.Observation)
	fmt.Printf("  Confidence: %.2f\n", trace.Confidence)
}

// Get self-reflections
reflections := reactAgent.GetReflections()
for _, reflection := range reflections {
	fmt.Printf("Quality: %s, Score: %.2f\n", reflection.Quality, reflection.Score)
	fmt.Printf("Issues: %v\n", reflection.Issues)
	fmt.Printf("Suggestions: %v\n", reflection.Suggestions)
}
```

### 4b. Plan-Execute Agents

```go
// Create Plan-Execute agent for complex multi-step tasks
planAgent, err := sdk.NewPlanExecuteAgentBuilder("project_manager").
	WithModel("gpt-4").
	WithProvider("openai").
	WithSystemPrompt("You are an expert at breaking down complex tasks into plans.").
	WithTools(fileTool, apiTool, dbTool, searchTool).
	WithLLMManager(llmManager).
	WithPlanStore(planStore).
	WithMemoryManager(memoryManager).
	WithAllowReplanning(true).    // Enable automatic replanning on failures
	WithVerifySteps(true).         // Verify each step's output
	WithMaxReplanAttempts(3).      // Max replanning attempts
	WithTimeout(10 * time.Minute). // Execution timeout
	WithLogger(logger).
	WithMetrics(metrics).
	Build()

// Execute: Plan → Execute → Verify
execution, err := planAgent.Execute(ctx, "Build a user authentication system with email verification and password reset")

// Get the generated plan
plan := planAgent.GetCurrentPlan()
fmt.Printf("Plan: %s (Status: %s)\n", plan.Goal, plan.Status)
for i, step := range plan.Steps {
	fmt.Printf("  Step %d: %s\n", i+1, step.Description)
	fmt.Printf("    Status: %s, Result: %v\n", step.Status, step.Result)
	if step.Verification != nil {
		fmt.Printf("    Verified: %v (Score: %.2f)\n", step.Verification.IsValid, step.Verification.Score)
	}
}

// Get plan history (including replans)
history := planAgent.GetPlanHistory()
fmt.Printf("Total plans created: %d (including %d replans)\n", len(history), len(history)-1)
```

### 5. Multi-Tier Memory System

```go
// Create memory manager
memoryManager := sdk.NewMemoryManager("agent_1", embeddingModel, vectorStore, logger, metrics, &sdk.MemoryManagerOptions{
	WorkingCapacity: 10,
	ShortTermTTL:    24 * time.Hour,
	ImportanceDecay: 0.1,
})

// Store memories
memoryManager.Store(ctx, "User prefers morning meetings", 0.8, nil) // High importance
memoryManager.Store(ctx, "Weather is sunny today", 0.3, nil)         // Low importance

// Recall memories
memories, err := memoryManager.Recall(ctx, "meetings", sdk.MemoryTierAll, 5)

// Promote important memory
memoryManager.Promote(ctx, memories[0].ID)

// Create episodic memory
episode := &sdk.EpisodicMemory{
	Title:       "Project Launch Meeting",
	Description: "Discussed Q1 goals and assigned tasks",
	Memories:    []string{"mem_id_1", "mem_id_2"},
	Tags:        []string{"project", "meeting"},
	Importance:  0.9,
}
memoryManager.CreateEpisode(ctx, episode)
```

### 6. Dynamic Tool System

```go
// Create tool registry
registry := sdk.NewToolRegistry(logger, metrics)

// Register from Go function
registry.RegisterFunc("calculate", "Performs arithmetic", func(ctx context.Context, a, b float64, op string) (float64, error) {
	switch op {
	case "add":
		return a + b, nil
	case "multiply":
		return a * b, nil
	default:
		return 0, fmt.Errorf("unsupported operation: %s", op)
	}
})

// Register with full definition
registry.RegisterTool(&sdk.ToolDefinition{
	Name:        "web_search",
	Version:     "1.0.0",
	Description: "Searches the web",
	Parameters: sdk.ToolParameterSchema{
		Type: "object",
		Properties: map[string]sdk.ToolParameterProperty{
			"query": {Type: "string", Description: "Search query"},
		},
		Required: []string{"query"},
	},
	Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		query := params["query"].(string)
		return performWebSearch(query), nil
	},
	Timeout: 10 * time.Second,
})

// Execute tool
result, err := registry.ExecuteTool(ctx, "calculate", "1.0.0", map[string]interface{}{
	"param0": 10.0,
	"param1": 5.0,
	"param2": "add",
})
// result.Result: 15.0
```

### 7. Workflow Engine (DAG)

```go
// Create workflow
workflow := sdk.NewWorkflow("onboarding", "User Onboarding", logger, metrics)

// Add nodes
workflow.AddNode(&sdk.WorkflowNode{
	ID:       "validate",
	Type:     sdk.NodeTypeTool,
	ToolName: "validate_email",
})

workflow.AddNode(&sdk.WorkflowNode{
	ID:      "send_welcome",
	Type:    sdk.NodeTypeAgent,
	AgentID: "email_agent",
})

workflow.AddNode(&sdk.WorkflowNode{
	ID:        "check_activation",
	Type:      sdk.NodeTypeCondition,
	Condition: "user.activated == true",
})

// Define dependencies
workflow.AddEdge("validate", "send_welcome")
workflow.AddEdge("send_welcome", "check_activation")
workflow.SetStartNode("validate")

// Execute
execution, err := workflow.Execute(ctx, map[string]interface{}{
	"email": "user@example.com",
})

fmt.Printf("Workflow completed: %s\n", execution.Status)
```

### 8. Prompt Templates with A/B Testing

```go
// Create template manager
templateMgr := sdk.NewPromptTemplateManager(logger, metrics)

// Register template
templateMgr.RegisterTemplate(&sdk.PromptTemplate{
	Name:     "greeting",
	Version:  "1.0.0",
	Template: "Hello, {{.Name}}! Welcome to {{.Service}}.",
})

// Render
text, err := templateMgr.Render("greeting", "1.0.0", map[string]interface{}{
	"Name":    "Alice",
	"Service": "AI Platform",
})
// Output: "Hello, Alice! Welcome to AI Platform."

// Create A/B test
templateMgr.RegisterTemplate(&sdk.PromptTemplate{
	Name:     "greeting",
	Version:  "2.0.0",
	Template: "Hi {{.Name}}! 👋 Excited to have you on {{.Service}}!",
})

templateMgr.CreateABTest(&sdk.ABTest{
	Name: "greeting",
	Variants: []sdk.ABVariant{
		{Name: "control", TemplateVersion: "1.0.0", TrafficWeight: 0.5},
		{Name: "friendly", TemplateVersion: "2.0.0", TrafficWeight: 0.5},
	},
})

// Render will automatically select variant
text, _ = templateMgr.Render("greeting", "", vars)

// Track results
templateMgr.RecordABTestResult("greeting", "friendly", true, 120*time.Millisecond, 0.001)
```

### 9. Safety Guardrails

```go
// Create guardrail manager
guardrails := sdk.NewGuardrailManager(logger, metrics, &sdk.GuardrailOptions{
	EnablePII:             true,
	EnableToxicity:        true,
	EnablePromptInjection: true,
	EnableContentFilter:   true,
	MaxInputLength:        10000,
})

// Validate input
violations, err := guardrails.ValidateInput(ctx, userInput)
if sdk.ShouldBlock(violations) {
	return fmt.Errorf("input blocked: %s", sdk.FormatViolations(violations))
}

// Validate output
violations, err = guardrails.ValidateOutput(ctx, aiOutput)

// Redact PII
clean := guardrails.RedactPII("My email is john@example.com")
// Output: "My email is [REDACTED]"
```

**Why Native Guardrails?**
- ✅ **Zero cost**: No external API calls or subscriptions required
- ✅ **Sub-millisecond latency**: Regex-based detection vs 50-200ms API calls
- ✅ **Privacy**: Data never leaves your infrastructure
- ✅ **Offline-capable**: Works in air-gapped environments
- ✅ **Predictable**: Deterministic behavior, no ML black boxes

**Comparison with Vercel AI SDK (v5):**
- **Vercel approach**: Middleware + external services (Portkey: 250+ LLMs, 50+ guardrails, $99-$499/mo)
  - Pros: ML-based detection, multi-language support, continuously updated
  - Cons: API latency (50-200ms), external dependency, monthly costs
- **Forge approach**: Native built-in (regex/pattern-based, free)
  - Pros: <1ms latency, no external calls, works offline, zero cost, privacy
  - Cons: Limited to patterns (not ML), primarily English, manual updates

**Best fit**: Use Forge for cost-sensitive, high-throughput, or air-gapped deployments. Use Vercel + Portkey for advanced ML-based detection and compliance requirements.

### 10. Cost Management

```go
// Create cost tracker
costTracker := sdk.NewCostTracker(logger, metrics, &sdk.CostTrackerOptions{
	RetentionPeriod: 30 * 24 * time.Hour,
})

// Set budget
costTracker.SetBudget("monthly", 1000.0, 30*24*time.Hour, 0.8) // Alert at 80%

// Record usage (automatic via SDK)
costTracker.RecordUsage(ctx, sdk.UsageRecord{
	Provider:     "openai",
	Model:        "gpt-4",
	InputTokens:  1000,
	OutputTokens: 500,
})

// Get insights
insights := costTracker.GetInsights()
fmt.Printf("Cost today: $%.2f\n", insights.CostToday)
fmt.Printf("Projected monthly: $%.2f\n", insights.ProjectedMonthly)
fmt.Printf("Cache hit rate: %.1f%%\n", insights.CacheHitRate*100)

// Get recommendations
recommendations := costTracker.GetOptimizationRecommendations()
for _, rec := range recommendations {
	fmt.Printf("💡 %s: %s (Save $%.2f)\n", rec.Title, rec.Description, rec.PotentialSavings)
}
```

### 11. Resilience Patterns

```go
// Circuit Breaker
cb := sdk.NewCircuitBreaker(sdk.CircuitBreakerConfig{
	Name:         "llm_api",
	MaxFailures:  5,
	ResetTimeout: 60 * time.Second,
}, logger, metrics)

err := cb.Execute(ctx, func(ctx context.Context) error {
	return callLLM(ctx)
})

// Retry with Exponential Backoff
err = sdk.Retry(ctx, sdk.RetryConfig{
	MaxAttempts:  3,
	InitialDelay: 100 * time.Millisecond,
	MaxDelay:     5 * time.Second,
	Multiplier:   2.0,
	Jitter:       true,
}, logger, func(ctx context.Context) error {
	return unreliableOperation(ctx)
})

// Rate Limiter
limiter := sdk.NewRateLimiter("api_calls", 10, 20, logger, metrics) // 10/sec, burst 20

if limiter.Allow() {
	callAPI()
}

// Or wait for token
limiter.Wait(ctx)

// Fallback Chain
fallback := sdk.NewFallbackChain("generation", logger, metrics)
err = fallback.Execute(ctx,
	func(ctx context.Context) error { return tryPrimaryModel(ctx) },
	func(ctx context.Context) error { return trySecondaryModel(ctx) },
	func(ctx context.Context) error { return useCachedResponse(ctx) },
)

// Bulkhead (concurrency limiting)
bulkhead := sdk.NewBulkhead("heavy_ops", 5, logger, metrics) // Max 5 concurrent

bulkhead.Execute(ctx, func(ctx context.Context) error {
	return heavyOperation(ctx)
})
```

### 12. Observability & Debugging

```go
// Distributed Tracing
tracer := sdk.NewTracer(logger, metrics)
span := tracer.StartSpan(ctx, "generate_response")
span.SetTag("model", "gpt-4")
span.SetTag("user_id", "user_123")
defer span.Finish()

// Add span to context
ctx = sdk.ContextWithSpan(ctx, span)

// Log events
span.LogEvent("info", "Processing request", map[string]interface{}{
	"tokens": 1000,
})

// Performance Profiling
profiler := sdk.NewProfiler(logger, metrics)
session := profiler.StartProfile("llm_call")
// ... do work ...
session.End()

// Get profile
profile := profiler.GetProfile("llm_call")
fmt.Printf("Avg: %v, P95: %v, P99: %v\n", profile.AvgTime, profile.Percentiles[95], profile.Percentiles[99])

// Debugging
debugger := sdk.NewDebugger(logger)
debugInfo := debugger.GetDebugInfo()
fmt.Printf("Goroutines: %d\n", debugInfo.Goroutines)
fmt.Printf("Memory: %d MB\n", debugInfo.MemoryStats.Alloc/1024/1024)

// Health Checks
healthChecker := sdk.NewHealthChecker(logger)
healthChecker.RegisterCheck("llm_api", func(ctx context.Context) error {
	return pingLLMAPI(ctx)
})
healthChecker.RegisterCheck("vector_store", func(ctx context.Context) error {
	return pingVectorStore(ctx)
})

status := healthChecker.GetOverallHealth(ctx)
// Output: "healthy", "degraded", or "unhealthy"
```

---

## 🎯 Advanced Use Cases

### Complete AI Agent with All Features

```go
func buildProductionAgent() *sdk.Agent {
	// Setup components
	llmManager := setupLLMManager()
	vectorStore := setupVectorStore()
	embeddingModel := setupEmbeddingModel()
	toolRegistry := setupTools()
	
	// Create memory system
	memory := sdk.NewMemoryManager("agent_id", embeddingModel, vectorStore, logger, metrics, &sdk.MemoryManagerOptions{
		WorkingCapacity: 20,
		ShortTermTTL:    24 * time.Hour,
	})
	
	// Setup RAG
	rag := sdk.NewRAG(embeddingModel, vectorStore, logger, metrics, nil)
	
	// Index knowledge base
	rag.IndexDocument(ctx, "kb_1", knowledgeBase)
	
	// Setup guardrails
	guardrails := sdk.NewGuardrailManager(logger, metrics, &sdk.GuardrailOptions{
		EnablePII:             true,
		EnableToxicity:        true,
		EnablePromptInjection: true,
	})
	
	// Create agent
	agent, _ := sdk.NewAgent("prod_agent", "Production Agent", llmManager, stateStore, logger, metrics, &sdk.AgentOptions{
		SystemPrompt: "You are a helpful AI assistant with access to tools and knowledge.",
		Tools:        toolRegistry.ListTools(),
		Guardrails:   guardrails,
		MaxIterations: 10,
	})
	
	return agent
}
```

---

## 🔌 Integrations

Forge AI SDK includes a comprehensive **integrations module** with production-ready implementations for popular services. All integrations use official Go SDKs where available and include complete test coverage.

### Vector Stores

| Integration | Status | SDK | Description |
|------------|--------|-----|-------------|
| **Memory** | ✅ | Built-in | In-memory store for testing and local development |
| **pgvector** | ✅ | `pgx/v5` | PostgreSQL with vector similarity search |
| **Qdrant** | ✅ | Official | High-performance vector database with gRPC |
| **Pinecone** | ✅ | Official | Managed vector database service |
| **Weaviate** | ✅ | Official | Vector database with GraphQL API |
| **ChromaDB** | ✅ | REST | Open-source embedding database |

### State & Cache Stores

| Integration | Status | SDK | Description |
|------------|--------|-----|-------------|
| **Memory (State)** | ✅ | Built-in | In-memory state with optional TTL |
| **Memory (Cache)** | ✅ | Built-in | LRU cache with TTL support |
| **PostgreSQL** | ✅ | `pgx/v5` | JSONB-based state storage |
| **Redis (State)** | ✅ | `go-redis/v9` | Distributed state management |
| **Redis (Cache)** | ✅ | `go-redis/v9` | High-performance caching |

### Embedding Models

| Integration | Status | SDK | Description |
|------------|--------|-----|-------------|
| **OpenAI** | ✅ | Official | text-embedding-3-small/large |
| **Cohere** | ✅ | Official | embed-english-v3.0, multilingual support |
| **Ollama** | ✅ | Built-in | Local embedding models |

### Usage Example

```go
import (
    "github.com/xraph/ai-sdk/integrations/vectorstores/pgvector"
    "github.com/xraph/ai-sdk/integrations/embeddings/openai"
    "github.com/xraph/ai-sdk/integrations/statestores/redis"
)

// Vector store
vectorStore, _ := pgvector.NewPgVectorStore(ctx, pgvector.Config{
    ConnString: "postgres://localhost/mydb",
    TableName:  "embeddings",
    Dimensions: 1536,
})

// Embeddings
embedder, _ := openai.NewOpenAIEmbeddings(openai.Config{
    APIKey: os.Getenv("OPENAI_API_KEY"),
    Model:  "text-embedding-3-small",
})

// State store
stateStore, _ := redis.NewRedisStateStore(redis.Config{
    Addr:   "localhost:6379",
    Prefix: "agent:",
})

// Use with RAG
rag := sdk.NewRAG(sdk.RAGConfig{
    VectorStore: vectorStore,
    Embedder:    embedder,
    TopK:        5,
})
```

### Testing & Benchmarking

All integrations include:
- ✅ **Unit tests** with mocks
- ✅ **Integration tests** with testcontainers-go
- ✅ **Benchmarks** for performance validation
- ✅ **Complete documentation** with examples

```bash
# Run integration tests (requires Docker)
cd integrations
go test -tags=integration ./tests/integration/...

# Run benchmarks
go test -bench=. -benchmem ./benchmarks/...
```

### Documentation

- [Integrations Overview](./integrations/README.md)
- [Vector Stores Guide](./integrations/vectorstores/)
- [State & Cache Stores](./integrations/statestores/)
- [Embeddings Guide](./integrations/embeddings/)
- [Integration Tests](./integrations/tests/integration/README.md)
- [Benchmarks Guide](./integrations/benchmarks/README.md)

---

## 📊 Performance & Scale

- **Throughput**: 1000+ requests/sec (with pooling)
- **Latency**: P99 < 200ms (streaming)
- **Memory**: < 50MB base, scales linearly
- **Concurrency**: Fully thread-safe
- **Test Coverage**: 81%+
- **Integrations**: 17 production-ready implementations

---

## 🤝 Comparison

> **Last Updated**: January 2026
> 
> **Major Updates**:
> - Vercel AI SDK v5: Multi-modal streaming (audio/video), Agent class, unified tool calling
> - Vercel AI SDK v4.2: Stable middleware support for guardrails/caching
> - LangChain: 100+ vector store integrations, LangGraph production-ready
> - Forge AI SDK: Native guardrails, 4-tier memory, cost optimization

### Framework Comparison (2026)

| Feature | Vercel AI SDK (v5) | LangChain (Python) | LangChain.js | **Forge AI SDK** |
|---------|-------------------|-------------------|--------------|------------------|
| **Language** | TypeScript/JS | Python | TypeScript/JS | **Go** |
| **Type Safety** | ✅ TypeScript | ⚠️ Type hints | ✅ TypeScript | **✅✅ Generics + Runtime** |
| **Structured Output** | ✅✅ Zod + native schemas | ✅ Pydantic | ✅ Zod | **✅ Go structs + validation** |
| **Streaming** | ✅✅ Multi-modal (SSE) | ✅ Text + tool calls | ✅ Text + Objects | **✅ Text + Objects + UI** |
| **Multi-Modal** | ✅✅ Text, image, audio, video | ⚠️ Via integrations | ⚠️ Via integrations | **⚠️ Text + image only** |
| **Memory System** | ⚠️ Basic conversation | ✅ Conversation buffer | ⚠️ Basic | **✅✅ 4-Tier + Episodic** |
| **RAG Support** | ⚠️ Basic utilities | ✅✅ Full pipeline | ✅ Full pipeline | **✅✅ Chunking + Reranking** |
| **Vector Stores** | ✅ Integrations | ✅✅ 100+ integrations | ✅ 70+ integrations | **✅✅ 6 built-in + pluggable** |
| **Agents** | ✅ Agent class (v5) | ✅✅ ReAct, Plan-Execute | ✅ ReAct agents | **✅✅ ReAct + Plan-Execute + Reflection** |
| **Workflow Engine** | ⚠️ Agent primitives | ✅✅ LangGraph (prod) | ✅ LangGraph.js | **✅ Native DAG engine** |
| **Tool Calling** | ✅✅ Unified 100+ models | ✅ Manual | ✅ Manual | **✅✅ Auto from Go funcs** |
| **Cost Tracking** | ⚠️ Via AI Gateway | ⚠️ Token counting | ⚠️ Token counting | **✅✅ Budget + Optimization** |
| **Guardrails** | ✅ Middleware + Portkey | ⚠️ Via integrations | ⚠️ Via integrations | **✅✅ Native built-in (free)** |
| **Middleware** | ✅✅ Stable (v4.2+) | ⚠️ Basic callbacks | ⚠️ Basic callbacks | **⚠️ Direct calls only** |
| **A/B Testing** | ❌ | ❌ | ❌ | **✅ Prompt variants** |
| **Resilience** | ⚠️ Via AI Gateway | ⚠️ Basic retry | ⚠️ Basic retry | **✅✅ Circuit breaker + more** |
| **Observability** | ✅ OpenTelemetry | ✅✅ LangSmith (prod) | ✅ LangSmith | **✅ Native tracing + metrics** |
| **Caching** | ✅✅ Prompt cache + Portkey | ✅ Via Redis/LangChain | ⚠️ Via Redis | **✅ Semantic + Provider** |
| **Provider Support** | ✅✅ 100+ via Gateway/40+ native | ✅✅ Many providers | ✅✅ 30+ providers | **✅ 5+ (extensible)** |
| **Framework Support** | ✅✅ React, Vue, Svelte, Angular | ❌ Backend only | ✅ Node.js/Edge | **❌ Backend only** |
| **External Dependencies** | ⚠️ Portkey for advanced features | ⚠️ Many optional deps | ⚠️ Many deps | **✅✅ Zero for core features** |
| **Production Ready** | ✅✅ Battle-tested (v5 stable) | ✅✅ Mature ecosystem | ✅ Mature | **⚠️ Alpha** |
| **Performance** | ⚠️ Node.js overhead | ⚠️ Python GIL | ⚠️ Node.js overhead | **✅✅ Native concurrency** |
| **Best For** | Next.js, React apps | Python ML stack | JS/TS full-stack | **Go microservices** |

### Key Differentiators

**Forge AI SDK (Go) excels at:**
- ✅ **Zero-cost guardrails**: Native PII/toxicity/injection detection with <1ms latency (no external API costs)
- ✅ **Native concurrency**: Goroutines for high-throughput production systems (1000+ req/sec)
- ✅ **Type safety**: Compile-time guarantees with Go generics + runtime validation
- ✅ **Enterprise features**: Built-in cost management, budgets, and resilience patterns (circuit breakers, bulkheads)
- ✅ **Production integrations**: 17 built-in integrations with official SDKs (pgvector, Qdrant, Pinecone, Weaviate, Redis, etc.)
- ✅ **Single binary**: Easy deployment, no runtime dependencies, works offline/air-gapped
- ✅ **Production-first**: Structured logging, distributed tracing, health checks built-in
- ✅ **Comprehensive testing**: Integration tests with testcontainers + performance benchmarks

**Vercel AI SDK (v5) excels at:**
- ✅ **Multi-modal streaming**: Native support for text, images, audio, video via unified API (SSE)
- ✅ **Agent primitives** (v5): `Agent` class with `prepareStep`, `stopWhen` for dynamic workflows
- ✅ **Composable middleware** (v4.2+): Stable pattern for guardrails, caching, logging across providers
- ✅ **AI Gateway**: Unified API to 100+ models with automatic fallback, usage tracking, billing
- ✅ **Framework-agnostic**: React, Vue, Svelte, Angular support with feature parity
- ✅ **Unified tool calling**: Standardized across 100+ models from 25+ providers
- ✅ **OpenTelemetry**: Native observability and distributed tracing
- ✅ **Portkey integration**: 50+ ML-based guardrails (requires $99-$499/mo)
- ✅ **Production-ready**: Battle-tested with Next.js ecosystem, v5 stable release
- ✅ **UI Elements library**: Pre-built React components for AI interfaces

**LangChain excels at:**
- ✅ **Massive ecosystem**: 100+ integrations with vector stores, tools, and services
- ✅ **Mature patterns**: ReAct agents, Plan-Execute, proven RAG architectures
- ✅ **LangGraph**: Industry-leading DAG-based workflow orchestration
- ✅ **LangSmith**: Production observability, prompt management, evaluation
- ✅ **Research-backed**: Strong community support and cutting-edge implementations

### Architecture Philosophy

**Forge AI SDK (Go)** prioritizes **self-contained, cost-effective operations** with native implementations of core features. Zero external dependencies for core functionality. Best for: Go microservices, cost-conscious teams, air-gapped/enterprise environments, high-throughput systems (1000+ req/sec).

**Vercel AI SDK (v5)** prioritizes **developer experience and multi-modal capabilities** with framework-first design. Best for: Next.js/React applications, teams needing multi-modal AI (audio/video), rapid prototyping, unified access to 100+ models, frontend-focused development.

**LangChain** prioritizes **flexibility and ecosystem maturity** with 100+ integrations and production-grade tooling (LangGraph, LangSmith). Best for: Python ML teams, complex agent workflows, research and experimentation, teams heavily invested in Python ecosystem.

### Cost Comparison (Monthly)

| Feature | Forge AI SDK | Vercel AI SDK | LangChain (Python) | Notes |
|---------|-------------|---------------|-------------------|-------|
| **SDK/License** | $0 | $0 | $0 | All open-source |
| **Guardrails** | $0 (native) | $99-$499/mo (Portkey) | $0-$499/mo (via integrations) | Forge: Built-in, Vercel/LangChain: External |
| **Observability** | $0 (built-in) | $0 (OpenTelemetry) | $0-$299/mo (LangSmith) | LangSmith: $39-$299/mo for prod features |
| **AI Gateway** | N/A | $0 (usage tracking) | N/A | Vercel Gateway included, future billing TBD |
| **Caching** | $0 (native) | $0-$99/mo (Portkey) | Redis hosting costs | Forge: No external dependencies |
| **Vector Store** | External (user choice) | External (user choice) | External (user choice) | Pinecone: $0-$90/mo, Weaviate: self-host |
| **100K req/day** | **LLM costs only** | **LLM + $0-$599/mo** | **LLM + $0-$598/mo** | Depends on optional services |

**Cost Scenarios:**

1. **Minimal Setup** (no guardrails/observability):
   - Forge: $0 infrastructure cost
   - Vercel: $0 infrastructure cost  
   - LangChain: $0-$40/mo (Redis for caching)

2. **Production Setup** (guardrails + observability + caching):
   - Forge: **$0 infrastructure cost** (all built-in)
   - Vercel: **$99-$598/mo** (Portkey $99-$499 + optional LangSmith)
   - LangChain: **$39-$598/mo** (LangSmith + optional guardrails + Redis)

**Annual Savings**: Forge saves **$1,188-$7,176/year** vs Vercel/LangChain production setups.

### Decision Matrix

| Your Priority | Recommended SDK | Reason |
|--------------|----------------|---------|
| **Cost optimization** | Forge AI SDK | Zero infrastructure costs, no external dependencies |
| **Go ecosystem** | Forge AI SDK | Native Go, goroutines, single binary deployment |
| **Multi-modal AI** (audio/video) | Vercel AI SDK v5 | Best-in-class support for audio/video streaming |
| **Frontend integration** | Vercel AI SDK v5 | React/Vue/Svelte hooks, Server Components |
| **Python ML stack** | LangChain | Mature ecosystem, 100+ integrations, LangSmith |
| **Complex workflows** | LangChain | LangGraph production-ready, proven patterns |
| **High throughput** | Forge AI SDK | Native concurrency, 1000+ req/sec, low latency |
| **Air-gapped/Enterprise** | Forge AI SDK | Works offline, no external API dependencies |
| **Rapid prototyping** | Vercel AI SDK v5 | 100+ models via Gateway, excellent DX |
| **Research/Experimentation** | LangChain | Flexible, extensive documentation, community |

---

## 📖 Documentation

### Core Documentation
- [API Reference](./docs/api.md)
- [Architecture](./docs/architecture.md)
- [Examples](./examples/)
- [Testing Guide](./docs/testing.md)
- [Migration Guide](./docs/migration.md)

### Agent Patterns
- [Agent Patterns Overview](./docs/AGENT_PATTERNS.md)
- [ReAct Agents Guide](./docs/REACT_AGENTS.md)
- [Plan-Execute Agents Guide](./docs/PLAN_EXECUTE_AGENTS.md)
- [Reflection & Replanning](./docs/REFLECTION_AND_REPLANNING.md)

---

## 🧪 Testing

```bash
# Run all tests
go test ./...

# With coverage
go test -cover ./...

# Benchmark
go test -bench=. -benchmem ./...

# Race detection
go test -race ./...
```

---

## 🛠️ Development

```bash
# Install dependencies
go mod download

# Run linters
golangci-lint run

# Format code
go fmt ./...

# Generate docs
godoc -http=:6060
```

---

## 📝 License

MIT License - see [LICENSE](LICENSE) for details.

---

## 🌟 Star History

If this project helped you, please ⭐ star it on GitHub!

---

## 🤝 Contributing

Contributions welcome! See [CONTRIBUTING.md](CONTRIBUTING.md).

---

## 💬 Support

- 📧 Email: rex@xraph.com
- 💬 Discord: [Join our community](https://discord.gg/xraph)
- 📖 Docs: [forge.xraph.com/docs](https://forge.xraph.com/docs)

---

**Built with ❤️ by the XRaph Team**
