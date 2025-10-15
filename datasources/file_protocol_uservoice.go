// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"fmt"
	"strings"

	pm "github.com/mattermost/mattermost-plugin-ai/datasources/segments/pm"
)

// fetchFromUserVoiceJSON processes UserVoice suggestions from JSON
func (f *FileProtocol) fetchFromUserVoiceJSON(suggestions []UserVoiceSuggestion, sourceName string, request ProtocolRequest) ([]Doc, error) {
	var docs []Doc
	totalSuggestions := len(suggestions)

	for _, suggestion := range suggestions {
		if f.matchesUserVoiceSearchBoolean(suggestion, request.Topic) {
			doc := f.uservoiceSuggestionToDoc(suggestion, sourceName)

			if sourceName == SourceFeatureRequests {
				docs = append(docs, doc)
			} else {
				accepted, reason := f.universalScorer.IsUniversallyAcceptableWithReason(doc.Content, doc.Title, sourceName, request.Topic)
				if accepted {
					docs = append(docs, doc)
				} else if f.pluginAPI != nil {
					f.pluginAPI.LogDebug(sourceName+": filtered out",
						"title", doc.Title,
						"id", suggestion.ID,
						"reason", reason)
				}
			}
		}
	}

	if f.pluginAPI != nil && request.Topic != "" {
		f.pluginAPI.LogDebug(sourceName+": search results", "total", totalSuggestions, "matched", len(docs))
	}

	return docs, nil
}

// matchesUserVoiceSearch checks if a UserVoice suggestion matches search terms
func (f *FileProtocol) matchesUserVoiceSearch(suggestion UserVoiceSuggestion, searchTerms []string) bool {
	if len(searchTerms) == 0 {
		return true
	}

	searchable := strings.ToLower(
		suggestion.Title + " " +
			suggestion.Description + " " +
			suggestion.Status + " " +
			suggestion.Category,
	)

	matchCount := 0
	for _, term := range searchTerms {
		if strings.Contains(searchable, term) {
			matchCount++
		}
	}

	return matchCount > 0
}

// matchesUserVoiceSearchBoolean checks if a UserVoice suggestion matches a boolean query
func (f *FileProtocol) matchesUserVoiceSearchBoolean(suggestion UserVoiceSuggestion, topic string) bool {
	if topic == "" {
		return true
	}

	queryNode, err := ParseBooleanQuery(topic)
	if err != nil {
		searchTerms := f.extractSearchTerms(topic)
		return f.matchesUserVoiceSearch(suggestion, searchTerms)
	}

	searchable := suggestion.Title + " " +
		suggestion.Description + " " +
		suggestion.Status + " " +
		suggestion.Category

	return EvaluateBoolean(queryNode, searchable)
}

// uservoiceSuggestionToDoc converts a UserVoice suggestion to a Doc with metadata extraction
func (f *FileProtocol) uservoiceSuggestionToDoc(suggestion UserVoiceSuggestion, sourceName string) Doc {
	meta := extractUserVoiceMetadata(suggestion)

	contentLines := []string{formatEntityMetadata(meta)}

	contentLines = append(contentLines, fmt.Sprintf("Title: %s", suggestion.Title))

	if suggestion.Description != "" {
		cleanedDesc := f.cleanUserVoiceDescription(suggestion.Description)
		if cleanedDesc != "" {
			contentLines = append(contentLines, fmt.Sprintf("\nDescription:\n%s", cleanedDesc))
		}
	}

	if suggestion.Status != "" {
		contentLines = append(contentLines, fmt.Sprintf("\nStatus: %s", suggestion.Status))
	}

	if suggestion.Votes > 0 {
		contentLines = append(contentLines, fmt.Sprintf("Votes: %d", suggestion.Votes))
	}

	if suggestion.Comments > 0 {
		contentLines = append(contentLines, fmt.Sprintf("Comments: %d", suggestion.Comments))
	}

	if suggestion.Category != "" {
		contentLines = append(contentLines, fmt.Sprintf("Category: %s", suggestion.Category))
	}

	if suggestion.CreatedAt != "" {
		contentLines = append(contentLines, fmt.Sprintf("Created: %s", suggestion.CreatedAt))
	}

	if suggestion.UpdatedAt != "" {
		contentLines = append(contentLines, fmt.Sprintf("Updated: %s", suggestion.UpdatedAt))
	}

	content := strings.Join(contentLines, "\n")

	labels := buildLabelsFromMetadata(meta)

	daysCreated := DaysSince(suggestion.CreatedAt)
	if recencyLabel := FormatRecencyLabel(daysCreated); recencyLabel != "" {
		labels = append(labels, recencyLabel+"_created")
	}
	daysUpdated := DaysSince(suggestion.UpdatedAt)
	if recencyLabel := FormatRecencyLabel(daysUpdated); recencyLabel != "" {
		labels = append(labels, recencyLabel+"_updated")
	}

	return Doc{
		Title:        suggestion.Title,
		Content:      content,
		URL:          suggestion.URL,
		Source:       sourceName,
		Section:      SectionFeatureRequests,
		Labels:       labels,
		Author:       suggestion.AuthorName,
		CreatedDate:  suggestion.CreatedAt,
		LastModified: suggestion.UpdatedAt,
	}
}

// cleanUserVoiceDescription removes HTML/JavaScript noise from UserVoice descriptions
func (f *FileProtocol) cleanUserVoiceDescription(desc string) string {
	if strings.Contains(desc, "var uvAuthElement") ||
		strings.Contains(desc, "Cookie access is needed") ||
		strings.Contains(desc, "We're glad you're here") {
		lines := strings.Split(desc, "\n")
		var cleanLines []string
		inNoise := false

		for _, line := range lines {
			trimmed := strings.TrimSpace(line)

			if trimmed == "" && !inNoise {
				continue
			}

			if strings.Contains(line, "We're glad you're here") ||
				strings.Contains(line, "var uvAuthElement") ||
				strings.Contains(line, "Cookie access") ||
				strings.Contains(line, "Sign in") ||
				strings.Contains(line, "uvAuthElement.subdomainSettings") {
				break
			}

			if !inNoise && trimmed != "" {
				cleanLines = append(cleanLines, trimmed)
			}
		}

		cleaned := strings.Join(cleanLines, "\n")

		if len(cleaned) < 50 {
			return ""
		}

		if len(cleaned) > 5000 {
			cleaned = cleaned[:5000] + "..."
		}

		return cleaned
	}

	if len(desc) > 5000 {
		return desc[:5000] + "..."
	}

	return desc
}

// mapUserVoiceStatusToPriority maps UserVoice status to priority
func mapUserVoiceStatusToPriority(status string) pm.Priority {
	status = strings.ToLower(status)
	switch status {
	case StatusCompleted, StatusShipped:
		return pm.PriorityCompleted
	case StatusInProgress, StatusPlanned:
		return pm.PriorityHigh
	case "under_review":
		return pm.PriorityMedium
	default:
		return pm.PriorityLow
	}
}

// mapUserVoiceVotesToPriority maps vote count to priority
func mapUserVoiceVotesToPriority(votes int) pm.Priority {
	if votes >= 100 {
		return pm.PriorityHigh
	} else if votes >= 20 {
		return pm.PriorityMedium
	}
	return pm.PriorityLow
}
