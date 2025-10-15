// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package semantic

import (
	"math"
	"sort"
	"strings"
)

// BuildBM25Index creates a BM25 lexical index from chunks
func BuildBM25Index(chunks []EvidenceChunk) *BM25Index {
	index := &BM25Index{
		Chunks:        chunks,
		TermFrequency: make(map[string][]int),
		DocFrequency:  make(map[string]int),
		K1:            1.5,  // Standard BM25 parameter
		B:             0.75, // Standard BM25 parameter
	}

	totalLength := 0
	for i, chunk := range chunks {
		tokens := tokenize(chunk.Text)
		totalLength += len(tokens)

		// Track which terms appear in this document
		seenTerms := make(map[string]bool)

		for _, term := range tokens {
			// Term frequency: add document index to term's posting list
			index.TermFrequency[term] = append(index.TermFrequency[term], i)

			// Document frequency: count unique documents containing term
			if !seenTerms[term] {
				index.DocFrequency[term]++
				seenTerms[term] = true
			}
		}
	}

	if len(chunks) > 0 {
		index.AvgDocLength = float64(totalLength) / float64(len(chunks))
	}

	return index
}

// Search performs BM25 search and returns top-k evidence
func (index *BM25Index) Search(query string, topK int) []Evidence {
	queryTokens := tokenize(query)

	if len(queryTokens) == 0 {
		return []Evidence{}
	}

	scores := make(map[int]float64)
	for _, term := range queryTokens {
		if docIndices, found := index.TermFrequency[term]; found {
			idf := index.calculateIDF(term)

			termCounts := make(map[int]int)
			for _, docIdx := range docIndices {
				termCounts[docIdx]++
			}

			for docIdx, termFreq := range termCounts {
				docLength := float64(len(tokenize(index.Chunks[docIdx].Text)))
				score := index.calculateBM25Score(termFreq, docLength, idf)
				scores[docIdx] += score
			}
		}
	}

	results := make([]Evidence, 0, len(scores))
	for docIdx, score := range scores {
		chunk := index.Chunks[docIdx]
		results = append(results, Evidence{
			ChunkID:    chunk.ID,
			ChunkText:  chunk.Text,
			Metadata:   chunk.Metadata,
			Similarity: score, // BM25 score stored as similarity
			Rank:       0,     // Will be set after sorting
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Similarity > results[j].Similarity
	})

	limit := topK
	if limit > len(results) {
		limit = len(results)
	}

	for i := 0; i < limit; i++ {
		results[i].Rank = i + 1
	}

	return results[:limit]
}

// calculateIDF calculates inverse document frequency for a term
func (index *BM25Index) calculateIDF(term string) float64 {
	numDocs := len(index.Chunks)
	docFreq := index.DocFrequency[term]

	if docFreq == 0 {
		return 0.0
	}

	// IDF = log((N - df + 0.5) / (df + 0.5))
	// Where N = total documents, df = document frequency
	idf := math.Log((float64(numDocs-docFreq) + 0.5) / (float64(docFreq) + 0.5))

	// Ensure IDF is non-negative
	if idf < 0 {
		return 0.0
	}

	return idf
}

// calculateBM25Score calculates BM25 score for a term in a document
func (index *BM25Index) calculateBM25Score(termFreq int, docLength, idf float64) float64 {
	// BM25 formula:
	// score = IDF * (tf * (k1 + 1)) / (tf + k1 * (1 - b + b * (docLength / avgDocLength)))

	tf := float64(termFreq)
	k1 := index.K1
	b := index.B
	avgDocLength := index.AvgDocLength

	numerator := tf * (k1 + 1)
	denominator := tf + k1*(1-b+b*(docLength/avgDocLength))

	score := idf * (numerator / denominator)
	return score
}

// tokenize converts text to lowercase tokens
func tokenize(text string) []string {
	text = strings.ToLower(text)

	// Remove punctuation and split on whitespace
	text = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == ' ' {
			return r
		}
		return ' '
	}, text)

	tokens := strings.Fields(text)

	// Filter stopwords and very short tokens
	filtered := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if len(token) > 2 && !isStopword(token) {
			filtered = append(filtered, token)
		}
	}

	return filtered
}

// isStopword checks if a token is a common stopword
func isStopword(token string) bool {
	stopwords := map[string]bool{
		"the": true, "is": true, "at": true, "which": true, "on": true,
		"a": true, "an": true, "as": true, "are": true, "was": true,
		"were": true, "been": true, "be": true, "have": true, "has": true,
		"had": true, "do": true, "does": true, "did": true, "but": true,
		"if": true, "or": true, "and": true, "for": true, "to": true,
		"of": true, "in": true, "it": true, "by": true, "with": true,
		"from": true, "this": true, "that": true, "will": true, "would": true,
		"can": true, "could": true, "should": true, "may": true, "might": true,
	}
	return stopwords[token]
}

// ComputeLexicalScore computes a simple lexical overlap score
// Used for validation when BM25 index is not available
func ComputeLexicalScore(query, document string) float64 {
	queryTokens := tokenize(query)
	docTokens := tokenize(document)

	if len(queryTokens) == 0 || len(docTokens) == 0 {
		return 0.0
	}

	docSet := make(map[string]bool)
	for _, token := range docTokens {
		docSet[token] = true
	}

	matches := 0
	for _, token := range queryTokens {
		if docSet[token] {
			matches++
		}
	}

	overlap := float64(matches) / float64(len(queryTokens))
	return overlap
}
