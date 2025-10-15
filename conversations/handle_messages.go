// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package conversations

import (
	"context"
	"errors"
	"fmt"

	"github.com/mattermost/mattermost-plugin-ai/bots"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

const (
	ActivateAIProp  = "activate_ai"
	FromWebhookProp = "from_webhook"
	FromBotProp     = "from_bot"
	FromPluginProp  = "from_plugin"
	WranglerProp    = "wrangler"
)

var (
	// ErrNoResponse is returned when no response is posted under a normal condition.
	ErrNoResponse = errors.New("no response")
)

func (c *Conversations) MessageHasBeenPosted(ctx *plugin.Context, post *model.Post) {
	requestCtx := context.Background()
	if err := c.handleMessages(requestCtx, post); err != nil {
		if errors.Is(err, ErrNoResponse) {
			c.mmClient.LogDebug("No response required", "reason", err.Error(), "post_id", post.Id)
		} else {
			c.mmClient.LogError("Error handling message", "error", err.Error(), "post_id", post.Id)
		}
	} else {
		c.mmClient.LogDebug("Message handled successfully", "post_id", post.Id)
	}
}

func (c *Conversations) handleMessages(ctx context.Context, post *model.Post) error {
	c.mmClient.LogDebug("Starting message filtering checks", "post_id", post.Id)

	// Don't respond to ourselves
	if c.bots.IsAnyBot(post.UserId) {
		c.mmClient.LogDebug("Ignoring bot message", "post_id", post.Id, "user_id", post.UserId)
		return fmt.Errorf("not responding to ourselves: %w", ErrNoResponse)
	}

	// Never respond to remote posts
	if post.RemoteId != nil && *post.RemoteId != "" {
		c.mmClient.LogDebug("Ignoring remote post", "post_id", post.Id, "remote_id", *post.RemoteId)
		return fmt.Errorf("not responding to remote posts: %w", ErrNoResponse)
	}

	// Wrangler posts should be ignored
	if post.GetProp(WranglerProp) != nil {
		c.mmClient.LogDebug("Ignoring wrangler post", "post_id", post.Id)
		return fmt.Errorf("not responding to wrangler posts: %w", ErrNoResponse)
	}

	// Don't respond to plugins unless they ask for it
	if post.GetProp(FromPluginProp) != nil && post.GetProp(ActivateAIProp) == nil {
		c.mmClient.LogDebug("Ignoring plugin post without activation", "post_id", post.Id)
		return fmt.Errorf("not responding to plugin posts: %w", ErrNoResponse)
	}

	// Don't respond to webhooks
	if post.GetProp(FromWebhookProp) != nil {
		c.mmClient.LogDebug("Ignoring webhook post", "post_id", post.Id)
		return fmt.Errorf("not responding to webhook posts: %w", ErrNoResponse)
	}

	c.mmClient.LogDebug("Getting channel info", "post_id", post.Id, "channel_id", post.ChannelId)
	channel, err := c.mmClient.GetChannel(post.ChannelId)
	if err != nil {
		c.mmClient.LogError("Failed to get channel", "error", err, "post_id", post.Id, "channel_id", post.ChannelId)
		return fmt.Errorf("unable to get channel: %w", err)
	}
	c.mmClient.LogDebug("Channel retrieved", "post_id", post.Id, "channel_type", channel.Type, "channel_name", channel.Name)

	c.mmClient.LogDebug("Getting user info", "post_id", post.Id, "user_id", post.UserId)
	postingUser, err := c.mmClient.GetUser(post.UserId)
	if err != nil {
		c.mmClient.LogError("Failed to get user", "error", err, "post_id", post.Id, "user_id", post.UserId)
		return err
	}
	c.mmClient.LogDebug("User retrieved", "post_id", post.Id, "username", postingUser.Username, "is_bot", postingUser.IsBot)

	// Don't respond to other bots unless they ask for it
	if (postingUser.IsBot || post.GetProp(FromBotProp) != nil) && post.GetProp(ActivateAIProp) == nil {
		c.mmClient.LogDebug("Ignoring other bot without activation", "post_id", post.Id, "user_id", post.UserId, "is_bot", postingUser.IsBot)
		return fmt.Errorf("not responding to other bots: %w", ErrNoResponse)
	}

	// Check we are mentioned like @ai
	c.mmClient.LogDebug("Checking for bot mentions", "post_id", post.Id, "message", post.Message)
	if bot := c.bots.GetBotMentioned(post.Message); bot != nil {
		c.mmClient.LogDebug("Bot mention detected", "post_id", post.Id, "bot_id", bot.GetMMBot().UserId, "bot_name", bot.GetConfig().Name)
		return c.handleMentions(ctx, bot, post, postingUser, channel)
	}

	// Check if this is post in the DM channel with any bot
	c.mmClient.LogDebug("Checking for DM with bot", "post_id", post.Id, "channel_type", channel.Type)
	if bot := c.bots.GetBotForDMChannel(channel); bot != nil {
		c.mmClient.LogDebug("DM with bot detected", "post_id", post.Id, "bot_id", bot.GetMMBot().UserId, "bot_name", bot.GetConfig().Name)
		return c.handleDMs(ctx, bot, channel, postingUser, post)
	}

	c.mmClient.LogDebug("No action needed - not a mention or DM", "post_id", post.Id)
	return nil
}

func (c *Conversations) handleMentions(ctx context.Context, bot *bots.Bot, post *model.Post, postingUser *model.User, channel *model.Channel) error {
	c.mmClient.LogDebug("Checking usage restrictions", "post_id", post.Id, "bot_name", bot.GetConfig().Name)
	if err := c.bots.CheckUsageRestrictions(postingUser.Id, bot, channel); err != nil {
		c.mmClient.LogError("Usage restrictions failed", "error", err, "post_id", post.Id, "bot_name", bot.GetConfig().Name)
		return err
	}

	c.mmClient.LogDebug("Processing user request", "post_id", post.Id, "bot_name", bot.GetConfig().Name)
	stream, err := c.ProcessUserRequest(bot, postingUser, channel, post)
	if err != nil {
		c.mmClient.LogError("Failed to process user request", "error", err, "post_id", post.Id, "bot_name", bot.GetConfig().Name)
		return fmt.Errorf("unable to process bot mention: %w", err)
	}

	responseRootID := post.Id
	if post.RootId != "" {
		responseRootID = post.RootId
	}

	c.mmClient.LogDebug("Streaming response", "post_id", post.Id, "response_root_id", responseRootID, "bot_name", bot.GetConfig().Name)
	responsePost := &model.Post{
		ChannelId: channel.Id,
		RootId:    responseRootID,
	}
	if err := c.streamingService.StreamToNewPost(ctx, bot.GetMMBot().UserId, postingUser.Id, stream, responsePost, post.Id); err != nil {
		c.mmClient.LogError("Failed to stream response", "error", err, "post_id", post.Id, "bot_name", bot.GetConfig().Name)
		return fmt.Errorf("unable to stream response: %w", err)
	}

	c.mmClient.LogDebug("Mention handled successfully", "post_id", post.Id, "bot_name", bot.GetConfig().Name)
	return nil
}

func (c *Conversations) handleDMs(ctx context.Context, bot *bots.Bot, channel *model.Channel, postingUser *model.User, post *model.Post) error {
	c.mmClient.LogDebug("Checking DM usage restrictions", "post_id", post.Id, "bot_name", bot.GetConfig().Name)
	if err := c.bots.CheckUsageRestrictionsForUser(bot, postingUser.Id); err != nil {
		c.mmClient.LogError("DM usage restrictions failed", "error", err, "post_id", post.Id, "bot_name", bot.GetConfig().Name)
		return err
	}

	c.mmClient.LogDebug("Processing DM user request", "post_id", post.Id, "bot_name", bot.GetConfig().Name)
	stream, err := c.ProcessUserRequest(bot, postingUser, channel, post)
	if err != nil {
		c.mmClient.LogError("Failed to process DM user request", "error", err, "post_id", post.Id, "bot_name", bot.GetConfig().Name)
		return fmt.Errorf("unable to process bot mention: %w", err)
	}

	responseRootID := post.Id
	if post.RootId != "" {
		responseRootID = post.RootId
	}

	c.mmClient.LogDebug("Streaming DM response", "post_id", post.Id, "response_root_id", responseRootID, "bot_name", bot.GetConfig().Name)
	responsePost := &model.Post{
		ChannelId: channel.Id,
		RootId:    responseRootID,
	}
	if err := c.streamingService.StreamToNewPost(ctx, bot.GetMMBot().UserId, postingUser.Id, stream, responsePost, post.Id); err != nil {
		c.mmClient.LogError("Failed to stream DM response", "error", err, "post_id", post.Id, "bot_name", bot.GetConfig().Name)
		return fmt.Errorf("unable to stream response: %w", err)
	}

	c.mmClient.LogDebug("DM handled successfully", "post_id", post.Id, "bot_name", bot.GetConfig().Name)
	return nil
}
