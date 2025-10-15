// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package thread

import (
	"context"
	"time"
)

// Embedder provides embedding functionality for thread validation
type Embedder interface {
	// Embed generates an embedding for a single text
	Embed(ctx context.Context, text string) ([]float32, error)

	// EmbedBatch generates embeddings for multiple texts
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)

	// ModelVersion returns the embedding model version/name
	ModelVersion() string
}

// Post represents a single post in a thread
type Post struct {
	ID        string    // Post ID
	Author    string    // Username
	Timestamp time.Time // Post timestamp
	Text      string    // Post content
	ReplyTo   string    // Parent post ID (if reply)
}

// PostChunk represents a piece of a post (for better granularity)
type PostChunk struct {
	PostID     string    // Parent post ID
	Author     string    // Post author
	Text       string    // Chunk text (sentence or window)
	StartIndex int       // Character start in original post
	EndIndex   int       // Character end in original post
	Embedding  []float32 // Cached embedding (optional)
}

// Evidence represents supporting evidence for a summary sentence
type Evidence struct {
	PostID     string  // Supporting post ID
	ChunkText  string  // Text snippet that supports claim
	Author     string  // Post author
	Similarity float64 // Semantic similarity score
	Rank       int     // Rank in retrieval results
}

// SentenceValidation contains validation result for one summary sentence
type SentenceValidation struct {
	Sentence       string           // The sentence from summary
	Index          int              // Sentence index in summary
	TopEvidence    []Evidence       // Top-k supporting chunks
	BestSimilarity float64          // Best similarity score
	Status         ValidationStatus // Grounding status
	Flags          ValidationFlags  // What checks passed/failed
}

// ValidationStatus represents the grounding status
type ValidationStatus string

const (
	StatusGrounded   ValidationStatus = "grounded"   // Well-supported
	StatusMarginal   ValidationStatus = "marginal"   // Weak support
	StatusUngrounded ValidationStatus = "ungrounded" // No support
)

// ValidationFlags tracks which validation checks passed
type ValidationFlags struct {
	HasSemanticSupport bool // Embedding similarity passed
	HasLexicalSupport  bool // Keyword match passed
	EntityMatch        bool // Named entities present in evidence
	NumberMatch        bool // Numbers/dates match
	NegationConsistent bool // No negation flip detected
	AttributionCorrect bool // Speaker attribution correct (if applicable)
}

// ValidationResult contains overall validation results
type ValidationResult struct {
	SentenceValidations []SentenceValidation
	TotalSentences      int
	GroundedCount       int
	MarginalCount       int
	UngroundedCount     int
	GroundingScore      float64 // Percentage grounded
	WeightedScore       float64 // Length-weighted score
	Pass                bool    // Meets threshold
	Reasoning           string  // Human-readable explanation
	ModelVersion        string  // Embedding model used
}

// ValidationThresholds defines pass/fail criteria
type ValidationThresholds struct {
	SemanticThreshold  float64 // Min cosine similarity (default: 0.75)
	LexicalThreshold   float64 // Min BM25 score (default: 0.60)
	GroundedThreshold  float64 // Combined threshold for "grounded" (default: 0.80)
	MarginalThreshold  float64 // Combined threshold for "marginal" (default: 0.65)
	PassThreshold      float64 // Min % sentences grounded+marginal (default: 0.75)
	StrictPassRequired float64 // Min % sentences strictly grounded (default: 0.50)
}

// DefaultThresholds returns sensible default thresholds
func DefaultThresholds() ValidationThresholds {
	return ValidationThresholds{
		SemanticThreshold:  0.75,
		LexicalThreshold:   0.60,
		GroundedThreshold:  0.80,
		MarginalThreshold:  0.65,
		PassThreshold:      0.75, // 75% grounded+marginal
		StrictPassRequired: 0.50, // 50% strictly grounded
	}
}

// ValidatorOptions configures validation behavior
type ValidatorOptions struct {
	TopK               int  // Number of evidence chunks to retrieve (default: 5)
	ChunkSize          int  // Sentences per chunk (default: 2)
	UseLexical         bool // Enable BM25 retrieval (default: true)
	RequireEntityMatch bool // Require entity presence (default: true)
	RequireNumberMatch bool // Require number exactness (default: true)
	CheckNegation      bool // Detect negation flips (default: true)
	CheckAttribution   bool // Verify speaker attribution (default: true)
}

// DefaultOptions returns sensible default options
func DefaultOptions() ValidatorOptions {
	return ValidatorOptions{
		TopK:               5,
		ChunkSize:          2, // 2-sentence windows
		UseLexical:         true,
		RequireEntityMatch: true,
		RequireNumberMatch: true,
		CheckNegation:      true,
		CheckAttribution:   true,
	}
}

// Validator interface for extensibility
type Validator interface {
	ValidateThreadSummary(
		ctx context.Context,
		summary string,
		threadPosts []Post,
		embedder Embedder,
		thresholds ValidationThresholds,
		opts ValidatorOptions,
	) (*ValidationResult, error)
}

// EvidenceIndex manages the searchable representation of thread posts
type EvidenceIndex struct {
	Chunks       []PostChunk          // All post chunks
	BM25Index    *BM25Index           // Lexical index (optional)
	EmbeddingMap map[string][]float32 // Chunk embeddings cache
	Participants map[string]bool      // Thread participants
	ThreadID     string               // For caching
	ModelVersion string               // Embedding model version
}

// BM25Index represents a simple BM25 lexical index
type BM25Index struct {
	Chunks        []PostChunk      // Reference to chunks
	TermFrequency map[string][]int // term -> chunk indices
	DocFrequency  map[string]int   // term -> number of docs containing term
	AvgDocLength  float64          // Average document length
	K1            float64          // BM25 parameter (default: 1.5)
	B             float64          // BM25 parameter (default: 0.75)
}
