// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package llm

import (
	"strings"
	"testing"
)

func TestDetectArtifacts(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Artifact
	}{
		{
			name: "single javascript artifact with title",
			input: `Here's a calculator:

` + "```artifact:javascript title=\"Simple Calculator\"" + `
function add(a, b) {
  return a + b;
}
` + "```" + `

That should work!`,
			expected: []Artifact{
				{
					Type:     ArtifactTypeCode,
					Title:    "Simple Calculator",
					Content:  "function add(a, b) {\n  return a + b;\n}",
					Language: "javascript",
				},
			},
		},
		{
			name: "artifact without title gets default",
			input: "```artifact:py\n" +
				"print('hello')\n" +
				"```",
			expected: []Artifact{
				{
					Type:     ArtifactTypeCode,
					Title:    "Untitled Artifact",
					Content:  "print('hello')",
					Language: "py",
				},
			},
		},
		{
			name: "mermaid diagram artifact",
			input: "```artifact:mermaid title=\"System Flow\"\n" +
				"graph TD\n" +
				"  A-->B\n" +
				"```",
			expected: []Artifact{
				{
					Type:     ArtifactTypeDiagram,
					Title:    "System Flow",
					Content:  "graph TD\n  A-->B",
					Language: "mermaid",
				},
			},
		},
		{
			name: "multiple artifacts",
			input: "```artifact:javascript title=\"App\"\n" +
				"console.log('app');\n" +
				"```\n" +
				"Some text in between\n" +
				"```artifact:html title=\"Page\"\n" +
				"<div>test</div>\n" +
				"```",
			expected: []Artifact{
				{
					Type:     ArtifactTypeCode,
					Title:    "App",
					Content:  "console.log('app');",
					Language: "javascript",
				},
				{
					Type:     ArtifactTypeDocument,
					Title:    "Page",
					Content:  "<div>test</div>",
					Language: "html",
				},
			},
		},
		{
			name:     "no artifacts",
			input:    "Just regular code:\n```javascript\nconsole.log('hi');\n```",
			expected: nil,
		},
		{
			name:     "empty artifact content ignored",
			input:    "```artifact:javascript\n\n```",
			expected: nil,
		},
		{
			name: "artifact with nested code blocks",
			input: "```artifact:js title=\"Code with Examples\"\n" +
				"function example() {\n" +
				"  // Example usage:\n" +
				"  ```\n" +
				"  example()\n" +
				"  ```\n" +
				"}\n" +
				"```",
			expected: []Artifact{
				{
					Type:     ArtifactTypeCode,
					Title:    "Code with Examples",
					Content:  "function example() {\n  // Example usage:\n  ```\n  example()\n  ```\n}",
					Language: "js",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectArtifacts(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d artifacts, got %d", len(tt.expected), len(result))
				return
			}

			for i, expected := range tt.expected {
				if result[i].Type != expected.Type {
					t.Errorf("artifact %d: expected type %s, got %s", i, expected.Type, result[i].Type)
				}
				if result[i].Title != expected.Title {
					t.Errorf("artifact %d: expected title %q, got %q", i, expected.Title, result[i].Title)
				}
				if result[i].Content != expected.Content {
					t.Errorf("artifact %d: expected content %q, got %q", i, expected.Content, result[i].Content)
				}
				if result[i].Language != expected.Language {
					t.Errorf("artifact %d: expected language %s, got %s", i, expected.Language, result[i].Language)
				}
			}
		})
	}
}

func TestDetermineArtifactType(t *testing.T) {
	tests := []struct {
		language string
		expected ArtifactType
	}{
		{"javascript", ArtifactTypeCode},
		{"python", ArtifactTypeCode},
		{"go", ArtifactTypeCode},
		{"mermaid", ArtifactTypeDiagram},
		{"svg", ArtifactTypeDiagram},
		{"plantuml", ArtifactTypeDiagram},
		{"markdown", ArtifactTypeDocument},
		{"html", ArtifactTypeDocument},
		{"text", ArtifactTypeDocument},
	}

	for _, tt := range tests {
		t.Run(tt.language, func(t *testing.T) {
			result := determineArtifactType(tt.language)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestRemoveArtifactMarkers(t *testing.T) {
	input := "Here's some code:\n" +
		"```artifact:javascript title=\"Test\"\n" +
		"console.log('hi');\n" +
		"```\n" +
		"And that's it!"

	expected := "Here's some code:\n\nAnd that's it!"

	result := RemoveArtifactMarkers(input)
	// TrimSpace is now applied in RemoveArtifactMarkers, so we compare trimmed
	if strings.TrimSpace(result) != strings.TrimSpace(expected) {
		t.Errorf("expected %q, got %q", expected, result)
	}
}
