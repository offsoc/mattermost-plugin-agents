// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package semanticcache

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/mattermost/mattermost-plugin-ai/openai"
)

const (
	embeddingDimensionsOpenAI = 1536
	embeddingDimensionsNomic  = 768
)

// createEmbedder returns a function that generates embeddings
// Supports OpenAI and OpenAI-compatible providers (Ollama, vLLM, etc.)
func createEmbedder() func(string) ([]float32, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	apiURL := os.Getenv("OPENAI_API_URL")

	// If API URL is set, use OpenAI-compatible mode (Ollama, vLLM, etc.)
	// Otherwise use standard OpenAI
	if apiURL != "" {
		return createCompatibleEmbedder(apiKey, apiURL)
	}

	if apiKey == "" {
		// Return a dummy embedder that always fails
		return func(string) ([]float32, error) {
			return nil, fmt.Errorf("OPENAI_API_KEY not configured")
		}
	}

	return createOpenAIEmbedder(apiKey)
}

// createOpenAIEmbedder creates an embedder using standard OpenAI API
func createOpenAIEmbedder(apiKey string) func(string) ([]float32, error) {
	config := openai.Config{
		APIKey:              apiKey,
		EmbeddingModel:      "text-embedding-3-small",
		EmbeddingDimensions: embeddingDimensionsOpenAI,
	}

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	provider := openai.NewEmbeddings(config, httpClient)

	return func(text string) ([]float32, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return provider.CreateEmbedding(ctx, text)
	}
}

// createCompatibleEmbedder creates an embedder using OpenAI-compatible API (Ollama, vLLM, etc.)
func createCompatibleEmbedder(apiKey, apiURL string) func(string) ([]float32, error) {
	// Use nomic-embed-text for Ollama by default, can be overridden with env var
	model := os.Getenv("OPENAI_EMBEDDING_MODEL")
	if model == "" {
		model = "nomic-embed-text"
	}

	// Set dimensions based on model
	dimensions := embeddingDimensionsNomic
	if model == "text-embedding-3-small" || model == "text-embedding-3-large" {
		dimensions = embeddingDimensionsOpenAI
	}

	config := openai.Config{
		APIKey:              apiKey,
		APIURL:              apiURL,
		EmbeddingModel:      model,
		EmbeddingDimensions: dimensions,
	}

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	provider := openai.NewCompatibleEmbeddings(config, httpClient)

	return func(text string) ([]float32, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return provider.CreateEmbedding(ctx, text)
	}
}

// createEmbedderWithURL creates an embedder pointed at a specific URL (for testing)
// Uses real openai package code, but allows tests to point it at mock HTTP servers
func createEmbedderWithURL(apiURL string) func(string) ([]float32, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		apiKey = "test-key"
	}
	return createCompatibleEmbedder(apiKey, apiURL)
}

// getEmbeddingDimensions returns the expected embedding dimensions based on the configured model
func getEmbeddingDimensions() int {
	apiURL := os.Getenv("OPENAI_API_URL")

	if apiURL != "" {
		model := os.Getenv("OPENAI_EMBEDDING_MODEL")
		if model == "" {
			model = "nomic-embed-text"
		}

		if model == "text-embedding-3-small" || model == "text-embedding-3-large" {
			return embeddingDimensionsOpenAI
		}
		return embeddingDimensionsNomic
	}

	return embeddingDimensionsOpenAI
}

// ThreadEmbedder is an interface-compatible wrapper for thread grounding validation
type ThreadEmbedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
	ModelVersion() string
}

// openAIThreadEmbedder implements ThreadEmbedder using OpenAI embeddings
type openAIThreadEmbedder struct {
	provider  *openai.OpenAI
	modelName string
}

func (e *openAIThreadEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	return e.provider.CreateEmbedding(ctx, text)
}

func (e *openAIThreadEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	return e.provider.BatchCreateEmbeddings(ctx, texts)
}

func (e *openAIThreadEmbedder) ModelVersion() string {
	return e.modelName
}

// NewThreadEmbedder creates a ThreadEmbedder using OpenAI API
// Returns an embedder suitable for thread grounding validation
func NewThreadEmbedder(apiKey string) ThreadEmbedder {
	config := openai.Config{
		APIKey:              apiKey,
		EmbeddingModel:      "text-embedding-3-small",
		EmbeddingDimensions: embeddingDimensionsOpenAI,
	}

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	provider := openai.NewEmbeddings(config, httpClient)

	return &openAIThreadEmbedder{
		provider:  provider,
		modelName: "text-embedding-3-small",
	}
}
