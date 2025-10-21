// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// Package client provides a client library for other Mattermost plugins to interact
// with the AI plugin's LLM Bridge API.
//
// Security Notice: The AI plugin's inter-plugin API does not perform permission checks.
// The calling plugin is responsible for verifying that the user has appropriate permissions
// before making requests on their behalf.
package client

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/pkg/errors"
)

const (
	aiPluginID = "mattermost-ai"
)

// PluginAPI is the minimal interface needed from the Mattermost plugin API
type PluginAPI interface {
	PluginHTTP(*http.Request) *http.Response
}

// Client is a client for the Mattermost AI Plugin LLM Bridge API
type Client struct {
	httpClient http.Client
}

// pluginAPIRoundTripper wraps the Mattermost plugin API for HTTP requests
type pluginAPIRoundTripper struct {
	api PluginAPI
}

func (p *pluginAPIRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	resp := p.api.PluginHTTP(req)
	if resp == nil {
		return nil, errors.Errorf("failed to make interplugin request")
	}
	return resp, nil
}

// Post represents a single message in the conversation
type Post struct {
	Role    string     `json:"role"` // user|assistant|system|tool
	Message string     `json:"message"`
	Files   []File     `json:"files,omitempty"`
	ToolUse []ToolCall `json:"toolUse,omitempty"`
}

// File represents a file attachment
type File struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	MimeType string `json:"mime_type"`
	Data     string `json:"data"` // base64 encoded
}

// ToolCall represents a tool call or response
type ToolCall struct {
	ID    string                 `json:"id"`
	Name  string                 `json:"name"`
	Input map[string]interface{} `json:"input"`
}

// CompletionRequest represents a completion request
type CompletionRequest struct {
	Posts []Post `json:"posts"`
}

// CompletionResponse represents a non-streaming completion response
type CompletionResponse struct {
	Completion string `json:"completion"`
}

// ErrorResponse represents an error response from the API
type ErrorResponse struct {
	Error string `json:"error"`
}

// StreamCallback is called for each chunk of text received in a streaming response
type StreamCallback func(chunk string) error

// NewClient creates a new LLM Bridge API client using the Mattermost plugin API
//
// Parameters:
//   - p: Your plugin instance (must embed plugin.MattermostPlugin)
//
// Example:
//
//	type MyPlugin struct {
//	    plugin.MattermostPlugin
//	    llmClient *client.Client
//	}
//
//	func (p *MyPlugin) OnActivate() error {
//	    p.llmClient = client.NewClient(&p.MattermostPlugin)
//	    return nil
//	}
func NewClient(p *plugin.MattermostPlugin) *Client {
	return NewClientFromAPI(p.API)
}

// NewClientFromAPI creates a new LLM Bridge API client using a PluginAPI interface
//
// This constructor is useful when you have a pluginapi.Client or want more control
// over the API implementation.
//
// Parameters:
//   - api: Any type that implements PluginAPI (has a PluginHTTP method)
//
// Example:
//
//	type MyPlugin struct {
//	    plugin.MattermostPlugin
//	    pluginAPI *pluginapi.Client
//	    llmClient *client.Client
//	}
//
//	func (p *MyPlugin) OnActivate() error {
//	    p.pluginAPI = pluginapi.NewClient(p.API, p.Driver)
//	    p.llmClient = client.NewClientFromAPI(p.API)
//	    return nil
//	}
func NewClientFromAPI(api PluginAPI) *Client {
	client := &Client{}
	client.httpClient.Transport = &pluginAPIRoundTripper{api}
	return client
}

// AgentCompletion makes a non-streaming completion request to a specific agent (bot)
//
// Parameters:
//   - agent: The username of the agent/bot to use
//   - request: The completion request containing the conversation
//
// Returns the complete response text or an error.
//
// Example:
//
//	response, err := client.AgentCompletion("gpt4", client.CompletionRequest{
//	    Posts: []client.Post{
//	        {Role: "user", Message: "What is the capital of France?"},
//	    },
//	})
func (c *Client) AgentCompletion(agent string, request CompletionRequest) (string, error) {
	url := fmt.Sprintf("/%s/api/v1/agent/%s/completion/nostream", aiPluginID, agent)
	return c.doCompletionRequest(url, request)
}

