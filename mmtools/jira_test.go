// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package mmtools

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"testing"

	"github.com/andygrunwald/go-jira"
	"github.com/mattermost/mattermost-plugin-ai/config"
	sharedconfig "github.com/mattermost/mattermost-plugin-ai/config/shared"
	"github.com/mattermost/mattermost-plugin-ai/datasources"
	"github.com/stretchr/testify/require"
)

// createTestConfigContainer creates a config.Container with Jira credentials for testing
func createTestConfigContainer(jiraURL, jiraUsername, jiraAPIToken string, jiraEnabled bool) *config.Container {
	container := &config.Container{}
	if jiraEnabled {
		sharedCfg := sharedconfig.Config{
			DataSources: &datasources.Config{
				Sources: []datasources.SourceConfig{
					{
						Name:     "jira_docs",
						Protocol: datasources.JiraProtocolType,
						Enabled:  true,
						Endpoints: map[string]string{
							"base_url": jiraURL,
						},
						Auth: datasources.AuthConfig{
							Key: jiraUsername + ":" + jiraAPIToken,
						},
					},
				},
			},
		}
		sharedConfigJSON, _ := json.Marshal(sharedCfg)
		testConfig := &config.Config{
			RoleConfigs: map[string]json.RawMessage{
				"shared": sharedConfigJSON,
			},
		}
		container.Update(testConfig)
	}
	return container
}

