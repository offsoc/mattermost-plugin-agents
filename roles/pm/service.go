// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package pm

import (
	"github.com/mattermost/mattermost-plugin-ai/bots"
	"github.com/mattermost/mattermost-plugin-ai/conversations"
	"github.com/mattermost/mattermost-plugin-ai/llm"
	"github.com/mattermost/mattermost-plugin-ai/mmapi"
	"github.com/mattermost/mattermost/server/public/model"
)

// Service manages PM role functionality for bots
type Service struct {
	conversationHandler *ConversationHandler
	mmClient            mmapi.Client
	prompts             *llm.Prompts
	conv                *conversations.Conversations
}

// NewService creates a new PM role service
func NewService(mmClient mmapi.Client, prompts *llm.Prompts) *Service {
	return &Service{
		conversationHandler: NewConversationHandler(mmClient, prompts),
		mmClient:            mmClient,
		prompts:             prompts,
	}
}

// RegisterWithConversations registers PM handlers with the conversations system
func (s *Service) RegisterWithConversations(conv *conversations.Conversations) {
	// Store conversations instance for use in PM bot processor
	s.conv = conv

	// Register PM intent
	conv.RegisterIntent(&Intent{})

	// Register PM bot processor - simple function that delegates to PM handler
	conv.RegisterBotProcessor("pm", s.processPMBot)
	conv.RegisterBotProcessor("project manager", s.processPMBot)
	conv.RegisterBotProcessor("product manager", s.processPMBot)
}

// processPMBot is the simple processor function that delegates to PM conversation handler
func (s *Service) processPMBot(bot *bots.Bot, postingUser *model.User, channel *model.Channel, post *model.Post, context *llm.Context) (*llm.TextStreamResult, error) {
	// Pass helper methods from the conversations instance
	return s.conversationHandler.ProcessPMBotRequest(
		bot, postingUser, channel, post, context,
		s.conv.PostToAIPost,
		s.conv.ExistingConversationToLLMPosts,
		s.conv.GenerateTitle,
	)
}
