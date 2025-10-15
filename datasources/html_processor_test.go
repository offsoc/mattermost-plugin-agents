// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"strings"
	"testing"
)

func TestHTMLProcessor_StripHTMLTags(t *testing.T) {
	processor := NewHTMLProcessor()

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "No HTML",
			input:    "Plain text without any HTML",
			expected: "Plain text without any HTML",
		},
		{
			name:     "Simple tags",
			input:    "<p>Paragraph</p> with <span>span</span>",
			expected: "Paragraph with span",
		},
		{
			name:     "Tags with attributes",
			input:    `<div class="container"><p id="text">Content</p></div>`,
			expected: "Content",
		},
		{
			name:     "Paragraph breaks preserved",
			input:    "<p>First paragraph</p><p>Second paragraph</p>",
			expected: "First paragraph\n\nSecond paragraph",
		},
		{
			name:     "Line breaks",
			input:    "Line 1<br>Line 2<br/>Line 3<br />Line 4",
			expected: "Line 1\nLine 2\nLine 3\nLine 4",
		},
		{
			name:     "Common tags fast path",
			input:    "<div>Text in <span>span</span> element</div>",
			expected: "Text in span element",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := processor.stripHTMLTags(tc.input)
			result = normalizeWhitespaceForTest(result)
			expected := normalizeWhitespaceForTest(tc.expected)
			if result != expected {
				t.Errorf("Expected '%s', got '%s'", expected, result)
			}
		})
	}
}

func TestHTMLProcessor_StripHTMLTagsPreserveStructure(t *testing.T) {
	processor := NewHTMLProcessor()

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:  "Script and style removal",
			input: "<p>Content</p><script>alert('test')</script><style>.red{color:red}</style><p>More</p>",
			expected: `

Content

More`,
		},
		{
			name:  "Heading preservation",
			input: "<h1>Title</h1><p>Content</p><h2>Subtitle</h2>",
			expected: `# Title

Content## Subtitle`,
		},
		{
			name:  "List preservation",
			input: "<ul><li>Item 1</li><li>Item 2</li></ul>",
			expected: `
- Item 1
- Item 2`,
		},
		{
			name:  "Paragraph structure",
			input: "<p>First paragraph</p><div>Division</div><p>Second paragraph</p>",
			expected: `

First paragraph
Division

Second paragraph`,
		},
		{
			name:  "Complex structure with headings and lists",
			input: "<h2>Section</h2><p>Text</p><ul><li>Point 1</li><li>Point 2</li></ul>",
			expected: `## Section

Text - Point 1
- Point 2`,
		},
		{
			name:     "CSS pattern cleanup",
			input:    "<p>Text</p><div>--var: value; color: red;</div>",
			expected: "Text",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := processor.stripHTMLTagsPreserveStructure(tc.input)
			result = normalizeWhitespaceForTest(result)
			expected := normalizeWhitespaceForTest(tc.expected)
			if result != expected {
				t.Errorf("Expected:\n'%s'\n\nGot:\n'%s'", expected, result)
			}
		})
	}
}

func TestHTMLProcessor_FallbackTextExtraction(t *testing.T) {
	processor := NewHTMLProcessor()

	testCases := []struct {
		name        string
		input       string
		contains    []string
		notContains []string
	}{
		{
			name:        "Complete document",
			input:       "<html><head><script>code</script></head><body><h1>Title</h1><p>Content</p></body></html>",
			contains:    []string{"# Title", "Content"},
			notContains: []string{"script", "code"},
		},
		{
			name:        "Style pollution removed",
			input:       "<div><style>.test{color:red}</style><p>Clean content</p></div>",
			contains:    []string{"Clean content"},
			notContains: []string{"color", "red", "test"},
		},
		{
			name:     "Structure preserved",
			input:    "<h2>Section</h2><p>Paragraph 1</p><p>Paragraph 2</p>",
			contains: []string{"## Section", "Paragraph 1", "Paragraph 2"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := processor.fallbackTextExtraction(tc.input)
			result = strings.TrimSpace(result)

			for _, substr := range tc.contains {
				if !strings.Contains(result, substr) {
					t.Errorf("Expected result to contain '%s', but it didn't. Got: %s", substr, result)
				}
			}

			for _, substr := range tc.notContains {
				if strings.Contains(result, substr) {
					t.Errorf("Expected result NOT to contain '%s', but it did. Got: %s", substr, result)
				}
			}
		})
	}
}

func TestHTMLProcessor_RemoveInlineCSSPatterns(t *testing.T) {
	processor := NewHTMLProcessor()

	testCases := []struct {
		name             string
		input            string
		shouldNotContain []string
	}{
		{
			name:             "CSS variables",
			input:            "Text --var-name: value; more text",
			shouldNotContain: []string{"--var-name:", "value;"},
		},
		{
			name:             "CSS properties",
			input:            "Text color: red; font-size: 12px; more text",
			shouldNotContain: []string{"color:", "font-size:"},
		},
		{
			name:             "CSS blocks",
			input:            "Text .class { color: red; } more text",
			shouldNotContain: []string{"color: red;"},
		},
		{
			name:             "Mixed CSS patterns",
			input:            "Text --var: val; color: blue; { prop: val } more",
			shouldNotContain: []string{"--var:", "color:", "prop:"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := processor.removeInlineCSSPatterns(tc.input)
			result = strings.TrimSpace(result)

			for _, pattern := range tc.shouldNotContain {
				if strings.Contains(result, pattern) {
					t.Errorf("Result should not contain '%s', but got: %s", pattern, result)
				}
			}

			if !strings.Contains(result, "Text") || !strings.Contains(result, "more") {
				t.Errorf("Result should contain 'Text' and 'more', got: %s", result)
			}
		})
	}
}

// normalizeWhitespaceForTest normalizes whitespace for test comparison
func normalizeWhitespaceForTest(text string) string {
	// Replace multiple spaces with single space
	text = strings.TrimSpace(text)
	// Collapse multiple newlines
	for strings.Contains(text, "\n\n\n") {
		text = strings.ReplaceAll(text, "\n\n\n", "\n\n")
	}
	// Trim spaces at end of lines
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " ")
	}
	return strings.Join(lines, "\n")
}
