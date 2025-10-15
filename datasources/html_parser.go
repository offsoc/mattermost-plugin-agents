// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

// ExtractStructuredText converts HTML to clean, structured text while preserving
// semantic structure like headings, lists, and code blocks
func (h *HTMLProcessor) ExtractStructuredText(htmlContent string) string {
	if htmlContent == "" {
		return ""
	}

	const maxHTMLSize = 10 * 1024 * 1024
	useFallback := false
	if len(htmlContent) > maxHTMLSize {
		fmt.Printf("WARN: HTML content exceeds size limit (%d bytes), using fallback extraction\n", len(htmlContent))
		lastCloseTag := strings.LastIndex(htmlContent[:maxHTMLSize], ">")
		if lastCloseTag > 0 && lastCloseTag < maxHTMLSize {
			htmlContent = htmlContent[:lastCloseTag+1]
		} else {
			useFallback = true
		}
	}

	if !useFallback {
		if doc, err := html.Parse(strings.NewReader(htmlContent)); err == nil {
			body := h.findBodyNode(doc)
			if body == nil {
				body = doc
			}

			var b strings.Builder
			h.processNode(body, &b, 0)

			rawText := b.String()
			text := h.normalizeWhitespace(rawText)
			filteredText := h.filterNonContentText(text)

			if len(strings.TrimSpace(filteredText)) == 0 || len(filteredText) < 40 {
				fallback := h.fallbackTextExtraction(htmlContent)
				if len(strings.TrimSpace(fallback)) > len(strings.TrimSpace(filteredText)) {
					if len(fallback) < 200 && h.containsJavaScriptPattern(fallback) {
						return ""
					}
					return fallback
				}
			}

			if len(filteredText) < 200 && h.containsJavaScriptPattern(filteredText) {
				return ""
			}

			return filteredText
		}
	}

	text := h.fallbackTextExtraction(htmlContent)

	if len(text) < 200 && h.containsJavaScriptPattern(text) {
		return ""
	}

	return text
}

// processNode recursively processes HTML nodes and converts them to structured text
func (h *HTMLProcessor) processNode(node *html.Node, builder *strings.Builder, depth int) {
	if node == nil {
		return
	}

	switch node.Type {
	case html.TextNode:
		text := strings.TrimSpace(node.Data)
		if text != "" && !h.isNavigationText(text) {
			builder.WriteString(text)
		}

	case html.ElementNode:
		if h.ShouldSkipNavigationElement(node) {
			return
		}

		switch strings.ToLower(node.Data) {
		case "h1":
			builder.WriteString("\n\n# ")
			h.processChildren(node, builder, depth)
			builder.WriteString("\n\n")
		case "h2":
			builder.WriteString("\n\n## ")
			h.processChildren(node, builder, depth)
			builder.WriteString("\n\n")
		case "h3":
			builder.WriteString("\n\n### ")
			h.processChildren(node, builder, depth)
			builder.WriteString("\n\n")
		case "h4":
			builder.WriteString("\n\n#### ")
			h.processChildren(node, builder, depth)
			builder.WriteString("\n\n")
		case "h5":
			builder.WriteString("\n\n##### ")
			h.processChildren(node, builder, depth)
			builder.WriteString("\n\n")
		case "h6":
			builder.WriteString("\n\n###### ")
			h.processChildren(node, builder, depth)
			builder.WriteString("\n\n")
		case "p":
			builder.WriteString("\n\n")
			h.processChildren(node, builder, depth)
			builder.WriteString("\n")
		case "br":
			builder.WriteString("\n")
		case "ul":
			builder.WriteString("\n")
			h.processChildren(node, builder, depth)
			builder.WriteString("\n")
		case "ol":
			builder.WriteString("\n")
			h.processChildren(node, builder, depth)
			builder.WriteString("\n")
		case "li":
			if h.isInsideOrderedList(node) {
				builder.WriteString("\n1. ")
			} else {
				builder.WriteString("\n- ")
			}
			h.processChildren(node, builder, depth)
		case "code":
			builder.WriteString("`")
			h.processChildren(node, builder, depth)
			builder.WriteString("`")
		case "pre":
			builder.WriteString("\n\n```\n")
			h.processChildren(node, builder, depth)
			builder.WriteString("\n```\n\n")
		case "blockquote":
			builder.WriteString("\n\n> ")
			h.processChildren(node, builder, depth)
			builder.WriteString("\n\n")
		case "strong", "b":
			builder.WriteString("**")
			h.processChildren(node, builder, depth)
			builder.WriteString("**")
		case "em", "i":
			builder.WriteString("*")
			h.processChildren(node, builder, depth)
			builder.WriteString("*")
		case "a":
			h.processChildren(node, builder, depth)
			for _, attr := range node.Attr {
				if attr.Key == "href" && attr.Val != "" {
					builder.WriteString(" (")
					builder.WriteString(attr.Val)
					builder.WriteString(")")
				}
			}
		case "table":
			builder.WriteString("\n\n")
			h.processTableNode(node, builder, depth)
			builder.WriteString("\n\n")
		case "div", "span", "section", "article":
			h.processChildren(node, builder, depth)
		case "script", "style", "meta", "title", "head":
			return
		default:
			h.processChildren(node, builder, depth)
		}
	}
}

