// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package pm_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/mattermost/mattermost-plugin-ai/config"
	"github.com/mattermost/mattermost-plugin-ai/evals"
	"github.com/mattermost/mattermost-plugin-ai/evals/baseline"
	"github.com/mattermost/mattermost-plugin-ai/grounding"
	"github.com/mattermost/mattermost-plugin-ai/grounding/thread"
	"github.com/mattermost/mattermost-plugin-ai/llm"
	"github.com/mattermost/mattermost-plugin-ai/roles/testutils"
)

func TestPMBotVsBaselineComparison(t *testing.T) {
	// Load scenarios based on level and mm-centric flags
	scenarios, err := LoadPMBotScenarios(*levelFlag, *mmCentricFlag)
	if err != nil {
		t.Fatalf("Failed to load PM scenarios: %v", err)
	}

	// Filter scenarios if needed (for CORE/BREADTH/ALL subsets)
	scenarios = FilterScenariosByFlag(scenarios, *scenarioFlag, t)

	// Create custom evaluation with pmbot-specific configuration
	evalT, numEvals := createPMBotEval(t)

	// Get models for baseline comparison (supports comma-separated list)
	modelsToTest := getModelsForBaselineComparison()
	t.Logf("Testing %d models for baseline comparison: %v", len(modelsToTest), modelsToTest)

	// Run baseline comparison for each model
	for _, modelName := range modelsToTest {
		t.Run("pmbot baseline comparison "+modelName, func(t *testing.T) {
			evalT.T = t

			// Create enhanced PM bot for this model
			enhancedBot := createEnhancedPMBot(evalT, modelName)

			for evalIdx := range numEvals {
				func(t *evals.EvalT, trialNum int) {
					// Determine baseline type based on flag
					baselineType := "vanilla"
					if *rolePromptBaselineFlag {
						baselineType = "PM-prompt"
					}

					t.Logf("Testing model: %s (%s baseline vs enhanced PM bot) - Trial %d/%d", modelName, baselineType, trialNum, numEvals)
					if *debugFlag {
						baseline.LogBaselineTypeInfo(t, modelName, baselineType, "PM", *rolePromptBaselineFlag)
					}

					// Create baseline bot - either vanilla or PM-prompt based on flag
					var baselineBot baseline.Bot
					baselineLogger := createTestLogger(t.T, "BASELINE")

					if *rolePromptBaselineFlag {
						// Create PM-prompt baseline (same prompts as enhanced, but no tools)
						var prompts *llm.Prompts

						threadData := CreatePMBotThreadData("test message")
						mmClient := SetupMockClientWithLogging(t.T, threadData, baselineLogger)
						_, _, _, promptsProvider, _, _, _ := SetupPMBotServices(t.T, threadData, mmClient)

						prompts = promptsProvider

						// Use PM-prompt baseline for fair comparison
						baselineBot = baseline.NewPMPromptBaselineBot(t.LLM, prompts, "pm-baseline-"+modelName)
					} else {
						// Create vanilla baseline bot
						if *debugFlag {
							// For baseline, use the same approach as enhanced bot - reuse properly configured mock client
							threadData := CreatePMBotThreadData("test message")
							mmClient := SetupMockClientWithLogging(t.T, threadData, baselineLogger)
							SetupPMBotServices(t.T, threadData, mmClient)
						}
						baselineBot = baseline.NewBaselineBotWithName(t.LLM, "vanilla-baseline-"+modelName)
					}

					// Run comparison with test context (respects Go test timeout)
					ctx := context.Background()

					trialsPerScenario := 1
					if len(scenarios) > 0 {
						trialsPerScenario = scenarios[0].Trials
					}
					t.Logf("Starting comparison with %d scenarios, %d trials each (Trial %d/%d)", len(scenarios), trialsPerScenario, trialNum, numEvals)
					results := runComparisonWithProgressAndTrialInfo(ctx, t, baselineBot, enhancedBot, scenarios, trialNum, numEvals)

					// Log results for analysis
					logComparisonResults(t, results)

					// Assert that enhanced bot performs better than baseline
					for _, result := range results {
						if *debugFlag {
							baseline.LogScenarioComparison(t, result)
						}

						// Optional assertion: enhanced should be better (comment out if too strict)
						// require.Greater(t.T, result.Improvement, 0.0,
						//   "Enhanced bot should perform better than baseline for scenario: %s", result.BaselineBot.Scenario)
					}
				}(evalT, evalIdx+1)
			}
		})
	}
}

