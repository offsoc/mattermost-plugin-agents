// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	pm "github.com/mattermost/mattermost-plugin-ai/datasources/segments/pm"

	"github.com/mattermost/mattermost-plugin-ai/mmapi"
)

// DiscourseProtocol implements the DataSourceProtocol for Discourse forums
type DiscourseProtocol struct {
	client      *http.Client
	auth        AuthConfig
	rateLimiter *RateLimiter
	pluginAPI   mmapi.Client
}

// DiscourseTopic represents a Discourse topic from API response
type DiscourseTopic struct {
	ID           int      `json:"id"`
	Title        string   `json:"title"`
	Slug         string   `json:"slug"`
	CategoryID   int      `json:"category_id"`
	PostsCount   int      `json:"posts_count"`
	CreatedAt    string   `json:"created_at"`
	LastPostedAt string   `json:"last_posted_at"`
	Views        int      `json:"views"`
	LikeCount    int      `json:"like_count"`
	Tags         []string `json:"tags"`
}

// DiscoursePost represents a Discourse post from API response
type DiscoursePost struct {
	ID         int    `json:"id"`
	Username   string `json:"username"`
	CreatedAt  string `json:"created_at"`
	Cooked     string `json:"cooked"`
	PostNumber int    `json:"post_number"`
	PostType   int    `json:"post_type"`
	UpdatedAt  string `json:"updated_at"`
	LikeCount  int    `json:"like_count"`
	TopicID    int    `json:"topic_id"`
	TopicSlug  string `json:"topic_slug"`
}

// DiscourseSearchResult represents search results from Discourse API
type DiscourseSearchResult struct {
	Topics []DiscourseTopic `json:"topics"`
	Posts  []DiscoursePost  `json:"posts"`
}

// NewDiscourseProtocol creates a new Discourse protocol instance
func NewDiscourseProtocol(httpClient *http.Client, pluginAPI mmapi.Client) *DiscourseProtocol {
	return &DiscourseProtocol{
		client:    httpClient,
		auth:      AuthConfig{Type: AuthTypeNone},
		pluginAPI: pluginAPI,
	}
}

// Fetch retrieves documents from Discourse community forums
func (d *DiscourseProtocol) Fetch(ctx context.Context, request ProtocolRequest) ([]Doc, error) {
	source := request.Source

	EnsureRateLimiter(&d.rateLimiter, source.RateLimit)

	baseURL := source.Endpoints["base_url"]
	if baseURL == "" {
		return nil, fmt.Errorf("missing required Discourse configuration: base_url")
	}

	var allDocs []Doc
	for _, section := range request.Sections {
		if len(allDocs) >= request.Limit {
			break
		}

		categoryDocs := d.fetchFromCategory(ctx, baseURL, section, request.Topic, request.Limit-len(allDocs), source.Name)
		allDocs = append(allDocs, categoryDocs...)
	}

	return allDocs, nil
}

// GetType returns the protocol type
func (d *DiscourseProtocol) GetType() ProtocolType {
	return DiscourseProtocolType
}

// SetAuth configures authentication for the protocol
func (d *DiscourseProtocol) SetAuth(auth AuthConfig) {
	d.auth = auth
}

// Close cleans up resources used by the protocol
func (d *DiscourseProtocol) Close() error {
	CloseRateLimiter(&d.rateLimiter)
	return nil
}

// fetchFromCategory retrieves documents from a specific Discourse category
func (d *DiscourseProtocol) fetchFromCategory(ctx context.Context, baseURL, category, topic string, limit int, sourceName string) []Doc {
	searchDocs := d.searchInCategory(ctx, baseURL, category, topic, limit, sourceName)
	if len(searchDocs) > 0 {
		return FilterDocsByBooleanQuery(searchDocs, topic)
	}

	// Fallback to browsing category topics
	browseDocs := d.browseCategory(ctx, baseURL, category, topic, limit, sourceName)
	return FilterDocsByBooleanQuery(browseDocs, topic)
}

