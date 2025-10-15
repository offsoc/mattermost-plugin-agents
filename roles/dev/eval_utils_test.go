// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package dev_test

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
func EvaluateSemanticGrounding(
	evalT *evals.EvalT,
	response string,
	toolResults []string,
	logPrefix string,
) *thread.ValidationResult {
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

	embedder, err := createEmbedderForGrounding()
	if err != nil {
		evalT.Logf("%s: Skipping semantic grounding - %v", logPrefix, err)
		return &thread.ValidationResult{
			Pass:      true,
			Reasoning: "Embedder not available",
		}
	}

	thresholds := thread.DefaultThresholds()
	thresholds.PassThreshold = 0.70
	thresholds.StrictPassRequired = 0.40
	thresholds.SemanticThreshold = 0.70
	thresholds.GroundedThreshold = 0.75
	thresholds.MarginalThreshold = 0.60

	opts := thread.DefaultOptions()
	opts.CheckNegation = false
	opts.CheckAttribution = false

	evalT.Logf("%s: === SEMANTIC GROUNDING ===", logPrefix)
	evalT.Logf("%s: Validating response against %d tool result chunks", logPrefix, len(posts))

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

	evalT.Logf("%s: Grounding breakdown: %d/%d sentences grounded (%.1f%%)",
		logPrefix, result.GroundedCount, result.TotalSentences, result.GroundingScore*100)

	if result.MarginalCount > 0 {
		evalT.Logf("%s:   - %d sentences marginal support", logPrefix, result.MarginalCount)
	}
	if result.UngroundedCount > 0 {
		evalT.Logf("%s:   - %d sentences ungrounded (fabricated/generic)", logPrefix, result.UngroundedCount)

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
		evalT.Logf("%s: SEMANTIC GROUNDING PASS - %s", logPrefix, result.Reasoning)
	} else {
		evalT.Logf("%s: SEMANTIC GROUNDING FAIL - %s", logPrefix, result.Reasoning)
	}

	return result
}

func createEmbedderForGrounding() (thread.Embedder, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY not set")
	}

	return semanticcache.NewThreadEmbedder(apiKey), nil
}
