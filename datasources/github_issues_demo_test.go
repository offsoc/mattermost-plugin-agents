// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"fmt"
	"testing"
)

// TestDemonstrateQueryDifference shows the difference between old and new query building
func TestDemonstrateQueryDifference(t *testing.T) {
	g := &GitHubProtocol{}

	// Example of the problematic query from PM bot logs
	longQuery := "market competitive analysis trends research comparison benchmarking strategy positioning vision business discussion feedback community insights industry announcements alternatives evaluation customer sales ecosystem third-party user needs voting requests requirements competitor alternative trend opportunity growth adoption usage alignment enterprise enterprise advanced commercial licensing compliance governance administration enterprise features security authentication authorization sso saml ldap mfa two-factor oauth encryption audit boards kanban tasks project management cards focalboard project boards task management"

	// Old method (used for code search, still valid there)
	codeQuery := g.buildGitHubSearchQuery(longQuery)

	// New method (for issues search, respects GitHub limits)
	issuesQuery := g.buildGitHubIssuesSearchQuery(longQuery)

	fmt.Printf("\n=== Query Transformation Comparison ===\n")
	fmt.Printf("\nOriginal query length: %d characters\n", len(longQuery))
	fmt.Printf("\nCode Search query (old method): length=%d\n%s\n", len(codeQuery), codeQuery)
	fmt.Printf("\nIssues Search query (new method): length=%d\n%s\n", len(issuesQuery), issuesQuery)

	// Verify Issues query meets GitHub requirements
	if len(issuesQuery) > 256 {
		t.Errorf("Issues query exceeds 256 character limit: %d", len(issuesQuery))
	}

	// Count OR operators
	orCount := 0
	for i := 0; i < len(issuesQuery)-3; i++ {
		if issuesQuery[i:i+4] == " OR " {
			orCount++
		}
	}
	if orCount > 5 {
		t.Errorf("Issues query has more than 5 OR operators: %d", orCount)
	}

	fmt.Printf("\n=== Validation Results ===\n")
	fmt.Printf("Issues query length: %d (limit: 256) ✓\n", len(issuesQuery))
	fmt.Printf("OR operator count: %d (limit: 5) ✓\n", orCount)
}
