// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package evals

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/mattermost/mattermost-plugin-ai/anthropic"
	"github.com/mattermost/mattermost-plugin-ai/llm"
	"github.com/mattermost/mattermost-plugin-ai/openai"
	"github.com/mattermost/mattermost-plugin-ai/prompts"
)

type EvalT struct {
	*testing.T
	*Eval
}

type Eval struct {
	LLM       llm.LanguageModel
	GraderLLM llm.LanguageModel
	Prompts   *llm.Prompts

	runNumber int
}

// ProviderConfig holds provider-specific configuration
type ProviderConfig struct {
	APIKey  string
	Model   string
	APIURL  string // For compatible providers like Azure
	Timeout time.Duration
}

// getProviderConfig reads environment variables for a specific provider
func getProviderConfig(providerName string) (ProviderConfig, error) {
	config := ProviderConfig{
		Timeout: 20 * time.Second,
	}

	switch strings.ToLower(providerName) {
	case "openai":
		config.APIKey = os.Getenv("OPENAI_API_KEY")
		config.Model = os.Getenv("OPENAI_MODEL")
		if config.Model == "" {
			config.Model = "gpt-4o"
		}
		if config.APIKey == "" {
			return config, errors.New("OPENAI_API_KEY environment variable is not set")
		}

	case "anthropic":
		config.APIKey = os.Getenv("ANTHROPIC_API_KEY")
		config.Model = os.Getenv("ANTHROPIC_MODEL")
		if config.Model == "" {
			config.Model = "claude-sonnet-4-20250514"
		}
		if config.APIKey == "" {
			return config, errors.New("ANTHROPIC_API_KEY environment variable is not set")
		}

	case "azure":
		config.APIKey = os.Getenv("AZURE_OPENAI_API_KEY")
		config.APIURL = os.Getenv("AZURE_OPENAI_ENDPOINT")
		config.Model = os.Getenv("AZURE_OPENAI_MODEL")
		if config.Model == "" {
			config.Model = "gpt-4o"
		}
		if config.APIKey == "" {
			return config, errors.New("AZURE_OPENAI_API_KEY environment variable is not set")
		}
		if config.APIURL == "" {
			return config, errors.New("AZURE_OPENAI_ENDPOINT environment variable is not set")
		}

	case "openaicompatible":
		config.APIKey = os.Getenv("OPENAI_COMPATIBLE_API_KEY")
		config.APIURL = os.Getenv("OPENAI_COMPATIBLE_API_URL")
		config.Model = os.Getenv("OPENAI_COMPATIBLE_MODEL")
		if config.Model == "" {
			return config, errors.New("OPENAI_COMPATIBLE_MODEL environment variable is not set")
		}
		if config.APIURL == "" {
			return config, errors.New("OPENAI_COMPATIBLE_API_URL environment variable is not set")
		}
		// API key is optional for local LLMs

	default:
		return config, fmt.Errorf("unknown provider: %s", providerName)
	}

	return config, nil
}

// createProvider creates an LLM provider based on the provider name and config
func createProvider(providerName string, config ProviderConfig) (llm.LanguageModel, error) {
	httpClient := &http.Client{}

	switch strings.ToLower(providerName) {
	case "openai":
		provider := openai.New(openai.Config{
			APIKey:           config.APIKey,
			DefaultModel:     config.Model,
			StreamingTimeout: config.Timeout,
		}, httpClient)
		if provider == nil {
			return nil, errors.New("failed to create OpenAI provider")
		}
		return provider, nil

	case "anthropic":
		provider := anthropic.New(llm.ServiceConfig{
			APIKey:       config.APIKey,
			DefaultModel: config.Model,
		}, httpClient)
		if provider == nil {
			return nil, errors.New("failed to create Anthropic provider")
		}
		return provider, nil

	case "azure":
		provider := openai.NewAzure(openai.Config{
			APIKey:           config.APIKey,
			APIURL:           config.APIURL,
			DefaultModel:     config.Model,
			StreamingTimeout: config.Timeout,
		}, httpClient)
		if provider == nil {
			return nil, errors.New("failed to create Azure OpenAI provider")
		}
		return provider, nil

	case "openaicompatible":
		provider := openai.NewCompatible(openai.Config{
			APIKey:           config.APIKey,
			APIURL:           config.APIURL,
			DefaultModel:     config.Model,
			StreamingTimeout: config.Timeout,
		}, httpClient)
		if provider == nil {
			return nil, errors.New("failed to create OpenAI Compatible provider")
		}
		return provider, nil

	default:
		return nil, fmt.Errorf("unknown provider: %s", providerName)
	}
}

