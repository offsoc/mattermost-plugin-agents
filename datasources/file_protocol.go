// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	pm "github.com/mattermost/mattermost-plugin-ai/datasources/segments/pm"
	"github.com/mattermost/mattermost-plugin-ai/mmapi"
)

const (
	DefaultFileProtocolMaxDocs = 20
)

// FileProtocol implements the DataSourceProtocol for local file-based datasources (JSON, CSV)
type FileProtocol struct {
	auth            AuthConfig
	pluginAPI       mmapi.Client
	topicAnalyzer   *TopicAnalyzer
	universalScorer *UniversalRelevanceScorer
}

// NewFileProtocol creates a new FileProtocol instance
func NewFileProtocol(pluginAPI mmapi.Client) *FileProtocol {
	return &FileProtocol{
		pluginAPI:       pluginAPI,
		topicAnalyzer:   NewTopicAnalyzer(),
		universalScorer: NewUniversalRelevanceScorer(),
	}
}

// GetType returns the protocol type
func (f *FileProtocol) GetType() ProtocolType {
	return FileProtocolType
}

// SetAuth sets the authentication config
func (f *FileProtocol) SetAuth(auth AuthConfig) {
	f.auth = auth
}

// Fetch retrieves documents from local file datasources
func (f *FileProtocol) Fetch(ctx context.Context, request ProtocolRequest) ([]Doc, error) {
	source := request.Source

	filePath, ok := source.Endpoints[EndpointFilePath]
	if !ok || filePath == "" {
		return nil, fmt.Errorf("file_path endpoint not configured for source %s", source.Name)
	}

	if !filepath.IsAbs(filePath) {
		var bundlePath string
		if f.pluginAPI != nil {
			bundlePath, _ = f.pluginAPI.GetBundlePath()
		}
		filePath = ResolveAssetPath(filePath, bundlePath)
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	var docs []Doc
	var err error

	switch ext {
	case ".json":
		docs, err = f.fetchFromJSON(filePath, source.Name, request)
	case ".txt":
		docs, err = f.fetchFromText(filePath, source.Name, request)
	default:
		if f.pluginAPI != nil {
			f.pluginAPI.LogWarn(source.Name+": unsupported file format", "file", filePath, "ext", ext)
		}
		return nil, fmt.Errorf("unsupported file format: %s", ext)
	}

	if err != nil {
		if f.pluginAPI != nil {
			f.pluginAPI.LogDebug(source.Name+": file read failed", "file", filePath, "error", err.Error())
		}
		return nil, fmt.Errorf("failed to fetch from file %s: %w", filePath, err)
	}

	if f.pluginAPI != nil {
		f.pluginAPI.LogDebug(source.Name+": file loaded", "file", filePath, "docs", len(docs))
	}

	limit := request.Limit
	if limit == 0 {
		limit = DefaultFileProtocolMaxDocs
	}
	if len(docs) > limit {
		docs = docs[:limit]
	}

	return docs, nil
}

// fetchFromJSON reads and searches a JSON file
func (f *FileProtocol) fetchFromJSON(filePath, sourceName string, request ProtocolRequest) ([]Doc, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		if f.pluginAPI != nil {
			f.pluginAPI.LogError(sourceName+": failed to read JSON file",
				"file", filePath,
				"error", err.Error())
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	if f.pluginAPI != nil {
		f.pluginAPI.LogDebug(sourceName+": JSON file loaded",
			"file", filePath,
			"size", len(data))
	}

	if len(data) == 0 {
		if f.pluginAPI != nil {
			f.pluginAPI.LogDebug(sourceName+": empty file, returning no documents",
				"file", filePath)
		}
		return []Doc{}, nil
	}

	var uservoiceSuggestions []UserVoiceSuggestion
	if err := json.Unmarshal(data, &uservoiceSuggestions); err == nil && len(uservoiceSuggestions) > 0 {
		if uservoiceSuggestions[0].ID != "" || uservoiceSuggestions[0].URL != "" {
			if f.pluginAPI != nil {
				f.pluginAPI.LogDebug(sourceName+": detected UserVoice format",
					"suggestions", len(uservoiceSuggestions))
			}
			return f.fetchFromUserVoiceJSON(uservoiceSuggestions, sourceName, request)
		}
	}

	var features []ProductBoardFeature
	if err := json.Unmarshal(data, &features); err != nil {
		if f.pluginAPI != nil {
			f.pluginAPI.LogError(sourceName+": failed to parse JSON",
				"file", filePath,
				"error", err.Error())
		}
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	if f.pluginAPI != nil {
		f.pluginAPI.LogDebug(sourceName+": detected ProductBoard format",
			"features", len(features))
	}

	return f.fetchFromProductBoardJSON(features, sourceName, request)
}

// fetchFromText reads and parses structured text files (e.g., Zendesk tickets, Hub posts)
func (f *FileProtocol) fetchFromText(filePath, sourceName string, request ProtocolRequest) ([]Doc, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	content := string(data)

	if len(strings.TrimSpace(content)) == 0 {
		if f.pluginAPI != nil {
			f.pluginAPI.LogDebug(sourceName+": empty file, returning no documents",
				"file", filePath)
		}
		return []Doc{}, nil
	}

	if sourceName == SourceMattermostHub {
		return f.fetchFromHubText(content, sourceName, request)
	}

	return f.fetchFromZendeskText(content, sourceName, request)
}

// extractSearchTerms extracts search terms from a topic string
func (f *FileProtocol) extractSearchTerms(topic string) []string {
	queryNode, err := ParseBooleanQuery(topic)
	if err == nil {
		return ExtractKeywords(queryNode)
	}

	normalized := strings.ToLower(topic)
	words := strings.Fields(normalized)

	var terms []string
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "from": true, "as": true, "is": true, "was": true,
	}

	for _, word := range words {
		if len(word) > 2 && !stopWords[word] {
			terms = append(terms, word)
		}
	}

	return terms
}

// ResolveAssetPath resolves a relative asset path to an absolute path
// by trying multiple possible locations where assets might be stored.
// This function is used both by file_protocol and tests to consistently
// resolve asset paths regardless of where the code is running from.
func ResolveAssetPath(relativePath string, pluginBundlePath string) string {
	if pluginBundlePath != "" {
		pluginAssetsPath := filepath.Join(pluginBundlePath, "assets", relativePath)
		if _, err := os.Stat(pluginAssetsPath); err == nil {
			return pluginAssetsPath
		}
	}

	repoRoot := findRepoRoot()
	if repoRoot != "" {
		rootAssetsPath := filepath.Join(repoRoot, "assets", relativePath)
		if _, err := os.Stat(rootAssetsPath); err == nil {
			return rootAssetsPath
		}
	}

	possiblePaths := []string{
		filepath.Join("assets", relativePath),
		filepath.Join("..", "assets", relativePath),
	}

	for _, p := range possiblePaths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	return filepath.Join("assets", relativePath)
}

func findRepoRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return ""
}

// containsSegment checks if a segment is already in the slice
func containsSegment(segments []pm.CustomerSegment, seg pm.CustomerSegment) bool {
	for _, s := range segments {
		if s == seg {
			return true
		}
	}
	return false
}

// containsCategory checks if a category is already in the slice
func containsCategory(categories []pm.TechnicalCategory, cat pm.TechnicalCategory) bool {
	for _, c := range categories {
		if c == cat {
			return true
		}
	}
	return false
}

// containsLabel checks if a label string is already in the slice
func containsLabel(labels []string, label string) bool {
	for _, l := range labels {
		if l == label {
			return true
		}
	}
	return false
}

// parseLicenseCount extracts numeric license count from various formats
// Supports: "500", "1000+", "5k", "10K", "1.5k", etc.
func parseLicenseCount(licenseStr string) int {
	if licenseStr == "" {
		return 0
	}

	licenseStr = strings.ToLower(strings.TrimSpace(licenseStr))
	licenseStr = strings.ReplaceAll(licenseStr, "licenses", "")
	licenseStr = strings.ReplaceAll(licenseStr, "users", "")
	licenseStr = strings.ReplaceAll(licenseStr, "seats", "")
	licenseStr = strings.ReplaceAll(licenseStr, "~", "")
	licenseStr = strings.TrimSpace(licenseStr)

	if strings.Contains(licenseStr, "-") {
		parts := strings.Split(licenseStr, "-")
		if len(parts) == 2 {
			licenseStr = strings.TrimSpace(parts[1])
		}
	}

	licenseStr = strings.TrimSuffix(licenseStr, "+")
	licenseStr = strings.TrimSpace(licenseStr)

	multiplier := 1
	if strings.HasSuffix(licenseStr, "k") {
		multiplier = 1000
		licenseStr = strings.TrimSuffix(licenseStr, "k")
	} else if strings.HasSuffix(licenseStr, "m") {
		multiplier = 1000000
		licenseStr = strings.TrimSuffix(licenseStr, "m")
	}

	var count float64
	if _, err := fmt.Sscanf(licenseStr, "%f", &count); err == nil {
		return int(count * float64(multiplier))
	}

	return 0
}

// ValidateSearchSyntax validates search syntax for file protocol
func (f *FileProtocol) ValidateSearchSyntax(ctx context.Context, request ProtocolRequest) (*SyntaxValidationResult, error) {
	return &SyntaxValidationResult{
		OriginalQuery:    request.Topic,
		IsValidSyntax:    true,
		SyntaxErrors:     []string{},
		RecommendedQuery: request.Topic,
		TestResultCount:  0,
		SupportsFeatures: []string{"text search", "multi-term"},
	}, nil
}
