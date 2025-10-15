// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package pm

// CompileMarketResearchArgs defines the arguments for the CompileMarketResearch tool
type CompileMarketResearchArgs struct {
	PrimaryFeatures []string `json:"primary_features" jsonschema_description:"The main Mattermost features to research. Select from: ai, channels, mobile, playbooks, boards, calls, plugins, enterprise, security, deployment, performance, api, desktop, web, notifications"`
	ResearchIntent  string   `json:"research_intent" jsonschema_description:"The type of market research requested. Select from: competitive_analysis, market_trends, customer_feedback, adoption_metrics, feature_gaps"`
	Context         string   `json:"context,omitempty" jsonschema_description:"Additional search keywords or topics to include (e.g., 'vision', 'mission critical', 'compliance'). DO NOT include directive phrases like 'Focus on' or 'Analyze'. Just keywords only (optional)"`
	TimeRange       string   `json:"time_range" jsonschema_description:"Time range for analysis: 'week', 'month', 'quarter' (default: 'month')"`
}

// AnalyzeFeatureGapsArgs defines the arguments for the AnalyzeFeatureGaps tool
type AnalyzeFeatureGapsArgs struct {
	PrimaryFeatures []string `json:"primary_features" jsonschema_description:"The main Mattermost features to analyze for gaps. Select from: ai, channels, mobile, playbooks, boards, calls, plugins, enterprise, security, deployment, performance, api, desktop, web, notifications"`
	GapAnalysisType string   `json:"gap_analysis_type" jsonschema_description:"The type of gap analysis to perform. Select from: competitive_gaps, user_request_gaps, technical_debt_gaps, compliance_gaps, usability_gaps"`
	Context         string   `json:"context,omitempty" jsonschema_description:"Additional search keywords or topics to include (e.g., 'enterprise', 'compliance', 'usability'). DO NOT include directive phrases like 'Focus on' or 'Analyze'. Just keywords only (optional)"`
	TimeRange       string   `json:"time_range" jsonschema_description:"Time range for analysis: 'week', 'month', 'quarter' (default: 'month')"`
}

// AnalyzeStrategicAlignmentArgs defines the arguments for the AnalyzeStrategicAlignment tool
type AnalyzeStrategicAlignmentArgs struct {
	PrimaryFeatures []string `json:"primary_features" jsonschema_description:"The main Mattermost features mentioned in the query. Select from: ai, channels, mobile, playbooks, boards, calls, plugins, enterprise, security, deployment, performance, api, desktop, web, notifications"`
	AnalysisIntent  string   `json:"analysis_intent" jsonschema_description:"The type of strategic analysis requested. Select from: strategic_alignment, vision_alignment, okr_alignment, roadmap_prioritization, competitive_analysis"`
	Context         string   `json:"context,omitempty" jsonschema_description:"Additional search keywords or topics to include (e.g., 'mission critical', 'vision', 'strategy'). DO NOT include directive phrases like 'Focus on' or 'Analyze'. Just keywords only (optional)"`
	Framework       string   `json:"framework" jsonschema_description:"PM framework to apply: 'vision-alignment', 'rice', 'stakeholder-balance', 'prioritization' (optional, defaults to comprehensive analysis)"`
}

// MarketResearchResults represents the results of conversation-based market research
type MarketResearchResults struct {
	Topic              string
	TimeRange          string
	CompetitorMentions []string
	MarketTrends       []string
	TeamInsights       []string
}

// CustomerFeedbackResults represents the results of customer feedback analysis
type CustomerFeedbackResults struct {
	FeatureName      string
	TimeRange        string
	CustomerRequests []string
	PainPoints       []string
	CompetitiveGaps  []string
}

// StrategicAlignmentResults represents the results of strategic alignment analysis
type StrategicAlignmentResults struct {
	Topic                   string
	VisionAlignment         []string
	FrameworkApplication    []string
	StakeholderPerspectives []string
}
