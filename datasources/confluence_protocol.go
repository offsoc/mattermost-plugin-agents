// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/mattermost/mattermost-plugin-ai/chunking"
	"github.com/mattermost/mattermost-plugin-ai/datasources/queryutils"
	"github.com/mattermost/mattermost-plugin-ai/mmapi"
)

// ConfluenceProtocol implements the DataSourceProtocol for Confluence Cloud REST API
type ConfluenceProtocol struct {
	client          *http.Client
	rateLimiter     *RateLimiter
	auth            AuthConfig
	pluginAPI       mmapi.Client
	source          *SourceConfig // Store source config to access email endpoint
	htmlProcessor   *HTMLProcessor
	topicAnalyzer   *TopicAnalyzer
	universalScorer *UniversalRelevanceScorer
}

// NewConfluenceProtocol creates a new Confluence protocol instance
func NewConfluenceProtocol(httpClient *http.Client, pluginAPI mmapi.Client) *ConfluenceProtocol {
	return &ConfluenceProtocol{
		client:          httpClient,
		auth:            AuthConfig{Type: AuthTypeNone},
		pluginAPI:       pluginAPI,
		htmlProcessor:   NewHTMLProcessor(),
		topicAnalyzer:   NewTopicAnalyzer(),
		universalScorer: NewUniversalRelevanceScorer(),
	}
}

// Fetch retrieves documents from Confluence Cloud REST API
func (c *ConfluenceProtocol) Fetch(ctx context.Context, request ProtocolRequest) ([]Doc, error) {
	source := request.Source
	c.source = &source // Store source for auth purposes

	EnsureRateLimiter(&c.rateLimiter, source.RateLimit)

	spaceKeys := c.getSpaceKeysForSections(source, request.Sections)
	if len(spaceKeys) == 0 {
		return nil, fmt.Errorf("no spaces configured for requested sections")
	}

	var allDocs []Doc
	for _, spaceKey := range spaceKeys {
		if len(allDocs) >= request.Limit {
			break
		}

		if err := WaitRateLimiter(ctx, c.rateLimiter); err != nil {
			if c.pluginAPI != nil {
				c.pluginAPI.LogWarn(source.Name+": rate limit error", "space", spaceKey, "error", err.Error())
			}
			return allDocs, err
		}

		docs, err := c.searchInSpace(ctx, source, spaceKey, request.Topic, request.Limit-len(allDocs))
		if err != nil {
			if c.pluginAPI != nil {
				c.pluginAPI.LogWarn(source.Name+": search failed", "space", spaceKey, "error", err.Error())
			}
			continue
		}

		allDocs = append(allDocs, docs...)
	}

	return allDocs, nil
}

// GetType returns the protocol type
func (c *ConfluenceProtocol) GetType() ProtocolType {
	return ConfluenceProtocolType
}

// SetAuth configures authentication for the protocol
func (c *ConfluenceProtocol) SetAuth(auth AuthConfig) {
	c.auth = auth
}

// Close cleans up resources used by the protocol
func (c *ConfluenceProtocol) Close() error {
	CloseRateLimiter(&c.rateLimiter)
	return nil
}

// getSpaceKeysForSections maps sections to Confluence space keys
func (c *ConfluenceProtocol) getSpaceKeysForSections(source SourceConfig, sections []string) []string {
	spacesConfig := source.Endpoints[EndpointSpaces]
	if spacesConfig == "" {
		return nil
	}

	spaceKeys := strings.Split(spacesConfig, ",")
	for i, key := range spaceKeys {
		spaceKeys[i] = strings.TrimSpace(key)
	}

	return spaceKeys
}

