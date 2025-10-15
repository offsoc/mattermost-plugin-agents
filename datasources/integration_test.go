// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

//go:build integration

package datasources

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestHTTPProtocol_Integration(t *testing.T) {
	// Test with Mattermost docs which doesn't require authentication
	config := CreateDefaultConfig()

	// Enable mattermost_docs for testing
	for i := range config.Sources {
		if config.Sources[i].Name == SourceMattermostDocs {
			config.Sources[i].Enabled = true
			break
		}
	}

	client := NewClient(config, nil)

	ctx, cancel := context.WithTimeout(context.Background(), DefaultTestTimeout)
	defer cancel()

	docs, err := client.FetchFromSource(ctx, SourceMattermostDocs, "administration", 2)
	if err != nil {
		t.Fatalf("Failed to fetch docs: %v", err)
	}

	if len(docs) == 0 {
		t.Error("Expected at least one document")
	}

	for _, doc := range docs {
		if doc.Content == "" {
			t.Error("Document content should not be empty")
		}
		if doc.URL == "" {
			t.Error("Document URL should not be empty")
		}
		if doc.Source != SourceMattermostDocs {
			t.Errorf("Expected source %s, got %s", SourceMattermostDocs, doc.Source)
		}
	}
}

func TestGitHubProtocol_Integration(t *testing.T) {
	githubToken := os.Getenv(EnvGitHubToken)
	if githubToken == "" {
		t.Skip("Skipping GitHub integration test - no GITHUB_TOKEN environment variable")
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

	docs, err := client.FetchFromSource(ctx, SourceGitHubRepos, "bug", 3)
	if err != nil {
		t.Fatalf("Failed to fetch docs: %v", err)
	}

	if len(docs) == 0 {
		t.Error("Expected at least one document")
	}

	for _, doc := range docs {
		if doc.Content == "" {
			t.Error("Document content should not be empty")
		}
		if doc.URL == "" {
			t.Error("Document URL should not be empty")
		}
		if doc.Source != SourceGitHubRepos {
			t.Errorf("Expected source %s, got %s", SourceGitHubRepos, doc.Source)
		}
	}
}

func TestGitHubCodeSearch_Integration(t *testing.T) {
	githubToken := os.Getenv(EnvGitHubToken)
	if githubToken == "" {
		t.Skip("Skipping GitHub code search integration test - no GITHUB_TOKEN environment variable")
	}

	protocol := NewGitHubProtocol(githubToken, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	t.Run("search Go authentication code", func(t *testing.T) {
		docs := protocol.searchCode(ctx, "mattermost", "mattermost", "authentication", "go", 5, "test_github_code_search")

		if len(docs) == 0 {
			t.Error("Expected at least one code search result")
		}

		for _, doc := range docs {
			if doc.Section != "code" {
				t.Errorf("Expected section 'code', got '%s'", doc.Section)
			}
			if doc.Content == "" {
				t.Error("Document content should not be empty")
			}
			if doc.URL == "" {
				t.Error("Document URL should not be empty")
			}
			if !strings.Contains(doc.URL, "github.com") {
				t.Errorf("Expected GitHub URL, got %s", doc.URL)
			}
			if !strings.Contains(doc.Title, "mattermost/mattermost") {
				t.Errorf("Expected title to contain 'mattermost/mattermost', got %s", doc.Title)
			}

			hasFileLabel := false
			hasPathLabel := false
			hasRepoLabel := false
			hasLangLabel := false

			for _, label := range doc.Labels {
				if strings.HasPrefix(label, "file:") {
					hasFileLabel = true
				}
				if strings.HasPrefix(label, "path:") {
					hasPathLabel = true
				}
				if strings.HasPrefix(label, "repo:") {
					hasRepoLabel = true
				}
				if strings.HasPrefix(label, "lang:") {
					hasLangLabel = true
				}
			}

			if !hasFileLabel {
				t.Error("Expected file: label")
			}
			if !hasPathLabel {
				t.Error("Expected path: label")
			}
			if !hasRepoLabel {
				t.Error("Expected repo: label")
			}
			if !hasLangLabel {
				t.Logf("Warning: Expected lang: label for Go files")
			}

			t.Logf("✓ Found code: %s", doc.Title)
			t.Logf("  Content length: %d", len(doc.Content))
			t.Logf("  Labels: %v", doc.Labels)
		}
	})

	t.Run("search TypeScript hooks", func(t *testing.T) {
		docs := protocol.searchCode(ctx, "mattermost", "mattermost-webapp", "useState", "typescript", 3, "test_github_code_search")

		if len(docs) == 0 {
			t.Skip("No TypeScript results found - may not be an error")
		}

		for _, doc := range docs {
			if doc.Section != "code" {
				t.Errorf("Expected section 'code', got '%s'", doc.Section)
			}
			if !strings.Contains(strings.ToLower(doc.Title), ".ts") && !strings.Contains(strings.ToLower(doc.Title), ".tsx") {
				t.Logf("Warning: Expected TypeScript file extension in title: %s", doc.Title)
			}

			t.Logf("✓ Found TypeScript code: %s", doc.Title)
		}
	})

	t.Run("verify binary files are filtered", func(t *testing.T) {
		docs := protocol.searchCode(ctx, "mattermost", "mattermost", "logo", "", 10, "test_github_code_search")

		for _, doc := range docs {
			if isBinaryFile(doc.Title) {
				t.Errorf("Binary file should have been filtered: %s", doc.Title)
			}
		}

		t.Logf("✓ Verified no binary files in results")
	})

	t.Run("verify content truncation for large files", func(t *testing.T) {
		docs := protocol.searchCode(ctx, "mattermost", "mattermost", "package", "go", 10, "test_github_code_search")

		if len(docs) == 0 {
			t.Skip("No results found for truncation test")
		}

		for _, doc := range docs {
			if len(doc.Content) > maxCodeFileSize+100 {
				if !strings.Contains(doc.Content, "truncated, file too large") {
					t.Errorf("Large file should be truncated and marked: %s (length: %d)", doc.Title, len(doc.Content))
				} else {
					t.Logf("✓ Large file properly truncated: %s", doc.Title)
					break
				}
			}
		}
	})
}

func TestMattermostProtocol_Integration(t *testing.T) {
	mattermostToken := os.Getenv("MM_AI_COMMUNITY_FORUM_TOKEN")
	if mattermostToken == "" {
		t.Skip("Skipping Mattermost Community integration test - no MM_AI_COMMUNITY_FORUM_TOKEN environment variable")
	}

	config := CreateDefaultConfig()

	// Enable community_forum for testing with REAL authentication
	for i := range config.Sources {
		if config.Sources[i].Name == SourceCommunityForum {
			config.Sources[i].Enabled = true
			config.Sources[i].Auth.Type = AuthTypeToken
			config.Sources[i].Auth.Key = mattermostToken
			t.Logf("Testing with REAL authentication token")
			break
		}
	}

	client := NewClient(config, nil)

	ctx, cancel := context.WithTimeout(context.Background(), DefaultTestTimeout)
	defer cancel()

	docs, err := client.FetchFromSource(ctx, SourceCommunityForum, "troubleshooting", 2)
	if err != nil {
		t.Fatalf("Failed to fetch docs: %v", err)
	}

	if len(docs) == 0 {
		t.Error("Expected at least one document")
	}

	for _, doc := range docs {
		if doc.Content == "" {
			t.Error("Document content should not be empty")
		}
		if doc.URL == "" {
			t.Error("Document URL should not be empty")
		}
		if doc.Source != SourceCommunityForum {
			t.Errorf("Expected source %s, got %s", SourceCommunityForum, doc.Source)
		}

		// Log what we got for debugging
		t.Logf("Document: %s", doc.Title)

		// Verify we got REAL content from API, not synthetic fallback
		if strings.Contains(doc.Content, "Community discussions in the ~") {
			t.Error("❌ Got synthetic fallback content instead of real API data!")
		} else {
			t.Logf("✓ Got real channel content from API")
		}

		// Real Mattermost posts should have actual post content, timestamps, etc.
		if !strings.Contains(doc.Content, "From ~") && !strings.Contains(doc.Title, "Post by") {
			t.Logf("⚠ Content doesn't look like real Mattermost post data: %s", doc.Content[:min(100, len(doc.Content))])
		}
	}
}

func TestMattermostHubProtocol_Integration(t *testing.T) {
	mattermostHubToken := os.Getenv("MM_AI_MATTERMOST_HUB_TOKEN")
	if mattermostHubToken == "" {
		t.Skip("Skipping Mattermost Hub integration test - no MM_AI_MATTERMOST_HUB_TOKEN environment variable")
	}

	config := CreateDefaultConfig()

	// Enable mattermost_hub for testing with REAL authentication
	for i := range config.Sources {
		if config.Sources[i].Name == SourceMattermostHub {
			config.Sources[i].Enabled = true
			config.Sources[i].Auth.Type = AuthTypeToken
			config.Sources[i].Auth.Key = mattermostHubToken
			t.Logf("Testing with REAL authentication token")
			break
		}
	}

	client := NewClient(config, nil)

	ctx, cancel := context.WithTimeout(context.Background(), DefaultTestTimeout)
	defer cancel()

	docs, err := client.FetchFromSource(ctx, SourceMattermostHub, "contact-sales", 2)
	if err != nil {
		t.Fatalf("Failed to fetch docs: %v", err)
	}

	if len(docs) == 0 {
		t.Error("Expected at least one document")
	}

	for _, doc := range docs {
		if doc.Content == "" {
			t.Error("Document content should not be empty")
		}
		if doc.URL == "" {
			t.Error("Document URL should not be empty")
		}
		if doc.Source != SourceMattermostHub {
			t.Errorf("Expected source %s, got %s", SourceMattermostHub, doc.Source)
		}

		// Log what we got for debugging
		t.Logf("Document: %s", doc.Title)

		// Verify we got REAL content from API, not synthetic fallback
		if strings.Contains(doc.Content, "Community discussions in the ~") {
			t.Error("❌ Got synthetic fallback content instead of real API data!")
		} else {
			t.Logf("✓ Got real channel content from API")
		}

		// Real Mattermost posts should have actual post content, timestamps, etc.
		if !strings.Contains(doc.Content, "From ~") && !strings.Contains(doc.Title, "Post by") {
			t.Logf("⚠ Content doesn't look like real Mattermost post data: %s", doc.Content[:min(100, len(doc.Content))])
		}
	}
}

func TestMattermostHubProtocol_FallbackData(t *testing.T) {
	config := CreateDefaultConfig()

	// Enable mattermost_hub WITHOUT authentication to test fallback data
	for i := range config.Sources {
		if config.Sources[i].Name == SourceMattermostHub {
			config.Sources[i].Enabled = true
			config.Sources[i].Auth.Type = AuthTypeNone
			config.Sources[i].Auth.Key = "" // Explicitly set empty key
			t.Logf("Testing Hub fallback data (no authentication)")
			t.Logf("Hub source sections: %v", config.Sources[i].Sections)
			break
		}
	}

	client := NewClient(config, nil)

	ctx, cancel := context.WithTimeout(context.Background(), DefaultTestTimeout)
	defer cancel()

	// Test fallback data (no topic - will browse by sections)
	t.Logf("Requesting Hub fallback data with no topic")
	docs, err := client.FetchFromSource(ctx, SourceMattermostHub, "", 2)
	if err != nil {
		t.Fatalf("Failed to fetch docs: %v", err)
	}

	t.Logf("Got %d documents back", len(docs))

	if len(docs) == 0 {
		t.Error("Expected at least one document from fallback data")
	}

	for _, doc := range docs {
		if doc.Content == "" {
			t.Error("Fallback document content should not be empty")
		}
		if doc.Source != SourceMattermostHub {
			t.Errorf("Expected source %s, got %s", SourceMattermostHub, doc.Source)
		}
		if !strings.Contains(doc.Title, "Fallback Data") {
			t.Errorf("Expected title to indicate fallback data, got: %s", doc.Title)
		}

		t.Logf("✓ Got Hub fallback data: %s", doc.Title)
		t.Logf("  Content preview: %s...", doc.Content[:min(100, len(doc.Content))])
	}
}

func TestConfluenceProtocol_Integration(t *testing.T) {
	confluenceToken := os.Getenv("MM_AI_CONFLUENCE_DOCS_TOKEN")
	if confluenceToken == "" {
		t.Skip("Skipping Confluence integration test - no MM_AI_CONFLUENCE_DOCS_TOKEN environment variable")
	}

	config := CreateDefaultConfig()

	// Enable confluence_docs for testing with a more likely space key
	for i := range config.Sources {
		if config.Sources[i].Name == SourceConfluenceDocs {
			config.Sources[i].Enabled = true
			config.Sources[i].Auth.Key = confluenceToken
			// Use actual space keys that exist in mattermost.atlassian.net
			config.Sources[i].Endpoints[EndpointSpaces] = "CLOUD,DATAENG,DE,DES,FF,ICU,TW,WD,WKFL"
			break
		}
	}

	client := NewClient(config, nil)

	ctx, cancel := context.WithTimeout(context.Background(), DefaultTestTimeout)
	defer cancel()

	// Try a generic search term that's more likely to find content
	t.Logf("Searching Confluence spaces: CLOUD,DATAENG,DE,DES,FF,ICU,TW,WD,WKFL")
	docs, err := client.FetchFromSource(ctx, SourceConfluenceDocs, "", 5)
	if err != nil {
		t.Fatalf("Failed to fetch docs: %v", err)
	}

	t.Logf("Retrieved %d documents from Confluence", len(docs))

	// Also try with a specific search term
	if len(docs) == 0 {
		t.Logf("No docs found with empty search, trying with 'mattermost' search term...")
		docs2, err2 := client.FetchFromSource(ctx, SourceConfluenceDocs, "mattermost", 5)
		if err2 != nil {
			t.Logf("Search with 'mattermost' term failed: %v", err2)
		} else {
			t.Logf("Found %d documents with 'mattermost' search term", len(docs2))
			docs = append(docs, docs2...)
		}
	}

	if len(docs) == 0 {
		t.Error("Expected at least one document - this might indicate no content found, spaces don't exist, or insufficient permissions")
	}

	for _, doc := range docs {
		if doc.Content == "" {
			t.Error("Document content should not be empty")
		}
		if doc.URL == "" {
			t.Error("Document URL should not be empty")
		}
		if doc.Source != SourceConfluenceDocs {
			t.Errorf("Expected source %s, got %s", SourceConfluenceDocs, doc.Source)
		}
	}
}

func TestRateLimiter_Integration(t *testing.T) {
	// Test rate limiting with actual HTTP requests to a public endpoint
	config := CreateDefaultConfig()

	// Use aggressive rate limiting for testing
	for i := range config.Sources {
		if config.Sources[i].Name == SourceMattermostDocs {
			config.Sources[i].Enabled = true
			config.Sources[i].RateLimit.RequestsPerMinute = 6 // 1 request per 10 seconds
			config.Sources[i].RateLimit.BurstSize = 2
			break
		}
	}

	client := NewClient(config, nil)

	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), ExtendedTestTimeout)
	defer cancel()

	docs, err := client.FetchFromSource(ctx, SourceMattermostDocs, "configuration", 5)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Failed to fetch docs: %v", err)
	}

	// Should have taken some time due to rate limiting after burst exhausted
	if duration < 5*time.Second {
		t.Errorf("Request completed too quickly (%v), rate limiting may not be working", duration)
	}

	if len(docs) == 0 {
		t.Error("Expected at least one document")
	}
}

