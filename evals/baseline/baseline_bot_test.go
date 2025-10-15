// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package baseline

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/mattermost/mattermost-plugin-ai/llm"
	"github.com/mattermost/mattermost-plugin-ai/prompts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockLLMClient implements llm.LanguageModel for testing
type MockLLMClient struct {
	Response       string
	Error          error
	CallCount      int
	LastRequest    *llm.CompletionRequest
	ShouldStream   bool
	StreamedChunks []string
}

func (m *MockLLMClient) ChatCompletion(request llm.CompletionRequest, opts ...llm.LanguageModelOption) (*llm.TextStreamResult, error) {
	m.CallCount++
	m.LastRequest = &request

	if m.Error != nil {
		return nil, m.Error
	}

	stream := make(chan llm.TextStreamEvent)
	result := &llm.TextStreamResult{
		Stream: stream,
	}

	go func() {
		defer close(stream)
		if m.ShouldStream {
			for _, chunk := range m.StreamedChunks {
				stream <- llm.TextStreamEvent{Type: llm.EventTypeText, Value: chunk}
			}
		} else {
			stream <- llm.TextStreamEvent{Type: llm.EventTypeText, Value: m.Response}
		}
		stream <- llm.TextStreamEvent{Type: llm.EventTypeEnd, Value: nil}
	}()

	return result, nil
}

func (m *MockLLMClient) ChatCompletionNoStream(request llm.CompletionRequest, opts ...llm.LanguageModelOption) (string, error) {
	m.CallCount++
	m.LastRequest = &request

	if m.Error != nil {
		return "", m.Error
	}

	return m.Response, nil
}

func (m *MockLLMClient) CountTokens(text string) int {
	return len(text) / 4 // Simple approximation
}

func (m *MockLLMClient) InputTokenLimit() int {
	return 100000 // Default limit for testing
}

// MockIntentDetector is a simple intent detector for testing
func MockIntentDetector(message string) string {
	return prompts.PromptDirectMessageQuestionSystem
}

func TestNewBaselineBot(t *testing.T) {
	mockLLM := &MockLLMClient{Response: "test response"}

	bot := NewBaselineBot(mockLLM)

	require.NotNil(t, bot)
	assert.Equal(t, mockLLM, bot.llmClient)
	assert.Equal(t, "baseline-llm", bot.name)
	assert.False(t, bot.useRolePrompts)
	assert.Nil(t, bot.prompts)
	assert.Nil(t, bot.intentDetector)
}

func TestNewBaselineBotWithName(t *testing.T) {
	mockLLM := &MockLLMClient{Response: "test response"}
	customName := "my-baseline-bot"

	bot := NewBaselineBotWithName(mockLLM, customName)

	require.NotNil(t, bot)
	assert.Equal(t, customName, bot.name)
	assert.Equal(t, mockLLM, bot.llmClient)
	assert.False(t, bot.useRolePrompts)
}

func TestNewRolePromptBaselineBot(t *testing.T) {
	mockLLM := &MockLLMClient{Response: "test response"}
	testPrompts, err := llm.NewPrompts(prompts.PromptsFolder)
	require.NoError(t, err)

	bot := NewRolePromptBaselineBot(mockLLM, testPrompts, MockIntentDetector, "pm-baseline")

	require.NotNil(t, bot)
	assert.Equal(t, "pm-baseline", bot.name)
	assert.True(t, bot.useRolePrompts)
	assert.NotNil(t, bot.prompts)
	assert.NotNil(t, bot.intentDetector)
}

func TestNewPMPromptBaselineBot(t *testing.T) {
	mockLLM := &MockLLMClient{Response: "test response"}
	testPrompts, err := llm.NewPrompts(prompts.PromptsFolder)
	require.NoError(t, err)

	bot := NewPMPromptBaselineBot(mockLLM, testPrompts, "pm-baseline")

	require.NotNil(t, bot)
	assert.Equal(t, "pm-baseline", bot.name)
	assert.True(t, bot.useRolePrompts)
	assert.NotNil(t, bot.prompts)
	assert.NotNil(t, bot.intentDetector)
}

