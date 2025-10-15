// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package baseline

import (
	"context"
	"time"

	"github.com/mattermost/mattermost-plugin-ai/llm"
	"github.com/mattermost/mattermost-plugin-ai/roles/dev"
	"github.com/mattermost/mattermost-plugin-ai/roles/pm"
	"github.com/mattermost/mattermost/server/public/model"
)

const (
	// BaselineSystemPrompt is the minimal system prompt used for baseline comparison
	BaselineSystemPrompt = "You are a helpful AI assistant."
)

// LLMBot implements a raw LLM with minimal prompt and no enhancements.
// This serves as the baseline for comparison against enhanced bots.
type LLMBot struct {
	llmClient      llm.LanguageModel
	prompts        *llm.Prompts        // Optional: for role-prompt baseline comparisons
	useRolePrompts bool                // Whether to use role prompts instead of minimal prompt
	intentDetector func(string) string // Optional: role-specific intent detection
	name           string
	LastRequest    *llm.CompletionRequest // Store last request for debugging
	LastResponse   string                 // Store last response for debugging
}

// NewBaselineBot creates a new baseline bot with the specified LLM.
func NewBaselineBot(llmClient llm.LanguageModel) *LLMBot {
	return &LLMBot{
		llmClient:      llmClient,
		name:           "baseline-llm",
		useRolePrompts: false,
	}
}

// NewBaselineBotWithName creates a baseline bot with a custom name for identification.
func NewBaselineBotWithName(llmClient llm.LanguageModel, name string) *LLMBot {
	return &LLMBot{
		llmClient:      llmClient,
		name:           name,
		useRolePrompts: false,
	}
}

// NewRolePromptBaselineBot creates a baseline bot that uses role-specific prompts without tools.
// This provides a fair comparison: same prompts, but no data source tools.
// The intentDetector function is used to detect intent for role-specific prompt formatting.
func NewRolePromptBaselineBot(llmClient llm.LanguageModel, prompts *llm.Prompts, intentDetector func(string) string, name string) *LLMBot {
	return &LLMBot{
		llmClient:      llmClient,
		prompts:        prompts,
		useRolePrompts: true,
		intentDetector: intentDetector,
		name:           name,
	}
}

// NewPMPromptBaselineBot creates a baseline bot that uses PM prompts without tools.
// This provides a fair comparison: same prompts, but no data source tools.
func NewPMPromptBaselineBot(llmClient llm.LanguageModel, prompts *llm.Prompts, name string) *LLMBot {
	return NewRolePromptBaselineBot(llmClient, prompts, pm.DetectIntent, name)
}

// NewDevPromptBaselineBot creates a baseline bot that uses Dev prompts without tools.
// This provides a fair comparison: same prompts, but no data source tools.
func NewDevPromptBaselineBot(llmClient llm.LanguageModel, prompts *llm.Prompts, name string) *LLMBot {
	return NewRolePromptBaselineBot(llmClient, prompts, dev.DetectIntent, name)
}

// Respond generates a response using the LLM with either minimal or role-specific prompts.
func (bb *LLMBot) Respond(ctx context.Context, msg string) (Answer, error) {
	start := time.Now()

	var systemPrompt string
	var metadata map[string]interface{}

	if bb.useRolePrompts && bb.prompts != nil && bb.intentDetector != nil {
		// Use role-specific prompts without tools - same prompt generation as enhanced bot
		// but no data source augmentation
		intent := bb.intentDetector(msg)

		// Create context with minimal required fields for template rendering
		llmContext := llm.NewContext()
		llmContext.RequestingUser = &model.User{
			Username: "testuser",
		}
		llmContext.Channel = &model.Channel{}

		// Format the role prompt using the same template system as enhanced bot
		formattedPrompt, err := bb.prompts.Format(intent, llmContext)
		if err != nil {
			// Fall back to minimal prompt if formatting fails
			systemPrompt = BaselineSystemPrompt
			metadata = map[string]interface{}{
				MetadataKeyModelType: ModelTypeBaseline,
				"prompt_error":       err.Error(),
			}
		} else {
			systemPrompt = formattedPrompt
			metadata = map[string]interface{}{
				MetadataKeyModelType: "role-prompt-baseline",
				"intent":             intent,
			}
		}
	} else {
		// Use minimal baseline prompt
		systemPrompt = BaselineSystemPrompt
		metadata = map[string]interface{}{
			MetadataKeyModelType: ModelTypeBaseline,
		}
	}

	request := llm.CompletionRequest{
		Posts: []llm.Post{
			{
				Role:    llm.PostRoleSystem,
				Message: systemPrompt,
			},
			{
				Role:    llm.PostRoleUser,
				Message: msg,
			},
		},
		Context: llm.NewContext(),
	}

	// Store request for debugging
	bb.LastRequest = &request

	response, err := bb.llmClient.ChatCompletionNoStream(request)
	if err != nil {
		return Answer{}, err
	}

	// Store response for debugging
	bb.LastResponse = response

	latency := time.Since(start)

	return Answer{
		Text:     response,
		Latency:  latency,
		Tokens:   TokenUsage{},
		Metadata: metadata,
	}, nil
}

// Name returns the bot's identifier for results tracking.
func (bb *LLMBot) Name() string {
	return bb.name
}
