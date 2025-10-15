// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package mmtools

import (
	"github.com/mattermost/mattermost-plugin-ai/config"
	sharedconfig "github.com/mattermost/mattermost-plugin-ai/config/shared"
	"github.com/mattermost/mattermost-plugin-ai/datasources"
)

// JiraCredentials contains Jira authentication information
type JiraCredentials struct {
	URL      string
	Username string
	APIToken string
	Enabled  bool
}

// GetJiraCredentials extracts Jira credentials from shared datasources config
// Returns nil if Jira is not configured or disabled
func GetJiraCredentials(configContainer *config.Container) *JiraCredentials {
	var sharedCfg sharedconfig.Config
	if err := configContainer.GetRoleConfig("shared", &sharedCfg); err != nil {
		return nil
	}

	if sharedCfg.DataSources == nil {
		return nil
	}

	for _, source := range sharedCfg.DataSources.Sources {
		if source.Name == "jira_docs" && source.Enabled {
			url, hasURL := source.Endpoints["base_url"]
			if !hasURL || url == "" {
				continue
			}

			if source.Auth.Key == "" {
				continue
			}

			username, token := datasources.ParseJiraAuth(source.Auth.Key)
			if username == "" || token == "" {
				continue
			}

			return &JiraCredentials{
				URL:      url,
				Username: username,
				APIToken: token,
				Enabled:  true,
			}
		}
	}

	return nil
}
