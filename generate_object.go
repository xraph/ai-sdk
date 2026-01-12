package sdk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"strings"
	"time"

	"github.com/xraph/ai-sdk/llm"
	logger "github.com/xraph/go-utils/log"
	"github.com/xraph/go-utils/metrics"
)

// ObjectGenerator provides a fluent API for generating structured outputs
// using Go generics for type safety. It automatically generates JSON schemas
// from Go types and validates LLM responses against them.
// Renamed from GenerateObjectBuilder for clarity.
//
// Example:
//
//	type Person struct {
//	    Name string `json:"name" description:"Full name"`
//	    Age  int    `json:"age" description:"Age in years"`
//	}
//
//	result, err := sdk.NewObjectGenerator[Person](ctx, llm, logger, metrics).
//	    WithPrompt("Extract person info: {{.text}}").
//	    WithVar("text", "John Doe is 30 years old").
//	    Execute()
type ObjectGenerator[T any] struct {
	ctx        context.Context
	llmManager LLMManager
	logger     logger.Logger
	metrics    metrics.Metrics

	// Model configuration
	provider string
	model    string

	// Prompt configuration
	prompt       string
	vars         map[string]any
	systemPrompt string
	messages     []llm.ChatMessage

	// LLM parameters
	temperature *float64
	maxTokens   *int
	topP        *float64
	topK        *int
	stop        []string

	// Schema configuration
	schema         map[string]any
	schemaStrict   bool
	fallbackOnFail bool

	// Execution configuration
	timeout    time.Duration
	retries    int
	retryDelay time.Duration

	// Callbacks
	onStart    func()
	onComplete func(T)
	onError    func(error)

	// Validation
	validators []func(T) error
}

// GenerateObjectBuilder is an alias for ObjectGenerator for backward compatibility.
// Deprecated: Use ObjectGenerator instead.
// Note: For generic types, use ObjectGenerator[T] directly.
type GenerateObjectBuilder[T any] = ObjectGenerator[T]

// NewObjectGenerator creates a new generator for structured output.
func NewObjectGenerator[T any](
	ctx context.Context,
	llmManager LLMManager,
	logger logger.Logger,
	metrics metrics.Metrics,
) *ObjectGenerator[T] {
	return &ObjectGenerator[T]{
		ctx:          ctx,
		llmManager:   llmManager,
		logger:       logger,
		metrics:      metrics,
		vars:         make(map[string]any),
		timeout:      30 * time.Second,
		retries:      3,
		retryDelay:   time.Second,
		schemaStrict: true,
	}
}

// NewGenerateObjectBuilder is an alias for NewObjectGenerator for backward compatibility.
// Deprecated: Use NewObjectGenerator instead.
func NewGenerateObjectBuilder[T any](
	ctx context.Context,
	llmManager LLMManager,
	logger logger.Logger,
	metrics metrics.Metrics,
) *ObjectGenerator[T] {
	return NewObjectGenerator[T](ctx, llmManager, logger, metrics)
}

// WithProvider sets the LLM provider.
func (b *ObjectGenerator[T]) WithProvider(provider string) *ObjectGenerator[T] {
	b.provider = provider

	return b
}

// WithModel sets the model to use.
func (b *ObjectGenerator[T]) WithModel(model string) *ObjectGenerator[T] {
	b.model = model

	return b
}

// WithPrompt sets the prompt template.
func (b *ObjectGenerator[T]) WithPrompt(prompt string) *ObjectGenerator[T] {
	b.prompt = prompt

	return b
}

// WithVars sets multiple template variables.
func (b *ObjectGenerator[T]) WithVars(vars map[string]any) *ObjectGenerator[T] {
	maps.Copy(b.vars, vars)

	return b
}

// WithVar sets a single template variable.
func (b *ObjectGenerator[T]) WithVar(key string, value any) *ObjectGenerator[T] {
	b.vars[key] = value

	return b
}

// WithSystemPrompt sets the system prompt.
func (b *ObjectGenerator[T]) WithSystemPrompt(prompt string) *ObjectGenerator[T] {
	b.systemPrompt = prompt

	return b
}

