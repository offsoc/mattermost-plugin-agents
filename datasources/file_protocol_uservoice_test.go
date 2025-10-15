// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"testing"

	pm "github.com/mattermost/mattermost-plugin-ai/datasources/segments/pm"
)

func TestExtractUserVoiceMetadata(t *testing.T) {
	tests := []struct {
		name                     string
		suggestion               UserVoiceSuggestion
		expectedSegments         []pm.CustomerSegment
		expectedCategories       []pm.TechnicalCategory
		expectedCompetitive      pm.Competitor
		expectedPriorityNotEmpty bool
	}{
		{
			name: "federal_mobile_high_votes",
			suggestion: UserVoiceSuggestion{
				ID:          "12345",
				Title:       "Mobile offline sync for tactical deployments",
				Description: "High-security customers need mobile app to work in disconnected environments for classified operations",
				Status:      "planned",
				Votes:       156,
				Comments:    23,
				Category:    "Mobile",
			},
			expectedSegments:         []pm.CustomerSegment{pm.SegmentFederal},
			expectedCategories:       []pm.TechnicalCategory{pm.CategoryMobile},
			expectedPriorityNotEmpty: true,
		},
		{
			name: "enterprise_sso_authentication",
			suggestion: UserVoiceSuggestion{
				ID:          "23456",
				Title:       "Enterprise SSO with SAML 2.0 support",
				Description: "Large organizations require enterprise-grade SAML authentication for 10000+ users with audit logging",
				Status:      "under_review",
				Votes:       89,
				Comments:    15,
				Category:    "Authentication",
			},
			expectedSegments:         []pm.CustomerSegment{pm.SegmentEnterprise},
			expectedCategories:       []pm.TechnicalCategory{pm.CategoryAuthentication},
			expectedPriorityNotEmpty: true,
		},
		{
			name: "healthcare_hipaa_compliance",
			suggestion: UserVoiceSuggestion{
				ID:          "34567",
				Title:       "HIPAA compliance features for healthcare providers",
				Description: "Hospital deployment needs HIPAA audit logs and database encryption for patient data protection",
				Status:      "in_progress",
				Votes:       124,
				Comments:    31,
				Category:    "Compliance",
			},
			expectedSegments:         []pm.CustomerSegment{pm.SegmentHealthcare, pm.SegmentEnterprise},
			expectedCategories:       []pm.TechnicalCategory{pm.CategoryCompliance, pm.CategoryDatabase},
			expectedPriorityNotEmpty: true,
		},
		{
			name: "devops_kubernetes_integration",
			suggestion: UserVoiceSuggestion{
				ID:          "45678",
				Title:       "Kubernetes operator for automated deployments",
				Description: "DevOps teams need k8s integration with CI/CD pipelines for container orchestration workflows",
				Status:      "open",
				Votes:       67,
				Comments:    12,
				Category:    "Integrations",
			},
			expectedSegments:         []pm.CustomerSegment{pm.SegmentDevOps},
			expectedCategories:       []pm.TechnicalCategory{pm.CategoryIntegrations, pm.CategoryPlugins},
			expectedPriorityNotEmpty: true,
		},
		{
			name: "slack_competitive_migration",
			suggestion: UserVoiceSuggestion{
				ID:          "56789",
				Title:       "Slack import tool for enterprise migration",
				Description: "Companies migrating from Slack need automated channel and message import with user mapping",
				Status:      "completed",
				Votes:       201,
				Comments:    45,
				Category:    "Integrations",
			},
			expectedSegments:         []pm.CustomerSegment{pm.SegmentEnterprise},
			expectedCategories:       []pm.TechnicalCategory{pm.CategoryChannels, pm.CategoryDatabase},
			expectedCompetitive:      pm.CompetitorSlack,
			expectedPriorityNotEmpty: true,
		},
		{
			name: "playbooks_automation",
			suggestion: UserVoiceSuggestion{
				ID:          "67890",
				Title:       "Add automation to Playbooks for incident response",
				Description: "DevOps and enterprise teams need automated playbook workflows for incident management",
				Status:      "open",
				Votes:       43,
				Comments:    8,
				Category:    "Playbooks",
			},
			expectedSegments:         []pm.CustomerSegment{pm.SegmentDevOps, pm.SegmentEnterprise},
			expectedCategories:       []pm.TechnicalCategory{pm.CategoryPlaybooks},
			expectedPriorityNotEmpty: true,
		},
		{
			name: "no_metadata",
			suggestion: UserVoiceSuggestion{
				ID:          "78901",
				Title:       "Update documentation",
				Description: "Please update the getting started guide with latest installation steps",
				Status:      "open",
				Votes:       3,
				Comments:    1,
				Category:    "Documentation",
			},
			expectedSegments:         []pm.CustomerSegment{},
			expectedCategories:       []pm.TechnicalCategory{},
			expectedPriorityNotEmpty: true, // Even low priority should be set
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := extractUserVoiceMetadata(tt.suggestion)
			pmMeta, ok := meta.RoleMetadata.(pm.Metadata)
			if !ok {
				t.Fatal("RoleMetadata should be PMMetadata")
			}

			// Check that expected segments are present (may detect more than expected)
			for _, expectedSeg := range tt.expectedSegments {
				found := false
				for _, seg := range pmMeta.Segments {
					if seg == expectedSeg {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected segment %s not found in %v", expectedSeg, pmMeta.Segments)
				}
			}

			// Check that expected categories are present (may detect more than expected)
			for _, expectedCat := range tt.expectedCategories {
				found := false
				for _, cat := range pmMeta.Categories {
					if cat == expectedCat {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected category %s not found in %v", expectedCat, pmMeta.Categories)
				}
			}

			// Check competitive context if specified
			if tt.expectedCompetitive != "" && pmMeta.Competitive != tt.expectedCompetitive {
				t.Errorf("Expected competitive %s, got %s", tt.expectedCompetitive, pmMeta.Competitive)
			}

			// Check priority
			if tt.expectedPriorityNotEmpty && pmMeta.Priority == "" {
				t.Errorf("Expected priority to not be empty, got empty")
			}
		})
	}
}

func TestMapUserVoiceStatusToPriority(t *testing.T) {
	tests := []struct {
		status   string
		expected pm.Priority
	}{
		{"completed", pm.PriorityCompleted},
		{"shipped", pm.PriorityCompleted},
		{"in_progress", pm.PriorityHigh},
		{"planned", pm.PriorityHigh},
		{"under_review", pm.PriorityMedium},
		{"open", pm.PriorityLow},
		{"declined", pm.PriorityLow},
		{"", pm.PriorityLow},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			result := mapUserVoiceStatusToPriority(tt.status)
			if result != tt.expected {
				t.Errorf("For status '%s', expected priority %s, got %s", tt.status, tt.expected, result)
			}
		})
	}
}

