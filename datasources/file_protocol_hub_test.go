// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"strings"
	"testing"
)

func TestFileProtocol_ParseHubPosts(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expectedLen int
		checkFirst  func(*testing.T, MattermostHubPost)
	}{
		{
			name: "federal_customer_segment",
			content: `
Contact Sales Bot
BOT
5:11 PM
New Contact Sales Request
First Name: Alex
Last Name: Smith
Company: Research Institute Alpha
Email: alex.smith@example.org
Phone: (555) 010-1234
Owner: Chris Johnson
Company Type: #Federal
Notes: I am looking for a quote on the minimum amount of extra seats we can add to a pre-existing enterprise instance of Mattermost and how much that would cost us.
Link to SFDC: https://example.lightning.force.com/lightning/r/Contact/CONTACT_ID/view
`,
			expectedLen: 1,
			checkFirst: func(t *testing.T, post MattermostHubPost) {
				if post.FirstName != "Alex" {
					t.Errorf("Expected FirstName 'Alex', got '%s'", post.FirstName)
				}
				if post.Company != "Research Institute Alpha" {
					t.Errorf("Expected Company 'Research Institute Alpha', got '%s'", post.Company)
				}
				if !containsString(post.Tags, "Federal") {
					t.Errorf("Expected tags to contain 'Federal', got %v", post.Tags)
				}
			},
		},
		{
			name: "smb_with_slack_replacement",
			content: `
Contact Sales Bot
BOT
2:29 AM
New Contact Sales Request
First Name: Taylor
Last Name: Brown
Company: Health Solutions Inc
Email: contact@example.com
Phone: (555) 020-5678
Owner: Jordan Lee
Company Type: #SMB
Notes: Would like to migrate from Slack for company chat platform, approx 150 seats and growing
Link to SFDC: https://example.lightning.force.com/lightning/r/Lead/LEAD_ID/view
`,
			expectedLen: 1,
			checkFirst: func(t *testing.T, post MattermostHubPost) {
				notesLower := strings.ToLower(post.Notes)
				if !contains(notesLower, "slack") {
					t.Errorf("Expected Notes to mention 'slack', got '%s'", post.Notes)
				}
				if !containsString(post.Tags, "SMB") {
					t.Errorf("Expected tags to contain 'SMB', got %v", post.Tags)
				}
			},
		},
		{
			name: "enterprise_with_licenses",
			content: `
Quote Request Bot
BOT
11:07 AM
New Quote Request
First Name: Casey
Last Name: Chen
Company: Enterprise Solutions Group
Email: contact@example.com
Phone:
Owner: Jordan Lee
Company Type: #MME
Message: I'm looking for a quote for 100 and 250 users. We would likely start with 100 users but for budgetary reasons want to have growth to 250 forecast.
Clarifying Info:
Number of licenses: 101+,
Team using MM: Multiple departments,
Tech replacement: slack
Link to SFDC: https://example.lightning.force.com/lightning/r/Lead/LEAD_ID/view
`,
			expectedLen: 1,
			checkFirst: func(t *testing.T, post MattermostHubPost) {
				if post.Company != "Enterprise Solutions Group" {
					t.Errorf("Expected Company 'Enterprise Solutions Group', got '%s'", post.Company)
				}
				licenses := strings.TrimSuffix(post.Licenses, ",")
				if licenses != "101+" {
					t.Errorf("Expected Licenses '101+', got '%s'", post.Licenses)
				}
				if post.Replacement != "slack" {
					t.Errorf("Expected Replacement 'slack', got '%s'", post.Replacement)
				}
			},
		},
	}

	protocol := NewFileProtocol(nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			posts := protocol.parseHubPosts(tt.content)

			if len(posts) != tt.expectedLen {
				t.Errorf("Expected %d posts, got %d", tt.expectedLen, len(posts))
			}

			if len(posts) > 0 && tt.checkFirst != nil {
				tt.checkFirst(t, posts[0])
			}
		})
	}
}

