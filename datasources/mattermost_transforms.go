// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"fmt"
	"strings"
	"time"
)

// MattermostTransformer handles data transformation from Mattermost API types to Doc format
type MattermostTransformer struct {
	topicAnalyzer   *TopicAnalyzer
	universalScorer *UniversalRelevanceScorer
}

// NewMattermostTransformer creates a new transformer instance
func NewMattermostTransformer() *MattermostTransformer {
	return &MattermostTransformer{
		topicAnalyzer:   NewTopicAnalyzer(),
		universalScorer: NewUniversalRelevanceScorer(),
	}
}

// ConvertPostsToDoc converts Mattermost posts to Doc format, preserving threading structure,
// user attributions, reply counts, pinned status, edit timestamps, and reaction metadata for context
func (t *MattermostTransformer) ConvertPostsToDoc(posts map[string]MattermostPost, order []string, baseURL, sourceName, topic string) []Doc {
	var docs []Doc

	for _, postID := range order {
		if post, exists := posts[postID]; exists {
			meta := extractMattermostMetadata(post)
			content := t.FormatPostContent(post)

			labels := buildLabelsFromMetadata(meta)
			if post.IsPinned {
				labels = append(labels, "pinned")
			}
			if post.ReplyCount > 0 {
				labels = append(labels, "has_replies")
			}
			if post.RootID != "" {
				labels = append(labels, "is_reply")
			}

			lastModified := t.FormatTimestamp(post.CreateAt)
			if post.EditAt > 0 {
				lastModified = t.FormatTimestamp(post.EditAt)
			}

			createdDate := t.FormatTimestamp(post.CreateAt)
			daysCreated := DaysSince(createdDate)
			if recencyLabel := FormatRecencyLabel(daysCreated); recencyLabel != "" {
				labels = append(labels, recencyLabel+"_created")
			}
			daysUpdated := DaysSince(lastModified)
			if recencyLabel := FormatRecencyLabel(daysUpdated); recencyLabel != "" {
				labels = append(labels, recencyLabel+"_updated")
			}

			doc := Doc{
				Title:        t.GeneratePostTitle(post),
				Content:      content,
				URL:          fmt.Sprintf("%s/pl/%s", strings.TrimRight(baseURL, "/"), postID),
				Section:      "posts",
				Source:       sourceName,
				Labels:       labels,
				Author:       post.UserID,
				CreatedDate:  createdDate,
				LastModified: lastModified,
			}

			if t.universalScorer.IsUniversallyAcceptable(doc.Content, doc.Title, doc.Source, topic) {
				docs = append(docs, doc)
			}
		}
	}

	return docs
}

// FilterAndConvertPosts filters posts by topic and converts them to Doc format
func (t *MattermostTransformer) FilterAndConvertPosts(posts []MattermostPost, topic string, channel *MattermostChannel, baseURL, sourceName, section, teamName string) []Doc {
	var docs []Doc

	for _, post := range posts {
		if post.DeleteAt != 0 || strings.HasPrefix(post.Type, "system_") {
			continue
		}

		if topic != "" && !t.PostMatchesTopic(post, topic) {
			continue
		}

		meta := extractMattermostMetadata(post)

		labels := buildLabelsFromMetadata(meta)
		labels = append(labels, fmt.Sprintf("channel:%s", channel.Name))
		labels = append(labels, fmt.Sprintf("team:%s", teamName))
		if post.IsPinned {
			labels = append(labels, "pinned")
		}
		if post.ReplyCount > 0 {
			labels = append(labels, "has_replies")
		}
		if post.RootID != "" {
			labels = append(labels, "is_reply")
		}

		lastModified := t.FormatTimestamp(post.CreateAt)
		if post.EditAt > 0 {
			lastModified = t.FormatTimestamp(post.EditAt)
		}

		createdDate := t.FormatTimestamp(post.CreateAt)
		daysCreated := DaysSince(createdDate)
		if recencyLabel := FormatRecencyLabel(daysCreated); recencyLabel != "" {
			labels = append(labels, recencyLabel+"_created")
		}
		daysUpdated := DaysSince(lastModified)
		if recencyLabel := FormatRecencyLabel(daysUpdated); recencyLabel != "" {
			labels = append(labels, recencyLabel+"_updated")
		}

		doc := Doc{
			Title:        t.GeneratePostTitle(post),
			Content:      t.FormatPostContentWithChannel(post, channel, teamName),
			URL:          t.BuildPostURL(baseURL, teamName, post.ID),
			Section:      section,
			Source:       sourceName,
			Labels:       labels,
			Author:       post.UserID,
			CreatedDate:  createdDate,
			LastModified: lastModified,
		}

		if t.universalScorer.IsUniversallyAcceptable(doc.Content, doc.Title, doc.Source, topic) {
			docs = append(docs, doc)
		}
	}

	return docs
}

