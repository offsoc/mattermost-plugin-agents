// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

//go:build integration

package datasources

import (
	"context"
	"os"
	"strings"
	"testing"
)

// TestCitationQuality_HTTPContentExtraction verifies that HTTP sources extract sufficient content for citations
func TestCitationQuality_HTTPContentExtraction(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := CreateDefaultConfig()
	config.EnableSource(SourceMattermostDocs)

	client := NewClient(config, nil)

	ctx, cancel := context.WithTimeout(context.Background(), DefaultTestTimeout)
	defer cancel()

	// Test various topics to ensure content extraction works
	topics := []string{
		"authentication",
		"deployment",
		"configuration",
		"plugins",
	}

	for _, topic := range topics {
		t.Run(topic, func(t *testing.T) {
			docs, err := client.FetchFromSource(ctx, SourceMattermostDocs, topic, 5)
			if err != nil {
				t.Fatalf("Failed to fetch docs for topic %s: %v", topic, err)
			}

			if len(docs) == 0 {
				t.Errorf("No documents returned for topic %s", topic)
				return
			}

			// Verify content length meets citation requirements
			for i, doc := range docs {
				contentLength := len(doc.Content)
				if contentLength < MinContentLengthForCitation {
					t.Errorf("Document %d has insufficient content for citation: %d chars (minimum: %d)",
						i, contentLength, MinContentLengthForCitation)
				}

				// Verify content is not just metadata
				if isMainlyMetadata(doc.Content) {
					t.Errorf("Document %d appears to be mainly metadata", i)
				}

				// Verify no truncation indicators
				if strings.HasSuffix(strings.TrimSpace(doc.Content), "...") {
					t.Errorf("Document %d appears to be truncated", i)
				}

				// Log content length for analysis
				t.Logf("Document %d for topic '%s': %d chars", i, topic, contentLength)
			}
		})
	}
}

// TestCitationQuality_GitHubContent verifies GitHub issues and PRs include full content
func TestCitationQuality_GitHubContent(t *testing.T) {
	githubToken := os.Getenv(EnvGitHubToken)
	if githubToken == "" {
		t.Skip("Skipping GitHub integration test - no MM_AI_GITHUB_TOKEN environment variable")
	}

	config := CreateDefaultConfig()
	config.GitHubToken = githubToken

	// Enable github_repos for testing
	for i := range config.Sources {
		if config.Sources[i].Name == SourceGitHubRepos {
			config.Sources[i].Enabled = true
			config.Sources[i].Auth.Key = githubToken
			break
		}
	}

	client := NewClient(config, nil)

	ctx, cancel := context.WithTimeout(context.Background(), DefaultTestTimeout)
	defer cancel()

	// Test fetching issues and PRs
	topics := []string{
		"performance",
		"security",
		"api",
		"plugin",
	}

	for _, topic := range topics {
		t.Run(topic, func(t *testing.T) {
			docs, err := client.FetchFromSource(ctx, SourceGitHubRepos, topic, 5)
			if err != nil {
				t.Fatalf("Failed to fetch GitHub docs for topic %s: %v", topic, err)
			}

			if len(docs) == 0 {
				t.Logf("No documents returned for topic %s (may not have matching issues)", topic)
				return
			}

			for i, doc := range docs {
				contentLength := len(doc.Content)

				// GitHub content should be substantial
				if contentLength < MinContentLengthForCitation {
					t.Errorf("GitHub document %d has insufficient content: %d chars (minimum: %d)",
						i, contentLength, MinContentLengthForCitation)
				}

				// Check for PR/Issue specific content
				if strings.Contains(doc.URL, "/pull/") {
					// PRs should have description content
					if !strings.Contains(doc.Content, "Description:") && !strings.Contains(doc.Content, "Changes:") {
						t.Errorf("PR %d missing description section", i)
					}
				} else if strings.Contains(doc.URL, "/issues/") {
					// Issues should have full description
					if !strings.Contains(doc.Content, "Description:") && !strings.Contains(doc.Content, "Details:") {
						t.Logf("Issue %d may be missing description section", i)
					}
				}

				// Verify comments are included (if applicable)
				if strings.Contains(doc.Content, "Recent Comments:") {
					commentCount := strings.Count(doc.Content, "- @") // Comments usually start with "- @username:"
					t.Logf("Document %d includes %d comments", i, commentCount)
				}

				t.Logf("GitHub document %d for topic '%s': %d chars", i, topic, contentLength)
			}
		})
	}
}

