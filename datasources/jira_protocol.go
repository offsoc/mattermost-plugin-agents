// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/andygrunwald/go-jira"
	"github.com/mattermost/mattermost-plugin-ai/mmapi"
)

// JiraProtocol implements the DataSourceProtocol for Jira REST API
type JiraProtocol struct {
	client          *http.Client
	rateLimiter     *RateLimiter
	auth            AuthConfig
	pluginAPI       mmapi.Client
	htmlProcessor   *HTMLProcessor
	topicAnalyzer   *TopicAnalyzer
	universalScorer *UniversalRelevanceScorer
}

// NewJiraProtocol creates a new Jira protocol instance
func NewJiraProtocol(httpClient *http.Client, pluginAPI mmapi.Client) *JiraProtocol {
	return &JiraProtocol{
		client:          httpClient,
		auth:            AuthConfig{Type: AuthTypeNone},
		pluginAPI:       pluginAPI,
		htmlProcessor:   NewHTMLProcessor(),
		topicAnalyzer:   NewTopicAnalyzer(),
		universalScorer: NewUniversalRelevanceScorer(),
	}
}

// FormatJiraAuth formats Jira authentication credentials into email:token format
// If the input is already in email:token format, it is returned as-is
// If the input is just a token, the provided email is prepended with a colon separator
func FormatJiraAuth(email, token string) string {
	if token == "" {
		return ""
	}

	if strings.Contains(token, ":") {
		return token
	}

	if email == "" {
		email = "user@example.com"
	}

	return email + ":" + token
}

// ParseJiraAuth parses Jira authentication in email:token format
// Returns the email and token components
func ParseJiraAuth(authKey string) (email, token string) {
	if authKey == "" {
		return "", ""
	}

	if strings.Contains(authKey, ":") {
		parts := strings.SplitN(authKey, ":", 2)
		if len(parts) == 2 {
			return parts[0], parts[1]
		}
	}

	return "user@example.com", authKey
}

// Fetch retrieves documents from Jira REST API using JQL search
func (j *JiraProtocol) Fetch(ctx context.Context, request ProtocolRequest) ([]Doc, error) {
	source := request.Source

	if j.pluginAPI != nil {
		j.pluginAPI.LogDebug(source.Name+": starting Jira fetch",
			"sections", request.Sections,
			"limit", request.Limit)
	}

	EnsureRateLimiter(&j.rateLimiter, source.RateLimit)

	if err := WaitRateLimiter(ctx, j.rateLimiter); err != nil {
		if j.pluginAPI != nil {
			j.pluginAPI.LogWarn(source.Name+": rate limiter wait failed", "error", err.Error())
		}
		return nil, fmt.Errorf("rate limiter wait failed: %w", err)
	}

	jiraClient, err := j.createJiraClient(source)
	if err != nil {
		if j.pluginAPI != nil {
			j.pluginAPI.LogWarn(source.Name+": failed to create Jira client", "error", err.Error())
		}
		return nil, fmt.Errorf("failed to create Jira client: %w", err)
	}

	jql := j.buildJQLQuery(request.Topic, request.Sections)

	if j.pluginAPI != nil {
		j.pluginAPI.LogDebug(source.Name+": executing JQL query", "jql", jql)
	}

	issues, err := j.searchIssues(ctx, jiraClient, jql, request.Limit)
	if err != nil {
		if j.pluginAPI != nil {
			j.pluginAPI.LogWarn(source.Name+": JQL search failed", "jql", jql, "error", err.Error())
		}
		return nil, fmt.Errorf("failed to search Jira issues: %w", err)
	}

	if j.pluginAPI != nil {
		j.pluginAPI.LogDebug(source.Name+": JQL search completed", "issue_count", len(issues))
	}

	docs := make([]Doc, 0, len(issues))
	conversionFailed := 0
	for i, issue := range issues {
		doc := j.convertIssueToDoc(issue, source.Name)
		if doc != nil {
			docs = append(docs, *doc)
			if j.pluginAPI != nil && i < 3 {
				j.pluginAPI.LogDebug(source.Name+": issue converted",
					"index", i,
					"key", issue.Key,
					"title", doc.Title)
			}
		} else {
			conversionFailed++
		}
	}

	if j.pluginAPI != nil && conversionFailed > 0 {
		j.pluginAPI.LogDebug(source.Name+": conversion stats",
			"total_issues", len(issues),
			"converted", len(docs),
			"failed", conversionFailed)
	}

	preFilterCount := len(docs)
	filteredDocs := j.applyRelevanceScoring(docs, request.Topic, source.Name)
	relevanceFiltered := preFilterCount - len(filteredDocs)

	preBooleanCount := len(filteredDocs)
	filteredDocs = FilterDocsByBooleanQuery(filteredDocs, request.Topic)
	booleanFiltered := preBooleanCount - len(filteredDocs)

	if j.pluginAPI != nil {
		j.pluginAPI.LogDebug(source.Name+": Jira fetch complete",
			"jql", jql,
			"issues_found", len(issues),
			"docs_converted", len(docs),
			"relevance_filtered", relevanceFiltered,
			"boolean_filtered", booleanFiltered,
			"final_results", len(filteredDocs))
	}

	return filteredDocs, nil
}

