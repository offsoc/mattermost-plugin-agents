// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"fmt"
	"regexp"
	"strings"
)

// Enhanced regex patterns for entity extraction
// Note: Patterns support both HTTP and HTTPS, and flexible domains for on-prem/enterprise deployments
var (
	// GitHub: 1-4 digit issues (avoids conflict with Zendesk 5+ digit tickets)
	GitHubIssueShortPattern = regexp.MustCompile(`(?:^|\s)#(\d{1,4})(?:\s|$|[.,;:)])`)
	// GitHub: cross-repo references (owner/repo#123)
	GitHubCrossRepoPattern = regexp.MustCompile(`\b([a-zA-Z0-9_-]+)/([a-zA-Z0-9_-]+)#(\d+)\b`)
	// GitHub URLs: Supports github.com and GitHub Enterprise (any domain with /issues/ path)
	GitHubIssueURLPattern = regexp.MustCompile(`https?://[^/]+/([^/]+)/([^/]+)/issues/(\d+)`)
	GitHubPRURLPattern    = regexp.MustCompile(`https?://[^/]+/([^/]+)/([^/]+)/pull/(\d+)`)

	JiraKeyPattern = regexp.MustCompile(`\b([A-Z]+-\d+)\b`)
	// Jira URLs: Supports atlassian.net and Jira Data Center (any domain with /browse/ path)
	JiraURLPattern = regexp.MustCompile(`https?://[^/]+/browse/([A-Z]+-\d+)`)

	// Commits: require Git keyword + 7-12 char SHAs (reduces false positives)
	CommitSHAPattern = regexp.MustCompile(`(?i)\b(?:commit|fix(?:ed)?|merge[d]?)[s]?\s+([a-f0-9]{7,12})\b`)
	// Commit URLs: Supports any Git hosting service with /commit/ path
	CommitURLPattern = regexp.MustCompile(`https?://[^/]+/([^/]+)/([^/]+)/commit/([a-f0-9]+)`)

	// File paths: Flexible pattern allowing any top-level directory (future-proof for project restructuring)
	// Matches: server/api/auth.go, shared/utils/logger.go, plugins/jira/main.go, packages/client/api.ts
	FilePathPattern = regexp.MustCompile(`(?:^|\s)([a-zA-Z0-9_-]+(?:/[a-zA-Z0-9_.-]+)+\.(?:go|ts|tsx|js|jsx|java|kt|swift|py|rb|rs))(?::(\d+)(?:-(\d+))?)?`)

	// Confluence URLs: Supports atlassian.net and self-hosted (any domain with /spaces/ or /display/ path)
	ConfluenceURLPattern = regexp.MustCompile(`https?://[^/]+/(?:spaces|display|wiki/spaces)/([A-Z]+)(?:/pages/(\d+))?`)

	// Zendesk: 5+ digit tickets (avoids conflict with GitHub 1-4 digit issues)
	ZendeskTicketPattern = regexp.MustCompile(`#(\d{5,})`)
	// Zendesk URLs: Supports zendesk.com and custom domains (any domain with /tickets/ or /agent/tickets/ path)
	ZendeskURLPattern = regexp.MustCompile(`https?://[^/]+/(?:agent/)?tickets/(\d+)`)

	// Mattermost permalinks: Supports any Mattermost instance domain
	MattermostPermalinkPattern = regexp.MustCompile(`https?://[^/]+/([^/]+)/pl/([a-z0-9]+)`)
)

