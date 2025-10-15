// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package rolebot

import (
	"io"
	"net/http"
	"testing"

	"github.com/mattermost/mattermost-plugin-ai/bots"
	"github.com/mattermost/mattermost-plugin-ai/conversations"
	"github.com/mattermost/mattermost-plugin-ai/llm"
	"github.com/mattermost/mattermost-plugin-ai/mmapi"
	"github.com/mattermost/mattermost-plugin-ai/prompts"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockMMClient implements mmapi.Client for testing
type MockMMClient struct {
	LogDebugCalls []string
	LogErrorCalls []string
}

func (m *MockMMClient) LogDebug(msg string, keyValuePairs ...interface{}) {
	m.LogDebugCalls = append(m.LogDebugCalls, msg)
}

func (m *MockMMClient) LogError(msg string, keyValuePairs ...interface{}) {
	m.LogErrorCalls = append(m.LogErrorCalls, msg)
}

func (m *MockMMClient) LogWarn(msg string, keyValuePairs ...interface{}) {}
func (m *MockMMClient) LogInfo(msg string, keyValuePairs ...interface{}) {}

// Implement remaining mmapi.Client methods as no-ops for testing
func (m *MockMMClient) GetDirectChannel(userID1, userID2 string) (*model.Channel, error) {
	return nil, nil
}
func (m *MockMMClient) GetChannel(channelID string) (*model.Channel, error) { return nil, nil }
func (m *MockMMClient) GetPost(postID string) (*model.Post, error)          { return nil, nil }
func (m *MockMMClient) GetPostThread(postID string) (*model.PostList, error) {
	return nil, nil
}
func (m *MockMMClient) GetPostsSince(channelID string, time int64) (*model.PostList, error) {
	return nil, nil
}
func (m *MockMMClient) GetPostsBefore(channelID, postID string, page, perPage int) (*model.PostList, error) {
	return nil, nil
}
func (m *MockMMClient) GetUser(userID string) (*model.User, error) { return nil, nil }
func (m *MockMMClient) GetUserByUsername(username string) (*model.User, error) {
	return nil, nil
}
func (m *MockMMClient) GetUserStatus(userID string) (*model.Status, error) { return nil, nil }
func (m *MockMMClient) UpdatePost(post *model.Post) error                  { return nil }
func (m *MockMMClient) CreatePost(post *model.Post) error                  { return nil }
func (m *MockMMClient) DM(senderID, receiverID string, post *model.Post) error {
	return nil
}
func (m *MockMMClient) AddReaction(reaction *model.Reaction) error        { return nil }
func (m *MockMMClient) SendEphemeralPost(userID string, post *model.Post) {}
func (m *MockMMClient) GetFileInfo(fileID string) (*model.FileInfo, error) {
	return nil, nil
}
func (m *MockMMClient) GetFile(fileID string) (io.ReadCloser, error) { return nil, nil }
func (m *MockMMClient) GetChannelByName(teamID, name string, includeDeleted bool) (*model.Channel, error) {
	return nil, nil
}
func (m *MockMMClient) HasPermissionTo(userID string, permission *model.Permission) bool {
	return false
}
func (m *MockMMClient) HasPermissionToChannel(userID, channelID string, permission *model.Permission) bool {
	return false
}
func (m *MockMMClient) GetPluginStatus(pluginID string) (*model.PluginStatus, error) {
	return nil, nil
}
func (m *MockMMClient) PluginHTTP(req *http.Request) *http.Response { return nil }
func (m *MockMMClient) PublishWebSocketEvent(event string, payload map[string]interface{}, broadcast *model.WebsocketBroadcast) {
}
func (m *MockMMClient) GetConfig() *model.Config                  { return nil }
func (m *MockMMClient) KVGet(key string, value interface{}) error { return nil }
func (m *MockMMClient) KVSet(key string, value interface{}) error { return nil }
func (m *MockMMClient) KVDelete(key string) error                 { return nil }
func (m *MockMMClient) GetBundlePath() (string, error)            { return "", nil }

// MockIntentHelper implements IntentHelper for testing
type MockIntentHelper struct {
	DetectedIntent   string
	DisplayNames     map[string]string
	IntentChanges    map[string]bool
	DetectCallCount  int
	DisplayCallCount int
	HasChangedCalls  int
}

func NewMockIntentHelper() *MockIntentHelper {
	return &MockIntentHelper{
		DetectedIntent: prompts.PromptDirectMessageQuestionSystem,
		DisplayNames: map[string]string{
			prompts.PromptDirectMessageQuestionSystem: "question",
			"task_create_prompt":                      "task creation",
			"status_report_prompt":                    "status report",
		},
		IntentChanges: make(map[string]bool),
	}
}

func (m *MockIntentHelper) DetectIntent(message string) string {
	m.DetectCallCount++
	return m.DetectedIntent
}

func (m *MockIntentHelper) GetDisplayName(intent string) string {
	m.DisplayCallCount++
	if name, ok := m.DisplayNames[intent]; ok {
		return name
	}
	return "unknown"
}

func (m *MockIntentHelper) HasIntentChanged(previousIntent, currentIntent string) bool {
	m.HasChangedCalls++
	key := previousIntent + "->" + currentIntent
	if changed, ok := m.IntentChanges[key]; ok {
		return changed
	}
	return previousIntent != currentIntent
}

func setupTestPrompts(t *testing.T) *llm.Prompts {
	p, err := llm.NewPrompts(prompts.PromptsFolder)
	require.NoError(t, err, "Failed to load prompts")
	return p
}

func TestNewBaseConversationHandler(t *testing.T) {
	mmClient := &MockMMClient{}
	testPrompts := setupTestPrompts(t)
	intentHelper := NewMockIntentHelper()
	roleType := "pm"

	handler := NewBaseConversationHandler(mmClient, testPrompts, intentHelper, roleType)

	require.NotNil(t, handler)
	assert.Equal(t, mmClient, handler.mmClient)
	assert.Equal(t, testPrompts, handler.prompts)
	assert.Equal(t, intentHelper, handler.intentHelper)
	assert.Equal(t, roleType, handler.roleType)
}

func TestBaseConversationHandler_buildTransitionMessage(t *testing.T) {
	mmClient := &MockMMClient{}
	testPrompts := setupTestPrompts(t)
	intentHelper := NewMockIntentHelper()

	handler := NewBaseConversationHandler(mmClient, testPrompts, intentHelper, "pm")

	tests := []struct {
		name             string
		previousIntent   string
		currentIntent    string
		expectedContains []string
		expectedEmpty    bool
	}{
		{
			name:             "Intent changes - creates transition message",
			previousIntent:   prompts.PromptDirectMessageQuestionSystem,
			currentIntent:    "task_create_prompt",
			expectedContains: []string{"Context:", "shifting from", "question", "task creation"},
			expectedEmpty:    false,
		},
		{
			name:             "Same intent - no transition message",
			previousIntent:   "task_create_prompt",
			currentIntent:    "task_create_prompt",
			expectedContains: []string{},
			expectedEmpty:    true,
		},
		{
			name:             "Different intents with same display name - no message",
			previousIntent:   "status_report_prompt",
			currentIntent:    "status_report_prompt",
			expectedContains: []string{},
			expectedEmpty:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.buildTransitionMessage(tt.previousIntent, tt.currentIntent)

			if tt.expectedEmpty {
				assert.Empty(t, result)
			} else {
				assert.NotEmpty(t, result)
				for _, expectedStr := range tt.expectedContains {
					assert.Contains(t, result, expectedStr)
				}
			}
		})
	}
}

