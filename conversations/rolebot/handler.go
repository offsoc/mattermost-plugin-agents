// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package rolebot

import (
	"fmt"

	"github.com/mattermost/mattermost-plugin-ai/bots"
	"github.com/mattermost/mattermost-plugin-ai/conversations"
	"github.com/mattermost/mattermost-plugin-ai/llm"
	"github.com/mattermost/mattermost-plugin-ai/mmapi"
	"github.com/mattermost/mattermost-plugin-ai/prompts"
	"github.com/mattermost/mattermost/server/public/model"
)

// BaseConversationHandler handles generic conversation logic for role-based bots
type BaseConversationHandler struct {
	mmClient     mmapi.Client
	prompts      *llm.Prompts
	intentHelper IntentHelper
	roleType     string
}

// NewBaseConversationHandler creates a new base conversation handler
func NewBaseConversationHandler(mmClient mmapi.Client, prompts *llm.Prompts, intentHelper IntentHelper, roleType string) *BaseConversationHandler {
	return &BaseConversationHandler{
		mmClient:     mmClient,
		prompts:      prompts,
		intentHelper: intentHelper,
		roleType:     roleType,
	}
}

// ProcessBotRequest handles requests to role-based bots with per-message intent detection
func (h *BaseConversationHandler) ProcessBotRequest(
	bot *bots.Bot,
	postingUser *model.User,
	channel *model.Channel,
	post *model.Post,
	context *llm.Context,
	postToAIPost func(*bots.Bot, *model.Post) llm.Post,
	existingConvToLLMPosts func(*bots.Bot, *mmapi.ThreadData, *llm.Context) ([]llm.Post, error),
	generateTitle func(*bots.Bot, string, string, *llm.Context) error,
	titlePromptPrefix string,
) (*llm.TextStreamResult, error) {
	var posts []llm.Post

	currentIntent := h.intentHelper.DetectIntent(post.Message)

	h.mmClient.LogDebug(h.roleType+" intent detected",
		"post_id", post.Id,
		"detected_intent", currentIntent)

	if post.RootId == "" {
		h.mmClient.LogDebug("New "+h.roleType+" bot conversation", "post_id", post.Id, "intent", currentIntent)
		post.AddProp(conversations.PromptTypeProp, currentIntent)

		prompt, err := h.prompts.Format(currentIntent, context)
		if err != nil {
			h.mmClient.LogError("Failed to format "+h.roleType+" bot prompt", "error", err, "post_id", post.Id, "intent", currentIntent)
			return nil, fmt.Errorf("failed to format %s bot prompt: %w", h.roleType, err)
		}
		h.mmClient.LogDebug(h.roleType+" bot prompt formatted", "post_id", post.Id, "intent", currentIntent, "prompt_length", len(prompt))
		posts = []llm.Post{
			{
				Role:    llm.PostRoleSystem,
				Message: prompt,
			},
		}
	} else {
		previousConversation, errThread := mmapi.GetThreadData(h.mmClient, post.Id)
		if errThread != nil {
			return nil, fmt.Errorf("failed to get previous %s bot conversation: %w", h.roleType, errThread)
		}
		previousConversation.CutoffBeforePostID(post.Id)

		var previousIntent string
		if len(previousConversation.Posts) > 0 {
			if rootPost := previousConversation.Posts[0]; rootPost != nil {
				if savedPrompt, ok := rootPost.GetProp(conversations.PromptTypeProp).(string); ok && savedPrompt != "" {
					previousIntent = savedPrompt
				} else {
					previousIntent = prompts.PromptDirectMessageQuestionSystem
				}
			}
		} else {
			previousIntent = prompts.PromptDirectMessageQuestionSystem
		}

		if h.intentHelper.HasIntentChanged(previousIntent, currentIntent) {
			post.AddProp(conversations.PromptTypeProp, currentIntent)

			var err error
			posts, err = h.buildIntentTransitionContext(bot, previousConversation, previousIntent, currentIntent, context, postToAIPost)
			if err != nil {
				return nil, fmt.Errorf("failed to build %s intent transition context: %w", h.roleType, err)
			}
		} else {
			var err error
			posts, err = existingConvToLLMPosts(bot, previousConversation, context)
			if err != nil {
				return nil, fmt.Errorf("failed to convert existing %s bot conversation to LLM posts: %w", h.roleType, err)
			}
		}
	}

	posts = append(posts, postToAIPost(bot, post))

	h.mmClient.LogDebug("Calling LLM ChatCompletion",
		"post_id", post.Id,
		"num_posts", len(posts),
		"bot_name", bot.GetConfig().Name)

	completionRequest := llm.CompletionRequest{
		Posts:   posts,
		Context: context,
	}

	result, err := bot.LLM().ChatCompletion(completionRequest)
	if err != nil {
		h.mmClient.LogError("LLM ChatCompletion failed",
			"error", err,
			"post_id", post.Id,
			"bot_name", bot.GetConfig().Name)
		return nil, err
	}

	h.mmClient.LogDebug("LLM ChatCompletion successful",
		"post_id", post.Id,
		"bot_name", bot.GetConfig().Name)

	if post.RootId == "" && generateTitle != nil {
		go func() {
			request := titlePromptPrefix + post.Message
			if err := generateTitle(bot, request, post.Id, context); err != nil {
				h.mmClient.LogError("Failed to generate "+h.roleType+" bot title", "error", err.Error())
				return
			}
		}()
	}

	return result, nil
}

