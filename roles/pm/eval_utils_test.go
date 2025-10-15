// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package pm_test

import (
	"context"
	"fmt"
	"os"

	"github.com/mattermost/mattermost-plugin-ai/evals"
	"github.com/mattermost/mattermost-plugin-ai/grounding"
	"github.com/mattermost/mattermost-plugin-ai/grounding/thread"
	"github.com/mattermost/mattermost-plugin-ai/roles"
	"github.com/mattermost/mattermost-plugin-ai/roles/testutils"
	"github.com/mattermost/mattermost-plugin-ai/semanticcache"
)

// Type aliases for shared threshold types
type ThresholdEvaluationResult = testutils.ThresholdEvaluationResult

// EvaluateRubricsWithThreshold evaluates all rubrics using threshold-based logic
// Uses direct LLMRubric calls for simplicity and consistency across all tests
func EvaluateRubricsWithThreshold(
	evalT *evals.EvalT,
	rubrics []string,
	response string,
	thresholdMode string,
	trialNum, totalTrials int,
	logPrefix string,
) (*ThresholdEvaluationResult, []string) {
	evalT.Logf("%s: Evaluating %d rubrics...", logPrefix, len(rubrics))

	var rubricResults []*evals.RubricResult
	var errors []string
	passedRubrics := 0

	for rubricIdx, rubric := range rubrics {
		rubricResult, err := evalT.LLMRubric(rubric, response)
		if err != nil {
			evalT.Logf("%s: RUBRIC %d/%d EVALUATION ERROR: %s - %v",
				logPrefix, rubricIdx+1, len(rubrics),
				testutils.TruncateString(rubric, 100), err)
			errors = append(errors, fmt.Sprintf("Trial %d, rubric '%s': %v", trialNum, rubric, err))
			break
		}

		rubricResults = append(rubricResults, rubricResult)

		if rubricResult.Pass {
			passedRubrics++
			evalT.Logf("%s: RUBRIC %d/%d PASS (score: %.2f): %s",
				logPrefix, rubricIdx+1, len(rubrics), rubricResult.Score,
				testutils.TruncateString(rubric, 100))
		} else {
			evalT.Logf("%s: RUBRIC %d/%d FAIL (score: %.2f): %s",
				logPrefix, rubricIdx+1, len(rubrics), rubricResult.Score,
				testutils.TruncateString(rubric, 100))
		}
	}

	// Calculate threshold and determine pass/fail
	threshold := testutils.GetThreshold(len(rubrics), thresholdMode)
	thresholdPassed := passedRubrics >= threshold

	result := &ThresholdEvaluationResult{
		PassedRubrics:   passedRubrics,
		TotalRubrics:    len(rubrics),
		Threshold:       threshold,
		ThresholdMode:   thresholdMode,
		ThresholdPassed: thresholdPassed,
		RubricResults:   rubricResults,
	}

	if thresholdPassed {
		evalT.Logf("%s: OVERALL PASS - %d/%d rubrics passed (threshold: %d, mode: %s)",
			logPrefix, passedRubrics, len(rubrics), threshold, thresholdMode)
	} else {
		evalT.Logf("%s: OVERALL FAIL - %d/%d rubrics passed (threshold: %d, mode: %s)",
			logPrefix, passedRubrics, len(rubrics), threshold, thresholdMode)
	}

	return result, errors
}

// LogThresholdConfiguration logs the current threshold configuration
func LogThresholdConfiguration(evalT *evals.EvalT, thresholdMode string) {
	evalT.Logf("=== THRESHOLD CONFIGURATION ===")
	evalT.Logf("Threshold mode: %s", thresholdMode)

	// Show what each mode means for 8 rubrics (most common case)
	strictThreshold := testutils.GetThreshold(8, "STRICT")
	moderateThreshold := testutils.GetThreshold(8, "MODERATE")
	laxThreshold := testutils.GetThreshold(8, "LAX")

	evalT.Logf("  STRICT: %d/8 rubrics must pass", strictThreshold)
	evalT.Logf("  MODERATE: %d/8 rubrics must pass (DEFAULT)", moderateThreshold)
	evalT.Logf("  LAX: %d/8 rubrics must pass", laxThreshold)
	evalT.Logf("Current setting requires: %d/8 rubrics", testutils.GetThreshold(8, thresholdMode))
	evalT.Logf("===============================")
}

// EvaluateGrounding performs grounding validation on a response
// Supports API validation via environment variables (see roles.CreateAPIClients for details)
func EvaluateGrounding(evalT *evals.EvalT, response string, toolResults []string, logPrefix string) *grounding.Result {
	result := evals.EvaluateGroundingWithLogging(
		evalT,
		response,
		toolResults,              // Tool results from conversation
		nil,                      // No metadata
		roles.CreateAPIClients(), // API clients for GitHub/Jira verification (nil if credentials not provided)
		grounding.DefaultThresholds(),
		logPrefix,
	)
	return &result
}

