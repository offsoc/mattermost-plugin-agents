// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"strings"

	"golang.org/x/net/html"
)

// ExtractSemanticSections extracts content organized by semantic sections (H2/H3 headings)
// This provides better context for LLM consumption compared to full-page extraction
func (h *HTMLProcessor) ExtractSemanticSections(htmlContent string) []DocumentSection {
	if htmlContent == "" {
		return nil
	}

	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return []DocumentSection{{
			Heading: "",
			Level:   0,
			Content: h.fallbackTextExtraction(htmlContent),
		}}
	}

	bodyNode := h.findBodyNode(doc)
	if bodyNode == nil {
		bodyNode = doc
	}

	return h.extractSectionsFromNode(bodyNode)
}

// extractSectionsFromNode extracts semantic sections based on heading elements
func (h *HTMLProcessor) extractSectionsFromNode(node *html.Node) []DocumentSection {
	var sections []DocumentSection
	var currentSection *DocumentSection

	h.traverseForSections(node, &sections, &currentSection)

	if currentSection != nil && strings.TrimSpace(currentSection.Content) != "" {
		sections = append(sections, *currentSection)
	}

	if len(sections) == 0 {
		var builder strings.Builder
		h.processNode(node, &builder, 0)
		content := h.normalizeWhitespace(builder.String())
		if strings.TrimSpace(content) != "" {
			sections = append(sections, DocumentSection{
				Heading: "",
				Level:   0,
				Content: strings.TrimSpace(content),
			})
		}
	}

	return sections
}

// traverseForSections recursively traverses nodes to extract heading-based sections
func (h *HTMLProcessor) traverseForSections(node *html.Node, sections *[]DocumentSection, currentSection **DocumentSection) {
	if node == nil {
		return
	}

	if h.ShouldSkipNavigationElement(node) {
		return
	}

	if node.Type == html.ElementNode {
		headingLevel := h.getHeadingLevel(node.Data)
		if headingLevel > 0 && headingLevel <= 3 {
			if *currentSection != nil && strings.TrimSpace((*currentSection).Content) != "" {
				*sections = append(*sections, **currentSection)
			}

			var headingBuilder strings.Builder
			h.processChildren(node, &headingBuilder, 0)
			headingText := strings.TrimSpace(headingBuilder.String())

			*currentSection = &DocumentSection{
				Heading: headingText,
				Level:   headingLevel,
				Content: "",
			}
		} else if *currentSection != nil {
			var contentBuilder strings.Builder
			h.processNode(node, &contentBuilder, 0)
			content := contentBuilder.String()

			if trimmed := strings.TrimSpace(content); trimmed != "" && !h.isNavigationText(trimmed) {
				if (*currentSection).Content != "" {
					(*currentSection).Content += "\n"
				}
				(*currentSection).Content += trimmed
			}
		}
	}

	if node.Type != html.ElementNode || h.getHeadingLevel(node.Data) == 0 {
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			h.traverseForSections(child, sections, currentSection)
		}
	}
}

// getHeadingLevel returns the heading level (1-6) for heading elements, 0 for non-headings
func (h *HTMLProcessor) getHeadingLevel(tagName string) int {
	switch strings.ToLower(tagName) {
	case "h1":
		return 1
	case "h2":
		return 2
	case "h3":
		return 3
	case "h4":
		return 4
	case "h5":
		return 5
	case "h6":
		return 6
	default:
		return 0
	}
}