// formatEntityMetadata converts EntityMetadata to inline metadata for LLM citation
// Returns a compact, citation-friendly format like: (Priority: high | Segments: enterprise, federal | Categories: mobile, performance)
func formatEntityMetadata(meta EntityMetadata) string {
	if meta.RoleMetadata == nil {
		return ""
	}

	// Get role-specific summary from the interface
	roleSummary := meta.RoleMetadata.Summary()

	// Add cross-references (role-agnostic)
	var crossRefs []string
	for _, ticket := range meta.JiraTickets {
		if ticket.Key != "" {
			crossRefs = append(crossRefs, ticket.Key)
		}
	}
	for _, issue := range meta.GitHubIssues {
		if issue.Number > 0 {
			crossRefs = append(crossRefs, fmt.Sprintf("#%d", issue.Number))
		}
	}
	for _, pr := range meta.GitHubPRs {
		if pr.Number > 0 {
			crossRefs = append(crossRefs, fmt.Sprintf("#%d", pr.Number))
		}
	}

	// If we have cross-references, append them to the role summary
	if len(crossRefs) > 0 {
		refsStr := "Refs: " + strings.Join(crossRefs, ", ")
		if roleSummary != "" {
			roleSummary = strings.TrimSuffix(roleSummary, ")")
			return roleSummary + " | " + refsStr + ")"
		}
		return "(" + refsStr + ")"
	}

	return roleSummary
}

// formatEntityMetadataDetailed creates a detailed metadata section for complex entities
// This is used when we want to preserve all metadata in a separate block (optional)
func formatEntityMetadataDetailed(meta EntityMetadata) string {
	var metadataLines []string

	// Entity linking - URLs if available, otherwise short form
	if len(meta.GitHubIssues) > 0 {
		issueStrs := make([]string, len(meta.GitHubIssues))
		for i, ref := range meta.GitHubIssues {
			if ref.URL != "" {
				issueStrs[i] = ref.URL
			} else {
				issueStrs[i] = fmt.Sprintf("#%d", ref.Number)
			}
		}
		metadataLines = append(metadataLines, fmt.Sprintf("GitHub Issues: %s", strings.Join(issueStrs, ", ")))
	}

	if len(meta.GitHubPRs) > 0 {
		prStrs := make([]string, len(meta.GitHubPRs))
		for i, ref := range meta.GitHubPRs {
			if ref.URL != "" {
				prStrs[i] = ref.URL
			} else {
				prStrs[i] = fmt.Sprintf("#%d", ref.Number)
			}
		}
		metadataLines = append(metadataLines, fmt.Sprintf("GitHub PRs: %s", strings.Join(prStrs, ", ")))
	}

	if len(meta.JiraTickets) > 0 {
		jiraStrs := make([]string, len(meta.JiraTickets))
		for i, ref := range meta.JiraTickets {
			if ref.URL != "" {
				jiraStrs[i] = ref.URL
			} else {
				jiraStrs[i] = ref.Key
			}
		}
		metadataLines = append(metadataLines, fmt.Sprintf("Jira Tickets: %s", strings.Join(jiraStrs, ", ")))
	}

	if len(meta.ModifiedFiles) > 0 {
		fileStrs := make([]string, len(meta.ModifiedFiles))
		for i, ref := range meta.ModifiedFiles {
			if ref.StartLine > 0 {
				if ref.EndLine > ref.StartLine {
					fileStrs[i] = fmt.Sprintf("%s:%d-%d", ref.Path, ref.StartLine, ref.EndLine)
				} else {
					fileStrs[i] = fmt.Sprintf("%s:%d", ref.Path, ref.StartLine)
				}
			} else {
				fileStrs[i] = ref.Path
			}
		}
		metadataLines = append(metadataLines, fmt.Sprintf("Modified Files: %s", strings.Join(fileStrs, ", ")))
	}

	if len(meta.Commits) > 0 {
		commitStrs := make([]string, len(meta.Commits))
		for i, ref := range meta.Commits {
			if ref.URL != "" {
				commitStrs[i] = ref.URL
			} else {
				commitStrs[i] = ref.ShortSHA
			}
		}
		metadataLines = append(metadataLines, fmt.Sprintf("Commits: %s", strings.Join(commitStrs, ", ")))
	}

	if len(meta.ConfluencePages) > 0 {
		confStrs := make([]string, len(meta.ConfluencePages))
		for i, ref := range meta.ConfluencePages {
			confStrs[i] = ref.URL
		}
		metadataLines = append(metadataLines, fmt.Sprintf("Confluence Pages: %s", strings.Join(confStrs, ", ")))
	}

	if len(meta.ZendeskTickets) > 0 {
		zdStrs := make([]string, len(meta.ZendeskTickets))
		for i, ref := range meta.ZendeskTickets {
			zdStrs[i] = ref.URL
		}
		metadataLines = append(metadataLines, fmt.Sprintf("Zendesk Tickets: %s", strings.Join(zdStrs, ", ")))
	}

	if len(meta.MattermostLinks) > 0 {
		mmStrs := make([]string, len(meta.MattermostLinks))
		for i, ref := range meta.MattermostLinks {
			mmStrs[i] = ref.URL
		}
		metadataLines = append(metadataLines, fmt.Sprintf("Mattermost Links: %s", strings.Join(mmStrs, ", ")))
	}

	// Only return if we have detailed metadata to show
	if len(metadataLines) == 0 {
		return ""
	}

	var lines []string
	lines = append(lines, "", "**Additional References:**")
	lines = append(lines, metadataLines...)
	lines = append(lines, "")
	return strings.Join(lines, "\n")
}

