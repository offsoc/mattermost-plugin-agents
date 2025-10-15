// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"golang.org/x/net/html"

	"github.com/mattermost/mattermost-plugin-ai/chunking"
	"github.com/mattermost/mattermost-plugin-ai/mmapi"
)

const (
	maxMetaRefreshDepth = 10
)

// HTTPProtocol implements the DataSourceProtocol for HTTP-based documentation sites
type HTTPProtocol struct {
	client          *http.Client
	rateLimiter     *RateLimiter
	auth            AuthConfig
	pluginAPI       mmapi.Client
	topicAnalyzer   *TopicAnalyzer
	htmlProcessor   *HTMLProcessor
	universalScorer *UniversalRelevanceScorer
	circuitBreaker  *HTTPCircuitBreaker
}

// NewHTTPProtocol creates a new HTTP protocol instance
func NewHTTPProtocol(httpClient *http.Client, pluginAPI mmapi.Client) *HTTPProtocol {
	return &HTTPProtocol{
		client:          httpClient,
		auth:            AuthConfig{Type: AuthTypeNone},
		pluginAPI:       pluginAPI,
		topicAnalyzer:   NewTopicAnalyzer(),
		htmlProcessor:   NewHTMLProcessor(),
		universalScorer: NewUniversalRelevanceScorer(),
		circuitBreaker:  newHTTPCircuitBreaker(),
	}
}

// Fetch retrieves documents from HTTP-based documentation sources
func (h *HTTPProtocol) Fetch(ctx context.Context, request ProtocolRequest) ([]Doc, error) {
	source := request.Source

	EnsureRateLimiter(&h.rateLimiter, source.RateLimit)

	refinedSections := h.refineSections(source, request.Sections, request.Topic)
	sectionURLs := h.buildSectionURLs(source, refinedSections)

	if h.pluginAPI != nil {
		h.pluginAPI.LogDebug(source.Name+": starting HTTP fetch",
			"sections", len(refinedSections),
			"urls", len(sectionURLs),
			"limit", request.Limit)
	}

	var allDocs []Doc
	fetchedCount := 0
	failedCount := 0
	lowRelevanceCount := 0

	for i, sectionURL := range sectionURLs {
		if len(allDocs) >= request.Limit {
			break
		}

		if h.pluginAPI != nil && i < 3 {
			h.pluginAPI.LogDebug(source.Name+": fetching URL",
				"index", i,
				"url", sectionURL,
				"current_docs", len(allDocs))
		}

		if err := WaitRateLimiter(ctx, h.rateLimiter); err != nil {
			if h.pluginAPI != nil {
				h.pluginAPI.LogWarn(source.Name+": rate limit error", "url", sectionURL, "error", err.Error())
			}
			return allDocs, err
		}

		if doc := h.fetchSingleDoc(ctx, sectionURL, request.Topic, source.Name, sectionURL, 0); doc != nil {
			fetchedCount++
			if request.Topic != "" {
				keywords := h.topicAnalyzer.ExtractTopicKeywords(request.Topic)
				score := h.topicAnalyzer.ScoreContentRelevanceWithTitle(doc.Content, keywords, doc.Title)
				threshold := h.topicAnalyzer.getMinimumThreshold(request.Topic)

				if score >= threshold {
					allDocs = append(allDocs, *doc)
					if h.pluginAPI != nil && len(allDocs) <= 3 {
						h.pluginAPI.LogDebug(source.Name+": doc accepted",
							"title", doc.Title,
							"score", score,
							"threshold", threshold)
					}
				} else {
					lowRelevanceCount++
					if h.pluginAPI != nil {
						h.pluginAPI.LogDebug(source.Name+": low relevance, trying articles",
							"title", doc.Title,
							"score", score,
							"threshold", threshold)
					}
					articleDocs := h.fetchArticlesFromListingPage(ctx, sectionURL, request.Topic, source.Name, sectionURL, request.Limit-len(allDocs))
					allDocs = append(allDocs, articleDocs...)
				}
			} else {
				allDocs = append(allDocs, *doc)
			}
		} else {
			failedCount++
			articleDocs := h.fetchArticlesFromListingPage(ctx, sectionURL, request.Topic, source.Name, sectionURL, request.Limit-len(allDocs))
			allDocs = append(allDocs, articleDocs...)
		}
	}

	preFilterCount := len(allDocs)
	allDocs = FilterDocsByBooleanQuery(allDocs, request.Topic)

	if h.pluginAPI != nil {
		h.pluginAPI.LogDebug(source.Name+": HTTP fetch complete",
			"urls_checked", len(sectionURLs),
			"fetched", fetchedCount,
			"failed", failedCount,
			"low_relevance", lowRelevanceCount,
			"pre_filter", preFilterCount,
			"final_results", len(allDocs))
	}

	return allDocs, nil
}

