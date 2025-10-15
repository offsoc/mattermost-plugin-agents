// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package thread

import (
	"context"
	"fmt"
)

// ValidateThreadSummary performs thread summary grounding validation
func ValidateThreadSummary(
	ctx context.Context,
	summary string,
	threadPosts []Post,
	embedder Embedder,
	thresholds ValidationThresholds,
	opts ValidatorOptions,
) (*ValidationResult, error) {
	threadID := extractThreadID(threadPosts)

	// 1. Build evidence index (chunking + indexing)
	index, err := BuildEvidenceIndex(ctx, threadID, threadPosts, embedder, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to build evidence index: %w", err)
	}

	// 2. Split summary into sentences
	sentences := SplitIntoSentences(summary)

	if len(sentences) == 0 {
		return &ValidationResult{
			SentenceValidations: []SentenceValidation{},
			TotalSentences:      0,
			GroundedCount:       0,
			MarginalCount:       0,
			UngroundedCount:     0,
			GroundingScore:      1.0, // Empty summary is technically "grounded"
			WeightedScore:       1.0,
			Pass:                true,
			Reasoning:           "Empty summary - no claims to validate",
			ModelVersion:        embedder.ModelVersion(),
		}, nil
	}

	// 3. Extract participants mentioned in summary
	summaryParticipants := ExtractParticipantNames(summary)

	// 4. Validate each sentence
	validations := make([]SentenceValidation, len(sentences))
	for i, sentence := range sentences {
		validation, err := validateSentence(
			ctx,
			sentence,
			i,
			index,
			embedder,
			thresholds,
			opts,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to validate sentence %d: %w", i, err)
		}
		validations[i] = validation
	}

	// 5. Check for fabricated participants
	fabricatedParticipants := FindFabricatedParticipants(
		summaryParticipants,
		index.Participants,
	)

	// 6. Calculate overall score
	result := calculateValidationResult(
		validations,
		fabricatedParticipants,
		thresholds,
		embedder.ModelVersion(),
	)

	return &result, nil
}

// validateSentence validates a single summary sentence
func validateSentence(
	ctx context.Context,
	sentence string,
	index int,
	evidenceIndex *EvidenceIndex,
	embedder Embedder,
	thresholds ValidationThresholds,
	opts ValidatorOptions,
) (SentenceValidation, error) {
	// Stage 1: Hybrid Retrieval (BM25 + embeddings)
	candidates, err := RetrieveCandidates(ctx, sentence, evidenceIndex, embedder, opts)
	if err != nil {
		return SentenceValidation{}, err
	}

	// Stage 2: Validation Heuristics
	flags := ValidationFlags{}

	if len(candidates) == 0 {
		return SentenceValidation{
			Sentence:       sentence,
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
		lexicalScore := ComputeLexicalScore(sentence, bestEvidence.ChunkText)
		flags.HasLexicalSupport = lexicalScore >= thresholds.LexicalThreshold
	} else {
		flags.HasLexicalSupport = false
	}

	// Entity matching
	if opts.RequireEntityMatch {
		flags.EntityMatch = CheckEntityMatch(sentence, candidates)
	} else {
		flags.EntityMatch = true // Not required
	}

	// Number matching
	if opts.RequireNumberMatch {
		flags.NumberMatch = CheckNumberMatch(sentence, candidates)
	} else {
		flags.NumberMatch = true // Not required
	}

	// Negation consistency
	if opts.CheckNegation {
		flags.NegationConsistent = CheckNegationConsistency(sentence, bestEvidence.ChunkText)
	} else {
		flags.NegationConsistent = true // Not checked
	}

	// Attribution validation
	if opts.CheckAttribution {
		flags.AttributionCorrect = CheckAttribution(sentence, candidates)
	} else {
		flags.AttributionCorrect = true // Not checked
	}

	// Determine status based on flags and similarity
	status := determineStatus(flags, bestEvidence.Similarity, thresholds)

	return SentenceValidation{
		Sentence:       sentence,
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
		flags.NegationConsistent &&
		flags.AttributionCorrect

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
	validations []SentenceValidation,
	fabricatedParticipants []string,
	thresholds ValidationThresholds,
	modelVersion string,
) ValidationResult {
	totalSentences := len(validations)
	groundedCount := 0
	marginalCount := 0
	ungroundedCount := 0
	totalLength := 0
	weightedGrounded := 0.0

	for _, v := range validations {
		sentenceLength := len(v.Sentence)
		totalLength += sentenceLength

		switch v.Status {
		case StatusGrounded:
			groundedCount++
			weightedGrounded += float64(sentenceLength)
		case StatusMarginal:
			marginalCount++
			weightedGrounded += float64(sentenceLength) * 0.5 // Half credit for marginal
		case StatusUngrounded:
			ungroundedCount++
		}
	}

	// Calculate grounding score (simple percentage)
	groundingScore := 0.0
	if totalSentences > 0 {
		groundingScore = float64(groundedCount+marginalCount) / float64(totalSentences)
	}

	// Calculate weighted score (by sentence length)
	weightedScore := 0.0
	if totalLength > 0 {
		weightedScore = weightedGrounded / float64(totalLength)
	}

	// Determine pass/fail
	strictGroundedRate := 0.0
	if totalSentences > 0 {
		strictGroundedRate = float64(groundedCount) / float64(totalSentences)
	}

	pass := groundingScore >= thresholds.PassThreshold &&
		strictGroundedRate >= thresholds.StrictPassRequired &&
		len(fabricatedParticipants) == 0

	reasoning := buildReasoning(
		totalSentences,
		groundedCount,
		marginalCount,
		ungroundedCount,
		groundingScore,
		fabricatedParticipants,
		pass,
	)

	return ValidationResult{
		SentenceValidations: validations,
		TotalSentences:      totalSentences,
		GroundedCount:       groundedCount,
		MarginalCount:       marginalCount,
		UngroundedCount:     ungroundedCount,
		GroundingScore:      groundingScore,
		WeightedScore:       weightedScore,
		Pass:                pass,
		Reasoning:           reasoning,
		ModelVersion:        modelVersion,
	}
}

// buildReasoning creates a human-readable explanation
func buildReasoning(
	totalSentences, groundedCount, marginalCount, ungroundedCount int,
	groundingScore float64,
	fabricatedParticipants []string,
	pass bool,
) string {
	reasoning := fmt.Sprintf(
		"Thread summary grounding: %d/%d sentences grounded (%.1f%%), %d marginal, %d ungrounded",
		groundedCount,
		totalSentences,
		groundingScore*100,
		marginalCount,
		ungroundedCount,
	)

	if len(fabricatedParticipants) > 0 {
		reasoning += fmt.Sprintf(
			". Fabricated participants: %v",
			fabricatedParticipants,
		)
	}

	if pass {
		reasoning += ". PASS - Summary is well-grounded in thread content"
	} else {
		reasoning += ". FAIL - Summary contains ungrounded or fabricated content"
	}

	return reasoning
}

// extractThreadID attempts to extract a thread ID from posts
func extractThreadID(posts []Post) string {
	if len(posts) == 0 {
		return ""
	}

	firstPost := posts[0]
	if firstPost.ReplyTo != "" {
		return firstPost.ReplyTo
	}

	return firstPost.ID
}
