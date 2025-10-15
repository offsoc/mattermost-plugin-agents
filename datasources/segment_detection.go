// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"fmt"
	"regexp"
	"strings"
)

// CrossReferencePatterns for detecting links to other systems
var CrossReferencePatterns = struct {
	JiraPattern       *regexp.Regexp
	GitHubPattern     *regexp.Regexp
	ConfluencePattern *regexp.Regexp
}{
	JiraPattern:       regexp.MustCompile(`[A-Z]+-\d+`),
	GitHubPattern:     regexp.MustCompile(`#\d+`),
	ConfluencePattern: regexp.MustCompile(`(?:atlassian\.net/wiki|confluence\..*)/(?:spaces|display)/([A-Z]+)(?:/pages/(\d+))?`),
}

// ExtractCrossReferences extracts Jira/GitHub/Confluence links from text
func ExtractCrossReferences(text string) []string {
	refs := []string{}

	// Jira patterns: MM-12345, JIRA-123
	if matches := CrossReferencePatterns.JiraPattern.FindAllString(text, -1); len(matches) > 0 {
		for _, match := range matches {
			refs = append(refs, fmt.Sprintf("jira:%s", match))
		}
	}

	// GitHub issue patterns: #1234
	if matches := CrossReferencePatterns.GitHubPattern.FindAllString(text, -1); len(matches) > 0 {
		for _, match := range matches {
			refs = append(refs, fmt.Sprintf("github:%s", match))
		}
	}

	if strings.Contains(text, MattermostDocsURL) {
		refs = append(refs, "has_docs")
	}

	if matches := CrossReferencePatterns.ConfluencePattern.FindAllStringSubmatch(text, -1); len(matches) > 0 {
		for _, match := range matches {
			if len(match) >= 3 && match[2] != "" {
				// Full page reference: space + page ID
				refs = append(refs, fmt.Sprintf("confluence:%s/pages/%s", match[1], match[2]))
			} else if len(match) >= 2 && match[1] != "" {
				// Space reference only
				refs = append(refs, fmt.Sprintf("confluence:%s", match[1]))
			}
		}
	} else if strings.Contains(text, ConfluenceURL) || strings.Contains(strings.ToLower(text), "confluence") {
		// Fallback for generic Confluence mentions
		refs = append(refs, "has_confluence")
	}

	return refs
}