// GetType returns the protocol type
func (h *HTTPProtocol) GetType() ProtocolType {
	return HTTPProtocolType
}

// SetAuth configures authentication for the protocol
func (h *HTTPProtocol) SetAuth(auth AuthConfig) {
	h.auth = auth
}

// Close cleans up resources used by the protocol
func (h *HTTPProtocol) Close() error {
	CloseRateLimiter(&h.rateLimiter)
	return nil
}

// isSameDomainOrSubdomain checks if targetURL is on the same domain or subdomain as allowedURL
func isSameDomainOrSubdomain(targetURL, allowedURL string) bool {
	target, err := url.Parse(targetURL)
	if err != nil {
		return false
	}

	allowed, err := url.Parse(allowedURL)
	if err != nil {
		return false
	}

	targetHost := strings.ToLower(target.Host)
	allowedHost := strings.ToLower(allowed.Host)

	if targetHost == allowedHost {
		return true
	}

	if strings.HasSuffix(targetHost, "."+allowedHost) {
		return true
	}

	return false
}

// buildSectionURLs constructs URLs for the requested sections
func (h *HTTPProtocol) buildSectionURLs(source SourceConfig, sections []string) []string {
	baseURL := source.Endpoints["base_url"]
	if baseURL == "" {
		if h.pluginAPI != nil {
			h.pluginAPI.LogWarn(source.Name + ": missing base_url")
		}
		return nil
	}

	var urls []string
	for _, section := range sections {
		if sectionPath, exists := source.Endpoints[section]; exists {
			url := BuildAPIURL(baseURL, sectionPath)
			urls = append(urls, url)
		} else if h.pluginAPI != nil {
			h.pluginAPI.LogWarn(source.Name+": missing section endpoint", "section", section)
		}
	}

	return urls
}

