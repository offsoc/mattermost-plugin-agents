// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package testutils

import "strings"

// TruncateString truncates a string to the specified maximum length.
// If the string is longer than maxLen, it appends "..." to indicate truncation.
//
// Example:
//
//	TruncateString("This is a long string", 10) // Returns: "This is..."
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen < 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// TruncateWithKeywords truncates a string, but for boolean queries it extracts
// the first N keywords instead of just truncating at maxLen.
// This provides better readability for complex query strings in test output.
//
// Example:
//
//	TruncateWithKeywords("(mobile OR ios) AND (bug OR issue)", 50, true)
//	// Returns: "mobile, ios, bug, issue..."
func TruncateWithKeywords(s string, maxLen int, extractKeywords bool) string {
	if len(s) <= maxLen {
		return s
	}

	// If keyword extraction is enabled and this looks like a boolean query
	if extractKeywords && (strings.Contains(s, " OR ") || strings.Contains(s, " AND ") ||
		strings.Contains(s, " or ") || strings.Contains(s, " and ")) {
		keywords := extractFirstKeywords(s, 5)
		if len(keywords) > 0 {
			return strings.Join(keywords, ", ") + "..."
		}
	}

	// Otherwise just truncate
	return TruncateString(s, maxLen)
}

// extractFirstKeywords extracts the first N meaningful keywords from a query string
func extractFirstKeywords(query string, n int) []string {
	// Split by common delimiters
	words := strings.FieldsFunc(query, func(r rune) bool {
		return r == '(' || r == ')' || r == '"' || r == ' '
	})

	var keywords []string
	seen := make(map[string]bool)

	for _, word := range words {
		word = strings.TrimSpace(strings.ToLower(word))
		// Skip operators and empty words
		if word == "" || word == "and" || word == "or" || word == "not" {
			continue
		}
		// Skip duplicates
		if seen[word] {
			continue
		}
		seen[word] = true
		keywords = append(keywords, word)
		if len(keywords) >= n {
			break
		}
	}

	return keywords
}
