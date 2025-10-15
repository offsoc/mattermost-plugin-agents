// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package evals

import (
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
	"github.com/stretchr/testify/require"
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

func NewEval() (*Eval, error) {
	prompts, err := llm.NewPrompts(prompts.PromptsFolder)
	if err != nil {
		return nil, err
	}

	// Determine model from environment or default
	defaultModel := "gpt-4o"
	if envModel := os.Getenv("TEST_MODEL"); envModel != "" {
		defaultModel = envModel
	}

	httpClient := &http.Client{}

	var provider llm.LanguageModel

	// Route to appropriate provider based on model name
	if strings.HasPrefix(strings.ToLower(defaultModel), "claude-") {
		// Use Anthropic for Claude models
		config := llm.ServiceConfig{
			APIKey:       os.Getenv("ANTHROPIC_API_KEY"),
			DefaultModel: defaultModel,
			Type:         "anthropic",
		}
		provider = anthropic.New(config, httpClient, false)
	} else {
		// Use OpenAI for other models (default)
		config := openai.Config{
			APIKey:           os.Getenv("OPENAI_API_KEY"),
			DefaultModel:     defaultModel,
			StreamingTimeout: 20 * time.Second,
		}
		provider = openai.New(config, httpClient)
	}

	return &Eval{
		Prompts:   prompts,
		LLM:       provider,
		GraderLLM: provider,
	}, nil
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

	eval, err := NewEval()
	require.NoError(t, err)

	e := &EvalT{T: t, Eval: eval}

	t.Run(name, func(t *testing.T) {
		e.T = t
		for i := range numEvals {
			e.runNumber = i
			f(e)
		}
	})
}
