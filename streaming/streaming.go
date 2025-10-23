// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package streaming

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/mattermost/mattermost-plugin-ai/i18n"
	"github.com/mattermost/mattermost-plugin-ai/llm"
	"github.com/mattermost/mattermost-plugin-ai/mmapi"
	"github.com/mattermost/mattermost/server/public/model"
)

const PostStreamingControlCancel = "cancel"
const PostStreamingControlEnd = "end"
const PostStreamingControlStart = "start"

const ToolCallProp = "pending_tool_call"
const ReasoningSummaryProp = "reasoning_summary"
const ReasoningSignatureProp = "reasoning_signature"
const AnnotationsProp = "annotations"
const ArtifactsProp = "artifacts"

type Service interface {
	StreamToNewPost(ctx context.Context, botID string, requesterUserID string, stream *llm.TextStreamResult, post *model.Post, respondingToPostID string) error
	StreamToNewDM(ctx context.Context, botID string, stream *llm.TextStreamResult, userID string, post *model.Post, respondingToPostID string) error
	StreamToPost(ctx context.Context, stream *llm.TextStreamResult, post *model.Post, userLocale string)
	StopStreaming(postID string)
	GetStreamingContext(inCtx context.Context, postID string) (context.Context, error)
	FinishStreaming(postID string)
}

type postStreamContext struct {
	cancel context.CancelFunc
}

var ErrAlreadyStreamingToPost = fmt.Errorf("already streaming to post")

type MMPostStreamService struct {
	contexts      map[string]postStreamContext
	contextsMutex sync.Mutex
	mmClient      mmapi.Client
	i18n          *i18n.Bundle
}

func NewMMPostStreamService(mmClient mmapi.Client, i18n *i18n.Bundle) *MMPostStreamService {
	return &MMPostStreamService{
		contexts: make(map[string]postStreamContext),
		mmClient: mmClient,
		i18n:     i18n,
	}
}

func (p *MMPostStreamService) StreamToNewPost(ctx context.Context, botID string, requesterUserID string, stream *llm.TextStreamResult, post *model.Post, respondingToPostID string) error {
	// We use ModifyPostForBot directly here to add the responding to post ID
	ModifyPostForBot(botID, requesterUserID, post, respondingToPostID)

	if err := p.mmClient.CreatePost(post); err != nil {
		return fmt.Errorf("unable to create post: %w", err)
	}

	// The callback is already set when creating the context

	ctx, err := p.GetStreamingContext(context.Background(), post.Id)
	if err != nil {
		return err
	}

	go func() {
		defer p.FinishStreaming(post.Id)
		user, err := p.mmClient.GetUser(requesterUserID)
		locale := *p.mmClient.GetConfig().LocalizationSettings.DefaultServerLocale
		if err != nil {
			p.StreamToPost(ctx, stream, post, locale)
			return
		}

		channel, err := p.mmClient.GetChannel(post.ChannelId)
		if err != nil {
			p.StreamToPost(ctx, stream, post, locale)
			return
		}

		if channel.Type == model.ChannelTypeDirect {
			if channel.Name == botID+"__"+user.Id || channel.Name == user.Id+"__"+botID {
				p.StreamToPost(ctx, stream, post, user.Locale)
				return
			}
		}
		p.StreamToPost(ctx, stream, post, locale)
	}()

	return nil
}

func (p *MMPostStreamService) StreamToNewDM(ctx context.Context, botID string, stream *llm.TextStreamResult, userID string, post *model.Post, respondingToPostID string) error {
	// We use ModifyPostForBot directly here to add the responding to post ID
	ModifyPostForBot(botID, userID, post, respondingToPostID)

	if err := p.mmClient.DM(botID, userID, post); err != nil {
		return fmt.Errorf("failed to post DM: %w", err)
	}

	// The callback is already set when creating the context

	ctx, err := p.GetStreamingContext(context.Background(), post.Id)
	if err != nil {
		return err
	}

	go func() {
		defer p.FinishStreaming(post.Id)
		user, err := p.mmClient.GetUser(userID)
		locale := *p.mmClient.GetConfig().LocalizationSettings.DefaultServerLocale
		if err != nil {
			p.StreamToPost(ctx, stream, post, locale)
			return
		}

		channel, err := p.mmClient.GetChannel(post.ChannelId)
		if err != nil {
			p.StreamToPost(ctx, stream, post, locale)
			return
		}

		if channel.Type == model.ChannelTypeDirect {
			if channel.Name == botID+"__"+user.Id || channel.Name == user.Id+"__"+botID {
				p.StreamToPost(ctx, stream, post, user.Locale)
				return
			}
		}
		p.StreamToPost(ctx, stream, post, locale)
	}()

	return nil
}