func TestNewDevPromptBaselineBot(t *testing.T) {
	mockLLM := &MockLLMClient{Response: "test response"}
	testPrompts, err := llm.NewPrompts(prompts.PromptsFolder)
	require.NoError(t, err)

	bot := NewDevPromptBaselineBot(mockLLM, testPrompts, "dev-baseline")

	require.NotNil(t, bot)
	assert.Equal(t, "dev-baseline", bot.name)
	assert.True(t, bot.useRolePrompts)
	assert.NotNil(t, bot.prompts)
	assert.NotNil(t, bot.intentDetector)
}

func TestLLMBot_Respond_MinimalPrompt(t *testing.T) {
	mockLLM := &MockLLMClient{Response: "This is the baseline response"}
	bot := NewBaselineBot(mockLLM)

	answer, err := bot.Respond(context.Background(), "What is Mattermost?")

	require.NoError(t, err)
	assert.Equal(t, "This is the baseline response", answer.Text)
	assert.Greater(t, answer.Latency, time.Duration(0))
	assert.Equal(t, 1, mockLLM.CallCount)

	// Verify metadata
	assert.Equal(t, ModelTypeBaseline, answer.Metadata[MetadataKeyModelType])

	// Verify request structure
	require.NotNil(t, mockLLM.LastRequest)
	assert.Len(t, mockLLM.LastRequest.Posts, 2)
	assert.Equal(t, llm.PostRoleSystem, mockLLM.LastRequest.Posts[0].Role)
	assert.Equal(t, BaselineSystemPrompt, mockLLM.LastRequest.Posts[0].Message)
	assert.Equal(t, llm.PostRoleUser, mockLLM.LastRequest.Posts[1].Role)
	assert.Equal(t, "What is Mattermost?", mockLLM.LastRequest.Posts[1].Message)

	// Verify LastRequest and LastResponse are stored
	assert.NotNil(t, bot.LastRequest)
	assert.Equal(t, "This is the baseline response", bot.LastResponse)
}

func TestLLMBot_Respond_RolePrompts(t *testing.T) {
	mockLLM := &MockLLMClient{Response: "Role-based response"}
	testPrompts, err := llm.NewPrompts(prompts.PromptsFolder)
	require.NoError(t, err)

	bot := NewRolePromptBaselineBot(mockLLM, testPrompts, MockIntentDetector, "pm-baseline")

	answer, err := bot.Respond(context.Background(), "What are the latest features?")

	require.NoError(t, err)
	assert.Equal(t, "Role-based response", answer.Text)
	assert.Greater(t, answer.Latency, time.Duration(0))

	// Verify metadata
	assert.Equal(t, "role-prompt-baseline", answer.Metadata[MetadataKeyModelType])
	assert.Equal(t, prompts.PromptDirectMessageQuestionSystem, answer.Metadata["intent"])

	// Verify request structure
	require.NotNil(t, mockLLM.LastRequest)
	assert.Len(t, mockLLM.LastRequest.Posts, 2)
	assert.Equal(t, llm.PostRoleSystem, mockLLM.LastRequest.Posts[0].Role)
	// System prompt should be formatted from template, not minimal
	assert.NotEqual(t, BaselineSystemPrompt, mockLLM.LastRequest.Posts[0].Message)
	assert.Greater(t, len(mockLLM.LastRequest.Posts[0].Message), len(BaselineSystemPrompt))
}

func TestLLMBot_Respond_RolePrompts_FallbackToBaseline(t *testing.T) {
	mockLLM := &MockLLMClient{Response: "Fallback response"}

	// Create bot with nil prompts - should use baseline prompt since prompts is nil
	bot := &LLMBot{
		llmClient:      mockLLM,
		prompts:        nil,
		useRolePrompts: true,
		intentDetector: MockIntentDetector,
		name:           "test-bot",
	}

	answer, err := bot.Respond(context.Background(), "Test message")

	require.NoError(t, err)
	assert.Equal(t, "Fallback response", answer.Text)

	// Verify fallback to minimal prompt (prompts == nil so condition fails)
	assert.Equal(t, ModelTypeBaseline, answer.Metadata[MetadataKeyModelType])
	// No prompt_error since the check fails early due to nil prompts
	_, hasPromptError := answer.Metadata["prompt_error"]
	assert.False(t, hasPromptError)

	// Verify minimal prompt was used
	require.NotNil(t, mockLLM.LastRequest)
	assert.Equal(t, BaselineSystemPrompt, mockLLM.LastRequest.Posts[0].Message)
}

