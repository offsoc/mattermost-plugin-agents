// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package grounding

// DefaultThresholds returns balanced thresholds suitable for most cases
func DefaultThresholds() Thresholds {
	return Thresholds{
		MinCitationRate:   0.70,
		MinMetadataFields: 2,
		CitationWeight:    0.7,
		MetadataWeight:    0.3,
	}
}

// StrictThresholds returns high bar for production/critical evaluations
func StrictThresholds() Thresholds {
	return Thresholds{
		MinCitationRate:   0.85,
		MinMetadataFields: 3,
		CitationWeight:    0.8,
		MetadataWeight:    0.2,
	}
}

// LaxThresholds returns lenient thresholds for development/testing
func LaxThresholds() Thresholds {
	return Thresholds{
		MinCitationRate:   0.50,
		MinMetadataFields: 1,
		CitationWeight:    0.6,
		MetadataWeight:    0.4,
	}
}
