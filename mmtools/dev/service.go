// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package dev

import (
	"context"
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-plugin-ai/datasources"
	"github.com/mattermost/mattermost-plugin-ai/llm"
	"github.com/mattermost/mattermost-plugin-ai/mmapi"
	"github.com/mattermost/mattermost-plugin-ai/mmtools/shared"
	"github.com/mattermost/mattermost-plugin-ai/search"
	"github.com/mattermost/mattermost-plugin-ai/semanticcache"
)

// Service implements the DevBot tool service
type Service struct {
	pluginAPI               mmapi.Client
	search                  *search.Search
	internalAnalyzer        *shared.InternalAnalyzer
	mockLoader              shared.MockLoader
	commonDataSourcesClient *datasources.Client
	vectorCache             *semanticcache.SimpleCache
	metadataProvider        shared.MetadataProvider
	cacheHelper             *shared.CacheHelper
}

// NewService creates a new DevBot tool service
func NewService(
	pluginAPI mmapi.Client,
	searchService *search.Search,
	mockLoader shared.MockLoader,
	commonDataSourcesClient *datasources.Client,
	vectorCache *semanticcache.SimpleCache,
	metadataProvider shared.MetadataProvider,
) *Service {
	return &Service{
		pluginAPI:               pluginAPI,
		search:                  searchService,
		internalAnalyzer:        shared.NewInternalAnalyzer(searchService, mockLoader, pluginAPI),
		mockLoader:              mockLoader,
		commonDataSourcesClient: commonDataSourcesClient,
		vectorCache:             vectorCache,
		metadataProvider:        metadataProvider,
		cacheHelper:             shared.NewCacheHelper(vectorCache, pluginAPI),
	}
}

// ExplainCodePattern explains Mattermost code patterns with examples
func (s *Service) ExplainCodePattern(llmContext *llm.Context, args ExplainCodePatternArgs, queryBuilder shared.MetadataProvider) (string, error) {
	return s.cacheHelper.ExecuteWithCache(ToolNameExplainCodePattern, args, func() (string, error) {
		topic := args.CodePattern
		if args.Component != "" {
			topic = fmt.Sprintf("%s %s", args.CodePattern, args.Component)
		}
		if args.Language != "" {
			topic = fmt.Sprintf("%s %s", topic, args.Language)
		}

		var internalInsights []string
		if s.internalAnalyzer != nil && queryBuilder != nil {
			query := queryBuilder.BuildCompositeQuery(ToolNameExplainCodePattern, topic+" code implementation example pattern", "")
			internalInsights = s.internalAnalyzer.Search(llmContext, query, "code_discussions")
		}

		docs, docsStatus := s.enhanceWithCommonDataSourcesInsights(ToolNameExplainCodePattern, topic)

		title := fmt.Sprintf(TemplateCodePatternTitle, args.CodePattern)
		summary := fmt.Sprintf("Code pattern explanation for **%s** in %s component", args.CodePattern, args.Component)
		if args.Language != "" {
			summary = fmt.Sprintf("%s (Language: %s)", summary, args.Language)
		}
		summary += ".\n\nThis report includes code examples and documentation from Mattermost repositories, documentation, and technical resources."

		report := buildReportWithDocsAndInternal(title, summary, internalInsights, docs, docsStatus)
		return report, nil
	})
}

// DebugIssue helps debug Mattermost-specific errors
func (s *Service) DebugIssue(llmContext *llm.Context, args DebugIssueArgs, queryBuilder shared.MetadataProvider) (string, error) {
	return s.cacheHelper.ExecuteWithCache(ToolNameDebugIssue, args, func() (string, error) {
		topic := args.ErrorMessage
		if args.Component != "" {
			topic = fmt.Sprintf("%s %s", args.ErrorMessage, args.Component)
		}
		if args.Context != "" {
			topic = fmt.Sprintf("%s %s", topic, args.Context)
		}

		var internalInsights []string
		if s.internalAnalyzer != nil && queryBuilder != nil {
			query := queryBuilder.BuildCompositeQuery(ToolNameDebugIssue, topic+" error issue problem solution fix", "")
			internalInsights = s.internalAnalyzer.Search(llmContext, query, "error_discussions")
		}

		docs, docsStatus := s.enhanceWithCommonDataSourcesInsights(ToolNameDebugIssue, topic)

		title := fmt.Sprintf(TemplateDebugIssueTitle, args.ErrorMessage)
		summary := fmt.Sprintf("Debug assistance for error: **%s** in %s component", args.ErrorMessage, args.Component)
		if args.Context != "" {
			summary = fmt.Sprintf("%s\n\nContext: %s", summary, args.Context)
		}
		summary += ".\n\nThis report includes troubleshooting guides, similar issues, and solutions from Mattermost resources."

		report := buildReportWithDocsAndInternal(title, summary, internalInsights, docs, docsStatus)
		return report, nil
	})
}

