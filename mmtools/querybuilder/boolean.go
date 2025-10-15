// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package querybuilder

import (
	"strings"
)

// BuildBooleanQuery creates a composite query string from intent and feature keywords
// Properly quotes multi-word keywords and joins them with OR operators within parentheses
// Example: BuildBooleanQuery(["api", "example"], ["mobile", "notifications"])
// Returns: "(api OR example) AND (mobile OR notifications)"
func BuildBooleanQuery(intentKeywords, featureKeywords []string) string {
	if len(intentKeywords) == 0 {
		return ""
	}

	// Quote multi-word keywords (contains spaces or hyphens) for exact phrase matching
	quotedIntent := make([]string, len(intentKeywords))
	for i, keyword := range intentKeywords {
		if strings.Contains(keyword, " ") || strings.Contains(keyword, "-") {
			quotedIntent[i] = "\"" + keyword + "\""
		} else {
			quotedIntent[i] = keyword
		}
	}

	// If no features detected, fall back to intent-only query
	if len(featureKeywords) == 0 {
		return strings.Join(quotedIntent, " OR ")
	}

	quotedFeatures := make([]string, len(featureKeywords))
	for i, keyword := range featureKeywords {
		if strings.Contains(keyword, " ") || strings.Contains(keyword, "-") {
			quotedFeatures[i] = "\"" + keyword + "\""
		} else {
			quotedFeatures[i] = keyword
		}
	}

	// Build intent part: (intent1 OR intent2 OR intent3)
	intentPart := "(" + strings.Join(quotedIntent, " OR ") + ")"

	// Build feature part: (feature1 OR feature2 OR feature3)
	featurePart := "(" + strings.Join(quotedFeatures, " OR ") + ")"

	// Combine with AND: (intent) AND (feature)
	return intentPart + " AND " + featurePart
}
