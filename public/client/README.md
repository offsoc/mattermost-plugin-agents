# Mattermost AI Plugin - LLM Bridge Client

This package provides a Go client library for Mattermost plugins to interact with the AI plugin's LLM Bridge API.

## Installation

```go
import "github.com/mattermost/mattermost-plugin-ai/public/client"
```

## Quick Start

### Basic Setup

```go
package main

import (
    "github.com/mattermost/mattermost-plugin-ai/public/client"
    "github.com/mattermost/mattermost/server/public/plugin"
)

type MyPlugin struct {
    plugin.MattermostPlugin
    llmClient *client.Client
}

func (p *MyPlugin) OnActivate() error {
    // Create the LLM Bridge client - it's this simple!
    p.llmClient = client.NewClient(&p.MattermostPlugin)
    return nil
}
```

That's it! The client automatically handles inter-plugin communication using the Mattermost plugin API.

### Alternative Setup with PluginAPI

If you want more flexibility or are already using `pluginapi.Client`, you can create the client directly from the API:

```go
import (
    "github.com/mattermost/mattermost-plugin-ai/public/client"
    "github.com/mattermost/mattermost/server/public/plugin"
)

type MyPlugin struct {
    plugin.MattermostPlugin
    llmClient *client.Client
}

func (p *MyPlugin) OnActivate() error {
    // Create client using the API interface directly
    p.llmClient = client.NewClientFromAPI(p.API)
    return nil
}
```

Both constructors work identically - choose whichever fits your plugin's architecture better.

## API Methods

### Agent Completion

Make a request to a specific agent (bot) by username:

```go
response, err := p.llmClient.AgentCompletion("gpt4", client.CompletionRequest{
    Posts: []client.Post{
        {Role: "user", Message: "What is the capital of France?"},
    },
})
if err != nil {
    // Handle error
}
fmt.Println(response) // "The capital of France is Paris."
```

### Service Completion

Make a request to a specific LLM service by ID or name:

```go
response, err := p.llmClient.ServiceCompletion("openai", client.CompletionRequest{
    Posts: []client.Post{
        {Role: "user", Message: "Write a haiku about coding"},
    },
})
```

## Request Structure

### Posts

A `Post` represents a single message in the conversation:

```go
type Post struct {
    Role    string     // "user", "assistant", "system", or "tool"
    Message string     // The message content
    Files   []File     // Optional file attachments
    ToolUse []ToolCall // Optional tool calls
}
```

### Multi-turn Conversations

```go
request := client.CompletionRequest{
    Posts: []client.Post{
        {Role: "system", Message: "You are a helpful assistant"},
        {Role: "user", Message: "What is AI?"},
        {Role: "assistant", Message: "AI stands for Artificial Intelligence..."},
        {Role: "user", Message: "Can you give me examples?"},
    },
}

response, err := p.llmClient.AgentCompletion("gpt4", request)
```

### Including Files

Files must be base64 encoded:

```go
import "encoding/base64"

fileData, _ := os.ReadFile("image.png")
encodedData := base64.StdEncoding.EncodeToString(fileData)

request := client.CompletionRequest{
    Posts: []client.Post{
        {
            Role:    "user",
            Message: "What's in this image?",
            Files: []client.File{
                {
                    ID:       "file123",
                    Name:     "image.png",
                    MimeType: "image/png",
                    Data:     encodedData,
                },
            },
        },
    },
}
```

### Tool Calls

Include tool calls in the conversation:

```go
request := client.CompletionRequest{
    Posts: []client.Post{
        {
            Role:    "assistant",
            Message: "I need to search for that information",
            ToolUse: []client.ToolCall{
                {
                    ID:   "call_123",
                    Name: "web_search",
                    Input: map[string]interface{}{
                        "query": "latest news",
                    },
                },
            },
        },
        {
            Role:    "tool",
            Message: "Search results: ...",
        },
    },
}
```

## Error Handling

All methods return errors that should be checked:

```go
response, err := p.llmClient.AgentCompletion("gpt4", request)
if err != nil {
    // Possible errors:
    // - Network errors
    // - Agent/service not found (404)
    // - Invalid request (400)
    // - Internal server errors (500)
    p.API.LogError("LLM request failed", "error", err.Error())
    return
}
```

## Complete Example

```go
package main

import (
    "fmt"
    "github.com/mattermost/mattermost-plugin-ai/public/client"
    "github.com/mattermost/mattermost/server/public/plugin"
    "github.com/mattermost/mattermost/server/public/model"
)

type MyPlugin struct {
    plugin.MattermostPlugin
    llmClient *client.Client
}

func (p *MyPlugin) OnActivate() error {
    p.llmClient = client.NewClient(&p.MattermostPlugin)
    return nil
}

func (p *MyPlugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
    // Example: Use AI to respond to a command
    response, err := p.llmClient.AgentCompletion("gpt4", client.CompletionRequest{
        Posts: []client.Post{
            {
                Role:    "system",
                Message: "You are a helpful assistant for my plugin",
            },
            {
                Role:    "user",
                Message: args.Command,
            },
        },
    })
    if err != nil {
        return nil, model.NewAppError("ExecuteCommand", "app.plugin.ai.error", nil, err.Error(), http.StatusInternalServerError)
    }

    return &model.CommandResponse{
        ResponseType: model.CommandResponseTypeEphemeral,
        Text:         response,
    }, nil
}
```

## Security Notice

⚠️ **Important**: The AI plugin's inter-plugin API does not perform permission checks. Your plugin is responsible for verifying that users have appropriate permissions before making requests on their behalf.

Example permission check:

```go
func (p *MyPlugin) handleUserRequest(userID, message string) error {
    // Check if user has permission to use AI features
    user, err := p.API.GetUser(userID)
    if err != nil {
        return err
    }

    // Add your permission logic here
    if !p.userCanUseAI(user) {
        return fmt.Errorf("user does not have permission to use AI features")
    }

    // Make the AI request
    response, err := p.llmClient.AgentCompletion("gpt4", client.CompletionRequest{
        Posts: []client.Post{
            {Role: "user", Message: message},
        },
    })
    // ... handle response
}
```

## API Endpoints

The client calls these endpoints on the AI plugin:

- `POST /mattermost-ai/api/v1/agent/{agent}/completion/nostream` - Non-streaming agent completion
- `POST /mattermost-ai/api/v1/service/{service}/completion/nostream` - Non-streaming service completion

All endpoints use inter-plugin communication via the Mattermost plugin API.

## Differences Between Agent and Service

- **Agent**: Refers to a specific bot by username (e.g., "gpt4", "claude")
  - Use when you want to target a specific bot configuration with custom settings, tools, and prompts

- **Service**: Refers to the underlying LLM service by ID or name (e.g., "openai", "anthropic")
  - Use when you want any bot that uses a particular LLM service
  - Useful when you don't care about bot-specific configuration

## Additional Resources

- See `example_test.go` for more usage examples
- Refer to `API.md` in the plugin root for the full API specification
- Check the AI plugin documentation for available agents and services
