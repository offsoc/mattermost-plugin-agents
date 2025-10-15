// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package baseline

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChiSquaredTest_ZeroTrials(t *testing.T) {
	tests := []struct {
		name       string
		successes1 int
		trials1    int
		successes2 int
		trials2    int
	}{
		{
			name:       "Both trials zero",
			successes1: 0,
			trials1:    0,
			successes2: 0,
			trials2:    0,
		},
		{
			name:       "First trial zero",
			successes1: 0,
			trials1:    0,
			successes2: 5,
			trials2:    10,
		},
		{
			name:       "Second trial zero",
			successes1: 5,
			trials1:    10,
			successes2: 0,
			trials2:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chiSquared, pValue := ChiSquaredTest(tt.successes1, tt.trials1, tt.successes2, tt.trials2)
			assert.Equal(t, 0.0, chiSquared, "Chi-squared should be 0 when trials are zero")
			assert.Equal(t, 1.0, pValue, "p-value should be 1 when trials are zero")
		})
	}
}

func TestChiSquaredTest_ZeroExpectedValues(t *testing.T) {
	chiSquared, pValue := ChiSquaredTest(0, 10, 0, 10)
	assert.Equal(t, 0.0, chiSquared, "Chi-squared should be 0 when expected values are zero")
	assert.Equal(t, 1.0, pValue, "p-value should be 1 when expected values are zero")
}

func TestChiSquaredTest_IdenticalProportions(t *testing.T) {
	tests := []struct {
		name       string
		successes1 int
		trials1    int
		successes2 int
		trials2    int
	}{
		{
			name:       "Same rates 50%",
			successes1: 5,
			trials1:    10,
			successes2: 5,
			trials2:    10,
		},
		{
			name:       "Same rates 80%",
			successes1: 8,
			trials1:    10,
			successes2: 16,
			trials2:    20,
		},
		{
			name:       "Same rates 100%",
			successes1: 10,
			trials1:    10,
			successes2: 20,
			trials2:    20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chiSquared, pValue := ChiSquaredTest(tt.successes1, tt.trials1, tt.successes2, tt.trials2)
			assert.InDelta(t, 0.0, chiSquared, 0.0001, "Chi-squared should be near 0 for identical proportions")
			assert.GreaterOrEqual(t, pValue, 0.95, "p-value should be high (>0.95) for identical proportions")
		})
	}
}

func TestChiSquaredTest_VeryDifferentProportions(t *testing.T) {
	tests := []struct {
		name       string
		successes1 int
		trials1    int
		successes2 int
		trials2    int
	}{
		{
			name:       "100% vs 0%",
			successes1: 20,
			trials1:    20,
			successes2: 0,
			trials2:    20,
		},
		{
			name:       "90% vs 10%",
			successes1: 18,
			trials1:    20,
			successes2: 2,
			trials2:    20,
		},
		{
			name:       "80% vs 20%",
			successes1: 16,
			trials1:    20,
			successes2: 4,
			trials2:    20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chiSquared, pValue := ChiSquaredTest(tt.successes1, tt.trials1, tt.successes2, tt.trials2)
			assert.Greater(t, chiSquared, 5.0, "Chi-squared should be large for very different proportions")
			assert.Less(t, pValue, 0.05, "p-value should be low (<0.05) for very different proportions")
		})
	}
}

func TestChiSquaredTest_ModeratelyDifferentProportions(t *testing.T) {
	successes1 := 12
	trials1 := 20
	successes2 := 8
	trials2 := 20

	chiSquared, pValue := ChiSquaredTest(successes1, trials1, successes2, trials2)

	assert.Greater(t, chiSquared, 0.0, "Chi-squared should be positive for different proportions")
	assert.GreaterOrEqual(t, pValue, 0.0, "p-value should be >= 0")
	assert.LessOrEqual(t, pValue, 1.0, "p-value should be <= 1")
}