// buildLabelsFromMetadata converts EntityMetadata to search labels
func buildLabelsFromMetadata(meta EntityMetadata) []string {
	labels := []string{}

	// Get role-specific labels from the interface
	if meta.RoleMetadata != nil {
		labels = append(labels, meta.RoleMetadata.GetLabels()...)
	}

	// Add cross-reference labels (role-agnostic)
	for _, issue := range meta.GitHubIssues {
		if issue.Number > 0 {
			labels = append(labels, fmt.Sprintf("github:#%d", issue.Number))
		}
	}

	for _, pr := range meta.GitHubPRs {
		if pr.Number > 0 {
			labels = append(labels, fmt.Sprintf("github:#%d", pr.Number))
		}
	}

	for _, ticket := range meta.JiraTickets {
		if ticket.Key != "" {
			labels = append(labels, fmt.Sprintf("jira:%s", ticket.Key))
		}
	}

	return labels
}

// extractGitHubReferences extracts GitHub issues and PRs from text
func extractGitHubReferences(text string) (issues []GitHubReference, prs []GitHubReference) {
	seen := make(map[string]bool)

	// Priority 1: Full URLs (issues) - highest confidence
	issueURLMatches := GitHubIssueURLPattern.FindAllStringSubmatch(text, -1)
	for _, match := range issueURLMatches {
		if len(match) == 4 {
			number := 0
			_, _ = fmt.Sscanf(match[3], "%d", &number)
			key := fmt.Sprintf("%s/%s#%d", match[1], match[2], number)
			seen[key] = true
			issues = append(issues, GitHubReference{
				Type:   GHIssue,
				Owner:  match[1],
				Repo:   match[2],
				Number: number,
				URL:    match[0],
			})
		}
	}

	// Priority 2: Full URLs (PRs)
	prURLMatches := GitHubPRURLPattern.FindAllStringSubmatch(text, -1)
	for _, match := range prURLMatches {
		if len(match) == 4 {
			number := 0
			_, _ = fmt.Sscanf(match[3], "%d", &number)
			key := fmt.Sprintf("%s/%s#%d", match[1], match[2], number)
			seen[key] = true
			prs = append(prs, GitHubReference{
				Type:   GHPullRequest,
				Owner:  match[1],
				Repo:   match[2],
				Number: number,
				URL:    match[0],
			})
		}
	}

	// Priority 3: Cross-repo references (owner/repo#123)
	crossRepoMatches := GitHubCrossRepoPattern.FindAllStringSubmatch(text, -1)
	for _, match := range crossRepoMatches {
		if len(match) == 4 {
			number := 0
			_, _ = fmt.Sscanf(match[3], "%d", &number)
			owner := match[1]
			repo := match[2]
			key := fmt.Sprintf("%s/%s#%d", owner, repo, number)

			if !seen[key] {
				seen[key] = true
				url := fmt.Sprintf("https://github.com/%s/%s/issues/%d", owner, repo, number)
				issues = append(issues, GitHubReference{
					Type:   GHIssue,
					Owner:  owner,
					Repo:   repo,
					Number: number,
					URL:    url,
				})
			}
		}
	}

	// Priority 4: Short form (#123) - same-repo references only
	// Note: Only extracts if not already found in URLs or cross-repo refs
	shortMatches := GitHubIssueShortPattern.FindAllStringSubmatch(text, -1)
	for _, match := range shortMatches {
		if len(match) == 2 {
			number := 0
			_, _ = fmt.Sscanf(match[1], "%d", &number)

			// Check if already found (URLs or cross-repo refs have priority)
			alreadyFound := false
			for _, existing := range issues {
				if existing.Number == number && existing.Owner == "" {
					alreadyFound = true
					break
				}
			}
			for _, existing := range prs {
				if existing.Number == number && existing.Owner == "" {
					alreadyFound = true
					break
				}
			}

			if !alreadyFound {
				issues = append(issues, GitHubReference{
					Type:   GHIssue,
					Number: number,
				})
			}
		}
	}

	return issues, prs
}

