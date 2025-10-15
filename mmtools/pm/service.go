// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package pm

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/mattermost/mattermost-plugin-ai/datasources"
	"github.com/mattermost/mattermost-plugin-ai/llm"
	"github.com/mattermost/mattermost-plugin-ai/mmapi"
	"github.com/mattermost/mattermost-plugin-ai/mmtools/shared"
	"github.com/mattermost/mattermost-plugin-ai/search"
	"github.com/mattermost/mattermost-plugin-ai/semanticcache"
)

// cleanDirectivePhrases removes directive phrases from context strings
func cleanDirectivePhrases(context string) string {
	if context == "" {
		return context
	}

	// Patterns to remove (case-insensitive)
	directivePatterns := []string{
		`(?i)focus\s+on\s+`,
		`(?i)analyze\s+alignment\s+with\s+`,
		`(?i)analyze\s+`,
		`(?i)consider\s+`,
		`(?i)evaluate\s+`,
		`(?i)assess\s+`,
		`(?i)compare\s+with\s+`,
		`(?i)compare\s+`,
		`(?i)ensure\s+`,
		`(?i)check\s+for\s+`,
	}

	cleaned := context
	for _, pattern := range directivePatterns {
		re := regexp.MustCompile(pattern)
		cleaned = re.ReplaceAllString(cleaned, "")
	}

	// Clean up extra spaces
	cleaned = strings.Join(strings.Fields(cleaned), " ")
	return strings.TrimSpace(cleaned)
}

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

func (s *Service) CompileMarketResearch(llmContext *llm.Context, args CompileMarketResearchArgs, queryBuilder shared.MetadataProvider) (string, error) {
	if args.TimeRange == "" {
		args.TimeRange = shared.DefaultTimeRange
	}

	cacheKeyArgs := struct {
		PrimaryFeatures []string
		ResearchIntent  string
		Context         string
		TimeRange       string
	}{
		PrimaryFeatures: args.PrimaryFeatures,
		ResearchIntent:  args.ResearchIntent,
		Context:         args.Context,
		TimeRange:       args.TimeRange,
	}

	return s.cacheHelper.ExecuteWithCache(ToolNameCompileMarketResearch, cacheKeyArgs, func() (string, error) {
		conversationResults := s.analyzeConversationsForMarketResearch(llmContext, args, queryBuilder)

		topic := strings.Join(args.PrimaryFeatures, " ")
		if args.Context != "" {
			topic = strings.TrimSpace(topic + " " + args.Context)
		}
		commonDataSourcesInsights, commonDataSourcesStatus := s.enhanceWithCommonDataSourcesInsights(ToolNameCompileMarketResearch, topic)

		report := s.generateMarketResearchReport(llmContext, args, conversationResults, commonDataSourcesInsights, commonDataSourcesStatus)
		return report, nil
	})
}

func (s *Service) AnalyzeFeatureGaps(llmContext *llm.Context, args AnalyzeFeatureGapsArgs, queryBuilder shared.MetadataProvider) (string, error) {
	if args.TimeRange == "" {
		args.TimeRange = shared.DefaultTimeRange
	}

	cacheKeyArgs := struct {
		PrimaryFeatures []string
		GapAnalysisType string
		Context         string
		TimeRange       string
	}{
		PrimaryFeatures: args.PrimaryFeatures,
		GapAnalysisType: args.GapAnalysisType,
		Context:         args.Context,
		TimeRange:       args.TimeRange,
	}

	return s.cacheHelper.ExecuteWithCache(ToolNameAnalyzeFeatureGaps, cacheKeyArgs, func() (string, error) {
		feedbackResults := s.analyzeCustomerFeedbackForFeature(llmContext, args, queryBuilder)

		topic := strings.Join(args.PrimaryFeatures, " ")
		if args.Context != "" {
			topic = strings.TrimSpace(topic + " " + args.Context)
		}
		commonDataSourcesInsights, commonDataSourcesStatus := s.enhanceWithCommonDataSourcesInsights(ToolNameAnalyzeFeatureGaps, topic)

		report := s.generateFeatureGapReport(llmContext, args, feedbackResults, commonDataSourcesInsights, commonDataSourcesStatus)
		return report, nil
	})
}

