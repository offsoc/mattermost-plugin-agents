// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/mattermost/mattermost-plugin-ai/httpexternal"
	"github.com/mattermost/mattermost-plugin-ai/mmapi"
)

// Client manages external documentation protocols and sources
type Client struct {
	protocols map[ProtocolType]DataSourceProtocol
	sources   map[string]SourceConfig
	cache     *TTLCache
	config    *Config
	pluginAPI mmapi.Client
}

// NewClient creates a new external documentation client
func NewClient(config *Config, pluginAPI mmapi.Client) *Client {
	if config == nil {
		config = CreateDefaultConfig()
	}

	client := &Client{
		protocols: make(map[ProtocolType]DataSourceProtocol),
		sources:   make(map[string]SourceConfig),
		cache:     NewTTLCache(config.CacheTTL),
		config:    config,
		pluginAPI: pluginAPI,
	}

	client.initializeProtocols()

	enabledSources := config.GetEnabledSources()
	if pluginAPI != nil {
		pluginAPI.LogDebug("datasources client initialization", "enabled_sources", len(enabledSources), "github_token_set", config.GitHubToken != "")
	}

	for _, sourceConfig := range enabledSources {
		client.sources[sourceConfig.Name] = sourceConfig
		if pluginAPI != nil {
			pluginAPI.LogDebug("loaded data source", "source", sourceConfig.Name, "protocol", sourceConfig.Protocol, "auth_type", sourceConfig.Auth.Type, "auth_key_set", sourceConfig.Auth.Key != "")

			// Extra logging for Jira to help debug auth issues
			if sourceConfig.Name == SourceJiraDocs {
				pluginAPI.LogDebug("Jira datasource details",
					"source", sourceConfig.Name,
					"auth_type", sourceConfig.Auth.Type,
					"auth_key_length", len(sourceConfig.Auth.Key),
					"base_url", sourceConfig.Endpoints[EndpointBaseURL],
					"email_endpoint", sourceConfig.Endpoints[EndpointEmail])
			}
		}
	}

	return client
}

// FetchFromSource fetches documents from a specific external source
func (c *Client) FetchFromSource(ctx context.Context, sourceName, topic string, limit int) ([]Doc, error) {
	if c.pluginAPI != nil {
		c.pluginAPI.LogDebug("query initiated", "source", sourceName, "limit", limit)
	}

	sourceConfig, exists := c.sources[sourceName]
	if !exists {
		if c.pluginAPI != nil {
			c.pluginAPI.LogDebug("source not found", "source", sourceName)
		}
		return nil, fmt.Errorf(ErrorSourceNotFound, sourceName)
	}

	canonicalTopic := canonicalizeTopicKey(topic)
	cacheKey := fmt.Sprintf("%s"+CacheKeySeparator+"%s"+CacheKeySeparator+"%d", sourceName, canonicalTopic, limit)
	if cached := c.cache.Get(cacheKey); cached != nil {
		if docs, ok := cached.([]Doc); ok {
			// Ensure we always return a non-nil slice even from cache
			if docs == nil {
				docs = []Doc{}
			}
			if c.pluginAPI != nil {
				c.pluginAPI.LogDebug("cache hit", "source", sourceName, "key", TruncateTopicForLogging(canonicalTopic), "docs", len(docs))
			}
			return docs, nil
		}
	}
	if fuzzyKey := c.findFuzzyCacheKey(sourceName, canonicalTopic, limit); fuzzyKey != "" {
		if cached := c.cache.Get(fuzzyKey); cached != nil {
			if docs, ok := cached.([]Doc); ok {
				if docs == nil {
					docs = []Doc{}
				}
				if c.pluginAPI != nil {
					topicFromKey := strings.TrimSuffix(strings.TrimPrefix(fuzzyKey, fmt.Sprintf("%s"+CacheKeySeparator, sourceName)), fmt.Sprintf(CacheKeySeparator+"%d", limit))
					c.pluginAPI.LogDebug("fuzzy cache hit", "source", sourceName, "key", TruncateTopicForLogging(topicFromKey), "docs", len(docs))
				}
				return docs, nil
			}
		}
	}
	if c.pluginAPI != nil {
		c.pluginAPI.LogDebug("cache miss", "source", sourceName, "key", TruncateTopicForLogging(canonicalTopic))
	}
	protocol, exists := c.protocols[sourceConfig.Protocol]
	if !exists {
		if c.pluginAPI != nil {
			c.pluginAPI.LogWarn("protocol not found", "source", sourceName, "protocol", string(sourceConfig.Protocol))
		}
		return nil, fmt.Errorf(ErrorProtocolNotSupported, sourceConfig.Protocol)
	}

	request := ProtocolRequest{
		Source:   sourceConfig,
		Topic:    topic,
		Sections: sourceConfig.Sections,
		Limit:    limit,
	}

	if limit > sourceConfig.MaxDocsPerCall {
		request.Limit = sourceConfig.MaxDocsPerCall
	}

	protocol.SetAuth(sourceConfig.Auth)

	docs, err := protocol.Fetch(ctx, request)
	if err != nil {
		return nil, fmt.Errorf(ErrorFailedToFetch, sourceName, err)
	}

	// Enforce limit on results (in case protocol doesn't respect the limit)
	if len(docs) > request.Limit {
		docs = docs[:request.Limit]
	}

	// Apply authority-based ranking to improve result quality
	docs = c.rankByAuthority(docs, sourceName)

	// Only cache non-empty results to allow retries for transient failures
	if len(docs) > 0 {
		c.cache.Set(cacheKey, docs)
	}

	// Ensure we always return a non-nil slice
	if docs == nil {
		docs = []Doc{}
	}

	return docs, nil
}

