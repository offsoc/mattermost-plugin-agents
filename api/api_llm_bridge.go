// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mattermost/mattermost-plugin-ai/bots"
	"github.com/mattermost/mattermost-plugin-ai/llm"
)

// LLMBridgeFile represents a file attachment in the API request
type LLMBridgeFile struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	MimeType string `json:"mime_type"`
	Data     string `json:"data"` // base64 encoded
}

// LLMBridgeToolCall represents a tool call in the API request
type LLMBridgeToolCall struct {
	ID    string                 `json:"id"`
	Name  string                 `json:"name"`
	Input map[string]interface{} `json:"input"`
}

// LLMBridgePost represents a single message in the conversation
type LLMBridgePost struct {
	Role    string              `json:"role"` // user|assistant|system|tool
	Message string              `json:"message"`
	Files   []LLMBridgeFile     `json:"files,omitempty"`
	ToolUse []LLMBridgeToolCall `json:"toolUse,omitempty"`
}

// LLMBridgeCompletionRequest represents the request body for completion endpoints
type LLMBridgeCompletionRequest struct {
	Posts []LLMBridgePost `json:"posts"`
}

// LLMBridgeCompletionResponse represents the response for non-streaming endpoints
type LLMBridgeCompletionResponse struct {
	Completion string `json:"completion"`
}

// LLMBridgeErrorResponse represents an error response
type LLMBridgeErrorResponse struct {
	Error string `json:"error"`
}

// convertLLMBridgeRequestToInternal converts the API request format to internal llm.CompletionRequest
func (a *API) convertLLMBridgeRequestToInternal(req LLMBridgeCompletionRequest) (llm.CompletionRequest, error) {
	posts := make([]llm.Post, len(req.Posts))

	for i, apiPost := range req.Posts {
		// Convert role
		var role llm.PostRole
		switch strings.ToLower(apiPost.Role) {
		case "user":
			role = llm.PostRoleUser
		case "assistant", "bot":
			role = llm.PostRoleBot
		case "system":
			role = llm.PostRoleSystem
		case "tool":
			// Tool responses are typically treated as user messages in our internal format
			role = llm.PostRoleUser
		default:
			return llm.CompletionRequest{}, fmt.Errorf("invalid role: %s", apiPost.Role)
		}

		// Convert files
		var files []llm.File
		if len(apiPost.Files) > 0 {
			files = make([]llm.File, len(apiPost.Files))
			for j, apiFile := range apiPost.Files {
				// Decode base64 data
				data, err := base64.StdEncoding.DecodeString(apiFile.Data)
				if err != nil {
					return llm.CompletionRequest{}, fmt.Errorf("failed to decode file data for file %s: %w", apiFile.Name, err)
				}

				files[j] = llm.File{
					MimeType: apiFile.MimeType,
					Size:     int64(len(data)),
					Reader:   strings.NewReader(string(data)),
				}
			}
		}

		// Convert tool calls
		var toolCalls []llm.ToolCall
		if len(apiPost.ToolUse) > 0 {
			toolCalls = make([]llm.ToolCall, len(apiPost.ToolUse))
			for j, apiToolCall := range apiPost.ToolUse {
				// Marshal the input map to JSON for Arguments field
				arguments, err := json.Marshal(apiToolCall.Input)
				if err != nil {
					return llm.CompletionRequest{}, fmt.Errorf("failed to marshal tool call arguments: %w", err)
				}
				toolCalls[j] = llm.ToolCall{
					ID:        apiToolCall.ID,
					Name:      apiToolCall.Name,
					Arguments: json.RawMessage(arguments),
				}
			}
		}

		posts[i] = llm.Post{
			Role:    role,
			Message: apiPost.Message,
			Files:   files,
			ToolUse: toolCalls,
		}
	}

	return llm.CompletionRequest{
		Posts:   posts,
		Context: &llm.Context{},
	}, nil
}