// searchInSpace searches for content within a specific Confluence space
func (c *ConfluenceProtocol) searchInSpace(ctx context.Context, source SourceConfig, spaceKey, topic string, limit int) ([]Doc, error) {
	baseURL := source.Endpoints[EndpointBaseURL]
	if baseURL == "" {
		return nil, fmt.Errorf("no base URL configured")
	}

	searchURL := BuildAPIURL(baseURL, "rest/api/content/search")

	cql := c.buildExpandedCQL(spaceKey, topic)

	params := url.Values{}
	params.Set("cql", cql)
	params.Set("expand", "body.view,space,history,metadata.labels,version.by")
	params.Set("limit", strconv.Itoa(limit))

	fullURL := searchURL + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.addAuthHeaders(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Read the response body to get more details about the error
		bodyBytes := make([]byte, 1024)
		n, _ := resp.Body.Read(bodyBytes)
		errorBody := string(bodyBytes[:n])
		return nil, fmt.Errorf("API request failed with status %d for URL: %s, response: %s", resp.StatusCode, fullURL, errorBody)
	}

	var searchResponse ConfluenceSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResponse); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// If no results found, return empty slice (not an error)
	if len(searchResponse.Results) == 0 {
		if c.pluginAPI != nil {
			c.pluginAPI.LogWarn(source.Name+": no results", "space", spaceKey)
		}
		return []Doc{}, nil
	}

	var docs []Doc
	for _, result := range searchResponse.Results {
		doc := c.convertToDoc(result, source.Name, topic)
		if doc != nil {
			// Apply universal quality filtering (Confluence previously had no quality checks)
			if c.universalScorer.IsUniversallyAcceptable(doc.Content, doc.Title, doc.Source, topic) {
				docs = append(docs, *doc)
			} else if c.pluginAPI != nil {
				c.pluginAPI.LogDebug(source.Name+": filtered out", "url", doc.URL, "title", doc.Title)
			}
		}
	}

	return docs, nil
}

// convertToDoc converts a Confluence search result to a Doc, extracting structured text from HTML,
// stripping XML tags, extracting space metadata, page types, labels, and handling both page and blog post types
func (c *ConfluenceProtocol) convertToDoc(result ConfluenceContent, sourceName, topic string) *Doc {
	if result.Body.View.Value == "" {
		return nil
	}

	textContent := c.htmlProcessor.ExtractStructuredText(result.Body.View.Value)
	if textContent == "" {
		return nil
	}

	chunkOpts := chunking.Options{
		ChunkSize:        EnhancedChunkSize,
		ChunkOverlap:     EnhancedChunkOverlap,
		MinChunkSize:     MinChunkSizeFactor,
		ChunkingStrategy: "paragraphs", // Paragraph-based for better coherence
	}

	chunks := chunking.ChunkText(textContent, chunkOpts)
	if len(chunks) == 0 {
		return nil
	}

	chunkContents := make([]string, len(chunks))
	for i, chunk := range chunks {
		chunkContents[i] = chunk.Content
	}
	content := c.topicAnalyzer.SelectBestChunkWithContext(chunkContents, topic)

	meta := extractConfluenceMetadata(result, textContent)

	metadataStr := formatEntityMetadata(meta)
	if metadataStr != "" {
		content = fmt.Sprintf("**%s** %s\n\n%s", result.Title, metadataStr, content)
	} else {
		content = fmt.Sprintf("**%s**\n\n%s", result.Title, content)
	}

	pageURL := result.Links.Web

	var configuredBaseURL string
	if c.source != nil {
		configuredBaseURL = c.source.Endpoints[EndpointBaseURL]
	}
	if configuredBaseURL == "" {
		configuredBaseURL = ConfluenceURL // Use constant as fallback
	}

	if pageURL != "" && strings.HasPrefix(pageURL, "/") {
		baseURL := strings.TrimSuffix(configuredBaseURL, "/wiki")
		pageURL = BuildAPIURL(baseURL, pageURL)
	}

	if pageURL == "" {
		baseURL := strings.TrimSuffix(strings.TrimRight(configuredBaseURL, "/"), "/wiki")
		pageURL = fmt.Sprintf("%s/wiki/spaces/%s/pages/%s/%s",
			baseURL,
			result.Space.Key,
			result.ID,
			url.QueryEscape(result.Title))
	}

	var labels []string
	for _, label := range result.Metadata.Labels.Results {
		if label.Name != "" {
			labels = append(labels, label.Name)
		}
	}

	labels = append(labels, buildLabelsFromMetadata(meta)...)

	pageType := c.extractPageType(result.Title)
	if pageType != "" {
		labels = append(labels, fmt.Sprintf("page_type:%s", pageType))
	}

	if result.Space.Key != "" {
		labels = append(labels, fmt.Sprintf("space:%s", result.Space.Key))
	}

	daysCreated := DaysSince(result.History.CreatedDate)
	if recencyLabel := FormatRecencyLabel(daysCreated); recencyLabel != "" {
		labels = append(labels, recencyLabel+"_created")
	}
	daysUpdated := DaysSince(result.Version.When)
	if recencyLabel := FormatRecencyLabel(daysUpdated); recencyLabel != "" {
		labels = append(labels, recencyLabel+"_updated")
	}

	return &Doc{
		Title:        result.Title,
		Content:      content,
		URL:          pageURL,
		Section:      c.mapSpaceToSection(result.Space.Key),
		Source:       sourceName,
		Author:       result.Version.By.DisplayName,
		LastModified: result.Version.When,
		Labels:       labels,
		CreatedDate:  result.History.CreatedDate,
	}
}