func TestChiSquaredTest_SmallSampleSize(t *testing.T) {
	successes1 := 2
	trials1 := 3
	successes2 := 1
	trials2 := 3

	chiSquared, pValue := ChiSquaredTest(successes1, trials1, successes2, trials2)

	assert.GreaterOrEqual(t, chiSquared, 0.0, "Chi-squared should be non-negative")
	assert.GreaterOrEqual(t, pValue, 0.0, "p-value should be >= 0")
	assert.LessOrEqual(t, pValue, 1.0, "p-value should be <= 1")
}

func TestChiSquaredTest_LargeSampleSize(t *testing.T) {
	successes1 := 550
	trials1 := 1000
	successes2 := 450
	trials2 := 1000

	chiSquared, pValue := ChiSquaredTest(successes1, trials1, successes2, trials2)

	assert.Greater(t, chiSquared, 10.0, "Chi-squared should be large for significant difference with large sample")
	assert.Less(t, pValue, 0.01, "p-value should be very low for significant difference with large sample")
}

func TestChiSquaredTest_PValueRange(t *testing.T) {
	tests := []struct {
		name       string
		successes1 int
		trials1    int
		successes2 int
		trials2    int
	}{
		{"Case 1", 5, 10, 6, 10},
		{"Case 2", 3, 10, 7, 10},
		{"Case 3", 10, 20, 15, 20},
		{"Case 4", 8, 15, 12, 15},
		{"Case 5", 1, 5, 4, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, pValue := ChiSquaredTest(tt.successes1, tt.trials1, tt.successes2, tt.trials2)
			assert.GreaterOrEqual(t, pValue, 0.0, "p-value must be >= 0")
			assert.LessOrEqual(t, pValue, 1.0, "p-value must be <= 1")
		})
	}
}

func TestChiSquaredTest_Symmetry(t *testing.T) {
	successes1 := 12
	trials1 := 20
	successes2 := 8
	trials2 := 20

	chiSquared1, pValue1 := ChiSquaredTest(successes1, trials1, successes2, trials2)
	chiSquared2, pValue2 := ChiSquaredTest(successes2, trials2, successes1, trials1)

	assert.InDelta(t, chiSquared1, chiSquared2, 0.0001, "Chi-squared should be symmetric")
	assert.InDelta(t, pValue1, pValue2, 0.0001, "p-value should be symmetric")
}

func TestCalculateImprovement_ZeroTrials(t *testing.T) {
	tests := []struct {
		name           string
		enhancedPasses int
		enhancedTrials int
		baselinePasses int
		baselineTrials int
	}{
		{
			name:           "Both trials zero",
			enhancedPasses: 0,
			enhancedTrials: 0,
			baselinePasses: 0,
			baselineTrials: 0,
		},
		{
			name:           "Enhanced trials zero",
			enhancedPasses: 0,
			enhancedTrials: 0,
			baselinePasses: 5,
			baselineTrials: 10,
		},
		{
			name:           "Baseline trials zero",
			enhancedPasses: 5,
			enhancedTrials: 10,
			baselinePasses: 0,
			baselineTrials: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			improvement, lower, upper := CalculateImprovement(tt.enhancedPasses, tt.enhancedTrials, tt.baselinePasses, tt.baselineTrials)
			assert.Equal(t, 0.0, improvement, "Improvement should be 0 when trials are zero")
			assert.Equal(t, 0.0, lower, "Lower bound should be 0 when trials are zero")
			assert.Equal(t, 0.0, upper, "Upper bound should be 0 when trials are zero")
		})
	}
}

func TestCalculateImprovement_NoImprovement(t *testing.T) {
	enhancedPasses := 5
	enhancedTrials := 10
	baselinePasses := 5
	baselineTrials := 10

	improvement, lower, upper := CalculateImprovement(enhancedPasses, enhancedTrials, baselinePasses, baselineTrials)

	assert.InDelta(t, 0.0, improvement, 0.0001, "Improvement should be 0 for identical rates")
	assert.Less(t, lower, 0.0, "Lower bound should be negative (accounting for uncertainty)")
	assert.Greater(t, upper, 0.0, "Upper bound should be positive (accounting for uncertainty)")
	assert.Greater(t, upper-lower, 0.0, "Confidence interval should have positive width")
}