// processChildren processes all child nodes
func (h *HTMLProcessor) processChildren(node *html.Node, builder *strings.Builder, depth int) {
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		h.processNode(child, builder, depth+1)
	}
}

// processTableNode handles table structure conversion
func (h *HTMLProcessor) processTableNode(node *html.Node, builder *strings.Builder, depth int) {
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode {
			switch strings.ToLower(child.Data) {
			case "thead", "tbody":
				h.processChildren(child, builder, depth)
			case "tr":
				builder.WriteString("| ")
				h.processTableRow(child, builder, depth)
				builder.WriteString("\n")
			}
		}
	}
}

// processTableRow processes table row cells
func (h *HTMLProcessor) processTableRow(node *html.Node, builder *strings.Builder, depth int) {
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode &&
			(strings.ToLower(child.Data) == "td" || strings.ToLower(child.Data) == "th") {
			h.processChildren(child, builder, depth)
			builder.WriteString(" | ")
		}
	}
}

// isInsideOrderedList checks if current node is inside an ordered list
func (h *HTMLProcessor) isInsideOrderedList(node *html.Node) bool {
	for parent := node.Parent; parent != nil; parent = parent.Parent {
		if parent.Type == html.ElementNode && strings.ToLower(parent.Data) == "ol" {
			return true
		}
	}
	return false
}

// normalizeWhitespace cleans up excessive whitespace while preserving structure
func (h *HTMLProcessor) normalizeWhitespace(text string) string {
	text = strings.ReplaceAll(text, "\n\n\n", "\n\n")
	text = strings.ReplaceAll(text, "\t", " ")

	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " ")
		if strings.TrimSpace(line) != "" {
			words := strings.Fields(line)
			if len(words) > 0 {
				leadingSpaces := len(line) - len(strings.TrimLeft(line, " "))
				prefix := strings.Repeat(" ", leadingSpaces)
				lines[i] = prefix + strings.Join(words, " ")
			}
		}
	}

	return strings.Join(lines, "\n")
}

// fallbackTextExtraction provides regex-based fallback when HTML parsing fails
func (h *HTMLProcessor) fallbackTextExtraction(htmlContent string) string {
	// Use the new preserve-structure method for better output quality
	htmlContent = h.stripHTMLTagsPreserveStructure(htmlContent)

	text := h.filterNonContentText(htmlContent)

	return text
}

// removeElementsWithRegex removes specific HTML elements using regex
func (h *HTMLProcessor) removeElementsWithRegex(text, element string) string {
	pattern := fmt.Sprintf(`(?is)<%s[^>]*?>.*?</%s>`, element, element)
	re := regexp.MustCompile(pattern)
	text = re.ReplaceAllString(text, "")

	pattern = fmt.Sprintf(`(?i)<%s[^>]*/>`, element)
	re = regexp.MustCompile(pattern)
	return re.ReplaceAllString(text, "")
}

// convertHeadingsWithRegex converts headings to markdown format using regex
func (h *HTMLProcessor) convertHeadingsWithRegex(text string) string {
	for i := 1; i <= 6; i++ {
		pattern := fmt.Sprintf(`(?i)<h%d[^>]*>(.*?)</h%d>`, i, i)
		replacement := strings.Repeat("#", i) + " $1\n\n"
		re := regexp.MustCompile(pattern)
		text = re.ReplaceAllString(text, replacement)
	}
	return text
}

