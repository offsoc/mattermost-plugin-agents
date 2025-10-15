// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"fmt"
	"strings"
)

// RelevanceScore represents a comprehensive relevance assessment with multiple quality dimensions
type RelevanceScore struct {
	Total             int  // 0-100 total score
	ContentQuality    int  // 0-40 topic-agnostic content quality
	SemanticRelevance int  // 0-40 topic-aware semantic relevance
	SourceAuthority   int  // 0-20 source credibility and freshness
	PassesFilter      bool // true if score meets minimum threshold
}

// UniversalRelevanceScorer provides topic-agnostic quality filtering and universal relevance scoring
type UniversalRelevanceScorer struct {
	topicAnalyzer *TopicAnalyzer
}

// NewUniversalRelevanceScorer creates a new universal relevance scorer
func NewUniversalRelevanceScorer() *UniversalRelevanceScorer {
	return &UniversalRelevanceScorer{
		topicAnalyzer: NewTopicAnalyzer(),
	}
}

// IsUniversallyAcceptableWithReason provides detailed rejection reasons for debugging
func (u *UniversalRelevanceScorer) IsUniversallyAcceptableWithReason(content, title, source, topic string) (bool, string) {
	if content == "" {
		return false, "empty_content"
	}

	// Basic length check
	contentLength := len(strings.TrimSpace(content))
	if contentLength < MinContentLength {
		return false, fmt.Sprintf("content_too_short: %d chars < %d minimum", contentLength, MinContentLength)
	}

	// Enhanced content quality check
	if reason := u.getPlainTextQualityRejectionReason(content, title); reason != "" {
		return false, "quality_check: " + reason
	}

	// Topic relevance check when topic is provided
	if topic != "" {
		switch source {
		case SourceMattermostDocs:
			keywords := u.topicAnalyzer.ExtractTopicKeywords(topic)
			score := u.topicAnalyzer.ScoreContentRelevanceWithTitle(content, keywords, title)
			if score <= 0 {
				return false, fmt.Sprintf("topic_relevance: score=%d, keywords=%v, no matches found", score, keywords)
			}
		case SourceConfluenceDocs, SourceCommunityForum, SourceJiraDocs, SourceGitHubRepos:
			// Confluence docs, Community Forum, Jira, and GitHub get a pass on strict topic relevance
			// These sources are already pre-filtered by their search APIs with JQL/GitHub query syntax
		default:
			keywords := u.topicAnalyzer.ExtractTopicKeywords(topic)
			score := u.topicAnalyzer.ScoreContentRelevanceWithTitle(content, keywords, title)
			threshold := u.topicAnalyzer.getMinimumThreshold(topic)
			if score < threshold {
				return false, fmt.Sprintf("topic_relevance: score=%d < threshold=%d, keywords=%v", score, threshold, keywords)
			}
		}
	}

	// Source-specific quality thresholds
	if reason := u.getSourceQualityRejectionReason(content, source); reason != "" {
		return false, "source_quality: " + reason
	}

	return true, ""
}

// IsUniversallyAcceptable provides standardized content quality filtering across all protocols
// Reuses existing HTMLProcessor and TopicAnalyzer logic while addressing obvious gaps
func (u *UniversalRelevanceScorer) IsUniversallyAcceptable(content, title, source, topic string) bool {
	accepted, _ := u.IsUniversallyAcceptableWithReason(content, title, source, topic)
	return accepted
}

// getPlainTextQualityRejectionReason returns a reason if content fails quality checks
func (u *UniversalRelevanceScorer) getPlainTextQualityRejectionReason(content, title string) string {
	contentLength := len(strings.TrimSpace(content))

	// Content should contain alphabetic characters
	hasAlphabetic := false
	for _, r := range content {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
			hasAlphabetic = true
			break
		}
	}
	if !hasAlphabetic {
		return "no_alphabetic_characters"
	}

	if contentLength < LongContentThreshold {
		if u.containsCSSJavaScriptContent(content) {
			return "css_javascript_content"
		}

		if u.containsPromotionalContent(content, title) {
			return "promotional_content"
		}
	}

	if u.isErrorOrEmptyPage(content, title) {
		return "error_or_empty_page"
	}

	// Detect navigation-heavy content
	navKeywordCount := 0
	navKeywords := []string{"next", "previous", "home", "menu", "search", "login", "register", "toggle", "sidebar", "navigation", "breadcrumb"}
	lowerContent := strings.ToLower(content)
	for _, keyword := range navKeywords {
		navKeywordCount += strings.Count(lowerContent, keyword)
	}
	if float64(navKeywordCount) > float64(contentLength)*MaxNavigationKeywordRatio {
		return fmt.Sprintf("navigation_heavy: %d nav keywords in %d chars", navKeywordCount, contentLength)
	}

	// Title quality checks
	lowerTitle := strings.ToLower(title)
	if lowerTitle == "home" || lowerTitle == "index" || strings.Contains(lowerTitle, "404") || strings.Contains(lowerTitle, "error") {
		return fmt.Sprintf("bad_title: '%s'", title)
	}

	return ""
}