func (p *MMPostStreamService) sendPostStreamingUpdateEvent(post *model.Post, message string) {
	p.mmClient.PublishWebSocketEvent("postupdate", map[string]interface{}{
		"post_id": post.Id,
		"next":    message,
	}, &model.WebsocketBroadcast{
		ChannelId: post.ChannelId,
	})
}

func (p *MMPostStreamService) sendPostStreamingControlEvent(post *model.Post, control string) {
	p.mmClient.PublishWebSocketEvent("postupdate", map[string]interface{}{
		"post_id": post.Id,
		"control": control,
	}, &model.WebsocketBroadcast{
		ChannelId: post.ChannelId,
	})
}

func (p *MMPostStreamService) sendPostStreamingReasoningEvent(post *model.Post, reasoning string, control string) {
	p.mmClient.PublishWebSocketEvent("postupdate", map[string]interface{}{
		"post_id":   post.Id,
		"control":   control,
		"reasoning": reasoning,
	}, &model.WebsocketBroadcast{
		ChannelId: post.ChannelId,
	})
}

func (p *MMPostStreamService) sendPostStreamingAnnotationsEvent(post *model.Post, annotations string) {
	p.mmClient.PublishWebSocketEvent("postupdate", map[string]interface{}{
		"post_id":     post.Id,
		"control":     "annotations",
		"annotations": annotations,
	}, &model.WebsocketBroadcast{
		ChannelId: post.ChannelId,
	})
}

func (p *MMPostStreamService) StopStreaming(postID string) {
	p.contextsMutex.Lock()
	defer p.contextsMutex.Unlock()
	if streamContext, ok := p.contexts[postID]; ok {
		streamContext.cancel()
	}
	delete(p.contexts, postID)
}

func (p *MMPostStreamService) GetStreamingContext(inCtx context.Context, postID string) (context.Context, error) {
	p.contextsMutex.Lock()
	defer p.contextsMutex.Unlock()

	if _, ok := p.contexts[postID]; ok {
		return nil, ErrAlreadyStreamingToPost
	}

	ctx, cancel := context.WithCancel(inCtx)

	streamingContext := postStreamContext{
		cancel: cancel,
	}

	p.contexts[postID] = streamingContext

	return ctx, nil
}

// FinishStreaming should be called when a post streaming operation is finished on success or failure.
// It is safe to call multiple times, must be called at least once.
func (p *MMPostStreamService) FinishStreaming(postID string) {
	p.contextsMutex.Lock()
	defer p.contextsMutex.Unlock()
	delete(p.contexts, postID)
}

