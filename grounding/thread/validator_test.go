// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package thread

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockEmbedder provides a simple mock embedder for testing
type MockEmbedder struct {
	embeddings map[string][]float32
}

func NewMockEmbedder() *MockEmbedder {
	return &MockEmbedder{
		embeddings: make(map[string][]float32),
	}
}

func (m *MockEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	// Return a deterministic embedding based on text
	// Similar texts get similar embeddings
	if emb, found := m.embeddings[text]; found {
		return emb, nil
	}

	// Generate simple embedding (just use text length and first char as features)
	embedding := make([]float32, 10)
	for i := 0; i < 10; i++ {
		embedding[i] = float32(len(text)) / 100.0
		if len(text) > 0 {
			embedding[i] += float32(text[0]) / 1000.0
		}
	}

	m.embeddings[text] = embedding
	return embedding, nil
}

func (m *MockEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	results := make([][]float32, len(texts))
	for i, text := range texts {
		emb, err := m.Embed(ctx, text)
		if err != nil {
			return nil, err
		}
		results[i] = emb
	}
	return results, nil
}

func (m *MockEmbedder) ModelVersion() string {
	return "mock-embedder-v1"
}

// SetSimilar sets two texts to have similar embeddings
func (m *MockEmbedder) SetSimilar(text1, text2 string) {
	base := []float32{0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5}
	m.embeddings[text1] = base
	m.embeddings[text2] = base
}

// SetDifferent sets two texts to have different embeddings
func (m *MockEmbedder) SetDifferent(text1, text2 string) {
	m.embeddings[text1] = []float32{0.9, 0.9, 0.9, 0.9, 0.9, 0.9, 0.9, 0.9, 0.9, 0.9}
	m.embeddings[text2] = []float32{0.1, 0.1, 0.1, 0.1, 0.1, 0.1, 0.1, 0.1, 0.1, 0.1}
}