func TestBaseConversationHandler_getRecentConversationHistory(t *testing.T) {
	mmClient := &MockMMClient{}
	testPrompts := setupTestPrompts(t)
	intentHelper := NewMockIntentHelper()
	handler := NewBaseConversationHandler(mmClient, testPrompts, intentHelper, "pm")

	postToAIPost := func(bot *bots.Bot, post *model.Post) llm.Post {
		if post.UserId == "bot_id" {
			return llm.Post{
				Role:    llm.PostRoleBot,
				Message: post.Message,
			}
		}
		return llm.Post{
			Role:    llm.PostRoleUser,
			Message: post.Message,
		}
	}

	tests := []struct {
		name          string
		conversation  *mmapi.ThreadData
		limit         int
		expectedCount int
		checkMessages func(t *testing.T, posts []llm.Post)
	}{
		{
			name: "Empty conversation - returns nil",
			conversation: &mmapi.ThreadData{
				Posts:     []*model.Post{},
				UsersByID: make(map[string]*model.User),
			},
			limit:         3,
			expectedCount: 0,
			checkMessages: func(t *testing.T, posts []llm.Post) {
				assert.Nil(t, posts)
			},
		},
		{
			name: "Single post (root) - not included in history",
			conversation: &mmapi.ThreadData{
				Posts: []*model.Post{
					{Id: "root", Message: "Root post"},
				},
				UsersByID: make(map[string]*model.User),
			},
			limit:         3,
			expectedCount: 0,
			checkMessages: func(t *testing.T, posts []llm.Post) {
				assert.Len(t, posts, 0)
			},
		},
		{
			name: "Two posts - returns one (skips root)",
			conversation: &mmapi.ThreadData{
				Posts: []*model.Post{
					{Id: "root", Message: "Root post"},
					{Id: "reply1", UserId: "user1", Message: "Reply 1"},
				},
				UsersByID: map[string]*model.User{
					"user1": {Username: "testuser"},
				},
			},
			limit:         3,
			expectedCount: 1,
			checkMessages: func(t *testing.T, posts []llm.Post) {
				assert.Len(t, posts, 1)
				assert.Contains(t, posts[0].Message, "@testuser: Reply 1")
			},
		},
		{
			name: "Five posts with limit 3 - returns last 3 (excluding root)",
			conversation: &mmapi.ThreadData{
				Posts: []*model.Post{
					{Id: "root", Message: "Root post"},
					{Id: "reply1", UserId: "user1", Message: "Reply 1"},
					{Id: "reply2", UserId: "bot_id", Message: "Bot reply 1"},
					{Id: "reply3", UserId: "user1", Message: "Reply 2"},
					{Id: "reply4", UserId: "bot_id", Message: "Bot reply 2"},
				},
				UsersByID: map[string]*model.User{
					"user1": {Username: "testuser"},
				},
			},
			limit:         3,
			expectedCount: 3,
			checkMessages: func(t *testing.T, posts []llm.Post) {
				assert.Len(t, posts, 3)
				assert.Contains(t, posts[0].Message, "Bot reply 1")
				assert.Contains(t, posts[1].Message, "@testuser: Reply 2")
				assert.Contains(t, posts[2].Message, "Bot reply 2")
			},
		},
		{
			name: "Bot messages don't get username prefix",
			conversation: &mmapi.ThreadData{
				Posts: []*model.Post{
					{Id: "root", Message: "Root post"},
					{Id: "reply1", UserId: "bot_id", Message: "Bot message"},
				},
				UsersByID: make(map[string]*model.User),
			},
			limit:         3,
			expectedCount: 1,
			checkMessages: func(t *testing.T, posts []llm.Post) {
				assert.Len(t, posts, 1)
				assert.Equal(t, "Bot message", posts[0].Message)
				assert.NotContains(t, posts[0].Message, "@")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.getRecentConversationHistory(nil, tt.conversation, tt.limit, postToAIPost)

			if tt.checkMessages != nil {
				tt.checkMessages(t, result)
			}
		})
	}
}