// TestCitationQuality_JiraContent verifies Jira issues include full descriptions and comments
func TestCitationQuality_JiraContent(t *testing.T) {
	jiraToken := os.Getenv("MM_AI_JIRA_TOKEN")
	jiraEmail := os.Getenv("MM_AI_JIRA_EMAIL")

	if jiraToken == "" || jiraEmail == "" {
		t.Skip("Skipping Jira integration test - missing MM_AI_JIRA_TOKEN or MM_AI_JIRA_EMAIL")
	}

	config := CreateDefaultConfig()

	// Configure Jira source
	for i := range config.Sources {
		if config.Sources[i].Name == SourceJiraDocs {
			config.Sources[i].Enabled = true
			config.Sources[i].Endpoints[EndpointBaseURL] = "https://mattermost.atlassian.net"
			config.Sources[i].Endpoints[EndpointEmail] = jiraEmail
			config.Sources[i].Auth.Key = jiraEmail + ":" + jiraToken
			break
		}
	}

	client := NewClient(config, nil)

	ctx, cancel := context.WithTimeout(context.Background(), DefaultTestTimeout)
	defer cancel()

	// Test various Jira queries
	topics := []string{
		"plugin",
		"api",
		"performance",
	}

	for _, topic := range topics {
		t.Run(topic, func(t *testing.T) {
			docs, err := client.FetchFromSource(ctx, SourceJiraDocs, topic, 3)
			if err != nil {
				t.Fatalf("Failed to fetch Jira docs for topic %s: %v", topic, err)
			}

			if len(docs) == 0 {
				t.Logf("No Jira documents returned for topic %s", topic)
				return
			}

			for i, doc := range docs {
				contentLength := len(doc.Content)

				// Jira content should include full descriptions
				if contentLength < MinContentLengthForCitation {
					// Some Jira issues might be legitimately short, so we log instead of error
					t.Logf("Jira document %d has short content: %d chars", i, contentLength)
				}

				// Verify description is included
				if !strings.Contains(doc.Content, "Description:") {
					t.Errorf("Jira document %d missing description section", i)
				}

				// Count comments
				commentCount := strings.Count(doc.Content, "Comment by")
				if commentCount > 0 {
					t.Logf("Jira document %d includes %d comments", i, commentCount)
					if commentCount > MaxJiraCommentsToInclude {
						t.Errorf("Too many comments included: %d (max: %d)", commentCount, MaxJiraCommentsToInclude)
					}
				}

				t.Logf("Jira document %d for topic '%s': %d chars", i, topic, contentLength)
			}
		})
	}
}

// TestCitationQuality_ConfluenceContent verifies Confluence pages extract full content
func TestCitationQuality_ConfluenceContent(t *testing.T) {
	confluenceToken := os.Getenv("MM_AI_CONFLUENCE_TOKEN")
	confluenceEmail := os.Getenv("MM_AI_CONFLUENCE_EMAIL")

	if confluenceToken == "" || confluenceEmail == "" {
		t.Skip("Skipping Confluence integration test - missing MM_AI_CONFLUENCE_TOKEN or MM_AI_CONFLUENCE_EMAIL")
	}

	config := CreateDefaultConfig()

	// Configure Confluence source
	for i := range config.Sources {
		if config.Sources[i].Name == SourceConfluenceDocs {
			config.Sources[i].Enabled = true
			config.Sources[i].Endpoints[EndpointBaseURL] = "https://mattermost.atlassian.net/wiki"
			config.Sources[i].Endpoints[EndpointEmail] = confluenceEmail
			config.Sources[i].Auth.Key = confluenceEmail + ":" + confluenceToken
			break
		}
	}

	client := NewClient(config, nil)

	ctx, cancel := context.WithTimeout(context.Background(), DefaultTestTimeout)
	defer cancel()

	topics := []string{
		"architecture",
		"deployment",
		"security",
	}

	for _, topic := range topics {
		t.Run(topic, func(t *testing.T) {
			docs, err := client.FetchFromSource(ctx, SourceConfluenceDocs, topic, 3)
			if err != nil {
				t.Fatalf("Failed to fetch Confluence docs for topic %s: %v", topic, err)
			}

			if len(docs) == 0 {
				t.Logf("No Confluence documents returned for topic %s", topic)
				return
			}

			for i, doc := range docs {
				contentLength := len(doc.Content)

				// Confluence pages should have substantial content
				if contentLength < MinContentLengthForCitation {
					t.Errorf("Confluence document %d has insufficient content: %d chars (minimum: %d)",
						i, contentLength, MinContentLengthForCitation)
				}

				// Verify it's not just metadata
				if isMainlyMetadata(doc.Content) {
					t.Errorf("Confluence document %d appears to be mainly metadata", i)
				}

				t.Logf("Confluence document %d for topic '%s': %d chars", i, topic, contentLength)
			}
		})
	}
}

