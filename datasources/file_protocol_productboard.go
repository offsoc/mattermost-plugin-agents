// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"fmt"
	"strings"
)

// ProductBoardFeature represents a feature from ProductBoard
type ProductBoardFeature struct {
	Name             string `json:"name"`
	Description      string `json:"description"`
	Type             string `json:"type"`
	State            string `json:"state"`
	Owner            string `json:"owner"`
	Parent           string `json:"parent"`
	Status           string `json:"status"`
	Score            string `json:"score"`
	Source           string `json:"source"`
	CustomerRequests string `json:"customer_requests"`
	BusinessPriority string `json:"business_priority"`
}

// fetchFromProductBoardJSON processes ProductBoard features from JSON
func (f *FileProtocol) fetchFromProductBoardJSON(features []ProductBoardFeature, sourceName string, request ProtocolRequest) ([]Doc, error) {
	var docs []Doc
	totalFeatures := len(features)

	if f.pluginAPI != nil {
		f.pluginAPI.LogDebug(sourceName+": starting search",
			"total_features", totalFeatures,
			"limit", request.Limit)
	}

	matchedCount := 0
	filteredCount := 0

	for i, feature := range features {
		matches := f.matchesSearchBoolean(feature, request.Topic)

		if f.pluginAPI != nil && i < 5 {
			f.pluginAPI.LogDebug(sourceName+": feature match check",
				"index", i,
				"name", feature.Name,
				"matches", matches)
		}

		if matches {
			matchedCount++
			doc := f.featureToDoc(feature, sourceName)

			if sourceName == SourceProductBoardFeatures {
				docs = append(docs, doc)
				if f.pluginAPI != nil && len(docs) <= 3 {
					f.pluginAPI.LogDebug(sourceName+": doc accepted",
						"title", doc.Title,
						"count", len(docs))
				}
			} else {
				accepted, reason := f.universalScorer.IsUniversallyAcceptableWithReason(doc.Content, doc.Title, sourceName, request.Topic)
				if accepted {
					docs = append(docs, doc)
				} else {
					filteredCount++
					if f.pluginAPI != nil {
						f.pluginAPI.LogDebug(sourceName+": filtered out",
							"title", doc.Title,
							"feature", feature.Name,
							"reason", reason)
					}
				}
			}
		}
	}

	if f.pluginAPI != nil {
		f.pluginAPI.LogDebug(sourceName+": search complete",
			"total_features", totalFeatures,
			"matched_query", matchedCount,
			"filtered_out", filteredCount,
			"final_results", len(docs))
	}

	return docs, nil
}

// matchesSearch checks if a feature matches search terms
func (f *FileProtocol) matchesSearch(feature ProductBoardFeature, searchTerms []string) bool {
	if len(searchTerms) == 0 {
		return true
	}

	searchable := strings.ToLower(
		feature.Name + " " +
			feature.Description + " " +
			feature.Type + " " +
			feature.State + " " +
			feature.Parent + " " +
			feature.CustomerRequests,
	)

	matchCount := 0
	for _, term := range searchTerms {
		if strings.Contains(searchable, term) {
			matchCount++
		}
	}

	threshold := len(searchTerms) / 2
	if threshold < 1 {
		threshold = 1
	}

	return matchCount >= threshold
}

// matchesSearchBoolean checks if a feature matches a boolean query
func (f *FileProtocol) matchesSearchBoolean(feature ProductBoardFeature, topic string) bool {
	if topic == "" {
		return true
	}

	queryNode, err := ParseBooleanQuery(topic)
	if err != nil {
		searchTerms := f.extractSearchTerms(topic)
		return f.matchesSearch(feature, searchTerms)
	}

	searchable := feature.Name + " " +
		feature.Description + " " +
		feature.Type + " " +
		feature.State + " " +
		feature.Parent + " " +
		feature.CustomerRequests

	return EvaluateBoolean(queryNode, searchable)
}

// featureToDoc converts a ProductBoard feature to a Doc, extracting status, priority, business priority,
// customer requests, revenue impact, parent features, and owner from ProductBoard's structured data
func (f *FileProtocol) featureToDoc(feature ProductBoardFeature, sourceName string) Doc {
	meta := extractProductBoardMetadata(feature)

	content := feature.Description
	if feature.Parent != "" {
		content = fmt.Sprintf("Parent: %s\n\n%s", feature.Parent, content)
	}
	if feature.CustomerRequests != "" {
		content = fmt.Sprintf("%s\n\nCustomer Requests: %s", content, feature.CustomerRequests)
	}

	content = formatEntityMetadata(meta) + content

	section := SectionFeatures
	switch {
	case feature.State == "Delivered":
		section = SectionDelivered
	case feature.State == "Idea":
		section = SectionIdeas
	case strings.Contains(strings.ToLower(feature.State), "progress"):
		section = SectionInProgress
	case strings.Contains(strings.ToLower(feature.State), "development"):
		section = SectionInProgress
	case feature.State == "Planned":
		section = SectionInProgress
	}

	url := fmt.Sprintf("productboard://feature/%s", strings.ReplaceAll(strings.ToLower(feature.Name), " ", "-"))

	labels := []string{feature.Type, feature.State}

	if feature.Parent != "" {
		labels = append(labels, fmt.Sprintf("parent:%s", feature.Parent))
	}

	labels = append(labels, buildLabelsFromMetadata(meta)...)

	if feature.Owner != "" {
		labels = append(labels, fmt.Sprintf("owner:%s", feature.Owner))
	}

	return Doc{
		Title:        feature.Name,
		Content:      content,
		URL:          url,
		Section:      section,
		Source:       sourceName,
		Author:       feature.Owner,
		LastModified: feature.State,
		Labels:       labels,
	}
}