func TestMattermostBlog_Integration(t *testing.T) {
	config := CreateDefaultConfig()

	// Enable mattermost_blog for testing
	for i := range config.Sources {
		if config.Sources[i].Name == SourceMattermostBlog {
			config.Sources[i].Enabled = true
			break
		}
	}

	client := NewClient(config, nil)

	ctx, cancel := context.WithTimeout(context.Background(), DefaultTestTimeout)
	defer cancel()

	docs, err := client.FetchFromSource(ctx, SourceMattermostBlog, "security", 2)
	if err != nil {
		t.Fatalf("Failed to fetch blog docs: %v", err)
	}

	if len(docs) == 0 {
		t.Error("Expected at least one document from blog")
	}

	for _, doc := range docs {
		if doc.Content == "" {
			t.Error("Blog document content should not be empty")
		}
		if doc.URL == "" {
			t.Error("Blog document URL should not be empty")
		}
		if doc.Source != SourceMattermostBlog {
			t.Errorf("Expected source %s, got %s", SourceMattermostBlog, doc.Source)
		}
		// Verify metadata is present
		if !strings.Contains(doc.Content, "Extracted Metadata:") {
			t.Logf("Warning: Document may not have extracted metadata: %s", doc.Title)
		}
		// Verify labels are populated
		if len(doc.Labels) == 0 {
			t.Logf("Warning: Document has no labels: %s", doc.Title)
		}
	}
}

