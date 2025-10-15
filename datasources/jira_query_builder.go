// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/andygrunwald/go-jira"
	"github.com/mattermost/mattermost-plugin-ai/datasources/queryutils"
)

// buildJQLQuery constructs a JQL query based on topic and sections
func (j *JiraProtocol) buildJQLQuery(topic string, sections []string) string {
	var conditions []string

	if topic != "" {
		cleanedTopic := strings.TrimSuffix(topic, "...")

		if strings.Contains(cleanedTopic, " AND ") || strings.Contains(cleanedTopic, " OR ") {
			if j.pluginAPI != nil {
				j.pluginAPI.LogDebug("jira_docs: detected boolean query topic", "topic", TruncateTopicForLogging(cleanedTopic))
			}

			if queryNode, err := ParseBooleanQuery(cleanedTopic); err == nil {
				keywords := ExtractKeywords(queryNode)

				if j.pluginAPI != nil {
					j.pluginAPI.LogDebug("jira_docs: extracted keywords from boolean query",
						"keyword_count", len(keywords),
						"keywords", strings.Join(keywords[:minInt(10, len(keywords))], ", "))
				}

				var searchTerms []string
				for _, keyword := range keywords {
					keyword = strings.TrimSpace(keyword)
					keyword = strings.Trim(keyword, `"`)

					if keyword != "" && len(keyword) > 2 {
						escapedKeyword := j.escapeJQLString(keyword)
						searchTerms = append(searchTerms, fmt.Sprintf(`text ~ "%s"`, escapedKeyword))
						searchTerms = append(searchTerms, fmt.Sprintf(`summary ~ "%s"`, escapedKeyword))
					}
				}

				if len(searchTerms) > 0 {
					if len(searchTerms) > 20 {
						searchTerms = searchTerms[:20]
					}
					textCondition := strings.Join(searchTerms, " OR ")
					conditions = append(conditions, fmt.Sprintf("(%s)", textCondition))
				}
				if len(conditions) == 0 {
					conditions = append(conditions, "updated >= -30d")
				}
				return strings.Join(conditions, " AND ") + " ORDER BY updated DESC"
			}
			if j.pluginAPI != nil {
				j.pluginAPI.LogWarn("jira_docs: failed to parse boolean query",
					"topic", cleanedTopic,
					"error", "parsing failed")
			}
		}

		expandedTerms := j.topicAnalyzer.BuildExpandedSearchTerms(cleanedTopic, MaxExpandedTermsJira)
		if len(expandedTerms) > 0 {
			var validSearchTerms []string
			for _, term := range expandedTerms {
				term = strings.TrimSpace(term)
				if term != "" && !strings.Contains(term, ",") && len(term) > 2 {
					escapedTerm := j.escapeJQLString(term)
					validSearchTerms = append(validSearchTerms, fmt.Sprintf(`text ~ "%s"`, escapedTerm))
				}
			}

			cleanedTopicEscaped := j.escapeJQLString(cleanedTopic)

			if strings.Contains(cleanedTopic, ",") {
				parts := strings.Split(cleanedTopic, ",")
				for _, part := range parts {
					part = strings.TrimSpace(part)
					if part != "" && len(part) > 2 {
						escapedPart := j.escapeJQLString(part)
						validSearchTerms = append(validSearchTerms, fmt.Sprintf(`summary ~ "%s"`, escapedPart))
						validSearchTerms = append(validSearchTerms, fmt.Sprintf(`description ~ "%s"`, escapedPart))
					}
				}
			} else if len(cleanedTopicEscaped) > 2 {
				validSearchTerms = append(validSearchTerms, fmt.Sprintf(`summary ~ "%s"`, cleanedTopicEscaped))
				validSearchTerms = append(validSearchTerms, fmt.Sprintf(`description ~ "%s"`, cleanedTopicEscaped))
			}

			if len(validSearchTerms) > 0 {
				textCondition := strings.Join(validSearchTerms, " OR ")
				conditions = append(conditions, fmt.Sprintf("(%s)", textCondition))
			}
		}
	}

	if len(sections) > 0 {
		issueTypes := make([]string, len(sections))
		for i, section := range sections {
			issueTypes[i] = fmt.Sprintf(`"%s"`, cases.Title(language.English).String(section))
		}
		conditions = append(conditions, fmt.Sprintf("issueType in (%s)", strings.Join(issueTypes, ", ")))
	}

	if len(conditions) == 0 {
		conditions = append(conditions, "updated >= -30d")
	}

	jql := strings.Join(conditions, " AND ") + " ORDER BY updated DESC"

	return jql
}

