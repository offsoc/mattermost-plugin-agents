// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

// ExtractArticleLinks extracts article/blog post links from a listing/category page
// Returns a list of URLs that appear to be content links (not navigation/footer links)
func (h *HTMLProcessor) ExtractArticleLinks(htmlContent, baseURL string) []string {
	if htmlContent == "" {
		return nil
	}

	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil
	}

	var links []string
	var extractLinks func(*html.Node)
	extractLinks = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "a" {
			var href, linkText string
			for _, attr := range node.Attr {
				if attr.Key == "href" {
					href = attr.Val
					break
				}
			}

			linkText = h.extractTextFromNode(node)

			if href != "" && h.looksLikeArticleLink(href, linkText, baseURL) {
				absoluteURL := h.makeAbsoluteURL(href, baseURL)
				if absoluteURL != "" {
					links = append(links, absoluteURL)
				}
			}
		}

		for child := node.FirstChild; child != nil; child = child.NextSibling {
			extractLinks(child)
		}
	}

	extractLinks(doc)
	return h.deduplicateLinks(links)
}

// makeAbsoluteURL converts a relative URL to absolute using proper URL parsing
func (h *HTMLProcessor) makeAbsoluteURL(href, baseURL string) string {
	base, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}

	ref, err := url.Parse(href)
	if err != nil {
		return ""
	}

	absolute := base.ResolveReference(ref)
	return absolute.String()
}

// looksLikeArticleLink determines if a link appears to be an article/blog post
// rather than navigation, footer, or other non-content links
func (h *HTMLProcessor) looksLikeArticleLink(href, linkText, baseURL string) bool {
	hrefLower := strings.ToLower(href)
	linkTextLower := strings.ToLower(linkText)

	// Domain validation: Only allow links within the same domain or subdomain
	if !h.isSameDomainOrSubdomain(href, baseURL) {
		return false
	}

	excludePatterns := []string{
		"#", "javascript:", "mailto:", "tel:",
		"/category/", "/tag/", "/author/", "/page/",
		"/privacy", "/terms", "/about", "/contact",
		"/login", "/register", "/account",
		"facebook.com", "twitter.com", "linkedin.com", "youtube.com",
		"hacktoberfest",
	}

	for _, pattern := range excludePatterns {
		if strings.Contains(hrefLower, pattern) {
			return false
		}
	}

	if len(linkTextLower) < 10 {
		return false
	}

	navKeywords := []string{
		"home", "back", "next", "previous", "more", "less",
		"menu", "navigation", "footer", "header", "sidebar",
		"search", "subscribe", "follow", "share", "comment",
	}

	for _, keyword := range navKeywords {
		if linkTextLower == keyword {
			return false
		}
	}

	// Known content URL patterns that indicate article/documentation pages
	if strings.Contains(hrefLower, "/blog/") ||
		strings.Contains(hrefLower, "/post/") ||
		strings.Contains(hrefLower, "/article/") ||
		strings.Contains(hrefLower, "/news/") ||
		strings.Contains(hrefLower, "/onboard/") ||
		strings.Contains(hrefLower, "/configure/") ||
		strings.Contains(hrefLower, "/deploy/") ||
		strings.Contains(hrefLower, "/manage/") {
		return true
	}

	// Accept links with reasonable text length (lowered from 20 to 12 to capture docs like "AD/LDAP setup")
	return len(linkText) > 12
}

// isSameDomainOrSubdomain checks if href is within the same domain as baseURL
// This prevents cross-domain navigation while allowing relative links
func (h *HTMLProcessor) isSameDomainOrSubdomain(href, baseURL string) bool {
	// Parse base URL
	base, err := url.Parse(baseURL)
	if err != nil {
		return false
	}

	// If href is relative (no protocol), it's definitely same domain
	if !strings.Contains(href, "://") {
		return true
	}

	// Parse href URL
	hrefURL, err := url.Parse(href)
	if err != nil {
		return false
	}

	// Extract hosts (case-insensitive comparison)
	baseHost := strings.ToLower(base.Host)
	hrefHost := strings.ToLower(hrefURL.Host)

	// Must be exact match - no cross-subdomain navigation
	// This prevents docs.mattermost.com from navigating to mattermost.com or vice versa
	return baseHost == hrefHost
}

// deduplicateLinks removes duplicate URLs from the list
func (h *HTMLProcessor) deduplicateLinks(links []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(links))

	for _, link := range links {
		if !seen[link] {
			seen[link] = true
			result = append(result, link)
		}
	}

	return result
}
