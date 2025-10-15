// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package thread

import (
	"regexp"
	"strings"
	"unicode"
)

// SplitIntoSentences splits text into sentences using proper boundary detection
// Handles: abbreviations, URLs, ellipsis, lists, etc.
func SplitIntoSentences(text string) []string {
	if text == "" {
		return []string{}
	}

	// Normalize whitespace
	text = normalizeWhitespace(text)

	// Pattern-based sentence splitting with proper handling of edge cases
	sentences := splitSentencesWithPatterns(text)

	// Clean and filter
	result := make([]string, 0, len(sentences))
	for _, s := range sentences {
		s = strings.TrimSpace(s)
		if s != "" && len(s) > 3 { // Minimum 3 chars to be a sentence
			result = append(result, s)
		}
	}

	return result
}

// splitSentencesWithPatterns uses regex patterns to split sentences
func splitSentencesWithPatterns(text string) []string {
	// Common abbreviations that should NOT trigger sentence breaks
	abbreviations := []string{
		"Dr", "Mr", "Mrs", "Ms", "Prof", "Sr", "Jr",
		"Inc", "Ltd", "Co", "Corp",
		"e.g", "i.e", "etc", "vs", "approx",
		"Jan", "Feb", "Mar", "Apr", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec",
	}

	// Protect abbreviations by marking them with special tokens
	// Replace "Dr." with "Dr.\x00" where \x00 is a marker
	protected := text
	placeholders := make(map[string]string)
	abbrMarker := "\x00ABBR\x00"

	for _, abbr := range abbreviations {
		// Replace "Dr." with "Dr.\x00ABBR\x00"
		original := abbr + "."
		placeholder := abbr + "." + abbrMarker
		protected = strings.ReplaceAll(protected, original, placeholder)
		placeholders[placeholder] = original
	}

	// Protect URLs (http://, https://)
	urlPattern := regexp.MustCompile(`https?://[^\s]+`)
	urls := urlPattern.FindAllString(protected, -1)
	urlMarker := "\x00URL\x00"
	for _, url := range urls {
		placeholder := url + urlMarker
		placeholders[placeholder] = url
		protected = strings.ReplaceAll(protected, url, placeholder)
	}

	// Split on sentence boundaries
	// Pattern: period/exclamation/question followed by space and capital letter
	// Match should NOT have the abbreviation marker right after the punctuation
	sentencePattern := regexp.MustCompile(`([.!?])(\s+)([A-Z])`)

	matches := sentencePattern.FindAllStringSubmatchIndex(protected, -1)

	if len(matches) == 0 {
		// No sentence boundaries found, restore and return as single sentence
		restored := protected
		for placeholder, original := range placeholders {
			restored = strings.ReplaceAll(restored, placeholder, original)
		}
		return []string{restored}
	}

	sentences := make([]string, 0, len(matches)+1)
	lastEnd := 0

	for _, match := range matches {
		// match[0] is start of entire match, match[1] is end
		// match[2] is start of period, match[3] is end of period (match[3] = match[2]+1)
		// match[6] is start of capital letter

		// Check if this period has the abbreviation marker right after it
		periodEnd := match[3]
		if periodEnd < len(protected) && protected[periodEnd] == '\x00' {
			// This is a protected abbreviation, skip
			continue
		}

		// This is a real sentence boundary
		// Include everything up to and including the period and whitespace
		sentenceEnd := match[6] // Start of capital letter (don't include it)
		sentence := protected[lastEnd:sentenceEnd]
		sentences = append(sentences, sentence)
		lastEnd = sentenceEnd
	}

	// Add remaining text as final sentence
	if lastEnd < len(protected) {
		sentences = append(sentences, protected[lastEnd:])
	}

	// Restore protected strings
	restored := make([]string, len(sentences))
	for i, sent := range sentences {
		restored[i] = sent
		for placeholder, original := range placeholders {
			restored[i] = strings.ReplaceAll(restored[i], placeholder, original)
		}
	}

	return restored
}

// normalizeWhitespace normalizes whitespace in text
func normalizeWhitespace(text string) string {
	// Replace multiple spaces with single space
	space := regexp.MustCompile(`\s+`)
	text = space.ReplaceAllString(text, " ")

	// Normalize newlines
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	return strings.TrimSpace(text)
}

// ChunkPosts splits posts into overlapping windows for better matching
func ChunkPosts(posts []Post, windowSize int) []PostChunk {
	if windowSize < 1 {
		windowSize = 2 // Default: 2-sentence windows
	}

	chunks := make([]PostChunk, 0, len(posts)*3) // Estimate 3 chunks per post

	for _, post := range posts {
		postChunks := chunkSinglePost(post, windowSize)
		chunks = append(chunks, postChunks...)
	}

	return chunks
}

