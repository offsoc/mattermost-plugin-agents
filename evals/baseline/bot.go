// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package baseline

import (
	"context"
	"time"
)

const (
	// Metadata keys for bot responses
	MetadataKeyModelType = "model_type"
	MetadataKeyBotID     = "bot_id"

	// Model type values
	ModelTypeBaseline = "baseline"
	ModelTypeEnhanced = "enhanced"
)

// Bot defines the interface for baseline evaluation comparisons.
// This allows us to compare any enhanced bot against a raw LLM baseline.
type Bot interface {
	Respond(ctx context.Context, msg string) (Answer, error)
	Name() string
}

// Answer represents the response from a bot including metadata for evaluation.
type Answer struct {
	Text     string
	Latency  time.Duration
	Tokens   TokenUsage
	Metadata map[string]interface{}
}

// TokenUsage tracks token consumption for cost analysis.
type TokenUsage struct {
	Prompt     int
	Completion int
	Total      int
}

// TestResults holds the results of running multiple trials for a scenario.
type TestResults struct {
	BotName                   string
	Scenario                  string
	Trials                    int
	Passes                    int
	Failures                  int
	PassRate                  float64
	GroundingPasses           int
	GroundingFails            int
	GroundingPassRate         float64
	GroundingValidCitations   int     // Total valid citations across all trials
	GroundingInvalidCitations int     // Total invalid citations across all trials
	GroundingValidRate        float64 // Average valid citation rate
	GroundingFabricationRate  float64 // Average fabrication rate
	// Semantic grounding metrics (content fabrication detection)
	SemanticGroundingPasses     int     // Trials that passed semantic grounding
	SemanticGroundingFails      int     // Trials that failed semantic grounding
	SemanticGroundingPassRate   float64 // Percentage of trials passing semantic grounding
	SemanticGroundingScore      float64 // Average semantic grounding score (0-1)
	SemanticGroundedSentences   int     // Total grounded sentences across trials
	SemanticUngroundedSentences int     // Total ungrounded sentences (fabricated)
	AvgLatency                  time.Duration
	AvgTokens                   TokenUsage
	Errors                      []string
}

// ComparisonResult holds the statistical comparison between two bots.
type ComparisonResult struct {
	BaselineBot   TestResults
	EnhancedBot   TestResults
	Improvement   float64    // Enhanced pass rate - Baseline pass rate
	Significance  float64    // P-value from statistical test
	ConfidenceInt [2]float64 // 95% confidence interval for improvement
}