// StreamToPost streams the result of a TextStreamResult to a post.
// it will internally handle logging needs and updating the post.
func (p *MMPostStreamService) StreamToPost(ctx context.Context, stream *llm.TextStreamResult, post *model.Post, userLocale string) {
	T := i18n.LocalizerFunc(p.i18n, userLocale)
	p.sendPostStreamingControlEvent(post, PostStreamingControlStart)
	defer func() {
		p.sendPostStreamingControlEvent(post, PostStreamingControlEnd)
	}()

	var reasoningBuffer strings.Builder
	var artifactGeneratingSent bool
	var suppressingPotentialArtifact bool

	for {
		select {
		case event := <-stream.Stream:
			switch event.Type {
			case llm.EventTypeText:
				// Handle text event
				if textChunk, ok := event.Value.(string); ok {
					post.Message += textChunk

					// Handle artifact detection and message cleaning
					hasArtifact := detectArtifactOpening(post.Message)
					mightBeArtifact := isPotentialArtifactStart(post.Message)

					switch {
					case !artifactGeneratingSent && hasArtifact:
						// Confirmed artifact - send generating event
						metadata := extractArtifactMetadata(post.Message)
						p.sendArtifactGeneratingEvent(post, metadata)
						artifactGeneratingSent = true
						suppressingPotentialArtifact = false

						// Remove incomplete artifact from displayed message
						cleanedMessage := removeIncompleteArtifact(post.Message)
						p.sendPostStreamingUpdateEvent(post, cleanedMessage)
					case !artifactGeneratingSent && mightBeArtifact:
						// Might be starting an artifact - suppress output until confirmed
						suppressingPotentialArtifact = true
						// Don't send update - wait to see if it's an artifact
					case !artifactGeneratingSent && suppressingPotentialArtifact:
						// Was suppressing but no artifact detected - it's a normal code block
						suppressingPotentialArtifact = false
						p.sendPostStreamingUpdateEvent(post, post.Message)
					case artifactGeneratingSent:
						// We're inside an artifact block, keep removing incomplete content
						cleanedMessage := removeIncompleteArtifact(post.Message)
						p.sendPostStreamingUpdateEvent(post, cleanedMessage)

						// Stream the partial artifact content to the artifact viewer
						artifactContent := extractArtifactContent(post.Message)
						if artifactContent != "" {
							p.sendArtifactStreamingEvent(post, artifactContent)
						}
					default:
						// Normal text, stream as-is
						p.sendPostStreamingUpdateEvent(post, post.Message)
					}
				}
			case llm.EventTypeEnd:
				// Stream has closed cleanly
				if strings.TrimSpace(post.Message) == "" {
					p.mmClient.LogError("LLM closed stream with no result")
					post.Message = T("agents.stream_to_post_llm_not_return", "Sorry! The LLM did not return a result.")
					p.sendPostStreamingUpdateEvent(post, post.Message)
				}

				// Inline citations have already been cleaned in EventTypeAnnotations handler
				// (if there were any citations, they were cleaned before annotations were sent)

				// Remove artifact code blocks from the displayed message
				// The artifacts are stored separately in post props and displayed in the artifact viewer
				if artifactsProp := post.GetProp(ArtifactsProp); artifactsProp != nil {
					cleanedMessage := llm.RemoveArtifactMarkers(post.Message)
					if cleanedMessage != post.Message {
						post.Message = cleanedMessage
						p.sendPostStreamingUpdateEvent(post, post.Message)
						p.mmClient.LogDebug("Removed artifact markers from post message", "post_id", post.Id)
					}
				}

				// Update post with all accumulated data
				// This includes the message and any reasoning that was added to props in EventTypeReasoningEnd
				if reasoningProp := post.GetProp(ReasoningSummaryProp); reasoningProp != nil {
					p.mmClient.LogDebug("Persisting post with reasoning summary", "post_id", post.Id)
				}
				if err := p.mmClient.UpdatePost(post); err != nil {
					p.mmClient.LogError("Streaming failed to update post", "error", err)
					return
				}
				return
			case llm.EventTypeError:
				// Handle error event
				var err error
				if errValue, ok := event.Value.(error); ok {
					err = errValue
				} else {
					err = fmt.Errorf("unknown error from LLM")
				}

				// Handle partial results
				if strings.TrimSpace(post.Message) == "" {
					post.Message = ""
				} else {
					post.Message += "\n\n"
				}
				p.mmClient.LogError("Streaming result to post failed partway", "error", err)
				post.Message = T("agents.stream_to_post_access_llm_error", "Sorry! An error occurred while accessing the LLM. See server logs for details.")

				// Persist any accumulated reasoning before erroring out
				if reasoningBuffer.Len() > 0 {
					post.AddProp(ReasoningSummaryProp, reasoningBuffer.String())
					p.mmClient.LogDebug("Saved partial reasoning summary on error", "post_id", post.Id, "reasoning_length", reasoningBuffer.Len())
				}

				if err := p.mmClient.UpdatePost(post); err != nil {
					p.mmClient.LogError("Error recovering from streaming error", "error", err)
					return
				}
				p.sendPostStreamingUpdateEvent(post, post.Message)
				return
			case llm.EventTypeReasoning:
				// Handle reasoning summary chunk - accumulate and stream
				if reasoningChunk, ok := event.Value.(string); ok {
					reasoningBuffer.WriteString(reasoningChunk)
					// Send reasoning event with accumulated text so far
					p.sendPostStreamingReasoningEvent(post, reasoningBuffer.String(), "reasoning_summary")
				}
			case llm.EventTypeReasoningEnd:
				// Reasoning summary completed - stream final and persist
				if reasoningData, ok := event.Value.(llm.ReasoningData); ok {
					// Send final reasoning event (only text goes to frontend)
					p.sendPostStreamingReasoningEvent(post, reasoningData.Text, "reasoning_summary_done")

					// Persist reasoning summary and signature to post props
					// This will be saved when the post is updated at the end of the stream
					if reasoningData.Text != "" {
						post.AddProp(ReasoningSummaryProp, reasoningData.Text)
						p.mmClient.LogDebug("Added reasoning summary to post props", "post_id", post.Id, "reasoning_length", len(reasoningData.Text))
					}
					if reasoningData.Signature != "" {
						post.AddProp(ReasoningSignatureProp, reasoningData.Signature)
						p.mmClient.LogDebug("Added reasoning signature to post props", "post_id", post.Id)
					}
					reasoningBuffer.Reset()
				}
			case llm.EventTypeToolCalls:
				// Handle tool call event
				if toolCalls, ok := event.Value.([]llm.ToolCall); ok {
					// Ensure all tool calls have Pending status
					for i := range toolCalls {
						toolCalls[i].Status = llm.ToolCallStatusPending
					}

					// Add the tool call as a prop to the post
					toolCallJSON, err := json.Marshal(toolCalls)
					if err != nil {
						p.mmClient.LogError("Failed to marshal tool call", "error", err)
					} else {
						post.AddProp(ToolCallProp, string(toolCallJSON))
					}

					// Update the post with the tool call and any reasoning that was previously added
					if err := p.mmClient.UpdatePost(post); err != nil {
						p.mmClient.LogError("Failed to update post with tool call", "error", err)
					}

					// Send websocket event with tool call data
					p.mmClient.PublishWebSocketEvent("postupdate", map[string]interface{}{
						"post_id":   post.Id,
						"control":   "tool_call",
						"tool_call": string(toolCallJSON),
					}, &model.WebsocketBroadcast{
						ChannelId: post.ChannelId,
					})
				}
				return
			case llm.EventTypeAnnotations:
				if annotations, ok := event.Value.([]llm.Annotation); ok {
					annotationsJSON, err := json.Marshal(annotations)
					if err != nil {
						p.mmClient.LogError("Failed to marshal annotations", "error", err)
					} else {
						post.AddProp(AnnotationsProp, string(annotationsJSON))
						p.mmClient.LogDebug("Added annotations to post props", "post_id", post.Id, "count", len(annotations))
						p.sendPostStreamingAnnotationsEvent(post, string(annotationsJSON))
					}
				}
			case llm.EventTypeArtifact:
				if artifact, ok := event.Value.(llm.Artifact); ok {
					// Get existing artifacts array or create new one
					var artifacts []llm.Artifact
					if existingProp := post.GetProp(ArtifactsProp); existingProp != nil {
						if existingJSON, ok := existingProp.(string); ok {
							_ = json.Unmarshal([]byte(existingJSON), &artifacts)
						}
					}

					// Append new artifact
					artifacts = append(artifacts, artifact)

					// Marshal and store
					artifactsJSON, err := json.Marshal(artifacts)
					if err != nil {
						p.mmClient.LogError("Failed to marshal artifacts", "error", err)
					} else {
						post.AddProp(ArtifactsProp, string(artifactsJSON))
						p.mmClient.LogDebug("Added artifact to post props", "post_id", post.Id, "title", artifact.Title, "type", artifact.Type)

						// Send WebSocket event for artifact
						p.mmClient.PublishWebSocketEvent("postupdate", map[string]interface{}{
							"post_id":  post.Id,
							"control":  "artifact",
							"artifact": string(artifactsJSON),
						}, &model.WebsocketBroadcast{
							ChannelId: post.ChannelId,
						})
					}
				}
			}
		case <-ctx.Done():
			// Persist any accumulated reasoning before canceling
			if reasoningBuffer.Len() > 0 {
				post.AddProp(ReasoningSummaryProp, reasoningBuffer.String())
				p.mmClient.LogDebug("Saved partial reasoning summary on cancel", "post_id", post.Id, "reasoning_length", reasoningBuffer.Len())
			}

			if err := p.mmClient.UpdatePost(post); err != nil {
				p.mmClient.LogError("Error updating post on stop signaled", "error", err)
				return
			}
			p.sendPostStreamingControlEvent(post, PostStreamingControlCancel)
			return
		}
	}
}