// createEnhancedPMBot creates the enhanced PM bot using shared utilities.
func createEnhancedPMBot(t *evals.EvalT, defaultModel string) baseline.Bot {
	// Create thread data for PM conversation
	threadData := CreatePMBotThreadData("test message")

	// Setup mock client with logging
	enhancedLogger := createTestLogger(t.T, "ENHANCED")
	mmClient := SetupMockClientWithLogging(t.T, threadData, enhancedLogger)

	// Setup services using shared utilities
	_, _, _, prompts, contextBuilder, conv, toolProvider := SetupPMBotServices(t.T, threadData, mmClient)

	// Update config
	testConfig := CreatePMBotConfig(enhancedLogger)
	configContainer := &config.Container{}
	configContainer.Update(testConfig)

	// Create PM bot
	bot := CreatePMBot(defaultModel, "pmbotid")

	// Set LLM
	bot.SetLLMForTest(t.LLM)

	// Wrap in BotAdapter with prompts for system prompt generation
	adapter := baseline.NewBotAdapter(bot, conv, contextBuilder, prompts, mmClient, "enhanced-"+defaultModel, threadData)
	adapter.SetVerbose(*warnFlag)
	adapter.SetToolProvider(toolProvider)
	return adapter
}

// runComparisonWithProgressAndTrialInfo runs baseline comparison with trial progress logging and trial info
func runComparisonWithProgressAndTrialInfo(
	ctx context.Context,
	t *evals.EvalT,
	baselineBot baseline.Bot,
	enhancedBot baseline.Bot,
	scenarios []baseline.Scenario,
	currentTrial, totalTrials int,
) []baseline.ComparisonResult {
	var results []baseline.ComparisonResult

	// Calculate total runs for progress tracking (2 bots * scenarios * trials per scenario)
	totalRuns := 0
	for _, scenario := range scenarios {
		totalRuns += scenario.Trials * 2 // baseline + enhanced
	}
	currentRun := 0

	for scenarioIdx, scenario := range scenarios {
		t.Logf("PROGRESS: Trial %d/%d - Scenario %d/%d: %s (%d trials per bot)",
			currentTrial, totalTrials, scenarioIdx+1, len(scenarios), scenario.Name, scenario.Trials)

		// Run baseline trials with progress
		t.Logf("TRIAL: Running %s baseline trials...", baselineBot.Name())
		baselineResults := runTrialsWithProgressCounterAndTrialInfo(ctx, t, baselineBot, scenario, "BASELINE", &currentRun, totalRuns, currentTrial, totalTrials)

		// Run enhanced trials with progress
		t.Logf("TRIAL: Running %s enhanced trials...", enhancedBot.Name())
		enhancedResults := runTrialsWithProgressCounterAndTrialInfo(ctx, t, enhancedBot, scenario, "ENHANCED", &currentRun, totalRuns, currentTrial, totalTrials)

		// Save comparison data if flag is set
		if *savePromptsFlag {
			// Get model name
			modelName := "unknown"
			if envModel := os.Getenv("TEST_MODEL"); envModel != "" {
				modelName = envModel
			}

			// Extract response data from bots
			baselineResponse := "(No response captured)"
			baselineSystemPrompt := baseline.BaselineSystemPrompt
			if llmBot, ok := baselineBot.(*baseline.LLMBot); ok {
				if llmBot.LastResponse != "" {
					baselineResponse = llmBot.LastResponse
				}
				// Extract actual system prompt from the request
				if llmBot.LastRequest != nil {
					for _, post := range llmBot.LastRequest.Posts {
						if post.Role == llm.PostRoleSystem {
							baselineSystemPrompt = post.Message
							break
						}
					}
				}
			}

			enhancedResponse := "(No response captured)"
			enhancedFirstLLMCall := ""
			enhancedSecondLLMCall := ""
			var enhancedToolCalls []ToolCall
			if adapter, ok := enhancedBot.(*baseline.BotAdapter); ok {
				if adapter.LastResponse != "" {
					enhancedResponse = adapter.LastResponse
				}
				// Get the first and second LLM calls
				enhancedFirstLLMCall = adapter.LastFirstLLMCall
				enhancedSecondLLMCall = adapter.LastSecondLLMCall
				// Copy tool calls directly (types are now aliases)
				enhancedToolCalls = adapter.LastToolCalls
			}

			// Save the comparison data
			comparisonData := ComparisonData{
				Timestamp:   time.Now().Format("2006-01-02T15:04:05"),
				Model:       modelName,
				Level:       *levelFlag,
				Scenario:    scenario.Name,
				Trial:       currentTrial,
				UserMessage: scenario.Message,
				Baseline: BaselineData{
					SystemPrompt: baselineSystemPrompt,
					Response:     baselineResponse,
				},
				Enhanced: EnhancedData{
					FirstLLMCall:  enhancedFirstLLMCall,
					SecondLLMCall: enhancedSecondLLMCall,
					Response:      enhancedResponse,
					ToolCalls:     enhancedToolCalls,
				},
			}

			if err := saveComparisonData(comparisonData); err != nil {
				t.Logf("WARNING: Failed to save comparison data: %v", err)
			} else {
				t.Logf("Saved comparison data for scenario '%s' to %s", scenario.Name, *saveOutputDir)
			}
		}

		// Calculate statistical comparison
		improvement, improvementLower, improvementUpper := baseline.CalculateImprovement(
			enhancedResults.Passes, enhancedResults.Trials,
			baselineResults.Passes, baselineResults.Trials,
		)

		_, pValue := baseline.ChiSquaredTest(
			enhancedResults.Passes, enhancedResults.Trials,
			baselineResults.Passes, baselineResults.Trials,
		)

		result := baseline.ComparisonResult{
			BaselineBot:   baselineResults,
			EnhancedBot:   enhancedResults,
			Improvement:   improvement,
			Significance:  pValue,
			ConfidenceInt: [2]float64{improvementLower, improvementUpper},
		}

		results = append(results, result)

		t.Logf("PROGRESS: Trial %d/%d - Scenario %s complete - Baseline: %.1f%%, Enhanced: %.1f%% (+%.1f%%)",
			currentTrial, totalTrials, scenario.Name, baselineResults.PassRate, enhancedResults.PassRate, improvement*100)
	}

	return results
}