func TestFileProtocol_HubPostToDoc_MetadataExtraction(t *testing.T) {
	tests := []struct {
		name                   string
		post                   MattermostHubPost
		sourceName             string
		expectedLabelsContain  []string
		expectedContentContain []string
	}{
		{
			name: "federal_customer_high_priority",
			post: MattermostHubPost{
				FirstName:   "Alex",
				LastName:    "Smith",
				Company:     "Research Institute Alpha",
				Email:       "alex.smith@example.org",
				CompanyType: "#Federal",
				Tags:        []string{"Federal"},
				Message:     "URGENT: Critical requirement - Looking for enterprise licensing quote for classified environment with CMMC compliance - this is blocking our deal",
			},
			sourceName:             SourceMattermostHub,
			expectedLabelsContain:  []string{"segment:federal", "priority:high"},
			expectedContentContain: []string{"Segments:", "federal", "Priority:", "high"},
		},
		{
			name: "smb_slack_replacement",
			post: MattermostHubPost{
				FirstName:   "Taylor",
				LastName:    "Brown",
				Company:     "Health Solutions Inc",
				CompanyType: "#SMB",
				Tags:        []string{"SMB"},
				Replacement: "slack",
				Message:     "Would like to migrate from Slack for company chat platform",
			},
			sourceName:             SourceMattermostHub,
			expectedLabelsContain:  []string{"segment:smb", "competitive:slack"},
			expectedContentContain: []string{"Segments:", "smb", "Competitive Context:", "slack"},
		},
		{
			name: "enterprise_with_mobile_inquiry",
			post: MattermostHubPost{
				Company:     "Enterprise Solutions Group",
				CompanyType: "#MME",
				Tags:        []string{"MME"},
				Licenses:    "101+",
				Team:        "Multiple departments",
				Message:     "Need mobile app deployment for iOS and Android across enterprise",
			},
			sourceName:             SourceMattermostHub,
			expectedLabelsContain:  []string{"segment:enterprise", "category:mobile"},
			expectedContentContain: []string{"Segments:", "enterprise", "Categories:", "mobile"},
		},
		{
			name: "healthcare_hipaa_compliance",
			post: MattermostHubPost{
				Company:     "Medical Center",
				CompanyType: "#Healthcare",
				Tags:        []string{"Healthcare"},
				Message:     "Hospital needs HIPAA compliance and audit capabilities for patient care coordination with data retention",
			},
			sourceName:             SourceMattermostHub,
			expectedLabelsContain:  []string{"segment:healthcare", "category:compliance"},
			expectedContentContain: []string{"Segments:", "healthcare", "Categories:", "compliance"},
		},
		{
			name: "devops_kubernetes_integration",
			post: MattermostHubPost{
				Company:     "DevOps Tech Corp",
				CompanyType: "#Enterprise",
				Tags:        []string{"Enterprise"},
				Message:     "DevOps team needs Kubernetes integration for CI/CD pipeline and infrastructure monitoring",
			},
			sourceName:             SourceMattermostHub,
			expectedLabelsContain:  []string{"segment:devops", "category:integrations"},
			expectedContentContain: []string{"Segments:", "devops", "Categories:"},
		},
		{
			name: "no_metadata",
			post: MattermostHubPost{
				Company:     "Generic Company",
				CompanyType: "",
				Message:     "Simple inquiry about pricing",
			},
			sourceName:             SourceMattermostHub,
			expectedLabelsContain:  []string{},
			expectedContentContain: []string{"Company:", "Generic Company"},
		},
	}

	protocol := NewFileProtocol(nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := protocol.hubPostToDoc(tt.post, tt.sourceName)

			for _, expectedLabel := range tt.expectedLabelsContain {
				if !containsString(doc.Labels, expectedLabel) {
					t.Errorf("Expected label '%s' not found in labels: %v", expectedLabel, doc.Labels)
				}
			}

			for _, expectedContent := range tt.expectedContentContain {
				if !contains(doc.Content, expectedContent) {
					t.Errorf("Expected content substring '%s' not found in content:\n%s", expectedContent, doc.Content)
				}
			}

			if doc.Source != tt.sourceName {
				t.Errorf("Expected source '%s', got '%s'", tt.sourceName, doc.Source)
			}
		})
	}
}

func containsString(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}
