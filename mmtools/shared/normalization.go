// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package shared

import (
	"regexp"
	"sort"
	"strings"
)

// NormalizeFeatureName normalizes feature names for consistent processing
// This is extracted from mmtools.NormalizeFeatureName to avoid circular dependencies
// It performs the following normalizations:
//   - lowercases the input
//   - prefers the longest quoted phrase if present
//   - strips quotes and non-alphanumeric characters (keeps spaces)
//   - collapses whitespace
//   - removes common stopwords
//   - applies lightweight synonym collapsing (e.g., artificial intelligence -> ai)
//   - sorts remaining tokens to make the key order-invariant
func NormalizeFeatureName(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}

	// Prefer longest quoted phrase (single or double quotes)
	quoted := longestQuotedPhrase(s)
	if quoted != "" {
		s = quoted
	}

	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "\"", "")
	s = strings.ReplaceAll(s, "'", "")

	nonAlphaNum := regexp.MustCompile(`[^a-z0-9]+`)
	s = nonAlphaNum.ReplaceAllString(s, " ")

	s = strings.Join(strings.Fields(s), " ")

	if s == "" {
		return s
	}

	stopwords := map[string]struct{}{
		"the": {}, "a": {}, "an": {}, "and": {}, "or": {}, "with": {}, "of": {}, "for": {},
		"to": {}, "on": {}, "in": {}, "by": {}, "about": {}, "feature": {}, "goal": {},
		"features": {}, "topic": {}, "request": {}, "requests": {},
	}

	synonyms := map[string]string{
		"artificial":              "ai",
		"intelligence":            "ai",
		"artificial intelligence": "ai",
		"assistant":               "ai",
		"copilot":                 "ai",
		"agent":                   "ai",
		"agents":                  "ai",
		"channel":                 "channels",
	}

	tokens := strings.Fields(s)
	norm := make([]string, 0, len(tokens))
	for _, t := range tokens {
		if _, skip := stopwords[t]; skip {
			continue
		}
		if mapped, ok := synonyms[t]; ok {
			t = mapped
		}
		if t != "" {
			norm = append(norm, t)
		}
	}
	if len(norm) == 0 {
		return ""
	}
	seen := map[string]struct{}{}
	uniq := make([]string, 0, len(norm))
	for _, t := range norm {
		if _, ok := seen[t]; !ok {
			seen[t] = struct{}{}
			uniq = append(uniq, t)
		}
	}
	sort.Strings(uniq)
	return strings.Join(uniq, "_")
}

// longestQuotedPhrase returns the longest phrase found within single or double quotes.
func longestQuotedPhrase(s string) string {
	rx := regexp.MustCompile(`['"]([^'"]{3,})['"]`)
	matches := rx.FindAllStringSubmatch(s, -1)
	longest := ""
	for _, m := range matches {
		if len(m) > 1 && len(m[1]) > len(longest) {
			longest = m[1]
		}
	}
	return strings.TrimSpace(longest)
}
