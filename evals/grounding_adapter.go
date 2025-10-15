// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package evals

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/mattermost/mattermost-plugin-ai/datasources"
	dsDev "github.com/mattermost/mattermost-plugin-ai/datasources/segments/dev"
	dsPM "github.com/mattermost/mattermost-plugin-ai/datasources/segments/pm"
	"github.com/mattermost/mattermost-plugin-ai/grounding"
	groundingDev "github.com/mattermost/mattermost-plugin-ai/grounding/roles/dev"
	groundingPM "github.com/mattermost/mattermost-plugin-ai/grounding/roles/pm"
)

// BuildReferenceIndex creates exact-match index from tool results
// Lives in evals (not grounding) because it depends on datasources.EntityMetadata
func BuildReferenceIndex(toolResults []string, metadata []*datasources.EntityMetadata) *grounding.ReferenceIndex {
	idx := &grounding.ReferenceIndex{
		GitHubIssues:    make(map[string]*grounding.GitHubRef),
		JiraTickets:     make(map[string]*grounding.JiraRef),
		ConfluencePages: make(map[string]*grounding.ConfluenceRef),
		ZendeskTickets:  make(map[string]*grounding.ZendeskRef),
		URLs:            make(map[string]bool),
	}

	// Helper to convert datasources RoleMetadata to grounding RoleMetadata
	convertMetadata := func(m *datasources.EntityMetadata) grounding.RoleMetadata {
		if m == nil || m.RoleMetadata == nil {
			return nil
		}

		// Type assert to specific role and convert
		switch roleMeta := m.RoleMetadata.(type) {
		case dsPM.Metadata:
			return groundingPM.NewMetadata(roleMeta)
		case dsDev.Metadata:
			return groundingDev.NewMetadata(roleMeta)
		default:
			return nil
		}
	}

	// Extract entity references from metadata
	for i, meta := range metadata {
		if meta == nil {
			continue
		}

		sourceID := fmt.Sprintf("tool_%d", i)
		groundingMeta := convertMetadata(meta)

		// If this metadata describes an entity, store its metadata
		if meta.EntityType == datasources.EntityTypeJira && meta.EntityID != "" {
			if existing, found := idx.JiraTickets[meta.EntityID]; found {
				// Update existing with metadata
				existing.Metadata = groundingMeta
				existing.Sources = append(existing.Sources, sourceID)
			} else {
				// Parse the Jira key
				parts := strings.Split(meta.EntityID, "-")
				project := ""
				number := 0
				if len(parts) == 2 {
					project = parts[0]
					number, _ = strconv.Atoi(parts[1])
				}

				// Create new entry with metadata
				idx.JiraTickets[meta.EntityID] = &grounding.JiraRef{
					Key:      meta.EntityID,
					Project:  project,
					Number:   number,
					Metadata: groundingMeta,
					Sources:  []string{sourceID},
				}
			}
		}

		if meta.EntityType == datasources.EntityTypeGitHub && meta.EntityID != "" {
			// Parse owner/repo#number
			parts := strings.Split(meta.EntityID, "/")
			if len(parts) >= 2 {
				repoParts := strings.Split(parts[1], "#")
				if len(repoParts) == 2 {
					owner := parts[0]
					repo := repoParts[0]
					number, _ := strconv.Atoi(repoParts[1])

					keys := []string{meta.EntityID, fmt.Sprintf("#%d", number)}
					for _, key := range keys {
						if existing, found := idx.GitHubIssues[key]; found {
							existing.Metadata = groundingMeta
							existing.Sources = append(existing.Sources, sourceID)
						} else {
							idx.GitHubIssues[key] = &grounding.GitHubRef{
								Owner:    owner,
								Repo:     repo,
								Number:   number,
								Metadata: groundingMeta,
								Sources:  []string{sourceID},
							}
						}
					}
				}
			}
		}

		// GitHub issues (cross-references found within the primary entity, no metadata)
		for _, issue := range meta.GitHubIssues {
			// Create keys: both full form and short form
			keys := []string{}
			if issue.Owner != "" && issue.Repo != "" {
				keys = append(keys, fmt.Sprintf("%s/%s#%d", issue.Owner, issue.Repo, issue.Number))
			}
			// Always add short form (e.g., "#19234")
			keys = append(keys, fmt.Sprintf("#%d", issue.Number))

			ref := &grounding.GitHubRef{
				Owner:  issue.Owner,
				Repo:   issue.Repo,
				Number: issue.Number,
			}

			for _, key := range keys {
				if existing, found := idx.GitHubIssues[key]; found {
					// Merge sources
					existing.Sources = append(existing.Sources, sourceID)
				} else {
					ref.Sources = []string{sourceID}
					idx.GitHubIssues[key] = ref
				}
			}
		}

		// GitHub PRs (same handling as issues)
		for _, pr := range meta.GitHubPRs {
			keys := []string{}
			if pr.Owner != "" && pr.Repo != "" {
				keys = append(keys, fmt.Sprintf("%s/%s#%d", pr.Owner, pr.Repo, pr.Number))
			}
			keys = append(keys, fmt.Sprintf("#%d", pr.Number))

			ref := &grounding.GitHubRef{
				Owner:  pr.Owner,
				Repo:   pr.Repo,
				Number: pr.Number,
			}

			for _, key := range keys {
				if existing, found := idx.GitHubIssues[key]; found {
					existing.Sources = append(existing.Sources, sourceID)
				} else {
					ref.Sources = []string{sourceID}
					idx.GitHubIssues[key] = ref
				}
			}
		}

		// Jira tickets
		for _, ticket := range meta.JiraTickets {
			key := ticket.Key // e.g., "MM-12345"
			if existing, found := idx.JiraTickets[key]; found {
				existing.Sources = append(existing.Sources, sourceID)
			} else {
				idx.JiraTickets[key] = &grounding.JiraRef{
					Key:     ticket.Key,
					Project: ticket.Project,
					Number:  ticket.Number,
					Sources: []string{sourceID},
				}
			}
		}

		// Confluence pages (use URL as key)
		for _, page := range meta.ConfluencePages {
			key := normalizeURL(page.URL)
			if existing, found := idx.ConfluencePages[key]; found {
				existing.Sources = append(existing.Sources, sourceID)
			} else {
				idx.ConfluencePages[key] = &grounding.ConfluenceRef{
					Space:   page.Space,
					PageID:  page.PageID,
					URL:     page.URL,
					Sources: []string{sourceID},
				}
			}
		}

		// Zendesk tickets
		for _, ticket := range meta.ZendeskTickets {
			key := ticket.TicketID
			if existing, found := idx.ZendeskTickets[key]; found {
				existing.Sources = append(existing.Sources, sourceID)
			} else {
				idx.ZendeskTickets[key] = &grounding.ZendeskRef{
					TicketID: ticket.TicketID,
					URL:      ticket.URL,
					Sources:  []string{sourceID},
				}
			}
		}
	}

	// Extract URLs from tool result content (for citations not in metadata)
	urlPattern := regexp.MustCompile(`https?://[^\s<>"]+`)
	for _, content := range toolResults {
		matches := urlPattern.FindAllString(content, -1)
		for _, match := range matches {
			normalized := normalizeURL(match)
			idx.URLs[normalized] = true
		}
	}

	return idx
}

// normalizeURL removes protocol and trailing slashes for comparison
func normalizeURL(rawURL string) string {
	// Parse URL to handle edge cases
	parsed, err := url.Parse(rawURL)
	if err != nil {
		// Fallback to simple normalization
		normalized := strings.ToLower(rawURL)
		normalized = strings.TrimPrefix(normalized, "https://")
		normalized = strings.TrimPrefix(normalized, "http://")
		normalized = strings.TrimSuffix(normalized, "/")
		return normalized
	}

	// Normalize: lowercase host, remove trailing slash from path
	host := strings.ToLower(parsed.Host)
	path := strings.TrimSuffix(parsed.Path, "/")
	query := parsed.RawQuery

	normalized := host + path
	if query != "" {
		normalized += "?" + query
	}

	return normalized
}