func TestMattermostNewsroom_Integration(t *testing.T) {
	config := CreateDefaultConfig()

	// Enable mattermost_newsroom for testing
	for i := range config.Sources {
		if config.Sources[i].Name == SourceMattermostNewsroom {
			config.Sources[i].Enabled = true
			break
		}
	}

	client := NewClient(config, nil)

	ctx, cancel := context.WithTimeout(context.Background(), DefaultTestTimeout)
	defer cancel()

	docs, err := client.FetchFromSource(ctx, SourceMattermostNewsroom, "announcement", 2)
	if err != nil {
		t.Fatalf("Failed to fetch newsroom docs: %v", err)
	}

	if len(docs) == 0 {
		t.Error("Expected at least one document from newsroom")
	}

	for _, doc := range docs {
		if doc.Content == "" {
			t.Error("Newsroom document content should not be empty")
		}
		if doc.URL == "" {
			t.Error("Newsroom document URL should not be empty")
		}
		if doc.Source != SourceMattermostNewsroom {
			t.Errorf("Expected source %s, got %s", SourceMattermostNewsroom, doc.Source)
		}
		// Verify metadata is present
		if !strings.Contains(doc.Content, "Extracted Metadata:") {
			t.Logf("Warning: Document may not have extracted metadata: %s", doc.Title)
		}
		// Verify labels are populated
		if len(doc.Labels) == 0 {
			t.Logf("Warning: Document has no labels: %s", doc.Title)
		}
	}
}