// mapSpaceToSection maps Confluence space keys to logical sections
func (c *ConfluenceProtocol) mapSpaceToSection(spaceKey string) string {
	spaceKey = strings.ToLower(spaceKey)
	switch {
	case spaceKey == "pm" || strings.Contains(spaceKey, "req"):
		return SectionProductRequirements
	case strings.Contains(spaceKey, "market"):
		return SectionMarketResearch
	case strings.Contains(spaceKey, "spec"):
		return SectionFeatureSpecs
	case strings.Contains(spaceKey, "road"):
		return SectionRoadmaps
	case strings.Contains(spaceKey, "comp"):
		return SectionCompetitiveAnalysis
	case strings.Contains(spaceKey, "customer"):
		return SectionCustomerInsights
	default:
		return SectionGeneral
	}
}

// extractPageType extracts the page type from Confluence page title
func (c *ConfluenceProtocol) extractPageType(title string) string {
	titleLower := strings.ToLower(title)

	// UX Specifications
	if strings.Contains(titleLower, "ux spec") ||
		strings.Contains(titleLower, "ux specification") ||
		strings.Contains(titleLower, "design spec") {
		return "ux_spec"
	}

	// Architecture Decision Records
	if strings.HasPrefix(titleLower, "adr:") ||
		strings.HasPrefix(titleLower, "adr ") ||
		strings.Contains(titleLower, "decision:") ||
		strings.Contains(titleLower, "architecture decision") {
		return "decision"
	}

	// Roadmaps
	if strings.Contains(titleLower, "roadmap") ||
		strings.Contains(titleLower, "product plan") ||
		strings.Contains(titleLower, "release plan") {
		return "roadmap"
	}

	// Requirements
	if strings.Contains(titleLower, "requirements") ||
		strings.Contains(titleLower, "prd:") ||
		strings.Contains(titleLower, "product requirements") {
		return "requirements"
	}

	// Meeting notes
	if strings.Contains(titleLower, "meeting notes") ||
		strings.Contains(titleLower, "standup") ||
		strings.Contains(titleLower, "retro") {
		return "meeting_notes"
	}

	// Technical specs
	if strings.Contains(titleLower, "tech spec") ||
		strings.Contains(titleLower, "technical spec") ||
		strings.Contains(titleLower, "api spec") {
		return "tech_spec"
	}

	// Research
	if strings.Contains(titleLower, "research") ||
		strings.Contains(titleLower, "analysis") ||
		strings.Contains(titleLower, "competitive") {
		return "research"
	}

	return ""
}

// addAuthHeaders adds authentication headers to the request
func (c *ConfluenceProtocol) addAuthHeaders(req *http.Request) {
	if c.auth.Key == "" {
		return
	}

	switch c.auth.Type {
	case AuthTypeAPIKey:
		// For Confluence Cloud API Key, use Bearer authentication
		req.Header.Set(HeaderAuthorization, AuthPrefixBearer+c.auth.Key)
	case AuthTypeToken:
		emailEndpoint := ""
		if c.source != nil {
			emailEndpoint = c.source.Endpoints[EndpointEmail]
		}

		email, token, err := ParseAtlassianAuth(c.auth.Key, emailEndpoint)
		if err != nil {
			// Fallback to Bearer token if parsing fails
			req.Header.Set(HeaderAuthorization, AuthPrefixBearer+c.auth.Key)
			return
		}

		// Empty email signals Bearer auth (e.g., ATATT tokens)
		if email == "" {
			req.Header.Set(HeaderAuthorization, AuthPrefixBearer+token)
		} else {
			// Basic auth with email:token
			authValue := email + ":" + token
			encoded := base64.StdEncoding.EncodeToString([]byte(authValue))
			req.Header.Set(HeaderAuthorization, AuthPrefixBasic+encoded)
		}
	}
}