func TestLLMBot_Respond_LLMError(t *testing.T) {
	expectedError := errors.New("LLM service unavailable")
	mockLLM := &MockLLMClient{Error: expectedError}
	bot := NewBaselineBot(mockLLM)

	_, err := bot.Respond(context.Background(), "Test message")

	require.Error(t, err)
	assert.Equal(t, expectedError, err)
	assert.Equal(t, 1, mockLLM.CallCount)
}

func TestLLMBot_Name(t *testing.T) {
	tests := []struct {
		name         string
		botName      string
		expectedName string
	}{
		{
			name:         "Default name",
			botName:      "",
			expectedName: "baseline-llm",
		},
		{
			name:         "Custom name",
			botName:      "my-custom-baseline",
			expectedName: "my-custom-baseline",
		},
		{
			name:         "PM baseline name",
			botName:      "pm-baseline",
			expectedName: "pm-baseline",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLLM := &MockLLMClient{Response: "test"}
			var bot *LLMBot
			if tt.botName == "" {
				bot = NewBaselineBot(mockLLM)
			} else {
				bot = NewBaselineBotWithName(mockLLM, tt.botName)
			}

			assert.Equal(t, tt.expectedName, bot.Name())
		})
	}
}

type contextKey string

const testContextKey contextKey = "test_key"

func TestLLMBot_Respond_ContextPassed(t *testing.T) {
	mockLLM := &MockLLMClient{Response: "response"}
	bot := NewBaselineBot(mockLLM)

	ctx := context.WithValue(context.Background(), testContextKey, "test_value")
	_, err := bot.Respond(ctx, "Test message")

	require.NoError(t, err)
	assert.Equal(t, 1, mockLLM.CallCount)
}

func TestLLMBot_Respond_MetadataStructure(t *testing.T) {
	mockLLM := &MockLLMClient{Response: "test response"}
	bot := NewBaselineBot(mockLLM)

	answer, err := bot.Respond(context.Background(), "Test")

	require.NoError(t, err)
	require.NotNil(t, answer.Metadata)

	// Verify metadata has correct keys
	_, hasModelType := answer.Metadata[MetadataKeyModelType]
	assert.True(t, hasModelType, "Metadata should have model_type key")

	// Verify metadata values
	assert.IsType(t, "", answer.Metadata[MetadataKeyModelType])
}

func TestLLMBot_Respond_TokenUsage(t *testing.T) {
	mockLLM := &MockLLMClient{Response: "test response"}
	bot := NewBaselineBot(mockLLM)

	answer, err := bot.Respond(context.Background(), "Test")

	require.NoError(t, err)

	// Verify TokenUsage structure exists (even if empty for mock)
	assert.Equal(t, 0, answer.Tokens.Prompt)
	assert.Equal(t, 0, answer.Tokens.Completion)
	assert.Equal(t, 0, answer.Tokens.Total)
}

func TestLLMBot_Respond_LatencyMeasurement(t *testing.T) {
	mockLLM := &MockLLMClient{Response: "test response"}
	bot := NewBaselineBot(mockLLM)

	start := time.Now()
	answer, err := bot.Respond(context.Background(), "Test")
	elapsed := time.Since(start)

	require.NoError(t, err)
	assert.Greater(t, answer.Latency, time.Duration(0))
	assert.Less(t, answer.Latency, elapsed+time.Millisecond)
}

func TestLLMBot_Respond_EmptyMessage(t *testing.T) {
	mockLLM := &MockLLMClient{Response: "response to empty"}
	bot := NewBaselineBot(mockLLM)

	answer, err := bot.Respond(context.Background(), "")

	require.NoError(t, err)
	assert.Equal(t, "response to empty", answer.Text)

	// Verify empty message was passed to LLM
	require.NotNil(t, mockLLM.LastRequest)
	assert.Equal(t, "", mockLLM.LastRequest.Posts[1].Message)
}