// WithMessages sets conversation history.
func (b *ObjectGenerator[T]) WithMessages(messages []llm.ChatMessage) *ObjectGenerator[T] {
	b.messages = messages

	return b
}

// WithTemperature sets the temperature parameter.
func (b *ObjectGenerator[T]) WithTemperature(temp float64) *ObjectGenerator[T] {
	b.temperature = &temp

	return b
}

// WithMaxTokens sets the maximum tokens to generate.
func (b *ObjectGenerator[T]) WithMaxTokens(tokens int) *ObjectGenerator[T] {
	b.maxTokens = &tokens

	return b
}

// WithTopP sets the top-p sampling parameter.
func (b *ObjectGenerator[T]) WithTopP(topP float64) *ObjectGenerator[T] {
	b.topP = &topP

	return b
}

// WithTopK sets the top-k sampling parameter.
func (b *ObjectGenerator[T]) WithTopK(topK int) *ObjectGenerator[T] {
	b.topK = &topK

	return b
}

// WithStop sets stop sequences.
func (b *ObjectGenerator[T]) WithStop(sequences ...string) *ObjectGenerator[T] {
	b.stop = sequences

	return b
}

// WithSchema sets a custom JSON schema (overrides auto-generation).
func (b *ObjectGenerator[T]) WithSchema(schema map[string]any) *ObjectGenerator[T] {
	b.schema = schema

	return b
}

// WithSchemaStrict enables/disables strict schema validation.
func (b *ObjectGenerator[T]) WithSchemaStrict(strict bool) *ObjectGenerator[T] {
	b.schemaStrict = strict

	return b
}

// WithFallbackOnFail allows returning partial/empty results on parse failures.
func (b *ObjectGenerator[T]) WithFallbackOnFail(fallback bool) *ObjectGenerator[T] {
	b.fallbackOnFail = fallback

	return b
}

// WithTimeout sets the execution timeout.
func (b *ObjectGenerator[T]) WithTimeout(timeout time.Duration) *ObjectGenerator[T] {
	b.timeout = timeout

	return b
}

// WithRetries sets retry behavior.
func (b *ObjectGenerator[T]) WithRetries(count int, delay time.Duration) *ObjectGenerator[T] {
	b.retries = count
	b.retryDelay = delay

	return b
}

// OnStart registers a callback to run before execution.
func (b *ObjectGenerator[T]) OnStart(fn func()) *ObjectGenerator[T] {
	b.onStart = fn

	return b
}

// OnComplete registers a callback to run after successful execution.
func (b *ObjectGenerator[T]) OnComplete(fn func(T)) *ObjectGenerator[T] {
	b.onComplete = fn

	return b
}

// OnError registers a callback to run on error.
func (b *ObjectGenerator[T]) OnError(fn func(error)) *ObjectGenerator[T] {
	b.onError = fn

	return b
}

// WithValidator adds a custom validation function for the output.
func (b *ObjectGenerator[T]) WithValidator(validator func(T) error) *ObjectGenerator[T] {
	b.validators = append(b.validators, validator)

	return b
}