func TestMattermostHandbook_Integration(t *testing.T) {
	config := CreateDefaultConfig()

	// Enable mattermost_handbook for testing
	for i := range config.Sources {
		if config.Sources[i].Name == SourceMattermostHandbook {
			config.Sources[i].Enabled = true
			break
		}
	}

	client := NewClient(config, nil)

	ctx, cancel := context.WithTimeout(context.Background(), DefaultTestTimeout)
	defer cancel()

	docs, err := client.FetchFromSource(ctx, SourceMattermostHandbook, "engineering", 2)
	if err != nil {
		t.Fatalf("Failed to fetch handbook docs: %v", err)
	}

	if len(docs) == 0 {
		t.Error("Expected at least one document from handbook")
	}

	for _, doc := range docs {
		if doc.Content == "" {
			t.Error("Handbook document content should not be empty")
		}
		if doc.URL == "" {
			t.Error("Handbook document URL should not be empty")
		}
		if doc.Source != SourceMattermostHandbook {
			t.Errorf("Expected source %s, got %s", SourceMattermostHandbook, doc.Source)
		}
		// Verify metadata is present
		if !strings.Contains(doc.Content, "Extracted Metadata:") {
			t.Logf("Warning: Document may not have extracted metadata: %s", doc.Title)
		}
		// Verify labels are populated
		if len(doc.Labels) == 0 {
			t.Logf("Warning: Document has no labels: %s", doc.Title)
		}
	}
}

