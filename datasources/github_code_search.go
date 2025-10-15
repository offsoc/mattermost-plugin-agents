// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
)

const (
	maxCodeFileSize           = 5000
	maxCodeSearchResults      = 100
	maxTextMatchesDisplayed   = 3
	codeSearchRateLimitPerMin = 10
)

var binaryFileExtensions = map[string]bool{
	".png":   true,
	".jpg":   true,
	".jpeg":  true,
	".gif":   true,
	".bmp":   true,
	".ico":   true,
	".pdf":   true,
	".zip":   true,
	".tar":   true,
	".gz":    true,
	".exe":   true,
	".dll":   true,
	".so":    true,
	".dylib": true,
	".class": true,
	".jar":   true,
	".war":   true,
	".ear":   true,
	".pyc":   true,
	".pyo":   true,
	".o":     true,
	".a":     true,
	".woff":  true,
	".woff2": true,
	".ttf":   true,
	".eot":   true,
	".mp3":   true,
	".mp4":   true,
	".avi":   true,
	".mov":   true,
	".wav":   true,
}

// GitHubCodeSearchResult represents a code search result
type GitHubCodeSearchResult struct {
	TotalCount        int              `json:"total_count"`
	IncompleteResults bool             `json:"incomplete_results"`
	Items             []GitHubCodeItem `json:"items"`
}

// GitHubCodeItem represents a single code file result
type GitHubCodeItem struct {
	Name        string           `json:"name"`
	Path        string           `json:"path"`
	SHA         string           `json:"sha"`
	URL         string           `json:"url"`
	GitURL      string           `json:"git_url"`
	HTMLURL     string           `json:"html_url"`
	Repository  GitHubRepository `json:"repository"`
	Score       float64          `json:"score"`
	TextMatches []TextMatch      `json:"text_matches"`
}

// GitHubRepository represents repository metadata in search results
type GitHubRepository struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	Owner    struct {
		Login string `json:"login"`
	} `json:"owner"`
}

// TextMatch represents highlighted code matches
type TextMatch struct {
	ObjectURL  string       `json:"object_url"`
	ObjectType string       `json:"object_type"`
	Property   string       `json:"property"`
	Fragment   string       `json:"fragment"`
	Matches    []MatchIndex `json:"matches"`
}

// MatchIndex represents match positions
type MatchIndex struct {
	Text    string `json:"text"`
	Indices []int  `json:"indices"`
}

// GitHubFileContent represents file content from blob API
type GitHubFileContent struct {
	Content  string `json:"content"`
	Encoding string `json:"encoding"`
	Size     int    `json:"size"`
	SHA      string `json:"sha"`
}

func (g *GitHubProtocol) searchCode(ctx context.Context, owner, repo, query string, language string, limit int, sourceName string) []Doc {
	return g.searchCodeSingleAttempt(ctx, owner, repo, query, language, limit, sourceName)
}

// parseCodeSearchResponse extracts docs from a successful code search response
func (g *GitHubProtocol) parseCodeSearchResponse(ctx context.Context, resp *http.Response, sourceName string) ([]Doc, error) {
	var searchResult GitHubCodeSearchResult
	if err := json.NewDecoder(resp.Body).Decode(&searchResult); err != nil {
		if g.pluginAPI != nil {
			g.pluginAPI.LogWarn("GitHub code search decode error", "error", err.Error())
		}
		return []Doc{}, err
	}

	var docs []Doc
	for _, item := range searchResult.Items {
		if isBinaryFile(item.Name) {
			continue
		}

		content := g.fetchFileContent(ctx, item.URL)
		if content == "" {
			content = g.formatCodeItemWithMatches(item)
		}

		doc := Doc{
			Title:        fmt.Sprintf("[%s] %s", item.Repository.FullName, item.Path),
			Content:      content,
			URL:          item.HTMLURL,
			Section:      "code",
			Source:       sourceName,
			Labels:       g.buildCodeLabels(item),
			CreatedDate:  "",
			LastModified: "",
		}

		docs = append(docs, doc)
	}

	return docs, nil
}

