// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package intentutils

import (
	"regexp"
	"strings"
)

// KeywordPattern represents a keyword with different matching strategies
type KeywordPattern struct {
	Word         string  // Exact word match
	Phrase       string  // Multi-word phrase match
	Pattern      string  // Regex pattern
	Weight       float64 // Weight multiplier for scoring
	WordBoundary bool    // Require word boundaries
}

// MatchesPattern checks if a message matches any of the keyword patterns
func MatchesPattern(message string, patterns []KeywordPattern) (bool, float64) {
	msg := strings.ToLower(strings.TrimSpace(message))
	totalScore := 0.0
	maxScore := 0.0

	for _, pattern := range patterns {
		score := 0.0

		// Try different matching strategies
		switch {
		case pattern.Pattern != "":
			// Regex pattern matching
			if regexp.MustCompile(pattern.Pattern).MatchString(msg) {
				score = pattern.Weight
			}
		case pattern.Phrase != "":
			// Phrase matching with optional word boundaries
			phrase := strings.ToLower(pattern.Phrase)
			if pattern.WordBoundary {
				// Use word boundary regex for phrase
				boundaryPattern := `\b` + regexp.QuoteMeta(phrase) + `\b`
				if regexp.MustCompile(boundaryPattern).MatchString(msg) {
					score = pattern.Weight
				}
			} else if strings.Contains(msg, phrase) {
				score = pattern.Weight
			}
		case pattern.Word != "":
			// Single word matching with word boundaries
			word := strings.ToLower(pattern.Word)
			if pattern.WordBoundary {
				boundaryPattern := `\b` + regexp.QuoteMeta(word) + `\b`
				if regexp.MustCompile(boundaryPattern).MatchString(msg) {
					score = pattern.Weight
				}
			} else if strings.Contains(msg, word) {
				score = pattern.Weight
			}
		}

		totalScore += score
		if score > maxScore {
			maxScore = score
		}
	}

	return totalScore > 0, maxScore
}
