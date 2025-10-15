// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package shared

import "github.com/mattermost/mattermost-plugin-ai/datasources"

// Config contains configuration shared across multiple roles
// This is stored in RoleConfigs["shared"] to avoid coupling core config to datasources
type Config struct {
	// DataSources provides shared access to external data sources (Confluence, GitHub, Discourse, etc.)
	// used by multiple role-specific bots (PM, Dev) for documentation lookup and research
	DataSources *datasources.Config `json:"dataSources,omitempty"`
}