// handleAgentCompletionStreaming handles streaming completion requests for a specific agent
func (a *API) handleAgentCompletionStreaming(c *gin.Context) {
	agent := c.Param("agent")
	if agent == "" {
		c.JSON(http.StatusBadRequest, LLMBridgeErrorResponse{
			Error: "agent parameter is required",
		})
		return
	}

	var req LLMBridgeCompletionRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, LLMBridgeErrorResponse{
			Error: fmt.Sprintf("invalid request body: %v", err),
		})
		return
	}

	if len(req.Posts) == 0 {
		c.JSON(http.StatusBadRequest, LLMBridgeErrorResponse{
			Error: "posts array cannot be empty",
		})
		return
	}

	// Find the bot by username
	bot := a.bots.GetBotByUsername(agent)
	if bot == nil {
		c.JSON(http.StatusNotFound, LLMBridgeErrorResponse{
			Error: fmt.Sprintf("agent not found: %s", agent),
		})
		return
	}

	// Convert request to internal format
	llmRequest, err := a.convertLLMBridgeRequestToInternal(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, LLMBridgeErrorResponse{
			Error: fmt.Sprintf("invalid request: %v", err),
		})
		return
	}

	// Start streaming response
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Status(http.StatusOK)

	// Make the streaming LLM call
	streamResult, err := bot.LLM().ChatCompletion(llmRequest)
	if err != nil {
		// If streaming hasn't started, we can still send a JSON error
		c.Writer.WriteString(fmt.Sprintf("data: Error: %v\n\n", err))
		c.Writer.Flush()
		return
	}

	// Stream the response
	for event := range streamResult.Stream {
		switch event.Type {
		case llm.EventTypeText:
			if text, ok := event.Value.(string); ok && text != "" {
				c.Writer.WriteString(fmt.Sprintf("data: %s\n", text))
				c.Writer.Flush()
			}
		case llm.EventTypeError:
			if err, ok := event.Value.(error); ok {
				c.Writer.WriteString(fmt.Sprintf("data: Error: %v\n\n", err))
				c.Writer.Flush()
			}
		case llm.EventTypeEnd:
			// Stream ended normally
			break
		}
	}

	c.Writer.WriteString("data: [DONE]\n\n")
	c.Writer.Flush()
}

