// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package dev_test

import (
	"testing"

	"github.com/mattermost/mattermost-plugin-ai/evals/baseline"
)

// FilterScenariosByFlag filters scenarios based on flag
func FilterScenariosByFlag(scenarios []baseline.Scenario, flag string, t *testing.T) []baseline.Scenario {
	var filteredScenarios []baseline.Scenario

	switch flag {
	case "ALL":
		filteredScenarios = scenarios
		t.Logf("Running ALL scenarios (%d questions)", len(filteredScenarios))
	default:
		t.Fatalf("Invalid scenarios flag: %s. Use 'ALL' (use --level for junior/senior selection)", flag)
	}
	return filteredScenarios
}

// getStringField extracts a string field from a map
func getStringField(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// getIntField extracts an int field from a map
func getIntField(m map[string]interface{}, key string) int {
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	return 0
}

// getTopicDisplayWithDescription extracts topic display with description
func getTopicDisplayWithDescription(topic string, t *testing.T) string {
	return topic
}

// CalculateComparisonStats calculates statistical comparison between baseline and enhanced results
func CalculateComparisonStats(baselineResults, enhancedResults baseline.TestResults) baseline.ComparisonResult {
	improvement, improvementLower, improvementUpper := baseline.CalculateImprovement(
		enhancedResults.Passes, enhancedResults.Trials,
		baselineResults.Passes, baselineResults.Trials,
	)

	_, pValue := baseline.ChiSquaredTest(
		enhancedResults.Passes, enhancedResults.Trials,
		baselineResults.Passes, baselineResults.Trials,
	)

	return baseline.ComparisonResult{
		BaselineBot:   baselineResults,
		EnhancedBot:   enhancedResults,
		Improvement:   improvement,
		Significance:  pValue,
		ConfidenceInt: [2]float64{improvementLower, improvementUpper},
	}
}