func (h *BaseConversationHandler) buildIntentTransitionContext(
	bot *bots.Bot,
	previousConversation *mmapi.ThreadData,
	previousIntent, currentIntent string,
	context *llm.Context,
	postToAIPost func(*bots.Bot, *model.Post) llm.Post,
) ([]llm.Post, error) {
	newPrompt, err := h.prompts.Format(currentIntent, context)
	if err != nil {
		return nil, fmt.Errorf("failed to format new %s intent prompt: %w", h.roleType, err)
	}

	posts := []llm.Post{
		{
			Role:    llm.PostRoleSystem,
			Message: newPrompt,
		},
	}

	transitionMessage := h.buildTransitionMessage(previousIntent, currentIntent)
	if transitionMessage != "" {
		posts = append(posts, llm.Post{
			Role:    llm.PostRoleSystem,
			Message: transitionMessage,
		})
	}

	recentHistory := h.getRecentConversationHistory(bot, previousConversation, 3, postToAIPost)
	posts = append(posts, recentHistory...)

	return posts, nil
}

func (h *BaseConversationHandler) buildTransitionMessage(previousIntent, currentIntent string) string {
	prevType := h.intentHelper.GetDisplayName(previousIntent)
	currType := h.intentHelper.GetDisplayName(currentIntent)

	if prevType == currType {
		return ""
	}

	return fmt.Sprintf("Context: The user is now shifting from %s to %s. Please adapt your responses accordingly while maintaining awareness of the previous conversation.", prevType, currType)
}

func (h *BaseConversationHandler) getRecentConversationHistory(
	bot *bots.Bot,
	conversation *mmapi.ThreadData,
	limit int,
	postToAIPost func(*bots.Bot, *model.Post) llm.Post,
) []llm.Post {
	if len(conversation.Posts) == 0 {
		return nil
	}

	start := len(conversation.Posts) - limit
	if start < 1 {
		start = 1
	}

	var recentPosts []*model.Post
	for i := start; i < len(conversation.Posts); i++ {
		recentPosts = append(recentPosts, conversation.Posts[i])
	}

	var llmPosts []llm.Post
	for _, post := range recentPosts {
		aiPost := postToAIPost(bot, post)

		if aiPost.Role == llm.PostRoleUser {
			if user, exists := conversation.UsersByID[post.UserId]; exists {
				aiPost.Message = "@" + user.Username + ": " + aiPost.Message
			}
		}

		llmPosts = append(llmPosts, aiPost)
	}

	return llmPosts
}