// FetchFromMultipleSources fetches documents from multiple sources for a topic in parallel
func (c *Client) FetchFromMultipleSources(ctx context.Context, sourceNames []string, topic string, limitPerSource int) (map[string][]Doc, error) {
	results := make(map[string][]Doc)
	resultsChan := make(chan sourceResult, len(sourceNames))

	for _, sourceName := range sourceNames {
		go func(source string) {
			select {
			case <-ctx.Done():
				resultsChan <- sourceResult{
					sourceName: source,
					docs:       nil,
					err:        ctx.Err(),
				}
				return
			default:
			}

			docs, err := c.FetchFromSource(ctx, source, topic, limitPerSource)

			select {
			case <-ctx.Done():
				// Context canceled, don't send result (receiver is gone)
				return
			case resultsChan <- sourceResult{
				sourceName: source,
				docs:       docs,
				err:        err,
			}:
			}
		}(sourceName)
	}

	for i := 0; i < len(sourceNames); i++ {
		select {
		case <-ctx.Done():
			// Context canceled, stop collecting results
			close(resultsChan)
			return results, ctx.Err()
		case result := <-resultsChan:
			if result.err != nil {
				if c.pluginAPI != nil && result.err != ctx.Err() {
					c.pluginAPI.LogDebug("source fetch error", "source", result.sourceName, "error", result.err.Error())
				}
				continue
			}
			if len(result.docs) > 0 {
				results[result.sourceName] = result.docs
			}
		}
	}
	close(resultsChan)

	return results, nil
}

type sourceResult struct {
	sourceName string
	docs       []Doc
	err        error
}

// GetAvailableSources returns a list of enabled source names
func (c *Client) GetAvailableSources() []string {
	var sources []string
	for name := range c.sources {
		sources = append(sources, name)
	}
	return sources
}

// IsSourceEnabled checks if a specific source is enabled
func (c *Client) IsSourceEnabled(sourceName string) bool {
	_, exists := c.sources[sourceName]
	return exists
}

// GetSourceConfig returns the configuration for a specific source
func (c *Client) GetSourceConfig(sourceName string) (SourceConfig, bool) {
	config, exists := c.sources[sourceName]
	return config, exists
}