func TestBaseConversationHandler_buildIntentTransitionContext_FormatError(t *testing.T) {
	mmClient := &MockMMClient{}
	testPrompts := setupTestPrompts(t)
	intentHelper := NewMockIntentHelper()
	handler := NewBaseConversationHandler(mmClient, testPrompts, intentHelper, "pm")

	conversation := &mmapi.ThreadData{
		Posts:     []*model.Post{{Id: "root", Message: "Root"}},
		UsersByID: make(map[string]*model.User),
	}

	postToAIPost := func(bot *bots.Bot, post *model.Post) llm.Post {
		return llm.Post{Role: llm.PostRoleUser, Message: post.Message}
	}

	_, err := handler.buildIntentTransitionContext(
		nil,
		conversation,
		"old_intent",
		"new_intent",
		llm.NewContext(),
		postToAIPost,
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to format new pm intent prompt")
}

func TestBaseConversationHandler_buildIntentTransitionContext_Success(t *testing.T) {
	mmClient := &MockMMClient{}
	testPrompts := setupTestPrompts(t)
	intentHelper := NewMockIntentHelper()
	intentHelper.DisplayNames[prompts.PromptDirectMessageQuestionSystem] = "question"
	intentHelper.DisplayNames[prompts.PromptDirectMessageQuestionSystem] = "question"

	handler := NewBaseConversationHandler(mmClient, testPrompts, intentHelper, "pm")

	conversation := &mmapi.ThreadData{
		Posts: []*model.Post{
			{Id: "root", Message: "Root"},
			{Id: "p1", UserId: "user1", Message: "Message 1"},
			{Id: "p2", UserId: "user1", Message: "Message 2"},
		},
		UsersByID: map[string]*model.User{
			"user1": {Username: "testuser"},
		},
	}

	postToAIPost := func(bot *bots.Bot, post *model.Post) llm.Post {
		return llm.Post{Role: llm.PostRoleUser, Message: post.Message}
	}

	context := llm.NewContext()
	context.RequestingUser = &model.User{
		Id:       "user1",
		Username: "testuser",
	}

	result, err := handler.buildIntentTransitionContext(
		nil,
		conversation,
		prompts.PromptDirectMessageQuestionSystem,
		prompts.PromptDirectMessageQuestionSystem,
		context,
		postToAIPost,
	)

	require.NoError(t, err)
	require.NotNil(t, result)

	assert.GreaterOrEqual(t, len(result), 2, "Should have at least system prompt and transition message")

	assert.Equal(t, llm.PostRoleSystem, result[0].Role)
	assert.NotEmpty(t, result[0].Message, "Should have a formatted system prompt")
}

func TestPromptTypeProp_Constant(t *testing.T) {
	assert.Equal(t, "prompt_type", conversations.PromptTypeProp)
}

func TestBaseConversationHandler_getRecentConversationHistory_LimitBoundaries(t *testing.T) {
	mmClient := &MockMMClient{}
	testPrompts := setupTestPrompts(t)
	intentHelper := NewMockIntentHelper()
	handler := NewBaseConversationHandler(mmClient, testPrompts, intentHelper, "pm")

	postToAIPost := func(bot *bots.Bot, post *model.Post) llm.Post {
		return llm.Post{Role: llm.PostRoleUser, Message: post.Message}
	}

	posts := []*model.Post{
		{Id: "root", Message: "Root"},
	}
	for i := 1; i <= 10; i++ {
		posts = append(posts, &model.Post{
			Id:      model.NewId(),
			UserId:  "user1",
			Message: "Message " + string(rune('0'+i)),
		})
	}

	conversation := &mmapi.ThreadData{
		Posts:     posts,
		UsersByID: map[string]*model.User{"user1": {Username: "test"}},
	}

	tests := []struct {
		name          string
		limit         int
		expectedCount int
	}{
		{"Limit 1", 1, 1},
		{"Limit 3", 3, 3},
		{"Limit 5", 5, 5},
		{"Limit larger than available", 20, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.getRecentConversationHistory(nil, conversation, tt.limit, postToAIPost)
			assert.Len(t, result, tt.expectedCount)
		})
	}
}
