// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package semantic

import (
	"math"
	"regexp"
	"strconv"
	"strings"
)

// CheckEntityMatch verifies named entities are present in evidence
func CheckEntityMatch(sentence string, evidence []Evidence) bool {
	entities := extractEntities(sentence)

	if len(entities) == 0 {
		return true // No entities to check
	}

	for _, entity := range entities {
		found := false
		for _, ev := range evidence {
			if ContainsName(ev.ChunkText, entity) {
				found = true
				break
			}
			// Also check metadata fields (like "author" if present)
			for _, metaValue := range ev.Metadata {
				if ContainsName(metaValue, entity) {
					found = true
					break
				}
			}
			if found {
				break
			}
		}

		if !found {
			return false // Entity not found in any evidence
		}
	}

	return true
}

// extractEntities extracts likely named entities (capitalized words)
func extractEntities(text string) []string {
	// Pattern: Capitalized words (2+ chars, not at sentence start)
	words := strings.Fields(text)
	entities := make([]string, 0)

	for i, word := range words {
		word = strings.Trim(word, ".,!?;:\"'")

		if len(word) >= 2 && IsCapitalized(word) {
			// Skip common non-entity words (includes technical terms)
			if isCommonWord(word) {
				continue
			}

			// Skip if it's the first word AND it's a common sentence starter
			// But keep proper names even at sentence start
			if i == 0 && isSentenceStarter(word) {
				continue
			}

			entities = append(entities, word)
		}
	}

	return entities
}

// isSentenceStarter checks if a word is a common sentence-starting word (not a name)
func isSentenceStarter(word string) bool {
	starters := map[string]bool{
		"The":   true,
		"This":  true,
		"That":  true,
		"These": true,
		"Those": true,
		"We":    true,
		"They":  true,
		"It":    true,
		"There": true,
		"Here":  true,
		"When":  true,
		"Where": true,
		"Why":   true,
		"How":   true,
		"What":  true,
		"Which": true,
		"Who":   true,
	}
	return starters[word]
}

// CheckNumberMatch verifies numbers/dates match in evidence
func CheckNumberMatch(sentence string, evidence []Evidence) bool {
	numbers := extractNumbers(sentence)

	if len(numbers) == 0 {
		return true // No numbers to check
	}

	// For each number, check if it appears in evidence (with tolerance)
	for _, num := range numbers {
		found := false
		for _, ev := range evidence {
			if containsNumber(ev.ChunkText, num, 0.01) { // 1% tolerance
				found = true
				break
			}
		}

		if !found {
			return false // Number not found in any evidence
		}
	}

	return true
}

// extractNumbers extracts numbers from text
func extractNumbers(text string) []float64 {
	// Pattern: integers, floats, percentages
	// Matches: 100, 1.5, 3.14, 50%, 1,000, etc.
	numPattern := regexp.MustCompile(`\b(\d+(?:,\d{3})*(?:\.\d+)?%?)\b`)
	matches := numPattern.FindAllString(text, -1)

	numbers := make([]float64, 0, len(matches))
	for _, match := range matches {
		// Clean and parse
		cleaned := strings.ReplaceAll(match, ",", "") // Remove thousand separators
		cleaned = strings.TrimSuffix(cleaned, "%")    // Remove percentage sign

		if num, err := strconv.ParseFloat(cleaned, 64); err == nil {
			numbers = append(numbers, num)
		}
	}

	return numbers
}

// containsNumber checks if text contains a number (with tolerance)
func containsNumber(text string, target float64, tolerance float64) bool {
	numbers := extractNumbers(text)

	for _, num := range numbers {
		diff := math.Abs(num - target)
		if diff <= target*tolerance {
			return true
		}
	}

	return false
}

