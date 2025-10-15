// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

// extractTitle extracts the title from HTML document using iterative traversal
func (h *HTTPProtocol) extractTitle(n *html.Node) string {
	stack := []*html.Node{n}

	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if current.Type == html.ElementNode && current.Data == "title" {
			return h.getTextContent(current)
		}

		for child := current.FirstChild; child != nil; child = child.NextSibling {
			stack = append(stack, child)
		}
	}

	if mt := h.extractMetaContent(n, "og:title"); mt != "" {
		return mt
	}

	if h1 := h.findFirstElementText(n, "h1"); h1 != "" {
		return h1
	}

	return ""
}

// getTextContent gets text content from a specific node using iterative traversal
func (h *HTTPProtocol) getTextContent(n *html.Node) string {
	var text strings.Builder

	stack := []*html.Node{n}

	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if current.Type == html.TextNode {
			text.WriteString(current.Data)
		}

		for child := current.FirstChild; child != nil; child = child.NextSibling {
			stack = append(stack, child)
		}
	}

	return strings.TrimSpace(text.String())
}

// extractMetaContent finds a meta tag with a specific property/name and returns its content
func (h *HTTPProtocol) extractMetaContent(n *html.Node, key string) string {
	stack := []*html.Node{n}
	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if current.Type == html.ElementNode && current.Data == "meta" {
			var prop, name, content string
			for _, a := range current.Attr {
				switch strings.ToLower(a.Key) {
				case "property":
					prop = strings.ToLower(a.Val)
				case "name":
					name = strings.ToLower(a.Val)
				case "content":
					content = a.Val
				}
			}
			if prop == strings.ToLower(key) || name == strings.ToLower(key) {
				return content
			}
		}
		for c := current.FirstChild; c != nil; c = c.NextSibling {
			stack = append(stack, c)
		}
	}
	return ""
}

// findFirstElementText returns the text for the first occurrence of a tag
func (h *HTTPProtocol) findFirstElementText(n *html.Node, tag string) string {
	stack := []*html.Node{n}
	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if current.Type == html.ElementNode && strings.EqualFold(current.Data, tag) {
			return strings.TrimSpace(h.getTextContent(current))
		}
		for c := current.FirstChild; c != nil; c = c.NextSibling {
			stack = append(stack, c)
		}
	}
	return ""
}

// cleanTitle removes common artifacts from titles
func (h *HTTPProtocol) cleanTitle(title string) string {
	t := strings.TrimSpace(title)
	if t == "" {
		return t
	}

	t = h.cleanDomainSpecificArtifacts(t)

	genericArtifacts := []string{" in dark mode"}
	for _, a := range genericArtifacts {
		t = strings.ReplaceAll(t, a, "")
	}

	t = strings.Trim(t, " ,-")

	t = regexp.MustCompile(`\s{2,}`).ReplaceAllString(t, " ")
	return strings.TrimSpace(t)
}

// cleanDomainSpecificArtifacts handles domain-specific title cleaning
func (h *HTTPProtocol) cleanDomainSpecificArtifacts(title string) string {
	t := strings.TrimSpace(title)

	if h.isMattermostDocsTitle(t) {
		return h.cleanMattermostTitle(t)
	}

	return t
}

// isMattermostDocsTitle checks if title contains Mattermost-specific patterns
func (h *HTTPProtocol) isMattermostDocsTitle(title string) bool {
	mattermostPatterns := []string{"mattermost", "Auto light/dark"}
	lowerTitle := strings.ToLower(title)

	for _, pattern := range mattermostPatterns {
		if strings.Contains(lowerTitle, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

// cleanMattermostTitle removes Mattermost-specific artifacts
func (h *HTTPProtocol) cleanMattermostTitle(title string) string {
	if strings.EqualFold(title, "Auto light/dark, in dark mode") ||
		strings.EqualFold(title, "Auto light/dark") ||
		strings.Contains(strings.ToLower(title), "auto light/dark") {
		return ""
	}

	mattermostArtifacts := []string{
		" - Mattermost documentation",
		" | Mattermost",
		" - Mattermost",
	}

	t := title
	for _, artifact := range mattermostArtifacts {
		t = strings.ReplaceAll(t, artifact, "")
	}

	return t
}

// extractMetaRefreshURL extracts the URL from meta refresh redirects
func (h *HTTPProtocol) extractMetaRefreshURL(n *html.Node) string {
	stack := []*html.Node{n}

	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if current.Type == html.ElementNode && current.Data == "meta" {
			var isRefresh bool
			var content string

			for _, attr := range current.Attr {
				if attr.Key == "http-equiv" && strings.ToLower(attr.Val) == "refresh" {
					isRefresh = true
				}
				if attr.Key == "content" {
					content = attr.Val
				}
			}

			if isRefresh && content != "" {
				if urlIdx := strings.Index(strings.ToLower(content), "url="); urlIdx != -1 {
					return strings.TrimSpace(content[urlIdx+4:])
				}
			}
		}

		for child := current.FirstChild; child != nil; child = child.NextSibling {
			stack = append(stack, child)
		}
	}

	return ""
}

// findMostRelevantChunk selects the most relevant chunk based on topic keywords
func (h *HTTPProtocol) findMostRelevantChunk(chunks []string, topic string) string {
	if len(chunks) == 0 {
		return ""
	}

	if topic == "" {
		return chunks[0]
	}

	topicLower := strings.ToLower(topic)
	topicKeywords := strings.Fields(topicLower)

	for _, chunk := range chunks {
		chunkLower := strings.ToLower(chunk)
		for _, keyword := range topicKeywords {
			if strings.Contains(chunkLower, keyword) {
				return chunk
			}
		}
	}

	return chunks[0]
}
