// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package client_test

import (
	"fmt"

	"github.com/mattermost/mattermost-plugin-ai/public/client"
	"github.com/mattermost/mattermost/server/public/plugin"
)

// Example demonstrates basic usage of the LLM Bridge API client
func Example() {
	// In a real plugin, you would embed plugin.MattermostPlugin in your plugin struct
	type MyPlugin struct {
		plugin.MattermostPlugin
		llmClient *client.Client
	}

	var p MyPlugin

	// Create the client in OnActivate
	p.llmClient = client.NewClient(&p.MattermostPlugin)

	// Make a non-streaming request to an agent
	response, err := p.llmClient.AgentCompletion("gpt4", client.CompletionRequest{
		Posts: []client.Post{
			{
				Role:    "user",
				Message: "What is the capital of France?",
			},
		},
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Response: %s\n", response)
}

// ExampleNewClient shows how to create a client in your plugin
func ExampleNewClient() {
	type MyPlugin struct {
		plugin.MattermostPlugin
		llmClient *client.Client
	}

	var p MyPlugin

	// Create the client - typically done in OnActivate
	p.llmClient = client.NewClient(&p.MattermostPlugin)

	fmt.Println("Client created successfully")
}

// ExampleNewClientFromAPI shows how to create a client using the PluginAPI interface
func ExampleNewClientFromAPI() {
	type MyPlugin struct {
		plugin.MattermostPlugin
		llmClient *client.Client
	}

	var p MyPlugin

	// Create the client using the API directly - gives more flexibility
	p.llmClient = client.NewClientFromAPI(p.API)

	fmt.Println("Client created successfully from API")
}

// ExampleClient_AgentCompletion shows how to make a simple non-streaming request
func ExampleClient_AgentCompletion() {
	type MyPlugin struct {
		plugin.MattermostPlugin
	}

	var p MyPlugin
	llmClient := client.NewClient(&p.MattermostPlugin)

	response, err := llmClient.AgentCompletion("gpt4", client.CompletionRequest{
		Posts: []client.Post{
			{Role: "user", Message: "Hello!"},
		},
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Response: %s\n", response)
}

// ExampleClient_AgentCompletionStream shows how to make a streaming request
func ExampleClient_AgentCompletionStream() {
	type MyPlugin struct {
		plugin.MattermostPlugin
	}

	var p MyPlugin
	llmClient := client.NewClient(&p.MattermostPlugin)

	err := llmClient.AgentCompletionStream("gpt4", client.CompletionRequest{
		Posts: []client.Post{
			{Role: "user", Message: "Tell me a story"},
		},
	}, func(chunk string) error {
		// This callback is called for each chunk of text
		fmt.Print(chunk)
		return nil
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
}

// ExampleClient_ServiceCompletion shows how to use a specific LLM service
func ExampleClient_ServiceCompletion() {
	type MyPlugin struct {
		plugin.MattermostPlugin
	}

	var p MyPlugin
	llmClient := client.NewClient(&p.MattermostPlugin)

	response, err := llmClient.ServiceCompletion("openai", client.CompletionRequest{
		Posts: []client.Post{
			{Role: "system", Message: "You are a helpful assistant"},
			{Role: "user", Message: "What is 2+2?"},
		},
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Response: %s\n", response)
}

// ExampleCompletionRequest_withFiles shows how to include files in a request
func ExampleCompletionRequest_withFiles() {
	type MyPlugin struct {
		plugin.MattermostPlugin
	}

	var p MyPlugin
	llmClient := client.NewClient(&p.MattermostPlugin)

	// Example with a file attachment (base64 encoded)
	response, err := llmClient.AgentCompletion("gpt4", client.CompletionRequest{
		Posts: []client.Post{
			{
				Role:    "user",
				Message: "What's in this image?",
				Files: []client.File{
					{
						ID:       "file123",
						Name:     "image.png",
						MimeType: "image/png",
						Data:     "base64encodeddata...",
					},
				},
			},
		},
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Response: %s\n", response)
}

// ExampleCompletionRequest_multiTurn shows a multi-turn conversation
func ExampleCompletionRequest_multiTurn() {
	type MyPlugin struct {
		plugin.MattermostPlugin
	}

	var p MyPlugin
	llmClient := client.NewClient(&p.MattermostPlugin)

	// Multi-turn conversation
	response, err := llmClient.AgentCompletion("gpt4", client.CompletionRequest{
		Posts: []client.Post{
			{Role: "system", Message: "You are a helpful math tutor"},
			{Role: "user", Message: "What is calculus?"},
			{Role: "assistant", Message: "Calculus is a branch of mathematics..."},
			{Role: "user", Message: "Can you give me an example?"},
		},
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Response: %s\n", response)
}

// Example_fullPlugin shows a complete plugin implementation
func Example_fullPlugin() {
	type MyPlugin struct {
		plugin.MattermostPlugin
		llmClient *client.Client
	}

	var p MyPlugin

	// OnActivate - create the client
	p.llmClient = client.NewClient(&p.MattermostPlugin)

	// Use the client in a command or other handler
	response, err := p.llmClient.AgentCompletion("gpt4", client.CompletionRequest{
		Posts: []client.Post{
			{
				Role:    "system",
				Message: "You are a helpful assistant for my plugin",
			},
			{
				Role:    "user",
				Message: "Help me with this task",
			},
		},
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("AI Response: %s\n", response)
}
