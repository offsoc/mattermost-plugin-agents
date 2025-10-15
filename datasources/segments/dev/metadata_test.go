// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package dev

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMetadata_GetPriority(t *testing.T) {
	tests := []struct {
		name     string
		severity Severity
		expected string
	}{
		{
			name:     "critical severity maps to high priority",
			severity: SeverityCritical,
			expected: "high",
		},
		{
			name:     "high severity maps to high priority",
			severity: SeverityHigh,
			expected: "high",
		},
		{
			name:     "medium severity maps to medium priority",
			severity: SeverityMedium,
			expected: "medium",
		},
		{
			name:     "low severity maps to low priority",
			severity: SeverityLow,
			expected: "low",
		},
		{
			name:     "trivial severity maps to low priority",
			severity: SeverityTrivial,
			expected: "low",
		},
		{
			name:     "empty severity maps to none",
			severity: "",
			expected: "none",
		},
		{
			name:     "unknown severity maps to none",
			severity: "unknown",
			expected: "none",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Metadata{Severity: tt.severity}
			result := m.GetPriority()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMetadata_GetLabels(t *testing.T) {
	tests := []struct {
		name     string
		metadata Metadata
		expected []string
	}{
		{
			name: "all fields populated",
			metadata: Metadata{
				Severity:   SeverityHigh,
				IssueType:  IssueTypeBug,
				Components: []Component{ComponentServerAPI, ComponentWebappComponents},
				Languages:  []Language{LanguageGo, LanguageTypeScript},
				Complexity: ComplexityComplex,
			},
			expected: []string{
				"severity:high",
				"type:bug",
				"component:server/api",
				"component:webapp/components",
				"lang:go",
				"lang:typescript",
				"complexity:complex",
			},
		},
		{
			name: "only severity",
			metadata: Metadata{
				Severity: SeverityCritical,
			},
			expected: []string{"severity:critical"},
		},
		{
			name: "only issue type",
			metadata: Metadata{
				IssueType: IssueTypeFeature,
			},
			expected: []string{"type:feature"},
		},
		{
			name: "multiple components",
			metadata: Metadata{
				Components: []Component{ComponentServerAPI, ComponentServerDatabase, ComponentServerAuth},
			},
			expected: []string{
				"component:server/api",
				"component:server/database",
				"component:server/auth",
			},
		},
		{
			name: "multiple languages",
			metadata: Metadata{
				Languages: []Language{LanguageGo, LanguageSQL, LanguagePython},
			},
			expected: []string{
				"lang:go",
				"lang:sql",
				"lang:python",
			},
		},
		{
			name: "only complexity",
			metadata: Metadata{
				Complexity: ComplexitySimple,
			},
			expected: []string{"complexity:simple"},
		},
		{
			name:     "empty metadata",
			metadata: Metadata{},
			expected: []string{},
		},
		{
			name: "partial metadata with empty strings",
			metadata: Metadata{
				Severity:   "",
				IssueType:  IssueTypeBug,
				Components: []Component{},
				Languages:  nil,
				Complexity: ComplexityMedium,
			},
			expected: []string{
				"type:bug",
				"complexity:medium",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.metadata.GetLabels()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMetadata_Summary(t *testing.T) {
	tests := []struct {
		name     string
		metadata Metadata
		expected string
	}{
		{
			name: "all fields populated",
			metadata: Metadata{
				Severity:   SeverityHigh,
				IssueType:  IssueTypeBug,
				Components: []Component{ComponentServerAPI, ComponentWebappComponents},
				Languages:  []Language{LanguageGo, LanguageTypeScript},
				Complexity: ComplexityComplex,
			},
			expected: "(Severity: high | Type: bug | Components: server/api, webapp/components | Languages: go, typescript | Complexity: complex)",
		},
		{
			name: "only severity",
			metadata: Metadata{
				Severity: SeverityCritical,
			},
			expected: "(Severity: critical)",
		},
		{
			name: "only issue type",
			metadata: Metadata{
				IssueType: IssueTypeFeature,
			},
			expected: "(Type: feature)",
		},
		{
			name: "severity and issue type",
			metadata: Metadata{
				Severity:  SeverityMedium,
				IssueType: IssueTypeRefactor,
			},
			expected: "(Severity: medium | Type: refactor)",
		},
		{
			name: "components only",
			metadata: Metadata{
				Components: []Component{ComponentServerAPI, ComponentServerDatabase},
			},
			expected: "(Components: server/api, server/database)",
		},
		{
			name: "single component",
			metadata: Metadata{
				Components: []Component{ComponentServerAPI},
			},
			expected: "(Components: server/api)",
		},
		{
			name: "languages only",
			metadata: Metadata{
				Languages: []Language{LanguageGo, LanguageSQL},
			},
			expected: "(Languages: go, sql)",
		},
		{
			name: "single language",
			metadata: Metadata{
				Languages: []Language{LanguageTypeScript},
			},
			expected: "(Languages: typescript)",
		},
		{
			name: "complexity only",
			metadata: Metadata{
				Complexity: ComplexitySimple,
			},
			expected: "(Complexity: simple)",
		},
		{
			name:     "empty metadata returns empty string",
			metadata: Metadata{},
			expected: "",
		},
		{
			name: "partial metadata with empty strings",
			metadata: Metadata{
				Severity:   "",
				IssueType:  IssueTypeTechDebt,
				Components: []Component{},
				Languages:  nil,
				Complexity: "",
			},
			expected: "(Type: tech_debt)",
		},
		{
			name: "tech debt with infrastructure component",
			metadata: Metadata{
				Severity:   SeverityLow,
				IssueType:  IssueTypeInfra,
				Components: []Component{ComponentServerDatabase},
				Complexity: ComplexityMedium,
			},
			expected: "(Severity: low | Type: infrastructure | Components: server/database | Complexity: medium)",
		},
		{
			name: "documentation issue",
			metadata: Metadata{
				IssueType:  IssueTypeDocumentation,
				Complexity: ComplexitySimple,
			},
			expected: "(Type: documentation | Complexity: simple)",
		},
		{
			name: "test issue with multiple components",
			metadata: Metadata{
				Severity:   SeverityTrivial,
				IssueType:  IssueTypeTest,
				Components: []Component{ComponentServerAPI, ComponentWebappComponents, ComponentServerDatabase},
			},
			expected: "(Severity: trivial | Type: test | Components: server/api, webapp/components, server/database)",
		},
		{
			name: "mobile components",
			metadata: Metadata{
				IssueType:  IssueTypeBug,
				Components: []Component{ComponentMobileIOS, ComponentMobileAndroid},
				Languages:  []Language{LanguageSwift, LanguageKotlin},
			},
			expected: "(Type: bug | Components: mobile/ios, mobile/android | Languages: swift, kotlin)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.metadata.Summary()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMetadata_ConsistencyBetweenLabelsAndSummary(t *testing.T) {
	metadata := Metadata{
		Severity:   SeverityHigh,
		IssueType:  IssueTypeBug,
		Components: []Component{ComponentServerAPI},
		Languages:  []Language{LanguageGo},
		Complexity: ComplexityMedium,
	}

	labels := metadata.GetLabels()
	summary := metadata.Summary()

	assert.Contains(t, labels, "severity:high")
	assert.Contains(t, summary, "Severity: high")

	assert.Contains(t, labels, "type:bug")
	assert.Contains(t, summary, "Type: bug")

	assert.Contains(t, labels, "component:server/api")
	assert.Contains(t, summary, "Components: server/api")

	assert.Contains(t, labels, "lang:go")
	assert.Contains(t, summary, "Languages: go")

	assert.Contains(t, labels, "complexity:medium")
	assert.Contains(t, summary, "Complexity: medium")
}
