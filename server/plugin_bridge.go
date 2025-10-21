// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/mattermost/mattermost-plugin-ai/llm"
	"github.com/mattermost/mattermost/server/public/plugin"
)

// BridgeRequest represents the request format for bridge calls
type BridgeRequest struct {
	Prompt    string                 `json:"prompt"`
	Context   map[string]interface{} `json:"context,omitempty"`
	MaxTokens int                    `json:"max_tokens,omitempty"`
	Model     string                 `json:"model,omitempty"`
}

// BridgeResponse represents the response format for bridge calls
type BridgeResponse struct {
	Content    string `json:"content"`
	TokensUsed int    `json:"tokens_used,omitempty"`
}

// ExecuteBridgeCall implements the Plugin Bridge hook to handle incoming calls from
// core Mattermost or other plugins. This enables other features to leverage this
// plugin's LLM capabilities without direct coupling.
//
// The method accepts JSON requests with prompts and context, and can optionally
// enforce structured outputs via JSON schemas passed in responseSchema.
//
// Requirements:
// - Uses the default bot/agent from configuration
// - No tool use is allowed (tools are explicitly disabled)
// - No streaming support (fire and wait for response)
// - Supports structured LLM outputs via responseSchema parameter
func (p *Plugin) ExecuteBridgeCall(c *plugin.Context, method string, request []byte, responseSchema []byte) ([]byte, error) {
	// Log the incoming bridge call
	p.pluginAPI.Log.Info("Plugin bridge call received",
		"method", method,
		"source", c.SourcePluginId,
		"has_schema", responseSchema != nil,
	)

	// Route to appropriate handler
	switch method {
	case "GenerateCompletion":
		return p.handleGenerateCompletion(c, request, responseSchema)
	default:
		return nil, fmt.Errorf("unknown method: %s", method)
	}
}

// handleGenerateCompletion processes a completion request from the bridge
func (p *Plugin) handleGenerateCompletion(c *plugin.Context, request []byte, responseSchema []byte) ([]byte, error) {
	// Parse the incoming request
	var req BridgeRequest
	if err := json.Unmarshal(request, &req); err != nil {
		return nil, fmt.Errorf("invalid request format: %w", err)
	}

	// Validate required fields
	if req.Prompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}

	// Authorization check: For now, we allow all callers
	// In a production system, you might want to implement authorization logic here
	p.pluginAPI.Log.Debug("Processing bridge completion request",
		"source", c.SourcePluginId,
		"prompt_length", len(req.Prompt),
	)

	// Get the default bot from configuration
	defaultBotName := p.configuration.GetDefaultBotName()
	if defaultBotName == "" {
		return nil, fmt.Errorf("no default bot configured")
	}

	bot := p.bots.GetBotByUsername(defaultBotName)
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

		// Add JSON output format option with the parsed schema
		opts = append(opts, func(cfg *llm.LanguageModelConfig) {
			cfg.JSONOutputFormat = schema
		})

		p.pluginAPI.Log.Debug("Using structured output mode with provided schema")
	}

	// Call the LLM (no streaming)
	p.pluginAPI.Log.Debug("Calling LLM via bridge",
		"bot", bot.GetConfig().Name,
		"model", bot.GetService().DefaultModel,
		"structured_output", responseSchema != nil,
	)

	content, err := bot.LLM().ChatCompletionNoStream(completionRequest, opts...)
	if err != nil {
		p.pluginAPI.Log.Error("LLM completion failed",
			"error", err,
			"bot", bot.GetConfig().Name,
		)
		return nil, fmt.Errorf("LLM completion failed: %w", err)
	}

	// If a response schema was provided, try to extract/validate JSON from the response
	if responseSchema != nil {
		// Try to parse the response as JSON
		var jsonTest interface{}
		jsonContent := content

		// Some LLM providers don't support strict JSON schema enforcement (e.g., Anthropic)
		// They might return JSON wrapped in markdown or with additional text
		// Try to extract JSON from common patterns
		if err := json.Unmarshal([]byte(content), &jsonTest); err != nil {
			// Try to extract JSON from markdown code blocks
			jsonContent = extractJSONFromMarkdown(content)
			if jsonContent != "" {
				if err := json.Unmarshal([]byte(jsonContent), &jsonTest); err == nil {
					p.pluginAPI.Log.Debug("Extracted JSON from markdown code block")
				} else {
					jsonContent = "" // Reset if extraction didn't help
				}
			}

			// If we still can't parse it, return a helpful error
			if jsonContent == "" {
				p.pluginAPI.Log.Warn("LLM did not return valid JSON despite schema constraint",
					"bot", bot.GetConfig().Name,
					"provider", bot.GetService().Type,
					"content_preview", truncateString(content, 200),
				)
				return nil, fmt.Errorf("LLM returned invalid JSON despite schema constraint. This may occur if your default bot uses a provider that doesn't support structured output (e.g., Anthropic). Consider using an OpenAI-based bot for structured output requests. Error: %w", err)
			}
		}

		p.pluginAPI.Log.Debug("Bridge call completed successfully with structured output")
		// Return the JSON content directly
		return []byte(jsonContent), nil
	}

	// For unstructured responses, wrap in our standard response format
	response := BridgeResponse{
		Content: content,
		// Note: Token counting would require accessing the stream events,
		// which we don't have in ChatCompletionNoStream. This could be enhanced
		// in the future by using ChatCompletion and consuming the stream.
	}

	responseJSON, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	p.pluginAPI.Log.Debug("Bridge call completed successfully")
	return responseJSON, nil
}

