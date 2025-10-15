// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package dev

import (
	"github.com/mattermost/mattermost-plugin-ai/bots"
	"github.com/mattermost/mattermost-plugin-ai/conversations"
	"github.com/mattermost/mattermost-plugin-ai/llm"
	"github.com/mattermost/mattermost-plugin-ai/mmapi"
	"github.com/mattermost/mattermost/server/public/model"
)

// Service manages Dev role functionality for bots
type Service struct {
	conversationHandler *ConversationHandler
	mmClient            mmapi.Client
	prompts             *llm.Prompts
	conv                *conversations.Conversations
}

// NewService creates a new dev role service
func NewService(mmClient mmapi.Client, prompts *llm.Prompts) *Service {
	return &Service{
		conversationHandler: NewConversationHandler(mmClient, prompts),
		mmClient:            mmClient,
		prompts:             prompts,
	}
}

// RegisterWithConversations registers dev handlers with the conversations system
func (s *Service) RegisterWithConversations(conv *conversations.Conversations) {
	s.conv = conv

	conv.RegisterIntent(&Intent{})

	conv.RegisterBotProcessor("dev", s.processDevBot)
	conv.RegisterBotProcessor("devbot", s.processDevBot)
	conv.RegisterBotProcessor("developer", s.processDevBot)
}

// processDevBot is the processor function that delegates to dev conversation handler
func (s *Service) processDevBot(bot *bots.Bot, postingUser *model.User, channel *model.Channel, post *model.Post, context *llm.Context) (*llm.TextStreamResult, error) {
	return s.conversationHandler.ProcessDevBotRequest(
		bot, postingUser, channel, post, context,
		s.conv.PostToAIPost,
		s.conv.ExistingConversationToLLMPosts,
		s.conv.GenerateTitle,
	)
}