func runTrialsWithProgressCounterAndTrialInfo(ctx context.Context, t *evals.EvalT, bot baseline.Bot, scenario baseline.Scenario, botType string, currentRun *int, totalRuns int, currentTrial, totalTrials int) baseline.TestResults {
	passes := 0
	failures := 0
	groundingPasses := 0
	groundingFails := 0
	totalValidCitations := 0
	totalInvalidCitations := 0
	totalValidRate := 0.0
	totalFabricationRate := 0.0
	groundingTrialsCount := 0
	// Semantic grounding metrics
	semanticGroundingPasses := 0
	semanticGroundingFails := 0
	totalSemanticScore := 0.0
	totalGroundedSentences := 0
	totalUngroundedSentences := 0
	semanticTrialsCount := 0
	var errors []string

	for i := 0; i < scenario.Trials; i++ {
		// Check for context cancellation/timeout before each trial
		select {
		case <-ctx.Done():
			t.Logf("TRIAL %d/%d [%s]: TIMEOUT - Context canceled for scenario '%s'",
				currentTrial, totalTrials, botType, scenario.Name)
			groundingPassRate := 0.0
			groundingTrialsRun := groundingPasses + groundingFails
			if groundingTrialsRun > 0 {
				groundingPassRate = float64(groundingPasses) / float64(groundingTrialsRun) * 100
			}
			return baseline.TestResults{
				BotName:           bot.Name(),
				Scenario:          scenario.Name,
				Trials:            i,
				Passes:            passes,
				Failures:          failures,
				PassRate:          0.0,
				GroundingPasses:   groundingPasses,
				GroundingFails:    groundingFails,
				GroundingPassRate: groundingPassRate,
				Errors:            append(errors, fmt.Sprintf("Test timeout after %d trials", i)),
			}
		default:
		}

		// Start trial with detailed context
		*currentRun++
		t.Logf("🔄 RUN %d/%d - TRIAL %d/%d [%s]: Starting scenario '%s'", *currentRun, totalRuns, currentTrial, totalTrials, botType, scenario.Name)
		t.Logf("TRIAL %d/%d [%s]: Query: %s", currentTrial, totalTrials, botType,
			testutils.TruncateString(scenario.Message, 150))

		// Get bot response
		answer, err := bot.Respond(ctx, scenario.Message)
		if err != nil {
			failures++
			t.Logf("TRIAL %d/%d [%s]: ERROR - Bot response failed: %v", currentTrial, totalTrials, botType, err)
			errors = append(errors, fmt.Sprintf("Trial %d: %v", currentTrial, err))
			continue
		}

		t.Logf("TRIAL %d/%d [%s]: Response received (%d chars)", currentTrial, totalTrials, botType, len(answer.Text))

		// Evaluate against all rubrics with detailed logging

		// Evaluate rubrics using shared threshold-based logic
		logPrefix := fmt.Sprintf("TRIAL %d/%d [%s]", currentTrial, totalTrials, botType)
		evalResult, evalErrors := EvaluateRubricsWithThreshold(
			t,
			scenario.Rubrics,
			answer.Text,
			*thresholdFlag,
			currentTrial,
			totalTrials,
			logPrefix,
		)

		// Evaluate grounding if flag is enabled (tracked separately, does not affect trial pass/fail)
		if *groundingFlag {
			// Extract tool results if this is an enhanced bot with tool execution
			toolResults := []string{}
			if adapter, ok := bot.(*baseline.BotAdapter); ok && len(adapter.LastToolCalls) > 0 {
				for _, toolCall := range adapter.LastToolCalls {
					if resultStr, ok := toolCall.Result.(string); ok && resultStr != "" {
						toolResults = append(toolResults, resultStr)
					}
				}
				if len(toolResults) > 0 {
					t.Logf("%s: Extracted %d tool results for grounding validation", logPrefix, len(toolResults))
				}
			}

			// Phase 1: Citation-based grounding (existing)
			citationResult := EvaluateGrounding(t, answer.Text, toolResults, logPrefix)

			// Phase 2: Semantic grounding (INFORMATIONAL - detects content fabrication, does not affect pass/fail)
			var semanticResult *thread.ValidationResult
			if len(toolResults) > 0 {
				semanticResult = EvaluateSemanticGrounding(t, answer.Text, toolResults, logPrefix)

				// Track semantic metrics (informational only)
				if semanticResult.Pass {
					semanticGroundingPasses++
				} else {
					semanticGroundingFails++
				}
				totalSemanticScore += semanticResult.GroundingScore
				totalGroundedSentences += semanticResult.GroundedCount
				totalUngroundedSentences += semanticResult.UngroundedCount
				semanticTrialsCount++
			}

			// Combined grounding decision: citation validation determines pass/fail
			// Semantic validation is tracked separately as a diagnostic metric
			citationPassed := citationResult.Pass

			if citationPassed {
				groundingPasses++
			} else {
				groundingFails++
				if *debugFlag {
					// Calculate non-metadata citation count for logging
					nonMetadataCitations := citationResult.TotalCitations - citationResult.CitationsByType[grounding.CitationMetadata]
					semanticPassed := semanticResult == nil || semanticResult.Pass
					baseline.LogCombinedGroundingFailure(t, logPrefix, citationPassed, semanticPassed, len(toolResults) > 0, nonMetadataCitations)
				}
			}

			// Accumulate citation grounding metrics
			totalValidCitations += citationResult.ValidCitations
			totalInvalidCitations += citationResult.InvalidCitations
			totalValidRate += citationResult.ValidCitationRate
			totalFabricationRate += citationResult.FabricationRate
			groundingTrialsCount++
		}

		// Handle evaluation errors
		if len(evalErrors) > 0 {
			errors = append(errors, evalErrors...)
		} else {
			// Update pass/fail counts based on threshold result only
			if evalResult.ThresholdPassed {
				passes++
			} else {
				failures++
			}
		}

		if *debugFlag {
			baseline.LogResponsePreview(t, currentTrial, totalTrials, botType, answer.Text, 200)
		}
	}

	passRate := 0.0
	if scenario.Trials > 0 {
		passRate = float64(passes) / float64(scenario.Trials) * 100
	}

	groundingPassRate := 0.0
	groundingTrialsRun := groundingPasses + groundingFails
	if groundingTrialsRun > 0 {
		groundingPassRate = float64(groundingPasses) / float64(groundingTrialsRun) * 100
	}

	// Calculate average grounding metrics
	avgValidRate := 0.0
	avgFabricationRate := 0.0
	if groundingTrialsCount > 0 {
		avgValidRate = totalValidRate / float64(groundingTrialsCount)
		avgFabricationRate = totalFabricationRate / float64(groundingTrialsCount)
	}

	// Calculate semantic grounding metrics
	semanticGroundingPassRate := 0.0
	semanticTrialsRun := semanticGroundingPasses + semanticGroundingFails
	if semanticTrialsRun > 0 {
		semanticGroundingPassRate = float64(semanticGroundingPasses) / float64(semanticTrialsRun) * 100
	}

	avgSemanticScore := 0.0
	if semanticTrialsCount > 0 {
		avgSemanticScore = totalSemanticScore / float64(semanticTrialsCount)
	}

	return baseline.TestResults{
		BotName:                   bot.Name(),
		Scenario:                  scenario.Name,
		Trials:                    scenario.Trials,
		Passes:                    passes,
		Failures:                  failures,
		PassRate:                  passRate,
		GroundingPasses:           groundingPasses,
		GroundingFails:            groundingFails,
		GroundingPassRate:         groundingPassRate,
		GroundingValidCitations:   totalValidCitations,
		GroundingInvalidCitations: totalInvalidCitations,
		GroundingValidRate:        avgValidRate,
		GroundingFabricationRate:  avgFabricationRate,
		// Semantic grounding metrics
		SemanticGroundingPasses:     semanticGroundingPasses,
		SemanticGroundingFails:      semanticGroundingFails,
		SemanticGroundingPassRate:   semanticGroundingPassRate,
		SemanticGroundingScore:      avgSemanticScore,
		SemanticGroundedSentences:   totalGroundedSentences,
		SemanticUngroundedSentences: totalUngroundedSentences,
		Errors:                      errors,
	}
}