// FindArchitecture locates and explains Mattermost architecture
func (s *Service) FindArchitecture(llmContext *llm.Context, args FindArchitectureArgs, queryBuilder shared.MetadataProvider) (string, error) {
	return s.cacheHelper.ExecuteWithCache(ToolNameFindArchitecture, args, func() (string, error) {
		topic := args.Topic
		if args.Scope != "" {
			topic = fmt.Sprintf("%s %s", args.Topic, args.Scope)
		}

		var internalInsights []string
		if s.internalAnalyzer != nil && queryBuilder != nil {
			query := queryBuilder.BuildCompositeQuery(ToolNameFindArchitecture, topic+" architecture design decision pattern", "")
			internalInsights = s.internalAnalyzer.Search(llmContext, query, "architecture_discussions")
		}

		docs, docsStatus := s.enhanceWithCommonDataSourcesInsights(ToolNameFindArchitecture, topic)

		title := fmt.Sprintf(TemplateArchitectureTitle, args.Topic)
		summary := fmt.Sprintf("Architecture overview for: **%s**", args.Topic)
		if args.Scope != "" {
			summary = fmt.Sprintf("%s (Scope: %s)", summary, args.Scope)
		}
		summary += ".\n\nThis report includes architecture documentation, ADRs, and design decisions from Mattermost resources."

		report := buildReportWithDocsAndInternal(title, summary, internalInsights, docs, docsStatus)
		return report, nil
	})
}

// GetAPIExamples finds Mattermost API usage examples
func (s *Service) GetAPIExamples(llmContext *llm.Context, args GetAPIExamplesArgs, queryBuilder shared.MetadataProvider) (string, error) {
	return s.cacheHelper.ExecuteWithCache(ToolNameGetAPIExamples, args, func() (string, error) {
		topic := args.APIName
		if args.UseCase != "" {
			topic = fmt.Sprintf("%s %s", args.APIName, args.UseCase)
		}

		var internalInsights []string
		if s.internalAnalyzer != nil && queryBuilder != nil {
			query := queryBuilder.BuildCompositeQuery(ToolNameGetAPIExamples, topic+" api usage example method", "")
			internalInsights = s.internalAnalyzer.Search(llmContext, query, "api_discussions")
		}

		docs, docsStatus := s.enhanceWithCommonDataSourcesInsights(ToolNameGetAPIExamples, topic)

		title := fmt.Sprintf(TemplateAPIExamplesTitle, args.APIName)
		summary := fmt.Sprintf("API usage examples for: **%s**", args.APIName)
		if args.UseCase != "" {
			summary = fmt.Sprintf("%s (Use case: %s)", summary, args.UseCase)
		}
		summary += ".\n\nThis report includes code examples, API documentation, and usage patterns from Mattermost plugins and repositories."

		report := buildReportWithDocsAndInternal(title, summary, internalInsights, docs, docsStatus)
		return report, nil
	})
}

