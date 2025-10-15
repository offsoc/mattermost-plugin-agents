// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJiraProtocol_escapeJQLString(t *testing.T) {
	j := &JiraProtocol{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "simple alphanumeric",
			input:    "hello123",
			expected: "hello123",
		},
		{
			name:     "backslash escaping",
			input:    `test\value`,
			expected: `test\\value`,
		},
		{
			name:     "double quote escaping",
			input:    `test"value`,
			expected: `test\"value`,
		},
		{
			name:     "plus sign escaping",
			input:    "C++",
			expected: `C\+\+`,
		},
		{
			name:     "minus sign escaping",
			input:    "test-value",
			expected: `test\-value`,
		},
		{
			name:     "ampersand escaping",
			input:    "test&value",
			expected: `test\&value`,
		},
		{
			name:     "pipe escaping",
			input:    "test|value",
			expected: `test\|value`,
		},
		{
			name:     "exclamation mark escaping",
			input:    "test!",
			expected: `test\!`,
		},
		{
			name:     "parentheses escaping",
			input:    "test(value)",
			expected: `test\(value\)`,
		},
		{
			name:     "curly braces escaping",
			input:    "test{value}",
			expected: `test\{value\}`,
		},
		{
			name:     "square brackets escaping",
			input:    "test[value]",
			expected: `test\[value\]`,
		},
		{
			name:     "caret escaping",
			input:    "test^value",
			expected: `test\^value`,
		},
		{
			name:     "tilde escaping",
			input:    "test~value",
			expected: `test\~value`,
		},
		{
			name:     "asterisk escaping",
			input:    "test*value",
			expected: `test\*value`,
		},
		{
			name:     "question mark escaping",
			input:    "test?value",
			expected: `test\?value`,
		},
		{
			name:     "colon escaping",
			input:    "test:value",
			expected: `test\:value`,
		},
		{
			name:     "forward slash escaping",
			input:    "test/value",
			expected: `test\/value`,
		},
		{
			name:     "multiple special characters",
			input:    `test"value"&other|thing`,
			expected: `test\"value\"\&other\|thing`,
		},
		{
			name:     "all JQL special characters",
			input:    `\"+&-|!(){}[]^~*?:/`,
			expected: `\\\"\+\&\-\|\!\(\)\{\}\[\]\^\~\*\?\:\/`,
		},
		{
			name:     "injection attempt with OR",
			input:    `test" OR project = "SECRET`,
			expected: `test\" OR project = \"SECRET`,
		},
		{
			name:     "injection attempt with AND",
			input:    `test" AND assignee = currentUser()`,
			expected: `test\" AND assignee = currentUser\(\)`,
		},
		{
			name:     "nested quotes",
			input:    `"test"`,
			expected: `\"test\"`,
		},
		{
			name:     "backslash followed by quote",
			input:    `test\"value`,
			expected: `test\\\"value`,
		},
		{
			name:     "realistic search term",
			input:    "mobile app",
			expected: "mobile app",
		},
		{
			name:     "realistic search with version",
			input:    "v1.2.3-beta",
			expected: `v1.2.3\-beta`,
		},
		{
			name:     "URL-like input",
			input:    "https://example.com/api",
			expected: `https\:\/\/example.com\/api`,
		},
		{
			name:     "email-like input",
			input:    "user@example.com",
			expected: `user@example.com`,
		},
		{
			name:     "SQL injection attempt - special chars escaped",
			input:    "'; DROP TABLE issues; --",
			expected: `'; DROP TABLE issues; \-\-`,
		},
		{
			name:     "path traversal attempt",
			input:    "../../etc/passwd",
			expected: `..\/..\/etc\/passwd`,
		},
		{
			name:     "regex metacharacters",
			input:    "test.*value+",
			expected: `test.\*value\+`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := j.escapeJQLString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestJiraProtocol_buildJQLQuery_NoInjection(t *testing.T) {
	j := &JiraProtocol{
		topicAnalyzer: NewTopicAnalyzer(),
	}

	tests := []struct {
		name             string
		topic            string
		sections         []string
		shouldContain    []string
		shouldNotContain []string
	}{
		{
			name:          "simple topic gets escaped",
			topic:         "test-value",
			sections:      []string{},
			shouldContain: []string{`test\-value`},
		},
		{
			name:             "injection attempt is neutralized",
			topic:            `test" OR project = "ADMIN`,
			sections:         []string{},
			shouldContain:    []string{`test\"`},
			shouldNotContain: []string{`OR project = "ADMIN"`},
		},
		{
			name:          "special characters are escaped in summary search",
			topic:         "C++",
			sections:      []string{},
			shouldContain: []string{`C\+\+`},
		},
		{
			name:          "multiple keywords all escaped",
			topic:         "test-value & other|thing",
			sections:      []string{},
			shouldContain: []string{`test\-value`, `\&`, `other\|thing`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := j.buildJQLQuery(tt.topic, tt.sections)

			for _, expected := range tt.shouldContain {
				assert.Contains(t, result, expected, "JQL should contain escaped value")
			}

			for _, notExpected := range tt.shouldNotContain {
				assert.NotContains(t, result, notExpected, "JQL should not contain unescaped injection attempt")
			}

			assert.Contains(t, result, "ORDER BY updated DESC", "JQL should contain ORDER BY clause")
		})
	}
}

func TestJiraProtocol_buildJQLQuery_BasicStructure(t *testing.T) {
	j := &JiraProtocol{
		topicAnalyzer: NewTopicAnalyzer(),
	}

	tests := []struct {
		name          string
		topic         string
		sections      []string
		expectedParts []string
	}{
		{
			name:     "topic with no sections",
			topic:    "mobile",
			sections: []string{},
			expectedParts: []string{
				"text ~",
				"summary ~",
				"ORDER BY updated DESC",
			},
		},
		{
			name:     "topic with sections",
			topic:    "mobile",
			sections: []string{"bug", "feature"},
			expectedParts: []string{
				"issueType in",
				"Bug",
				"Feature",
				"ORDER BY updated DESC",
			},
		},
		{
			name:     "no topic defaults to recent updates",
			topic:    "",
			sections: []string{},
			expectedParts: []string{
				"updated >= -30d",
				"ORDER BY updated DESC",
			},
		},
		{
			name:     "comma-separated topic creates multiple conditions",
			topic:    "mobile,desktop",
			sections: []string{},
			expectedParts: []string{
				"summary ~",
				"description ~",
				"mobile",
				"desktop",
				"ORDER BY updated DESC",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := j.buildJQLQuery(tt.topic, tt.sections)

			for _, part := range tt.expectedParts {
				assert.Contains(t, result, part, "JQL should contain expected part")
			}
		})
	}
}

func TestJiraProtocol_escapeJQLString_PreservesBackslashOrder(t *testing.T) {
	j := &JiraProtocol{}

	input := `test\value"other`
	result := j.escapeJQLString(input)

	assert.Equal(t, `test\\value\"other`, result, "Backslash must be escaped before quotes to prevent double-escaping")
	assert.NotEqual(t, `test\\"value"other`, result, "Should not have improperly escaped backslash")
}

func TestJiraProtocol_escapeJQLString_SecurityRegressionTest(t *testing.T) {
	j := &JiraProtocol{}

	injectionAttempts := []struct {
		name  string
		input string
	}{
		{
			name:  "OR injection",
			input: `" OR 1=1 --`,
		},
		{
			name:  "AND injection",
			input: `" AND project = "ADMIN`,
		},
		{
			name:  "function injection",
			input: `") OR currentUser() = "admin`,
		},
		{
			name:  "nested quotes",
			input: `"""OR"""`,
		},
		{
			name:  "comment injection",
			input: `test /* comment */ OR true`,
		},
	}

	for _, tt := range injectionAttempts {
		t.Run(tt.name, func(t *testing.T) {
			result := j.escapeJQLString(tt.input)

			quotesEscaped := true
			unescapedQuoteCount := 0
			for i := 0; i < len(result); i++ {
				if result[i] == '"' {
					if i == 0 || result[i-1] != '\\' {
						quotesEscaped = false
						unescapedQuoteCount++
					}
				}
			}
			assert.True(t, quotesEscaped, "All quotes should be escaped")
			assert.Equal(t, 0, unescapedQuoteCount, "Should have zero unescaped quotes")

			parensEscaped := true
			for i := 0; i < len(result); i++ {
				if (result[i] == '(' || result[i] == ')') && (i == 0 || result[i-1] != '\\') {
					parensEscaped = false
					break
				}
			}
			assert.True(t, parensEscaped, "All parentheses should be escaped")
		})
	}
}
