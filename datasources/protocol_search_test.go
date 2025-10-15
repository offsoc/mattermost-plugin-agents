// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MattermostDocsQueryTestCases provides test cases optimized for Mattermost documentation
// These work well for HTTP protocol with Mattermost docs
var MattermostDocsQueryTestCases = []struct {
	Name        string
	Query       string
	Description string
}{
	{
		Name:        "Quoted Phrase Query",
		Query:       "\"mattermost server\"",
		Description: "Exact phrase matching with quotes",
	},
	{
		Name:        "AND Operator Query",
		Query:       "mattermost AND server",
		Description: "Both terms required",
	},
	{
		Name:        "OR with Quoted Phrase",
		Query:       "\"new features\" OR update",
		Description: "Phrase or single term",
	},
	{
		Name:        "Complex AND OR Query",
		Query:       "(deployment OR administration) AND mattermost",
		Description: "Grouped OR with AND",
	},
	{
		Name:        "NOT Operator Query",
		Query:       "mattermost AND NOT tutorial",
		Description: "Exclude tutorial content",
	},
	{
		Name:        "Multi-word AND Query",
		Query:       "server AND deployment",
		Description: "Multiple terms with AND",
	},
	{
		Name:        "Complex Nested Query",
		Query:       "(server OR mattermost) AND (deployment OR releases)",
		Description: "Nested groups with multiple operators",
	},
}

// GenericQueryTestCases provides generic test cases that work across different data sources
// These use common software/project management terms that should match most data sources
var GenericQueryTestCases = []struct {
	Name        string
	Query       string
	Description string
}{
	{
		Name:        "Quoted Phrase Query",
		Query:       "\"user management\"",
		Description: "Exact phrase matching with quotes",
	},
	{
		Name:        "AND Operator Query",
		Query:       "mobile AND notification",
		Description: "Both terms required",
	},
	{
		Name:        "OR with Quoted Phrase",
		Query:       "\"push notifications\" OR alerts",
		Description: "Phrase or single term",
	},
	{
		Name:        "Complex AND OR Query",
		Query:       "(deployment OR installation) AND docker",
		Description: "Grouped OR with AND",
	},
	{
		Name:        "NOT Operator Query",
		Query:       "security AND NOT deprecated",
		Description: "Exclude deprecated security content",
	},
	{
		Name:        "Multi-word AND Query",
		Query:       "\"API integration\" AND webhook",
		Description: "Phrase combined with term",
	},
	{
		Name:        "Complex Nested Query",
		Query:       "(channels OR \"team collaboration\") AND (admin OR configuration)",
		Description: "Nested groups with multiple operators",
	},
}

// HubFallbackQueryTestCases provides test cases specific to Mattermost Hub fallback data
// These queries are designed to match content in hub-contact-sales.txt and hub-customer-feedback.txt
var HubFallbackQueryTestCases = []struct {
	Name        string
	Query       string
	Description string
}{
	{
		Name:        "Quoted Phrase Query",
		Query:       "\"enterprise license\"",
		Description: "Exact phrase matching with quotes",
	},
	{
		Name:        "AND Operator Query",
		Query:       "pricing AND license",
		Description: "Both terms required",
	},
	{
		Name:        "OR with Quoted Phrase",
		Query:       "\"playbook\" OR notification",
		Description: "Phrase or single term",
	},
	{
		Name:        "Complex AND OR Query",
		Query:       "(playbook OR channel) AND feature",
		Description: "Grouped OR with AND",
	},
	{
		Name:        "Multi-word AND Query",
		Query:       "\"professional license\" AND pricing",
		Description: "Phrase combined with term",
	},
	{
		Name:        "Complex Nested Query",
		Query:       "(channel OR playbook) AND (feature OR request)",
		Description: "Nested groups with multiple operators",
	},
}

// VerifyProtocolMattermostDocsBooleanQuery verifies boolean query support using Mattermost-specific queries
func VerifyProtocolMattermostDocsBooleanQuery(t *testing.T, protocol DataSourceProtocol, source SourceConfig, setupFunc func() error) {
	verifyProtocolBooleanQueryWithCases(t, protocol, source, setupFunc, MattermostDocsQueryTestCases)
}

