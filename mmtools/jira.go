// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package mmtools

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/andygrunwald/go-jira"
	"github.com/mattermost/mattermost-plugin-ai/llm"
)

type GetJiraIssueArgs struct {
	InstanceURL string   `jsonschema_description:"The URL of the Jira instance to get the issue from. Example: 'https://mattermost.atlassian.net'"`
	IssueKeys   []string `jsonschema_description:"The issue keys of the Jira issues to get. Example: 'MM-1234'"`
}

type SearchJiraIssuesArgs struct {
	InstanceURL string `jsonschema_description:"The URL of the Jira instance to search issues from. Example: 'https://mattermost.atlassian.net'"`
	JQL         string `jsonschema_description:"The JQL (Jira Query Language) string to search for issues. Example: 'project = MM AND status != Done'"`
	MaxResults  int    `jsonschema_description:"Maximum number of results to return. Default is 50, maximum is 100"`
}

type CreateJiraIssueArgs struct {
	InstanceURL  string            `jsonschema_description:"The URL of the Jira instance to create the issue in. Example: 'https://mattermost.atlassian.net'"`
	ProjectKey   string            `jsonschema_description:"The project key where the issue will be created. Example: 'MM'"`
	IssueType    string            `jsonschema_description:"The issue type. Example: 'Task', 'Bug', 'Story'"`
	Summary      string            `jsonschema_description:"The issue summary/title"`
	Description  string            `jsonschema_description:"The issue description"`
	Priority     string            `jsonschema_description:"The issue priority. Example: 'High', 'Medium', 'Low'"`
	Labels       []string          `jsonschema_description:"Labels to add to the issue"`
	Assignee     string            `jsonschema_description:"Username of the assignee"`
	CustomFields map[string]string `jsonschema_description:"Custom fields as key-value pairs"`
}

type UpdateJiraIssueArgs struct {
	InstanceURL string   `jsonschema_description:"The URL of the Jira instance. Example: 'https://mattermost.atlassian.net'"`
	IssueKey    string   `jsonschema_description:"The issue key to update. Example: 'MM-1234'"`
	Summary     string   `jsonschema_description:"New summary/title for the issue"`
	Description string   `jsonschema_description:"New description for the issue"`
	Priority    string   `jsonschema_description:"New priority for the issue. Example: 'High', 'Medium', 'Low'"`
	Labels      []string `jsonschema_description:"Labels to set on the issue (replaces existing labels)"`
	Assignee    string   `jsonschema_description:"Username of the new assignee"`
	Status      string   `jsonschema_description:"New status for the issue. Example: 'In Progress', 'Done'"`
}

type LinkJiraIssuesArgs struct {
	InstanceURL string `jsonschema_description:"The URL of the Jira instance. Example: 'https://mattermost.atlassian.net'"`
	FromIssue   string `jsonschema_description:"The issue key that links from. Example: 'MM-1234'"`
	ToIssue     string `jsonschema_description:"The issue key that links to. Example: 'MM-5678'"`
	LinkType    string `jsonschema_description:"The link type. Example: 'Blocks', 'Depends', 'Relates'"`
}

var validJiraIssueKey = regexp.MustCompile(`^([[:alnum:]]+)-([[:digit:]]+)$`)

const (
	defaultMaxResults = 50
	maxAllowedResults = 100
	maxIssueKeyLength = 50
)

func validateJiraIssueKey(issueKey string) error {
	if issueKey == "" {
		return errors.New("issue key cannot be empty")
	}
	if len(issueKey) > maxIssueKeyLength {
		return errors.New("issue key is too long")
	}
	if !validJiraIssueKey.MatchString(issueKey) {
		return errors.New("invalid issue key format")
	}
	return nil
}

func validateRequiredFields(fields ...string) error {
	for i, field := range fields {
		if field == "" {
			return fmt.Errorf("field %d is required but empty", i+1)
		}
	}
	return nil
}

var fetchedFields = []string{
	"summary",
	"description",
	"status",
	"assignee",
	"created",
	"updated",
	"issuetype",
	"labels",
	"reporter",
	"creator",
	"priority",
	"duedate",
	"timetracking",
	"comment",
}

var searchFields = []string{
	"summary",
	"description",
	"status",
	"assignee",
	"updated",
	"issuetype",
	"labels",
	"priority",
}

