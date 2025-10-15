// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package thread

import (
	"context"
	"strconv"
)

// BuildEvidenceIndex creates a searchable representation of thread posts
func BuildEvidenceIndex(
	ctx context.Context,
	threadID string,
	posts []Post,
	embedder Embedder,
	opts ValidatorOptions,
) (*EvidenceIndex, error) {
	// 1. Chunk posts into smaller windows
	chunks := ChunkPosts(posts, opts.ChunkSize)

	if len(chunks) == 0 {
		// Empty thread - return empty index
		return &EvidenceIndex{
			Chunks:       []PostChunk{},
			EmbeddingMap: make(map[string][]float32),
			Participants: make(map[string]bool),
			ThreadID:     threadID,
			ModelVersion: embedder.ModelVersion(),
		}, nil
	}

	// 2. Extract chunk texts for batch embedding
	chunkTexts := make([]string, len(chunks))
	for i, chunk := range chunks {
		chunkTexts[i] = chunk.Text
	}

	// 3. Batch embed all chunks
	embeddings, err := embedder.EmbedBatch(ctx, chunkTexts)
	if err != nil {
		return nil, err
	}

	// 4. Store embeddings in chunks and create embedding map
	embeddingMap := make(map[string][]float32, len(chunks))
	for i := range chunks {
		chunks[i].Embedding = embeddings[i]

		// Create key: postID:chunkIndex
		key := chunks[i].PostID + ":" + strconv.Itoa(i)
		embeddingMap[key] = embeddings[i]
	}

	// 5. Build BM25 index for lexical search (optional)
	var bm25Index *BM25Index
	if opts.UseLexical {
		bm25Index = BuildBM25Index(chunks)
	}

	// 6. Extract participant names from posts
	participants := extractParticipants(posts)

	return &EvidenceIndex{
		Chunks:       chunks,
		BM25Index:    bm25Index,
		EmbeddingMap: embeddingMap,
		Participants: participants,
		ThreadID:     threadID,
		ModelVersion: embedder.ModelVersion(),
	}, nil
}

// extractParticipants extracts unique participant names from posts
func extractParticipants(posts []Post) map[string]bool {
	participants := make(map[string]bool)
	for _, post := range posts {
		if post.Author != "" {
			participants[post.Author] = true
		}
	}
	return participants
}

// FindFabricatedParticipants identifies names in summary not in thread participants
func FindFabricatedParticipants(summaryNames []string, threadParticipants map[string]bool) []string {
	fabricated := make([]string, 0)

	for _, name := range summaryNames {
		// Check if name exists in thread participants (case-insensitive)
		found := false
		for participant := range threadParticipants {
			if equalsCaseInsensitive(name, participant) {
				found = true
				break
			}
		}

		if !found {
			fabricated = append(fabricated, name)
		}
	}

	return fabricated
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
