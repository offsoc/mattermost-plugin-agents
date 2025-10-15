// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package dev

import (
	"github.com/mattermost/mattermost-plugin-ai/bots"
	"github.com/mattermost/mattermost-plugin-ai/conversations/rolebot"
	"github.com/mattermost/mattermost-plugin-ai/llm"
	"github.com/mattermost/mattermost-plugin-ai/mmapi"
	"github.com/mattermost/mattermost/server/public/model"
)

// ConversationHandler handles dev-specific conversation logic
type ConversationHandler struct {
	base *rolebot.BaseConversationHandler
}

// NewConversationHandler creates a new dev conversation handler
func NewConversationHandler(mmClient mmapi.Client, prompts *llm.Prompts) *ConversationHandler {
	intentHelper := &IntentHelper{}
	base := rolebot.NewBaseConversationHandler(mmClient, prompts, intentHelper, "Dev")
	return &ConversationHandler{
		base: base,
	}
}

// ProcessDevBotRequest handles requests to dev bots with per-message intent detection
func (h *ConversationHandler) ProcessDevBotRequest(
	bot *bots.Bot,
	postingUser *model.User,
	channel *model.Channel,
	post *model.Post,
	context *llm.Context,
	postToAIPost func(*bots.Bot, *model.Post) llm.Post,
	existingConvToLLMPosts func(*bots.Bot, *mmapi.ThreadData, *llm.Context) ([]llm.Post, error),
	generateTitle func(*bots.Bot, string, string, *llm.Context) error,
) (*llm.TextStreamResult, error) {
	titlePromptPrefix := "Write a short title for the following development question. Include only the title and nothing else, no quotations. Question:\n"
	return h.base.ProcessBotRequest(bot, postingUser, channel, post, context, postToAIPost, existingConvToLLMPosts, generateTitle, titlePromptPrefix)
}

// DetectIntent analyzes a message and returns the appropriate dev intent prompt
func DetectIntent(message string) string {
	helper := &IntentHelper{}
	return helper.DetectIntent(message)
}