// convertListsWithRegex converts lists to markdown format using regex
func (h *HTMLProcessor) convertListsWithRegex(text string) string {
	pattern := `(?i)<li[^>]*>(.*?)</li>`
	re := regexp.MustCompile(pattern)
	text = re.ReplaceAllString(text, "- $1\n")
	return text
}

// stripHTMLTags removes all HTML tags using regex with performance optimizations
func (h *HTMLProcessor) stripHTMLTags(text string) string {
	// Fast path: no HTML present
	if !strings.Contains(text, "<") {
		return text
	}

	// Preserve paragraph breaks before stripping tags (better readability)
	paragraphPattern := regexp.MustCompile(`(?i)<p[^>]*>`)
	text = paragraphPattern.ReplaceAllString(text, "\n\n")
	text = strings.ReplaceAll(text, "</p>", "")

	// Fast path: handle common simple tags without regex overhead
	commonTags := map[string]string{
		"<div>":   " ",
		"</div>":  " ",
		"<span>":  "",
		"</span>": "",
		"<br>":    "\n",
		"<br/>":   "\n",
		"<br />":  "\n",
	}
	for tag, replacement := range commonTags {
		if strings.Contains(text, tag) {
			text = strings.ReplaceAll(text, tag, replacement)
		}
	}

	// Strip all remaining HTML tags with regex
	re := regexp.MustCompile(`<[^>]*>`)
	return re.ReplaceAllString(text, " ")
}

// stripHTMLTagsPreserveStructure removes HTML tags while preserving document structure
func (h *HTMLProcessor) stripHTMLTagsPreserveStructure(text string) string {
	// Remove script/style/noscript first to prevent pollution
	text = h.removeElementsWithRegex(text, "script")
	text = h.removeElementsWithRegex(text, "style")
	text = h.removeElementsWithRegex(text, "noscript")

	// Preserve semantic structure
	text = h.convertHeadingsWithRegex(text)
	text = h.convertListsWithRegex(text)

	// Convert paragraphs and breaks to newlines for readability
	text = strings.ReplaceAll(text, "<p>", "\n\n")
	text = strings.ReplaceAll(text, "</p>", "")
	text = strings.ReplaceAll(text, "<div>", "\n")
	text = strings.ReplaceAll(text, "</div>", "")
	text = strings.ReplaceAll(text, "<br>", "\n")
	text = strings.ReplaceAll(text, "<br/>", "\n")
	text = strings.ReplaceAll(text, "<br />", "\n")

	// Strip all remaining HTML tags
	text = h.stripHTMLTags(text)

	// Clean CSS patterns that may remain
	text = h.removeInlineCSSPatterns(text)

	// Normalize whitespace
	return h.normalizeWhitespace(text)
}

// removeInlineCSSPatterns removes CSS patterns that remain after HTML tag removal
func (h *HTMLProcessor) removeInlineCSSPatterns(text string) string {
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

// findBodyNode finds the body element in the HTML document
func (h *HTMLProcessor) findBodyNode(node *html.Node) *html.Node {
	if node == nil {
		return nil
	}

	if node.Type == html.ElementNode && node.Data == "body" {
		return node
	}

	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if result := h.findBodyNode(child); result != nil {
			return result
		}
	}

	return nil
}

// getAttr gets the value of an attribute from a node
func (h *HTMLProcessor) getAttr(node *html.Node, key string) string {
	for _, attr := range node.Attr {
		if strings.EqualFold(attr.Key, key) {
			return attr.Val
		}
	}
	return ""
}

// extractTextFromNode extracts all text content from a node
func (h *HTMLProcessor) extractTextFromNode(node *html.Node) string {
	var text strings.Builder
	h.extractTextRecursive(node, &text)
	return text.String()
}

// extractTextRecursive recursively extracts text from HTML nodes
func (h *HTMLProcessor) extractTextRecursive(node *html.Node, builder *strings.Builder) {
	if node == nil {
		return
	}

	if node.Type == html.TextNode {
		builder.WriteString(node.Data)
	}

	for child := node.FirstChild; child != nil; child = child.NextSibling {
		h.extractTextRecursive(child, builder)
	}
}