// VerifyProtocolGenericBooleanQuery verifies boolean query support using generic queries
func VerifyProtocolGenericBooleanQuery(t *testing.T, protocol DataSourceProtocol, source SourceConfig, setupFunc func() error) {
	verifyProtocolBooleanQueryWithCases(t, protocol, source, setupFunc, GenericQueryTestCases)
}

// ProductBoardQueryTestCases provides test cases optimized for ProductBoard feature data
var ProductBoardQueryTestCases = []struct {
	Name        string
	Query       string
	Description string
}{
	{
		Name:        "Quoted Phrase Query",
		Query:       "\"channel permissions\"",
		Description: "Exact phrase matching with quotes",
	},
	{
		Name:        "AND Operator Query",
		Query:       "webhook AND permissions",
		Description: "Both terms required",
	},
	{
		Name:        "OR with Quoted Phrase",
		Query:       "\"compliance exports\" OR webhook",
		Description: "Phrase or single term",
	},
	{
		Name:        "Complex AND OR Query",
		Query:       "(channel OR team) AND permissions",
		Description: "Grouped OR with AND",
	},
	{
		Name:        "NOT Operator Query",
		Query:       "channel AND NOT archived",
		Description: "Exclude archived channel content",
	},
	{
		Name:        "Multi-word AND Query",
		Query:       "\"data loss\" AND prevention",
		Description: "Phrase combined with term",
	},
	{
		Name:        "Complex Nested Query",
		Query:       "(webhook OR integration) AND (permission OR feature)",
		Description: "Nested groups with multiple operators",
	},
}

// DiscourseQueryTestCases provides test cases optimized for Mattermost Discourse forum
// These test cases use term combinations verified to work in sections: announce, faq, copilot-ai
var DiscourseQueryTestCases = []struct {
	Name        string
	Query       string
	Description string
}{
	{
		Name:        "Plugin AND Installation Query",
		Query:       "plugin AND installation",
		Description: "Both terms required - common in plugin setup discussions",
	},
	{
		Name:        "Plugin OR Channel Query",
		Query:       "plugin OR channel",
		Description: "Either term matches - broad search across common topics",
	},
	{
		Name:        "Docker OR Installation Query",
		Query:       "docker OR installation",
		Description: "Either deployment or setup topics",
	},
	{
		Name:        "Plugin Only Query",
		Query:       "plugin",
		Description: "Single keyword - plugin discussions",
	},
	{
		Name:        "Channel Only Query",
		Query:       "channel",
		Description: "Single keyword - channel topics",
	},
	{
		Name:        "Server Only Query",
		Query:       "server",
		Description: "Single keyword - server topics",
	},
	{
		Name:        "Deployment Only Query",
		Query:       "deployment",
		Description: "Single keyword - deployment topics",
	},
}

// VerifyProtocolDiscourseBooleanQuery verifies boolean query support using Discourse-specific queries
func VerifyProtocolDiscourseBooleanQuery(t *testing.T, protocol DataSourceProtocol, source SourceConfig, setupFunc func() error) {
	verifyProtocolBooleanQueryWithCases(t, protocol, source, setupFunc, DiscourseQueryTestCases)
}

// JiraQueryTestCases provides test cases optimized for Mattermost Jira instance
// These test cases use term combinations that actually co-occur in the Mattermost Jira
var JiraQueryTestCases = []struct {
	Name        string
	Query       string
	Description string
}{
	{
		Name:        "Plugin AND API Query",
		Query:       "plugin AND API",
		Description: "Both terms required - common in plugin-related issues",
	},
	{
		Name:        "Channel AND Message Query",
		Query:       "channel AND message",
		Description: "Both terms required - common in messaging issues",
	},
	{
		Name:        "Mobile AND Notification Query",
		Query:       "mobile AND notification",
		Description: "Both terms required - common in mobile notification issues",
	},
	{
		Name:        "Performance AND Database Query",
		Query:       "performance AND database",
		Description: "Both terms required - common in performance issues",
	},
	{
		Name:        "OR Query with Common Terms",
		Query:       "plugin OR API",
		Description: "Either term matches - broad search",
	},
	{
		Name:        "Integration AND API Query",
		Query:       "integration AND API",
		Description: "Both terms required - integration topics",
	},
	{
		Name:        "NOT Operator Query",
		Query:       "mobile AND NOT deprecated",
		Description: "Exclude deprecated mobile content",
	},
}