func (s *Service) AnalyzeStrategicAlignment(llmContext *llm.Context, args AnalyzeStrategicAlignmentArgs) (string, error) {
	cacheKeyArgs := struct {
		PrimaryFeatures []string
		AnalysisIntent  string
		Context         string
		Framework       string
	}{
		PrimaryFeatures: args.PrimaryFeatures,
		AnalysisIntent:  args.AnalysisIntent,
		Context:         args.Context,
		Framework:       args.Framework,
	}

	return s.cacheHelper.ExecuteWithCache(ToolNameAnalyzeStrategicAlignment, cacheKeyArgs, func() (string, error) {
		alignmentResults := s.analyzeStrategicAlignmentData(llmContext, args)

		topic := strings.Join(args.PrimaryFeatures, " ")
		if args.Context != "" {
			topic = strings.TrimSpace(topic + " " + args.Context)
		}
		commonDataSourcesInsights, commonDataSourcesStatus := s.enhanceWithCommonDataSourcesInsights(ToolNameAnalyzeStrategicAlignment, topic)

		report := s.generateStrategicAlignmentReport(llmContext, args, alignmentResults, commonDataSourcesInsights, commonDataSourcesStatus)
		return report, nil
	})
}

func (s *Service) analyzeConversationsForMarketResearch(llmContext *llm.Context, args CompileMarketResearchArgs, queryBuilder shared.MetadataProvider) MarketResearchResults {
	cleanContext := cleanDirectivePhrases(args.Context)
	topic := strings.Join(args.PrimaryFeatures, " ") + " " + cleanContext

	var competitorMentions, marketTrends, teamInsights []string
	if s.internalAnalyzer != nil && queryBuilder != nil {
		queryTopic := strings.Join(args.PrimaryFeatures, " ") + " " + args.Context

		query := queryBuilder.BuildCompositeQuery(ToolNameCompileMarketResearch, queryTopic+" competitor alternative vs", "")
		competitorMentions = s.internalAnalyzer.Search(llmContext, query, "competitor_mentions")

		query = queryBuilder.BuildCompositeQuery(ToolNameCompileMarketResearch, queryTopic+" trend market opportunity growth", "")
		marketTrends = s.internalAnalyzer.Search(llmContext, query, "market_trends")

		query = queryBuilder.BuildCompositeQuery(ToolNameCompileMarketResearch, queryTopic+" team discussion insight", "")
		teamInsights = s.internalAnalyzer.Search(llmContext, query, "team_insights")
	}

	return MarketResearchResults{
		Topic:              topic,
		TimeRange:          args.TimeRange,
		CompetitorMentions: competitorMentions,
		MarketTrends:       marketTrends,
		TeamInsights:       teamInsights,
	}
}

func (s *Service) analyzeCustomerFeedbackForFeature(llmContext *llm.Context, args AnalyzeFeatureGapsArgs, queryBuilder shared.MetadataProvider) CustomerFeedbackResults {
	featureName := strings.Join(args.PrimaryFeatures, " ")

	var customerRequests, painPoints, competitiveGaps []string
	if s.internalAnalyzer != nil && queryBuilder != nil {
		queryTopic := strings.Join(args.PrimaryFeatures, " ") + " " + args.Context

		query := queryBuilder.BuildCompositeQuery(ToolNameAnalyzeFeatureGaps, queryTopic+" customer request feature request feedback", "")
		customerRequests = s.internalAnalyzer.Search(llmContext, query, "customer_requests")

		query = queryBuilder.BuildCompositeQuery(ToolNameAnalyzeFeatureGaps, queryTopic+" issue problem limitation bug pain", "")
		painPoints = s.internalAnalyzer.Search(llmContext, query, "pain_points")

		query = queryBuilder.BuildCompositeQuery(ToolNameAnalyzeFeatureGaps, queryTopic+" competitive competitor alternative vs gap", "")
		competitiveGaps = s.internalAnalyzer.Search(llmContext, query, "competitive_gaps")
	}

	return CustomerFeedbackResults{
		FeatureName:      featureName,
		TimeRange:        args.TimeRange,
		CustomerRequests: customerRequests,
		PainPoints:       painPoints,
		CompetitiveGaps:  competitiveGaps,
	}
}

