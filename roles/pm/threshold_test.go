// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package pm_test

import (
	"testing"

	"github.com/mattermost/mattermost-plugin-ai/roles/testutils"
	"github.com/stretchr/testify/assert"
)

func TestGetThreshold(t *testing.T) {
	tests := []struct {
		name           string
		totalRubrics   int
		thresholdMode  string
		expectedResult int
	}{
		{
			name:           "STRICT mode with 8 rubrics",
			totalRubrics:   8,
			thresholdMode:  "STRICT",
			expectedResult: 8,
		},
		{
			name:           "MODERATE mode with 8 rubrics",
			totalRubrics:   8,
			thresholdMode:  "MODERATE",
			expectedResult: 5, // (8+1)/2 = 4.5 -> 4, but we want majority so 5
		},
		{
			name:           "LAX mode with 8 rubrics",
			totalRubrics:   8,
			thresholdMode:  "LAX",
			expectedResult: 2,
		},
		{
			name:           "Default mode (invalid) with 8 rubrics",
			totalRubrics:   8,
			thresholdMode:  "INVALID",
			expectedResult: 5, // Should default to MODERATE
		},
		{
			name:           "STRICT mode with 4 rubrics",
			totalRubrics:   4,
			thresholdMode:  "STRICT",
			expectedResult: 4,
		},
		{
			name:           "MODERATE mode with 4 rubrics",
			totalRubrics:   4,
			thresholdMode:  "MODERATE",
			expectedResult: 3, // (4+1)/2 = 2.5 -> 2, but we want majority so 3
		},
		{
			name:           "LAX mode with 3 rubrics",
			totalRubrics:   3,
			thresholdMode:  "LAX",
			expectedResult: 1, // Less than 4, so minimum 1
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testutils.GetThreshold(tt.totalRubrics, tt.thresholdMode)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}