// detectArtifactOpening checks if the text contains the start of an artifact block
func detectArtifactOpening(text string) bool {
	return strings.Contains(text, "```artifact:")
}

// isPotentialArtifactStart checks if text might be starting an artifact block
// This detects earlier than detectArtifactOpening to prevent showing "```artifact" during streaming
func isPotentialArtifactStart(text string) bool {
	// Check if the last line starts with ``` or if the whole message starts with ```
	trimmed := strings.TrimSpace(text)
	if strings.HasPrefix(trimmed, "```") {
		return true
	}

	// Check if the last line (after last newline) starts with ```
	lastNewline := strings.LastIndex(text, "\n")
	if lastNewline != -1 && lastNewline < len(text)-1 {
		lastLine := strings.TrimSpace(text[lastNewline+1:])
		if strings.HasPrefix(lastLine, "```") {
			return true
		}
	}

	return false
}

// extractArtifactMetadata extracts language and title from a partial or complete artifact opening
// Returns map with "language" and optionally "title" keys
func extractArtifactMetadata(text string) map[string]interface{} {
	metadata := make(map[string]interface{})

	// Find the artifact opening pattern
	// Pattern: ```artifact:language title="optional title"
	pattern := regexp.MustCompile("```artifact:([a-zA-Z0-9]+)(?:\\s+title=\"([^\"]+)\")?")
	matches := pattern.FindStringSubmatch(text)

	if len(matches) > 1 {
		metadata["language"] = matches[1]
		if len(matches) > 2 && matches[2] != "" {
			metadata["title"] = matches[2]
		}
	}

	return metadata
}