func (s *Service) analyzeStrategicAlignmentData(llmContext *llm.Context, args AnalyzeStrategicAlignmentArgs) StrategicAlignmentResults {
	cleanContext := cleanDirectivePhrases(args.Context)
	topic := strings.Join(args.PrimaryFeatures, " ") + " " + cleanContext

	var visionAlignment, frameworkApplication, stakeholderPerspectives []string
	if s.internalAnalyzer != nil {
		queryTopic := strings.Join(args.PrimaryFeatures, " ") + " " + args.Context

		query := fmt.Sprintf("%s vision mission strategy %s", queryTopic, args.Framework)
		visionAlignment = s.internalAnalyzer.SearchWithLogging(llmContext, query, "vision_alignment", "vision alignment search failed")

		query = fmt.Sprintf("%s %s framework analysis methodology", queryTopic, args.AnalysisIntent)
		frameworkApplication = s.internalAnalyzer.SearchWithLogging(llmContext, query, "framework_application", "framework application search failed")

		query = fmt.Sprintf("%s stakeholder customer engineering sales product", queryTopic)
		stakeholderPerspectives = s.internalAnalyzer.SearchWithLogging(llmContext, query, "stakeholder_perspectives", "stakeholder perspectives search failed")
	}

	return StrategicAlignmentResults{
		Topic:                   topic,
		VisionAlignment:         visionAlignment,
		FrameworkApplication:    frameworkApplication,
		StakeholderPerspectives: stakeholderPerspectives,
	}
}

func (s *Service) generateMarketResearchReport(llmContext *llm.Context, args CompileMarketResearchArgs, conversationResults MarketResearchResults, commonDataSourcesInsights map[string][]datasources.Doc, commonDataSourcesStatus string) string {
	reportBuilder := NewReportBuilder()

	featureString := strings.Join(args.PrimaryFeatures, ", ")
	reportBuilder.AddTitle(TemplateMarketResearchTitle, fmt.Sprintf("%s %s", featureString, args.Context))
	reportBuilder.AddHeader(HeaderExecutiveSummary)

	summary := fmt.Sprintf(TemplateMarketResearchSummary, featureString)
	if commonDataSourcesStatus == shared.StatusCompleted {
		summary += TemplateCommonDataSourcesSummary
	}
	reportBuilder.AddText(summary + ".\n\n")

	reportBuilder.AddHeader(HeaderTeamInsights)
	reportBuilder.AddText(TemplateConversationAnalysis)
	reportBuilder.AddBulletList(conversationResults.CompetitorMentions)
	reportBuilder.AddBulletList(conversationResults.MarketTrends)
	reportBuilder.AddBulletList(conversationResults.TeamInsights)

	if commonDataSourcesStatus == shared.StatusCompleted {
		reportBuilder.AddHeader(HeaderExternalDocumentation)
		for sourceName, docs := range commonDataSourcesInsights {
			reportBuilder.AddDocumentList(sourceName, docs, shared.ExcerptMaxLength, EllipsisText)
		}
	} else {
		reportBuilder.AddText(fmt.Sprintf("%s\n\n", commonDataSourcesStatus))
	}

	reportBuilder.AddHeader(HeaderSummaryRecommendations)
	reportBuilder.AddText(fmt.Sprintf(TemplateCurrentState, featureString))
	reportBuilder.AddText(TemplateKeyFindings)
	reportBuilder.AddText(fmt.Sprintf(RecommendationMarketOpportunity, featureString))
	reportBuilder.AddText(RecommendationCustomerFeedback)
	if commonDataSourcesStatus == shared.StatusCompleted {
		reportBuilder.AddText(RecommendationCommonDataSources)
	}

	return reportBuilder.String()
}

