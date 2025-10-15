// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"testing"
)

func TestFileProtocol_FeatureToDoc(t *testing.T) {
	protocol := NewFileProtocol(nil)

	feature := ProductBoardFeature{
		Name:             "Read Only Channels",
		Description:      "Ability to disable create post permission for specified channels",
		Type:             "subfeature",
		State:            "Delivered",
		Owner:            "John Doe",
		Parent:           "Channel Permissions",
		CustomerRequests: "Customer A, Customer B",
	}

	doc := protocol.featureToDoc(feature, SourceProductBoardFeatures)

	if doc.Title != feature.Name {
		t.Errorf("Expected title %s, got %s", feature.Name, doc.Title)
	}

	if doc.Source != SourceProductBoardFeatures {
		t.Errorf("Expected source %s, got %s", SourceProductBoardFeatures, doc.Source)
	}

	if doc.Author != feature.Owner {
		t.Errorf("Expected author %s, got %s", feature.Owner, doc.Author)
	}

	if doc.Section != SectionDelivered {
		t.Errorf("Expected section %s for delivered state, got %s", SectionDelivered, doc.Section)
	}

	if doc.LastModified != feature.State {
		t.Errorf("Expected last modified %s, got %s", feature.State, doc.LastModified)
	}

	if len(doc.Labels) < 2 {
		t.Errorf("Expected at least 2 labels, got %d", len(doc.Labels))
	}

	if !contains(doc.Labels[0], feature.Type) && !contains(doc.Labels[1], feature.Type) {
		t.Errorf("Expected labels to contain type %s, got %v", feature.Type, doc.Labels)
	}

	if !contains(doc.Content, "Parent: Channel Permissions") {
		t.Error("Expected content to include parent information")
	}

	if !contains(doc.Content, "Customer Requests: Customer A, Customer B") {
		t.Error("Expected content to include customer requests")
	}
}

func TestFileProtocol_FeatureToDoc_Sections(t *testing.T) {
	protocol := NewFileProtocol(nil)

	tests := []struct {
		name            string
		state           string
		expectedSection string
	}{
		{
			name:            "delivered",
			state:           "Delivered",
			expectedSection: SectionDelivered,
		},
		{
			name:            "idea",
			state:           "Idea",
			expectedSection: SectionIdeas,
		},
		{
			name:            "in progress",
			state:           "In Progress",
			expectedSection: SectionInProgress,
		},
		{
			name:            "progress variant",
			state:           "Work in Progress",
			expectedSection: SectionInProgress,
		},
		{
			name:            "planned maps to in-progress",
			state:           "Planned",
			expectedSection: SectionInProgress,
		},
		{
			name:            "in development maps to in-progress",
			state:           "In Development",
			expectedSection: SectionInProgress,
		},
		{
			name:            "default fallback",
			state:           "Under Consideration",
			expectedSection: SectionFeatures,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			feature := ProductBoardFeature{
				Name:  "Test Feature",
				State: tt.state,
			}

			doc := protocol.featureToDoc(feature, SourceProductBoardFeatures)

			if doc.Section != tt.expectedSection {
				t.Errorf("Expected section %s, got %s", tt.expectedSection, doc.Section)
			}
		})
	}
}