// fetchFileContent retrieves actual file content from GitHub
func (g *GitHubProtocol) fetchFileContent(ctx context.Context, contentURL string) string {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, contentURL, nil)
	if err != nil {
		return ""
	}

	g.addAuthHeaders(req)

	if g.rateLimiter != nil {
		if waitErr := g.rateLimiter.Wait(ctx); waitErr != nil {
			return ""
		}
	}

	resp, err := g.client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return ""
	}

	var fileContent GitHubFileContent
	if err := json.NewDecoder(resp.Body).Decode(&fileContent); err != nil {
		return ""
	}

	if fileContent.Encoding == "base64" {
		decoded, err := base64.StdEncoding.DecodeString(fileContent.Content)
		if err != nil {
			return ""
		}

		content := string(decoded)
		if len(content) > maxCodeFileSize {
			content = content[:maxCodeFileSize] + "\n\n... (truncated, file too large)"
		}

		return content
	}

	return fileContent.Content
}

// formatCodeItemWithMatches formats code search result with highlighted matches
func (g *GitHubProtocol) formatCodeItemWithMatches(item GitHubCodeItem) string {
	content := fmt.Sprintf("File: %s\n", item.Path)
	content += fmt.Sprintf("Repository: %s\n\n", item.Repository.FullName)

	if len(item.TextMatches) > 0 {
		content += "Code Matches:\n\n"
		for i, match := range item.TextMatches {
			if i >= maxTextMatchesDisplayed {
				break
			}
			content += fmt.Sprintf("```\n%s\n```\n\n", match.Fragment)
		}
	}

	return content
}

// buildCodeLabels creates labels for code search results
func (g *GitHubProtocol) buildCodeLabels(item GitHubCodeItem) []string {
	labels := []string{
		fmt.Sprintf("file:%s", item.Name),
		fmt.Sprintf("path:%s", item.Path),
		fmt.Sprintf("repo:%s", item.Repository.FullName),
	}

	if ext := filepath.Ext(item.Name); ext != "" {
		labels = append(labels, fmt.Sprintf("lang:%s", strings.TrimPrefix(ext, ".")))
	}

	return labels
}

// detectLanguageFromExtension maps file extensions to languages
func detectLanguageFromExtension(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	languageMap := map[string]string{
		".go":   "go",
		".js":   "javascript",
		".ts":   "typescript",
		".tsx":  "typescript",
		".jsx":  "javascript",
		".py":   "python",
		".java": "java",
		".rb":   "ruby",
		".php":  "php",
		".c":    "c",
		".cpp":  "cpp",
		".cs":   "csharp",
		".rs":   "rust",
	}

	if lang, exists := languageMap[ext]; exists {
		return lang
	}
	return ""
}

// isBinaryFile checks if a file is likely binary based on extension
func isBinaryFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return binaryFileExtensions[ext]
}

// buildGitHubSearchQuery transforms complex boolean queries into GitHub-compatible syntax
func (g *GitHubProtocol) buildGitHubSearchQuery(query string) string {
	if query == "" {
		return ""
	}

	queryUpper := strings.ToUpper(query)
	if strings.Contains(queryUpper, " AND ") || strings.Contains(queryUpper, " OR ") || strings.Contains(query, "(") {
		queryNode, err := ParseBooleanQuery(query)
		if err != nil {
			if g.pluginAPI != nil {
				g.pluginAPI.LogWarn("GitHub: failed to parse boolean query, extracting simple terms", "error", err.Error())
			}
			return extractSimpleTerms(query)
		}

		keywords := ExtractKeywords(queryNode)
		result := deduplicateAndJoinKeywords(keywords)

		return result
	}

	return query
}