// Execute runs the generation and returns the structured output.
func (b *ObjectGenerator[T]) Execute() (T, error) {
	var zero T

	// Call onStart callback
	if b.onStart != nil {
		b.onStart()
	}

	// Create timeout context
	ctx, cancel := context.WithTimeout(b.ctx, b.timeout)
	defer cancel()

	// Generate JSON schema if not provided
	schema := b.schema
	if schema == nil {
		var err error

		schema, err = b.generateSchema()
		if err != nil {
			if b.onError != nil {
				b.onError(err)
			}

			if b.metrics != nil {
				b.metrics.Counter("forge.ai.sdk.generate_object.errors", metrics.WithLabel("error", "schema_generation")).Inc()
			}

			return zero, fmt.Errorf("schema generation failed: %w", err)
		}
	}

	// Render prompt with variables
	renderedPrompt, err := b.renderPrompt()
	if err != nil {
		if b.onError != nil {
			b.onError(err)
		}

		if b.metrics != nil {
			b.metrics.Counter("forge.ai.sdk.generate_object.errors", metrics.WithLabel("error", "prompt_render")).Inc()
		}

		return zero, fmt.Errorf("prompt rendering failed: %w", err)
	}

	// Build messages
	messages := b.buildMessages(renderedPrompt, schema)

	// Log execution
	if b.logger != nil {
		b.logger.Debug("Executing structured generation",
			F("provider", b.provider),
			F("model", b.model),
			F("schema_type", reflect.TypeOf(zero).Name()),
		)
	}

	// Execute with retries
	var (
		result  T
		lastErr error
	)

	for attempt := 0; attempt <= b.retries; attempt++ {
		if attempt > 0 {
			if b.logger != nil {
				b.logger.Debug("Retrying structured generation",
					F("attempt", attempt),
					F("delay", b.retryDelay),
				)
			}

			time.Sleep(b.retryDelay)
		}

		// Build LLM request
		request := llm.ChatRequest{
			Provider: b.provider,
			Model:    b.model,
			Messages: messages,
		}

		if b.temperature != nil {
			request.Temperature = b.temperature
		}

		if b.maxTokens != nil {
			request.MaxTokens = b.maxTokens
		}

		if b.topP != nil {
			request.TopP = b.topP
		}

		if b.topK != nil {
			request.TopK = b.topK
		}

		if len(b.stop) > 0 {
			request.Stop = b.stop
		}

		// Note: JSON schema enforcement would be handled by the system prompt
		// Some providers support ResponseFormat, but it's not in the base ChatRequest

		// Call LLM
		response, err := b.llmManager.Chat(ctx, request)
		if err != nil {
			lastErr = fmt.Errorf("LLM request failed: %w", err)

			continue
		}

		// Extract content
		if len(response.Choices) == 0 {
			lastErr = errors.New("no choices in response")

			continue
		}

		content := response.Choices[0].Message.Content

		// Parse JSON response
		if err := json.Unmarshal([]byte(content), &result); err != nil {
			lastErr = fmt.Errorf("JSON parse failed: %w", err)

			if b.fallbackOnFail {
				break // Return zero value
			}

			continue
		}

		// Run validators
		validationFailed := false

		for i, validator := range b.validators {
			if err := validator(result); err != nil {
				lastErr = fmt.Errorf("validation %d failed: %w", i, err)
				validationFailed = true

				break // Stop checking validators
			}
		}

		// If validation failed, retry
		if validationFailed {
			continue
		}

		// Success
		if b.logger != nil {
			b.logger.Info("Structured generation completed",
				F("type", reflect.TypeOf(result).Name()),
				F("attempts", attempt+1),
			)
		}

		if b.metrics != nil {
			b.metrics.Counter("forge.ai.sdk.generate_object.success", metrics.WithLabel("success", "true")).Inc()

			if response.Usage != nil {
				b.metrics.Histogram("forge.ai.sdk.generate_object.tokens").Observe(float64(response.Usage.TotalTokens))
			}
		}

		if b.onComplete != nil {
			b.onComplete(result)
		}

		return result, nil
	}

	// All retries failed
	if b.onError != nil {
		b.onError(lastErr)
	}

	if b.metrics != nil {
		b.metrics.Counter("forge.ai.sdk.generate_object.errors", metrics.WithLabel("error", "max_retries")).Inc()
	}

	if b.fallbackOnFail {
		if b.logger != nil {
			b.logger.Warn("Returning fallback result after failures",
				F("error", lastErr.Error()),
			)
		}

		return result, nil // Return whatever we have (possibly zero value)
	}

	return zero, fmt.Errorf("generation failed after %d attempts: %w", b.retries+1, lastErr)
}

// renderPrompt renders the prompt template with variables.
func (b *ObjectGenerator[T]) renderPrompt() (string, error) {
	if len(b.vars) == 0 {
		return b.prompt, nil
	}

	result := b.prompt
	for key, value := range b.vars {
		placeholder := fmt.Sprintf("{{.%s}}", key)
		result = strings.ReplaceAll(result, placeholder, fmt.Sprint(value))
	}

	return result, nil
}