// searchInCategory searches for topics in a specific category using Discourse search API
func (d *DiscourseProtocol) searchInCategory(ctx context.Context, baseURL, category, topic string, limit int, sourceName string) []Doc {
	searchURL := BuildAPIURL(baseURL, "search.json")

	searchTopic := SimplifyBooleanQueryToKeywords(topic)

	query := fmt.Sprintf("category:%s %s", category, searchTopic)

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		if d.pluginAPI != nil {
			d.pluginAPI.LogWarn(sourceName+": request create failed", "category", category, "error", err.Error())
		}
		return nil
	}

	queryParams := req.URL.Query()
	queryParams.Add("q", query)
	queryParams.Add("max_results", fmt.Sprintf("%d", limit))
	req.URL.RawQuery = queryParams.Encode()

	d.addAuthHeaders(req)

	if waitErr := WaitRateLimiter(ctx, d.rateLimiter); waitErr != nil {
		if d.pluginAPI != nil {
			d.pluginAPI.LogWarn(sourceName+": rate limit error", "category", category, "error", waitErr.Error())
		}
		return nil
	}

	resp, err := d.client.Do(req)
	if err != nil {
		if d.pluginAPI != nil {
			d.pluginAPI.LogWarn(sourceName+": API request failed", "category", category, "error", err.Error())
		}
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		if d.pluginAPI != nil {
			d.pluginAPI.LogWarn(sourceName+": API error", "category", category, "status", resp.StatusCode)
		}
		return nil
	}

	var searchResult DiscourseSearchResult
	if err := json.NewDecoder(resp.Body).Decode(&searchResult); err != nil {
		if d.pluginAPI != nil {
			d.pluginAPI.LogWarn(sourceName+": parse error", "category", category, "error", err.Error())
		}
		return nil
	}

	docs := d.convertSearchResultsToDocs(searchResult, baseURL, category, sourceName)

	return docs
}