// TestCitationQuality_DiscourseContent verifies Discourse forum posts are complete
func TestCitationQuality_DiscourseContent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := CreateDefaultConfig()

	// Enable Mattermost forum for testing
	config.EnableSource(SourceMattermostForum)

	client := NewClient(config, nil)

	ctx, cancel := context.WithTimeout(context.Background(), DefaultTestTimeout)
	defer cancel()

	topics := []string{
		"plugin",
		"deployment",
		"performance",
	}

	for _, topic := range topics {
		t.Run(topic, func(t *testing.T) {
			docs, err := client.FetchFromSource(ctx, SourceMattermostForum, topic, 5)
			if err != nil {
				t.Fatalf("Failed to fetch forum docs for topic %s: %v", topic, err)
			}

			// Forum might legitimately return no results for some topics
			if len(docs) == 0 {
				t.Logf("No forum documents returned for topic %s", topic)
				return
			}

			for i, doc := range docs {
				contentLength := len(doc.Content)

				// Forum posts should have actual content, not just metadata
				if contentLength < 500 { // Forum posts can be shorter
					t.Logf("Forum document %d has short content: %d chars", i, contentLength)
				}

				// Verify it's actual post content, not category metadata
				if strings.Contains(doc.Content, "Category browsing") ||
					strings.Contains(doc.Content, "browsing for") {
					t.Errorf("Forum document %d appears to be category metadata, not actual post", i)
				}

				t.Logf("Forum document %d for topic '%s': %d chars", i, topic, contentLength)
			}
		})
	}
}

// TestCitationQuality_FileProtocolSources verifies file-based sources extract properly
func TestCitationQuality_FileProtocolSources(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := CreateDefaultConfig()

	// Enable file-based sources
	config.EnableSources(
		SourceFeatureRequests,
		SourceProductBoardFeatures,
		SourceZendeskTickets,
		SourceMattermostHub,
	)

	fileSources := []string{
		SourceFeatureRequests,
		SourceProductBoardFeatures,
		SourceZendeskTickets,
		SourceMattermostHub,
	}

	client := NewClient(config, nil)

	ctx, cancel := context.WithTimeout(context.Background(), DefaultTestTimeout)
	defer cancel()

	for _, sourceName := range fileSources {
		t.Run(sourceName, func(t *testing.T) {
			docs, err := client.FetchFromSource(ctx, sourceName, "plugin", 5)
			if err != nil {
				// File sources might not exist, so we log instead of fail
				t.Logf("Could not fetch from %s: %v", sourceName, err)
				return
			}

			if len(docs) == 0 {
				t.Logf("No documents returned from %s", sourceName)
				return
			}

			for i, doc := range docs {
				contentLength := len(doc.Content)

				// File sources should have reasonable content
				if contentLength < 100 {
					t.Errorf("%s document %d has very short content: %d chars", sourceName, i, contentLength)
				}

				// Check for specific source patterns
				switch sourceName {
				case SourceFeatureRequests:
					if !strings.Contains(doc.Content, "votes") && !strings.Contains(doc.Content, "Votes") {
						t.Logf("%s document %d missing vote information", sourceName, i)
					}
				case SourceProductBoardFeatures:
					if !strings.Contains(doc.Content, "pm.Priority:") && !strings.Contains(doc.Content, "priority") {
						t.Logf("%s document %d missing priority information", sourceName, i)
					}
				case SourceZendeskTickets:
					if !strings.Contains(doc.Content, "Ticket") && !strings.Contains(doc.Content, "ticket") {
						t.Logf("%s document %d doesn't look like a ticket", sourceName, i)
					}
				}

				t.Logf("%s document %d: %d chars", sourceName, i, contentLength)
			}
		})
	}
}