// fetchSingleDoc retrieves and processes a single document from a URL
func (h *HTTPProtocol) fetchSingleDoc(ctx context.Context, url, topic, sourceName, allowedURL string, depth int) *Doc {
	if depth > maxMetaRefreshDepth {
		if h.pluginAPI != nil {
			h.pluginAPI.LogWarn("HTTP meta-refresh depth limit exceeded", "source", sourceName, "url", url, "depth", depth)
		}
		return nil
	}

	// SSRF protection: validate URL is on same domain as allowed URL
	if allowedURL != "" && !isSameDomainOrSubdomain(url, allowedURL) {
		if h.pluginAPI != nil {
			h.pluginAPI.LogWarn("HTTP SSRF protection: blocked cross-domain request", "source", sourceName, "url", url, "allowed", allowedURL)
		}
		return nil
	}

	if h.circuitBreaker.isOpen(url) {
		if h.pluginAPI != nil {
			h.pluginAPI.LogWarn("HTTP circuit breaker open - blocking request", "source", sourceName, "url", url)
		}
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil
	}

	req.Header.Set(HeaderUserAgent, UserAgentMattermostBot)
	req.Header.Set(HeaderAccept, AcceptHTML)

	h.addAuthHeaders(req)

	resp, err := h.client.Do(req)
	if err != nil {
		if h.pluginAPI != nil {
			h.pluginAPI.LogWarn(sourceName+": request failed", "url", url, "error", err.Error())
		}
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		if h.pluginAPI != nil {
			h.pluginAPI.LogWarn(sourceName+": body read failed", "url", url, "error", err.Error())
		}
		return nil
	}
	rawHTML := string(bodyBytes)

	doc, err := html.Parse(strings.NewReader(rawHTML))
	if err != nil {
		if h.pluginAPI != nil {
			h.pluginAPI.LogWarn(sourceName+": HTML parse failed", "url", url, "error", err.Error())
		}
		return nil
	}

	if redirectURL := h.extractMetaRefreshURL(doc); redirectURL != "" {
		return h.fetchSingleDoc(ctx, redirectURL, topic, sourceName, allowedURL, depth+1)
	}

	title := h.extractTitle(doc)
	title = h.cleanTitle(title)

	if title == "" {
		if mt := h.extractMetaContent(doc, "og:title"); mt != "" {
			title = h.cleanTitle(mt)
		}
		if title == "" {
			if h1 := h.findFirstElementText(doc, "h1"); h1 != "" {
				title = h.cleanTitle(h1)
			}
		}
	}

	content := h.htmlProcessor.ExtractStructuredText(rawHTML)
	if content == "" {
		h.circuitBreaker.recordFailure(url)
		if h.pluginAPI != nil {
			h.pluginAPI.LogWarn(sourceName+": empty content after extraction", "url", url, "html_length", len(rawHTML))
		}
		return nil
	}

	if !h.universalScorer.IsUniversallyAcceptable(content, title, sourceName, topic) {
		h.circuitBreaker.recordFailure(url)
		if h.pluginAPI != nil {
			h.debugQualityFilterRejection(content, title, sourceName, topic, url)
		}
		return nil
	}

	chunkOpts := chunking.Options{
		ChunkSize:        EnhancedChunkSize,
		ChunkOverlap:     EnhancedChunkOverlap,
		MinChunkSize:     MinChunkSizeFactor,
		ChunkingStrategy: "paragraphs", // Use paragraph-based chunking for better coherence
	}

	chunks := chunking.ChunkText(content, chunkOpts)
	if len(chunks) == 0 {
		if h.pluginAPI != nil {
			h.pluginAPI.LogWarn(sourceName+": no chunks generated", "url", url, "content_length", len(content))
		}
		return nil
	}

	chunkStrings := make([]string, len(chunks))
	for i, chunk := range chunks {
		chunkStrings[i] = chunk.Content
	}

	relevantContent := h.topicAnalyzer.SelectBestChunkWithContext(chunkStrings, topic)
	inferredSection := h.inferSection(url)

	meta := extractHTTPMetadata(title, relevantContent, url)

	metadataStr := formatEntityMetadata(meta)
	if metadataStr != "" {
		relevantContent = strings.TrimSpace(fmt.Sprintf("**%s** %s\n\n%s", title, metadataStr, relevantContent))
	} else {
		relevantContent = strings.TrimSpace(fmt.Sprintf("**%s**\n\n%s", title, relevantContent))
	}

	labels := buildLabelsFromMetadata(meta)

	return &Doc{
		Title:   title,
		Content: relevantContent,
		URL:     url,
		Section: inferredSection,
		Source:  sourceName,
		Labels:  labels,
	}
}