// escapeCQLString escapes special characters in CQL search strings
func (c *ConfluenceProtocol) escapeCQLString(s string) string {
	if s == "" {
		return s
	}

	// CQL special characters that need escaping: " \ + - ! ( ) { } [ ] ^ ~ * ? : /
	replacer := strings.NewReplacer(
		`\`, `\\`, // Backslash must be first
		`"`, `\"`, // Double quote
		`+`, `\+`, // Plus
		`-`, `\-`, // Minus
		`!`, `\!`, // Exclamation
		`(`, `\(`, // Left parenthesis
		`)`, `\)`, // Right parenthesis
		`{`, `\{`, // Left brace
		`}`, `\}`, // Right brace
		`[`, `\[`, // Left bracket
		`]`, `\]`, // Right bracket
		`^`, `\^`, // Caret
		`~`, `\~`, // Tilde
		`*`, `\*`, // Asterisk
		`?`, `\?`, // Question mark
		`:`, `\:`, // Colon
		`/`, `\/`, // Forward slash
	)

	return replacer.Replace(s)
}

// buildExpandedCQL constructs CQL with topic expansion and character budget management
func (c *ConfluenceProtocol) buildExpandedCQL(spaceKey, topic string) string {
	baseCQL := fmt.Sprintf("space = %s AND type = page", spaceKey)

	if topic == "" {
		return baseCQL + " ORDER BY lastModified DESC"
	}

	expandedTerms := c.topicAnalyzer.BuildExpandedSearchTerms(topic, MaxExpandedTermsConfluence)
	if len(expandedTerms) == 0 {
		return baseCQL + " ORDER BY lastModified DESC"
	}

	var searchConditions []string
	const maxCQLLength = 8000 // Conservative CQL length limit

	for _, term := range expandedTerms {
		escapedTerm := c.escapeCQLString(term)
		condition := fmt.Sprintf("(title ~ \"%s\" OR text ~ \"%s\")", escapedTerm, escapedTerm)

		testCQL := baseCQL + " AND (" + strings.Join(append(searchConditions, condition), " OR ") + ") ORDER BY lastModified DESC"
		if len(testCQL) > maxCQLLength {
			if c.pluginAPI != nil {
				c.pluginAPI.LogWarn(spaceKey+": CQL limit exceeded", "terms_used", len(searchConditions), "terms_total", len(expandedTerms))
			}
			break
		}

		searchConditions = append(searchConditions, condition)
	}

	if len(searchConditions) == 0 {
		// Fallback to simple search if expansion failed
		escapedTopic := c.escapeCQLString(topic)
		return baseCQL + fmt.Sprintf(" AND (title ~ \"%s\" OR text ~ \"%s\") ORDER BY lastModified DESC", escapedTopic, escapedTopic)
	}

	finalCQL := baseCQL + " AND (" + strings.Join(searchConditions, " OR ") + ") ORDER BY lastModified DESC"

	return finalCQL
}

// Confluence API response structures
type ConfluenceSearchResponse struct {
	Results []ConfluenceContent `json:"results"`
	Start   int                 `json:"start"`
	Limit   int                 `json:"limit"`
	Size    int                 `json:"size"`
}

type ConfluenceContent struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Title string `json:"title"`
	Space struct {
		Key  string `json:"key"`
		Name string `json:"name"`
	} `json:"space"`
	Body struct {
		View struct {
			Value          string `json:"value"`
			Representation string `json:"representation"`
		} `json:"view"`
	} `json:"body"`
	Links struct {
		Base string `json:"base"`
		Web  string `json:"webui"`
	} `json:"_links"`
	History struct {
		CreatedDate string `json:"createdDate"`
	} `json:"history"`
	Version struct {
		By struct {
			DisplayName string `json:"displayName"`
			Email       string `json:"email"`
		} `json:"by"`
		When string `json:"when"`
	} `json:"version"`
	Metadata struct {
		Labels struct {
			Results []struct {
				Name string `json:"name"`
			} `json:"results"`
		} `json:"labels"`
	} `json:"metadata"`
}

