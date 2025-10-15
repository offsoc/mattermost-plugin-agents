// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

// HTMLProcessor provides centralized HTML processing utilities for converting
// HTML content to structured text suitable for LLM consumption
type HTMLProcessor struct{}

// DocumentSection represents a semantic section of a document with its heading and content
type DocumentSection struct {
	Heading string // The heading text (e.g., "Installation Guide")
	Level   int    // Heading level: H1=1, H2=2, H3=3, etc.
	Content string // The structured text content under this heading
}

// NewHTMLProcessor creates a new HTML processor instance
func NewHTMLProcessor() *HTMLProcessor {
	return &HTMLProcessor{}
}