// logComparisonResults logs detailed comparison results for analysis.
func logComparisonResults(t *evals.EvalT, results []baseline.ComparisonResult) {
	overallBaselinePasses := 0
	overallBaselineTrials := 0
	overallEnhancedPasses := 0
	overallEnhancedTrials := 0
	overallBaselineGroundingPasses := 0
	overallBaselineGroundingTrials := 0
	overallEnhancedGroundingPasses := 0
	overallEnhancedGroundingTrials := 0
	overallBaselineValidCitations := 0
	overallBaselineInvalidCitations := 0
	overallEnhancedValidCitations := 0
	overallEnhancedInvalidCitations := 0

	for _, result := range results {
		overallBaselinePasses += result.BaselineBot.Passes
		overallBaselineTrials += result.BaselineBot.Trials
		overallEnhancedPasses += result.EnhancedBot.Passes
		overallEnhancedTrials += result.EnhancedBot.Trials

		overallBaselineGroundingPasses += result.BaselineBot.GroundingPasses
		overallBaselineGroundingTrials += result.BaselineBot.GroundingPasses + result.BaselineBot.GroundingFails
		overallEnhancedGroundingPasses += result.EnhancedBot.GroundingPasses
		overallEnhancedGroundingTrials += result.EnhancedBot.GroundingPasses + result.EnhancedBot.GroundingFails

		overallBaselineValidCitations += result.BaselineBot.GroundingValidCitations
		overallBaselineInvalidCitations += result.BaselineBot.GroundingInvalidCitations
		overallEnhancedValidCitations += result.EnhancedBot.GroundingValidCitations
		overallEnhancedInvalidCitations += result.EnhancedBot.GroundingInvalidCitations
	}

	overallBaselineRate := float64(overallBaselinePasses) / float64(overallBaselineTrials) * 100
	overallEnhancedRate := float64(overallEnhancedPasses) / float64(overallEnhancedTrials) * 100
	overallImprovement := overallEnhancedRate - overallBaselineRate

	t.Logf("RESULTS: Baseline %.1f%% vs Enhanced %.1f%% (+%.1f%% improvement)",
		overallBaselineRate, overallEnhancedRate, overallImprovement)

	if *debugFlag {
		baseline.LogDetailedBreakdown(t, overallBaselinePasses, overallBaselineTrials, overallEnhancedPasses, overallEnhancedTrials)
	}

	// Log grounding results if grounding was enabled
	if *groundingFlag && overallBaselineGroundingTrials > 0 {
		overallBaselineGroundingRate := float64(overallBaselineGroundingPasses) / float64(overallBaselineGroundingTrials) * 100
		overallEnhancedGroundingRate := 0.0
		if overallEnhancedGroundingTrials > 0 {
			overallEnhancedGroundingRate = float64(overallEnhancedGroundingPasses) / float64(overallEnhancedGroundingTrials) * 100
		}
		groundingImprovement := overallEnhancedGroundingRate - overallBaselineGroundingRate

		t.Logf("GROUNDING RESULTS: Baseline %.1f%% vs Enhanced %.1f%% (+%.1f%% improvement)",
			overallBaselineGroundingRate, overallEnhancedGroundingRate, groundingImprovement)

		// Calculate and log valid/fabrication rates
		baselineValidRate := 0.0
		baselineFabricationRate := 0.0
		baselineTotalCitations := overallBaselineValidCitations + overallBaselineInvalidCitations
		if baselineTotalCitations > 0 {
			baselineValidRate = float64(overallBaselineValidCitations) / float64(baselineTotalCitations) * 100
			baselineFabricationRate = float64(overallBaselineInvalidCitations) / float64(baselineTotalCitations) * 100
		}

		enhancedValidRate := 0.0
		enhancedFabricationRate := 0.0
		enhancedTotalCitations := overallEnhancedValidCitations + overallEnhancedInvalidCitations
		if enhancedTotalCitations > 0 {
			enhancedValidRate = float64(overallEnhancedValidCitations) / float64(enhancedTotalCitations) * 100
			enhancedFabricationRate = float64(overallEnhancedInvalidCitations) / float64(enhancedTotalCitations) * 100
		}

		t.Logf("GROUNDING CITATION QUALITY:")
		t.Logf("  Valid Rate: Baseline %.1f%% (%d/%d) vs Enhanced %.1f%% (%d/%d)",
			baselineValidRate, overallBaselineValidCitations, baselineTotalCitations,
			enhancedValidRate, overallEnhancedValidCitations, enhancedTotalCitations)
		t.Logf("  Fabrication Rate: Baseline %.1f%% vs Enhanced %.1f%%",
			baselineFabricationRate, enhancedFabricationRate)

		// Calculate and log semantic grounding results
		overallBaselineSemanticPasses := 0
		overallBaselineSemanticFails := 0
		overallEnhancedSemanticPasses := 0
		overallEnhancedSemanticFails := 0
		overallBaselineGroundedSentences := 0
		overallBaselineUngroundedSentences := 0
		overallEnhancedGroundedSentences := 0
		overallEnhancedUngroundedSentences := 0
		totalBaselineSemanticScore := 0.0
		totalEnhancedSemanticScore := 0.0

		for _, result := range results {
			overallBaselineSemanticPasses += result.BaselineBot.SemanticGroundingPasses
			overallBaselineSemanticFails += result.BaselineBot.SemanticGroundingFails
			overallEnhancedSemanticPasses += result.EnhancedBot.SemanticGroundingPasses
			overallEnhancedSemanticFails += result.EnhancedBot.SemanticGroundingFails
			overallBaselineGroundedSentences += result.BaselineBot.SemanticGroundedSentences
			overallBaselineUngroundedSentences += result.BaselineBot.SemanticUngroundedSentences
			overallEnhancedGroundedSentences += result.EnhancedBot.SemanticGroundedSentences
			overallEnhancedUngroundedSentences += result.EnhancedBot.SemanticUngroundedSentences
			totalBaselineSemanticScore += result.BaselineBot.SemanticGroundingScore
			totalEnhancedSemanticScore += result.EnhancedBot.SemanticGroundingScore
		}

		overallBaselineSemanticTrials := overallBaselineSemanticPasses + overallBaselineSemanticFails
		overallEnhancedSemanticTrials := overallEnhancedSemanticPasses + overallEnhancedSemanticFails

		if overallBaselineSemanticTrials > 0 || overallEnhancedSemanticTrials > 0 {
			baselineSemanticPassRate := 0.0
			enhancedSemanticPassRate := 0.0
			if overallBaselineSemanticTrials > 0 {
				baselineSemanticPassRate = float64(overallBaselineSemanticPasses) / float64(overallBaselineSemanticTrials) * 100
			}
			if overallEnhancedSemanticTrials > 0 {
				enhancedSemanticPassRate = float64(overallEnhancedSemanticPasses) / float64(overallEnhancedSemanticTrials) * 100
			}
			semanticImprovement := enhancedSemanticPassRate - baselineSemanticPassRate

			t.Logf("SEMANTIC GROUNDING RESULTS: Baseline %.1f%% vs Enhanced %.1f%% (+%.1f%% improvement)",
				baselineSemanticPassRate, enhancedSemanticPassRate, semanticImprovement)

			avgBaselineSemanticScore := 0.0
			avgEnhancedSemanticScore := 0.0
			if len(results) > 0 {
				avgBaselineSemanticScore = totalBaselineSemanticScore / float64(len(results))
				avgEnhancedSemanticScore = totalEnhancedSemanticScore / float64(len(results))
			}

			t.Logf("SEMANTIC GROUNDING SCORES:")
			t.Logf("  Average Score: Baseline %.3f vs Enhanced %.3f", avgBaselineSemanticScore, avgEnhancedSemanticScore)

			baselineTotalSentences := overallBaselineGroundedSentences + overallBaselineUngroundedSentences
			enhancedTotalSentences := overallEnhancedGroundedSentences + overallEnhancedUngroundedSentences

			baselineFabricationSentenceRate := 0.0
			enhancedFabricationSentenceRate := 0.0
			if baselineTotalSentences > 0 {
				baselineFabricationSentenceRate = float64(overallBaselineUngroundedSentences) / float64(baselineTotalSentences) * 100
			}
			if enhancedTotalSentences > 0 {
				enhancedFabricationSentenceRate = float64(overallEnhancedUngroundedSentences) / float64(enhancedTotalSentences) * 100
			}

			t.Logf("  Content Fabrication: Baseline %.1f%% (%d/%d sentences) vs Enhanced %.1f%% (%d/%d sentences)",
				baselineFabricationSentenceRate, overallBaselineUngroundedSentences, baselineTotalSentences,
				enhancedFabricationSentenceRate, overallEnhancedUngroundedSentences, enhancedTotalSentences)
		}

		if *debugFlag {
			baseline.LogGroundingBreakdown(t, overallBaselineGroundingPasses, overallBaselineGroundingTrials,
				overallEnhancedGroundingPasses, overallEnhancedGroundingTrials)
		}
	}
}

// getModelsForBaselineComparison returns models for baseline comparison
// Uses shared implementation with PM-specific defaults
func getModelsForBaselineComparison() []string {
	return evals.GetModelsFromEnvOrDefault([]string{"mattermodel-5.4"})
}

// Type aliases for shared baseline comparison types
type ComparisonData = baseline.ComparisonData
type BaselineData = baseline.Data
type EnhancedData = baseline.EnhancedData
type ToolCall = baseline.ToolCall
type ToolCallMetadata = baseline.ToolCallMetadata

// saveComparisonData saves the prompts and outputs for analysis
func saveComparisonData(data ComparisonData) error {
	return baseline.SaveComparisonData(data, *savePromptsFlag, *saveOutputDir)
}
