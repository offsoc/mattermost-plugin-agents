// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsBinaryFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected bool
	}{
		{"PNG image", "logo.png", true},
		{"JPEG image", "photo.jpg", true},
		{"PDF document", "report.pdf", true},
		{"ZIP archive", "package.zip", true},
		{"Executable", "program.exe", true},
		{"Go source", "main.go", false},
		{"TypeScript source", "app.ts", false},
		{"JavaScript source", "script.js", false},
		{"Text file", "README.txt", false},
		{"Markdown file", "docs.md", false},
		{"Mixed case PNG", "Image.PNG", true},
		{"No extension", "README", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isBinaryFile(tt.filename)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDetectLanguageFromExtension(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected string
	}{
		{"Go file", "main.go", "go"},
		{"TypeScript file", "app.ts", "typescript"},
		{"JavaScript file", "script.js", "javascript"},
		{"Python file", "test.py", "python"},
		{"Java file", "Main.java", "java"},
		{"Ruby file", "app.rb", "ruby"},
		{"PHP file", "index.php", "php"},
		{"C file", "program.c", "c"},
		{"C++ file", "program.cpp", "cpp"},
		{"C# file", "Program.cs", "csharp"},
		{"Rust file", "main.rs", "rust"},
		{"TSX file", "Component.tsx", "typescript"},
		{"JSX file", "Component.jsx", "javascript"},
		{"Unknown extension", "file.xyz", ""},
		{"No extension", "README", ""},
		{"Mixed case", "File.GO", "go"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectLanguageFromExtension(tt.filename)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatCodeItemWithMatches(t *testing.T) {
	protocol := NewGitHubProtocol("test-token", nil)

	item := GitHubCodeItem{
		Name: "auth.go",
		Path: "server/api/auth.go",
		Repository: GitHubRepository{
			FullName: "mattermost/mattermost",
		},
		TextMatches: []TextMatch{
			{
				Fragment: "func Authenticate(token string) error {\n  // authentication logic\n}",
			},
			{
				Fragment: "func ValidateToken(token string) bool {\n  return token != \"\"\n}",
			},
		},
	}

	content := protocol.formatCodeItemWithMatches(item)

	assert.Contains(t, content, "server/api/auth.go")
	assert.Contains(t, content, "mattermost/mattermost")
	assert.Contains(t, content, "Code Matches:")
	assert.Contains(t, content, "func Authenticate")
	assert.Contains(t, content, "```")
}

func TestBuildCodeLabels(t *testing.T) {
	protocol := NewGitHubProtocol("test-token", nil)

	item := GitHubCodeItem{
		Name: "handler.ts",
		Path: "webapp/src/components/handler.ts",
		Repository: GitHubRepository{
			FullName: "mattermost/mattermost-webapp",
		},
	}

	labels := protocol.buildCodeLabels(item)

	assert.Contains(t, labels, "file:handler.ts")
	assert.Contains(t, labels, "path:webapp/src/components/handler.ts")
	assert.Contains(t, labels, "repo:mattermost/mattermost-webapp")
	assert.Contains(t, labels, "lang:ts")
}

func TestFetchFileContent(t *testing.T) {
	t.Run("successful fetch with base64 encoding", func(t *testing.T) {
		fileContent := "package main\n\nfunc main() {\n  println(\"Hello, World!\")\n}"
		encoded := base64.StdEncoding.EncodeToString([]byte(fileContent))

		mockResponse := GitHubFileContent{
			Content:  encoded,
			Encoding: "base64",
			Size:     len(fileContent),
			SHA:      "abc123",
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(mockResponse)
		}))
		defer server.Close()

		protocol := &GitHubProtocol{
			client: &http.Client{},
			token:  "test-token",
		}

		ctx := context.Background()
		content := protocol.fetchFileContent(ctx, server.URL)

		assert.Equal(t, fileContent, content)
	})

	t.Run("truncates large files", func(t *testing.T) {
		largeContent := make([]byte, 10000)
		for i := range largeContent {
			largeContent[i] = 'a'
		}
		encoded := base64.StdEncoding.EncodeToString(largeContent)

		mockResponse := GitHubFileContent{
			Content:  encoded,
			Encoding: "base64",
			Size:     len(largeContent),
			SHA:      "abc123",
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(mockResponse)
		}))
		defer server.Close()

		protocol := &GitHubProtocol{
			client: &http.Client{},
			token:  "test-token",
		}

		ctx := context.Background()
		content := protocol.fetchFileContent(ctx, server.URL)

		assert.Len(t, content, maxCodeFileSize+len("\n\n... (truncated, file too large)"))
		assert.Contains(t, content, "... (truncated, file too large)")
	})

	t.Run("handles HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		protocol := &GitHubProtocol{
			client: &http.Client{},
			token:  "test-token",
		}

		ctx := context.Background()
		content := protocol.fetchFileContent(ctx, server.URL)

		assert.Empty(t, content)
	})

	t.Run("handles decode error", func(t *testing.T) {
		mockResponse := GitHubFileContent{
			Content:  "invalid-base64!!!",
			Encoding: "base64",
			Size:     100,
			SHA:      "abc123",
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(mockResponse)
		}))
		defer server.Close()

		protocol := &GitHubProtocol{
			client: &http.Client{},
			token:  "test-token",
		}

		ctx := context.Background()
		content := protocol.fetchFileContent(ctx, server.URL)

		assert.Empty(t, content)
	})
}

func TestSearchCodeQueryBuilding(t *testing.T) {
	tests := []struct {
		name               string
		owner              string
		repo               string
		query              string
		language           string
		expectedInURLQuery string
	}{
		{
			name:               "basic query with repo and language",
			owner:              "mattermost",
			repo:               "mattermost",
			query:              "authentication",
			language:           "go",
			expectedInURLQuery: "authentication repo:mattermost/mattermost language:go",
		},
		{
			name:               "query without language",
			owner:              "mattermost",
			repo:               "mattermost-mobile",
			query:              "login",
			language:           "",
			expectedInURLQuery: "login repo:mattermost/mattermost-mobile",
		},
		{
			name:               "query without repo",
			owner:              "mattermost",
			repo:               "",
			query:              "websocket",
			language:           "typescript",
			expectedInURLQuery: "websocket language:typescript",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				query := r.URL.Query().Get("q")
				assert.Contains(t, query, tt.expectedInURLQuery)
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(GitHubCodeSearchResult{Items: []GitHubCodeItem{}})
			}))
			defer server.Close()

			protocol := &GitHubProtocol{
				client: &http.Client{},
				token:  "test-token",
			}

			ctx := context.Background()
			protocol.searchCode(ctx, tt.owner, tt.repo, tt.query, tt.language, 10, "test")
		})
	}
}