func formatJiraIssue(issue *jira.Issue) string {
	result := strings.Builder{}
	result.WriteString("Issue Key: ")
	result.WriteString(issue.Key)
	result.WriteRune('\n')

	if issue.Fields != nil {
		result.WriteString("Summary: ")
		result.WriteString(issue.Fields.Summary)
		result.WriteRune('\n')

		result.WriteString("Description: ")
		result.WriteString(issue.Fields.Description)
		result.WriteRune('\n')

		result.WriteString("Status: ")
		if issue.Fields.Status != nil {
			result.WriteString(issue.Fields.Status.Name)
		} else {
			result.WriteString("Unknown")
		}
		result.WriteRune('\n')

		result.WriteString("Assignee: ")
		if issue.Fields.Assignee != nil {
			result.WriteString(issue.Fields.Assignee.DisplayName)
		} else {
			result.WriteString("Unassigned")
		}
		result.WriteRune('\n')

		result.WriteString("Created: ")
		result.WriteString(time.Time(issue.Fields.Created).Format(time.RFC1123))
		result.WriteRune('\n')

		result.WriteString("Updated: ")
		result.WriteString(time.Time(issue.Fields.Updated).Format(time.RFC1123))
		result.WriteRune('\n')

		if issue.Fields.Type.Name != "" {
			result.WriteString("Type: ")
			result.WriteString(issue.Fields.Type.Name)
			result.WriteRune('\n')
		}

		if issue.Fields.Labels != nil {
			result.WriteString("Labels: ")
			result.WriteString(strings.Join(issue.Fields.Labels, ", "))
			result.WriteRune('\n')
		}

		if issue.Fields.Reporter != nil {
			result.WriteString("Reporter: ")
			result.WriteString(issue.Fields.Reporter.DisplayName)
			result.WriteRune('\n')
		} else if issue.Fields.Creator != nil {
			result.WriteString("Creator: ")
			result.WriteString(issue.Fields.Creator.DisplayName)
			result.WriteRune('\n')
		}

		if issue.Fields.Priority != nil {
			result.WriteString("Priority: ")
			result.WriteString(issue.Fields.Priority.Name)
			result.WriteRune('\n')
		}

		if !time.Time(issue.Fields.Duedate).IsZero() {
			result.WriteString("Due Date: ")
			result.WriteString(time.Time(issue.Fields.Duedate).Format(time.RFC1123))
			result.WriteRune('\n')
		}

		if issue.Fields.TimeTracking != nil {
			if issue.Fields.TimeTracking.OriginalEstimate != "" {
				result.WriteString("Original Estimate: ")
				result.WriteString(issue.Fields.TimeTracking.OriginalEstimate)
				result.WriteRune('\n')
			}
			if issue.Fields.TimeTracking.TimeSpent != "" {
				result.WriteString("Time Spent: ")
				result.WriteString(issue.Fields.TimeTracking.TimeSpent)
				result.WriteRune('\n')
			}
			if issue.Fields.TimeTracking.RemainingEstimate != "" {
				result.WriteString("Remaining Estimate: ")
				result.WriteString(issue.Fields.TimeTracking.RemainingEstimate)
				result.WriteRune('\n')
			}
		}

		if issue.Fields.Comments != nil {
			for _, comment := range issue.Fields.Comments.Comments {
				result.WriteString(fmt.Sprintf("Comment from %s at %s: %s\n", comment.Author.DisplayName, comment.Created, comment.Body))
			}
		}
	}

	return result.String()
}

func (p *MMToolProvider) getAuthenticatedJiraClient(instanceURL, username, apiToken string) (*jira.Client, error) {
	// If no per-call credentials provided, try to get from configuration
	if instanceURL == "" || username == "" || apiToken == "" {
		jiraCreds := GetJiraCredentials(p.configContainer)
		if jiraCreds != nil && jiraCreds.Enabled {
			if instanceURL == "" {
				instanceURL = jiraCreds.URL
			}
			if username == "" {
				username = jiraCreds.Username
			}
			if apiToken == "" {
				apiToken = jiraCreds.APIToken
			}
		}
	}

	// Validate HTTP client exists
	if p.httpClient == nil {
		return nil, fmt.Errorf("HTTP client is not initialized")
	}

	// Use the protected HTTP client that's already configured with SSRF protection
	client := p.httpClient

	// Add authentication if credentials available
	if apiToken != "" && username != "" {
		// Create a new client with BasicAuth transport that wraps the protected client's transport
		authTransport := &jira.BasicAuthTransport{
			Username:  username,
			Password:  apiToken,
			Transport: client.Transport,
		}

		// Create a new client with the authenticated transport but keep other settings
		authenticatedClient := &http.Client{
			Transport: authTransport,
			Timeout:   client.Timeout,
			Jar:       client.Jar,
		}

		return jira.NewClient(authenticatedClient, instanceURL)
	}

	// Use the protected client for anonymous access
	return jira.NewClient(client, instanceURL)
}