// SummarizePRs summarizes recent Mattermost pull requests
func (s *Service) SummarizePRs(llmContext *llm.Context, args SummarizePRsArgs, queryBuilder shared.MetadataProvider) (string, error) {
	if args.TimeRange == "" {
		args.TimeRange = shared.DefaultTimeRange
	}
	if args.Repository == "" {
		args.Repository = RepoMattermostServer
	}

	return s.cacheHelper.ExecuteWithCache(ToolNameSummarizePRs, args, func() (string, error) {
		topic := args.Repository
		if args.Category != "" {
			topic = fmt.Sprintf("%s %s", args.Repository, args.Category)
		}
		topic = fmt.Sprintf("%s %s", topic, args.TimeRange)

		var internalInsights []string
		if s.internalAnalyzer != nil && queryBuilder != nil {
			query := queryBuilder.BuildCompositeQuery(ToolNameSummarizePRs, topic+" pull request pr change release", "")
			internalInsights = s.internalAnalyzer.Search(llmContext, query, "pr_discussions")
		}

		docs, docsStatus := s.enhanceWithCommonDataSourcesInsights(ToolNameSummarizePRs, topic)

		title := fmt.Sprintf(TemplatePRSummaryTitle, args.Repository)
		summary := fmt.Sprintf("Recent pull requests for **%s** (%s)", args.Repository, args.TimeRange)
		if args.Category != "" {
			summary = fmt.Sprintf("%s - Category: %s", summary, args.Category)
		}
		summary += ".\n\nThis report includes recent PRs, release notes, and code changes from Mattermost repositories and documentation."

		report := buildReportWithDocsAndInternal(title, summary, internalInsights, docs, docsStatus)
		return report, nil
	})
}

// enhanceWithCommonDataSourcesInsights searches external documentation and returns results with status
func (s *Service) enhanceWithCommonDataSourcesInsights(toolName, topic string) (map[string][]datasources.Doc, string) {
	// Skip search if topic is empty
	topic = strings.TrimSpace(topic)
	if topic == "" {
		if s.pluginAPI != nil {
			s.pluginAPI.LogDebug("common data sources skipped - empty topic", "tool", toolName)
		}
		return nil, shared.StatusNoResults
	}

	docs, err := shared.SearchMultipleSources(
		context.Background(),
		toolName,
		topic,
		shared.MaxDocsPerSource,
		s.metadataProvider,
		s.commonDataSourcesClient,
		s.pluginAPI,
		"dev",
	)
	if err != nil {
		if s.pluginAPI != nil {
			s.pluginAPI.LogError("common data sources search failed", "tool", toolName, "topic", topic, "error", err.Error())
		}
		s.logCommonDataSourcesOperation(toolName, topic, shared.StatusFailed, 0, 0)
		return nil, shared.StatusFailed
	}

	totalDocs := 0
	for _, docList := range docs {
		totalDocs += len(docList)
	}

	if totalDocs == 0 {
		return nil, shared.StatusNoResults
	}

	s.logCommonDataSourcesOperation(toolName, topic, shared.StatusCompleted, len(docs), totalDocs)
	return docs, shared.StatusCompleted
}

func (s *Service) logCommonDataSourcesOperation(tool, topic, status string, sources, docs int) {
	if s.pluginAPI != nil {
		s.pluginAPI.LogDebug(
			"common data sources operation",
			"tool", tool,
			"status", status,
			"sources", sources,
			"docs", docs,
		)
	}
}

// buildReportWithDocsAndInternal builds a report combining internal discussions and external documentation
func buildReportWithDocsAndInternal(title, summary string, internalInsights []string, docs map[string][]datasources.Doc, docsStatus string) string {
	var report strings.Builder

	report.WriteString(title)
	report.WriteString("\n\n")
	report.WriteString(summary)
	report.WriteString("\n\n")

	if len(internalInsights) > 0 {
		report.WriteString(shared.FormatInternalInsights(internalInsights))
	}

	if docsStatus == shared.StatusCompleted && len(docs) > 0 {
		report.WriteString(HeaderCodeExamples)
		for sourceName, sourceDocs := range docs {
			report.WriteString(fmt.Sprintf("### %s\n\n", sourceName))
			for i, doc := range sourceDocs {
				if i >= shared.MaxDocsPerSource {
					break
				}
				report.WriteString(fmt.Sprintf("**%s**\n", doc.Title))
				if doc.URL != "" {
					report.WriteString(fmt.Sprintf("URL: %s\n", doc.URL))
				}

				// Add excerpt with length limit
				excerpt := doc.Content
				if len(excerpt) > shared.ExcerptMaxLength {
					excerpt = excerpt[:shared.ExcerptMaxLength] + "..."
				}
				report.WriteString(fmt.Sprintf("\n%s\n\n", excerpt))
			}
		}
	} else if docsStatus != shared.StatusCompleted {
		report.WriteString(fmt.Sprintf("External documentation search: %s\n\n", docsStatus))
	}

	return report.String()
}
