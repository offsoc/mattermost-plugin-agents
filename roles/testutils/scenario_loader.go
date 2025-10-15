// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package testutils

import (
	"fmt"
	"io/fs"

	"gopkg.in/yaml.v3"
)

// YAMLRubrics represents the simple rubrics format used by both PM and Dev bots
type YAMLRubrics struct {
	Rubrics []string `yaml:"rubrics"`
}

// LoadRubricsFromFS loads rubrics from an embedded filesystem
func LoadRubricsFromFS(fsys fs.FS, filePath string) ([]string, error) {
	data, err := fs.ReadFile(fsys, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read rubrics file %s: %w", filePath, err)
	}

	var rubrics YAMLRubrics
	if err := yaml.Unmarshal(data, &rubrics); err != nil {
		return nil, fmt.Errorf("failed to parse rubrics file %s: %w", filePath, err)
	}

	if len(rubrics.Rubrics) == 0 {
		return nil, fmt.Errorf("rubrics file %s contains no rubrics", filePath)
	}

	return rubrics.Rubrics, nil
}
