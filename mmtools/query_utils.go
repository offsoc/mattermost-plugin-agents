// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package mmtools

import (
	"strings"

	"github.com/mattermost/mattermost-plugin-ai/datasources"
	"github.com/mattermost/mattermost-plugin-ai/mmtools/querybuilder"
)

// ExtractFeaturesFromTopic detects Mattermost features mentioned in a topic string
func ExtractFeaturesFromTopic(topic string) []string {
	analyzer := datasources.NewTopicAnalyzer()
	return analyzer.GetMattermostFeatures(topic)
}

// ExpandFeatureSynonyms expands canonical feature names to all their synonyms
func ExpandFeatureSynonyms(features []string) []string {
	analyzer := datasources.NewTopicAnalyzer()
	return analyzer.ExpandMattermostFeatureSynonyms(features)
}

// BuildBooleanQuery creates a composite query string from intent and feature keywords
// Delegates to the shared querybuilder package to avoid code duplication
func BuildBooleanQuery(intentKeywords, featureKeywords []string) string {
	return querybuilder.BuildBooleanQuery(intentKeywords, featureKeywords)
}

// IsTopicRelevantForTool checks if a topic is relevant for a specific tool using new architecture
func (p *MMToolProvider) IsTopicRelevantForTool(toolName, topic string) bool {
	if topic == "" {
		return false
	}

	mentionedFeatures := ExtractFeaturesFromTopic(sanitizeQueryTopic(topic))

	// Tool is relevant if ANY features are detected
	// (Tool intent is implicit when tool is invoked)
	return len(mentionedFeatures) > 0
}

// BuildCompositeQuery creates a search query combining tool intent with detected features
func (p *MMToolProvider) BuildCompositeQuery(toolName, topic, dataSource string) string {
	metadata, exists := p.GetToolMetadata(toolName)
	if !exists {
		return ""
	}

	intentKeywords := metadata.IntentKeywords
	if len(intentKeywords) == 0 {
		return ""
	}

	// Preprocess topic to improve feature detection and downstream keyword extraction
	// Notably, LLMs may pass normalized feature tokens like "ai_channels" which
	// won't match the taxonomy unless we replace underscores with spaces.
	cleanedTopic := sanitizeQueryTopic(topic)

	features := ExtractFeaturesFromTopic(cleanedTopic)

	analyzer := datasources.NewTopicAnalyzer()

	// If we detected features, expand them to include synonyms
	if len(features) > 0 {
		featureKeywords := ExpandFeatureSynonyms(features)
		return BuildBooleanQuery(intentKeywords, featureKeywords)
	}

	// Fallback: build a broader query using extracted topic keywords when
	// no explicit Mattermost features are detected. This ensures we still
	// fetch relevant external docs for queries like "product vision for channels with AI".
	topicKeywords := analyzer.ExtractTopicKeywords(cleanedTopic)
	if len(topicKeywords) > 0 {
		return BuildBooleanQuery(intentKeywords, topicKeywords)
	}

	// Final fallback: return the cleaned topic so protocols can apply their
	// own relevance scoring. Avoid returning empty which halts fetching.
	return cleanedTopic
}

// sanitizeQueryTopic normalizes a topic string for feature detection and search
// - replaces underscores with spaces (e.g., "ai_channels" -> "ai channels")
// - collapses repeated whitespace
func sanitizeQueryTopic(topic string) string {
	if topic == "" {
		return topic
	}
	// Replace underscores which often come from NormalizeFeatureName
	t := strings.ReplaceAll(topic, "_", " ")
	t = strings.Join(strings.Fields(t), " ")
	return t
}

// splitDirectiveTopic splits a topic at directive boundaries
// Returns the features part and context part separately
// Example: "ai channels Focus on secure deployment" -> ("ai channels", "secure deployment")
func splitDirectiveTopic(topic string) (features string, context string) {
	// Directive patterns that split the topic
	directives := []string{
		"Focus on", "focus on",
		"Analyze alignment with", "analyze alignment with",
		"Analyze", "analyze",
		"Consider", "consider",
		"Evaluate", "evaluate",
		"Assess", "assess",
		"Compare with", "compare with",
		"Compare", "compare",
	}

	topicLower := strings.ToLower(topic)
	splitIndex := -1
	directiveLen := 0

	// Find the first directive in the topic
	for _, directive := range directives {
		if idx := strings.Index(topicLower, strings.ToLower(directive)); idx != -1 {
			if splitIndex == -1 || idx < splitIndex {
				splitIndex = idx
				directiveLen = len(directive)
			}
		}
	}

	// If no directive found, return the whole topic as features
	if splitIndex == -1 {
		return topic, ""
	}

	// Split at the directive boundary
	features = strings.TrimSpace(topic[:splitIndex])
	context = strings.TrimSpace(topic[splitIndex+directiveLen:])

	return features, context
}

// BuildSearchQueries builds search queries for external docs
func (p *MMToolProvider) BuildSearchQueries(toolName, topic string) map[string]string {
	queries := make(map[string]string)

	// Skip empty topics entirely - no point searching with empty string
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return queries
	}

	metadata, exists := p.GetToolMetadata(toolName)
	if !exists {
		return queries
	}
	intentKeywords := metadata.IntentKeywords

	featuresPart, contextPart := splitDirectiveTopic(topic)

	features := ExtractFeaturesFromTopic(sanitizeQueryTopic(featuresPart))
	featureKeywords := ExpandFeatureSynonyms(features)

	// If there's context after the directive, extract keywords from it too
	if contextPart != "" {
		contextFeatures := ExtractFeaturesFromTopic(sanitizeQueryTopic(contextPart))
		contextKeywords := ExpandFeatureSynonyms(contextFeatures)

		// Combine all keywords
		featureKeywords = append(featureKeywords, contextKeywords...)

		// Also extract any non-feature keywords from context (like "vision", "mission", etc.)
		analyzer := datasources.NewTopicAnalyzer()
		additionalKeywords := analyzer.ExtractTopicKeywords(contextPart)
		featureKeywords = append(featureKeywords, additionalKeywords...)
	}

	// Generate a single combined query with all keywords
	booleanQuery := BuildBooleanQuery(intentKeywords, featureKeywords)

	// Only add the query if it's not empty
	if booleanQuery != "" {
		queries["search"] = booleanQuery
	}

	return queries
}