// removeIncompleteArtifact removes everything from the artifact opening tag onwards
// This is used during streaming when the artifact block is still being generated
func removeIncompleteArtifact(text string) string {
	// Find the position of ```artifact:
	index := strings.Index(text, "```artifact:")
	if index == -1 {
		return text
	}
	// Return everything before the artifact marker, trimming trailing whitespace
	result := strings.TrimRight(text[:index], " \n\t")
	return result
}

// extractArtifactContent extracts the partial content from an incomplete artifact block
// Returns the code content between the opening tag and current position
func extractArtifactContent(text string) string {
	// Find the artifact opening
	index := strings.Index(text, "```artifact:")
	if index == -1 {
		return ""
	}

	// Find the end of the first line (title line)
	afterOpening := text[index:]
	firstNewline := strings.Index(afterOpening, "\n")
	if firstNewline == -1 {
		return ""
	}

	// Extract content after the title line
	contentStart := index + firstNewline + 1
	if contentStart >= len(text) {
		return ""
	}

	content := text[contentStart:]

	// Check if artifact is complete by looking for closing fence
	// Must be ``` followed by newline or end of string (not ```language)
	// Search from the end to find the last occurrence
	for i := len(content) - 3; i >= 0; i-- {
		if i >= 1 && content[i-1] == '\n' && content[i] == '`' && content[i+1] == '`' && content[i+2] == '`' {
			// Found \n``` - check what comes after
			afterBackticks := i + 3
			if afterBackticks >= len(content) {
				// ``` is at the end - this is the closing fence
				content = content[:i-1]
				break
			}
			nextChar := content[afterBackticks]
			if nextChar == '\n' || nextChar == '\r' {
				// ``` followed by newline - this is the closing fence
				content = content[:i-1]
				break
			}
			// ``` followed by other characters (like ```javascript) - this is a nested code block, keep searching
		}
	}

	return content
}

// sendArtifactGeneratingEvent sends a WebSocket event indicating an artifact is being generated
func (p *MMPostStreamService) sendArtifactGeneratingEvent(post *model.Post, metadata map[string]interface{}) {
	p.mmClient.PublishWebSocketEvent("postupdate", map[string]interface{}{
		"post_id":  post.Id,
		"control":  "artifact_generating",
		"metadata": metadata,
	}, &model.WebsocketBroadcast{
		ChannelId: post.ChannelId,
	})
}

// sendArtifactStreamingEvent sends the current partial artifact content
func (p *MMPostStreamService) sendArtifactStreamingEvent(post *model.Post, content string) {
	p.mmClient.PublishWebSocketEvent("postupdate", map[string]interface{}{
		"post_id": post.Id,
		"control": "artifact_streaming",
		"content": content,
	}, &model.WebsocketBroadcast{
		ChannelId: post.ChannelId,
	})
}