func TestValidateJiraIssueKey(t *testing.T) {
	tests := []struct {
		name      string
		issueKey  string
		expectErr bool
		errMsg    string
	}{
		{
			name:      "valid issue key",
			issueKey:  "MM-1234",
			expectErr: false,
		},
		{
			name:      "valid issue key with letters and numbers",
			issueKey:  "ABC123-456",
			expectErr: false,
		},
		{
			name:      "empty issue key",
			issueKey:  "",
			expectErr: true,
			errMsg:    "issue key cannot be empty",
		},
		{
			name:      "too long issue key",
			issueKey:  "VERYLONGPROJECTKEYVERYLONGPROJECTKEYVERYLONGKEY-1234",
			expectErr: true,
			errMsg:    "issue key is too long",
		},
		{
			name:      "invalid format - missing dash",
			issueKey:  "MM1234",
			expectErr: true,
			errMsg:    "invalid issue key format",
		},
		{
			name:      "invalid format - missing project key",
			issueKey:  "-1234",
			expectErr: true,
			errMsg:    "invalid issue key format",
		},
		{
			name:      "invalid format - missing number",
			issueKey:  "MM-",
			expectErr: true,
			errMsg:    "invalid issue key format",
		},
		{
			name:      "invalid format - letters after dash",
			issueKey:  "MM-ABC",
			expectErr: true,
			errMsg:    "invalid issue key format",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateJiraIssueKey(test.issueKey)
			if test.expectErr {
				require.Error(t, err)
				require.Equal(t, test.errMsg, err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateRequiredFields(t *testing.T) {
	tests := []struct {
		name      string
		fields    []string
		expectErr bool
		errMsg    string
	}{
		{
			name:      "all fields provided",
			fields:    []string{"field1", "field2", "field3"},
			expectErr: false,
		},
		{
			name:      "first field empty",
			fields:    []string{"", "field2", "field3"},
			expectErr: true,
			errMsg:    "field 1 is required but empty",
		},
		{
			name:      "middle field empty",
			fields:    []string{"field1", "", "field3"},
			expectErr: true,
			errMsg:    "field 2 is required but empty",
		},
		{
			name:      "last field empty",
			fields:    []string{"field1", "field2", ""},
			expectErr: true,
			errMsg:    "field 3 is required but empty",
		},
		{
			name:      "no fields provided",
			fields:    []string{},
			expectErr: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateRequiredFields(test.fields...)
			if test.expectErr {
				require.Error(t, err)
				require.Equal(t, test.errMsg, err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestToolSearchJiraIssues_ArgumentValidation(t *testing.T) {
	// Skip if no real Jira configuration available
	jiraToken := os.Getenv("MM_AI_JIRA_TOKEN")
	if jiraToken == "" {
		t.Skip("Skipping Jira integration test - no MM_AI_JIRA_TOKEN environment variable")
	}

	provider := &MMToolProvider{
		httpClient:      &http.Client{},
		configContainer: createTestConfigContainer("https://test.atlassian.net", "test", jiraToken, true),
	}

	tests := []struct {
		name        string
		args        SearchJiraIssuesArgs
		expectError bool
		expectedMsg string
	}{
		{
			name: "valid arguments",
			args: SearchJiraIssuesArgs{
				InstanceURL: "https://mattermost.atlassian.net",
				JQL:         "project = MM",
				MaxResults:  10,
			},
			expectError: false,
		},
		{
			name: "empty JQL",
			args: SearchJiraIssuesArgs{
				InstanceURL: "https://mattermost.atlassian.net",
				JQL:         "",
				MaxResults:  10,
			},
			expectError: true,
			expectedMsg: "invalid parameters to function",
		},
		{
			name: "max results too high gets clamped",
			args: SearchJiraIssuesArgs{
				InstanceURL: "https://mattermost.atlassian.net",
				JQL:         "project = MM",
				MaxResults:  200,
			},
			expectError: false,
		},
		{
			name: "zero max results gets default",
			args: SearchJiraIssuesArgs{
				InstanceURL: "https://mattermost.atlassian.net",
				JQL:         "project = MM",
				MaxResults:  0,
			},
			expectError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			argsGetter := func(args interface{}) error {
				if searchArgs, ok := args.(*SearchJiraIssuesArgs); ok {
					*searchArgs = test.args
					return nil
				}
				return errors.New("invalid args")
			}

			result, err := provider.toolSearchJiraIssues(nil, argsGetter)

			if test.expectError {
				require.Error(t, err)
				require.Equal(t, test.expectedMsg, result)
			} else if err != nil {
				// With real API, we expect either success or a reasonable error message
				// but at least argument validation should pass
				// Real API errors are acceptable for argument validation tests
				require.NotEqual(t, "invalid parameters to function", result)
			}
		})
	}
}

func TestToolCreateJiraIssue_ArgumentValidation(t *testing.T) {
	// Skip if no real Jira configuration available
	jiraToken := os.Getenv("MM_AI_JIRA_TOKEN")
	if jiraToken == "" {
		t.Skip("Skipping Jira integration test - no MM_AI_JIRA_TOKEN environment variable")
	}

	provider := &MMToolProvider{
		httpClient:      &http.Client{},
		configContainer: createTestConfigContainer("https://test.atlassian.net", "test", jiraToken, true),
	}

	tests := []struct {
		name        string
		args        CreateJiraIssueArgs
		expectError bool
		expectedMsg string
	}{
		{
			name: "valid arguments",
			args: CreateJiraIssueArgs{
				InstanceURL: "https://mattermost.atlassian.net",
				ProjectKey:  "MM",
				IssueType:   "Task",
				Summary:     "Test issue",
				Description: "Test description",
			},
			expectError: false,
		},
		{
			name: "missing project key",
			args: CreateJiraIssueArgs{
				InstanceURL: "https://mattermost.atlassian.net",
				ProjectKey:  "",
				IssueType:   "Task",
				Summary:     "Test issue",
			},
			expectError: true,
			expectedMsg: "invalid parameters to function",
		},
		{
			name: "missing issue type",
			args: CreateJiraIssueArgs{
				InstanceURL: "https://mattermost.atlassian.net",
				ProjectKey:  "MM",
				IssueType:   "",
				Summary:     "Test issue",
			},
			expectError: true,
			expectedMsg: "invalid parameters to function",
		},
		{
			name: "missing summary",
			args: CreateJiraIssueArgs{
				InstanceURL: "https://mattermost.atlassian.net",
				ProjectKey:  "MM",
				IssueType:   "Task",
				Summary:     "",
			},
			expectError: true,
			expectedMsg: "invalid parameters to function",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			argsGetter := func(args interface{}) error {
				if createArgs, ok := args.(*CreateJiraIssueArgs); ok {
					*createArgs = test.args
					return nil
				}
				return errors.New("invalid args")
			}

			result, err := provider.toolCreateJiraIssue(nil, argsGetter)

			if test.expectError {
				require.Error(t, err)
				require.Equal(t, test.expectedMsg, result)
			} else if err != nil {
				// With real API, we expect either success or a reasonable error message
				// but at least argument validation should pass
				// Real API errors are acceptable for argument validation tests
				require.NotEqual(t, "invalid parameters to function", result)
			}
		})
	}
}

func TestToolUpdateJiraIssue_ArgumentValidation(t *testing.T) {
	// Skip if no real Jira configuration available
	jiraToken := os.Getenv("MM_AI_JIRA_TOKEN")
	if jiraToken == "" {
		t.Skip("Skipping Jira integration test - no MM_AI_JIRA_TOKEN environment variable")
	}

	provider := &MMToolProvider{
		httpClient:      &http.Client{},
		configContainer: createTestConfigContainer("https://test.atlassian.net", "test", jiraToken, true),
	}

	tests := []struct {
		name        string
		args        UpdateJiraIssueArgs
		expectError bool
		expectedMsg string
	}{
		{
			name: "valid arguments with summary",
			args: UpdateJiraIssueArgs{
				InstanceURL: "https://mattermost.atlassian.net",
				IssueKey:    "MM-1234",
				Summary:     "Updated summary",
			},
			expectError: false,
		},
		{
			name: "valid arguments with status only",
			args: UpdateJiraIssueArgs{
				InstanceURL: "https://mattermost.atlassian.net",
				IssueKey:    "MM-1234",
				Status:      "In Progress",
			},
			expectError: false,
		},
		{
			name: "missing issue key",
			args: UpdateJiraIssueArgs{
				InstanceURL: "https://mattermost.atlassian.net",
				IssueKey:    "",
				Summary:     "Updated summary",
			},
			expectError: true,
			expectedMsg: "invalid parameters to function",
		},
		{
			name: "invalid issue key format",
			args: UpdateJiraIssueArgs{
				InstanceURL: "https://mattermost.atlassian.net",
				IssueKey:    "INVALID",
				Summary:     "Updated summary",
			},
			expectError: true,
			expectedMsg: "invalid parameters to function",
		},
		{
			name: "no fields to update",
			args: UpdateJiraIssueArgs{
				InstanceURL: "https://mattermost.atlassian.net",
				IssueKey:    "MM-1234",
			},
			expectError: true,
			expectedMsg: "invalid parameters to function",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			argsGetter := func(args interface{}) error {
				if updateArgs, ok := args.(*UpdateJiraIssueArgs); ok {
					*updateArgs = test.args
					return nil
				}
				return errors.New("invalid args")
			}

			result, err := provider.toolUpdateJiraIssue(nil, argsGetter)

			if test.expectError {
				require.Error(t, err)
				require.Equal(t, test.expectedMsg, result)
			} else if err != nil {
				// With real API, we expect either success or a reasonable error message
				// but at least argument validation should pass
				// Real API errors are acceptable for argument validation tests
				require.NotEqual(t, "invalid parameters to function", result)
			}
		})
	}
}

func TestToolLinkJiraIssues_ArgumentValidation(t *testing.T) {
	// Skip if no real Jira configuration available
	jiraToken := os.Getenv("MM_AI_JIRA_TOKEN")
	if jiraToken == "" {
		t.Skip("Skipping Jira integration test - no MM_AI_JIRA_TOKEN environment variable")
	}

	provider := &MMToolProvider{
		httpClient:      &http.Client{},
		configContainer: createTestConfigContainer("https://test.atlassian.net", "test", jiraToken, true),
	}

	tests := []struct {
		name        string
		args        LinkJiraIssuesArgs
		expectError bool
		expectedMsg string
	}{
		{
			name: "valid arguments",
			args: LinkJiraIssuesArgs{
				InstanceURL: "https://mattermost.atlassian.net",
				FromIssue:   "MM-1234",
				ToIssue:     "MM-5678",
				LinkType:    "Blocks",
			},
			expectError: false,
		},
		{
			name: "missing from issue",
			args: LinkJiraIssuesArgs{
				InstanceURL: "https://mattermost.atlassian.net",
				FromIssue:   "",
				ToIssue:     "MM-5678",
				LinkType:    "Blocks",
			},
			expectError: true,
			expectedMsg: "invalid parameters to function",
		},
		{
			name: "missing to issue",
			args: LinkJiraIssuesArgs{
				InstanceURL: "https://mattermost.atlassian.net",
				FromIssue:   "MM-1234",
				ToIssue:     "",
				LinkType:    "Blocks",
			},
			expectError: true,
			expectedMsg: "invalid parameters to function",
		},
		{
			name: "missing link type",
			args: LinkJiraIssuesArgs{
				InstanceURL: "https://mattermost.atlassian.net",
				FromIssue:   "MM-1234",
				ToIssue:     "MM-5678",
				LinkType:    "",
			},
			expectError: true,
			expectedMsg: "invalid parameters to function",
		},
		{
			name: "invalid from issue key",
			args: LinkJiraIssuesArgs{
				InstanceURL: "https://mattermost.atlassian.net",
				FromIssue:   "INVALID",
				ToIssue:     "MM-5678",
				LinkType:    "Blocks",
			},
			expectError: true,
			expectedMsg: "invalid parameters to function",
		},
		{
			name: "invalid to issue key",
			args: LinkJiraIssuesArgs{
				InstanceURL: "https://mattermost.atlassian.net",
				FromIssue:   "MM-1234",
				ToIssue:     "INVALID",
				LinkType:    "Blocks",
			},
			expectError: true,
			expectedMsg: "invalid parameters to function",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			argsGetter := func(args interface{}) error {
				if linkArgs, ok := args.(*LinkJiraIssuesArgs); ok {
					*linkArgs = test.args
					return nil
				}
				return errors.New("invalid args")
			}

			result, err := provider.toolLinkJiraIssues(nil, argsGetter)

			if test.expectError {
				require.Error(t, err)
				require.Equal(t, test.expectedMsg, result)
			} else {
				// We expect this to fail with client creation since we don't have a real client
				// but at least we can verify argument validation passes
				require.Contains(t, result, "internal failure")
			}
		})
	}
}

func TestToolGetJiraIssue_ArgumentValidation(t *testing.T) {
	// Skip if no real Jira configuration available
	jiraToken := os.Getenv("MM_AI_JIRA_TOKEN")
	if jiraToken == "" {
		t.Skip("Skipping Jira integration test - no MM_AI_JIRA_TOKEN environment variable")
	}

	provider := &MMToolProvider{
		httpClient:      &http.Client{},
		configContainer: createTestConfigContainer("https://test.atlassian.net", "test", jiraToken, true),
	}

	tests := []struct {
		name        string
		args        GetJiraIssueArgs
		expectError bool
		expectedMsg string
	}{
		{
			name: "valid arguments",
			args: GetJiraIssueArgs{
				InstanceURL: "https://mattermost.atlassian.net",
				IssueKeys:   []string{"MM-1234", "MM-5678"},
			},
			expectError: false,
		},
		{
			name: "invalid issue key format",
			args: GetJiraIssueArgs{
				InstanceURL: "https://mattermost.atlassian.net",
				IssueKeys:   []string{"MM-1234", "INVALID"},
			},
			expectError: true,
			expectedMsg: "invalid parameters to function",
		},
		{
			name: "issue key too long",
			args: GetJiraIssueArgs{
				InstanceURL: "https://mattermost.atlassian.net",
				IssueKeys:   []string{"VERYLONGPROJECTKEYVERYLONGPROJECTKEYVERYLONGKEY-1234"},
			},
			expectError: true,
			expectedMsg: "invalid parameters to function",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			argsGetter := func(args interface{}) error {
				if getArgs, ok := args.(*GetJiraIssueArgs); ok {
					*getArgs = test.args
					return nil
				}
				return errors.New("invalid args")
			}

			result, err := provider.toolGetJiraIssue(nil, argsGetter)

			if test.expectError {
				require.Error(t, err)
				require.Equal(t, test.expectedMsg, result)
			} else if err != nil {
				// With real API, we expect either success or a reasonable error message
				// but at least argument validation should pass
				// Real API errors are acceptable for argument validation tests
				require.NotEqual(t, "invalid parameters to function", result)
			}
		})
	}
}

func TestCreateJiraUser(t *testing.T) {
	provider := &MMToolProvider{}

	tests := []struct {
		name         string
		assignee     string
		instanceURL  string
		expectedUser *jira.User
	}{
		{
			name:        "jira cloud with account id",
			assignee:    "AIDAPKCEVSQ6C2C2V5J6EUVW3I",
			instanceURL: "https://test.atlassian.net",
			expectedUser: &jira.User{
				AccountID: "AIDAPKCEVSQ6C2C2V5J6EUVW3I",
			},
		},
		{
			name:        "jira cloud with username",
			assignee:    "john.doe",
			instanceURL: "https://test.atlassian.net",
			expectedUser: &jira.User{
				Name:      "john.doe",
				AccountID: "john.doe",
			},
		},
		{
			name:        "jira server with username",
			assignee:    "john.doe",
			instanceURL: "https://jira.company.com",
			expectedUser: &jira.User{
				Name: "john.doe",
			},
		},
		{
			name:        "jira data center with username",
			assignee:    "john.doe",
			instanceURL: "https://jira.internal.company.com",
			expectedUser: &jira.User{
				Name: "john.doe",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := provider.createJiraUser(test.assignee, test.instanceURL)
			require.Equal(t, test.expectedUser, result)
		})
	}
}