func TestCalculateImprovement_PositiveImprovement(t *testing.T) {
	tests := []struct {
		name            string
		enhancedPasses  int
		enhancedTrials  int
		baselinePasses  int
		baselineTrials  int
		expectedImprove float64
	}{
		{
			name:            "20% improvement (70% vs 50%)",
			enhancedPasses:  7,
			enhancedTrials:  10,
			baselinePasses:  5,
			baselineTrials:  10,
			expectedImprove: 0.2,
		},
		{
			name:            "30% improvement (80% vs 50%)",
			enhancedPasses:  8,
			enhancedTrials:  10,
			baselinePasses:  5,
			baselineTrials:  10,
			expectedImprove: 0.3,
		},
		{
			name:            "50% improvement (100% vs 50%)",
			enhancedPasses:  10,
			enhancedTrials:  10,
			baselinePasses:  5,
			baselineTrials:  10,
			expectedImprove: 0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			improvement, lower, upper := CalculateImprovement(tt.enhancedPasses, tt.enhancedTrials, tt.baselinePasses, tt.baselineTrials)

			assert.InDelta(t, tt.expectedImprove, improvement, 0.0001, "Improvement calculation incorrect")
			assert.Greater(t, improvement, 0.0, "Improvement should be positive")
			assert.Less(t, lower, improvement, "Lower bound should be less than improvement")
			assert.Greater(t, upper, improvement, "Upper bound should be greater than improvement")
			assert.Greater(t, upper-lower, 0.0, "Confidence interval should have positive width")
		})
	}
}

func TestCalculateImprovement_NegativeImprovement(t *testing.T) {
	enhancedPasses := 3
	enhancedTrials := 10
	baselinePasses := 7
	baselineTrials := 10

	improvement, lower, upper := CalculateImprovement(enhancedPasses, enhancedTrials, baselinePasses, baselineTrials)

	assert.InDelta(t, -0.4, improvement, 0.0001, "Should show negative improvement (regression)")
	assert.Less(t, improvement, 0.0, "Improvement should be negative")
	assert.Less(t, lower, improvement, "Lower bound should be less than improvement")
	assert.Greater(t, upper, improvement, "Upper bound should be greater than improvement")
}

func TestCalculateImprovement_PerfectScores(t *testing.T) {
	tests := []struct {
		name            string
		enhancedPasses  int
		enhancedTrials  int
		baselinePasses  int
		baselineTrials  int
		expectedImprove float64
		expectCollapsed bool
	}{
		{
			name:            "Both 100%",
			enhancedPasses:  10,
			enhancedTrials:  10,
			baselinePasses:  10,
			baselineTrials:  10,
			expectedImprove: 0.0,
			expectCollapsed: true,
		},
		{
			name:            "Enhanced 100%, Baseline 50%",
			enhancedPasses:  10,
			enhancedTrials:  10,
			baselinePasses:  5,
			baselineTrials:  10,
			expectedImprove: 0.5,
			expectCollapsed: false,
		},
		{
			name:            "Both 0%",
			enhancedPasses:  0,
			enhancedTrials:  10,
			baselinePasses:  0,
			baselineTrials:  10,
			expectedImprove: 0.0,
			expectCollapsed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			improvement, lower, upper := CalculateImprovement(tt.enhancedPasses, tt.enhancedTrials, tt.baselinePasses, tt.baselineTrials)

			assert.InDelta(t, tt.expectedImprove, improvement, 0.0001, "Improvement calculation incorrect")

			if tt.expectCollapsed {
				assert.Equal(t, lower, upper, "Confidence interval should collapse to a point when variance is zero")
				assert.Equal(t, improvement, lower, "Collapsed interval should equal improvement")
			} else {
				assert.Less(t, lower, upper, "Lower bound should be less than upper bound")
			}

			if !math.IsNaN(lower) && !math.IsInf(lower, 0) {
				assert.LessOrEqual(t, lower, improvement, "Lower bound should be <= improvement")
			}
			if !math.IsNaN(upper) && !math.IsInf(upper, 0) {
				assert.GreaterOrEqual(t, upper, improvement, "Upper bound should be >= improvement")
			}
		})
	}
}