// fetchArticlesFromListingPage extracts article links from a category/listing page
// and fetches content from those article URLs
func (h *HTTPProtocol) fetchArticlesFromListingPage(ctx context.Context, listingURL, topic, sourceName, allowedURL string, limit int) []Doc {
	if limit <= 0 {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, "GET", listingURL, nil)
	if err != nil {
		return nil
	}

	h.addAuthHeaders(req)

	resp, err := h.client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	articleLinks := h.htmlProcessor.ExtractArticleLinks(string(bodyBytes), listingURL)

	if len(articleLinks) == 0 {
		return nil
	}

	if topic != "" {
		articleLinks = h.prioritizeArticleLinksByTopic(articleLinks, topic, limit)
	} else if len(articleLinks) > limit {
		articleLinks = articleLinks[:limit]
	}

	var docs []Doc
	for _, articleURL := range articleLinks {
		if len(docs) >= limit {
			break
		}

		if err := WaitRateLimiter(ctx, h.rateLimiter); err != nil {
			break
		}

		if doc := h.fetchSingleDoc(ctx, articleURL, topic, sourceName, allowedURL, 0); doc != nil {
			docs = append(docs, *doc)
		}
	}

	return docs
}

// prioritizeArticleLinksByTopic filters and sorts article links by topic relevance
// Returns the top N most relevant links up to the specified limit
func (h *HTTPProtocol) prioritizeArticleLinksByTopic(links []string, topic string, limit int) []string {
	if len(links) == 0 || topic == "" {
		return links
	}

	keywords := h.topicAnalyzer.ExtractTopicKeywords(topic)
	if len(keywords) == 0 {
		return links
	}

	// Score each link based on keyword matches in URL
	type scoredLink struct {
		url   string
		score int
	}

	scored := make([]scoredLink, 0, len(links))
	for _, link := range links {
		linkLower := strings.ToLower(link)
		score := 0

		for _, keyword := range keywords {
			if strings.Contains(linkLower, keyword) {
				score += 10 // High score for keyword in URL
			}
		}

		if score > 0 {
			scored = append(scored, scoredLink{url: link, score: score})
		}
	}

	if len(scored) == 0 {
		if len(links) > limit {
			return links[:limit]
		}
		return links
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	result := make([]string, 0, limit)
	for i := 0; i < len(scored) && i < limit; i++ {
		result = append(result, scored[i].url)
	}

	return result
}

// addAuthHeaders adds authentication headers to the request if configured
func (h *HTTPProtocol) addAuthHeaders(req *http.Request) {
	switch h.auth.Type {
	case "api_key":
		if h.auth.Key != "" {
			req.Header.Set(HeaderAuthorization, AuthPrefixBearer+h.auth.Key)
		}
	case "token":
		if h.auth.Key != "" {
			req.Header.Set(HeaderAuthorization, AuthPrefixToken+h.auth.Key)
		}
	}
}

// refineSections reorders/filters sections to prioritize topic-relevant content
func (h *HTTPProtocol) refineSections(source SourceConfig, sections []string, topic string) []string {
	if len(sections) == 0 {
		return sections
	}

	relevantSections := h.topicAnalyzer.GetTopicRelevantSections(topic)
	if len(relevantSections) == 0 {
		return sections
	}

	relevantSet := make(map[string]bool)
	for _, s := range relevantSections {
		relevantSet[s] = true
	}

	var prioritized []string
	var others []string

	for _, s := range sections {
		if relevantSet[s] {
			prioritized = append(prioritized, s)
		} else {
			others = append(others, s)
		}
	}

	if len(prioritized) > 0 {
		return append(prioritized, others...)
	}
	return sections
}

// inferSection attempts to infer the section from the URL path
func (h *HTTPProtocol) inferSection(url string) string {
	// Parse URL to get just the path
	if idx := strings.Index(url, "://"); idx != -1 {
		if slashIdx := strings.Index(url[idx+3:], "/"); slashIdx != -1 {
			path := url[idx+3+slashIdx:]
			parts := strings.Split(path, "/")
			// Return the last meaningful part of the path
			for i := len(parts) - 1; i >= 0; i-- {
				if parts[i] != "" && parts[i] != "index.html" {
					return parts[i]
				}
			}
		}
	}
	return "general"
}

// ValidateSearchSyntax validates search syntax by testing real queries against HTTP endpoints
func (h *HTTPProtocol) ValidateSearchSyntax(ctx context.Context, request ProtocolRequest) (*SyntaxValidationResult, error) {
	result := &SyntaxValidationResult{
		OriginalQuery:    request.Topic,
		SupportsFeatures: []string{"simple_terms", "site_search", "content_scanning"},
	}

	baseURL := request.Source.Endpoints["base_url"]
	if baseURL == "" {
		result.IsValidSyntax = false
		result.SyntaxErrors = append(result.SyntaxErrors, "missing base_url configuration")
		return result, nil
	}

	// HTTP protocol doesn't use search APIs - it scans content
	// Validation here means checking if content exists that matches the topic
	testTerms := []string{
		request.Topic,
		"mobile",        // Always test with a known good term
		"documentation", // Another safe term
	}

	var foundResults []string
	var failedTerms []string

	for _, term := range testTerms {
		if term == "" {
			continue
		}

		// Test by fetching a sample section and checking for content
		testSections := []string{"index", "getting-started", "administration"}
		if len(request.Sections) > 0 {
			testSections = request.Sections[:1] // Use first section
		}

		found := false
		for _, section := range testSections {
			sectionURLs := h.buildSectionURLs(request.Source, []string{section})
			if len(sectionURLs) > 0 {
				doc := h.fetchSingleDoc(ctx, sectionURLs[0], term, request.Source.Name, sectionURLs[0], 0)
				if doc != nil {
					if strings.Contains(strings.ToLower(doc.Content), strings.ToLower(term)) ||
						strings.Contains(strings.ToLower(doc.Title), strings.ToLower(term)) {
						found = true
						break
					}
				}
			}
			if found {
				break
			}
		}

		if found {
			foundResults = append(foundResults, term)
			if term == request.Topic {
				result.TestResultCount = 1
			}
		} else {
			failedTerms = append(failedTerms, term)
		}
	}

	// Analyze results
	if len(foundResults) == 0 {
		result.IsValidSyntax = false
		result.SyntaxErrors = append(result.SyntaxErrors, "no content found matching any test terms")
		result.RecommendedQuery = "documentation" // Safe fallback
	} else {
		result.IsValidSyntax = true

		// If original query failed but simpler ones worked
		originalFailed := false
		for _, failed := range failedTerms {
			if failed == request.Topic {
				originalFailed = true
				break
			}
		}

		if originalFailed && len(foundResults) > 0 {
			// Recommend the simplest working term
			result.RecommendedQuery = foundResults[0]
		}
	}

	// HTTP protocol works differently - complex queries don't apply
	if strings.Contains(request.Topic, "AND") || strings.Contains(request.Topic, "OR") {
		result.SyntaxErrors = append(result.SyntaxErrors, "boolean operators not applicable for HTTP content scanning")
		// Extract simple terms from complex query
		words := strings.Fields(request.Topic)
		var simpleTerms []string
		for _, word := range words {
			if word != "AND" && word != "OR" && len(word) > 2 {
				simpleTerms = append(simpleTerms, word)
			}
		}
		if len(simpleTerms) > 0 {
			result.RecommendedQuery = simpleTerms[0] // Use first meaningful term
		}
	}

	return result, nil
}

// debugQualityFilterRejection identifies why content was filtered out
func (h *HTTPProtocol) debugQualityFilterRejection(content, title, sourceName, topic, url string) {
	if h.pluginAPI == nil {
		return
	}

	var failedChecks []string

	// Check each quality filter component individually
	contentLength := len(strings.TrimSpace(content))
	if contentLength < MinContentLength {
		failedChecks = append(failedChecks, "length")
	}

	if !h.universalScorer.isPlainTextQualityAcceptable(content, title) {
		failedChecks = append(failedChecks, "plain_text_quality")
	}

	if topic != "" && !h.topicAnalyzer.IsTopicRelevantContentWithTitle(content, topic, title) {
		failedChecks = append(failedChecks, "topic_relevance")
	}

	if !h.universalScorer.meetsSourceQualityStandards(content, sourceName) {
		_ = append(failedChecks, "source_quality_standards")
	}
}