// getSourceQualityRejectionReason returns a reason if content fails source-specific standards
func (u *UniversalRelevanceScorer) getSourceQualityRejectionReason(content, source string) string {
	contentLength := len(strings.TrimSpace(content))

	// Higher standards for official documentation
	if source == SourceMattermostDocs || source == SourceConfluenceDocs {
		if contentLength < 100 {
			return fmt.Sprintf("docs_too_short: %d chars < 100 minimum for docs", contentLength)
		}
		return ""
	}

	// GitHub content should be substantial
	if source == SourceGitHubRepos {
		if contentLength < MinContentLength {
			return fmt.Sprintf("github_too_short: %d chars", contentLength)
		}
		if strings.Contains(strings.ToLower(content), "todo") {
			return "github_todo_content"
		}
		return ""
	}

	// Community content can be shorter but should be meaningful
	if source == SourceCommunityForum || source == SourceMattermostHub {
		if contentLength < 60 {
			return fmt.Sprintf("community_too_short: %d chars < 60 minimum", contentLength)
		}
		return ""
	}

	// Default standard
	if contentLength < MinContentLength {
		return fmt.Sprintf("default_too_short: %d chars < %d minimum", contentLength, MinContentLength)
	}
	return ""
}

// isPlainTextQualityAcceptable adapts HTMLProcessor logic to work on plain text
// Reuses existing navigation detection and quality patterns
func (u *UniversalRelevanceScorer) isPlainTextQualityAcceptable(content, title string) bool {
	contentLength := len(strings.TrimSpace(content))

	// Content should contain alphabetic characters
	hasAlphabetic := false
	for _, r := range content {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
			hasAlphabetic = true
			break
		}
	}
	if !hasAlphabetic {
		return false
	}

	// Only apply stricter content pattern checks to shorter content
	// Long-form documentation is less likely to be promotional and may contain code snippets
	if contentLength < LongContentThreshold {
		if u.containsCSSJavaScriptContent(content) {
			return false
		}

		if u.containsPromotionalContent(content, title) {
			return false
		}
	}

	if u.isErrorOrEmptyPage(content, title) {
		return false
	}

	// Detect navigation-heavy content (reuses HTMLProcessor navigation detection)
	navKeywordCount := 0
	navKeywords := []string{"next", "previous", "home", "menu", "search", "login", "register", "toggle", "sidebar", "navigation", "breadcrumb"}
	lowerContent := strings.ToLower(content)
	for _, keyword := range navKeywords {
		navKeywordCount += strings.Count(lowerContent, keyword)
	}
	if float64(navKeywordCount) > float64(contentLength)*MaxNavigationKeywordRatio {
		return false
	}

	// Title quality checks (reuses HTMLProcessor title logic)
	lowerTitle := strings.ToLower(title)
	if lowerTitle == "home" || lowerTitle == "index" || strings.Contains(lowerTitle, "404") || strings.Contains(lowerTitle, "error") {
		return false
	}

	return true
}

// containsCSSJavaScriptContent detects if content contains CSS or JavaScript patterns
func (u *UniversalRelevanceScorer) containsCSSJavaScriptContent(content string) bool {
	lowerContent := strings.ToLower(content)

	// CSS patterns
	cssPatterns := []string{
		"--font-stack:", "--primary-", "rgba(", "var(--",
		"@media", "@import", "@keyframes",
		"background:", "margin:", "padding:", "border:",
		"px;", "rem;", "em;", "%;",
		"{", "}",
	}

	// JavaScript patterns
	jsPatterns := []string{
		"window.datalayer", "gtag(", "ga(", "google_tag",
		"document.addeventlistener", "document.cookie",
		"console.log", "settimeout(", "queryselector",
		"var ", "let ", "const ", "function(",
		"typeof ", "===", "!==",
	}

	// Count CSS patterns
	cssCount := 0
	for _, pattern := range cssPatterns {
		cssCount += strings.Count(lowerContent, pattern)
	}

	// Count JavaScript patterns
	jsCount := 0
	for _, pattern := range jsPatterns {
		jsCount += strings.Count(lowerContent, pattern)
	}

	contentLength := len(content)
	if contentLength == 0 {
		return false
	}

	// If more than 10% of content length consists of CSS/JS patterns, reject
	totalPatternCount := cssCount + jsCount
	if totalPatternCount > contentLength/10 {
		return true
	}

	// Also check for high density of CSS/JS symbols
	symbolCount := 0
	for _, char := range content {
		if char == '{' || char == '}' || char == ':' || char == ';' {
			symbolCount++
		}
	}

	// If more than 8% are CSS/JS symbols, likely code
	if float64(symbolCount)/float64(contentLength) > 0.08 {
		return true
	}

	return false
}

