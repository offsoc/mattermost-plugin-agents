// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package testutils

import (
	"strings"

	"github.com/mattermost/mattermost-plugin-ai/evals"
)

// ThresholdMode represents the different evaluation threshold modes
type ThresholdMode string

const (
	ThresholdStrict   ThresholdMode = "STRICT"
	ThresholdModerate ThresholdMode = "MODERATE"
	ThresholdLax      ThresholdMode = "LAX"
)

// ThresholdEvaluationResult contains the results of a threshold-based evaluation
type ThresholdEvaluationResult struct {
	PassedRubrics   int
	TotalRubrics    int
	Threshold       int
	ThresholdMode   string
	ThresholdPassed bool
	RubricResults   []*evals.RubricResult
}

// GetThreshold calculates the rubric pass threshold based on the threshold mode.
// Returns the minimum number of rubrics that must pass for the evaluation to succeed.
//
// Modes:
//   - STRICT: All rubrics must pass (100% pass rate)
//   - MODERATE: True majority must pass (more than half)
//   - LAX: Minimum 2 rubrics for 4+ rubrics, 1 rubric otherwise
//
// Examples:
//
//	GetThreshold(8, "STRICT")   // Returns: 8 (all must pass)
//	GetThreshold(8, "MODERATE") // Returns: 5 (majority of 8)
//	GetThreshold(8, "LAX")      // Returns: 2 (minimum for LAX with 4+)
//	GetThreshold(3, "LAX")      // Returns: 1 (minimum for LAX with <4)
func GetThreshold(totalRubrics int, thresholdMode string) int {
	switch ThresholdMode(strings.ToUpper(thresholdMode)) {
	case ThresholdStrict:
		return totalRubrics
	case ThresholdLax:
		if totalRubrics >= 4 {
			return 2
		}
		return 1
	case ThresholdModerate:
		fallthrough
	default:
		return (totalRubrics / 2) + 1
	}
}
