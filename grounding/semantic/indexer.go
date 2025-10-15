// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package semantic

import (
	"context"
)

// BuildEvidenceIndex creates a searchable representation of evidence texts
func BuildEvidenceIndex(
	ctx context.Context,
	chunks []EvidenceChunk,
	embedder Embedder,
	opts ValidatorOptions,
) (*EvidenceIndex, error) {
	if len(chunks) == 0 {
		return &EvidenceIndex{
			Chunks:       []EvidenceChunk{},
			EmbeddingMap: make(map[string][]float32),
			ModelVersion: embedder.ModelVersion(),
		}, nil
	}

	// Extract chunk texts for batch embedding
	chunkTexts := make([]string, len(chunks))
	for i, chunk := range chunks {
		chunkTexts[i] = chunk.Text
	}

	// Batch embed all chunks
	embeddings, err := embedder.EmbedBatch(ctx, chunkTexts)
	if err != nil {
		return nil, err
	}

	// Store embeddings in chunks and create embedding map
	embeddingMap := make(map[string][]float32, len(chunks))
	for i := range chunks {
		chunks[i].Embedding = embeddings[i]
		embeddingMap[chunks[i].ID] = embeddings[i]
	}

	// Build BM25 index for lexical search (optional)
	var bm25Index *BM25Index
	if opts.UseLexical {
		bm25Index = BuildBM25Index(chunks)
	}

	return &EvidenceIndex{
		Chunks:       chunks,
		BM25Index:    bm25Index,
		EmbeddingMap: embeddingMap,
		ModelVersion: embedder.ModelVersion(),
	}, nil
}

// equalsCaseInsensitive compares two strings case-insensitively
func equalsCaseInsensitive(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if toLower(a[i]) != toLower(b[i]) {
			return false
		}
	}
	return true
}

// toLower converts a byte to lowercase
func toLower(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b + ('a' - 'A')
	}
	return b
}
