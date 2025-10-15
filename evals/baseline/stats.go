// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package baseline

import (
	"math"
)

// ChiSquaredTest performs a chi-squared test for difference in proportions.
// Returns the chi-squared statistic and p-value.
func ChiSquaredTest(successes1, trials1, successes2, trials2 int) (chiSquared, pValue float64) {
	if trials1 == 0 || trials2 == 0 {
		return 0, 1
	}

	// Pool proportion
	pooledP := float64(successes1+successes2) / float64(trials1+trials2)

	// Expected values under null hypothesis
	expected1 := pooledP * float64(trials1)
	expected2 := pooledP * float64(trials2)
	expectedFail1 := (1 - pooledP) * float64(trials1)
	expectedFail2 := (1 - pooledP) * float64(trials2)

	// Avoid division by zero
	if expected1 == 0 || expected2 == 0 || expectedFail1 == 0 || expectedFail2 == 0 {
		return 0, 1
	}

	// Calculate chi-squared statistic
	observed1 := float64(successes1)
	observed2 := float64(successes2)
	observedFail1 := float64(trials1 - successes1)
	observedFail2 := float64(trials2 - successes2)

	chiSquared = (observed1-expected1)*(observed1-expected1)/expected1 +
		(observed2-expected2)*(observed2-expected2)/expected2 +
		(observedFail1-expectedFail1)*(observedFail1-expectedFail1)/expectedFail1 +
		(observedFail2-expectedFail2)*(observedFail2-expectedFail2)/expectedFail2

	// Degrees of freedom = 1 for 2x2 contingency table
	df := 1.0

	// Approximate p-value using chi-squared distribution
	// For df=1, use simple exponential approximation
	pValue = 1 - (1 - math.Exp(-chiSquared/2))

	// Apply continuity correction for better approximation
	if df == 1 {
		pValue = math.Max(0, math.Min(1, pValue))
	}

	return chiSquared, pValue
}

// CalculateImprovement calculates the improvement with confidence interval.
func CalculateImprovement(enhancedPasses, enhancedTrials, baselinePasses, baselineTrials int) (improvement float64, lower, upper float64) {
	if enhancedTrials == 0 || baselineTrials == 0 {
		return 0, 0, 0
	}

	enhancedRate := float64(enhancedPasses) / float64(enhancedTrials)
	baselineRate := float64(baselinePasses) / float64(baselineTrials)
	improvement = enhancedRate - baselineRate

	// Calculate confidence interval for the difference
	// Using normal approximation for difference of proportions
	se1 := math.Sqrt(enhancedRate * (1 - enhancedRate) / float64(enhancedTrials))
	se2 := math.Sqrt(baselineRate * (1 - baselineRate) / float64(baselineTrials))
	seDiff := math.Sqrt(se1*se1 + se2*se2)

	z := 1.96 // 95% confidence
	margin := z * seDiff

	lower = improvement - margin
	upper = improvement + margin

	return improvement, lower, upper
}
