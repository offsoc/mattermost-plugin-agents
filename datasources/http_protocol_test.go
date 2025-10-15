// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"net/http"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func TestHTTPProtocolType_GetType(t *testing.T) {
	protocol := NewHTTPProtocol(&http.Client{}, nil)
	if protocol.GetType() != HTTPProtocolType {
		t.Errorf("Expected HTTPProtocolType, got %v", protocol.GetType())
	}
}

func TestHTTPProtocolType_SetAuth(t *testing.T) {
	protocol := NewHTTPProtocol(&http.Client{}, nil)
	auth := AuthConfig{Type: "api_key", Key: "test-key"}

	protocol.SetAuth(auth)

	if protocol.auth.Type != "api_key" || protocol.auth.Key != "test-key" {
		t.Errorf("Auth not set correctly: %+v", protocol.auth)
	}
}

func TestHTTPProtocolType_BuildSectionURLs(t *testing.T) {
	protocol := NewHTTPProtocol(&http.Client{}, nil)

	source := SourceConfig{
		Endpoints: map[string]string{
			"base_url": "https://docs.example.com",
			"admin":    "/admin/",
			"api":      "/api/",
		},
	}

	sections := []string{"admin", "api"}
	urls := protocol.buildSectionURLs(source, sections)

	expected := []string{
		"https://docs.example.com/admin",
		"https://docs.example.com/api",
	}

	if len(urls) != len(expected) {
		t.Errorf("Expected %d URLs, got %d", len(expected), len(urls))
	}

	for i, url := range urls {
		if url != expected[i] {
			t.Errorf("Expected URL %s, got %s", expected[i], url)
		}
	}
}

func TestHTTPProtocolType_BuildSectionURLs_MissingBaseURL(t *testing.T) {
	protocol := NewHTTPProtocol(&http.Client{}, nil)

	source := SourceConfig{
		Endpoints: map[string]string{
			"admin": "/admin/",
		},
	}

	sections := []string{"admin"}
	urls := protocol.buildSectionURLs(source, sections)

	if urls != nil {
		t.Errorf("Expected nil URLs for missing base_url, got %v", urls)
	}
}

func TestHTTPProtocolType_BuildSectionURLs_MissingSectionEndpoints(t *testing.T) {
	protocol := NewHTTPProtocol(&http.Client{}, nil)

	// Source with only base_url (no section-specific endpoints)
	source := SourceConfig{
		Name: "test_source",
		Endpoints: map[string]string{
			"base_url": "https://handbook.example.com",
		},
	}

	sections := []string{"company", "operations", "engineering"}
	urls := protocol.buildSectionURLs(source, sections)

	// Should return no URLs since section endpoints are missing
	expected := []string{}

	if len(urls) != len(expected) {
		t.Errorf("Expected %d URLs, got %d: %v", len(expected), len(urls), urls)
	}

	for i, url := range urls {
		if url != expected[i] {
			t.Errorf("Expected URL %s, got %s", expected[i], url)
		}
	}
}

func TestHTTPProtocolType_BuildSectionURLs_NoFallbackWhenSpecificEndpointsExist(t *testing.T) {
	protocol := NewHTTPProtocol(&http.Client{}, nil)

	// Source with specific endpoints - should not fallback for missing sections
	source := SourceConfig{
		Name: "test_source",
		Endpoints: map[string]string{
			"base_url": "https://docs.example.com",
			"admin":    "/admin/",
			"api":      "/api/",
		},
	}

	sections := []string{"admin", "missing_section"}
	urls := protocol.buildSectionURLs(source, sections)

	// Should only include admin URL, not fallback for missing_section
	expected := []string{"https://docs.example.com/admin"}

	if len(urls) != len(expected) {
		t.Errorf("Expected %d URLs, got %d: %v", len(expected), len(urls), urls)
	}

	for i, url := range urls {
		if url != expected[i] {
			t.Errorf("Expected URL %s, got %s", expected[i], url)
		}
	}
}

