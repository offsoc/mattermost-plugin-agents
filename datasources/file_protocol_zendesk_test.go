// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileProtocol_ParseTextTickets(t *testing.T) {
	protocol := NewFileProtocol(nil)

	testContent := `Zendesk
BOT
9:25 PM
#zd48572: No Autoscrolling During Active Conversation
Updated by Alice Smith on 2025-08-14T19:25:08Z
Public: true
Ticket owner:
Acme Defense Systems - Alice Smith - alice.smith@example.com

 Tags:100k_customer amer_schedule americas

Alice Smith, Aug 14, 2025, 3:25 PM

Over the last couple weeks Mattermost chats sporadically stop autoscrolling mid-conversation.

#zd48573: Mobile Push Notification Issue
Updated by Test User on 2025-08-15T10:00:00Z
Public: true
Ticket owner:
Test Company - Test User - test.user@example.com

 Tags:bug mobile production

Test User, Aug 15, 2025, 10:00 AM

Mobile push notifications are not working for iOS devices.`

	tickets := protocol.parseTextTickets(testContent)

	if len(tickets) != 2 {
		t.Errorf("Expected 2 tickets, got %d", len(tickets))
	}

	if tickets[0].ID != "#zd48572" {
		t.Errorf("Expected ticket ID #zd48572, got %s", tickets[0].ID)
	}

	if tickets[0].Title != "No Autoscrolling During Active Conversation" {
		t.Errorf("Expected title 'No Autoscrolling During Active Conversation', got %s", tickets[0].Title)
	}

	if !contains(tickets[0].Customer, "Acme Defense Systems") {
		t.Errorf("Expected customer to contain 'Acme Defense Systems', got %s", tickets[0].Customer)
	}

	if len(tickets[0].Tags) != 3 {
		t.Errorf("Expected 3 tags, got %d", len(tickets[0].Tags))
	}

	if !tickets[0].Public {
		t.Error("Expected ticket to be public")
	}

	if tickets[1].ID != "#zd48573" {
		t.Errorf("Expected second ticket ID #zd48573, got %s", tickets[1].ID)
	}
}