// buildMessages constructs the message array for the LLM request.
func (b *ObjectGenerator[T]) buildMessages(prompt string, schema map[string]any) []llm.ChatMessage {
	messages := make([]llm.ChatMessage, 0)

	// Add custom messages first
	if len(b.messages) > 0 {
		messages = append(messages, b.messages...)
	}

	// Add system prompt with schema instructions
	systemPrompt := b.systemPrompt
	if systemPrompt == "" {
		systemPrompt = "You are a helpful assistant that returns structured data in JSON format."
	}

	// Enhance system prompt with schema
	schemaJSON, _ := json.MarshalIndent(schema, "", "  ")
	systemPrompt += fmt.Sprintf("\n\nYou must return a valid JSON object that matches this schema:\n```json\n%s\n```", string(schemaJSON))

	messages = append(messages, llm.ChatMessage{
		Role:    "system",
		Content: systemPrompt,
	})

	// Add user prompt
	if prompt != "" {
		messages = append(messages, llm.ChatMessage{
			Role:    "user",
			Content: prompt,
		})
	}

	return messages
}

// generateSchema generates a JSON schema from the Go type T.
func (b *ObjectGenerator[T]) generateSchema() (map[string]any, error) {
	var zero T

	t := reflect.TypeOf(zero)

	// Handle pointer types
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Only support structs for now
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("type %s is not a struct", t.Name())
	}

	schema := map[string]any{
		"type":       "object",
		"properties": make(map[string]any),
		"required":   make([]string, 0),
	}

	properties := schema["properties"].(map[string]any)
	required := make([]string, 0)

	// Iterate over struct fields
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Get JSON tag
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		// Parse JSON tag
		jsonName := strings.Split(jsonTag, ",")[0]
		if jsonName == "" {
			jsonName = field.Name
		}

		// Get description from tag
		description := field.Tag.Get("description")

		// Generate property schema
		propSchema := b.generatePropertySchema(field.Type, description)
		properties[jsonName] = propSchema

		// Check if required (no omitempty tag)
		if !strings.Contains(jsonTag, "omitempty") {
			required = append(required, jsonName)
		}
	}

	if len(required) > 0 {
		schema["required"] = required
	}

	return schema, nil
}

// generatePropertySchema generates a schema for a single property.
func (b *ObjectGenerator[T]) generatePropertySchema(t reflect.Type, description string) map[string]any {
	schema := make(map[string]any)

	if description != "" {
		schema["description"] = description
	}

	// Handle pointer types
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.String:
		schema["type"] = "string"

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		schema["type"] = "integer"

	case reflect.Float32, reflect.Float64:
		schema["type"] = "number"

	case reflect.Bool:
		schema["type"] = "boolean"

	case reflect.Slice, reflect.Array:
		schema["type"] = "array"
		schema["items"] = b.generatePropertySchema(t.Elem(), "")

	case reflect.Map:
		schema["type"] = "object"
		schema["additionalProperties"] = b.generatePropertySchema(t.Elem(), "")

	case reflect.Struct:
		// Nested struct - recursively generate schema
		schema["type"] = "object"
		props := make(map[string]any)
		required := make([]string, 0)

		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if !field.IsExported() {
				continue
			}

			jsonTag := field.Tag.Get("json")
			if jsonTag == "" || jsonTag == "-" {
				continue
			}

			jsonName := strings.Split(jsonTag, ",")[0]
			if jsonName == "" {
				jsonName = field.Name
			}

			fieldDesc := field.Tag.Get("description")
			props[jsonName] = b.generatePropertySchema(field.Type, fieldDesc)

			if !strings.Contains(jsonTag, "omitempty") {
				required = append(required, jsonName)
			}
		}

		schema["properties"] = props
		if len(required) > 0 {
			schema["required"] = required
		}

	default:
		schema["type"] = "string" // Fallback
	}

	return schema
}