func TestCalculateImprovement_ConfidenceIntervalNarrowsWithLargeSample(t *testing.T) {
	smallSampleImprovement, smallLower, smallUpper := CalculateImprovement(7, 10, 5, 10)
	largeSampleImprovement, largeLower, largeUpper := CalculateImprovement(700, 1000, 500, 1000)

	smallWidth := smallUpper - smallLower
	largeWidth := largeUpper - largeLower

	assert.InDelta(t, 0.2, smallSampleImprovement, 0.0001, "Small sample improvement should be 0.2")
	assert.InDelta(t, 0.2, largeSampleImprovement, 0.0001, "Large sample improvement should be 0.2")
	assert.Less(t, largeWidth, smallWidth, "Confidence interval should be narrower with larger sample size")
}

func TestCalculateImprovement_Uses95PercentConfidence(t *testing.T) {
	enhancedPasses := 70
	enhancedTrials := 100
	baselinePasses := 50
	baselineTrials := 100

	improvement, lower, upper := CalculateImprovement(enhancedPasses, enhancedTrials, baselinePasses, baselineTrials)

	enhancedRate := 0.7
	baselineRate := 0.5
	expectedImprovement := enhancedRate - baselineRate

	se1 := math.Sqrt(enhancedRate * (1 - enhancedRate) / float64(enhancedTrials))
	se2 := math.Sqrt(baselineRate * (1 - baselineRate) / float64(baselineTrials))
	seDiff := math.Sqrt(se1*se1 + se2*se2)
	expectedMargin := 1.96 * seDiff

	assert.InDelta(t, expectedImprovement, improvement, 0.0001, "Improvement calculation incorrect")
	assert.InDelta(t, expectedImprovement-expectedMargin, lower, 0.0001, "Lower bound calculation incorrect")
	assert.InDelta(t, expectedImprovement+expectedMargin, upper, 0.0001, "Upper bound calculation incorrect")
}

func TestCalculateImprovement_ConfidenceIntervalContainsTrue(t *testing.T) {
	tests := []struct {
		name           string
		enhancedPasses int
		enhancedTrials int
		baselinePasses int
		baselineTrials int
	}{
		{"Test 1", 15, 20, 10, 20},
		{"Test 2", 8, 10, 5, 10},
		{"Test 3", 30, 50, 20, 50},
		{"Test 4", 75, 100, 50, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			improvement, lower, upper := CalculateImprovement(tt.enhancedPasses, tt.enhancedTrials, tt.baselinePasses, tt.baselineTrials)

			assert.LessOrEqual(t, lower, improvement, "Lower bound should be <= improvement")
			assert.GreaterOrEqual(t, upper, improvement, "Upper bound should be >= improvement")
			assert.Less(t, lower, upper, "Lower bound should be < upper bound")
		})
	}
}

func TestCalculateImprovement_ReturnValues(t *testing.T) {
	enhancedPasses := 8
	enhancedTrials := 10
	baselinePasses := 5
	baselineTrials := 10

	improvement, lower, upper := CalculateImprovement(enhancedPasses, enhancedTrials, baselinePasses, baselineTrials)

	assert.False(t, math.IsNaN(improvement), "Improvement should not be NaN")
	assert.False(t, math.IsInf(improvement, 0), "Improvement should not be infinite")
	assert.False(t, math.IsNaN(lower), "Lower bound should not be NaN")
	assert.False(t, math.IsInf(lower, 0), "Lower bound should not be infinite")
	assert.False(t, math.IsNaN(upper), "Upper bound should not be NaN")
	assert.False(t, math.IsInf(upper, 0), "Upper bound should not be infinite")
}
