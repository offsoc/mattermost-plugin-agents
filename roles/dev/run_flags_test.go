// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package dev_test

import (
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/mattermost/mattermost-plugin-ai/evals"
	"github.com/mattermost/mattermost-plugin-ai/llm"
	"github.com/mattermost/mattermost-plugin-ai/prompts"
	"github.com/mattermost/mattermost-plugin-ai/roles"
	"github.com/stretchr/testify/require"
)

// Re-export shared flags for convenience
var (
	debugFlag              = evals.DebugFlag
	warnFlag               = evals.WarnFlag
	temperatureFlag        = evals.TemperatureFlag
	timeoutFlag            = evals.TimeoutFlag
	scenarioFlag           = evals.ScenarioFlag
	thresholdFlag          = evals.ThresholdFlag
	levelFlag              = evals.LevelFlag
	graderModelFlag        = evals.GraderModelFlag
	rolePromptBaselineFlag = evals.RolePromptBaselineFlag
	groundingFlag          = evals.GroundingFlag
	savePromptsFlag        = evals.SavePromptsFlag
	saveOutputDir          = evals.SaveOutputDir
	comparisonMode         = evals.ComparisonMode
)

// createLLMProvider uses the shared implementation
func createLLMProvider(modelName string, streamingTimeout time.Duration, temperature *float32, httpClient *http.Client) llm.LanguageModel {
	return evals.CreateLLMProvider(modelName, streamingTimeout, temperature, httpClient)
}

// createDevBotEval creates a custom evaluation instance with devbot-specific configuration
func createDevBotEval(t *testing.T) (*evals.EvalT, int) {
	numEvals := evals.NumEvalsOrSkip(t)

	// Configure timeout
	streamingTimeout := *timeoutFlag
	if streamingTimeout <= 0 {
		streamingTimeout = 60 * time.Second
	}

	// Configure temperature
	var temperature *float32
	if *temperatureFlag >= 0 {
		temp := float32(*temperatureFlag)
		temperature = &temp
	}

	// Create LLM with devbot-specific configuration
	httpClient := &http.Client{
		Timeout: streamingTimeout * 2,
	}

	// Determine model from environment or default
	defaultModel := "gpt-4o"
	if envModel := os.Getenv("TEST_MODEL"); envModel != "" {
		defaultModel = envModel
	}

	// Create main test provider
	provider := createLLMProvider(defaultModel, streamingTimeout, temperature, httpClient)
	require.NotNil(t, provider, "Failed to create LLM provider")

	// Create separate grader provider if grader-model flag is set
	var graderProvider llm.LanguageModel
	if *graderModelFlag != "" {
		graderProvider = createLLMProvider(*graderModelFlag, streamingTimeout, temperature, httpClient)
		require.NotNil(t, graderProvider, "Failed to create grader LLM provider")
	} else {
		graderProvider = provider
	}

	prompts, err := llm.NewPrompts(prompts.PromptsFolder)
	require.NoError(t, err, "Failed to load prompts")

	customEval := &evals.Eval{
		LLM:       provider,
		GraderLLM: graderProvider,
		Prompts:   prompts,
	}

	evalT := &evals.EvalT{T: t, Eval: customEval}

	// Log configuration
	t.Logf("=== DEVBOT EVAL CONFIGURATION ===")
	switch {
	case *debugFlag:
		t.Logf("Logging: DEBUG (includes all detailed operations + progress tracking)")
	case *warnFlag:
		t.Logf("Logging: WARN (high-level progress without detailed operations)")
	default:
		t.Logf("Logging: MINIMAL (use --warn for progress, --debug for full details)")
	}
	if temperature != nil {
		t.Logf("Temperature: %.2f (set via --temperature flag)", *temperature)
	} else {
		t.Logf("Temperature: using model default (use --temperature to override)")
	}
	t.Logf("Streaming timeout: %v (set via --timeout flag)", streamingTimeout)
	t.Logf("Scenario subset: %s (set via --scenarios flag)", *scenarioFlag)
	t.Logf("Threshold mode: %s (set via --threshold flag)", *thresholdFlag)
	t.Logf("Scenario type: MATTERMOST-CENTRIC (DevBot is always MM-specific)")
	if *graderModelFlag != "" {
		t.Logf("Grader model: %s (set via --grader-model flag)", *graderModelFlag)
	} else {
		t.Logf("Grader model: using same as test model (use --grader-model to override)")
	}
	if *rolePromptBaselineFlag {
		t.Logf("Baseline type: DEV-PROMPT (same prompts as enhanced but without data sources)")
	} else {
		t.Logf("Baseline type: VANILLA (minimal 'helpful assistant' prompt)")
	}
	if *groundingFlag {
		t.Logf("Grounding validation: ENABLED (citation and metadata analysis)")
	} else {
		t.Logf("Grounding validation: DISABLED (use --grounding to enable)")
	}
	t.Logf("===============================")

	return evalT, numEvals
}

// createTestLogger creates a TestLogger from the global debug and warn flags
func createTestLogger(t *testing.T, modelPrefix string) *roles.TestLogger {
	return roles.NewTestLogger(t, *debugFlag, *warnFlag, modelPrefix)
}