// CheckNegationConsistency detects negation flips
func CheckNegationConsistency(sentence string, evidenceText string) bool {
	sentenceHasNegation := hasNegation(sentence)
	evidenceHasNegation := hasNegation(evidenceText)

	sentenceHasDecision := hasDecisionVerb(sentence)
	evidenceHasDecision := hasDecisionVerb(evidenceText)

	// If sentence has decision + negation mismatch → inconsistent
	if (sentenceHasDecision || evidenceHasDecision) &&
		(sentenceHasNegation != evidenceHasNegation) {
		return false
	}

	return true
}

// hasNegation checks if text contains negation words
func hasNegation(text string) bool {
	lowerText := strings.ToLower(text)

	negationWords := []string{
		"not", "no", "never", "neither", "nor", "none",
		"nothing", "nobody", "nowhere", "without",
		"didn't", "don't", "doesn't", "won't", "wouldn't",
		"can't", "cannot", "couldn't", "shouldn't",
		"isn't", "aren't", "wasn't", "weren't",
	}

	for _, neg := range negationWords {
		// Use word boundaries to avoid matching substrings
		pattern := `\b` + neg + `\b`
		if matched, _ := regexp.MatchString(pattern, lowerText); matched {
			return true
		}
	}

	return false
}

// hasDecisionVerb checks if text contains decision-related verbs
func hasDecisionVerb(text string) bool {
	lowerText := strings.ToLower(text)

	decisionVerbs := []string{
		"decide", "decided", "decision",
		"approve", "approved", "approval",
		"reject", "rejected", "rejection",
		"agree", "agreed", "agreement",
		"accept", "accepted",
		"deny", "denied",
		"confirm", "confirmed",
		"choose", "chose", "chosen",
	}

	for _, verb := range decisionVerbs {
		pattern := `\b` + verb + `\b`
		if matched, _ := regexp.MatchString(pattern, lowerText); matched {
			return true
		}
	}

	return false
}

// CheckAttribution verifies speaker attribution
func CheckAttribution(sentence string, evidence []Evidence) bool {
	attributedPerson := extractAttribution(sentence)

	if attributedPerson == "" {
		return true // No attribution claim to verify
	}

	if len(evidence) == 0 {
		return false
	}

	// Check if the BEST matching evidence (top ranked) is from the attributed person
	topEvidence := evidence[0]

	if author, hasAuthor := topEvidence.Metadata["author"]; hasAuthor {
		if equalsCaseInsensitive(author, attributedPerson) {
			return true
		}
	}

	if containsQuoteFrom(topEvidence.ChunkText, attributedPerson) {
		return true
	}

	return false
}

// extractAttribution extracts the person attributed in a sentence
func extractAttribution(sentence string) string {
	// Patterns:
	// "John said X" → "John"
	// "According to Sarah, X" → "Sarah"
	// "Mike suggested X" → "Mike"

	attributionPatterns := []string{
		`(\w+)\s+(?:said|says|mentioned|suggested|proposed|stated|explained|noted|argued|claimed|believes|thinks)`,
		`(?:according to|as per)\s+(\w+)`,
		`(\w+)'s\s+(?:suggestion|proposal|idea|view|opinion)`,
	}

	for _, pattern := range attributionPatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(sentence)
		if len(matches) > 1 {
			person := matches[1]
			// Verify it's a capitalized word (likely a name)
			if IsCapitalized(person) && !isCommonWord(person) {
				return person
			}
		}
	}

	return ""
}

// containsQuoteFrom checks if text contains a quote attributed to a person
func containsQuoteFrom(text, person string) bool {
	lowerText := strings.ToLower(text)
	lowerPerson := strings.ToLower(person)

	// Patterns: "John: ...", "@john said", etc.
	quotePatterns := []string{
		lowerPerson + `\s*:`,
		`@` + lowerPerson + `\s+(?:said|says)`,
		lowerPerson + `\s+(?:said|says)`,
	}

	for _, pattern := range quotePatterns {
		if matched, _ := regexp.MatchString(pattern, lowerText); matched {
			return true
		}
	}

	return false
}