func TestSplitIntoSentences(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "simple sentences",
			input:    "This is sentence one. This is sentence two.",
			expected: []string{"This is sentence one.", "This is sentence two."},
		},
		{
			name:     "abbreviations",
			input:    "Dr. Smith said hello. Mr. Jones agreed.",
			expected: []string{"Dr. Smith said hello.", "Mr. Jones agreed."},
		},
		{
			name:     "question and exclamation",
			input:    "What do you think? I agree! Let's do it.",
			expected: []string{"What do you think?", "I agree!", "Let's do it."},
		},
		{
			name:     "empty input",
			input:    "",
			expected: []string{},
		},
		{
			name:     "single sentence",
			input:    "This is a single sentence.",
			expected: []string{"This is a single sentence."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SplitIntoSentences(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestChunkPosts(t *testing.T) {
	posts := []Post{
		{
			ID:     "post1",
			Author: "john",
			Text:   "First sentence. Second sentence. Third sentence.",
		},
		{
			ID:     "post2",
			Author: "sarah",
			Text:   "Single sentence post.",
		},
	}

	chunks := ChunkPosts(posts, 2) // 2-sentence windows

	// Should have multiple chunks
	assert.Greater(t, len(chunks), 0)

	// All chunks should have metadata
	for _, chunk := range chunks {
		assert.NotEmpty(t, chunk.PostID)
		assert.NotEmpty(t, chunk.Author)
		assert.NotEmpty(t, chunk.Text)
	}
}

func TestBM25Search(t *testing.T) {
	chunks := []PostChunk{
		{PostID: "1", Author: "john", Text: "Redis is a great caching solution"},
		{PostID: "2", Author: "sarah", Text: "I prefer PostgreSQL for caching"},
		{PostID: "3", Author: "mike", Text: "We should monitor memory usage"},
	}

	index := BuildBM25Index(chunks)

	// Search for "caching"
	results := index.Search("caching solution", 2)

	assert.Equal(t, 2, len(results))
	// First result should be most relevant
	assert.Contains(t, results[0].ChunkText, "caching")
}

func TestExtractParticipantNames(t *testing.T) {
	text := "John proposed using Redis. Sarah raised concerns. Mike agreed with John."

	names := ExtractParticipantNames(text)

	assert.Contains(t, names, "John")
	assert.Contains(t, names, "Sarah")
	assert.Contains(t, names, "Mike")
	// Should not contain common words
	assert.NotContains(t, names, "Redis")
}

func TestCheckEntityMatch(t *testing.T) {
	tests := []struct {
		name     string
		sentence string
		evidence []Evidence
		expected bool
	}{
		{
			name:     "entity present",
			sentence: "John proposed using Redis for caching.",
			evidence: []Evidence{
				{ChunkText: "John said we should use Redis"},
			},
			expected: true,
		},
		{
			name:     "entity missing",
			sentence: "Sarah proposed using Redis.",
			evidence: []Evidence{
				{ChunkText: "John said we should use Redis"},
			},
			expected: false,
		},
		{
			name:     "no entities to check",
			sentence: "we should use caching",
			evidence: []Evidence{
				{ChunkText: "caching is important"},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CheckEntityMatch(tt.sentence, tt.evidence)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCheckNumberMatch(t *testing.T) {
	tests := []struct {
		name     string
		sentence string
		evidence []Evidence
		expected bool
	}{
		{
			name:     "number matches exactly",
			sentence: "We have 100 users.",
			evidence: []Evidence{
				{ChunkText: "The system has 100 users"},
			},
			expected: true,
		},
		{
			name:     "number within tolerance",
			sentence: "We have 100 users.",
			evidence: []Evidence{
				{ChunkText: "The system has 101 users"}, // Within 1% tolerance
			},
			expected: true,
		},
		{
			name:     "number mismatch",
			sentence: "We have 100 users.",
			evidence: []Evidence{
				{ChunkText: "The system has 500 users"},
			},
			expected: false,
		},
		{
			name:     "no numbers to check",
			sentence: "We should improve performance.",
			evidence: []Evidence{
				{ChunkText: "Performance is important"},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CheckNumberMatch(tt.sentence, tt.evidence)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCheckNegationConsistency(t *testing.T) {
	tests := []struct {
		name     string
		sentence string
		evidence string
		expected bool
	}{
		{
			name:     "both positive",
			sentence: "We decided to approve the feature.",
			evidence: "Team approved the feature",
			expected: true,
		},
		{
			name:     "both negative",
			sentence: "We decided not to approve the feature.",
			evidence: "Team did not approve the feature",
			expected: true,
		},
		{
			name:     "negation flip - positive to negative",
			sentence: "We decided to approve the feature.",
			evidence: "Team did not approve the feature",
			expected: false,
		},
		{
			name:     "negation flip - negative to positive",
			sentence: "We decided not to approve the feature.",
			evidence: "Team approved the feature",
			expected: false,
		},
		{
			name:     "no decision verbs",
			sentence: "We discussed the feature.",
			evidence: "Team talked about the feature",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CheckNegationConsistency(tt.sentence, tt.evidence)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCheckAttribution(t *testing.T) {
	tests := []struct {
		name     string
		sentence string
		evidence []Evidence
		expected bool
	}{
		{
			name:     "correct attribution",
			sentence: "John said we should use Redis.",
			evidence: []Evidence{
				{Author: "john", ChunkText: "We should use Redis"},
			},
			expected: true,
		},
		{
			name:     "wrong attribution",
			sentence: "Sarah said we should use Redis.",
			evidence: []Evidence{
				{Author: "john", ChunkText: "We should use Redis"},
			},
			expected: false,
		},
		{
			name:     "no attribution claim",
			sentence: "We should use Redis for caching.",
			evidence: []Evidence{
				{Author: "john", ChunkText: "Redis is good for caching"},
			},
			expected: true,
		},
		{
			name:     "according to pattern",
			sentence: "According to Mike, we need more memory.",
			evidence: []Evidence{
				{Author: "mike", ChunkText: "We need more memory"},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CheckAttribution(tt.sentence, tt.evidence)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateThreadSummary_Grounded(t *testing.T) {
	ctx := context.Background()

	posts := []Post{
		{
			ID:        "post1",
			Author:    "john",
			Timestamp: time.Now(),
			Text:      "I think we should use Redis for caching. It's fast and reliable.",
		},
		{
			ID:        "post2",
			Author:    "sarah",
			Timestamp: time.Now(),
			Text:      "I'm concerned about memory usage. Redis can be memory-intensive.",
		},
		{
			ID:        "post3",
			Author:    "mike",
			Timestamp: time.Now(),
			Text:      "Good point Sarah. We should monitor it closely.",
		},
	}

	summary := "John proposed using Redis for caching. Sarah raised concerns about memory usage. Mike agreed to monitor it."

	embedder := NewMockEmbedder()

	post1Text := posts[0].Text
	post2Text := posts[1].Text
	post3Text := posts[2].Text

	embedder.SetSimilar("John proposed using Redis for caching.", post1Text)
	embedder.SetSimilar("Sarah raised concerns about memory usage.", post2Text)
	embedder.SetSimilar("Mike agreed to monitor it.", post3Text)

	thresholds := DefaultThresholds()
	opts := DefaultOptions()

	result, err := ValidateThreadSummary(ctx, summary, posts, embedder, thresholds, opts)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 3, result.TotalSentences)
	assert.True(t, result.Pass, "Well-grounded summary should pass")
	assert.Greater(t, result.GroundingScore, 0.7)
}

func TestValidateThreadSummary_FabricatedParticipant(t *testing.T) {
	ctx := context.Background()

	posts := []Post{
		{
			ID:        "post1",
			Author:    "john",
			Timestamp: time.Now(),
			Text:      "We should use Redis for caching.",
		},
	}

	// Summary mentions "Alice" who isn't in the thread
	summary := "John and Alice decided to use Redis for caching."

	embedder := NewMockEmbedder()
	embedder.SetSimilar("John and Alice decided to use Redis for caching.", "We should use Redis for caching.")

	thresholds := DefaultThresholds()
	opts := DefaultOptions()

	result, err := ValidateThreadSummary(ctx, summary, posts, embedder, thresholds, opts)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Pass, "Summary with fabricated participant should fail")
	assert.Contains(t, result.Reasoning, "Fabricated participants")
}

func TestValidateThreadSummary_AttributionError(t *testing.T) {
	ctx := context.Background()

	posts := []Post{
		{
			ID:        "post1",
			Author:    "sarah",
			Timestamp: time.Now(),
			Text:      "I prefer PostgreSQL for caching.",
		},
	}

	// Summary incorrectly attributes Sarah's statement to John
	summary := "John said we should use PostgreSQL for caching."

	embedder := NewMockEmbedder()
	embedder.SetSimilar("John said we should use PostgreSQL for caching.", "I prefer PostgreSQL for caching.")

	thresholds := DefaultThresholds()
	opts := DefaultOptions()

	result, err := ValidateThreadSummary(ctx, summary, posts, embedder, thresholds, opts)

	require.NoError(t, err)
	assert.NotNil(t, result)

	// Check that attribution error was detected
	// The content matches sarah's post, but summary says "John said"
	// So attribution should be incorrect
	foundAttributionError := false
	for _, sv := range result.SentenceValidations {
		if !sv.Flags.AttributionCorrect {
			foundAttributionError = true
			assert.Equal(t, StatusUngrounded, sv.Status)
		}
	}
	assert.True(t, foundAttributionError, "Should detect attribution error")
}

func TestValidateThreadSummary_NegationFlip(t *testing.T) {
	ctx := context.Background()

	posts := []Post{
		{
			ID:        "post1",
			Author:    "john",
			Timestamp: time.Now(),
			Text:      "The team decided not to approve the feature for this release.",
		},
	}

	// Summary flips the negation
	summary := "The team decided to approve the feature for this release."

	embedder := NewMockEmbedder()
	embedder.SetSimilar("The team decided to approve the feature for this release.", "The team decided not to approve the feature for this release.")

	thresholds := DefaultThresholds()
	opts := DefaultOptions()

	result, err := ValidateThreadSummary(ctx, summary, posts, embedder, thresholds, opts)

	require.NoError(t, err)
	assert.NotNil(t, result)

	// Check that negation flip was detected
	foundNegationError := false
	for _, sv := range result.SentenceValidations {
		if !sv.Flags.NegationConsistent {
			foundNegationError = true
			assert.Equal(t, StatusUngrounded, sv.Status)
		}
	}
	assert.True(t, foundNegationError, "Should detect negation flip")
}

func TestValidateThreadSummary_NumberMismatch(t *testing.T) {
	ctx := context.Background()

	posts := []Post{
		{
			ID:        "post1",
			Author:    "john",
			Timestamp: time.Now(),
			Text:      "We currently have 50 active users on the platform.",
		},
	}

	// Summary has wrong number (off by 10x)
	summary := "We currently have 500 active users on the platform."

	embedder := NewMockEmbedder()
	embedder.SetSimilar("We currently have 500 active users on the platform.", "We currently have 50 active users on the platform.")

	thresholds := DefaultThresholds()
	opts := DefaultOptions()

	result, err := ValidateThreadSummary(ctx, summary, posts, embedder, thresholds, opts)

	require.NoError(t, err)
	assert.NotNil(t, result)

	// Check that number mismatch was detected
	foundNumberError := false
	for _, sv := range result.SentenceValidations {
		if !sv.Flags.NumberMatch {
			foundNumberError = true
			assert.Equal(t, StatusUngrounded, sv.Status)
		}
	}
	assert.True(t, foundNumberError, "Should detect number mismatch")
}

func TestValidateThreadSummary_EmptySummary(t *testing.T) {
	ctx := context.Background()

	posts := []Post{
		{ID: "post1", Author: "john", Text: "Some content"},
	}

	summary := ""

	embedder := NewMockEmbedder()
	thresholds := DefaultThresholds()
	opts := DefaultOptions()

	result, err := ValidateThreadSummary(ctx, summary, posts, embedder, thresholds, opts)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 0, result.TotalSentences)
	assert.True(t, result.Pass) // Empty summary is technically valid
}

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a        []float32
		b        []float32
		expected float64
	}{
		{
			name:     "identical vectors",
			a:        []float32{1.0, 0.0, 0.0},
			b:        []float32{1.0, 0.0, 0.0},
			expected: 1.0,
		},
		{
			name:     "orthogonal vectors",
			a:        []float32{1.0, 0.0},
			b:        []float32{0.0, 1.0},
			expected: 0.0,
		},
		{
			name:     "opposite vectors",
			a:        []float32{1.0, 0.0},
			b:        []float32{-1.0, 0.0},
			expected: -1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cosineSimilarity(tt.a, tt.b)
			assert.InDelta(t, tt.expected, result, 0.01)
		})
	}
}
