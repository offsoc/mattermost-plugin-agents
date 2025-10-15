// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package semantic

import "context"

// Embedder provides embedding functionality for semantic validation
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
	ModelVersion() string
}

// EvidenceChunk represents a piece of evidence text with its embedding
// Generic structure that works for any text validation scenario
type EvidenceChunk struct {
	ID         string
	Text       string
	Embedding  []float32
	Metadata   map[string]string
	StartIndex int
	EndIndex   int
}

// Evidence represents supporting evidence for a claim
type Evidence struct {
	ChunkID    string
	ChunkText  string
	Similarity float64
	Rank       int
	Metadata   map[string]string
}

// EvidenceIndex manages the searchable representation of evidence
type EvidenceIndex struct {
	Chunks       []EvidenceChunk
	BM25Index    *BM25Index
	EmbeddingMap map[string][]float32
	ModelVersion string
}

// BM25Index represents a simple BM25 lexical index
type BM25Index struct {
	Chunks        []EvidenceChunk
	TermFrequency map[string][]int
	DocFrequency  map[string]int
	AvgDocLength  float64
	K1            float64
	B             float64
}

// ClaimValidation contains validation result for one claim
type ClaimValidation struct {
	Claim          string
	Index          int
	TopEvidence    []Evidence
	BestSimilarity float64
	Status         ValidationStatus
	Flags          ValidationFlags
}

// ValidationStatus represents the grounding status
type ValidationStatus string

const (
	StatusGrounded   ValidationStatus = "grounded"
	StatusMarginal   ValidationStatus = "marginal"
	StatusUngrounded ValidationStatus = "ungrounded"
)

// ValidationFlags tracks which validation checks passed
type ValidationFlags struct {
	HasSemanticSupport bool
	HasLexicalSupport  bool
	EntityMatch        bool
	NumberMatch        bool
	NegationConsistent bool
}

// ValidationResult contains overall validation results
type ValidationResult struct {
	ClaimValidations []ClaimValidation
	TotalClaims      int
	GroundedCount    int
	MarginalCount    int
	UngroundedCount  int
	GroundingScore   float64
	WeightedScore    float64
	Pass             bool
	Reasoning        string
	ModelVersion     string
}

// ValidationThresholds defines pass/fail criteria
type ValidationThresholds struct {
	SemanticThreshold  float64
	LexicalThreshold   float64
	GroundedThreshold  float64
	MarginalThreshold  float64
	PassThreshold      float64
	StrictPassRequired float64
}

// DefaultThresholds returns sensible default thresholds
func DefaultThresholds() ValidationThresholds {
	return ValidationThresholds{
		SemanticThreshold:  0.75,
		LexicalThreshold:   0.60,
		GroundedThreshold:  0.80,
		MarginalThreshold:  0.65,
		PassThreshold:      0.75,
		StrictPassRequired: 0.50,
	}
}

// ValidatorOptions configures validation behavior
type ValidatorOptions struct {
	TopK               int
	ChunkSize          int
	UseLexical         bool
	RequireEntityMatch bool
	RequireNumberMatch bool
	CheckNegation      bool
}

// DefaultOptions returns sensible default options
func DefaultOptions() ValidatorOptions {
	return ValidatorOptions{
		TopK:               5,
		ChunkSize:          2,
		UseLexical:         true,
		RequireEntityMatch: true,
		RequireNumberMatch: true,
		CheckNegation:      true,
	}
}
