// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

// filterNonContentText filters out CSS-like content and other non-meaningful text
func (h *HTMLProcessor) filterNonContentText(text string) string {
	lines := strings.Split(text, "\n")
	var contentLines []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if h.isCSSLikeLine(line) {
			continue
		}

		if h.isPromotionalContent(line) {
			continue
		}

		if len(line) < 5 {
			continue
		}

		contentLines = append(contentLines, line)
	}

	return strings.Join(contentLines, "\n")
}

// isCSSLikeLine detects if a line contains CSS-like content
func (h *HTMLProcessor) isCSSLikeLine(line string) bool {
	cssPatterns := []string{
		`{.*}`,
		`.*:.*px`,
		`.*:.*rem`,
		`.*:.*%;`,
		`@media`,
		`\..*{`,
		`#.*{`,
		`.*border.*:`,
		`.*margin.*:`,
		`.*padding.*:`,
		`.*display.*:`,
		`.*color.*:`,
	}

	for _, pattern := range cssPatterns {
		matched, _ := regexp.MatchString(pattern, line)
		if matched {
			return true
		}
	}

	return false
}

// isPromotionalContent detects promotional banners and marketing content
func (h *HTMLProcessor) isPromotionalContent(line string) bool {
	lowerLine := strings.ToLower(line)

	promotionalPatterns := []string{
		"hacktoberfest",
		"contribute, collaborate",
		"earn rewards",
		"click here to",
		"sign up now",
		"learn more",
		"get started",
		"try for free",
		"limited time",
		"special offer",
		"don't miss out",
		"subscribe",
		"newsletter",
		"follow us",
		"social media",
		"twitter",
		"linkedin",
		"facebook",
		"banner",
		"advertisement",
		"promo",
		"discount",
	}

	for _, pattern := range promotionalPatterns {
		if strings.Contains(lowerLine, pattern) {
			return true
		}
	}

	return false
}

// IsContentQualityAcceptable performs enhanced quality checks on extracted content
func (h *HTMLProcessor) IsContentQualityAcceptable(title, content string) bool {
	contentLength := len(strings.TrimSpace(content))

	if contentLength < 80 {
		return false
	}

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

	linkCount := strings.Count(strings.ToLower(content), "<a ")
	if linkCount > 10 && (float64(contentLength)/float64(linkCount)) < 50 {
		return false
	}

	navKeywords := []string{"next", "previous", "home", "menu", "search", "login", "register", "toggle", "sidebar", "breadcrumb", "navigation"}
	lowerContent := strings.ToLower(content)
	navKeywordCount := 0
	for _, keyword := range navKeywords {
		navKeywordCount += strings.Count(lowerContent, keyword)
	}
	if float64(navKeywordCount) > float64(contentLength)*MaxNavigationKeywordRatio {
		return false
	}

	lowerTitle := strings.ToLower(title)
	if lowerTitle == "home" || lowerTitle == "index" || strings.Contains(lowerTitle, "dark mode") {
		return false
	}

	return true
}

// ShouldSkipNavigationElement determines if an HTML element should be skipped based on navigation patterns
func (h *HTMLProcessor) ShouldSkipNavigationElement(node *html.Node) bool {
	if node.Type != html.ElementNode {
		return false
	}

	switch node.Data {
	case "script", "style", "nav", "header", "footer", "aside", "noscript", "svg", "button", "form":
		return true
	}

	switch strings.ToLower(node.Data) {
	case "body", "main", "article":
		return false
	}

	classes := strings.ToLower(h.getAttr(node, "class"))
	id := strings.ToLower(h.getAttr(node, "id"))
	role := strings.ToLower(h.getAttr(node, "role"))

	switch role {
	case "navigation", "search", "banner", "contentinfo", "complementary":
		if !h.containsMainContent(node) {
			return true
		}
	}

	combined := classes + " " + id

	skipPatterns := []string{
		"nav", "navbar", "navigation", "menu", "menubar", "sidebar",
		"breadcrumbs", "breadcrumb", "toc", "table-of-contents",
		"footer", "header", "search", "theme-toggle", "toggle-theme",
		"announcement", "cookie", "consent", "gdpr",
		"pagination", "pager", "related-links", "related-posts",
		"social", "share", "follow", "subscribe", "newsletter",
		"popup", "modal", "overlay",
	}

	for _, pattern := range skipPatterns {
		if strings.Contains(combined, pattern) {
			if !h.containsMainContent(node) {
				return true
			}
			break
		}
	}

	return false
}