func (p *MMToolProvider) toolSearchJiraIssues(context *llm.Context, argsGetter llm.ToolArgumentGetter) (string, error) {
	// Check if Jira is configured
	jiraCreds := GetJiraCredentials(p.configContainer)
	if jiraCreds == nil || !jiraCreds.Enabled {
		return "Jira integration not configured", errors.New("Jira integration is not properly configured") //nolint:staticcheck // Jira is a proper noun
	}

	var args SearchJiraIssuesArgs
	err := argsGetter(&args)
	if err != nil {
		return "invalid parameters to function", fmt.Errorf("failed to get arguments for tool SearchJiraIssues: %w", err)
	}

	if args.JQL == "" {
		return "invalid parameters to function", errors.New("JQL query is required")
	}

	// Use instance URL from args or fall back to configuration
	instanceURL := args.InstanceURL
	if instanceURL == "" {
		instanceURL = jiraCreds.URL
	}

	maxResults := args.MaxResults
	if maxResults <= 0 {
		maxResults = defaultMaxResults
	}
	if maxResults > maxAllowedResults {
		maxResults = maxAllowedResults
	}

	client, err := p.getAuthenticatedJiraClient(instanceURL, "", "")
	if err != nil {
		return "internal failure", fmt.Errorf("failed to create Jira client: %w", err)
	}

	issues, _, err := client.Issue.SearchV2JQL(args.JQL, &jira.SearchOptionsV2{
		Fields:     searchFields,
		MaxResults: maxResults,
	})
	if err != nil {
		return "internal failure", fmt.Errorf("failed to search issues: %w", err)
	}

	if len(issues) == 0 {
		return "No issues found matching the JQL query", nil
	}

	result := strings.Builder{}
	result.WriteString(fmt.Sprintf("Found %d issues:\n\n", len(issues)))
	for i := range issues {
		result.WriteString(formatJiraIssue(&issues[i]))
		result.WriteString("------\n")
	}

	return result.String(), nil
}

func (p *MMToolProvider) toolCreateJiraIssue(context *llm.Context, argsGetter llm.ToolArgumentGetter) (string, error) {
	// Check if Jira is configured
	jiraCreds := GetJiraCredentials(p.configContainer)
	if jiraCreds == nil || !jiraCreds.Enabled {
		return "Jira integration not configured", errors.New("Jira integration is not properly configured") //nolint:staticcheck // Jira is a proper noun
	}

	var args CreateJiraIssueArgs
	err := argsGetter(&args)
	if err != nil {
		return "invalid parameters to function", fmt.Errorf("failed to get arguments for tool CreateJiraIssue: %w", err)
	}

	if args.ProjectKey == "" || args.IssueType == "" || args.Summary == "" {
		return "invalid parameters to function", errors.New("project key, issue type, and summary are required")
	}

	// Use instance URL from args or fall back to configuration
	instanceURL := args.InstanceURL
	if instanceURL == "" {
		instanceURL = jiraCreds.URL
	}

	client, err := p.getAuthenticatedJiraClient(instanceURL, "", "")
	if err != nil {
		return "internal failure", fmt.Errorf("failed to create Jira client: %w", err)
	}

	issueFields := &jira.IssueFields{
		Type: jira.IssueType{
			Name: args.IssueType,
		},
		Project: jira.Project{
			Key: args.ProjectKey,
		},
		Summary:     args.Summary,
		Description: args.Description,
	}

	if len(args.Labels) > 0 {
		issueFields.Labels = args.Labels
	}

	if args.Priority != "" {
		issueFields.Priority = &jira.Priority{
			Name: args.Priority,
		}
	}

	if args.Assignee != "" {
		issueFields.Assignee = p.createJiraUser(args.Assignee, instanceURL)
	}

	issue := &jira.Issue{
		Fields: issueFields,
	}

	createdIssue, _, err := client.Issue.Create(issue)
	if err != nil {
		return "internal failure", fmt.Errorf("failed to create issue: %w", err)
	}

	return fmt.Sprintf("Issue created successfully: %s", createdIssue.Key), nil
}

