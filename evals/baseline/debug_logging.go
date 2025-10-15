// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package baseline

import (
	"github.com/mattermost/mattermost-plugin-ai/evals"
)

// LogBaselineTypeInfo logs the baseline type and model configuration
func LogBaselineTypeInfo(t *evals.EvalT, modelName, baselineType, botRole string, rolePromptBaseline bool) {
	if rolePromptBaseline {
		t.Logf("WARN: Baseline bot using model: %s (%s prompts WITHOUT data source tools)", modelName, botRole)
	} else {
		t.Logf("WARN: Baseline bot using model: %s (raw LLM with minimal system prompt)", modelName)
	}
	t.Logf("WARN: Enhanced bot using model: %s (%s prompts WITH data source tools)", modelName, botRole)
}

// LogScenarioComparison logs scenario-level comparison results
func LogScenarioComparison(t *evals.EvalT, result ComparisonResult) {
	t.Logf("WARN: Scenario: %s", result.BaselineBot.Scenario)
	t.Logf("WARN:   Baseline (%s): %.1f%% pass rate (%d/%d)",
		result.BaselineBot.BotName, result.BaselineBot.PassRate,
		result.BaselineBot.Passes, result.BaselineBot.Trials)
	t.Logf("WARN:   Enhanced (%s): %.1f%% pass rate (%d/%d)",
		result.EnhancedBot.BotName, result.EnhancedBot.PassRate,
		result.EnhancedBot.Passes, result.EnhancedBot.Trials)
	t.Logf("WARN:   Improvement: %.1f%% (95%% CI: [%.1f%%, %.1f%%])",
		result.Improvement*100, result.ConfidenceInt[0]*100, result.ConfidenceInt[1]*100)
	t.Logf("WARN:   Statistical significance: p=%.4f", result.Significance)
}

// LogDetailedBreakdown logs detailed pass/fail statistics
func LogDetailedBreakdown(t *evals.EvalT, overallBaselinePasses, overallBaselineTrials, overallEnhancedPasses, overallEnhancedTrials int) {
	t.Logf("WARN: Detailed breakdown - Baseline (%d/%d), Enhanced (%d/%d)",
		overallBaselinePasses, overallBaselineTrials, overallEnhancedPasses, overallEnhancedTrials)
}

// LogGroundingBreakdown logs grounding validation statistics
func LogGroundingBreakdown(t *evals.EvalT, baselineGroundingPasses, baselineGroundingTrials, enhancedGroundingPasses, enhancedGroundingTrials int) {
	t.Logf("WARN: Grounding breakdown - Baseline (%d/%d), Enhanced (%d/%d)",
		baselineGroundingPasses, baselineGroundingTrials,
		enhancedGroundingPasses, enhancedGroundingTrials)
}

// LogResponsePreview logs a truncated preview of the bot response
func LogResponsePreview(t *evals.EvalT, currentTrial, totalTrials int, botType, response string, maxLen int) {
	preview := response
	if len(preview) > maxLen {
		preview = preview[:maxLen] + "..."
	}
	t.Logf("DEBUG: TRIAL %d/%d [%s]: Response preview: %s", currentTrial, totalTrials, botType, preview)
}

// LogCombinedGroundingFailure logs why combined grounding validation failed
func LogCombinedGroundingFailure(t *evals.EvalT, logPrefix string, citationPassed, semanticPassed bool, hasToolResults bool, nonMetadataCitationCount int) {
	t.Logf("%s: === COMBINED GROUNDING FAIL ===", logPrefix)
	t.Logf("%s: Citation validation: %v (determines pass/fail), Semantic validation: %v (informational only)", logPrefix, citationPassed, semanticPassed)

	if !hasToolResults {
		t.Logf("%s: Context: Baseline bot with no tool results", logPrefix)
		if nonMetadataCitationCount == 0 {
			t.Logf("%s: Citation FAIL reason: No non-metadata citations found (only metadata citations present)", logPrefix)
		} else {
			t.Logf("%s: Citation FAIL reason: %d non-metadata citations but no validation performed (no tool results or API clients)", logPrefix, nonMetadataCitationCount)
		}
		t.Logf("%s: Semantic SKIPPED: Cannot perform semantic validation without tool results to compare against", logPrefix)
	} else {
		t.Logf("%s: Context: Enhanced bot with %d non-metadata citations", logPrefix, nonMetadataCitationCount)
		if !citationPassed {
			if nonMetadataCitationCount == 0 {
				t.Logf("%s: Citation FAIL reason: No non-metadata citations found", logPrefix)
			} else {
				t.Logf("%s: Citation FAIL reason: Citations present but validation failed (see detailed logs above)", logPrefix)
			}
		}
		if !semanticPassed {
			t.Logf("%s: Semantic FAIL reason: Content not sufficiently grounded in tool results (see detailed logs above)", logPrefix)
		}
	}
}
