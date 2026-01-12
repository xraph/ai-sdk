package benchmarks

import (
	"context"
	"testing"

	"github.com/xraph/ai-sdk/integrations/embeddings/openai"
)

// BenchmarkEmbeddings_OpenAI benchmarks OpenAI embeddings.
// Note: These benchmarks make real API calls and require an API key.
// Run with: go test -bench=BenchmarkEmbeddings -benchtime=10x
func BenchmarkEmbeddings_OpenAI(b *testing.B) {
	b.Skip("requires valid OpenAI API key and incurs costs")

	embedder, err := openai.NewOpenAIEmbeddings(openai.OpenAIConfig{
		APIKey: "your-api-key", // Set via environment variable in practice
		Model:  "text-embedding-3-small",
	})
	if err != nil {
		b.Fatal(err)
	}

	texts := []string{
		"The quick brown fox jumps over the lazy dog",
		"Machine learning is a subset of artificial intelligence",
		"Natural language processing enables computers to understand text",
	}

	b.Run("Embed/Single", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = embedder.Embed(context.Background(), texts[:1])
		}
	})

	b.Run("Embed/Batch3", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = embedder.Embed(context.Background(), texts)
		}
	})

	b.Run("Embed/Batch10", func(b *testing.B) {
		largeTexts := make([]string, 10)
		for i := 0; i < 10; i++ {
			largeTexts[i] = texts[i%3]
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = embedder.Embed(context.Background(), largeTexts)
		}
	})
}

// Note: Similar benchmarks would be added for Cohere and other embedding providers.
// They are commented out as they require API keys and incur costs.

// BenchmarkEmbeddings_Comparison compares different embedding providers.
func BenchmarkEmbeddings_Comparison(b *testing.B) {
	b.Skip("requires API keys for multiple providers")

	texts := []string{
		"The quick brown fox jumps over the lazy dog",
		"Machine learning is a subset of artificial intelligence",
	}

	b.Run("OpenAI/text-embedding-3-small", func(b *testing.B) {
		embedder, _ := openai.NewOpenAIEmbeddings(openai.OpenAIConfig{
			APIKey: "key",
			Model:  "text-embedding-3-small",
		})
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = embedder.Embed(context.Background(), texts)
		}
	})

	// Add more providers here
	// b.Run("Cohere/embed-english-v3.0", ...)
	// b.Run("HuggingFace/...", ...)
}
