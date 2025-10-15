// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package grounding

import (
	"context"

	"github.com/mattermost/mattermost-plugin-ai/grounding/semantic"
)

// ContentEmbedder is an alias for the semantic.Embedder interface
// Used for content grounding validation
type ContentEmbedder = semantic.Embedder

// ContentGroundingResult wraps semantic validation result with content-specific info
type ContentGroundingResult struct {
	semantic.ValidationResult
	ResponseLength      int
	ToolResultCount     int
	ToolResultsProvided bool
}

// ValidateContentGrounding validates if an LLM response is grounded in tool results
// This distinguishes between:
// - Baseline bots (no tool results) → should have low grounding
// - Enhanced bots (with tool results) → should have high grounding
func ValidateContentGrounding(
	ctx context.Context,
	response string,
	toolResults []string,
	embedder semantic.Embedder,
	thresholds semantic.ValidationThresholds,
	opts semantic.ValidatorOptions,
) (*ContentGroundingResult, error) {
	result, err := semantic.ValidateTextGrounding(
		ctx,
		response,
		toolResults,
		embedder,
		thresholds,
		opts,
	)
	if err != nil {
		return nil, err
	}

	// Wrap with content-specific metadata
	return &ContentGroundingResult{
		ValidationResult:    *result,
		ResponseLength:      len(response),
		ToolResultCount:     len(toolResults),
		ToolResultsProvided: len(toolResults) > 0,
	}, nil
}

// DefaultContentGroundingThresholds returns thresholds tuned for content grounding
func DefaultContentGroundingThresholds() semantic.ValidationThresholds {
	return semantic.ValidationThresholds{
		SemanticThreshold:  0.70, // Slightly lower than thread validation
		LexicalThreshold:   0.55,
		GroundedThreshold:  0.75,
		MarginalThreshold:  0.60,
		PassThreshold:      0.70, // 70% of claims must be grounded or marginal
		StrictPassRequired: 0.40, // 40% must be strictly grounded
	}
}

// DefaultContentGroundingOptions returns options tuned for content grounding
func DefaultContentGroundingOptions() semantic.ValidatorOptions {
	return semantic.ValidatorOptions{
		TopK:               5,
		ChunkSize:          2,
		UseLexical:         true,
		RequireEntityMatch: true,
		RequireNumberMatch: true,
		CheckNegation:      true,
	}
}