// ValidateSearchSyntax tests search queries against the Confluence CQL API to validate syntax
func (c *ConfluenceProtocol) ValidateSearchSyntax(ctx context.Context, request ProtocolRequest) (*SyntaxValidationResult, error) {
	result := &SyntaxValidationResult{
		OriginalQuery:    request.Topic,
		IsValidSyntax:    true,
		SyntaxErrors:     []string{},
		SupportsFeatures: []string{"CQL", "title search", "text search", "space filtering", "type filtering"},
	}

	if request.Topic == "" {
		result.RecommendedQuery = "mobile"
		return result, nil
	}

	c.source = &request.Source

	spaceKeys := c.getSpaceKeysForSections(request.Source, request.Sections)
	if len(spaceKeys) == 0 {
		result.IsValidSyntax = false
		result.SyntaxErrors = append(result.SyntaxErrors, "No Confluence spaces configured for search")
		result.RecommendedQuery = request.Topic
		return result, nil
	}

	testSpaceKey := spaceKeys[0]

	searchCount := c.testConfluenceSearchQuery(ctx, request.Source, testSpaceKey, request.Topic)
	result.TestResultCount = searchCount

	if searchCount == 0 {
		// Try simplified query
		simplifiedQuery := c.simplifyConfluenceQuery(request.Topic)
		simpleCount := c.testConfluenceSearchQuery(ctx, request.Source, testSpaceKey, simplifiedQuery)

		if simpleCount > 0 {
			result.IsValidSyntax = false
			result.SyntaxErrors = append(result.SyntaxErrors,
				fmt.Sprintf("Complex CQL query returned 0 results, but simplified query returned %d results", simpleCount))
			result.RecommendedQuery = simplifiedQuery
		} else {
			// Try very simple query
			verySimpleQuery := "mobile"
			verySimpleCount := c.testConfluenceSearchQuery(ctx, request.Source, testSpaceKey, verySimpleQuery)
			if verySimpleCount > 0 {
				result.IsValidSyntax = false
				result.SyntaxErrors = append(result.SyntaxErrors, "CQL query syntax may be too complex or invalid")
				result.RecommendedQuery = verySimpleQuery
			} else {
				// May indicate authentication issues or empty space
				result.SyntaxErrors = append(result.SyntaxErrors, "No search results found - may require authentication or space has no content")
				result.RecommendedQuery = request.Topic
			}
		}
	} else {
		result.RecommendedQuery = request.Topic
	}

	return result, nil
}

// testConfluenceSearchQuery performs a lightweight search test to validate CQL query syntax
func (c *ConfluenceProtocol) testConfluenceSearchQuery(ctx context.Context, source SourceConfig, spaceKey, topic string) int {
	if err := WaitRateLimiter(ctx, c.rateLimiter); err != nil {
		return 0
	}

	baseURL := source.Endpoints[EndpointBaseURL]
	if baseURL == "" {
		return 0
	}

	cql := c.buildExpandedCQL(spaceKey, topic)

	searchURL := BuildAPIURL(baseURL, "wiki/rest/api/content/search")
	searchParams := url.Values{}
	searchParams.Add("cql", cql)
	searchParams.Add("limit", "1") // Only need to know if results exist

	fullURL := searchURL + "?" + searchParams.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return 0
	}

	c.addAuthHeaders(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return 0
	}

	var searchResponse ConfluenceSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResponse); err != nil {
		return 0
	}

	return searchResponse.Size
}

// simplifyConfluenceQuery creates a Confluence CQL-friendly version of a complex query
func (c *ConfluenceProtocol) simplifyConfluenceQuery(query string) string {
	// Confluence CQL is very strict - take only the first term
	return queryutils.SimplifyQueryToKeywords(query, 1, "mobile")
}