func TestHTTPProtocolType_BuildSectionURLs_ProperlyConfiguredSources(t *testing.T) {
	protocol := NewHTTPProtocol(&http.Client{}, nil)

	// Test mattermost_handbook configuration
	handbookSource := SourceConfig{
		Name: "mattermost_handbook",
		Endpoints: map[string]string{
			"base_url":    "https://handbook.mattermost.com",
			"company":     "/company",
			"operations":  "/operations",
			"engineering": "/operations/research-and-development",
		},
	}

	sections := []string{"company", "operations", "engineering"}
	urls := protocol.buildSectionURLs(handbookSource, sections)

	expected := []string{
		"https://handbook.mattermost.com/company",
		"https://handbook.mattermost.com/operations",
		"https://handbook.mattermost.com/operations/research-and-development",
	}

	if len(urls) != len(expected) {
		t.Errorf("Expected %d URLs, got %d: %v", len(expected), len(urls), urls)
	}

	for i, url := range urls {
		if url != expected[i] {
			t.Errorf("Expected URL %s, got %s", expected[i], url)
		}
	}

	// Test mattermost_blog configuration
	blogSource := SourceConfig{
		Name: "mattermost_blog",
		Endpoints: map[string]string{
			"base_url":      "https://mattermost.com/blog",
			"blog-posts":    "/category/platform/",
			"technical":     "/category/engineering/",
			"announcements": "/category/community/",
		},
	}

	blogSections := []string{"technical", "announcements"}
	blogUrls := protocol.buildSectionURLs(blogSource, blogSections)

	expectedBlog := []string{
		"https://mattermost.com/blog/category/engineering",
		"https://mattermost.com/blog/category/community",
	}

	if len(blogUrls) != len(expectedBlog) {
		t.Errorf("Expected %d URLs, got %d: %v", len(expectedBlog), len(blogUrls), blogUrls)
	}

	for i, url := range blogUrls {
		if url != expectedBlog[i] {
			t.Errorf("Expected URL %s, got %s", expectedBlog[i], url)
		}
	}
}

func TestHTTPProtocolType_FindMostRelevantChunk(t *testing.T) {
	protocol := NewHTTPProtocol(&http.Client{}, nil)

	chunks := []string{
		"This is about database configuration",
		"This section covers mobile features",
		"General information about the product",
	}

	// Test with relevant topic
	result := protocol.findMostRelevantChunk(chunks, "mobile")
	expected := "This section covers mobile features"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}

	// Test with no topic
	result = protocol.findMostRelevantChunk(chunks, "")
	expected = "This is about database configuration" // First chunk
	if result != expected {
		t.Errorf("Expected first chunk for empty topic, got '%s'", result)
	}

	// Test with irrelevant topic
	result = protocol.findMostRelevantChunk(chunks, "irrelevant")
	expected = "This is about database configuration" // First chunk as fallback
	if result != expected {
		t.Errorf("Expected first chunk for irrelevant topic, got '%s'", result)
	}
}

func TestHTTPProtocolType_InferSection(t *testing.T) {
	protocol := NewHTTPProtocol(&http.Client{}, nil)

	tests := []struct {
		url      string
		expected string
	}{
		{"https://docs.example.com/admin/config", "config"},
		{"https://docs.example.com/api/reference/", "reference"},
		{"https://docs.example.com/", "general"},
		{"https://docs.example.com/index.html", "general"},
	}

	for _, test := range tests {
		result := protocol.inferSection(test.url)
		if result != test.expected {
			t.Errorf("For URL %s, expected section '%s', got '%s'", test.url, test.expected, result)
		}
	}
}

func TestHTTPProtocolType_ExtractTextContent(t *testing.T) {
	protocol := NewHTTPProtocol(&http.Client{}, nil)

	htmlContent := `<html>
		<head><title>Test Page</title></head>
		<body>
			<h1>Main Title</h1>
			<p>This is a paragraph with <strong>bold text</strong>.</p>
			<script>console.log('script content');</script>
			<style>body { color: red; }</style>
			<div>Another section</div>
		</body>
	</html>`

	reader := strings.NewReader(htmlContent)
	doc, err := html.Parse(reader)
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	title := protocol.extractTitle(doc)
	content := protocol.htmlProcessor.ExtractStructuredText(htmlContent)

	if title != "Test Page" {
		t.Errorf("Expected title 'Test Page', got '%s'", title)
	}

	// Check that text content is extracted and script/style are excluded
	if !strings.Contains(content, "Main Title") {
		t.Errorf("Expected content to contain 'Main Title'")
	}
	if !strings.Contains(content, "bold text") {
		t.Errorf("Expected content to contain 'bold text'")
	}
	if strings.Contains(content, "console.log") {
		t.Errorf("Content should not contain script content")
	}
	if strings.Contains(content, "color: red") {
		t.Errorf("Content should not contain style content")
	}
}