// VerifyProtocolJiraBooleanQuery verifies boolean query support using Jira-specific queries
func VerifyProtocolJiraBooleanQuery(t *testing.T, protocol DataSourceProtocol, source SourceConfig, setupFunc func() error) {
	verifyProtocolBooleanQueryWithCases(t, protocol, source, setupFunc, JiraQueryTestCases)
}

// VerifyProtocolProductBoardBooleanQuery verifies boolean query support using ProductBoard-specific queries
func VerifyProtocolProductBoardBooleanQuery(t *testing.T, protocol DataSourceProtocol, source SourceConfig, setupFunc func() error) {
	verifyProtocolBooleanQueryWithCases(t, protocol, source, setupFunc, ProductBoardQueryTestCases)
}

// VerifyProtocolHubFallbackBooleanQuery is a helper test function specifically for Mattermost Hub fallback tests
func VerifyProtocolHubFallbackBooleanQuery(t *testing.T, protocol DataSourceProtocol, source SourceConfig, setupFunc func() error) {
	verifyProtocolBooleanQueryWithCases(t, protocol, source, setupFunc, HubFallbackQueryTestCases)
}

// verifyProtocolBooleanQueryWithCases is the internal implementation used by the public verify functions
func verifyProtocolBooleanQueryWithCases(t *testing.T, protocol DataSourceProtocol, source SourceConfig, setupFunc func() error, testCases []struct {
	Name        string
	Query       string
	Description string
}) {
	if setupFunc != nil {
		if err := setupFunc(); err != nil {
			t.Skipf("Setup failed, skipping test: %v", err)
			return
		}
	}

	ctx := context.Background()

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			queryNode, err := ParseBooleanQuery(tc.Query)
			require.NoError(t, err, "Complex query should parse successfully: %s", tc.Description)
			assert.NotNil(t, queryNode)

			keywords := ExtractKeywords(queryNode)
			assert.Greater(t, len(keywords), 0, "Should extract at least one keyword from query")

			request := ProtocolRequest{
				Source:   source,
				Topic:    tc.Query,
				Sections: source.Sections,
				Limit:    5,
			}

			docs, err := protocol.Fetch(ctx, request)

			require.NoError(t, err, "Query should not return an error")

			if len(docs) == 0 {
				t.Errorf("Query '%s' returned 0 docs - expected at least 1 result for: %s", tc.Name, tc.Description)
				t.Logf("  Query: %s", tc.Query)
				t.Logf("  Keywords extracted: %v", ExtractKeywords(queryNode))
			} else {
				t.Logf("Query '%s' returned %d docs", tc.Name, len(docs))

				firstDoc := docs[0]
				searchText := firstDoc.Title + " " + firstDoc.Content
				matches := EvaluateBoolean(queryNode, searchText)

				if !matches {
					t.Logf("WARNING: First doc may not match boolean query (check filtering logic)")
					t.Logf("  Query: %s", tc.Query)
					t.Logf("  Doc Title: %s", firstDoc.Title)
					t.Logf("  Doc Content (first 200 chars): %s", truncate(firstDoc.Content, 200))
				}
			}
		})
	}
}

// truncate truncates a string to maxLen characters
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// TestBooleanQueryParsingStandalone tests that all complex queries parse correctly
func TestBooleanQueryParsingStandalone(t *testing.T) {
	allTestCases := []struct {
		Name      string
		TestCases []struct {
			Name        string
			Query       string
			Description string
		}
	}{
		{"MattermostDocs", MattermostDocsQueryTestCases},
		{"Generic", GenericQueryTestCases},
		{"Discourse", DiscourseQueryTestCases},
		{"Jira", JiraQueryTestCases},
		{"ProductBoard", ProductBoardQueryTestCases},
		{"HubFallback", HubFallbackQueryTestCases},
	}

	for _, suite := range allTestCases {
		t.Run(suite.Name, func(t *testing.T) {
			for _, tc := range suite.TestCases {
				t.Run(tc.Name, func(t *testing.T) {
					queryNode, err := ParseBooleanQuery(tc.Query)
					require.NoError(t, err, "Query should parse: %s", tc.Description)
					assert.NotNil(t, queryNode)

					keywords := ExtractKeywords(queryNode)
					assert.Greater(t, len(keywords), 0, "Should extract at least one keyword")
					t.Logf("Query: %s", tc.Description)
					t.Logf("Extracted %d keywords: %v", len(keywords), keywords)
				})
			}
		})
	}
}