// extractSimpleTerms extracts meaningful search terms from a query when boolean parsing fails
func extractSimpleTerms(query string) string {
	query = strings.ReplaceAll(query, "(", " ")
	query = strings.ReplaceAll(query, ")", " ")
	query = strings.ReplaceAll(query, "\"", " ")

	queryUpper := strings.ToUpper(query)
	queryUpper = strings.ReplaceAll(queryUpper, " AND ", " ")
	queryUpper = strings.ReplaceAll(queryUpper, " OR ", " ")
	queryUpper = strings.ReplaceAll(queryUpper, " NOT ", " ")

	fields := strings.Fields(queryUpper)
	var terms []string
	seen := make(map[string]bool)

	for _, field := range fields {
		field = strings.ToLower(field)
		if len(field) > 2 && !seen[field] {
			terms = append(terms, field)
			seen[field] = true
		}
	}

	return strings.Join(terms, " ")
}

// buildGitHubIssuesSearchQuery builds a query string specifically for GitHub Issues Search API
// Issues Search has stricter limits: 256 chars max, 5 boolean operators max
func (g *GitHubProtocol) buildGitHubIssuesSearchQuery(query string) string {
	if query == "" {
		return ""
	}

	const (
		maxQueryLength   = 256
		maxBoolOperators = 5
	)

	queryUpper := strings.ToUpper(query)
	if strings.Contains(queryUpper, " AND ") || strings.Contains(queryUpper, " OR ") || strings.Contains(query, "(") {
		queryNode, err := ParseBooleanQuery(query)
		if err != nil {
			if g.pluginAPI != nil {
				g.pluginAPI.LogWarn("GitHub Issues: failed to parse boolean query, using simple terms", "error", err.Error())
			}
			return buildSimpleIssuesQuery(query, maxQueryLength)
		}

		keywords := ExtractKeywords(queryNode)
		return buildIssuesQueryWithOR(keywords, maxQueryLength, maxBoolOperators)
	}

	return buildSimpleIssuesQuery(query, maxQueryLength)
}

// buildSimpleIssuesQuery creates a simple space-separated query for issues, respecting length limits
func buildSimpleIssuesQuery(query string, maxLength int) string {
	fields := strings.Fields(query)
	var terms []string
	seen := make(map[string]bool)
	currentLength := 0

	for _, field := range fields {
		field = strings.ToLower(strings.TrimSpace(field))
		if len(field) <= 2 || seen[field] {
			continue
		}

		if currentLength+len(field)+1 > maxLength {
			break
		}

		terms = append(terms, field)
		seen[field] = true
		currentLength += len(field) + 1
	}

	return strings.Join(terms, " ")
}

// buildIssuesQueryWithOR creates an OR-based query respecting GitHub's 5 operator limit
func buildIssuesQueryWithOR(keywords []string, maxLength, maxOperators int) string {
	seen := make(map[string]bool)
	var unique []string

	for _, kw := range keywords {
		kw = strings.TrimSpace(kw)
		kw = strings.Trim(kw, `"`)
		kw = strings.ToLower(kw)
		if kw != "" && !seen[kw] {
			unique = append(unique, kw)
			seen[kw] = true
		}
	}

	if len(unique) == 0 {
		return ""
	}

	maxTerms := maxOperators + 1
	if len(unique) > maxTerms {
		unique = unique[:maxTerms]
	}

	result := strings.Join(unique, " OR ")

	if len(result) > maxLength {
		var shortened []string
		currentLength := 0
		operatorCount := 0

		for _, term := range unique {
			termLength := len(term)
			if operatorCount > 0 {
				termLength += 4
			}

			if currentLength+termLength > maxLength || operatorCount >= maxOperators {
				break
			}

			shortened = append(shortened, term)
			currentLength += termLength
			if len(shortened) > 1 {
				operatorCount++
			}
		}

		result = strings.Join(shortened, " OR ")
	}

	return result
}

// deduplicateAndJoinKeywords removes duplicate keywords and joins them into a search string
func deduplicateAndJoinKeywords(keywords []string) string {
	seen := make(map[string]bool)
	var unique []string

	for _, kw := range keywords {
		kw = strings.TrimSpace(kw)
		kw = strings.Trim(kw, `"`)
		kw = strings.ToLower(kw)

		if len(kw) >= 3 && !seen[kw] {
			unique = append(unique, kw)
			seen[kw] = true
		}
	}

	return strings.Join(unique, " ")
}