// chunkSinglePost splits a single post into chunks
func chunkSinglePost(post Post, windowSize int) []PostChunk {
	// Split post into sentences
	sentences := SplitIntoSentences(post.Text)

	if len(sentences) == 0 {
		return []PostChunk{}
	}

	chunks := make([]PostChunk, 0, len(sentences))

	// Create overlapping windows
	// For windowSize=2: [sent0, sent1], [sent1, sent2], [sent2, sent3], ...
	// For single sentence: just use that sentence
	if len(sentences) == 1 {
		chunks = append(chunks, PostChunk{
			PostID:     post.ID,
			Author:     post.Author,
			Text:       sentences[0],
			StartIndex: 0,
			EndIndex:   len(sentences[0]),
		})
		return chunks
	}

	// Create sliding windows
	for i := 0; i < len(sentences); i++ {
		// Determine window end
		end := i + windowSize
		if end > len(sentences) {
			end = len(sentences)
		}

		// Combine sentences in window
		windowSentences := sentences[i:end]
		chunkText := strings.Join(windowSentences, " ")

		// Calculate character indices (approximate)
		startIndex := findSentenceStart(post.Text, windowSentences[0])
		endIndex := findSentenceEnd(post.Text, windowSentences[len(windowSentences)-1])

		chunks = append(chunks, PostChunk{
			PostID:     post.ID,
			Author:     post.Author,
			Text:       chunkText,
			StartIndex: startIndex,
			EndIndex:   endIndex,
		})

		// If we've reached the end, stop
		if end == len(sentences) {
			break
		}
	}

	return chunks
}

// findSentenceStart finds the approximate start index of a sentence in text
func findSentenceStart(text, sentence string) int {
	words := strings.Fields(sentence)
	if len(words) == 0 {
		return 0
	}

	// Search for first 2-3 words
	searchTerms := words[0]
	if len(words) > 1 {
		searchTerms = words[0] + " " + words[1]
	}

	index := strings.Index(text, searchTerms)
	if index >= 0 {
		return index
	}
	return 0
}

// findSentenceEnd finds the approximate end index of a sentence in text
func findSentenceEnd(text, sentence string) int {
	words := strings.Fields(sentence)
	if len(words) == 0 {
		return len(text)
	}

	// Search for last 2-3 words
	searchTerms := words[len(words)-1]
	if len(words) > 1 {
		searchTerms = words[len(words)-2] + " " + words[len(words)-1]
	}

	index := strings.LastIndex(text, searchTerms)
	if index >= 0 {
		return index + len(searchTerms)
	}
	return len(text)
}

// ExtractParticipantNames extracts likely participant names from text
// Simple heuristic: capitalized words that appear multiple times
func ExtractParticipantNames(text string) []string {
	// Pattern: capitalized words (potential names)
	namePattern := regexp.MustCompile(`\b[A-Z][a-z]+\b`)
	matches := namePattern.FindAllString(text, -1)

	nameCounts := make(map[string]int)
	for _, match := range matches {
		// Filter out common non-name words
		if !isCommonWord(match) {
			nameCounts[match]++
		}
	}

	// Extract names that appear at least once
	names := make([]string, 0, len(nameCounts))
	for name := range nameCounts {
		names = append(names, name)
	}

	return names
}

// isCommonWord checks if a capitalized word is likely NOT a name
func isCommonWord(word string) bool {
	commonWords := map[string]bool{
		"The": true, "This": true, "That": true, "These": true, "Those": true,
		"We": true, "They": true, "He": true, "She": true, "It": true,
		"My": true, "Your": true, "Our": true, "Their": true,
		"I": true, "You": true, "Me": true,
		"Monday": true, "Tuesday": true, "Wednesday": true, "Thursday": true,
		"Friday": true, "Saturday": true, "Sunday": true,
		"January": true, "February": true, "March": true, "April": true,
		"May": true, "June": true, "July": true, "August": true,
		"September": true, "October": true, "November": true, "December": true,
		// Technical terms and product names
		"Redis": true, "PostgreSQL": true, "MySQL": true, "MongoDB": true,
		"Docker": true, "Kubernetes": true, "AWS": true, "Azure": true, "GCP": true,
		"Linux": true, "Windows": true, "MacOS": true, "Ubuntu": true,
		"Python": true, "JavaScript": true, "TypeScript": true, "Java": true,
		"React": true, "Vue": true, "Angular": true, "Node": true,
		"GitHub": true, "GitLab": true, "Bitbucket": true,
		"Slack": true, "Teams": true, "Zoom": true,
		"Mattermost": true, "Jira": true, "Confluence": true,
	}
	return commonWords[word]
}

// ContainsName checks if text contains a name (case-insensitive)
func ContainsName(text, name string) bool {
	lowerText := strings.ToLower(text)
	lowerName := strings.ToLower(name)
	return strings.Contains(lowerText, lowerName)
}

// IsCapitalized checks if a word starts with a capital letter
func IsCapitalized(word string) bool {
	if len(word) == 0 {
		return false
	}
	return unicode.IsUpper(rune(word[0]))
}
