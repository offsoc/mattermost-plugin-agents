// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package llm

import (
	"regexp"
	"strings"
)

// Artifact marker patterns:
// ```artifact:javascript title="Calculator App"
// ```artifact:python
// ```artifact:react title="Todo Component"
// ```artifact:svg title="Architecture Diagram"
// ```artifact:mermaid

var (
	// Match code blocks with artifact prefix
	// Pattern: ```artifact:language title="optional title"
	// Uses (?sm) for multiline mode where ^ matches line start
	// Matches until ``` at the beginning of a line to handle nested backticks in content
	artifactPattern = regexp.MustCompile(`(?sm)` + "`" + "`" + "`" + `artifact:([a-zA-Z0-9]+)(?:\s+title="([^"]+)")?\s*\n(.*?)\n^` + "`" + "`" + "`")
)

// DetectArtifacts scans text for artifact markers and extracts them
func DetectArtifacts(text string) []Artifact {
	matches := artifactPattern.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return nil
	}

	artifacts := make([]Artifact, 0, len(matches))
	for _, match := range matches {
		if len(match) < 4 {
			continue
		}

		language := match[1]
		title := match[2]
		content := strings.TrimSpace(match[3])

		// Skip empty content
		if content == "" {
			continue
		}

		// Default title if not provided
		if title == "" {
			title = "Untitled Artifact"
		}

		// Determine artifact type based on language
		artifactType := determineArtifactType(language)

		artifacts = append(artifacts, Artifact{
			Type:     artifactType,
			Title:    title,
			Content:  content,
			Language: language,
		})
	}

	return artifacts
}

// RemoveArtifactMarkers removes artifact code blocks from text, leaving only the surrounding conversation
func RemoveArtifactMarkers(text string) string {
	// Remove artifact blocks and clean up extra newlines
	result := artifactPattern.ReplaceAllString(text, "")
	// Clean up multiple consecutive newlines left behind
	result = regexp.MustCompile(`\n\n\n+`).ReplaceAllString(result, "\n\n")
	return strings.TrimSpace(result)
}

// determineArtifactType infers the artifact type from the language
func determineArtifactType(language string) ArtifactType {
	language = strings.ToLower(language)

	// Diagram types
	if language == "mermaid" || language == "svg" || language == "dot" || language == "plantuml" {
		return ArtifactTypeDiagram
	}

	// Document types
	if language == "markdown" || language == "md" || language == "text" || language == "txt" || language == "html" {
		return ArtifactTypeDocument
	}

	// Everything else is code
	return ArtifactTypeCode
}