func (p *MMToolProvider) toolUpdateJiraIssue(context *llm.Context, argsGetter llm.ToolArgumentGetter) (string, error) {
	// Check if Jira is configured
	jiraCreds := GetJiraCredentials(p.configContainer)
	if jiraCreds == nil || !jiraCreds.Enabled {
		return "Jira integration not configured", errors.New("Jira integration is not properly configured") //nolint:staticcheck // Jira is a proper noun
	}

	var args UpdateJiraIssueArgs
	err := argsGetter(&args)
	if err != nil {
		return "invalid parameters to function", fmt.Errorf("failed to get arguments for tool UpdateJiraIssue: %w", err)
	}

	if args.IssueKey == "" {
		return "invalid parameters to function", errors.New("issue key is required")
	}

	if len(args.IssueKey) > maxIssueKeyLength || !validJiraIssueKey.MatchString(args.IssueKey) {
		return "invalid parameters to function", errors.New("invalid issue key")
	}

	// Use instance URL from args or fall back to configuration
	instanceURL := args.InstanceURL
	if instanceURL == "" {
		instanceURL = jiraCreds.URL
	}

	// Pre-validate that we have at least one field to update
	hasUpdates := args.Summary != "" || args.Description != "" || args.Priority != "" || len(args.Labels) > 0 || args.Assignee != ""
	if !hasUpdates && args.Status == "" {
		return "invalid parameters to function", errors.New("at least one field must be provided for update")
	}

	client, err := p.getAuthenticatedJiraClient(instanceURL, "", "")
	if err != nil {
		return "internal failure", fmt.Errorf("failed to create Jira client: %w", err)
	}

	updateFields := &jira.IssueFields{}
	hasFieldUpdates := false

	if args.Summary != "" {
		updateFields.Summary = args.Summary
		hasFieldUpdates = true
	}

	if args.Description != "" {
		updateFields.Description = args.Description
		hasFieldUpdates = true
	}

	if args.Priority != "" {
		updateFields.Priority = &jira.Priority{
			Name: args.Priority,
		}
		hasFieldUpdates = true
	}

	if len(args.Labels) > 0 {
		updateFields.Labels = args.Labels
		hasFieldUpdates = true
	}

	if args.Assignee != "" {
		updateFields.Assignee = p.createJiraUser(args.Assignee, instanceURL)
		hasFieldUpdates = true
	}

	if hasFieldUpdates {
		updateIssue := &jira.Issue{
			Key:    args.IssueKey,
			Fields: updateFields,
		}

		_, _, err = client.Issue.Update(updateIssue)
		if err != nil {
			return "internal failure", fmt.Errorf("failed to update issue: %w", err)
		}
	}

	// Handle status transition separately if provided
	if args.Status != "" {
		transitions, _, err := client.Issue.GetTransitions(args.IssueKey)
		if err != nil {
			return "internal failure", fmt.Errorf("failed to get transitions: %w", err)
		}

		var targetTransition *jira.Transition
		for _, transition := range transitions {
			if transition.To.Name == args.Status {
				targetTransition = &transition
				break
			}
		}

		if targetTransition == nil {
			return "internal failure", fmt.Errorf("status '%s' is not a valid transition for this issue", args.Status)
		}

		_, err = client.Issue.DoTransition(args.IssueKey, targetTransition.ID)
		if err != nil {
			return "internal failure", fmt.Errorf("failed to transition issue: %w", err)
		}
	}

	return fmt.Sprintf("Issue %s updated successfully", args.IssueKey), nil
}