// containsMainContent returns true if the node subtree appears to contain
// main article content. We use this to avoid skipping large wrappers that
// happen to have utility classes but contain the actual content.
func (h *HTMLProcessor) containsMainContent(node *html.Node) bool {
	if node == nil {
		return false
	}

	for c := node.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode {
			switch strings.ToLower(c.Data) {
			case "main", "article", "h1", "h2", "h3":
				return true
			}
		}
		if h.containsMainContent(c) {
			return true
		}
	}
	return false
}

// isNavigationText filters out common navigation and UI text
func (h *HTMLProcessor) isNavigationText(text string) bool {
	lowerText := strings.ToLower(text)

	if len(lowerText) < 4 {
		return true
	}

	navigationKeywords := []string{
		"toggle", "sidebar", "edit this page", "back to top",
		"link to this heading", "previous", "next", "home",
		"search", "menu", "close", "show", "hide", "expand", "collapse",
		"auto light/dark", "dark mode", "light mode", "cookie", "consent",
		"skip to content", "table of contents", "toc",
	}

	topicAnalyzer := NewTopicAnalyzer()
	promotionalKeywords := topicAnalyzer.GetTopicSynonyms("promotional")
	technicalKeywords := topicAnalyzer.GetTopicSynonyms("technical")

	words := strings.Fields(lowerText)
	wordSet := make(map[string]bool)
	for _, word := range words {
		word = strings.Trim(word, ".,!?;:")
		wordSet[word] = true
	}

	for _, keyword := range navigationKeywords {
		if strings.Contains(keyword, " ") {
			if strings.Contains(lowerText, keyword) {
				return true
			}
		} else {
			if wordSet[keyword] {
				return true
			}
		}
	}

	for _, keyword := range promotionalKeywords {
		if strings.Contains(keyword, " ") {
			if strings.Contains(lowerText, keyword) {
				return true
			}
		} else {
			if wordSet[keyword] {
				return true
			}
		}
	}

	for _, keyword := range technicalKeywords {
		if strings.Contains(keyword, " ") {
			if strings.Contains(lowerText, keyword) {
				return true
			}
		} else {
			if wordSet[keyword] {
				return true
			}
		}
	}

	if strings.HasPrefix(lowerText, "#") || strings.HasPrefix(lowerText, "http") {
		return true
	}

	return false
}

// containsJavaScriptPattern detects if text contains JavaScript code patterns
func (h *HTMLProcessor) containsJavaScriptPattern(text string) bool {
	if text == "" {
		return false
	}

	lowerText := strings.ToLower(text)

	jsPatterns := []string{
		"window.datalayer", "gtag(", "ga(", "google_tag",
		"document.addeventlistener", "document.cookie",
		"var ", "let ", "const ", "function(", "=>{",
		"console.log", "settimeout(", "queryselector",
		"||", "&&", "===", "!==", "typeof ",
	}

	cssPatterns := []string{
		"--primary-", "rgba(", "var(--",
		"@media", "@import", "@keyframes",
		"background:", "margin:", "padding:", "border:",
		"px;", "rem;", "em;", "%;",
	}

	for _, pattern := range jsPatterns {
		if strings.Contains(lowerText, pattern) {
			return true
		}
	}

	for _, pattern := range cssPatterns {
		if strings.Contains(lowerText, pattern) {
			return true
		}
	}

	symbolCount := 0
	totalChars := len(text)
	for _, char := range text {
		if char == '{' || char == '}' || char == '(' || char == ')' ||
			char == ';' || char == ':' || char == '=' {
			symbolCount++
		}
	}

	if totalChars > 50 && float64(symbolCount)/float64(totalChars) > 0.15 {
		return true
	}

	return false
}
