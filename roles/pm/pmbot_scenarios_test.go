// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package pm_test

import (
	"fmt"
	"testing"

	"github.com/mattermost/mattermost-plugin-ai/evals/baseline"
)

// FilterScenariosByFlag filters scenarios based on command line flag
func FilterScenariosByFlag(scenarios []baseline.Scenario, flag string, t *testing.T) []baseline.Scenario {
	var filteredScenarios []baseline.Scenario

	switch flag {
	case "CORE":
		filteredScenarios = scenarios[:10] // First 10 scenarios (core prioritization questions)
		t.Logf("Running CORE scenarios (%d questions)", len(filteredScenarios))
	case "BREADTH":
		filteredScenarios = scenarios[10:] // Last 15 scenarios (breadth questions)
		t.Logf("Running BREADTH scenarios (%d questions)", len(filteredScenarios))
	case "ALL":
		filteredScenarios = scenarios // All 25 scenarios
		t.Logf("Running ALL scenarios (%d questions)", len(filteredScenarios))
	default:
		t.Fatalf("Invalid scenarios flag: %s. Use 'CORE', 'BREADTH', or 'ALL' (use --level for junior/senior selection)", flag)
	}
	return filteredScenarios
}

// LoadPMBotScenarios loads PM scenarios from YAML files based on level and mm-centric flags
// level: "junior" or "senior"
// mmCentric: true for Mattermost-specific scenarios, false for generic scenarios
func LoadPMBotScenarios(level string, mmCentric bool) ([]baseline.Scenario, error) {
	switch level {
	case "junior":
		return LoadJuniorPMScenarios(mmCentric)
	case "senior":
		return LoadSeniorPMScenarios(mmCentric)
	default:
		return nil, fmt.Errorf("invalid level: %s; must be 'junior' or 'senior'", level)
	}
}