func (s *Service) generateFeatureGapReport(llmContext *llm.Context, args AnalyzeFeatureGapsArgs, feedbackResults CustomerFeedbackResults, commonDataSourcesInsights map[string][]datasources.Doc, commonDataSourcesStatus string) string {
	reportBuilder := NewReportBuilder()

	featureString := strings.Join(args.PrimaryFeatures, ", ")
	reportBuilder.AddTitle(TemplateFeatureGapTitle, fmt.Sprintf("%s %s", featureString, args.Context))
	reportBuilder.AddHeader(HeaderExecutiveSummary)

	summary := fmt.Sprintf(TemplateFeatureGapSummary, featureString)
	if commonDataSourcesStatus == shared.StatusCompleted {
		summary += TemplateCommonDataSourcesSummary
	}
	reportBuilder.AddText(summary + ".\n\n")

	reportBuilder.AddHeader(HeaderCustomerFeedback)
	reportBuilder.AddText(TemplateCustomerFeedback)
	reportBuilder.AddBulletList(feedbackResults.CustomerRequests)
	reportBuilder.AddBulletList(feedbackResults.PainPoints)
	reportBuilder.AddBulletList(feedbackResults.CompetitiveGaps)

	if commonDataSourcesStatus == shared.StatusCompleted {
		reportBuilder.AddHeader(HeaderExternalDocumentation)
		for sourceName, docs := range commonDataSourcesInsights {
			reportBuilder.AddDocumentList(sourceName, docs, shared.ExcerptMaxLength, EllipsisText)
		}
	} else {
		reportBuilder.AddText(fmt.Sprintf("%s\n\n", commonDataSourcesStatus))
	}

	reportBuilder.AddHeader(HeaderSummaryRecommendations)
	reportBuilder.AddText(fmt.Sprintf(TemplateCurrentState, featureString))
	reportBuilder.AddText(TemplateKeyFindings)
	reportBuilder.AddText(fmt.Sprintf(RecommendationFeatureGaps, featureString))
	reportBuilder.AddText(RecommendationCustomerFeedback)
	if commonDataSourcesStatus == shared.StatusCompleted {
		reportBuilder.AddText(RecommendationCommonDataSources)
	}

	return reportBuilder.String()
}

func (s *Service) generateStrategicAlignmentReport(llmContext *llm.Context, args AnalyzeStrategicAlignmentArgs, alignmentResults StrategicAlignmentResults, commonDataSourcesInsights map[string][]datasources.Doc, commonDataSourcesStatus string) string {
	reportBuilder := NewReportBuilder()

	featureString := strings.Join(args.PrimaryFeatures, ", ")
	reportBuilder.AddTitle(TemplateStrategicAlignmentTitle, fmt.Sprintf("%s %s", featureString, args.Context))
	reportBuilder.AddHeader(HeaderExecutiveSummary)

	summary := fmt.Sprintf(TemplateStrategicAlignmentSummary, featureString)
	if commonDataSourcesStatus == shared.StatusCompleted {
		summary += TemplateCommonDataSourcesSummary
	}
	reportBuilder.AddText(summary + ".\n\n")

	reportBuilder.AddHeader(HeaderStrategicGoals)
	reportBuilder.AddText(TemplateGoalsAlignment)
	reportBuilder.AddBulletList(alignmentResults.VisionAlignment)
	reportBuilder.AddBulletList(alignmentResults.FrameworkApplication)
	reportBuilder.AddBulletList(alignmentResults.StakeholderPerspectives)

	if commonDataSourcesStatus == shared.StatusCompleted {
		reportBuilder.AddHeader(HeaderExternalDocumentation)
		for sourceName, docs := range commonDataSourcesInsights {
			reportBuilder.AddDocumentList(sourceName, docs, shared.ExcerptMaxLength, EllipsisText)
		}
	} else {
		reportBuilder.AddText(fmt.Sprintf("%s\n\n", commonDataSourcesStatus))
	}

	reportBuilder.AddHeader(HeaderSummaryRecommendations)
	reportBuilder.AddText(fmt.Sprintf(TemplateCurrentState, featureString))
	reportBuilder.AddText(TemplateKeyFindings)
	reportBuilder.AddText(fmt.Sprintf(RecommendationStrategicAlignment, featureString))
	if commonDataSourcesStatus == shared.StatusCompleted {
		reportBuilder.AddText(RecommendationCommonDataSources)
	}

	return reportBuilder.String()
}

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
		"pm",
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

func (s *Service) GetChannelName(channelID string) string {
	if s.pluginAPI == nil {
		return "Unknown Channel"
	}
	channel, err := s.pluginAPI.GetChannel(channelID)
	if err != nil {
		return "Unknown Channel"
	}
	return channel.DisplayName
}