// EvaluateSemanticGrounding validates if a response is semantically grounded in tool results
// Reuses thread grounding's semantic similarity engine to detect content fabrication
func EvaluateSemanticGrounding(
	evalT *evals.EvalT,
	response string,
	toolResults []string,
	logPrefix string,
) *thread.ValidationResult {
	// Convert tool results to Post format (simple type conversion)
	posts := make([]thread.Post, 0, len(toolResults))
	for i, result := range toolResults {
		if result == "" {
			continue
		}
		posts = append(posts, thread.Post{
			ID:     fmt.Sprintf("tool_%d", i),
			Author: "tool",
			Text:   result,
		})
	}

	if len(posts) == 0 {
		evalT.Logf("%s: No tool results for semantic grounding", logPrefix)
		return &thread.ValidationResult{
			Pass:      false,
			Reasoning: "No tool results available to validate against",
		}
	}

	// Create embedder
	embedder, err := createEmbedderForGrounding()
	if err != nil {
		evalT.Logf("%s: Skipping semantic grounding - %v", logPrefix, err)
		return &thread.ValidationResult{
			Pass:      true, // Don't fail if embedder unavailable
			Reasoning: "Embedder not available",
		}
	}

	// Configure validation for baseline use case (disable thread-specific checks)
	// NOTE: PM responses are analytical syntheses, not verbatim quotes
	// Thresholds are relaxed to account for executive summary style
	thresholds := thread.DefaultThresholds()
	thresholds.PassThreshold = 0.40      // 40% sentences must be grounded (relaxed for analytical content)
	thresholds.StrictPassRequired = 0.20 // 20% strictly grounded (relaxed for analytical content)
	thresholds.SemanticThreshold = 0.60  // Lower threshold for semantic similarity (analytical vs verbatim)
	thresholds.GroundedThreshold = 0.70  // Lowered from 0.75 (allow more synthesis)
	thresholds.MarginalThreshold = 0.55  // Lowered from 0.60 (allow borderline matches)

	opts := thread.DefaultOptions()
	opts.CheckNegation = false    // Not relevant for Q&A responses
	opts.CheckAttribution = false // No speakers in tool results

	evalT.Logf("%s: === SEMANTIC GROUNDING (INFORMATIONAL) ===", logPrefix)
	evalT.Logf("%s: Validating response against %d tool result chunks (diagnostic metric, does not affect pass/fail)", logPrefix, len(posts))

	// Use thread validator directly!
	result, err := thread.ValidateThreadSummary(
		context.Background(),
		response,
		posts,
		embedder,
		thresholds,
		opts,
	)

	if err != nil {
		evalT.Logf("%s: ERROR - Semantic grounding validation failed: %v", logPrefix, err)
		return &thread.ValidationResult{
			Pass:      false,
			Reasoning: fmt.Sprintf("Validation error: %v", err),
		}
	}

	// Log detailed results
	evalT.Logf("%s: Grounding breakdown: %d/%d sentences grounded (%.1f%%)",
		logPrefix, result.GroundedCount, result.TotalSentences, result.GroundingScore*100)

	if result.MarginalCount > 0 {
		evalT.Logf("%s:   - %d sentences marginal support", logPrefix, result.MarginalCount)
	}
	if result.UngroundedCount > 0 {
		evalT.Logf("%s:   - %d sentences ungrounded (fabricated/generic)", logPrefix, result.UngroundedCount)

		// Show examples of ungrounded sentences
		ungroundedCount := 0
		for _, sv := range result.SentenceValidations {
			if sv.Status == thread.StatusUngrounded && ungroundedCount < 3 {
				evalT.Logf("%s:     UNGROUNDED: \"%s\" (similarity: %.2f)",
					logPrefix, testutils.TruncateString(sv.Sentence, 80), sv.BestSimilarity)
				ungroundedCount++
			}
		}
	}

	if result.Pass {
		evalT.Logf("%s: SEMANTIC GROUNDING PASS (informational) - %s", logPrefix, result.Reasoning)
	} else {
		evalT.Logf("%s: SEMANTIC GROUNDING FAIL (informational) - %s", logPrefix, result.Reasoning)
	}

	return result
}

// createEmbedderForGrounding creates an embedder for semantic grounding validation
func createEmbedderForGrounding() (thread.Embedder, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY not set")
	}

	return semanticcache.NewThreadEmbedder(apiKey), nil
}
