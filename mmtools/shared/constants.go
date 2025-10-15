// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package shared

// Status messages for external docs operations
// Shared across all tool services (dev, pm, etc.)
const (
	StatusCompleted         = "completed"
	StatusSkipped           = "skipped"
	StatusFailed            = "failed"
	StatusNoResults         = "no_results"
	StatusClientUnavailable = "client_unavailable"
)

// Time ranges for analysis
const (
	TimeRangeWeek    = "week"
	TimeRangeMonth   = "month"
	TimeRangeQuarter = "quarter"
	DefaultTimeRange = TimeRangeMonth
)

// Content length limits
const (
	ExcerptMaxLength = 2000
	MinTitleLength   = 5
	MaxDocsPerSource = 5
)

// Data availability responses
const (
	NoDataFoundMessage  = "No relevant data found for %s in the specified timeframe"
	LimitedDataMessage  = "Limited data available for %s"
	SearchFailedMessage = "Search completed but returned no results for %s"
)
