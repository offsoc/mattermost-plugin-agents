// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

// TestFileProtocolSources_Debug tests all file protocol sources for data quality
func TestFileProtocolSources_Debug(t *testing.T) {
	config := CreateDefaultConfig()
	client := NewClient(config, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	testCases := []struct {
		sourceName string
		topic      string
		filePath   string
	}{
		{
			sourceName: SourceFeatureRequests,
			topic:      "authentication",
			filePath:   "assets/uservoice_suggestions.json",
		},
		{
			sourceName: SourceProductBoardFeatures,
			topic:      "permissions",
			filePath:   "assets/productboard-features.json",
		},
		{
			sourceName: SourceZendeskTickets,
			topic:      "plugin",
			filePath:   "assets/zendesk_tickets.txt",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.sourceName, func(t *testing.T) {
			// Check if file exists and has content
			fileInfo, err := os.Stat(tc.filePath)
			fileExists := !os.IsNotExist(err)
			hasContent := fileExists && err == nil && fileInfo.Size() > 0

			if !fileExists {
				t.Skipf("Data file not found: %s", tc.filePath)
				return
			}

			if !hasContent {
				t.Logf("%s: Data file is empty (placeholder file)", tc.sourceName)
			}

			docs, fetchErr := client.FetchFromSource(ctx, tc.sourceName, tc.topic, 5)

			if fetchErr != nil {
				t.Logf("Error fetching from %s: %v", tc.sourceName, fetchErr)
			}

			t.Logf("%s: Retrieved %d documents", tc.sourceName, len(docs))

			// Adaptive expectations: only expect docs if file has content
			if hasContent && len(docs) == 0 {
				t.Logf("Warning: File has content but no documents matched search criteria")
			} else if !hasContent && len(docs) > 0 {
				t.Errorf("Unexpected: Empty file returned %d documents", len(docs))
			}

			// Analyze document quality
			for i, doc := range docs {
				contentLen := len(doc.Content)
				hasTitle := len(doc.Title) > 5
				hasContent := contentLen > 100

				t.Logf("  Doc %d:", i)
				t.Logf("    Title: %s", doc.Title)
				t.Logf("    Content length: %d chars", contentLen)
				t.Logf("    Has proper title: %v", hasTitle)
				t.Logf("    Has sufficient content: %v", hasContent)

				// Check for HTML/JavaScript contamination
				if strings.Contains(doc.Content, "uvAuthElement") {
					t.Errorf("    ⚠️ Document contains JavaScript/HTML noise")
				}

				// Check for placeholder content
				if strings.HasPrefix(doc.Title, "I suggest you ...") {
					t.Errorf("    ⚠️ Document has placeholder title")
				}

				// Show first 200 chars of content
				if contentLen > 0 {
					preview := doc.Content
					if len(preview) > 200 {
						preview = preview[:200] + "..."
					}
					t.Logf("    Preview: %s", preview)
				}
			}
		})
	}
}

// TestUserVoiceDataQuality specifically tests UserVoice JSON quality
func TestUserVoiceDataQuality(t *testing.T) {
	filePath := "assets/uservoice_suggestions.json"
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Skip("UserVoice data file not available (data source disabled)")
	}

	protocol := NewFileProtocol(nil)

	request := ProtocolRequest{
		Source: SourceConfig{
			Name: SourceFeatureRequests,
			Endpoints: map[string]string{
				EndpointFilePath: "uservoice_suggestions.json",
			},
		},
		Topic: "",
		Limit: 100,
	}

	docs, err := protocol.Fetch(context.Background(), request)
	if err != nil {
		t.Fatalf("Failed to fetch UserVoice data: %v", err)
	}

	t.Logf("Total UserVoice documents fetched: %d", len(docs))

	// Analyze quality issues
	var (
		emptyTitles       int
		placeholderTitles int
		htmlContent       int
		shortContent      int
		goodDocs          int
	)

	for _, doc := range docs {
		if doc.Title == "" {
			emptyTitles++
		}
		if strings.HasPrefix(doc.Title, "I suggest you ...") {
			placeholderTitles++
		}
		if strings.Contains(doc.Content, "uvAuthElement") || strings.Contains(doc.Content, "<script") {
			htmlContent++
		}
		if len(doc.Content) < 100 {
			shortContent++
		}
		if len(doc.Title) > 10 && !strings.HasPrefix(doc.Title, "I suggest") &&
			len(doc.Content) > 100 && !strings.Contains(doc.Content, "uvAuthElement") {
			goodDocs++
		}
	}

	t.Logf("Quality Analysis:")
	t.Logf("  Empty titles: %d/%d (%.1f%%)", emptyTitles, len(docs), 100.0*float64(emptyTitles)/float64(len(docs)))
	t.Logf("  Placeholder titles: %d/%d (%.1f%%)", placeholderTitles, len(docs), 100.0*float64(placeholderTitles)/float64(len(docs)))
	t.Logf("  HTML/JS content: %d/%d (%.1f%%)", htmlContent, len(docs), 100.0*float64(htmlContent)/float64(len(docs)))
	t.Logf("  Short content (<100 chars): %d/%d (%.1f%%)", shortContent, len(docs), 100.0*float64(shortContent)/float64(len(docs)))
	t.Logf("  Good quality docs: %d/%d (%.1f%%)", goodDocs, len(docs), 100.0*float64(goodDocs)/float64(len(docs)))

	if goodDocs == 0 {
		t.Error("UserVoice data is completely corrupted - no usable documents")
	}
}
