// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package pm

// Tool names
const (
	ToolNameCompileMarketResearch     = "CompileMarketResearch"
	ToolNameAnalyzeFeatureGaps        = "AnalyzeFeatureGaps"
	ToolNameAnalyzeStrategicAlignment = "AnalyzeStrategicAlignment"
	ToolNameGetCustomerFeedback       = "GetCustomerFeedback"
	ToolNameAnalyzeProductMetrics     = "AnalyzeProductMetrics"
)

// Search term patterns for customer feedback (keywords for boolean queries)
const (
	SearchPatternCustomerRequest = "customer request"
	SearchPatternFeatureRequest  = "feature request"
	SearchPatternUserFeedback    = "user feedback"
	SearchPatternEnhancement     = "enhancement"
	SearchPatternIssue           = "issue"
	SearchPatternProblem         = "problem"
	SearchPatternLimitation      = "limitation"
	SearchPatternBug             = "bug"
	SearchPatternCompetitor      = "competitor"
	SearchPatternCompetitive     = "competitive"
	SearchPatternAlternative     = "alternative"
	SearchPatternVs              = "vs"
	SearchPatternVision          = "vision"
	SearchPatternStrategy        = "strategy"
	SearchPatternAlignment       = "alignment"
	SearchPatternFramework       = "framework"
	SearchPatternPrioritization  = "prioritization"
	SearchPatternStakeholder     = "stakeholder"
	SearchPatternTrend           = "trend"
	SearchPatternMarket          = "market"
	SearchPatternOpportunity     = "opportunity"
	SearchPatternGrowth          = "growth"
	SearchPatternAdoption        = "adoption"
	SearchPatternUsage           = "usage"
)

// Log levels
const (
	LogLevelInfo = "info"
	LogLevelWarn = "warn"
)

// Log message formats
const (
	LogFormatCommonDataSourcesOperation   = "[%s] Common data sources operation: tool=%s topic=%s status=%s sources=%d docs=%d\n"
	LogFormatCommonDataSourcesError       = "[warn] Common data sources error: %v\n"
	LogFormatCommonDataSourcesPerformance = "[info] Common data sources performance: tool=%s topic=%s sources=%v docs=%d duration_ms=%d\n"
)

// Status constants moved to mmtools/constants.go (shared across all services)
// Import them as: mmtools.StatusCompleted, mmtools.StatusSkipped, etc.

// Report section headers
const (
	HeaderExecutiveSummary         = "## Executive Summary\n\n"
	HeaderTeamInsights             = "## Team Insights (Conversation Analysis)\n\n"
	HeaderCompetitiveLandscape     = "**Competitive Landscape:**\n"
	HeaderMarketTrends             = "**Market Trends:**\n"
	HeaderInternalDiscussions      = "**Internal Discussions:**\n"
	HeaderExternalDocumentation    = "## External Documentation Insights\n\n"
	HeaderSummaryRecommendations   = "## Summary & Recommendations\n\n"
	HeaderCustomerFeedbackAnalysis = "## Customer Feedback Analysis\n\n"
	HeaderCustomerFeedback         = "## Customer Feedback\n\n"
	HeaderCustomerRequests         = "**Customer Requests:**\n"
	HeaderIdentifiedPainPoints     = "**Identified Pain Points:**\n"
	HeaderCompetitiveGaps          = "**Competitive Gaps:**\n"
	HeaderGapAnalysisSummary       = "## Gap Analysis Summary\n\n"
	HeaderStrategicRecommendations = "## Strategic Recommendations\n\n"
	HeaderStrategicGoals           = "## Strategic Goals\n\n"
)

// Report content templates
const (
	TemplateMarketResearchTitle       = "# Market Research: %s\n\n"
	TemplateFeatureGapAnalysisTitle   = "# Feature Gap Analysis: %s\n\n"
	TemplateFeatureGapTitle           = "# Feature Gap Analysis: %s\n\n"
	TemplateStrategicAlignmentTitle   = "# Strategic Alignment Analysis: %s\n\n"
	TemplateMarketResearchSummary     = "Comprehensive feature gap analysis for %s based on customer feedback"
	TemplateFeatureGapSummary         = "Detailed gap analysis for %s based on customer feedback and competitive positioning"
	TemplateStrategicAlignmentSummary = "Strategic alignment analysis for %s considering organizational goals and priorities"
	TemplateCommonDataSourcesSummary  = " and common data sources documentation validation"
	TemplateConversationAnalysis      = "Based on team conversation analysis:\n\n"
	TemplateCustomerFeedback          = "Based on customer feedback analysis:\n\n"
	TemplateGoalsAlignment            = "Analyzing strategic goals and feature alignment:\n\n"
	TemplateConversationAndExternal   = "Based on team conversations and common data sources documentation analysis:\n\n"
	TemplateCurrentState              = "**Current State:** %s functionality shows gaps in customer satisfaction and competitive positioning.\n\n"
	TemplateKeyFindings               = "**Key Findings:**\n"
)

// Strategic recommendations
const (
	RecommendationMarketOpportunity        = "- Market research for %s indicates strategic opportunities\n"
	RecommendationCustomerFeedback         = "- Customer feedback and competitive analysis provide actionable insights\n"
	RecommendationCommonDataSources        = "- Common data sources support market positioning decisions\n"
	RecommendationFeatureGaps              = "- Feature gap analysis for %s identifies key improvement areas\n"
	RecommendationStrategicAlignment       = "- Strategic alignment analysis for %s ensures organizational goal consistency\n"
	RecommendationCustomerLimitations      = "- Customer feedback indicates significant feature limitations\n"
	RecommendationCompetitiveGaps          = "- Competitive analysis reveals functionality gaps\n"
	RecommendationTeamImpact               = "- Support and sales teams report customer impact\n"
	RecommendationCommonDataSourcePatterns = "- Common data sources support customer feedback patterns\n"
	RecommendationPriorityEnhancement      = "1. **Priority Enhancement:** Address %s limitations based on customer feedback patterns\n"
	RecommendationCompetitivePositioning   = "2. **Competitive Positioning:** Implement features to close competitive gaps\n"
	RecommendationCustomerCommunication    = "3. **Customer Communication:** Provide roadmap visibility for requested capabilities\n"
	RecommendationPerformanceOptimization  = "4. **Performance Optimization:** Address scalability and performance concerns\n"
	RecommendationDocumentationAlignment   = "5. **Documentation Alignment:** Ensure feature development aligns with external best practices\n"
)

// Content length limits
const (
	FeatureGapExcerptLength = 1500 // Increased from 150 for better context
	DebugReportMaxLength    = 5000 // Increased from 500 for comprehensive debugging
	EllipsisText            = "..."
	MaxDocsPerSourceFetch   = 5 // Maximum documents to fetch from a single external source per query
)
