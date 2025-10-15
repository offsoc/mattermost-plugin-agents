// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package semantic

import (
	"context"
	"fmt"
)

// ValidateTextGrounding validates if claims in a text are grounded in evidence texts
// This is the generic core validation function that works for any text validation scenario
func ValidateTextGrounding(
	ctx context.Context,
	claimText string,
	evidenceTexts []string,
	embedder Embedder,
	thresholds ValidationThresholds,
	opts ValidatorOptions,
) (*ValidationResult, error) {
	// 1. Chunk evidence texts into searchable pieces
	evidenceChunks := ChunkTexts(evidenceTexts, opts.ChunkSize, "evidence")

	// 2. Build evidence index (chunking + embedding + BM25)
	index, err := BuildEvidenceIndex(ctx, evidenceChunks, embedder, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to build evidence index: %w", err)
	}

	// 3. Split claim text into sentences
	claimSentences := SplitIntoSentences(claimText)

	if len(claimSentences) == 0 {
		return &ValidationResult{
			ClaimValidations: []ClaimValidation{},
			TotalClaims:      0,
			GroundedCount:    0,
			MarginalCount:    0,
			UngroundedCount:  0,
			GroundingScore:   1.0,
			WeightedScore:    1.0,
			Pass:             true,
			Reasoning:        "Empty claim text - no claims to validate",
			ModelVersion:     embedder.ModelVersion(),
		}, nil
	}

	// 4. Validate each claim sentence
	validations := make([]ClaimValidation, len(claimSentences))
	for i, sentence := range claimSentences {
		validation, err := validateClaim(
			ctx,
			sentence,
			i,
			index,
			embedder,
			thresholds,
			opts,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to validate claim %d: %w", i, err)
		}
		validations[i] = validation
	}

	// 5. Calculate overall score
	result := calculateValidationResult(validations, thresholds, embedder.ModelVersion())

	return &result, nil
}

// validateClaim validates a single claim sentence
func validateClaim(
	ctx context.Context,
	claim string,
	index int,
	evidenceIndex *EvidenceIndex,
	embedder Embedder,
	thresholds ValidationThresholds,
	opts ValidatorOptions,
) (ClaimValidation, error) {
	// Stage 1: Hybrid Retrieval (BM25 + embeddings)
	candidates, err := RetrieveCandidates(ctx, claim, evidenceIndex, embedder, opts)
	if err != nil {
		return ClaimValidation{}, err
	}

	// Stage 2: Validation Heuristics
	flags := ValidationFlags{}

	if len(candidates) == 0 {
		return ClaimValidation{
			Claim:          claim,
			Index:          index,
			TopEvidence:    []Evidence{},
			BestSimilarity: 0.0,
			Status:         StatusUngrounded,
			Flags:          flags,
		}, nil
	}

	bestEvidence := candidates[0]
	flags.HasSemanticSupport = bestEvidence.Similarity >= thresholds.SemanticThreshold

	// Check lexical overlap (if available)
	if opts.UseLexical {
		lexicalScore := ComputeLexicalScore(claim, bestEvidence.ChunkText)
		flags.HasLexicalSupport = lexicalScore >= thresholds.LexicalThreshold
	} else {
		flags.HasLexicalSupport = false
	}

	// Entity matching
	if opts.RequireEntityMatch {
		flags.EntityMatch = CheckEntityMatch(claim, candidates)
	} else {
		flags.EntityMatch = true
	}

	// Number matching
	if opts.RequireNumberMatch {
		flags.NumberMatch = CheckNumberMatch(claim, candidates)
	} else {
		flags.NumberMatch = true
	}

	// Negation consistency
	if opts.CheckNegation {
		flags.NegationConsistent = CheckNegationConsistency(claim, bestEvidence.ChunkText)
	} else {
		flags.NegationConsistent = true
	}

	// Determine status based on flags and similarity
	status := determineStatus(flags, bestEvidence.Similarity, thresholds)

	return ClaimValidation{
		Claim:          claim,
		Index:          index,
		TopEvidence:    candidates,
		BestSimilarity: bestEvidence.Similarity,
		Status:         status,
		Flags:          flags,
	}, nil
}

// determineStatus determines the grounding status based on validation flags
func determineStatus(
	flags ValidationFlags,
	similarity float64,
	thresholds ValidationThresholds,
) ValidationStatus {
	// Must pass ALL required checks
	requiredChecksPassed := flags.EntityMatch &&
		flags.NumberMatch &&
		flags.NegationConsistent

	if !requiredChecksPassed {
		return StatusUngrounded
	}

	// Must have either semantic OR lexical support
	hasSupport := flags.HasSemanticSupport || flags.HasLexicalSupport
	if !hasSupport {
		return StatusUngrounded
	}

	// Classification based on similarity threshold
	if similarity >= thresholds.GroundedThreshold {
		return StatusGrounded
	}
	if similarity >= thresholds.MarginalThreshold {
		return StatusMarginal
	}
	return StatusUngrounded
}

// calculateValidationResult computes overall metrics and pass/fail
func calculateValidationResult(
	validations []ClaimValidation,
	thresholds ValidationThresholds,
	modelVersion string,
) ValidationResult {
	totalClaims := len(validations)
	groundedCount := 0
	marginalCount := 0
	ungroundedCount := 0
	totalLength := 0
	weightedGrounded := 0.0

	for _, v := range validations {
		claimLength := len(v.Claim)
		totalLength += claimLength

		switch v.Status {
		case StatusGrounded:
			groundedCount++
			weightedGrounded += float64(claimLength)
		case StatusMarginal:
			marginalCount++
			weightedGrounded += float64(claimLength) * 0.5
		case StatusUngrounded:
			ungroundedCount++
		}
	}

	// Calculate grounding score (simple percentage)
	groundingScore := 0.0
	if totalClaims > 0 {
		groundingScore = float64(groundedCount+marginalCount) / float64(totalClaims)
	}

	// Calculate weighted score (by claim length)
	weightedScore := 0.0
	if totalLength > 0 {
		weightedScore = weightedGrounded / float64(totalLength)
	}

	// Determine pass/fail
	strictGroundedRate := 0.0
	if totalClaims > 0 {
		strictGroundedRate = float64(groundedCount) / float64(totalClaims)
	}

	pass := groundingScore >= thresholds.PassThreshold &&
		strictGroundedRate >= thresholds.StrictPassRequired

	reasoning := buildReasoning(
		totalClaims,
		groundedCount,
		marginalCount,
		ungroundedCount,
		groundingScore,
		pass,
	)

	return ValidationResult{
		ClaimValidations: validations,
		TotalClaims:      totalClaims,
		GroundedCount:    groundedCount,
		MarginalCount:    marginalCount,
		UngroundedCount:  ungroundedCount,
		GroundingScore:   groundingScore,
		WeightedScore:    weightedScore,
		Pass:             pass,
		Reasoning:        reasoning,
		ModelVersion:     modelVersion,
	}
}

// buildReasoning creates a human-readable explanation
func buildReasoning(
	totalClaims, groundedCount, marginalCount, ungroundedCount int,
	groundingScore float64,
	pass bool,
) string {
	reasoning := fmt.Sprintf(
		"Content grounding: %d/%d claims grounded (%.1f%%), %d marginal, %d ungrounded",
		groundedCount,
		totalClaims,
		groundingScore*100,
		marginalCount,
		ungroundedCount,
	)

	if pass {
		reasoning += ". PASS - Content is well-grounded in evidence"
	} else {
		reasoning += ". FAIL - Content contains ungrounded claims"
	}

	return reasoning
}