func TestLLMBot_Respond_LongMessage(t *testing.T) {
	mockLLM := &MockLLMClient{Response: "response"}
	bot := NewBaselineBot(mockLLM)

	longMessage := string(make([]byte, 10000))
	for i := range longMessage {
		longMessage = string([]byte{byte('a' + (i % 26))})
	}

	answer, err := bot.Respond(context.Background(), longMessage)

	require.NoError(t, err)
	assert.Equal(t, "response", answer.Text)
	assert.Equal(t, 1, mockLLM.CallCount)
}

func TestLLMBot_Respond_SpecialCharacters(t *testing.T) {
	mockLLM := &MockLLMClient{Response: "response"}
	bot := NewBaselineBot(mockLLM)

	specialMsg := "Test with 特殊字符 and émojis 🎉 and newlines\n\ttabs"
	answer, err := bot.Respond(context.Background(), specialMsg)

	require.NoError(t, err)
	assert.Equal(t, "response", answer.Text)

	// Verify special characters preserved
	require.NotNil(t, mockLLM.LastRequest)
	assert.Equal(t, specialMsg, mockLLM.LastRequest.Posts[1].Message)
}

func TestBaselineSystemPrompt_Constant(t *testing.T) {
	assert.Equal(t, "You are a helpful AI assistant.", BaselineSystemPrompt)
}

func TestAnswer_Structure(t *testing.T) {
	answer := Answer{
		Text:    "test response",
		Latency: 100 * time.Millisecond,
		Tokens: TokenUsage{
			Prompt:     10,
			Completion: 20,
			Total:      30,
		},
		Metadata: map[string]interface{}{
			MetadataKeyModelType: ModelTypeBaseline,
		},
	}

	assert.Equal(t, "test response", answer.Text)
	assert.Equal(t, 100*time.Millisecond, answer.Latency)
	assert.Equal(t, 10, answer.Tokens.Prompt)
	assert.Equal(t, 20, answer.Tokens.Completion)
	assert.Equal(t, 30, answer.Tokens.Total)
	assert.Equal(t, ModelTypeBaseline, answer.Metadata[MetadataKeyModelType])
}

func TestLLMBot_MultipleResponds(t *testing.T) {
	mockLLM := &MockLLMClient{}
	bot := NewBaselineBot(mockLLM)

	messages := []string{"First message", "Second message", "Third message"}

	for i, msg := range messages {
		mockLLM.Response = "Response " + string(rune('A'+i))
		answer, err := bot.Respond(context.Background(), msg)

		require.NoError(t, err)
		assert.Equal(t, "Response "+string(rune('A'+i)), answer.Text)
		assert.Equal(t, i+1, mockLLM.CallCount)

		// Verify LastRequest/LastResponse updated
		assert.NotNil(t, bot.LastRequest)
		assert.Equal(t, msg, bot.LastRequest.Posts[1].Message)
		assert.Equal(t, "Response "+string(rune('A'+i)), bot.LastResponse)
	}
}

func TestLLMBot_RolePrompts_IntentDetection(t *testing.T) {
	mockLLM := &MockLLMClient{Response: "response"}
	testPrompts, err := llm.NewPrompts(prompts.PromptsFolder)
	require.NoError(t, err)

	// Custom intent detector that returns different intents
	intentDetector := func(message string) string {
		if len(message) > 20 {
			return "long_message_intent"
		}
		return prompts.PromptDirectMessageQuestionSystem
	}

	bot := NewRolePromptBaselineBot(mockLLM, testPrompts, intentDetector, "test-bot")

	tests := []struct {
		name           string
		message        string
		expectedIntent string
	}{
		{
			name:           "Short message",
			message:        "Short",
			expectedIntent: prompts.PromptDirectMessageQuestionSystem,
		},
		{
			name:           "Long message",
			message:        "This is a very long message that should trigger different intent",
			expectedIntent: "long_message_intent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			answer, err := bot.Respond(context.Background(), tt.message)

			require.NoError(t, err)
			assert.NotNil(t, answer)

			// For standard prompt, we get the intent in metadata
			// For unknown prompt, we fall back to baseline
			if tt.expectedIntent == prompts.PromptDirectMessageQuestionSystem {
				assert.Equal(t, "role-prompt-baseline", answer.Metadata[MetadataKeyModelType])
			}
		})
	}
}