// browseCategory fetches latest topics from a category
func (d *DiscourseProtocol) browseCategory(ctx context.Context, baseURL, category, topic string, limit int, sourceName string) []Doc {
	// Fetch latest topics from the category using the category latest endpoint
	categoryURL := BuildAPIURL(baseURL, fmt.Sprintf("c/%s.json", category))

	req, err := http.NewRequestWithContext(ctx, "GET", categoryURL, nil)
	if err != nil {
		if d.pluginAPI != nil {
			d.pluginAPI.LogWarn(sourceName+": request create failed", "category", category, "error", err.Error())
		}
		return nil
	}

	d.addAuthHeaders(req)

	if waitErr := WaitRateLimiter(ctx, d.rateLimiter); waitErr != nil {
		if d.pluginAPI != nil {
			d.pluginAPI.LogWarn(sourceName+": rate limit error", "category", category, "error", waitErr.Error())
		}
		return nil
	}

	resp, err := d.client.Do(req)
	if err != nil {
		if d.pluginAPI != nil {
			d.pluginAPI.LogWarn(sourceName+": API request failed", "category", category, "error", err.Error())
		}
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		if d.pluginAPI != nil {
			d.pluginAPI.LogWarn(sourceName+": API error", "category", category, "status", resp.StatusCode)
		}
		return nil
	}

	var categoryData struct {
		TopicList struct {
			Topics []DiscourseTopic `json:"topics"`
		} `json:"topic_list"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&categoryData); err != nil {
		if d.pluginAPI != nil {
			d.pluginAPI.LogWarn(sourceName+": parse error", "category", category, "error", err.Error())
		}
		return nil
	}

	searchResult := DiscourseSearchResult{
		Topics: categoryData.TopicList.Topics,
	}

	docs := d.convertSearchResultsToDocs(searchResult, baseURL, category, sourceName)

	if topic != "" {
		var filteredDocs []Doc
		for _, doc := range docs {
			if strings.Contains(strings.ToLower(doc.Title), strings.ToLower(topic)) ||
				strings.Contains(strings.ToLower(doc.Content), strings.ToLower(topic)) {
				filteredDocs = append(filteredDocs, doc)
				if len(filteredDocs) >= limit {
					break
				}
			}
		}
		return filteredDocs
	}

	if len(docs) > limit {
		docs = docs[:limit]
	}

	return docs
}

// convertSearchResultsToDocs converts Discourse search results to Doc format with metadata
func (d *DiscourseProtocol) convertSearchResultsToDocs(searchResult DiscourseSearchResult, baseURL, category, sourceName string) []Doc {
	var docs []Doc

	for _, discTopic := range searchResult.Topics {
		content := d.formatTopicContent(discTopic)

		meta := extractDiscourseMetadata(discTopic)

		labels := []string{category}
		labels = append(labels, buildLabelsFromMetadata(meta)...)

		if discTopic.LikeCount >= 20 {
			labels = append(labels, "high_engagement")
		}

		if discTopic.Views >= 500 {
			labels = append(labels, "high_visibility")
		}

		labels = append(labels, fmt.Sprintf("likes:%d", discTopic.LikeCount))
		labels = append(labels, fmt.Sprintf("views:%d", discTopic.Views))
		labels = append(labels, fmt.Sprintf("posts:%d", discTopic.PostsCount))

		for _, tag := range discTopic.Tags {
			labels = append(labels, fmt.Sprintf("tag:%s", tag))
		}

		daysCreated := DaysSince(discTopic.CreatedAt)
		if recencyLabel := FormatRecencyLabel(daysCreated); recencyLabel != "" {
			labels = append(labels, recencyLabel+"_created")
		}
		daysUpdated := DaysSince(discTopic.LastPostedAt)
		if recencyLabel := FormatRecencyLabel(daysUpdated); recencyLabel != "" {
			labels = append(labels, recencyLabel+"_updated")
		}

		// Map to appropriate section based on priority and category
		section := category
		priority := "none"
		if meta.RoleMetadata != nil {
			priority = meta.RoleMetadata.GetPriority()
		}
		if priority == "high" && category == SectionBugs {
			section = SectionCritical
		} else if strings.Contains(strings.ToLower(discTopic.Title), "bug") {
			section = SectionBugs
		}

		doc := Doc{
			Title:        discTopic.Title,
			Content:      content,
			URL:          BuildAPIURL(baseURL, fmt.Sprintf("t/%s/%d", discTopic.Slug, discTopic.ID)),
			Section:      section,
			Source:       sourceName,
			Labels:       labels,
			CreatedDate:  discTopic.CreatedAt,
			LastModified: discTopic.LastPostedAt,
		}
		docs = append(docs, doc)
	}

	for _, post := range searchResult.Posts {
		if post.PostNumber == 1 {
			continue // Skip first posts as they're usually covered by topics
		}

		content := d.formatPostContent(post)

		labels := []string{category}

		if post.LikeCount >= 10 {
			labels = append(labels, "high_engagement")
		}

		labels = append(labels, fmt.Sprintf("likes:%d", post.LikeCount))

		daysCreated := DaysSince(post.CreatedAt)
		if recencyLabel := FormatRecencyLabel(daysCreated); recencyLabel != "" {
			labels = append(labels, recencyLabel+"_created")
		}
		daysUpdated := DaysSince(post.UpdatedAt)
		if recencyLabel := FormatRecencyLabel(daysUpdated); recencyLabel != "" {
			labels = append(labels, recencyLabel+"_updated")
		}

		doc := Doc{
			Title:        fmt.Sprintf("Post in %s", post.TopicSlug),
			Content:      content,
			URL:          BuildAPIURL(baseURL, fmt.Sprintf("t/%s/%d/%d", post.TopicSlug, post.TopicID, post.PostNumber)),
			Section:      category,
			Source:       sourceName,
			Labels:       labels,
			Author:       post.Username,
			CreatedDate:  post.CreatedAt,
			LastModified: post.UpdatedAt,
		}
		docs = append(docs, doc)
	}

	return docs
}

// addAuthHeaders adds Discourse API authentication headers
func (d *DiscourseProtocol) addAuthHeaders(req *http.Request) {
	switch d.auth.Type {
	case "api_key":
		if d.auth.Key != "" {
			req.Header.Set("Api-Key", d.auth.Key)
		}
	case "token":
		if d.auth.Key != "" {
			req.Header.Set(HeaderAuthorization, AuthPrefixBearer+d.auth.Key)
		}
	}
	req.Header.Set(HeaderUserAgent, UserAgentMattermostPM)
}

// inferDiscourseTopicPriority infers priority from engagement signals
func inferDiscourseTopicPriority(topic DiscourseTopic) pm.Priority {
	score := 0

	// High engagement signals from likes
	switch {
	case topic.LikeCount >= 20:
		score += 3
	case topic.LikeCount >= 10:
		score += 2
	case topic.LikeCount >= 5:
		score++
	}

	// High visibility from views
	switch {
	case topic.Views >= 500:
		score += 3
	case topic.Views >= 200:
		score += 2
	case topic.Views >= 100:
		score++
	}

	// Active discussion from post count
	if topic.PostsCount >= 30 {
		score += 2
	} else if topic.PostsCount >= 10 {
		score++
	}

	// Recent activity bonus
	lastPosted, err := time.Parse(time.RFC3339, topic.LastPostedAt)
	if err == nil {
		daysSinceActivity := time.Since(lastPosted).Hours() / 24
		if daysSinceActivity <= 7 {
			score += 2
		} else if daysSinceActivity <= 30 {
			score++
		}
	}

	// Score thresholds
	if score >= 7 {
		return pm.PriorityHigh
	} else if score >= 4 {
		return pm.PriorityMedium
	}
	return pm.PriorityLow
}

// formatTopicContent formats a Discourse topic for consumption with metadata
func (d *DiscourseProtocol) formatTopicContent(topic DiscourseTopic) string {
	meta := extractDiscourseMetadata(topic)
	content := formatEntityMetadata(meta)

	if content != "" {
		content += fmt.Sprintf("Community Engagement: %d posts, %d views, %d likes\n", topic.PostsCount, topic.Views, topic.LikeCount)
		content += "---\n"
	}

	content += fmt.Sprintf("Topic: %s\n", topic.Title)
	content += fmt.Sprintf("Posts: %d\n", topic.PostsCount)
	content += fmt.Sprintf("Views: %d\n", topic.Views)
	content += fmt.Sprintf("Likes: %d\n", topic.LikeCount)
	content += fmt.Sprintf("Created: %s\n", d.formatDate(topic.CreatedAt))
	content += fmt.Sprintf("Last Posted: %s\n", d.formatDate(topic.LastPostedAt))

	if len(topic.Tags) > 0 {
		content += fmt.Sprintf("Tags: %s\n", strings.Join(topic.Tags, ", "))
	}

	content += "\n"
	content += fmt.Sprintf("Community discussion about %s with %d posts and %d views.", topic.Title, topic.PostsCount, topic.Views)

	return content
}

// formatPostContent formats a Discourse post for consumption
func (d *DiscourseProtocol) formatPostContent(post DiscoursePost) string {
	content := fmt.Sprintf("Post #%d by %s\n", post.PostNumber, post.Username)
	content += fmt.Sprintf("Created: %s\n", d.formatDate(post.CreatedAt))
	content += fmt.Sprintf("Likes: %d\n", post.LikeCount)

	content += "\n"
	if post.Cooked != "" {
		cleaned := d.stripHTML(post.Cooked)
		content += cleaned
	}

	return content
}

// formatDate formats a date string for display
func (d *DiscourseProtocol) formatDate(dateStr string) string {
	if dateStr == "" {
		return "Unknown"
	}

	parsedTime, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		return dateStr
	}

	return parsedTime.Format("2006-01-02 15:04")
}

// stripHTML removes HTML tags from content with enhanced cleaning
func (d *DiscourseProtocol) stripHTML(htmlContent string) string {
	content := htmlContent

	content = d.removeScriptAndStyle(content)

	content = strings.ReplaceAll(content, "<p>", "\n")
	content = strings.ReplaceAll(content, "</p>", "\n")
	content = strings.ReplaceAll(content, "<br>", "\n")
	content = strings.ReplaceAll(content, "<br/>", "\n")
	content = strings.ReplaceAll(content, "<br />", "\n")

	content = d.removeTagsWithRegex(content, []string{"div", "span", "strong", "em", "a", "img", "code", "pre"})

	content = d.removeInlineCSSPatterns(content)

	// Clean up multiple newlines
	content = strings.ReplaceAll(content, "\n\n\n", "\n\n")
	content = strings.TrimSpace(content)

	return content
}

// removeScriptAndStyle removes script, style, and noscript elements
func (d *DiscourseProtocol) removeScriptAndStyle(content string) string {
	scriptRe := regexp.MustCompile(`(?is)<script[^>]*?>.*?</script>`)
	content = scriptRe.ReplaceAllString(content, "")

	styleRe := regexp.MustCompile(`(?is)<style[^>]*?>.*?</style>`)
	content = styleRe.ReplaceAllString(content, "")

	noscriptRe := regexp.MustCompile(`(?is)<noscript[^>]*?>.*?</noscript>`)
	content = noscriptRe.ReplaceAllString(content, "")

	return content
}

// removeTagsWithRegex removes specified HTML tags using regex (handles attributes efficiently)
func (d *DiscourseProtocol) removeTagsWithRegex(content string, tags []string) string {
	for _, tag := range tags {
		// Matches both <tag> and <tag attr="val"> efficiently
		re := regexp.MustCompile(fmt.Sprintf(`(?i)</?%s(?:\s+[^>]*)?>`, tag))
		content = re.ReplaceAllString(content, "")
	}
	return content
}

// removeInlineCSSPatterns removes CSS patterns that remain after HTML tag removal
func (d *DiscourseProtocol) removeInlineCSSPatterns(text string) string {
	cssVarPattern := regexp.MustCompile(`--[a-zA-Z-]+:\s*[^;}\n]+[;}]?`)
	text = cssVarPattern.ReplaceAllString(text, "")

	cssBlockPattern := regexp.MustCompile(`(?i)\s*[a-zA-Z-]+\s*\{\s*[^}]*\}`)
	text = cssBlockPattern.ReplaceAllString(text, "")

	cssPropPattern := regexp.MustCompile(`[a-zA-Z-]+:\s*[^;\n}]+[;}]?`)
	text = cssPropPattern.ReplaceAllString(text, "")

	cssPatternWithBraces := regexp.MustCompile(`\s*\{[^}]*\}`)
	text = cssPatternWithBraces.ReplaceAllString(text, "")

	return text
}

// ValidateSearchSyntax validates search syntax by testing real queries against the Discourse API
func (d *DiscourseProtocol) ValidateSearchSyntax(ctx context.Context, request ProtocolRequest) (*SyntaxValidationResult, error) {
	result := &SyntaxValidationResult{
		OriginalQuery:    request.Topic,
		SupportsFeatures: []string{"simple_terms", "quotes", "basic_OR"},
	}

	baseURL := request.Source.Endpoints["base_url"]
	if baseURL == "" {
		result.IsValidSyntax = false
		result.SyntaxErrors = append(result.SyntaxErrors, "missing base_url configuration")
		return result, nil
	}

	testQueries := []struct {
		query       string
		description string
	}{
		{request.Topic, "original query"},
		{"mobile", "simple term"},
		{"mobile OR ios", "simple OR query"},
		{strings.Split(request.Topic, " AND ")[0], "first part of AND query"},
	}

	var successfulQueries []string
	var failedQueries []string

	for _, test := range testQueries {
		if test.query == "" {
			continue
		}

		testSection := "announcements"
		if len(request.Sections) > 0 {
			testSection = request.Sections[0]
		}

		if d.pluginAPI != nil {
			d.pluginAPI.LogDebug("Testing Discourse query syntax", "query", test.query, "description", test.description)
		}

		docs := d.searchInCategory(ctx, baseURL, testSection, test.query, 1, request.Source.Name)

		if len(docs) > 0 {
			successfulQueries = append(successfulQueries, test.query)
			if test.description == "original query" {
				result.TestResultCount = len(docs)
			}
		} else {
			failedQueries = append(failedQueries, test.query)
		}
	}

	// Analyze results and provide recommendations
	originalFailed := false
	for _, failed := range failedQueries {
		if failed == request.Topic {
			originalFailed = true
			break
		}
	}

	switch {
	case len(successfulQueries) == 0:
		result.IsValidSyntax = false
		result.SyntaxErrors = append(result.SyntaxErrors, "all test queries returned 0 results")
		result.RecommendedQuery = "mobile" // Safe fallback
	case originalFailed:
		// Original query failed but simpler queries worked - this indicates syntax issues
		result.IsValidSyntax = false
		result.SyntaxErrors = append(result.SyntaxErrors, "complex query returned 0 results but simpler queries returned results")

		// Find the simplest working query as recommendation
		for _, success := range successfulQueries {
			if strings.Contains(success, "OR") {
				continue // Skip OR queries for simplicity
			}
			result.RecommendedQuery = success
			break
		}
		if result.RecommendedQuery == "" && len(successfulQueries) > 0 {
			result.RecommendedQuery = successfulQueries[0]
		}
	default:
		// Original query worked
		result.IsValidSyntax = true
		result.RecommendedQuery = request.Topic
	}

	// Check what features are actually supported
	if strings.Contains(request.Topic, "AND") && len(successfulQueries) == 0 {
		result.SyntaxErrors = append(result.SyntaxErrors, "AND operator appears to return 0 results")
	}

	if d.pluginAPI != nil {
		d.pluginAPI.LogDebug("Discourse syntax validation completed",
			"original_query", result.OriginalQuery,
			"is_valid", result.IsValidSyntax,
			"recommended_query", result.RecommendedQuery,
			"successful_queries", successfulQueries,
			"failed_queries", failedQueries)
	}

	return result, nil
}