// searchIssues performs the JQL search and returns issues
func (j *JiraProtocol) searchIssues(ctx context.Context, client *jira.Client, jql string, limit int) ([]jira.Issue, error) {
	fields := []string{
		"summary", "description", "status", "assignee", "creator", "reporter",
		"created", "updated", "issuetype", "labels", "priority", "comment",
		"project", "fixVersions", "components",
	}

	maxResults := limit
	if maxResults > DefaultMaxDocsPerCallJira {
		maxResults = DefaultMaxDocsPerCallJira
	}

	searchOptions := &jira.SearchOptionsV2{
		Fields:     fields,
		MaxResults: maxResults,
	}

	issues, _, err := client.Issue.SearchV2JQL(jql, searchOptions)
	if err != nil {
		return nil, fmt.Errorf("JQL search failed: %w", err)
	}

	return issues, nil
}

// ValidateSearchSyntax tests search queries against the Jira JQL API to validate syntax
func (j *JiraProtocol) ValidateSearchSyntax(ctx context.Context, request ProtocolRequest) (*SyntaxValidationResult, error) {
	result := &SyntaxValidationResult{
		OriginalQuery:    request.Topic,
		IsValidSyntax:    true,
		SyntaxErrors:     []string{},
		SupportsFeatures: []string{"JQL", "text search", "project filtering", "status filtering", "date ranges"},
	}

	if request.Topic == "" {
		result.RecommendedQuery = "mobile"
		return result, nil
	}

	baseURL := request.Source.Endpoints[EndpointBaseURL]
	if baseURL == "" {
		result.IsValidSyntax = false
		result.SyntaxErrors = append(result.SyntaxErrors, "Jira search requires base_url configuration")
		result.RecommendedQuery = request.Topic
		return result, nil
	}

	issueCount := j.testJiraSearchQuery(ctx, baseURL, request.Topic, request.Source)
	result.TestResultCount = issueCount

	if issueCount == 0 {
		simplifiedQuery := j.simplifyJiraQuery(request.Topic)
		simpleCount := j.testJiraSearchQuery(ctx, baseURL, simplifiedQuery, request.Source)

		if simpleCount > 0 {
			result.IsValidSyntax = false
			result.SyntaxErrors = append(result.SyntaxErrors,
				fmt.Sprintf("Complex JQL query returned 0 results, but simplified query returned %d results", simpleCount))
			result.RecommendedQuery = simplifiedQuery
		} else {
			verySimpleQuery := "mobile"
			verySimpleCount := j.testJiraSearchQuery(ctx, baseURL, verySimpleQuery, request.Source)
			if verySimpleCount > 0 {
				result.IsValidSyntax = false
				result.SyntaxErrors = append(result.SyntaxErrors, "JQL query syntax may be too complex or invalid")
				result.RecommendedQuery = verySimpleQuery
			} else {
				result.SyntaxErrors = append(result.SyntaxErrors, "No search results found - may require authentication or project has no content")
				result.RecommendedQuery = request.Topic
			}
		}
	} else {
		result.RecommendedQuery = request.Topic
	}

	return result, nil
}

// testJiraSearchQuery performs a lightweight search test to validate JQL query syntax
func (j *JiraProtocol) testJiraSearchQuery(ctx context.Context, baseURL, topic string, source SourceConfig) int {
	if err := WaitRateLimiter(ctx, j.rateLimiter); err != nil {
		return 0
	}

	sections := []string{"issues"}
	jql := j.buildJQLQuery(topic, sections)

	searchURL := BuildAPIURL(baseURL, "rest/api/2/search")
	searchParams := url.Values{}
	searchParams.Add("jql", jql)
	searchParams.Add("maxResults", "1")
	searchParams.Add("fields", "key")

	fullURL := searchURL + "?" + searchParams.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return 0
	}

	j.addAuthHeaders(req)

	resp, err := j.client.Do(req)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return 0
	}

	var searchResponse struct {
		Total int `json:"total"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&searchResponse); err != nil {
		return 0
	}

	return searchResponse.Total
}

// simplifyJiraQuery creates a Jira JQL-friendly version of a complex query
func (j *JiraProtocol) simplifyJiraQuery(query string) string {
	return queryutils.SimplifyQueryToKeywords(query, 2, "mobile")
}

// escapeJQLString escapes special characters in JQL search strings
// JQL special characters that need escaping: \ " + - & | ! ( ) { } [ ] ^ ~ * ? : /
// Note: Backslash must be escaped first to avoid double-escaping
func (j *JiraProtocol) escapeJQLString(s string) string {
	if s == "" {
		return s
	}

	replacer := strings.NewReplacer(
		`\`, `\\`,
		`"`, `\"`,
		`+`, `\+`,
		`-`, `\-`,
		`&`, `\&`,
		`|`, `\|`,
		`!`, `\!`,
		`(`, `\(`,
		`)`, `\)`,
		`{`, `\{`,
		`}`, `\}`,
		`[`, `\[`,
		`]`, `\]`,
		`^`, `\^`,
		`~`, `\~`,
		`*`, `\*`,
		`?`, `\?`,
		`:`, `\:`,
		`/`, `\/`,
	)

	return replacer.Replace(s)
}
