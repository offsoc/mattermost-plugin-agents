// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package conversations

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/mattermost/mattermost-plugin-ai/llm"
	"github.com/mattermost/mattermost-plugin-ai/mcp"
	"github.com/mattermost/mattermost-plugin-ai/mmapi"
	"github.com/mattermost/mattermost-plugin-ai/streaming"
	"github.com/mattermost/mattermost/server/public/model"
)

// HandleToolCall handles tool call approval/rejection
func (c *Conversations) HandleToolCall(userID string, post *model.Post, channel *model.Channel, acceptedToolIDs []string, autoApproveTool string) error {
	bot := c.bots.GetBotByID(post.UserId)
	if bot == nil {
		return fmt.Errorf("unable to get bot")
	}

	user, err := c.mmClient.GetUser(userID)
	if err != nil {
		return err
	}

	// Get the root post ID for conversation-scoped permissions
	rootPostID := post.RootId
	if rootPostID == "" {
		rootPostID = post.Id
	}

	// Handle permission updates
	if autoApproveTool != "" {
		if addErr := mcp.AddAutoApproval(c.mmClient, userID, rootPostID, autoApproveTool); addErr != nil {
			c.mmClient.LogError("Failed to add auto-approval for tool", "error", addErr, "tool", autoApproveTool)
		}
	}

	toolsJSON := post.GetProp(streaming.ToolCallProp)
	if toolsJSON == nil {
		return errors.New("post missing pending tool calls")
	}

	var tools []llm.ToolCall
	unmarshalErr := json.Unmarshal([]byte(toolsJSON.(string)), &tools)
	if unmarshalErr != nil {
		return errors.New("post pending tool calls not valid JSON")
	}

	// Load auto-approved tools for this conversation
	autoApprovedTools, err := mcp.GetAutoApprovedTools(c.mmClient, userID, rootPostID)
	if err != nil {
		c.mmClient.LogError("Failed to load auto-approved tools", "error", err)
		autoApprovedTools = []string{} // Continue with empty list on error
	}

	llmContext := c.contextBuilder.BuildLLMContextUserRequest(
		bot,
		user,
		channel,
		c.contextBuilder.WithLLMContextDefaultTools(bot, mmapi.IsDMWith(bot.GetMMBot().UserId, channel)),
	)

	// Check if tool is auto-approved
	isAutoApproved := func(toolName string) bool {
		return slices.Contains(autoApprovedTools, toolName)
	}

	for i := range tools {
		// Mark if tool was auto-approved
		if isAutoApproved(tools[i].Name) {
			tools[i].AutoApproved = true
		}

		// Check if tool should be executed (either explicitly accepted OR auto-approved)
		shouldExecute := slices.Contains(acceptedToolIDs, tools[i].ID) || isAutoApproved(tools[i].Name)

		if shouldExecute {
			result, resolveErr := llmContext.Tools.ResolveTool(tools[i].Name, func(args any) error {
				return json.Unmarshal(tools[i].Arguments, args)
			}, llmContext)
			if resolveErr != nil {
				// Maybe in the future we can return this to the user and have a retry. For now just tell the LLM it failed.
				tools[i].Result = "Tool call failed"
				tools[i].Status = llm.ToolCallStatusError
				continue
			}
			tools[i].Result = result
			tools[i].Status = llm.ToolCallStatusSuccess
		} else {
			tools[i].Result = "Tool call rejected by user"
			tools[i].Status = llm.ToolCallStatusRejected
		}
	}

	responseRootID := post.Id
	if post.RootId != "" {
		responseRootID = post.RootId
	}

	// Update post with the tool call results
	resolvedToolsJSON, err := json.Marshal(tools)
	if err != nil {
		return fmt.Errorf("failed to marshal tool call results: %w", err)
	}
	post.AddProp(streaming.ToolCallProp, string(resolvedToolsJSON))

	if updateErr := c.mmClient.UpdatePost(post); updateErr != nil {
		return fmt.Errorf("failed to update post with tool call results: %w", updateErr)
	}

	// Only continue if at lest one tool call was successful
	if !slices.ContainsFunc(tools, func(tc llm.ToolCall) bool {
		return tc.Status == llm.ToolCallStatusSuccess
	}) {
		return nil
	}

	previousConversation, err := mmapi.GetThreadData(c.mmClient, responseRootID)
	if err != nil {
		return fmt.Errorf("failed to get previous conversation: %w", err)
	}
	previousConversation.CutoffBeforePostID(post.Id)
	previousConversation.Posts = append(previousConversation.Posts, post)

	posts, err := c.existingConversationToLLMPosts(bot, previousConversation, llmContext)
	if err != nil {
		return fmt.Errorf("failed to convert existing conversation to LLM posts: %w", err)
	}

	completionRequest := llm.CompletionRequest{
		Posts:   posts,
		Context: llmContext,
	}
	result, err := bot.LLM().ChatCompletion(completionRequest)
	if err != nil {
		return fmt.Errorf("failed to get chat completion: %w", err)
	}

	responsePost := &model.Post{
		ChannelId: channel.Id,
		RootId:    responseRootID,
	}
	if err := c.streamingService.StreamToNewPost(context.Background(), bot.GetMMBot().UserId, user.Id, result, responsePost, post.Id); err != nil {
		return fmt.Errorf("failed to stream result to new post: %w", err)
	}

	return nil
}

// HandleAutoApprovedToolCall is a simplified handler for auto-approved tool calls
// This is called by the streaming service when all tools in a call are auto-approved
func (c *Conversations) HandleAutoApprovedToolCall(postID string, toolIDs []string) {
	post, err := c.mmClient.GetPost(postID)
	if err != nil {
		c.mmClient.LogError("Failed to get post for auto-approved tool call", "error", err, "post_id", postID)
		return
	}

	channel, err := c.mmClient.GetChannel(post.ChannelId)
	if err != nil {
		c.mmClient.LogError("Failed to get channel for auto-approved tool call", "error", err, "post_id", postID)
		return
	}

	requesterUserID := post.GetProp(streaming.LLMRequesterUserID)
	if requesterUserID == nil {
		c.mmClient.LogError("Post missing requester user ID", "post_id", postID)
		return
	}

	userID, ok := requesterUserID.(string)
	if !ok {
		c.mmClient.LogError("Requester user ID is not a string", "post_id", postID)
		return
	}

	// Verify permissions before executing auto-approved tools.
	// Tool execution can trigger multi-turn conversations (tool results sent back to LLM,
	// which may request more tools), so permissions could have changed since the conversation started.
	// Check channel read permission
	if !c.mmClient.HasPermissionToChannel(userID, channel.Id, model.PermissionReadChannel) {
		c.mmClient.LogError("User no longer has channel permission for auto-approved tool call", "user_id", userID, "channel_id", channel.Id, "post_id", postID)
		return
	}

	// Get bot to check usage restrictions
	bot := c.bots.GetBotByID(post.UserId)
	if bot == nil {
		c.mmClient.LogError("Failed to get bot for auto-approved tool call", "post_id", postID)
		return
	}

	// Check bot usage restrictions
	if err := c.bots.CheckUsageRestrictions(userID, bot, channel); err != nil {
		c.mmClient.LogError("Bot usage restrictions violated for auto-approved tool call", "error", err, "user_id", userID, "post_id", postID)
		return
	}

	// Call HandleToolCall with all tool IDs accepted and no permission changes
	if err := c.HandleToolCall(userID, post, channel, toolIDs, ""); err != nil {
		c.mmClient.LogError("Failed to handle auto-approved tool call", "error", err, "post_id", postID)
	}
}