func TestMapUserVoiceVotesToPriority(t *testing.T) {
	tests := []struct {
		votes    int
		expected pm.Priority
	}{
		{150, pm.PriorityHigh},
		{100, pm.PriorityHigh},
		{99, pm.PriorityMedium},
		{50, pm.PriorityMedium},
		{20, pm.PriorityMedium},
		{19, pm.PriorityLow},
		{5, pm.PriorityLow},
		{0, pm.PriorityLow},
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.votes)), func(t *testing.T) {
			result := mapUserVoiceVotesToPriority(tt.votes)
			if result != tt.expected {
				t.Errorf("For %d votes, expected priority %s, got %s", tt.votes, tt.expected, result)
			}
		})
	}
}

func TestUserVoiceSuggestionToDoc(t *testing.T) {
	protocol := NewFileProtocol(nil)

	suggestion := UserVoiceSuggestion{
		ID:          "12345",
		Title:       "Federal mobile offline sync",
		Description: "High-security customers need mobile offline capability for classified operations",
		Status:      "planned",
		Votes:       156,
		Comments:    23,
		Category:    "Mobile",
		URL:         "https://mattermost.uservoice.com/forums/306457/suggestions/12345",
		CreatedAt:   "2024-03-15T10:30:00Z",
		UpdatedAt:   "2025-09-28T15:45:00Z",
	}

	doc := protocol.uservoiceSuggestionToDoc(suggestion, SourceFeatureRequests)

	// Check title
	if doc.Title != suggestion.Title {
		t.Errorf("Expected title '%s', got '%s'", suggestion.Title, doc.Title)
	}

	// Check source
	if doc.Source != SourceFeatureRequests {
		t.Errorf("Expected source '%s', got '%s'", SourceFeatureRequests, doc.Source)
	}

	// Check section
	if doc.Section != SectionFeatureRequests {
		t.Errorf("Expected section '%s', got '%s'", SectionFeatureRequests, doc.Section)
	}

	// Check URL
	if doc.URL != suggestion.URL {
		t.Errorf("Expected URL '%s', got '%s'", suggestion.URL, doc.URL)
	}

	// Check content contains inline metadata (format: (Priority: X | ...))
	if !contains(doc.Content, "Priority:") {
		t.Errorf("Expected content to contain inline metadata with priority")
	}

	// Check content contains key fields
	if !contains(doc.Content, suggestion.Title) {
		t.Errorf("Expected content to contain title")
	}
	if !contains(doc.Content, suggestion.Description) {
		t.Errorf("Expected content to contain description")
	}
	if !contains(doc.Content, "planned") {
		t.Errorf("Expected content to contain status")
	}
	if !contains(doc.Content, "156") {
		t.Errorf("Expected content to contain vote count")
	}

	// Check labels
	foundSegment := false
	foundCategory := false
	foundPriority := false
	for _, label := range doc.Labels {
		if label == "segment:federal" || label == "segment:enterprise" {
			foundSegment = true
		}
		if label == "category:mobile" {
			foundCategory = true
		}
		if label == "priority:high" {
			foundPriority = true
		}
	}

	if !foundSegment {
		t.Errorf("Expected to find segment label in %v", doc.Labels)
	}
	if !foundCategory {
		t.Errorf("Expected to find category:mobile label in %v", doc.Labels)
	}
	if !foundPriority {
		t.Errorf("Expected to find priority:high label in %v", doc.Labels)
	}
}

