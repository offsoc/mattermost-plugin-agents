// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package pm_test

import (
	"embed"
	"fmt"

	"github.com/mattermost/mattermost-plugin-ai/evals/baseline"
	"github.com/mattermost/mattermost-plugin-ai/roles/testutils"
	"gopkg.in/yaml.v3"
)

//go:embed scenarios/*.yaml scenarios/rubrics/*.yaml
var scenariosFS embed.FS

type YAMLScenario struct {
	Name    string `yaml:"name"`
	Message string `yaml:"message"`
	Trials  int    `yaml:"trials"`
}

type YAMLScenarios struct {
	Scenarios []YAMLScenario `yaml:"scenarios"`
}

func loadRubrics(filePath string) ([]string, error) {
	return testutils.LoadRubricsFromFS(scenariosFS, filePath)
}

func loadScenarios(scenariosFile, rubricsFile string) ([]baseline.Scenario, error) {
	scenariosData, err := scenariosFS.ReadFile(scenariosFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read scenarios file %s: %w", scenariosFile, err)
	}

	var yamlScenarios YAMLScenarios
	err = yaml.Unmarshal(scenariosData, &yamlScenarios)
	if err != nil {
		return nil, fmt.Errorf("failed to parse scenarios file %s: %w", scenariosFile, err)
	}

	rubrics, err := loadRubrics(rubricsFile)
	if err != nil {
		return nil, err
	}

	scenarios := make([]baseline.Scenario, 0, len(yamlScenarios.Scenarios))
	for _, s := range yamlScenarios.Scenarios {
		scenarios = append(scenarios, baseline.Scenario{
			Name:    s.Name,
			Message: s.Message,
			Rubrics: rubrics,
			Trials:  s.Trials,
		})
	}

	return scenarios, nil
}

func LoadJuniorPMScenarios(mmCentric bool) ([]baseline.Scenario, error) {
	if mmCentric {
		return loadScenarios(
			"scenarios/junior_pm_mattermost.yaml",
			"scenarios/rubrics/junior_mattermost.yaml",
		)
	}
	return loadScenarios(
		"scenarios/junior_pm_generic.yaml",
		"scenarios/rubrics/junior_generic.yaml",
	)
}

func LoadSeniorPMScenarios(mmCentric bool) ([]baseline.Scenario, error) {
	if mmCentric {
		return loadScenarios(
			"scenarios/senior_pm_mattermost.yaml",
			"scenarios/rubrics/senior_mattermost.yaml",
		)
	}
	return loadScenarios(
		"scenarios/senior_pm_generic.yaml",
		"scenarios/rubrics/senior_generic.yaml",
	)
}