func TestFileProtocol_MatchesTicketSearch(t *testing.T) {
	protocol := NewFileProtocol(nil)

	ticket := ZendeskTicket{
		ID:       "#zd48572",
		Title:    "No Autoscrolling During Active Conversation",
		Content:  "Mattermost chats stop autoscrolling",
		Customer: "Acme Defense Systems",
		Tags:     []string{"100k_customer", "bug", "production"},
	}

	tests := []struct {
		name        string
		searchTerms []string
		expected    bool
	}{
		{
			name:        "matching terms",
			searchTerms: []string{"autoscrolling", "conversation"},
			expected:    true,
		},
		{
			name:        "partial match",
			searchTerms: []string{"autoscrolling", "notification"},
			expected:    true,
		},
		{
			name:        "tag match",
			searchTerms: []string{"bug", "production"},
			expected:    true,
		},
		{
			name:        "no match",
			searchTerms: []string{"calendar", "integration"},
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := protocol.matchesTicketSearch(ticket, tt.searchTerms)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestFileProtocol_TicketToDoc(t *testing.T) {
	protocol := NewFileProtocol(nil)

	ticket := ZendeskTicket{
		ID:        "#zd48572",
		Title:     "No Autoscrolling Issue",
		Content:   "The scrollbar stops following the conversation",
		Customer:  "Acme Defense Systems",
		UpdatedBy: "Alice Smith",
		Tags:      []string{"bug", "production"},
	}

	doc := protocol.ticketToDoc(ticket, SourceZendeskTickets)

	if doc.Title != ticket.Title {
		t.Errorf("Expected title %s, got %s", ticket.Title, doc.Title)
	}

	if doc.Source != SourceZendeskTickets {
		t.Errorf("Expected source %s, got %s", SourceZendeskTickets, doc.Source)
	}

	if !contains(doc.Content, "Customer: Acme Defense Systems") {
		t.Error("Expected content to include customer information")
	}

	if !contains(doc.URL, "zd48572") {
		t.Errorf("Expected URL to contain ticket ID, got %s", doc.URL)
	}

	if len(doc.Labels) < 2 {
		t.Errorf("Expected at least 2 labels, got %d", len(doc.Labels))
	}
}

func TestFileProtocol_TicketToDoc_MetadataExtraction(t *testing.T) {
	protocol := NewFileProtocol(nil)

	tests := []struct {
		name                   string
		ticket                 ZendeskTicket
		expectedLabelsContain  []string
		expectedContentContain []string
		expectedSection        string
	}{
		{
			name: "federal customer with high priority and production",
			ticket: ZendeskTicket{
				ID:        "#zd48592",
				Title:     "Jira plugin not working after Jira 10 update",
				Content:   "Jira plugin not working after update",
				Customer:  "Research Institute Alpha - Bob Johnson - bob.johnson@example.org",
				UpdatedBy: "Carol Manager",
				Tags: []string{
					"100k_customer", "tier_1", "customer_confirmed_priority",
					"production", "premier", "technical_support__plugins__jira",
					"us_only", "amer_schedule",
				},
			},
			expectedLabelsContain: []string{
				"segment:federal",
				"segment:enterprise",
				"priority:high",
				"env:production",
				"region:americas",
				"category:plugins",
				"tier_1",
			},
			expectedContentContain: []string{
				"Customer: Research Institute Alpha",
				"Segments:",
				"federal",
				"enterprise",
				"Tier: tier_1",
				"Environment: production",
				"Categories:",
				"plugins",
			},
			expectedSection: SectionCritical,
		},
		{
			name: "finance customer mobile issue",
			ticket: ZendeskTicket{
				ID:        "#zd49046",
				Title:     "Mattermost Mobile App SAML Login is not redirecting",
				Content:   "SAML login not redirecting properly",
				Customer:  "Global Financial Bank - David Lee - david.lee@example.com",
				UpdatedBy: "David Lee",
				Tags: []string{
					"100k_customer", "tier_1", "enterprise",
					"production", "premier",
				},
			},
			expectedLabelsContain: []string{
				"segment:finance",
				"segment:enterprise",
				"priority:high",
				"env:production",
				"category:mobile",
				"category:authentication",
			},
			expectedContentContain: []string{
				"Customer: Global Financial Bank",
				"Segments:",
				"enterprise",
				"finance",
				"Categories:",
				"mobile",
				"authentication",
			},
			expectedSection: SectionCritical,
		},
		{
			name: "performance issue with database migration",
			ticket: ZendeskTicket{
				ID:        "#zd48650",
				Title:     "Inquiry Regarding PostgreSQL Migration Process",
				Content:   "PostgreSQL migration configuration issue",
				Customer:  "Financial Research Institute - Support Team - support-request@example.com",
				UpdatedBy: "Support Team",
				Tags: []string{
					"100k_customer", "tier_1", "enterprise", "mysql",
					"staging/uat/qa", "apac_schedule",
				},
			},
			expectedLabelsContain: []string{
				"segment:enterprise",
				"segment:finance",
				"priority:high",
				"env:staging",
				"region:asia-pacific",
				"category:database",
			},
			expectedContentContain: []string{
				"Customer: Financial Research Institute",
				"Segments:",
				"enterprise",
				"finance",
				"Categories:",
				"database",
			},
			expectedSection: SectionGeneral,
		},
		{
			name: "devops customer with tier 2",
			ticket: ZendeskTicket{
				ID:        "#zd48664",
				Title:     "Need help with Mattermost plugin Autolink",
				Content:   "Need Autolink plugin configuration help",
				Customer:  "DevOps Automotive Inc - Emily Chen - emily.chen@example.com",
				UpdatedBy: "Emily Chen",
				Tags: []string{
					"tier_1", "enterprise", "production",
					"amer_schedule",
				},
			},
			expectedLabelsContain: []string{
				"segment:devops",
				"segment:enterprise",
				"priority:high",
				"env:production",
				"category:plugins",
			},
			expectedContentContain: []string{
				"Customer: DevOps Automotive Inc",
				"Segments:",
				"enterprise",
				"devops",
				"Categories:",
				"plugins",
			},
			expectedSection: SectionCritical,
		},
		{
			name: "channel scrolling issue",
			ticket: ZendeskTicket{
				ID:        "#zd48582",
				Title:     "Issue with scroll position when opening a thread",
				Content:   "Thread scroll position issue",
				Customer:  "Financial Research Institute - Support Team",
				UpdatedBy: "Support Team",
				Tags: []string{
					"100k_customer", "enterprise", "mysql",
					"staging/uat/qa", "apac_schedule",
				},
			},
			expectedLabelsContain: []string{
				"segment:enterprise",
				"segment:finance",
				"priority:medium",
				"env:staging",
				"category:channels",
			},
			expectedContentContain: []string{
				"Categories:",
				"channels",
			},
			expectedSection: SectionGeneral,
		},
		{
			name: "playbooks bug",
			ticket: ZendeskTicket{
				ID:        "#zd48868",
				Title:     "Issue with Plugin",
				Content:   "Error when deleting incidents",
				Customer:  "Global Motors Ltd - Frank Williams",
				UpdatedBy: "Frank Williams",
				Tags: []string{
					"100k_customer", "tier_1", "premier", "production",
					"customer_confirmed_priority",
				},
			},
			expectedLabelsContain: []string{
				"segment:enterprise",
				"priority:high",
				"env:production",
				"category:playbooks",
			},
			expectedContentContain: []string{
				"Categories:",
				"playbooks",
			},
			expectedSection: SectionCritical,
		},
		{
			name: "mobile app crash with high engagement",
			ticket: ZendeskTicket{
				ID:        "#zd49123",
				Title:     "Mobile app crashes on Android 14 after latest update",
				Content:   "Mobile app crashes on Android 14",
				Customer:  "Field Operations Company - George Martinez",
				UpdatedBy: "George Martinez",
				Tags: []string{
					"100k_customer", "tier_1", "production",
					"customer_confirmed_priority", "mobile", "android",
				},
			},
			expectedLabelsContain: []string{
				"segment:enterprise",
				"priority:high",
				"env:production",
				"category:mobile",
				"category:performance",
			},
			expectedContentContain: []string{
				"Categories:",
				"mobile",
				"performance",
			},
			expectedSection: SectionCritical,
		},
		{
			name: "competitive mention of Slack in feature request",
			ticket: ZendeskTicket{
				ID:        "#zd49234",
				Title:     "Customer wants Slack-like threading and notifications",
				Content:   "Request for Slack-like threading",
				Customer:  "Enterprise Solutions Corp",
				UpdatedBy: "Helen Anderson",
				Tags: []string{
					"100k_customer", "enterprise", "feature_request",
				},
			},
			expectedLabelsContain: []string{
				"segment:enterprise",
				"priority:medium",
				"competitive:slack",
				"category:channels",
			},
			expectedContentContain: []string{
				"Competitive: slack",
				"Categories:",
				"channels",
			},
			expectedSection: SectionGeneral,
		},
		{
			name: "boards card dependencies issue",
			ticket: ZendeskTicket{
				ID:        "#zd49345",
				Title:     "Boards: Card dependencies not showing correctly",
				Content:   "Card dependencies not displaying",
				Customer:  "Innovation Tech Inc",
				UpdatedBy: "Ivan Rodriguez",
				Tags: []string{
					"professional", "production", "boards",
				},
			},
			expectedLabelsContain: []string{
				"segment:enterprise",
				"priority:medium",
				"env:production",
				"category:boards",
			},
			expectedContentContain: []string{
				"Categories:",
				"boards",
			},
			expectedSection: SectionProduction,
		},
		{
			name: "calls plugin echo cancellation",
			ticket: ZendeskTicket{
				ID:        "#zd49456",
				Title:     "Calls plugin: Echo cancellation not working in large meetings",
				Content:   "Echo cancellation not working",
				Customer:  "Worldwide Enterprise Group",
				UpdatedBy: "Jane Wilson",
				Tags: []string{
					"100k_customer", "tier_1", "premier", "production",
					"customer_confirmed_priority",
				},
			},
			expectedLabelsContain: []string{
				"segment:enterprise",
				"priority:high",
				"env:production",
				"category:calls",
				"category:plugins",
			},
			expectedContentContain: []string{
				"Categories:",
				"calls",
				"plugins",
			},
			expectedSection: SectionCritical,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := protocol.ticketToDoc(tt.ticket, SourceZendeskTickets)

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

			if doc.Section != tt.expectedSection {
				t.Errorf("Expected section %s, got %s", tt.expectedSection, doc.Section)
			}
		})
	}
}

func TestFileProtocol_TicketToDoc_NoMetadata(t *testing.T) {
	protocol := NewFileProtocol(nil)

	ticket := ZendeskTicket{
		ID:        "#zd12345",
		Title:     "Simple Question",
		Content:   "How do I configure basic settings?",
		Customer:  "Small Company - User",
		UpdatedBy: "User",
		Tags:      []string{},
	}

	doc := protocol.ticketToDoc(ticket, SourceZendeskTickets)

	if len(doc.Labels) < 1 {
		t.Errorf("Expected at least 1 label (priority), got %d: %v", len(doc.Labels), doc.Labels)
	}

	hasDefaultPriority := false
	for _, label := range doc.Labels {
		if label == "priority:medium" {
			hasDefaultPriority = true
			break
		}
	}
	if !hasDefaultPriority {
		t.Errorf("Expected priority:medium label for ticket with no priority signals, got labels: %v", doc.Labels)
	}

	if doc.Section != SectionGeneral {
		t.Errorf("Expected section %s, got %s", SectionGeneral, doc.Section)
	}
}

func TestFileProtocol_Fetch_TextFile(t *testing.T) {
	protocol := NewFileProtocol(nil)

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test-tickets.txt")

	testContent := `#zd001: Mobile Autoscroll Bug
Updated by Test User on 2025-08-14T19:25:08Z
Public: true
Ticket owner:
Test Company - Test User - test@example.com

 Tags:bug mobile

Mobile app autoscroll issue.

#zd002: Performance Issue
Updated by Another User on 2025-08-15T10:00:00Z
Public: true
Ticket owner:
Company B - Another User - user@example.com

 Tags:performance production

Performance degradation in channels.`

	if err := os.WriteFile(testFile, []byte(testContent), 0600); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	request := ProtocolRequest{
		Source: SourceConfig{
			Name: SourceZendeskTickets,
			Endpoints: map[string]string{
				EndpointFilePath: testFile,
			},
		},
		Topic: "mobile",
		Limit: 10,
	}

	docs, err := protocol.Fetch(context.Background(), request)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(docs) == 0 {
		t.Error("Expected at least one matching document")
	}

	for _, doc := range docs {
		if doc.Title == "" {
			t.Error("Document title should not be empty")
		}
		if doc.Source != SourceZendeskTickets {
			t.Errorf("Expected source %s, got %s", SourceZendeskTickets, doc.Source)
		}
	}
}

func TestFileProtocol_Fetch_ZendeskIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := CreateDefaultConfig()
	config.EnableSource(SourceZendeskTickets)

	client := NewClient(config, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	docs, err := client.FetchFromSource(ctx, SourceZendeskTickets, "mobile", 5)
	if err != nil {
		if contains(err.Error(), "no such file") {
			t.Skip("Skipping integration test - zendesk_tickets.txt not found")
		}
		t.Fatalf("Failed to fetch docs: %v", err)
	}

	if len(docs) == 0 {
		t.Log("Note: No documents matched search criteria")
	}

	for _, doc := range docs {
		if doc.Title == "" {
			t.Error("Document title should not be empty")
		}
		if doc.Source != SourceZendeskTickets {
			t.Errorf("Expected source %s, got %s", SourceZendeskTickets, doc.Source)
		}
	}
}
