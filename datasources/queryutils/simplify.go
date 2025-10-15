// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package queryutils

import "strings"

// SimplifyQueryToKeywords extracts meaningful keywords from a complex query string.
// Removes boolean operators (AND, OR, NOT), punctuation, and limits to maxTerms.
//
// Parameters:
//   - query: The original search query
//   - maxTerms: Maximum number of terms to return (0 means no limit)
//   - fallback: Default value to return if no valid terms are found
//
// Returns: Simplified query string with space-separated keywords
//
// Example:
//
//	SimplifyQueryToKeywords("(mobile OR web) AND bug", 3, "mobile")
//	// Returns: "mobile web bug"
func SimplifyQueryToKeywords(query string, maxTerms int, fallback string) string {
	words := strings.Fields(strings.ToLower(query))
	var terms []string

	for _, word := range words {
		word = strings.Trim(word, "()\"'")
		// Skip boolean operators and short words
		if word != "and" && word != "or" && word != "not" && len(word) > 2 {
			terms = append(terms, word)
		}
	}

	if maxTerms > 0 && len(terms) > maxTerms {
		terms = terms[:maxTerms]
	}

	if len(terms) == 0 {
		return fallback
	}

	return strings.Join(terms, " ")
}