// initializeProtocols sets up the available protocols with required dependencies
func (c *Client) initializeProtocols() {
	// Validate AllowedDomains before creating HTTP client to prevent SSRF
	if len(c.config.AllowedDomains) == 0 {
		if c.pluginAPI != nil {
			c.pluginAPI.LogError("SECURITY: AllowedDomains is empty, using safe defaults to prevent SSRF")
		}
		c.config.AllowedDomains = DefaultAllowedDomains
	}

	for _, domain := range c.config.AllowedDomains {
		if domain == "*" || strings.Contains(domain, "*") {
			if c.pluginAPI != nil {
				c.pluginAPI.LogError("SECURITY: AllowedDomains contains wildcard, replacing with safe defaults to prevent SSRF", "domain", domain)
			}
			c.config.AllowedDomains = DefaultAllowedDomains
			break
		}
	}

	baseHTTPClient := httpexternal.CreateRestrictedClient(nil, c.config.AllowedDomains)

	redirectHTTPClient := &http.Client{
		Transport: baseHTTPClient.Transport,
		Timeout:   DefaultHTTPClientTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Allow up to 10 redirects
			if len(via) >= 10 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	// HTTP Protocol for documentation sites
	c.protocols[HTTPProtocolType] = NewHTTPProtocol(redirectHTTPClient, c.pluginAPI)

	// GitHub Protocol (works with or without token for public repos)
	c.protocols[GitHubAPIProtocolType] = NewGitHubProtocol(c.config.GitHubToken, c.pluginAPI)

	// Mattermost Protocol for Mattermost server instances
	c.protocols[MattermostProtocolType] = NewMattermostProtocol(redirectHTTPClient, c.pluginAPI, c.config.FallbackDirectory)

	// Confluence Protocol for Atlassian Confluence sites
	c.protocols[ConfluenceProtocolType] = NewConfluenceProtocol(redirectHTTPClient, c.pluginAPI)

	// UserVoice Protocol for UserVoice feature request sites
	c.protocols[UserVoiceProtocolType] = NewUserVoiceProtocol(redirectHTTPClient, c.pluginAPI)

	// Discourse Protocol for Discourse forum sites
	c.protocols[DiscourseProtocolType] = NewDiscourseProtocol(redirectHTTPClient, c.pluginAPI)

	// Jira Protocol for Jira issue tracking
	c.protocols[JiraProtocolType] = NewJiraProtocol(redirectHTTPClient, c.pluginAPI)

	// File Protocol for local file-based datasources
	c.protocols[FileProtocolType] = NewFileProtocol(c.pluginAPI)
}

// Close cleans up resources used by the client
func (c *Client) Close() error {
	if c.cache != nil {
		c.cache.Close()
	}

	for _, protocol := range c.protocols {
		if closer, ok := protocol.(interface{ Close() error }); ok {
			closer.Close()
		}
	}

	return nil
}

// GetCacheStats returns basic cache statistics for monitoring
func (c *Client) GetCacheStats() map[string]interface{} {
	return map[string]interface{}{
		"cache_ttl_hours": c.config.CacheTTL.Hours(),
		"enabled_sources": len(c.sources),
		"protocols":       len(c.protocols),
		"config_domains":  len(c.config.AllowedDomains),
	}
}

// canonicalizeTopicKey creates a deterministic representation of a topic
// for use in cache keys. Preserves the original query context while normalizing
// for consistent cache lookups. Does NOT extract keywords or apply synonyms,
// ensuring different queries don't get conflated into the same cache key.
func canonicalizeTopicKey(topic string) string {
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return ""
	}

	// Simple normalization that preserves query structure:
	// 1. Convert to lowercase for case-insensitive matching
	// 2. Normalize whitespace (multiple spaces -> single space)
	// 3. Remove quotes but keep the content
	// 4. Keep punctuation that affects meaning (?, !, -, etc.)

	s := strings.ToLower(topic)

	s = strings.Trim(s, "\"'")

	s = strings.Join(strings.Fields(s), " ")

	replacer := strings.NewReplacer(
		"\n", " ",
		"\t", " ",
		"\r", " ",
	)
	s = replacer.Replace(s)

	s = strings.TrimSpace(s)

	return s
}

// findFuzzyCacheKey attempts to find a similar existing cache key for the same source and limit.
// Uses string similarity on the actual query text rather than keyword-based matching.
func (c *Client) findFuzzyCacheKey(sourceName, canonicalTopic string, limit int) string {
	keys := c.cache.Keys()
	targetPrefix := fmt.Sprintf("%s"+CacheKeySeparator, sourceName)
	targetSuffix := fmt.Sprintf(CacheKeySeparator+"%d", limit)
	bestKey := ""
	bestScore := 0.0

	for _, k := range keys {
		if !strings.HasPrefix(k, targetPrefix) || !strings.HasSuffix(k, targetSuffix) {
			continue
		}
		cachedTopic := strings.TrimSuffix(strings.TrimPrefix(k, targetPrefix), targetSuffix)

		score := stringSimilarity(canonicalTopic, cachedTopic)
		if score > bestScore {
			bestScore = score
			bestKey = k
		}
	}

	// Require very high similarity to avoid false positives with query-based caching
	if bestScore >= 0.9 {
		return bestKey
	}
	return ""
}