func TestFileProtocol_FeatureToDoc_MetadataExtraction(t *testing.T) {
	protocol := NewFileProtocol(nil)

	tests := []struct {
		name                   string
		feature                ProductBoardFeature
		expectedSegments       []string
		expectedCompetitive    string
		expectedPriority       string
		expectedCrossRefs      []string
		expectedLabelsContain  []string
		expectedContentContain []string
	}{
		{
			name: "federal customer segment detection",
			feature: ProductBoardFeature{
				Name:        "Federal DDIL Offline Sync",
				Description: "Tactical edge mobile deployment for classified operations",
				Type:        "feature",
				State:       "In Development",
				Parent:      "Mobile",
			},
			expectedSegments:      []string{"segment:federal"},
			expectedPriority:      "priority:high",
			expectedLabelsContain: []string{"segment:federal", "priority:high", "parent:Mobile", "category:mobile"},
			expectedContentContain: []string{
				"Segments:",
				"federal",
				"Categories:",
				"mobile",
				"Priority: high",
			},
		},
		{
			name: "competitive context with Slack",
			feature: ProductBoardFeature{
				Name:        "Support Slack Corporate Import",
				Description: "Slack Corporate exports with private channels support",
				Type:        "feature",
				State:       "Delivered",
			},
			expectedCompetitive:   "competitive:slack",
			expectedPriority:      "priority:completed",
			expectedLabelsContain: []string{"competitive:slack", "priority:completed"},
			expectedContentContain: []string{
				"Competitive: slack",
			},
		},
		{
			name: "enterprise with SSO compliance",
			feature: ProductBoardFeature{
				Name:        "Enterprise SSO SAML Integration",
				Description: "Enterprise-grade SAML SSO with audit logging",
				Type:        "feature",
				State:       "Idea",
				Parent:      "Security",
			},
			expectedSegments:      []string{"segment:enterprise"},
			expectedPriority:      "priority:medium",
			expectedLabelsContain: []string{"segment:enterprise", "priority:medium", "parent:Security", "category:authentication"},
			expectedContentContain: []string{
				"Segments:",
				"enterprise",
				"Categories:",
				"authentication",
			},
		},
		{
			name: "jira cross-reference extraction",
			feature: ProductBoardFeature{
				Name:        "Global Custom Themes",
				Description: "JIRA Epic: https://example.atlassian.net/browse/PROJ-12345",
				Type:        "feature",
				State:       "Needs Validation",
			},
			expectedCrossRefs:     []string{"jira:PROJ-12345"},
			expectedLabelsContain: []string{"jira:PROJ-12345"},
			expectedContentContain: []string{
				"Refs: PROJ-12345",
			},
		},
		{
			name: "high priority blocking deal",
			feature: ProductBoardFeature{
				Name:        "Advanced Permissions",
				Description: "Deal breaker for largest customer",
				Type:        "feature",
				State:       "Under Consideration",
			},
			expectedPriority:      "priority:high",
			expectedLabelsContain: []string{"priority:high"},
			expectedContentContain: []string{
				"Priority: high",
			},
		},
		{
			name: "multi-segment healthcare and enterprise",
			feature: ProductBoardFeature{
				Name:        "HIPAA Compliant Audit Logs",
				Description: "HIPAA compliant audit logging with PHI protection",
				Type:        "feature",
				State:       "Planned",
				Parent:      "Compliance",
			},
			expectedSegments: []string{"segment:healthcare", "segment:enterprise"},
			expectedLabelsContain: []string{
				"segment:healthcare",
				"segment:enterprise",
				"priority:high",
				"parent:Compliance",
				"category:compliance",
			},
			expectedContentContain: []string{
				"Segments:",
				"healthcare",
				"enterprise",
				"Categories:",
				"compliance",
			},
		},
		{
			name: "devops with kubernetes deployment",
			feature: ProductBoardFeature{
				Name:        "Kubernetes Operator",
				Description: "Kubernetes operator with CI/CD integration",
				Type:        "feature",
				State:       "Idea",
			},
			expectedSegments:      []string{"segment:devops"},
			expectedLabelsContain: []string{"segment:devops", "category:integrations"},
			expectedContentContain: []string{
				"Segments:",
				"devops",
				"Categories:",
			},
		},
		{
			name: "github issue cross-reference",
			feature: ProductBoardFeature{
				Name:        "Fix Mobile Crash",
				Description: "See GitHub issue #5432",
				Type:        "subfeature",
				State:       "In Development",
			},
			expectedCrossRefs:     []string{"github:#5432"},
			expectedLabelsContain: []string{"github:#5432", "priority:high", "category:mobile"},
			expectedContentContain: []string{
				"Refs: #5432",
				"Categories:",
				"mobile",
			},
		},
		{
			name: "jira plugin integration feature",
			feature: ProductBoardFeature{
				Name:        "Jira Plugin Two-Way Sync",
				Description: "Two-way synchronization for Jira plugin",
				Type:        "feature",
				State:       "Planned",
				Parent:      "Integrations",
			},
			expectedLabelsContain: []string{
				"category:plugins",
				"priority:high",
				"parent:Integrations",
			},
			expectedContentContain: []string{
				"Categories:",
				"plugins",
			},
		},
		{
			name: "playbooks incident response",
			feature: ProductBoardFeature{
				Name:        "Playbooks Automated Retrospectives",
				Description: "Automated retrospective creation for playbooks",
				Type:        "feature",
				State:       "Idea",
				Parent:      "Playbooks",
			},
			expectedLabelsContain: []string{
				"category:playbooks",
				"priority:medium",
				"parent:Playbooks",
			},
			expectedContentContain: []string{
				"Categories:",
				"playbooks",
			},
		},
		{
			name: "boards kanban improvements",
			feature: ProductBoardFeature{
				Name:        "Boards Card Dependencies",
				Description: "Card dependencies for project tracking",
				Type:        "feature",
				State:       "Under Consideration",
				Parent:      "Boards",
			},
			expectedLabelsContain: []string{
				"category:boards",
				"priority:medium",
				"parent:Boards",
			},
			expectedContentContain: []string{
				"Categories:",
				"boards",
			},
		},
		{
			name: "calls plugin voice quality",
			feature: ProductBoardFeature{
				Name:        "Calls Plugin Echo Cancellation",
				Description: "Improved echo cancellation for calls",
				Type:        "feature",
				State:       "In Development",
				Parent:      "Calls",
			},
			expectedLabelsContain: []string{
				"category:calls",
				"category:plugins",
				"priority:high",
				"parent:Calls",
			},
			expectedContentContain: []string{
				"Categories:",
				"calls",
				"plugins",
			},
		},
		{
			name: "database performance optimization",
			feature: ProductBoardFeature{
				Name:        "PostgreSQL Query Optimization",
				Description: "Database query optimization for performance",
				Type:        "feature",
				State:       "Planned",
			},
			expectedLabelsContain: []string{
				"category:database",
				"category:performance",
				"priority:high",
			},
			expectedContentContain: []string{
				"Categories:",
				"database",
				"performance",
			},
		},
		{
			name: "multi-category channel notification feature",
			feature: ProductBoardFeature{
				Name:        "Smart Channel Notifications",
				Description: "Smart notification defaults for channels",
				Type:        "feature",
				State:       "Idea",
			},
			expectedLabelsContain: []string{
				"category:channels",
				"priority:medium",
			},
			expectedContentContain: []string{
				"Categories:",
				"channels",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := protocol.featureToDoc(tt.feature, SourceProductBoardFeatures)

			for _, expectedLabel := range tt.expectedLabelsContain {
				found := false
				for _, label := range doc.Labels {
					if label == expectedLabel {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected label %s not found in labels: %v", expectedLabel, doc.Labels)
				}
			}

			for _, expectedContent := range tt.expectedContentContain {
				if !contains(doc.Content, expectedContent) {
					t.Errorf("Expected content to contain '%s', got: %s", expectedContent, doc.Content)
				}
			}

			if doc.Title != tt.feature.Name {
				t.Errorf("Expected title %s, got %s", tt.feature.Name, doc.Title)
			}

			if doc.Source != SourceProductBoardFeatures {
				t.Errorf("Expected source %s, got %s", SourceProductBoardFeatures, doc.Source)
			}
		})
	}
}

func TestFileProtocol_FeatureToDoc_NoMetadata(t *testing.T) {
	protocol := NewFileProtocol(nil)

	feature := ProductBoardFeature{
		Name:        "Simple Feature",
		Description: "A basic feature description without keywords",
		Type:        "feature",
		State:       "Idea",
	}

	doc := protocol.featureToDoc(feature, SourceProductBoardFeatures)

	if len(doc.Labels) < 3 {
		t.Errorf("Expected at least 3 labels, got %d: %v", len(doc.Labels), doc.Labels)
	}

	hasDefaultPriority := false
	for _, label := range doc.Labels {
		if label == "priority:medium" {
			hasDefaultPriority = true
			break
		}
	}
	if !hasDefaultPriority {
		t.Errorf("Expected priority:medium label for feature with no priority signals, got labels: %v", doc.Labels)
	}

	if contains(doc.Content, "Extracted Metadata:") {
		t.Error("Expected no metadata section for feature without extractable metadata")
	}
}