// GetType returns the protocol type
func (j *JiraProtocol) GetType() ProtocolType {
	return JiraProtocolType
}

// SetAuth sets the authentication configuration
func (j *JiraProtocol) SetAuth(auth AuthConfig) {
	j.auth = auth
}

// Close closes the protocol and cleans up resources
func (j *JiraProtocol) Close() error {
	CloseRateLimiter(&j.rateLimiter)
	return nil
}

// createJiraClient creates an authenticated Jira client
func (j *JiraProtocol) createJiraClient(source SourceConfig) (*jira.Client, error) {
	baseURL := source.Endpoints[EndpointBaseURL]
	if baseURL == "" {
		return nil, fmt.Errorf("base URL not configured for Jira source")
	}

	if j.pluginAPI != nil {
		j.pluginAPI.LogDebug(source.Name+": Jira auth configuration",
			"auth_type", source.Auth.Type,
			"auth_key_present", source.Auth.Key != "",
			"auth_key_length", len(source.Auth.Key),
			"email_endpoint", source.Endpoints[EndpointEmail],
			"base_url", baseURL)
	}

	if source.Auth.Type == AuthTypeAPIKey && source.Auth.Key != "" {
		emailEndpoint := source.Endpoints[EndpointEmail]
		username, password, err := ParseAtlassianAuth(source.Auth.Key, emailEndpoint)
		if err != nil {
			if j.pluginAPI != nil {
				j.pluginAPI.LogWarn(source.Name+": ParseAtlassianAuth failed",
					"error", err.Error(),
					"auth_key_length", len(source.Auth.Key),
					"email_endpoint", emailEndpoint)
			}
			return nil, fmt.Errorf("Jira authentication error: %w", err) //nolint:staticcheck // Jira is a proper noun
		}

		if j.pluginAPI != nil {
			j.pluginAPI.LogDebug(source.Name+": Jira credentials parsed",
				"username_present", username != "",
				"username_length", len(username),
				"password_present", password != "",
				"password_length", len(password))
		}

		transport := &jira.BasicAuthTransport{
			Username:  username,
			Password:  password,
			Transport: j.client.Transport,
		}

		jiraClient, err := jira.NewClient(transport.Client(), baseURL)
		if err != nil {
			if j.pluginAPI != nil {
				j.pluginAPI.LogWarn(source.Name+": jira.NewClient failed",
					"error", err.Error(),
					"base_url", baseURL)
			}
			return nil, fmt.Errorf("failed to create Jira client: %w", err)
		}

		if j.pluginAPI != nil {
			j.pluginAPI.LogDebug(source.Name + ": Jira client created successfully")
		}

		return jiraClient, nil
	}

	if j.pluginAPI != nil {
		j.pluginAPI.LogWarn(source.Name+": Jira authentication check failed",
			"auth_type", source.Auth.Type,
			"auth_type_matches", source.Auth.Type == AuthTypeAPIKey,
			"auth_key_empty", source.Auth.Key == "",
			"auth_key_length", len(source.Auth.Key))
	}

	return nil, fmt.Errorf("Jira authentication not configured") //nolint:staticcheck // Jira is a proper noun
}

// minInt returns the minimum of two integers
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// addAuthHeaders adds Jira API authentication headers
func (j *JiraProtocol) addAuthHeaders(req *http.Request) {
	if j.auth.Key == "" {
		req.Header.Set(HeaderAccept, AcceptJSON)
		req.Header.Set(HeaderContentType, AcceptJSON)
		return
	}

	switch j.auth.Type {
	case AuthTypeAPIKey:
		authKey := j.auth.Key
		if strings.Contains(authKey, ":") {
			encoded := base64.StdEncoding.EncodeToString([]byte(authKey))
			req.Header.Set(HeaderAuthorization, AuthPrefixBasic+encoded)
		} else {
			req.Header.Set(HeaderAuthorization, AuthPrefixBearer+authKey)
		}
	case AuthTypeToken:
		authKey := j.auth.Key
		if strings.Contains(authKey, ":") {
			encoded := base64.StdEncoding.EncodeToString([]byte(authKey))
			req.Header.Set(HeaderAuthorization, AuthPrefixBasic+encoded)
		} else {
			req.Header.Set(HeaderAuthorization, AuthPrefixBearer+authKey)
		}
	}
	req.Header.Set(HeaderAccept, AcceptJSON)
	req.Header.Set(HeaderContentType, AcceptJSON)
}
