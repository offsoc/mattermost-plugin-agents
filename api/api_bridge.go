// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/mattermost/mattermost-plugin-ai/llm"
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

// handleCompletion handles POST /inter-plugin/v1/completion
// This is the HTTP-based Plugin Bridge endpoint that replaces ExecuteBridgeCall
func (a *API) handleCompletion(c *gin.Context) {
	// Extract bridge call metadata from headers
	sourcePluginID := c.GetHeader("X-Mattermost-Source-Plugin-Id")
	requestID := c.GetHeader("X-Mattermost-Request-Id")
	responseSchemaEncoded := c.GetHeader("X-Mattermost-Response-Schema")

	// Decode response schema if provided (base64-encoded)
	var responseSchema []byte
	if responseSchemaEncoded != "" {
		decoded, err := base64.StdEncoding.DecodeString(responseSchemaEncoded)
		if err == nil {
			responseSchema = decoded
		} else {
			a.pluginAPI.Log.Warn("Failed to decode response schema header", "error", err)
		}
	}

	// Log incoming bridge call
	a.pluginAPI.Log.Info("Bridge call received",
		"endpoint", c.Request.URL.Path,
		"source", sourcePluginID,
		"request_id", requestID,
		"has_schema", responseSchema != nil,
	)

	// Read and parse request body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		a.pluginAPI.Log.Error("Failed to read request body", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}
	defer c.Request.Body.Close()

	var req BridgeRequest
	if err := json.Unmarshal(body, &req); err != nil {
		a.pluginAPI.Log.Error("Invalid JSON in request body", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return
	}

	// Validate required fields
	if req.Prompt == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "prompt is required"})
		return
	}

	// Authorization check: For now, we allow all callers (core and plugins)
	// Empty sourcePluginID means call from core server
	a.pluginAPI.Log.Debug("Processing bridge completion request",
		"source", sourcePluginID,
		"prompt_length", len(req.Prompt),
	)

	// Get the default bot from configuration
	defaultBotName := a.config.GetDefaultBotName()
	if defaultBotName == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no default bot configured"})
		return
	}

	bot := a.bots.GetBotByUsername(defaultBotName)
	if bot == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("default bot not found: %s", defaultBotName)})
		return
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
			a.pluginAPI.Log.Error("Invalid response schema", "error", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid response schema: %v", err)})
			return
		}

		// Add JSON output format option with the parsed schema
		opts = append(opts, func(cfg *llm.LanguageModelConfig) {
			cfg.JSONOutputFormat = schema
		})

		a.pluginAPI.Log.Debug("Using structured output mode with provided schema")
	}

	// Call the LLM (no streaming)
	a.pluginAPI.Log.Debug("Calling LLM via bridge",
		"bot", bot.GetConfig().Name,
		"model", bot.GetService().DefaultModel,
		"structured_output", responseSchema != nil,
	)

	content, err := bot.LLM().ChatCompletionNoStream(completionRequest, opts...)
	if err != nil {
		a.pluginAPI.Log.Error("LLM completion failed",
			"error", err,
			"bot", bot.GetConfig().Name,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("LLM completion failed: %v", err)})
		return
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
					a.pluginAPI.Log.Debug("Extracted JSON from markdown code block")
				} else {
					jsonContent = "" // Reset if extraction didn't help
				}
			}

			// If we still can't parse it, return a helpful error
			if jsonContent == "" {
				a.pluginAPI.Log.Warn("LLM did not return valid JSON despite schema constraint",
					"bot", bot.GetConfig().Name,
					"provider", bot.GetService().Type,
					"content_preview", truncateString(content, 200),
				)
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "LLM returned invalid JSON despite schema constraint. This may occur if your default bot uses a provider that doesn't support structured output (e.g., Anthropic). Consider using an OpenAI-based bot for structured output requests.",
				})
				return
			}
		}

		a.pluginAPI.Log.Debug("Bridge call completed successfully with structured output")
		// Return the JSON content directly
		c.Data(http.StatusOK, "application/json", []byte(jsonContent))
		return
	}

	// For unstructured responses, wrap in our standard response format
	response := BridgeResponse{
		Content: content,
		// Note: Token counting would require accessing the stream events,
		// which we don't have in ChatCompletionNoStream. This could be enhanced
		// in the future by using ChatCompletion and consuming the stream.
	}

	a.pluginAPI.Log.Debug("Bridge call completed successfully")
	c.JSON(http.StatusOK, response)
}

// parseJSONSchema parses a JSON schema from bytes into a jsonschema.Schema
func parseJSONSchema(schemaBytes []byte) (*jsonschema.Schema, error) {
	var schemaMap map[string]interface{}
	if err := json.Unmarshal(schemaBytes, &schemaMap); err != nil {
		return nil, fmt.Errorf("failed to parse schema JSON: %w", err)
	}

	// Unmarshal into the Schema struct directly
	schema := &jsonschema.Schema{}
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
