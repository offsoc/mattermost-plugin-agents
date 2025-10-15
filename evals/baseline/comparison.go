// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package baseline

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Scenario represents a test scenario for baseline comparison.
type Scenario struct {
	Name    string
	Message string
	Rubrics []string
	Trials  int
}

// ComparisonData stores all the data we want to save for analysis
type ComparisonData struct {
	Timestamp   string                 `json:"timestamp"`
	Model       string                 `json:"model"`
	Level       string                 `json:"level"`
	Scenario    string                 `json:"scenario"`
	Trial       int                    `json:"trial"`
	UserMessage string                 `json:"user_message"`
	Baseline    Data                   `json:"baseline"`
	Enhanced    EnhancedData           `json:"enhanced"`
	EvalResults map[string]interface{} `json:"eval_results,omitempty"`
}

// Data stores baseline bot data
type Data struct {
	SystemPrompt string `json:"system_prompt"`
	Response     string `json:"response"`
	Latency      string `json:"latency,omitempty"`
}

// EnhancedData stores enhanced bot data including tool calls
type EnhancedData struct {
	FirstLLMCall  string     `json:"first_llm_call"`
	SecondLLMCall string     `json:"second_llm_call,omitempty"`
	Response      string     `json:"response"`
	ToolCalls     []ToolCall `json:"tool_calls,omitempty"`
	Latency       string     `json:"latency,omitempty"`
}

// SaveComparisonData saves the prompts and outputs for analysis
func SaveComparisonData(data ComparisonData, savePromptsFlag bool, saveOutputDir string) error {
	if !savePromptsFlag {
		return nil
	}

	if err := os.MkdirAll(saveOutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	timestamp := time.Now().Format("20060102_150405")
	safeScenarioName := strings.ReplaceAll(data.Scenario, " ", "_")
	safeScenarioName = strings.ReplaceAll(safeScenarioName, "/", "-")
	safeLevel := strings.ToLower(data.Level)
	if safeLevel == "" {
		safeLevel = "unknown"
	}
	filename := fmt.Sprintf("%s_%s_%s_trial%d_%s.json",
		timestamp, safeLevel, safeScenarioName, data.Trial, data.Model)

	filePath := filepath.Join(saveOutputDir, filename)

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal comparison data: %w", err)
	}

	if err := os.WriteFile(filePath, jsonData, 0600); err != nil {
		return fmt.Errorf("failed to write comparison data to file: %w", err)
	}

	return nil
}