func TestPluginMarketplace_Integration(t *testing.T) {
	config := CreateDefaultConfig()

	// Enable plugin_marketplace for testing
	for i := range config.Sources {
		if config.Sources[i].Name == SourcePluginMarketplace {
			config.Sources[i].Enabled = true
			break
		}
	}

	client := NewClient(config, nil)

	ctx, cancel := context.WithTimeout(context.Background(), DefaultTestTimeout)
	defer cancel()

	docs, err := client.FetchFromSource(ctx, SourcePluginMarketplace, "integration", 2)
	if err != nil {
		t.Fatalf("Failed to fetch marketplace docs: %v", err)
	}

	if len(docs) == 0 {
		t.Error("Expected at least one document from marketplace")
	}

	for _, doc := range docs {
		if doc.Content == "" {
			t.Error("Marketplace document content should not be empty")
		}
		if doc.URL == "" {
			t.Error("Marketplace document URL should not be empty")
		}
		if doc.Source != SourcePluginMarketplace {
			t.Errorf("Expected source %s, got %s", SourcePluginMarketplace, doc.Source)
		}
		// Verify metadata is present
		if !strings.Contains(doc.Content, "Extracted Metadata:") {
			t.Logf("Warning: Document may not have extracted metadata: %s", doc.Title)
		}
		// Verify labels are populated
		if len(doc.Labels) == 0 {
			t.Logf("Warning: Document has no labels: %s", doc.Title)
		}
	}
}