// ServiceCompletion makes a non-streaming completion request to a specific service
//
// Parameters:
//   - service: The ID or name of the LLM service to use
//   - request: The completion request containing the conversation
//
// Returns the complete response text or an error.
//
// Example:
//
//	response, err := client.ServiceCompletion("openai", client.CompletionRequest{
//	    Posts: []client.Post{
//	        {Role: "user", Message: "Write a haiku about coding"},
//	    },
//	})
func (c *Client) ServiceCompletion(service string, request CompletionRequest) (string, error) {
	url := fmt.Sprintf("/%s/api/v1/service/%s/completion/nostream", aiPluginID, service)
	return c.doCompletionRequest(url, request)
}

// AgentCompletionStream makes a streaming completion request to a specific agent (bot)
//
// Parameters:
//   - agent: The username of the agent/bot to use
//   - request: The completion request containing the conversation
//   - callback: A function that will be called for each chunk of text received
//
// The callback function will be called multiple times as chunks arrive. If the callback
// returns an error, streaming will be stopped and that error will be returned.
//
// Example:
//
//	err := client.AgentCompletionStream("gpt4", client.CompletionRequest{
//	    Posts: []client.Post{
//	        {Role: "user", Message: "Tell me a story"},
//	    },
//	}, func(chunk string) error {
//	    fmt.Print(chunk)
//	    return nil
//	})
func (c *Client) AgentCompletionStream(agent string, request CompletionRequest, callback StreamCallback) error {
	url := fmt.Sprintf("/%s/api/v1/agent/%s/completion", aiPluginID, agent)
	return c.doStreamingRequest(url, request, callback)
}

// ServiceCompletionStream makes a streaming completion request to a specific service
//
// Parameters:
//   - service: The ID or name of the LLM service to use
//   - request: The completion request containing the conversation
//   - callback: A function that will be called for each chunk of text received
//
// The callback function will be called multiple times as chunks arrive. If the callback
// returns an error, streaming will be stopped and that error will be returned.
//
// Example:
//
//	err := client.ServiceCompletionStream("anthropic", client.CompletionRequest{
//	    Posts: []client.Post{
//	        {Role: "system", Message: "You are a helpful assistant"},
//	        {Role: "user", Message: "Explain quantum computing"},
//	    },
//	}, func(chunk string) error {
//	    fmt.Print(chunk)
//	    return nil
//	})
func (c *Client) ServiceCompletionStream(service string, request CompletionRequest, callback StreamCallback) error {
	url := fmt.Sprintf("/%s/api/v1/service/%s/completion", aiPluginID, service)
	return c.doStreamingRequest(url, request, callback)
}

// doCompletionRequest performs a non-streaming completion request
func (c *Client) doCompletionRequest(url string, request CompletionRequest) (string, error) {
	// Marshal the request body
	body, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create the HTTP request
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")

	// Make the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for error status codes
	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if err := json.Unmarshal(respBody, &errResp); err != nil {
			return "", fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
		}
		return "", fmt.Errorf("request failed with status %d: %s", resp.StatusCode, errResp.Error)
	}

	// Parse the success response
	var completionResp CompletionResponse
	if err := json.Unmarshal(respBody, &completionResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return completionResp.Completion, nil
}

// doStreamingRequest performs a streaming completion request
func (c *Client) doStreamingRequest(url string, request CompletionRequest, callback StreamCallback) error {
	// Marshal the request body
	body, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create the HTTP request
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	// Make the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check for error status codes
	if resp.StatusCode != http.StatusOK {
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("request failed with status %d", resp.StatusCode)
		}
		var errResp ErrorResponse
		if err := json.Unmarshal(respBody, &errResp); err != nil {
			return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
		}
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, errResp.Error)
	}

	// Read the SSE stream
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()

		// SSE lines start with "data: "
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		// Extract the data portion
		data := strings.TrimPrefix(line, "data: ")

		// Check for end of stream
		if data == "[DONE]" {
			break
		}

		// Check for empty data lines
		if data == "" {
			continue
		}

		// Check for error messages
		if strings.HasPrefix(data, "Error: ") {
			return fmt.Errorf("streaming error: %s", strings.TrimPrefix(data, "Error: "))
		}

		// Call the callback with the chunk
		if err := callback(data); err != nil {
			return fmt.Errorf("callback error: %w", err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading stream: %w", err)
	}

	return nil
}