func (p *MMToolProvider) toolLinkJiraIssues(context *llm.Context, argsGetter llm.ToolArgumentGetter) (string, error) {
	// Check if Jira is configured
	jiraCreds := GetJiraCredentials(p.configContainer)
	if jiraCreds == nil || !jiraCreds.Enabled {
		return "Jira integration not configured", errors.New("Jira integration is not properly configured") //nolint:staticcheck // Jira is a proper noun
	}

	var args LinkJiraIssuesArgs
	err := argsGetter(&args)
	if err != nil {
		return "invalid parameters to function", fmt.Errorf("failed to get arguments for tool LinkJiraIssues: %w", err)
	}

	if args.FromIssue == "" || args.ToIssue == "" || args.LinkType == "" {
		return "invalid parameters to function", errors.New("from issue, to issue, and link type are required")
	}

	if len(args.FromIssue) > maxIssueKeyLength || !validJiraIssueKey.MatchString(args.FromIssue) {
		return "invalid parameters to function", errors.New("invalid from issue key")
	}

	if len(args.ToIssue) > maxIssueKeyLength || !validJiraIssueKey.MatchString(args.ToIssue) {
		return "invalid parameters to function", errors.New("invalid to issue key")
	}

	// Use instance URL from args or fall back to configuration
	instanceURL := args.InstanceURL
	if instanceURL == "" {
		instanceURL = jiraCreds.URL
	}

	client, err := p.getAuthenticatedJiraClient(instanceURL, "", "")
	if err != nil {
		return "internal failure", fmt.Errorf("failed to create Jira client: %w", err)
	}

	issueLink := &jira.IssueLink{
		Type: jira.IssueLinkType{
			Name: args.LinkType,
		},
		InwardIssue: &jira.Issue{
			Key: args.ToIssue,
		},
		OutwardIssue: &jira.Issue{
			Key: args.FromIssue,
		},
	}

	_, err = client.Issue.AddLink(issueLink)
	if err != nil {
		return "internal failure", fmt.Errorf("failed to create issue link: %w", err)
	}

	return fmt.Sprintf("Successfully linked %s to %s with link type '%s'", args.FromIssue, args.ToIssue, args.LinkType), nil
}

func (p *MMToolProvider) toolGetJiraIssue(context *llm.Context, argsGetter llm.ToolArgumentGetter) (string, error) {
	// Check if Jira is configured
	jiraCreds := GetJiraCredentials(p.configContainer)
	if jiraCreds == nil || !jiraCreds.Enabled {
		return "Jira integration not configured", errors.New("Jira integration is not properly configured") //nolint:staticcheck // Jira is a proper noun
	}

	var args GetJiraIssueArgs
	err := argsGetter(&args)
	if err != nil {
		return "invalid parameters to function", fmt.Errorf("failed to get arguments for tool GetJiraIssue: %w", err)
	}

	// Use instance URL from args or fall back to configuration
	instanceURL := args.InstanceURL
	if instanceURL == "" {
		instanceURL = jiraCreds.URL
	}

	// Fail for over-length issue key. or doesn't look like an issue key
	for _, issueKey := range args.IssueKeys {
		if len(issueKey) > maxIssueKeyLength || !validJiraIssueKey.MatchString(issueKey) {
			return "invalid parameters to function", errors.New("invalid issue key")
		}
	}

	client, err := p.getAuthenticatedJiraClient(instanceURL, "", "")
	if err != nil {
		return "internal failure", fmt.Errorf("failed to create Jira client: %w", err)
	}

	jql := fmt.Sprintf("key in (%s)", strings.Join(args.IssueKeys, ","))
	issues, _, err := client.Issue.SearchV2JQL(jql, &jira.SearchOptionsV2{Fields: fetchedFields})
	if err != nil {
		return "internal failure", fmt.Errorf("failed to get issues: %w", err)
	}
	if issues == nil {
		return "internal failure", fmt.Errorf("failed to get issues: issues not found")
	}

	result := strings.Builder{}
	for i := range issues {
		result.WriteString(formatJiraIssue(&issues[i]))
		result.WriteString("------\n")
	}

	return result.String(), nil
}

// createJiraUser creates a jira.User with appropriate field for Cloud vs Server
func (p *MMToolProvider) createJiraUser(assignee, instanceURL string) *jira.User {
	// Detect if this is Jira Cloud by checking if URL contains atlassian.net
	isJiraCloud := strings.Contains(instanceURL, "atlassian.net")

	if isJiraCloud {
		// For Jira Cloud, check if assignee looks like an AccountID (AIDAPK...)
		// If it does, use AccountID; otherwise treat as a username and use AccountID field
		if strings.HasPrefix(assignee, "AIDAPK") || len(assignee) == 28 {
			// Looks like an AccountID
			return &jira.User{
				AccountID: assignee,
			}
		}
		// Username provided - for Cloud we should ideally look up the AccountID
		// For now, set both fields to maintain compatibility
		return &jira.User{
			Name:      assignee,
			AccountID: assignee, // This may not work but provides fallback
		}
	}
	// For Jira Server/Data Center, use Name field
	return &jira.User{
		Name: assignee,
	}
}