func TestMattermostForum_Integration(t *testing.T) {
	config := CreateDefaultConfig()

	// Enable mattermost_forum for testing
	for i := range config.Sources {
		if config.Sources[i].Name == SourceMattermostForum {
			config.Sources[i].Enabled = true
			break
		}
	}

	client := NewClient(config, nil)

	ctx, cancel := context.WithTimeout(context.Background(), DefaultTestTimeout)
	defer cancel()

	docs, err := client.FetchFromSource(ctx, SourceMattermostForum, "mobile", 2)
	if err != nil {
		t.Fatalf("Failed to fetch forum docs: %v", err)
	}

	if len(docs) == 0 {
		t.Error("Expected at least one document from forum")
	}

	for _, doc := range docs {
		if doc.Content == "" {
			t.Error("Forum document content should not be empty")
		}
		if doc.URL == "" {
			t.Error("Forum document URL should not be empty")
		}
		if doc.Source != SourceMattermostForum {
			t.Errorf("Expected source %s, got %s", SourceMattermostForum, doc.Source)
		}
		// Verify metadata is present
		if !strings.Contains(doc.Content, "Extracted Metadata:") {
			t.Logf("Warning: Document may not have extracted metadata: %s", doc.Title)
		}
		// Verify labels are populated
		if len(doc.Labels) == 0 {
			t.Logf("Warning: Document has no labels: %s", doc.Title)
		}
	}
}

func TestHTTPMetadataExtraction_Integration(t *testing.T) {
	config := CreateDefaultConfig()

	// Enable mattermost_docs for testing (has public URLs we can reliably test)
	for i := range config.Sources {
		if config.Sources[i].Name == SourceMattermostDocs {
			config.Sources[i].Enabled = true
			break
		}
	}

	client := NewClient(config, nil)

	ctx, cancel := context.WithTimeout(context.Background(), DefaultTestTimeout)
	defer cancel()

	// Fetch docs with terms that should trigger metadata extraction
	docs, err := client.FetchFromSource(ctx, SourceMattermostDocs, "mobile authentication", 3)
	if err != nil {
		t.Fatalf("Failed to fetch docs: %v", err)
	}

	if len(docs) == 0 {
		t.Skip("No documents returned, cannot verify metadata extraction")
	}

	foundMetadata := false
	foundLabels := false

	for _, doc := range docs {
		t.Logf("Document: %s (%s)", doc.Title, doc.URL)

		// Check if metadata section exists
		if strings.Contains(doc.Content, "Extracted Metadata:") {
			foundMetadata = true
			t.Logf("  ✓ Has metadata section")

			// Check for specific metadata types
			if strings.Contains(doc.Content, "Categories:") {
				t.Logf("  ✓ Has categories")
			}
			if strings.Contains(doc.Content, "Segments:") {
				t.Logf("  ✓ Has segments")
			}
			if strings.Contains(doc.Content, "pm.Priority:") {
				t.Logf("  ✓ Has priority")
			}
		}

		// Check if labels are populated
		if len(doc.Labels) > 0 {
			foundLabels = true
			t.Logf("  ✓ Has %d labels: %v", len(doc.Labels), doc.Labels)

			// Verify label format
			for _, label := range doc.Labels {
				if !strings.Contains(label, ":") {
					t.Errorf("  ✗ Invalid label format (should be 'type:value'): %s", label)
				}
			}
		}
	}

	if !foundMetadata {
		t.Error("No documents had extracted metadata - metadata extraction may not be working")
	}

	if !foundLabels {
		t.Error("No documents had labels - label generation may not be working")
	}
}
