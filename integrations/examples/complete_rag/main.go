package main

import (
	"context"
	"fmt"
	"log"
	"os"

	sdk "github.com/xraph/ai-sdk"
	"github.com/xraph/ai-sdk/integrations/embeddings/openai"
	"github.com/xraph/ai-sdk/integrations/vectorstores/pinecone"
	logger "github.com/xraph/go-utils/log"
)

func main() {
	ctx := context.Background()

	// Initialize logger
	log := logger.NewLogger(logger.LevelInfo)

	// 1. Create OpenAI embeddings
	embedder, err := openai.NewOpenAIEmbeddings(openai.Config{
		APIKey: os.Getenv("OPENAI_API_KEY"),
		Model:  openai.ModelTextEmbedding3Small,
	})
	if err != nil {
		log.Fatal("Failed to create embedder", logger.Error(err))
	}

	fmt.Printf("Using OpenAI embeddings (%s, %d dimensions)\n",
		embedder.Model(), embedder.Dimensions())

	// 2. Create Pinecone vector store
	vectorStore, err := pinecone.NewPineconeVectorStore(ctx, pinecone.Config{
		APIKey:    os.Getenv("PINECONE_API_KEY"),
		IndexName: os.Getenv("PINECONE_INDEX"),
		Namespace: "demo",
	})
	if err != nil {
		log.Fatal("Failed to create vector store", logger.Error(err))
	}
	defer vectorStore.Close()

	// 3. Create RAG
	rag := sdk.NewRAG(vectorStore, embedder, log, nil, &sdk.RAGOptions{
		TopK:             10,
		SimilarityThresh: 0.7,
		ChunkSize:        512,
		ChunkOverlap:     50,
	})

	fmt.Println("\n=== RAG System Initialized ===")

	// 4. Index some documents
	documents := []sdk.Document{
		{
			ID:      "doc1",
			Content: "Artificial Intelligence (AI) is transforming software development. Machine learning models can now generate code, find bugs, and optimize performance.",
			Metadata: map[string]any{
				"title":    "AI in Software Development",
				"category": "technology",
				"author":   "Tech Writer",
			},
		},
		{
			ID:      "doc2",
			Content: "Vector databases are essential for AI applications. They enable similarity search across high-dimensional embeddings, powering semantic search and RAG systems.",
			Metadata: map[string]any{
				"title":    "Vector Databases Explained",
				"category": "technology",
				"author":   "DB Expert",
			},
		},
		{
			ID:      "doc3",
			Content: "Go is an excellent language for building AI infrastructure. Its performance, concurrency model, and robust standard library make it ideal for production AI systems.",
			Metadata: map[string]any{
				"title":    "Go for AI Systems",
				"category": "programming",
				"author":   "Go Developer",
			},
		},
	}

	fmt.Println("\n=== Indexing Documents ===")
	for _, doc := range documents {
		if err := rag.IndexDocument(ctx, doc); err != nil {
			log.Error("Failed to index document", logger.Error(err))
			continue
		}
		fmt.Printf("✓ Indexed: %s\n", doc.Metadata["title"])
	}

	// 5. Perform queries
	fmt.Println("\n=== Performing Queries ===")

	queries := []string{
		"What is the role of AI in software development?",
		"How do vector databases work?",
		"Why use Go for AI systems?",
	}

	for i, query := range queries {
		fmt.Printf("\nQuery %d: %s\n", i+1, query)

		result, err := rag.Retrieve(ctx, query)
		if err != nil {
			log.Error("Failed to retrieve", logger.Error(err))
			continue
		}

		fmt.Printf("Retrieved %d documents in %v\n", len(result.Documents), result.Took)

		for j, doc := range result.Documents {
			fmt.Printf("  %d. Score: %.4f | %s\n",
				j+1,
				doc.Score,
				doc.Document.Content[:min(100, len(doc.Document.Content))]+"...")

			if title, ok := doc.Document.Metadata["title"]; ok {
				fmt.Printf("     Title: %s\n", title)
			}
		}
	}

	// 6. Generate with context
	fmt.Println("\n=== Generate with RAG Context ===")

	// Note: Requires LLM provider setup
	// This is a placeholder showing how to use the retrieved context
	query := "Explain how AI is used in software development"
	result, err := rag.Retrieve(ctx, query)
	if err != nil {
		log.Fatal("Failed to retrieve", logger.Error(err))
	}

	fmt.Printf("\nContext for query: %s\n", query)
	fmt.Printf("Retrieved %d relevant documents\n", len(result.Documents))
	fmt.Println("\nYou would now pass this context to your LLM for generation...")

	// Example with actual LLM (commented out)
	/*
		llmManager := llm.NewLLMManager(llm.Config{
			Providers: map[string]llm.ProviderConfig{
				"openai": {
					APIKey: os.Getenv("OPENAI_API_KEY"),
				},
			},
		})

		generator := sdk.New(sdk.Options{
			LLM:    llmManager,
			Logger: log,
		})

		response, err := rag.GenerateWithContext(ctx, query, generator)
		if err != nil {
			log.Fatal("Failed to generate", logger.Error(err))
		}

		fmt.Printf("\nGenerated Response:\n%s\n", response.Content)
	*/
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