// TestCitationQuality_ValidationMetrics validates content quality metrics
func TestCitationQuality_ValidationMetrics(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := CreateDefaultConfig()

	sourcesToTest := []string{
		SourceMattermostDocs,
		SourceMattermostBlog,
		SourceMattermostHandbook,
	}

	for _, sourceName := range sourcesToTest {
		for i := range config.Sources {
			if config.Sources[i].Name == sourceName {
				config.Sources[i].Enabled = true
				break
			}
		}
	}

	client := NewClient(config, nil)

	ctx, cancel := context.WithTimeout(context.Background(), DefaultTestTimeout)
	defer cancel()

	for _, sourceName := range sourcesToTest {
		t.Run(sourceName, func(t *testing.T) {
			docs, err := client.FetchFromSource(ctx, sourceName, "security", 10)
			if err != nil {
				t.Fatalf("Failed to fetch from %s: %v", sourceName, err)
			}

			if len(docs) == 0 {
				t.Logf("No documents returned from %s", sourceName)
				return
			}

			totalDocs := len(docs)
			docsUnderMinLength := 0
			docsWithNoContent := 0
			docsMetadataOnly := 0
			totalLength := 0
			minLength := -1
			maxLength := 0

			for _, doc := range docs {
				contentLength := len(doc.Content)
				totalLength += contentLength

				if minLength == -1 || contentLength < minLength {
					minLength = contentLength
				}
				if contentLength > maxLength {
					maxLength = contentLength
				}

				if strings.TrimSpace(doc.Content) == "" {
					docsWithNoContent++
					continue
				}

				if contentLength < MinContentLengthForCitation {
					docsUnderMinLength++
				}

				if isMainlyMetadata(doc.Content) {
					docsMetadataOnly++
				}
			}

			avgLength := 0
			if totalDocs > 0 {
				avgLength = totalLength / totalDocs
			}

			t.Logf("Metrics for %s:", sourceName)
			t.Logf("  Total docs: %d", totalDocs)
			t.Logf("  Average length: %d chars", avgLength)
			t.Logf("  Min/Max length: %d / %d chars", minLength, maxLength)

			if docsWithNoContent > 0 {
				t.Errorf("  ⚠️ Empty content: %d docs (%.1f%%)",
					docsWithNoContent,
					float64(docsWithNoContent)*100/float64(totalDocs))
			}

			if docsUnderMinLength > 0 {
				pct := float64(docsUnderMinLength) * 100 / float64(totalDocs)
				if pct > 20 {
					t.Errorf("  ⚠️ Under min length: %d docs (%.1f%%) - too many!",
						docsUnderMinLength, pct)
				} else {
					t.Logf("  Under min length: %d docs (%.1f%%)",
						docsUnderMinLength, pct)
				}
			}

			if docsMetadataOnly > 0 {
				t.Errorf("  ⚠️ Metadata only: %d docs (%.1f%%)",
					docsMetadataOnly,
					float64(docsMetadataOnly)*100/float64(totalDocs))
			}
		})
	}
}

// Helper function to check if content is mainly metadata
func isMainlyMetadata(content string) bool {
	lines := strings.Split(content, "\n")
	metadataLines := 0
	contentLines := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || trimmed == "---" {
			continue
		}

		// Common metadata patterns
		if strings.HasPrefix(trimmed, "pm.Priority:") ||
			strings.HasPrefix(trimmed, "Segments:") ||
			strings.HasPrefix(trimmed, "Categories:") ||
			strings.HasPrefix(trimmed, "Tags:") ||
			strings.HasPrefix(trimmed, "Status:") ||
			strings.HasPrefix(trimmed, "Created:") ||
			strings.HasPrefix(trimmed, "Updated:") ||
			strings.HasPrefix(trimmed, "Author:") ||
			strings.HasPrefix(trimmed, "Type:") {
			metadataLines++
		} else {
			contentLines++
		}
	}

	if metadataLines+contentLines > 0 {
		metadataRatio := float64(metadataLines) / float64(metadataLines+contentLines)
		return metadataRatio > 0.8
	}

	return false
}