// parseJSONSchema parses a JSON schema from bytes into a jsonschema.Schema
func parseJSONSchema(schemaBytes []byte) (*jsonschema.Schema, error) {
	var schemaMap map[string]interface{}
	if err := json.Unmarshal(schemaBytes, &schemaMap); err != nil {
		return nil, fmt.Errorf("failed to parse schema JSON: %w", err)
	}

	// Convert the map back to JSON for the jsonschema library
	// The jsonschema-go library expects Schema objects, but we can pass
	// the raw JSON through as a map which it will handle
	schema := &jsonschema.Schema{
		Properties: make(map[string]*jsonschema.Schema),
	}

	// Basic parsing - extract type and properties
	if schemaType, ok := schemaMap["type"].(string); ok {
		// The jsonschema library will handle this internally
		_ = schemaType
	}

	// For more complex schemas, we need to use the library's parsing capabilities
	// But for now, we'll pass the raw schema bytes through by reconstructing it
	// This is a simplified approach that works with the jsonschema-go library

	// Actually, let's use a simpler approach: unmarshal into the Schema struct directly
	schema = &jsonschema.Schema{}
	if err := json.Unmarshal(schemaBytes, schema); err != nil {
		return nil, fmt.Errorf("failed to unmarshal into Schema struct: %w", err)
	}

	return schema, nil
}

// extractJSONFromMarkdown attempts to extract JSON from markdown code blocks
// Some LLM providers wrap JSON in ```json...``` or ```...``` blocks
func extractJSONFromMarkdown(content string) string {
	// Try to find JSON in markdown code blocks
	// Pattern: ```json\n{...}\n``` or ```\n{...}\n```
	lines := strings.Split(content, "\n")
	var jsonLines []string
	inCodeBlock := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check for code block start
		if strings.HasPrefix(trimmed, "```") {
			if !inCodeBlock {
				inCodeBlock = true
				continue // Skip the opening ```
			} else {
				inCodeBlock = false
				break // Found closing ```, stop collecting
			}
		}

		// Collect lines inside code block
		if inCodeBlock {
			jsonLines = append(jsonLines, line)
		}
	}

	// If we found content in a code block, try to use it
	if len(jsonLines) > 0 {
		return strings.Join(jsonLines, "\n")
	}

	// If no code block found, try to find JSON-like content (starts with { or [)
	trimmed := strings.TrimSpace(content)
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		// Find the matching closing bracket
		bracketCount := 0
		startChar := trimmed[0]
		var endChar byte
		if startChar == '{' {
			endChar = '}'
		} else {
			endChar = ']'
		}

		for i, char := range trimmed {
			if byte(char) == startChar {
				bracketCount++
			} else if byte(char) == endChar {
				bracketCount--
				if bracketCount == 0 {
					return trimmed[:i+1]
				}
			}
		}
	}

	return ""
}

// truncateString truncates a string to maxLen characters, adding "..." if truncated
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