// extractJiraReferences extracts Jira ticket references from text
func extractJiraReferences(text string) []JiraReference {
	refs := []JiraReference{}
	seen := make(map[string]bool)

	// Full URLs
	urlMatches := JiraURLPattern.FindAllStringSubmatch(text, -1)
	for _, match := range urlMatches {
		if len(match) == 2 {
			key := match[1]
			if seen[key] {
				continue
			}
			seen[key] = true

			parts := strings.Split(key, "-")
			if len(parts) == 2 {
				number := 0
				_, _ = fmt.Sscanf(parts[1], "%d", &number)
				refs = append(refs, JiraReference{
					Key:     key,
					Project: parts[0],
					Number:  number,
					URL:     match[0],
				})
			}
		}
	}

	// Ticket keys (MM-12345)
	keyMatches := JiraKeyPattern.FindAllStringSubmatch(text, -1)
	for _, match := range keyMatches {
		if len(match) == 2 {
			key := match[1]
			if seen[key] {
				continue
			}
			seen[key] = true

			parts := strings.Split(key, "-")
			if len(parts) == 2 {
				number := 0
				_, _ = fmt.Sscanf(parts[1], "%d", &number)
				refs = append(refs, JiraReference{
					Key:     key,
					Project: parts[0],
					Number:  number,
				})
			}
		}
	}

	return refs
}

// extractCommitReferences extracts commit SHAs from text
func extractCommitReferences(text string) []CommitReference {
	refs := []CommitReference{}
	seen := make(map[string]bool)

	// Priority 1: Full URLs (highest confidence)
	urlMatches := CommitURLPattern.FindAllStringSubmatch(text, -1)
	for _, match := range urlMatches {
		if len(match) == 4 {
			sha := match[3]
			if seen[sha] {
				continue
			}
			seen[sha] = true

			shortSHA := sha
			if len(sha) > 7 {
				shortSHA = sha[:7]
			}

			refs = append(refs, CommitReference{
				SHA:      sha,
				Owner:    match[1],
				Repo:     match[2],
				URL:      match[0],
				ShortSHA: shortSHA,
			})
		}
	}

	// Priority 2: SHAs with Git keywords (e.g., "commit a3f5c2d", "fixed 1a2b3c4")
	shaMatches := CommitSHAPattern.FindAllStringSubmatch(text, -1)
	for _, match := range shaMatches {
		if len(match) == 2 {
			sha := match[1]
			if seen[sha] {
				continue
			}
			seen[sha] = true

			shortSHA := sha
			if len(sha) > 7 {
				shortSHA = sha[:7]
			}

			refs = append(refs, CommitReference{
				SHA:      sha,
				ShortSHA: shortSHA,
			})
		}
	}

	return refs
}

