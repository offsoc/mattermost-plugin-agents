// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/mattermost/mattermost-plugin-ai/bots"
	"github.com/mattermost/mattermost-plugin-ai/config"
	"github.com/mattermost/mattermost-plugin-ai/llm"
	llmmocks "github.com/mattermost/mattermost-plugin-ai/llm/mocks"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// testPlugin wraps Plugin for testing, allowing us to inject test dependencies
type testPlugin struct {
	Plugin
	testBotGetter func(username string) *bots.Bot
}

// Override GetBotByUsername for testing
func (tp *testPlugin) getBotByUsername(username string) *bots.Bot {
	if tp.testBotGetter != nil {
		return tp.testBotGetter(username)
	}
	if tp.bots != nil {
		return tp.bots.GetBotByUsername(username)
	}
	return nil
}

// createTestPlugin creates a plugin instance for testing with mock dependencies
func createTestPlugin(cfg *config.Config, botGetter func(string) *bots.Bot) *testPlugin {
	container := &config.Container{}
	container.Update(cfg)

	// Use plugintest mock API
	mockAPI := &plugintest.API{}
	// Mock logging methods with variadic args - each one can have many arguments
	// Match any number of arguments by listing mock.Anything multiple times
	mockAPI.On("LogDebug", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe().Return()
	mockAPI.On("LogInfo", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe().Return()
	mockAPI.On("LogWarn", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe().Return()
	mockAPI.On("LogError", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe().Return()

	tp := &testPlugin{
		testBotGetter: botGetter,
	}
	// Create pluginapi.Client with our mock API
	tp.Plugin.pluginAPI = pluginapi.NewClient(mockAPI, nil)
	// Access configuration through the pointer to avoid copying
	tp.Plugin.configuration = *container

	return tp
}

// Override handleGenerateCompletion to use our test bot getter
func (tp *testPlugin) handleGenerateCompletion(c *plugin.Context, request []byte, responseSchema []byte) ([]byte, error) {
	// Parse the incoming request
	var req BridgeRequest
	if err := json.Unmarshal(request, &req); err != nil {
		return nil, fmt.Errorf("invalid request format: %w", err)
	}

	// Validate required fields
	if req.Prompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}

	tp.pluginAPI.Log.Debug("Processing bridge completion request",
		"source", c.SourcePluginId,
		"prompt_length", len(req.Prompt),
	)

	// Get the default bot from configuration
	defaultBotName := tp.configuration.GetDefaultBotName()
	if defaultBotName == "" {
		return nil, fmt.Errorf("no default bot configured")
	}

	bot := tp.getBotByUsername(defaultBotName)
	if bot == nil {
		return nil, fmt.Errorf("default bot not found: %s", defaultBotName)
	}

	// Create LLM context with NO tools (as per requirements)
	llmContext := llm.NewContext(
		func(ctx *llm.Context) {
			ctx.BotName = bot.GetConfig().DisplayName
			ctx.BotUsername = bot.GetConfig().Name
			ctx.BotModel = bot.GetService().DefaultModel
			// Explicitly set Tools to nil to disable tool use
			ctx.Tools = nil
			// Add any custom parameters from the request context
			if req.Context != nil {
				ctx.Parameters = req.Context
			}
		},
	)

	// Build the completion request
	completionRequest := llm.CompletionRequest{
		Posts: []llm.Post{
			{
				Role:    llm.PostRoleUser,
				Message: req.Prompt,
			},
		},
		Context: llmContext,
	}

	// Prepare LLM options
	var opts []llm.LanguageModelOption

	// Override model if specified in request
	if req.Model != "" {
		opts = append(opts, llm.WithModel(req.Model))
	}

	// Override max tokens if specified
	if req.MaxTokens > 0 {
		opts = append(opts, llm.WithMaxGeneratedTokens(req.MaxTokens))
	}

	// Parse and apply response schema if provided
	if responseSchema != nil {
		schema, err := parseJSONSchema(responseSchema)
		if err != nil {
			return nil, fmt.Errorf("invalid response schema: %w", err)
		}

		opts = append(opts, func(cfg *llm.LanguageModelConfig) {
			cfg.JSONOutputFormat = schema
		})

		tp.pluginAPI.Log.Debug("Using structured output mode with provided schema")
	}

	tp.pluginAPI.Log.Debug("Calling LLM via bridge",
		"bot", bot.GetConfig().Name,
		"model", bot.GetService().DefaultModel,
		"structured_output", responseSchema != nil,
	)

	content, err := bot.LLM().ChatCompletionNoStream(completionRequest, opts...)
	if err != nil {
		tp.pluginAPI.Log.Error("LLM completion failed",
			"error", err,
			"bot", bot.GetConfig().Name,
		)
		return nil, fmt.Errorf("LLM completion failed: %w", err)
	}

	// If a response schema was provided, return the JSON content directly
	if responseSchema != nil {
		var jsonTest interface{}
		if err := json.Unmarshal([]byte(content), &jsonTest); err != nil {
			return nil, fmt.Errorf("LLM returned invalid JSON despite schema constraint: %w", err)
		}

		tp.pluginAPI.Log.Debug("Bridge call completed successfully with structured output")
		return []byte(content), nil
	}

	// For unstructured responses, wrap in our standard response format
	response := BridgeResponse{
		Content: content,
	}

	responseJSON, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	tp.pluginAPI.Log.Debug("Bridge call completed successfully")
	return responseJSON, nil
}

func TestExecuteBridgeCall_UnknownMethod(t *testing.T) {
	p := createTestPlugin(&config.Config{}, nil)

	ctx := &plugin.Context{
		SourcePluginId: "test-plugin",
	}

	response, err := p.ExecuteBridgeCall(ctx, "UnknownMethod", []byte(`{}`), nil)
	assert.Nil(t, response)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown method")
}

func TestHandleGenerateCompletion_InvalidJSON(t *testing.T) {
	p := createTestPlugin(&config.Config{}, nil)

	ctx := &plugin.Context{
		SourcePluginId: "test-plugin",
	}

	response, err := p.ExecuteBridgeCall(ctx, "GenerateCompletion", []byte(`invalid json`), nil)
	assert.Nil(t, response)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid request format")
}

func TestHandleGenerateCompletion_MissingPrompt(t *testing.T) {
	p := createTestPlugin(&config.Config{}, nil)

	ctx := &plugin.Context{
		SourcePluginId: "test-plugin",
	}

	requestJSON, _ := json.Marshal(BridgeRequest{
		Prompt: "",
	})

	response, err := p.ExecuteBridgeCall(ctx, "GenerateCompletion", requestJSON, nil)
	assert.Nil(t, response)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "prompt is required")
}

func TestHandleGenerateCompletion_NoDefaultBot(t *testing.T) {
	p := createTestPlugin(&config.Config{
		DefaultBotName: "",
	}, nil)

	ctx := &plugin.Context{
		SourcePluginId: "test-plugin",
	}

	requestJSON, _ := json.Marshal(BridgeRequest{
		Prompt: "Test prompt",
	})

	response, err := p.ExecuteBridgeCall(ctx, "GenerateCompletion", requestJSON, nil)
	assert.Nil(t, response)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no default bot configured")
}

func TestHandleGenerateCompletion_Success(t *testing.T) {
	// Create mock LLM
	mockLLM := &llmmocks.MockLanguageModel{}
	mockLLM.On("ChatCompletionNoStream", mock.Anything, mock.Anything).
		Return("This is the LLM response", nil)

	// Create bot configuration
	botConfig := llm.BotConfig{
		Name:        "test-bot",
		DisplayName: "Test Bot",
	}

	serviceConfig := llm.ServiceConfig{
		ID:           "test-service",
		DefaultModel: "gpt-4",
	}

	bot := bots.NewBot(botConfig, serviceConfig, &model.Bot{Username: "test-bot"}, mockLLM)

	cfg := &config.Config{
		DefaultBotName: "test-bot",
		Bots:           []llm.BotConfig{botConfig},
	}

	botGetter := func(username string) *bots.Bot {
		if username == "test-bot" {
			return bot
		}
		return nil
	}

	p := createTestPlugin(cfg, botGetter)

	ctx := &plugin.Context{
		SourcePluginId: "test-plugin",
	}

	requestJSON, _ := json.Marshal(BridgeRequest{
		Prompt: "Test prompt",
	})

	// Call the test version of handleGenerateCompletion directly
	response, err := p.handleGenerateCompletion(ctx, requestJSON, nil)
	require.NoError(t, err)
	require.NotNil(t, response)

	var bridgeResponse BridgeResponse
	err = json.Unmarshal(response, &bridgeResponse)
	require.NoError(t, err)
	assert.Equal(t, "This is the LLM response", bridgeResponse.Content)

	// Verify that the LLM was called with the correct parameters
	mockLLM.AssertExpectations(t)

	// Verify that tools were NOT passed (tools should be nil)
	mockLLM.AssertCalled(t, "ChatCompletionNoStream", mock.MatchedBy(func(req llm.CompletionRequest) bool {
		// Check that tools are nil
		if req.Context.Tools != nil {
			return false
		}
		// Check that the prompt was included
		if len(req.Posts) != 1 {
			return false
		}
		if req.Posts[0].Message != "Test prompt" {
			return false
		}
		if req.Posts[0].Role != llm.PostRoleUser {
			return false
		}
		return true
	}), mock.Anything)
}

func TestHandleGenerateCompletion_WithStructuredOutput(t *testing.T) {
	// Create mock LLM that returns structured JSON
	mockLLM := &llmmocks.MockLanguageModel{}
	structuredResponse := `{"summary": "Test summary", "key_points": ["point1", "point2"]}`
	mockLLM.On("ChatCompletionNoStream", mock.Anything, mock.Anything).
		Return(structuredResponse, nil)

	// Create bot
	botConfig := llm.BotConfig{
		Name:        "test-bot",
		DisplayName: "Test Bot",
	}

	serviceConfig := llm.ServiceConfig{
		ID:           "test-service",
		DefaultModel: "gpt-4",
	}

	bot := bots.NewBot(botConfig, serviceConfig, &model.Bot{Username: "test-bot"}, mockLLM)

	cfg := &config.Config{
		DefaultBotName: "test-bot",
		Bots:           []llm.BotConfig{botConfig},
	}

	botGetter := func(username string) *bots.Bot {
		if username == "test-bot" {
			return bot
		}
		return nil
	}

	p := createTestPlugin(cfg, botGetter)

	ctx := &plugin.Context{
		SourcePluginId: "test-plugin",
	}

	requestJSON, _ := json.Marshal(BridgeRequest{
		Prompt: "Summarize this content",
	})

	// Define response schema
	responseSchema := []byte(`{
		"type": "object",
		"properties": {
			"summary": {"type": "string"},
			"key_points": {"type": "array", "items": {"type": "string"}}
		},
		"required": ["summary"]
	}`)

	// Call the test version of handleGenerateCompletion directly
	response, err := p.handleGenerateCompletion(ctx, requestJSON, responseSchema)
	require.NoError(t, err)
	require.NotNil(t, response)

	// Verify the response is valid JSON matching the schema
	var structuredResp map[string]interface{}
	err = json.Unmarshal(response, &structuredResp)
	require.NoError(t, err)
	assert.Equal(t, "Test summary", structuredResp["summary"])
	assert.NotNil(t, structuredResp["key_points"])

	mockLLM.AssertExpectations(t)

	// Verify that JSON output format was configured
	mockLLM.AssertCalled(t, "ChatCompletionNoStream", mock.Anything, mock.MatchedBy(func(opts []llm.LanguageModelOption) bool {
		// Apply options to a config to check if JSONOutputFormat was set
		cfg := llm.LanguageModelConfig{}
		for _, opt := range opts {
			opt(&cfg)
		}
		return cfg.JSONOutputFormat != nil
	}))
}

func TestHandleGenerateCompletion_WithCustomParameters(t *testing.T) {
	mockLLM := &llmmocks.MockLanguageModel{}
	mockLLM.On("ChatCompletionNoStream", mock.Anything, mock.Anything).
		Return("Custom response", nil)

	botConfig := llm.BotConfig{
		Name:        "test-bot",
		DisplayName: "Test Bot",
	}

	serviceConfig := llm.ServiceConfig{
		ID:           "test-service",
		DefaultModel: "gpt-4",
	}

	bot := bots.NewBot(botConfig, serviceConfig, &model.Bot{Username: "test-bot"}, mockLLM)

	cfg := &config.Config{
		DefaultBotName: "test-bot",
	}

	botGetter := func(username string) *bots.Bot {
		if username == "test-bot" {
			return bot
		}
		return nil
	}

	p := createTestPlugin(cfg, botGetter)

	ctx := &plugin.Context{
		SourcePluginId: "test-plugin",
	}

	requestJSON, _ := json.Marshal(BridgeRequest{
		Prompt:    "Test prompt",
		MaxTokens: 500,
		Model:     "gpt-3.5-turbo",
		Context: map[string]interface{}{
			"custom_field": "custom_value",
		},
	})

	// Call the test version of handleGenerateCompletion directly
	response, err := p.handleGenerateCompletion(ctx, requestJSON, nil)
	require.NoError(t, err)
	require.NotNil(t, response)

	// Verify options were applied
	mockLLM.AssertCalled(t, "ChatCompletionNoStream", mock.MatchedBy(func(req llm.CompletionRequest) bool {
		// Check custom context parameters
		if req.Context.Parameters == nil {
			return false
		}
		if req.Context.Parameters["custom_field"] != "custom_value" {
			return false
		}
		return true
	}), mock.MatchedBy(func(opts []llm.LanguageModelOption) bool {
		// Apply options to verify MaxTokens and Model
		cfg := llm.LanguageModelConfig{}
		for _, opt := range opts {
			opt(&cfg)
		}
		return cfg.MaxGeneratedTokens == 500 && cfg.Model == "gpt-3.5-turbo"
	}))
}

func TestParseJSONSchema(t *testing.T) {
	tests := []struct {
		name        string
		schemaJSON  string
		expectError bool
	}{
		{
			name: "valid simple schema",
			schemaJSON: `{
				"type": "object",
				"properties": {
					"name": {"type": "string"}
				}
			}`,
			expectError: false,
		},
		{
			name: "valid complex schema",
			schemaJSON: `{
				"type": "object",
				"properties": {
					"summary": {"type": "string"},
					"items": {
						"type": "array",
						"items": {"type": "string"}
					}
				},
				"required": ["summary"]
			}`,
			expectError: false,
		},
		{
			name:        "invalid JSON",
			schemaJSON:  `{invalid json`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema, err := parseJSONSchema([]byte(tt.schemaJSON))
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, schema)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, schema)
			}
		})
	}
}

func TestExtractJSONFromMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "JSON in code block",
			content:  "```json\n{\"key\": \"value\"}\n```",
			expected: "{\"key\": \"value\"}",
		},
		{
			name:     "JSON in plain code block",
			content:  "```\n{\"key\": \"value\"}\n```",
			expected: "{\"key\": \"value\"}",
		},
		{
			name:     "JSON with text before",
			content:  "Here's the JSON:\n```json\n{\"key\": \"value\"}\n```",
			expected: "{\"key\": \"value\"}",
		},
		{
			name:     "Plain JSON object",
			content:  `{"key": "value", "nested": {"foo": "bar"}}`,
			expected: `{"key": "value", "nested": {"foo": "bar"}}`,
		},
		{
			name:     "JSON with extra text after",
			content:  `{"key": "value"} some extra text`,
			expected: `{"key": "value"}`,
		},
		{
			name:     "JSON array",
			content:  `[{"key": "value"}, {"key2": "value2"}]`,
			expected: `[{"key": "value"}, {"key2": "value2"}]`,
		},
		{
			name:     "No JSON",
			content:  "This is just plain text without JSON",
			expected: "",
		},
		{
			name:     "Multiline JSON in code block",
			content:  "```json\n{\n  \"key\": \"value\",\n  \"key2\": \"value2\"\n}\n```",
			expected: "{\n  \"key\": \"value\",\n  \"key2\": \"value2\"\n}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSONFromMarkdown(tt.content)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "short string",
			input:    "hello",
			maxLen:   10,
			expected: "hello",
		},
		{
			name:     "exact length",
			input:    "hello",
			maxLen:   5,
			expected: "hello",
		},
		{
			name:     "long string",
			input:    "hello world this is a test",
			maxLen:   10,
			expected: "hello worl...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateString(tt.input, tt.maxLen)
			assert.Equal(t, tt.expected, result)
		})
	}
}
