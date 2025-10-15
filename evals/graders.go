// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package evals

import (
	"encoding/json"
	"fmt"

	"github.com/mattermost/mattermost-plugin-ai/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type RubricResult struct {
	Reasoning string  `json:"reasoning"`
	Score     float64 `json:"score"`
	Pass      bool    `json:"pass"`
}

// RubricSetResult aggregates results across multiple rubrics for a single model
type RubricSetResult struct {
	ToolName      string
	RubricResults []RubricResult
	Model         string
	PassRate      float64 // Percentage of rubrics that passed
	AvgScore      float64 // Average score across all rubrics
}

// BaselineComparisonResult compares enhanced vs baseline performance
type BaselineComparisonResult struct {
	ToolName            string
	BaselineResult      RubricSetResult
	EnhancedResult      RubricSetResult
	RubricBreakdown     map[string]float64 // rubric -> improvement in score
	PassRateImprovement float64            // Difference in pass rates
	AvgScoreImprovement float64            // Difference in average scores
}

const llmRubricSystem = `You are grading output according to the specificed rebric. If the statemnt in the rubric is true, then the output passes the test. You must respond with a JSON object with this structure: {reasoning: string, score: number, pass: boolean}
Examples:
<Output>The steamclock is broken</Output>
<Rubric>The content contains the state of the clock</Rubric>
{"reasoning": "The output says the clock is broken", "score": 1.0, "pass": true}

<Output>I am sorry I can not find the thread you referenced</Output>
<Rubric>Contains a reference to the mentos project</Rubric>
{"reasoning": "The output contains a failure message instead of a reference to the mentos project", "score": 0.0, "pass": false}`

func (e *Eval) LLMRubric(rubric, output string) (*RubricResult, error) {
	req := llm.CompletionRequest{
		Posts: []llm.Post{
			{
				Role:    llm.PostRoleSystem,
				Message: llmRubricSystem,
			},
			{
				Role:    llm.PostRoleUser,
				Message: fmt.Sprintf("<Output>%s</Output>\n<Rubric>%s</Rubric>", output, rubric),
			},
		},
		Context: llm.NewContext(),
	}

	llmResult, gradeErr := e.GraderLLM.ChatCompletionNoStream(req, llm.WithMaxGeneratedTokens(1000), llm.WithJSONOutput[RubricResult]())
	if gradeErr != nil {
		return nil, fmt.Errorf("failed to grade with llm: %w", gradeErr)
	}

	rubricResult := RubricResult{}
	unmarshalErr := json.Unmarshal([]byte(llmResult), &rubricResult)
	if unmarshalErr != nil {
		return nil, fmt.Errorf("failed to unmarshal llm result: %w", unmarshalErr)
	}

	return &rubricResult, nil
}

func LLMRubricT(e *EvalT, rubric, output string) {
	LLMRubricTWithModel(e, rubric, output, "")
}

func LLMRubricTWithModel(e *EvalT, rubric, output, model string) {
	e.Helper()
	result, err := e.LLMRubric(rubric, output)
	require.NoError(e.T, err)

	// Debug output when score is low
	if result.Score < 0.6 {
		e.Logf("DEBUG - Low score detected (%.2f):", result.Score)
		e.Logf("Rubric: %s", rubric)
		e.Logf("Output: %s", output)
		e.Logf("Reasoning: %s", result.Reasoning)
	}

	RecordScore(e, &EvalResult{
		Model:     model,
		Rubric:    rubric,
		Output:    output,
		Reasoning: result.Reasoning,
		Score:     result.Score,
		Pass:      result.Pass,
	})
	assert.True(e.T, result.Pass, "LLM Rubric Failed")
	assert.GreaterOrEqual(e.T, result.Score, 0.6, "LLM Rubric Score is too low")
}

// CalculateRubricSetResult aggregates multiple rubric results into summary metrics
func CalculateRubricSetResult(toolName, model string, results []RubricResult) RubricSetResult {
	if len(results) == 0 {
		return RubricSetResult{
			ToolName:      toolName,
			Model:         model,
			RubricResults: results,
			PassRate:      0.0,
			AvgScore:      0.0,
		}
	}

	var totalScore float64
	var passCount int

	for _, result := range results {
		totalScore += result.Score
		if result.Pass {
			passCount++
		}
	}

	return RubricSetResult{
		ToolName:      toolName,
		Model:         model,
		RubricResults: results,
		PassRate:      float64(passCount) / float64(len(results)),
		AvgScore:      totalScore / float64(len(results)),
	}
}

// CompareBaselineResults creates a comparison between baseline and enhanced results
func CompareBaselineResults(toolName string, baseline, enhanced RubricSetResult) BaselineComparisonResult {
	rubricBreakdown := make(map[string]float64)

	// Calculate per-rubric improvements (assuming rubrics are in same order)
	for i, baselineResult := range baseline.RubricResults {
		if i < len(enhanced.RubricResults) {
			rubricName := fmt.Sprintf("rubric_%d", i+1)
			rubricBreakdown[rubricName] = enhanced.RubricResults[i].Score - baselineResult.Score
		}
	}

	return BaselineComparisonResult{
		ToolName:            toolName,
		BaselineResult:      baseline,
		EnhancedResult:      enhanced,
		RubricBreakdown:     rubricBreakdown,
		PassRateImprovement: enhanced.PassRate - baseline.PassRate,
		AvgScoreImprovement: enhanced.AvgScore - baseline.AvgScore,
	}
}