// extractFileReferences extracts file path references from text
func extractFileReferences(text string) []FileReference {
	refs := []FileReference{}

	matches := FilePathPattern.FindAllStringSubmatch(text, -1)
	for _, match := range matches {
		if len(match) >= 2 {
			path := match[1]

			ref := FileReference{
				Path: path,
			}

			// Line numbers
			if len(match) >= 3 && match[2] != "" {
				_, _ = fmt.Sscanf(match[2], "%d", &ref.StartLine)
			}
			if len(match) >= 4 && match[3] != "" {
				_, _ = fmt.Sscanf(match[3], "%d", &ref.EndLine)
			}

			// Infer component
			parts := strings.Split(path, "/")
			if len(parts) >= 2 {
				ref.Component = parts[0] + "/" + parts[1]
			}

			// Infer language
			switch {
			case strings.HasSuffix(path, ".go"):
				ref.Language = "go"
			case strings.HasSuffix(path, ".ts"), strings.HasSuffix(path, ".tsx"):
				ref.Language = "typescript"
			case strings.HasSuffix(path, ".js"), strings.HasSuffix(path, ".jsx"):
				ref.Language = "javascript"
			case strings.HasSuffix(path, ".java"):
				ref.Language = "java"
			case strings.HasSuffix(path, ".kt"):
				ref.Language = "kotlin"
			case strings.HasSuffix(path, ".swift"):
				ref.Language = "swift"
			}

			refs = append(refs, ref)
		}
	}

	return refs
}

// extractConfluenceReferences extracts Confluence page references from text
func extractConfluenceReferences(text string) []ConfluenceReference {
	refs := []ConfluenceReference{}

	matches := ConfluenceURLPattern.FindAllStringSubmatch(text, -1)
	for _, match := range matches {
		if len(match) >= 2 {
			ref := ConfluenceReference{
				Space: match[1],
				URL:   match[0],
			}

			if len(match) >= 3 && match[2] != "" {
				ref.PageID = match[2]
			}

			refs = append(refs, ref)
		}
	}

	return refs
}

// extractZendeskReferences extracts Zendesk ticket references from text
func extractZendeskReferences(text string) []ZendeskReference {
	refs := []ZendeskReference{}
	seen := make(map[string]bool)

	// Full URLs
	urlMatches := ZendeskURLPattern.FindAllStringSubmatch(text, -1)
	for _, match := range urlMatches {
		if len(match) == 2 {
			ticketID := match[1]
			if seen[ticketID] {
				continue
			}
			seen[ticketID] = true

			refs = append(refs, ZendeskReference{
				TicketID: ticketID,
				URL:      match[0],
			})
		}
	}

	// Ticket numbers (#12345)
	ticketMatches := ZendeskTicketPattern.FindAllStringSubmatch(text, -1)
	for _, match := range ticketMatches {
		if len(match) == 2 {
			ticketID := match[1]
			if seen[ticketID] {
				continue
			}
			seen[ticketID] = true

			refs = append(refs, ZendeskReference{
				TicketID: ticketID,
			})
		}
	}

	return refs
}

// extractMattermostReferences extracts Mattermost permalink references from text
func extractMattermostReferences(text string) []MattermostReference {
	refs := []MattermostReference{}

	matches := MattermostPermalinkPattern.FindAllStringSubmatch(text, -1)
	for _, match := range matches {
		if len(match) == 3 {
			refs = append(refs, MattermostReference{
				TeamName: match[1],
				PostID:   match[2],
				URL:      match[0],
			})
		}
	}

	return refs
}
