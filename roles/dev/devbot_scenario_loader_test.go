// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package dev_test

import (
	"embed"
	"fmt"

	"github.com/mattermost/mattermost-plugin-ai/evals/baseline"
	"github.com/mattermost/mattermost-plugin-ai/roles/testutils"
	"gopkg.in/yaml.v3"
)

//go:embed scenarios/*.yaml scenarios/rubrics/*.yaml
var scenariosFS embed.FS

type YAMLDevScenario struct {
	ID              string   `yaml:"id"`
	Name            string   `yaml:"name"`
	Query           string   `yaml:"query"`
	Category        string   `yaml:"category"`
	ExpectedTools   []string `yaml:"expected_tools"`
	ExpectedContent []string `yaml:"expected_content"`
	Difficulty      string   `yaml:"difficulty"`
}

type YAMLDevScenarios struct {
	Scenarios []YAMLDevScenario `yaml:"scenarios"`
}

func loadDevRubrics(filePath string) ([]string, error) {
	return testutils.LoadRubricsFromFS(scenariosFS, filePath)
}

func loadDevScenarios(scenariosFile, rubricsFile string) ([]baseline.Scenario, error) {
	scenariosData, err := scenariosFS.ReadFile(scenariosFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read scenarios file %s: %w", scenariosFile, err)
	}

	var yamlScenarios YAMLDevScenarios
	err = yaml.Unmarshal(scenariosData, &yamlScenarios)
	if err != nil {
		return nil, fmt.Errorf("failed to parse scenarios file %s: %w", scenariosFile, err)
	}

	rubrics, err := loadDevRubrics(rubricsFile)
	if err != nil {
		return nil, err
	}

	scenarios := make([]baseline.Scenario, 0, len(yamlScenarios.Scenarios))
	for _, s := range yamlScenarios.Scenarios {
		scenarios = append(scenarios, baseline.Scenario{
			Name:    s.Name,
			Message: s.Query,
			Rubrics: rubrics,
			Trials:  1,
		})
	}

	return scenarios, nil
}

func LoadJuniorDevScenarios() ([]baseline.Scenario, error) {
	return loadDevScenarios(
		"scenarios/junior_dev_mattermost.yaml",
		"scenarios/rubrics/junior_mattermost.yaml",
	)
}

func LoadSeniorDevScenarios() ([]baseline.Scenario, error) {
	return loadDevScenarios(
		"scenarios/senior_dev_mattermost.yaml",
		"scenarios/rubrics/senior_mattermost.yaml",
	)
}

func LoadDevBotScenarios(level string) ([]baseline.Scenario, error) {
	switch level {
	case "junior":
		return LoadJuniorDevScenarios()
	case "senior":
		return LoadSeniorDevScenarios()
	default:
		return nil, fmt.Errorf("invalid level: %s; must be 'junior' or 'senior'", level)
	}
}
