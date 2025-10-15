// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"time"
)

// PRFilterOptions contains optional filters for PR fetching
type PRFilterOptions struct {
	// Since filters PRs updated after this time
	Since *time.Time
	// Until filters PRs updated before this time
	Until *time.Time
	// Author filters by PR author login (exact match)
	Author string
	// State filters by PR state: "open", "closed", "all" (default: "all")
	State string
	// Labels filters by label names (PR must have all specified labels)
	Labels []string
}