func NewEval() (*Eval, error) {
	// Default to OpenAI for backward compatibility
	return NewEvalWithProvider("openai")
}

// NewEvalWithProvider creates an Eval instance with a specific provider
func NewEvalWithProvider(providerName string) (*Eval, error) {
	// Setup prompts
	prompts, err := llm.NewPrompts(prompts.PromptsFolder)
	if err != nil {
		return nil, err
	}

	// Get provider configuration
	config, err := getProviderConfig(providerName)
	if err != nil {
		return nil, err
	}

	// Create provider
	provider, err := createProvider(providerName, config)
	if err != nil {
		return nil, err
	}

	// Setup grader LLM (separate from main LLM)
	graderLLM, err := createGraderLLM()
	if err != nil {
		return nil, fmt.Errorf("failed to create grader LLM: %w", err)
	}

	return &Eval{
		Prompts:   prompts,
		LLM:       provider,
		GraderLLM: graderLLM,
	}, nil
}

// createGraderLLM creates a separate LLM for grading based on environment variables
// Defaults to OpenAI with gpt-5 model if not specified
func createGraderLLM() (llm.LanguageModel, error) {
	// Get grader provider name from environment, default to "openai"
	graderProvider := os.Getenv("GRADER_LLM_PROVIDER")
	if graderProvider == "" {
		graderProvider = "openai"
	}

	// Get provider configuration (uses existing API key env vars)
	graderConfig, err := getProviderConfig(graderProvider)
	if err != nil {
		return nil, err
	}

	// Override model if GRADER_LLM_MODEL is set, otherwise default to gpt-5
	graderModel := os.Getenv("GRADER_LLM_MODEL")
	if graderModel != "" {
		graderConfig.Model = graderModel
	} else if graderProvider == "openai" {
		// Default to gpt-5 for OpenAI grader
		graderConfig.Model = "gpt-5"
	}

	// Create grader provider
	return createProvider(graderProvider, graderConfig)
}

func NumEvalsOrSkip(t *testing.T) int {
	t.Helper()
	numEvals, err := strconv.Atoi(os.Getenv("GOEVALS"))
	if err != nil || numEvals < 1 {
		t.Skip("Skipping evals. Use GOEVALS=1 flag to run.")
	}

	return numEvals
}

func Run(t *testing.T, name string, f func(e *EvalT)) {
	numEvals := NumEvalsOrSkip(t)

	// Get list of providers to test
	providers := getProvidersToTest()

	// Run evaluations for each provider
	for _, providerName := range providers {
		providerName := providerName // Capture for closure

		// Try to create eval for this provider
		eval, err := NewEvalWithProvider(providerName)
		if err != nil {
			t.Logf("Skipping %s provider: %v", providerName, err)
			t.Error(err)
			return
		}

		e := &EvalT{T: t, Eval: eval}

		// Prefix test name with provider
		testName := fmt.Sprintf("[%s] %s", providerName, name)

		t.Run(testName, func(t *testing.T) {
			e.T = t
			for i := range numEvals {
				e.runNumber = i
				f(e)
			}
		})
	}
}

// getProvidersToTest returns the list of providers to test based on LLM_PROVIDER env var
func getProvidersToTest() []string {
	providerEnv := os.Getenv("LLM_PROVIDER")
	if providerEnv == "" {
		providerEnv = "all"
	}

	providerEnv = strings.ToLower(strings.TrimSpace(providerEnv))

	// Handle "all" case
	if providerEnv == "all" {
		return []string{"openai", "anthropic", "azure"}
	}

	// Handle comma-separated list
	if strings.Contains(providerEnv, ",") {
		providers := strings.Split(providerEnv, ",")
		result := make([]string, 0, len(providers))
		for _, p := range providers {
			p = strings.TrimSpace(p)
			if p != "" {
				result = append(result, p)
			}
		}
		return result
	}

	// Single provider
	return []string{providerEnv}
}