// FormatPostContent formats a Mattermost post for consumption
func (t *MattermostTransformer) FormatPostContent(post MattermostPost) string {
	content := fmt.Sprintf("Posted: %s\n", t.FormatTimestamp(post.CreateAt))

	if post.EditAt > 0 {
		content += fmt.Sprintf("Edited: %s\n", t.FormatTimestamp(post.EditAt))
	}

	if post.ReplyCount > 0 {
		content += fmt.Sprintf("Replies: %d\n", post.ReplyCount)
	}

	content += "\n"
	content += post.Message

	if len(content) > 1000 {
		content = content[:1000] + "..."
	}

	return content
}

// FormatPostContentWithChannel formats a post with channel context
func (t *MattermostTransformer) FormatPostContentWithChannel(post MattermostPost, channel *MattermostChannel, teamName string) string {
	content := fmt.Sprintf("From ~%s", channel.Name)
	if teamName != "" {
		content += fmt.Sprintf(" in %s team", teamName)
	}
	content += "\n"

	content += fmt.Sprintf("Posted: %s\n", t.FormatTimestamp(post.CreateAt))

	if post.EditAt > 0 {
		content += fmt.Sprintf("Edited: %s\n", t.FormatTimestamp(post.EditAt))
	}

	if post.IsPinned {
		content += "📌 Pinned message\n"
	}

	if post.ReplyCount > 0 {
		content += fmt.Sprintf("💬 %d replies\n", post.ReplyCount)
	}

	content += "\n"
	content += post.Message

	if len(content) > 1000 {
		content = content[:1000] + "..."
	}

	return content
}

// GeneratePostTitle generates a meaningful title for a post
func (t *MattermostTransformer) GeneratePostTitle(post MattermostPost) string {
	if post.Message == "" {
		return "Mattermost Post"
	}

	lines := strings.Split(post.Message, "\n")
	title := lines[0]

	if len(title) > 50 {
		title = title[:50] + "..."
	}

	if title == "" {
		title = "Mattermost Post"
	}

	return title
}

// FormatTimestamp formats a Mattermost timestamp for display
func (t *MattermostTransformer) FormatTimestamp(timestamp int64) string {
	if timestamp == 0 {
		return "Unknown"
	}

	tm := time.Unix(timestamp/1000, 0)
	return tm.Format("2006-01-02 15:04")
}

// BuildPostURL constructs the proper URL for a post
func (t *MattermostTransformer) BuildPostURL(baseURL, teamName, postID string) string {
	if teamName != "" {
		return fmt.Sprintf("%s/%s/pl/%s", strings.TrimRight(baseURL, "/"), teamName, postID)
	}
	return fmt.Sprintf("%s/pl/%s", strings.TrimRight(baseURL, "/"), postID)
}

// PostMatchesTopic checks if a post matches the given topic using centralized analysis
func (t *MattermostTransformer) PostMatchesTopic(post MattermostPost, topic string) bool {
	return t.topicAnalyzer.IsTopicRelevantContent(post.Message, topic)
}

// FilterDocsByTopic filters documents based on boolean topic query
func (t *MattermostTransformer) FilterDocsByTopic(docs []Doc, topic string) []Doc {
	if topic == "" {
		return docs
	}

	queryNode, err := ParseBooleanQuery(topic)
	if err != nil {
		return t.filterDocsByKeywords(docs, topic)
	}

	var filtered []Doc
	for _, doc := range docs {
		searchText := doc.Title + " " + doc.Content
		if EvaluateBoolean(queryNode, searchText) {
			filtered = append(filtered, doc)
		}
	}

	return filtered
}

// filterDocsByKeywords is a fallback simple keyword matcher when boolean parsing fails
func (t *MattermostTransformer) filterDocsByKeywords(docs []Doc, topic string) []Doc {
	cleanTopic := strings.NewReplacer(
		"(", " ",
		")", " ",
		" AND ", " ",
		" OR ", " ",
		" NOT ", " ",
		"\"", " ",
	).Replace(strings.ToLower(topic))

	keywords := strings.Fields(cleanTopic)
	if len(keywords) == 0 {
		return docs
	}

	var filtered []Doc
	for _, doc := range docs {
		searchText := strings.ToLower(doc.Title + " " + doc.Content)

		matched := false
		for _, keyword := range keywords {
			if len(keyword) >= 3 && strings.Contains(searchText, keyword) {
				matched = true
				break
			}
		}

		if matched {
			filtered = append(filtered, doc)
		}
	}

	return filtered
}