// handleAgentCompletionNoStream handles non-streaming completion requests for a specific agent
func (a *API) handleAgentCompletionNoStream(c *gin.Context) {
	agent := c.Param("agent")
	if agent == "" {
		c.JSON(http.StatusBadRequest, LLMBridgeErrorResponse{
			Error: "agent parameter is required",
		})
		return
	}

	var req LLMBridgeCompletionRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, LLMBridgeErrorResponse{
			Error: fmt.Sprintf("invalid request body: %v", err),
		})
		return
	}

	if len(req.Posts) == 0 {
		c.JSON(http.StatusBadRequest, LLMBridgeErrorResponse{
			Error: "posts array cannot be empty",
		})
		return
	}

	// Find the bot by username
	bot := a.bots.GetBotByUsername(agent)
	if bot == nil {
		c.JSON(http.StatusNotFound, LLMBridgeErrorResponse{
			Error: fmt.Sprintf("agent not found: %s", agent),
		})
		return
	}

	// Convert request to internal format
	llmRequest, err := a.convertLLMBridgeRequestToInternal(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, LLMBridgeErrorResponse{
			Error: fmt.Sprintf("invalid request: %v", err),
		})
		return
	}

	// Make the non-streaming LLM call
	response, err := bot.LLM().ChatCompletionNoStream(llmRequest)
	if err != nil {
		c.JSON(http.StatusInternalServerError, LLMBridgeErrorResponse{
			Error: fmt.Sprintf("failed to complete LLM request: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, LLMBridgeCompletionResponse{
		Completion: response,
	})
}

// handleServiceCompletionStreaming handles streaming completion requests for a specific service
func (a *API) handleServiceCompletionStreaming(c *gin.Context) {
	service := c.Param("service")
	if service == "" {
		c.JSON(http.StatusBadRequest, LLMBridgeErrorResponse{
			Error: "service parameter is required",
		})
		return
	}

	var req LLMBridgeCompletionRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, LLMBridgeErrorResponse{
			Error: fmt.Sprintf("invalid request body: %v", err),
		})
		return
	}

	if len(req.Posts) == 0 {
		c.JSON(http.StatusBadRequest, LLMBridgeErrorResponse{
			Error: "posts array cannot be empty",
		})
		return
	}

	// Find a bot that uses the specified service (by ID or name)
	var targetBot *bots.Bot
	for _, bot := range a.bots.GetAllBots() {
		botService := bot.GetService()
		if botService.ID == service || botService.Name == service {
			targetBot = bot
			break
		}
	}

	if targetBot == nil {
		c.JSON(http.StatusNotFound, LLMBridgeErrorResponse{
			Error: fmt.Sprintf("no bot found for service: %s", service),
		})
		return
	}

	// Convert request to internal format
	llmRequest, err := a.convertLLMBridgeRequestToInternal(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, LLMBridgeErrorResponse{
			Error: fmt.Sprintf("invalid request: %v", err),
		})
		return
	}

	// Start streaming response
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Status(http.StatusOK)

	// Make the streaming LLM call
	streamResult, err := targetBot.LLM().ChatCompletion(llmRequest)
	if err != nil {
		c.Writer.WriteString(fmt.Sprintf("data: Error: %v\n\n", err))
		c.Writer.Flush()
		return
	}

	// Stream the response
	for event := range streamResult.Stream {
		switch event.Type {
		case llm.EventTypeText:
			if text, ok := event.Value.(string); ok && text != "" {
				c.Writer.WriteString(fmt.Sprintf("data: %s\n", text))
				c.Writer.Flush()
			}
		case llm.EventTypeError:
			if err, ok := event.Value.(error); ok {
				c.Writer.WriteString(fmt.Sprintf("data: Error: %v\n\n", err))
				c.Writer.Flush()
			}
		case llm.EventTypeEnd:
			// Stream ended normally
			break
		}
	}

	c.Writer.WriteString("data: [DONE]\n\n")
	c.Writer.Flush()
}

// handleServiceCompletionNoStream handles non-streaming completion requests for a specific service
func (a *API) handleServiceCompletionNoStream(c *gin.Context) {
	service := c.Param("service")
	if service == "" {
		c.JSON(http.StatusBadRequest, LLMBridgeErrorResponse{
			Error: "service parameter is required",
		})
		return
	}

	var req LLMBridgeCompletionRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, LLMBridgeErrorResponse{
			Error: fmt.Sprintf("invalid request body: %v", err),
		})
		return
	}

	if len(req.Posts) == 0 {
		c.JSON(http.StatusBadRequest, LLMBridgeErrorResponse{
			Error: "posts array cannot be empty",
		})
		return
	}

	// Find a bot that uses the specified service (by ID or name)
	var targetBot *bots.Bot
	for _, bot := range a.bots.GetAllBots() {
		botService := bot.GetService()
		if botService.ID == service || botService.Name == service {
			targetBot = bot
			break
		}
	}

	if targetBot == nil {
		c.JSON(http.StatusNotFound, LLMBridgeErrorResponse{
			Error: fmt.Sprintf("no bot found for service: %s", service),
		})
		return
	}

	// Convert request to internal format
	llmRequest, err := a.convertLLMBridgeRequestToInternal(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, LLMBridgeErrorResponse{
			Error: fmt.Sprintf("invalid request: %v", err),
		})
		return
	}

	// Make the non-streaming LLM call
	response, err := targetBot.LLM().ChatCompletionNoStream(llmRequest)
	if err != nil {
		c.JSON(http.StatusInternalServerError, LLMBridgeErrorResponse{
			Error: fmt.Sprintf("failed to complete LLM request: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, LLMBridgeCompletionResponse{
		Completion: response,
	})
}