// meetsSourceQualityStandards applies source-specific quality requirements
// Addresses the gap where different protocols need different standards
func (u *UniversalRelevanceScorer) meetsSourceQualityStandards(content, source string) bool {
	contentLength := len(strings.TrimSpace(content))

	// Higher standards for official documentation
	if source == SourceMattermostDocs || source == SourceConfluenceDocs {
		return contentLength >= 100 // Slightly higher bar for docs
	}

	// GitHub content should be substantial
	if source == SourceGitHubRepos {
		return contentLength >= MinContentLength && !strings.Contains(strings.ToLower(content), "todo")
	}

	// Community content can be shorter but should be meaningful
	if source == SourceCommunityForum || source == SourceMattermostHub {
		return contentLength >= 60 // Slightly lower for community content
	}

	// Default standard
	return contentLength >= MinContentLength
}

// containsPromotionalContent detects promotional/marketing content that should be filtered
func (u *UniversalRelevanceScorer) containsPromotionalContent(content, title string) bool {
	lowerContent := strings.ToLower(content)
	lowerTitle := strings.ToLower(title)

	// Promotional patterns found in current noisy output
	promoPatterns := []string{
		"hacktoberfest", "contribute, collaborate & earn rewards", "contribute.*collaborate.*earn rewards",
		"subscribe", "sign up", "contact sales", "free trial", "get started free",
		"cookie policy", "privacy policy", "terms of service", "newsletter",
		"buy now", "purchase", "pricing plans", "upgrade now", "start free",
		"limited time", "special offer", "discount", "promo code",
	}

	for _, pattern := range promoPatterns {
		if strings.Contains(lowerContent, pattern) || strings.Contains(lowerTitle, pattern) {
			return true
		}
	}

	promoKeywords := []string{"subscribe", "buy", "purchase", "sale", "offer", "deal", "discount", "free", "trial"}
	promoCount := 0
	contentWords := strings.Fields(lowerContent)

	for _, word := range contentWords {
		for _, keyword := range promoKeywords {
			if strings.Contains(word, keyword) {
				promoCount++
				break
			}
		}
	}

	// If more than 5% of words are promotional, consider it promotional content
	if len(contentWords) > 0 && float64(promoCount)/float64(len(contentWords)) > 0.05 {
		return true
	}

	return false
}

// isErrorOrEmptyPage detects error pages and empty content that should be filtered
func (u *UniversalRelevanceScorer) isErrorOrEmptyPage(content, title string) bool {
	lowerContent := strings.ToLower(content)
	lowerTitle := strings.ToLower(title)
	contentLength := len(strings.TrimSpace(content))

	// For short content, be strict about error patterns
	// For long content (documentation), only reject if error patterns dominate
	strictThreshold := 500 // Short content threshold

	// Error page indicators
	errorPatterns := []string{
		"404", "not found", "page not found", "file not found",
		"403", "access denied", "forbidden", "unauthorized",
		"500", "internal server error", "server error",
		"maintenance", "under construction", "coming soon",
		"loading...", "please wait", "redirecting",
	}

	// Count how many error patterns match
	errorMatchCount := 0
	for _, pattern := range errorPatterns {
		if strings.Contains(lowerContent, pattern) {
			errorMatchCount++
		}
		// Title matches are more significant
		if strings.Contains(lowerTitle, pattern) {
			errorMatchCount += 3
		}
	}

	// For short content, any error pattern is suspicious
	if contentLength < strictThreshold && errorMatchCount > 0 {
		return true
	}

	// For longer content (documentation), only reject if error patterns are excessive
	// This allows documentation that mentions "404" or "maintenance" in passing
	if contentLength >= strictThreshold {
		// Reject only if multiple error patterns match (suggesting it's actually an error page)
		// or if the ratio of error pattern mentions to content length is high
		if errorMatchCount >= 3 {
			return true
		}
	}

	// Generic/empty page indicators
	emptyPatterns := []string{
		"home", "index", "welcome", "dashboard",
	}

	// Only filter if title matches exactly and content is very short
	for _, pattern := range emptyPatterns {
		if lowerTitle == pattern && contentLength < 100 {
			return true
		}
	}

	return false
}
