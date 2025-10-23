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
	// Match the opening of an artifact block
	artifactOpenPattern = regexp.MustCompile(`^` + "`" + "`" + "`" + `artifact:([a-zA-Z0-9]+)(?:\s+title="([^"]+)")?`)
	// Match any code fence (three backticks at start of line)
	codeFencePattern = regexp.MustCompile(`^` + "`" + "`" + "`")
)

// DetectArtifacts scans text for artifact markers and extracts them
// Properly handles nested code blocks by tracking fence depth
func DetectArtifacts(text string) []Artifact {
	lines := strings.Split(text, "\n")
	var artifacts []Artifact

	i := 0
	for i < len(lines) {
		line := lines[i]

		// Check if this line starts an artifact block
		if match := artifactOpenPattern.FindStringSubmatch(line); match != nil {
			language := match[1]
			title := match[2]
			if title == "" {
				title = "Untitled Artifact"
			}

			// Collect content until we find the closing fence
			// Track nested code fences to handle nested code blocks
			contentLines := []string{}
			i++
			nestLevel := 0

		contentLoop:
			for i < len(lines) {
				contentLine := lines[i]

				// Check if this is a code fence
				if codeFencePattern.MatchString(contentLine) {
					// Check if it's just ``` (a nested code block fence)
					trimmed := strings.TrimSpace(contentLine)
					switch {
					case trimmed == "```" && nestLevel == 0:
						// This is the closing fence for the artifact
						break contentLoop
					case trimmed == "```":
						// This closes a nested code block
						nestLevel--
						contentLines = append(contentLines, contentLine)
					case strings.HasPrefix(trimmed, "```") && !strings.HasPrefix(trimmed, "```artifact:"):
						// This opens a nested code block (e.g., ```javascript)
						nestLevel++
						contentLines = append(contentLines, contentLine)
					default:
						// Shouldn't happen, but treat as content
						contentLines = append(contentLines, contentLine)
					}
				} else {
					contentLines = append(contentLines, contentLine)
				}
				i++
			}

			content := strings.TrimSpace(strings.Join(contentLines, "\n"))

			// Skip empty content
			if content != "" {
				artifactType := determineArtifactType(language)
				artifacts = append(artifacts, Artifact{
					Type:     artifactType,
					Title:    title,
					Content:  content,
					Language: language,
				})
			}
		}
		i++
	}

	if len(artifacts) == 0 {
		return nil
	}

	return artifacts
}

// RemoveArtifactMarkers removes artifact code blocks from text, leaving only the surrounding conversation
func RemoveArtifactMarkers(text string) string {
	lines := strings.Split(text, "\n")
	var resultLines []string

	i := 0
	for i < len(lines) {
		line := lines[i]

		// Check if this line starts an artifact block
		if artifactOpenPattern.MatchString(line) {
			// Add a blank line to preserve spacing where artifact was removed
			resultLines = append(resultLines, "")

			// Skip everything until we find the closing fence
			i++
			nestLevel := 0

		removeLoop:
			for i < len(lines) {
				contentLine := lines[i]

				if codeFencePattern.MatchString(contentLine) {
					trimmed := strings.TrimSpace(contentLine)
					switch {
					case trimmed == "```" && nestLevel == 0:
						// Found the closing fence, skip it and break
						i++
						break removeLoop
					case trimmed == "```":
						nestLevel--
					case strings.HasPrefix(trimmed, "```") && !strings.HasPrefix(trimmed, "```artifact:"):
						nestLevel++
					}
				}
				i++
			}
			// Continue to next line (already incremented)
			continue
		}

		resultLines = append(resultLines, line)
		i++
	}

	result := strings.Join(resultLines, "\n")
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
