// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package semantic

import (
	"context"
	"math"
	"sort"
)

// RetrieveCandidates performs hybrid retrieval (BM25 + embeddings)
func RetrieveCandidates(
	ctx context.Context,
	sentence string,
	index *EvidenceIndex,
	embedder Embedder,
	opts ValidatorOptions,
) ([]Evidence, error) {
	var allCandidates []Evidence

	// Stage 1: BM25 lexical retrieval
	if opts.UseLexical && index.BM25Index != nil {
		lexicalResults := index.BM25Index.Search(sentence, opts.TopK*2) // Get 2x for diversity
		allCandidates = append(allCandidates, lexicalResults...)
	}

	// Stage 2: Embedding semantic retrieval
	sentenceEmbed, err := embedder.Embed(ctx, sentence)
	if err != nil {
		// If embedding fails, fall back to lexical only
		return deduplicateAndRank(allCandidates, opts.TopK), nil
	}

	semanticResults := searchBySimilarity(sentenceEmbed, index, opts.TopK*2)
	allCandidates = append(allCandidates, semanticResults...)

	// Stage 3: Deduplicate and re-rank by combined score
	finalResults := deduplicateAndRank(allCandidates, opts.TopK)

	return finalResults, nil
}

// searchBySimilarity finds top-k similar chunks using cosine similarity
func searchBySimilarity(queryEmbed []float32, index *EvidenceIndex, topK int) []Evidence {
	if len(index.Chunks) == 0 {
		return []Evidence{}
	}

	// Calculate cosine similarity with all chunks
	similarities := make([]struct {
		chunkIdx   int
		similarity float64
	}, len(index.Chunks))

	for i, chunk := range index.Chunks {
		if chunk.Embedding == nil {
			similarities[i] = struct {
				chunkIdx   int
				similarity float64
			}{chunkIdx: i, similarity: 0.0}
			continue
		}

		sim := cosineSimilarity(queryEmbed, chunk.Embedding)
		similarities[i] = struct {
			chunkIdx   int
			similarity float64
		}{chunkIdx: i, similarity: sim}
	}

	sort.Slice(similarities, func(i, j int) bool {
		return similarities[i].similarity > similarities[j].similarity
	})

	limit := topK
	if limit > len(similarities) {
		limit = len(similarities)
	}

	results := make([]Evidence, limit)
	for i := 0; i < limit; i++ {
		chunkIdx := similarities[i].chunkIdx
		chunk := index.Chunks[chunkIdx]

		results[i] = Evidence{
			ChunkID:    chunk.ID,
			ChunkText:  chunk.Text,
			Metadata:   chunk.Metadata,
			Similarity: similarities[i].similarity,
			Rank:       i + 1,
		}
	}

	return results
}

// cosineSimilarity computes cosine similarity between two embeddings
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0.0
	}

	var dotProduct, normA, normB float64
	for i := 0; i < len(a); i++ {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0.0 || normB == 0.0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// deduplicateAndRank deduplicates evidence by chunk and re-ranks
func deduplicateAndRank(candidates []Evidence, topK int) []Evidence {
	if len(candidates) == 0 {
		return []Evidence{}
	}

	// Deduplicate by ChunkID+ChunkText (keep highest score)
	unique := make(map[string]Evidence)
	for _, candidate := range candidates {
		key := candidate.ChunkID + ":" + candidate.ChunkText

		if existing, found := unique[key]; found {
			// Keep the one with higher similarity
			if candidate.Similarity > existing.Similarity {
				unique[key] = candidate
			}
		} else {
			unique[key] = candidate
		}
	}

	deduplicated := make([]Evidence, 0, len(unique))
	for _, evidence := range unique {
		deduplicated = append(deduplicated, evidence)
	}

	sort.Slice(deduplicated, func(i, j int) bool {
		return deduplicated[i].Similarity > deduplicated[j].Similarity
	})

	limit := topK
	if limit > len(deduplicated) {
		limit = len(deduplicated)
	}

	results := deduplicated[:limit]
	for i := range results {
		results[i].Rank = i + 1
	}

	return results
}