func TestMatchesUserVoiceSearch(t *testing.T) {
	protocol := NewFileProtocol(nil)

	tests := []struct {
		name       string
		suggestion UserVoiceSuggestion
		terms      []string
		expected   bool
	}{
		{
			name: "matches_title",
			suggestion: UserVoiceSuggestion{
				Title:       "Mobile offline sync feature",
				Description: "Add offline support",
			},
			terms:    []string{"mobile", "offline"},
			expected: true,
		},
		{
			name: "matches_description",
			suggestion: UserVoiceSuggestion{
				Title:       "Feature request",
				Description: "Add Kubernetes integration for DevOps teams",
			},
			terms:    []string{"kubernetes", "devops"},
			expected: true,
		},
		{
			name: "matches_category",
			suggestion: UserVoiceSuggestion{
				Title:       "Feature request",
				Description: "Improve performance",
				Category:    "Mobile",
			},
			terms:    []string{"mobile"},
			expected: true,
		},
		{
			name: "no_match",
			suggestion: UserVoiceSuggestion{
				Title:       "Documentation update",
				Description: "Update the README file",
			},
			terms:    []string{"mobile", "authentication"},
			expected: false,
		},
		{
			name: "empty_terms",
			suggestion: UserVoiceSuggestion{
				Title: "Any suggestion",
			},
			terms:    []string{},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := protocol.matchesUserVoiceSearch(tt.suggestion, tt.terms)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}
