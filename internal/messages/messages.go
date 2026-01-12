// Package messages provides shared message building utilities.
package messages

import (
	"github.com/xraph/ai-sdk/llm"
)

// Build constructs a slice of ChatMessages from optional system prompt, history, and user prompt.
// Messages are added in order: system (if non-empty), history, user (if non-empty).
func Build(systemPrompt string, history []llm.ChatMessage, userPrompt string) []llm.ChatMessage {
	// Pre-calculate capacity
	capacity := len(history)
	if systemPrompt != "" {
		capacity++
	}
	if userPrompt != "" {
		capacity++
	}

	messages := make([]llm.ChatMessage, 0, capacity)

	// Add system prompt if provided
	if systemPrompt != "" {
		messages = append(messages, llm.ChatMessage{
			Role:    "system",
			Content: systemPrompt,
		})
	}

	// Add history
	messages = append(messages, history...)

	// Add user prompt if provided
	if userPrompt != "" {
		messages = append(messages, llm.ChatMessage{
			Role:    "user",
			Content: userPrompt,
		})
	}

	return messages
}

// BuildWithHistoryFirst constructs messages with history before system prompt.
// This is used when messages need to be prepended before the system prompt.
func BuildWithHistoryFirst(history []llm.ChatMessage, systemPrompt string, userPrompt string) []llm.ChatMessage {
	// Pre-calculate capacity
	capacity := len(history)
	if systemPrompt != "" {
		capacity++
	}
	if userPrompt != "" {
		capacity++
	}

	messages := make([]llm.ChatMessage, 0, capacity)

	// Add history first
	messages = append(messages, history...)

	// Add system prompt if provided
	if systemPrompt != "" {
		messages = append(messages, llm.ChatMessage{
			Role:    "system",
			Content: systemPrompt,
		})
	}

	// Add user prompt if provided
	if userPrompt != "" {
		messages = append(messages, llm.ChatMessage{
			Role:    "user",
			Content: userPrompt,
		})
	}

	return messages
}