// stringSimilarity calculates similarity between two query strings using a simple
// word-based approach. Returns 1.0 for identical queries, 0.0 for completely different ones.
func stringSimilarity(a, b string) float64 {
	if a == b {
		return 1.0
	}
	if a == "" || b == "" {
		return 0.0
	}

	wordsA := strings.Fields(a)
	wordsB := strings.Fields(b)

	if len(wordsA) == 0 && len(wordsB) == 0 {
		return 1.0
	}
	if len(wordsA) == 0 || len(wordsB) == 0 {
		return 0.0
	}

	setA := map[string]struct{}{}
	setB := map[string]struct{}{}

	for _, word := range wordsA {
		setA[word] = struct{}{}
	}
	for _, word := range wordsB {
		setB[word] = struct{}{}
	}

	intersection := 0
	union := len(setA)

	for word := range setB {
		if _, exists := setA[word]; exists {
			intersection++
		} else {
			union++
		}
	}

	if union == 0 {
		return 0.0
	}

	return float64(intersection) / float64(union)
}

// rankByAuthority applies authority-based scoring and ranking to improve result quality
func (c *Client) rankByAuthority(docs []Doc, sourceName string) []Doc {
	if len(docs) == 0 {
		return docs
	}

	authorityMap := map[string]int{
		"docs.mattermost.com":      100, // Official docs - highest authority
		"mattermost.atlassian.net": 80,  // Internal Confluence and Jira - high authority
		"github.com/mattermost":    60,  // Official GitHub - medium-high authority
		"community.mattermost.com": 40,  // Community forum - medium authority
		"mattermost.com":           30,  // Marketing site - lower authority
	}

	for i := range docs {
		docs[i].AuthorityScore = c.getSourceAuthority(docs[i].URL, authorityMap)
	}

	sort.Slice(docs, func(i, j int) bool {
		return docs[i].AuthorityScore > docs[j].AuthorityScore
	})

	return docs
}

// getSourceAuthority determines the authority score for a given URL
func (c *Client) getSourceAuthority(docURL string, authorityMap map[string]int) int {
	if docURL == "" {
		return 10 // Default low score for empty URLs
	}

	parsedURL, err := url.Parse(docURL)
	if err != nil {
		return 10 // Default low score for malformed URLs
	}

	hostname := strings.ToLower(parsedURL.Host)

	if score, exists := authorityMap[hostname]; exists {
		return score
	}

	// Check for subdomain matches
	for domain, score := range authorityMap {
		// Skip domains with paths - they're handled separately
		if strings.Contains(domain, "/") {
			continue
		}
		if strings.HasSuffix(hostname, "."+domain) {
			return score
		}
	}

	// Check for path-based domain matches (e.g., github.com/mattermost)
	for domain, score := range authorityMap {
		if !strings.Contains(domain, "/") {
			continue
		}
		// Split domain and path
		parts := strings.SplitN(domain, "/", 2)
		if len(parts) != 2 {
			continue
		}
		domainPart := parts[0]
		pathPart := parts[1]

		// Check if hostname matches AND URL path starts with expected path
		if hostname == domainPart && strings.HasPrefix(parsedURL.Path, "/"+pathPart) {
			return score
		}
	}

	// Default score for unknown domains
	return 20
}

// truncateTopicForLogging creates a shortened version of a topic for logging
// Extracts first few keywords to avoid overwhelming logs with long boolean queries
func truncateTopicForLogging(topic string, maxTerms int) string {
	if len(topic) <= 80 {
		return topic
	}

	// Extract keywords using boolean query parser
	queryNode, err := ParseBooleanQuery(topic)
	if err != nil {
		// Fallback: just truncate
		if len(topic) > 80 {
			return topic[:77] + "..."
		}
		return topic
	}

	keywords := ExtractKeywords(queryNode)
	if len(keywords) == 0 {
		return topic[:77] + "..."
	}

	if len(keywords) > maxTerms {
		keywords = keywords[:maxTerms]
	}

	return strings.Join(keywords, ", ") + "..."
}

// TruncateTopicForLogging is the exported version for use by protocol implementations
func TruncateTopicForLogging(topic string) string {
	return truncateTopicForLogging(topic, 5)
}
