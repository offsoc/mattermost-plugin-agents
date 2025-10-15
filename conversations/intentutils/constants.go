// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package intentutils

// Weight multipliers for pattern scoring
const (
	WeightHigh       = 1.0
	WeightMediumHigh = 0.9
	WeightMedium     = 0.8
	WeightMediumLow  = 0.7
	WeightLow        = 0.6
)

// Common confidence values
const (
	MaxConfidenceScore = 1.0
	NoConfidence       = 0.0
)

// Intent confidence thresholds
// Standardized across all role-based bots (PM, Dev, etc.)
const (
	HighConfidenceThreshold    = 0.9
	MediumConfidenceThreshold  = 0.8
	LowConfidenceThreshold     = 0.6
	MinimumConfidenceThreshold = 0.5
)
