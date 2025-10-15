// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package pm

import (
	"fmt"
	"strings"
)

// CustomerSegment represents a customer segment type
type CustomerSegment string

const (
	SegmentFederal    CustomerSegment = "federal"
	SegmentEnterprise CustomerSegment = "enterprise"
	SegmentHealthcare CustomerSegment = "healthcare"
	SegmentFinance    CustomerSegment = "finance"
	SegmentDevOps     CustomerSegment = "devops"
	SegmentSMB        CustomerSegment = "smb"
)

// Competitor represents a competitor product
type Competitor string

const (
	CompetitorSlack        Competitor = "slack"
	CompetitorTeams        Competitor = "teams"
	CompetitorDiscord      Competitor = "discord"
	CompetitorRocketChat   Competitor = "rocket_chat"
	CompetitorZoom         Competitor = "zoom"
	CompetitorWebex        Competitor = "webex"
	CompetitorMicrosoftAny Competitor = "microsoft"
)

// Priority represents inferred feature priority
type Priority string

const (
	PriorityHigh      Priority = "high"
	PriorityMedium    Priority = "medium"
	PriorityLow       Priority = "low"
	PriorityCompleted Priority = "completed"
)

// TechnicalCategory represents a technical area or feature category
type TechnicalCategory string

const (
	CategoryPlugins        TechnicalCategory = "plugins"
	CategoryMobile         TechnicalCategory = "mobile"
	CategoryAuthentication TechnicalCategory = "authentication"
	CategoryPerformance    TechnicalCategory = "performance"
	CategoryDatabase       TechnicalCategory = "database"
	CategoryCompliance     TechnicalCategory = "compliance"
	CategoryChannels       TechnicalCategory = "channels"
	CategoryIntegrations   TechnicalCategory = "integrations"
	CategoryPlaybooks      TechnicalCategory = "playbooks"
	CategoryBoards         TechnicalCategory = "boards"
	CategoryCalls          TechnicalCategory = "calls"
)

// CustomerSegmentKeywords maps segment types to their keyword indicators
var CustomerSegmentKeywords = map[CustomerSegment][]string{
	SegmentFederal: {
		"federal", "dod", "department of defense", "government", "fedramp", "fisma",
		"ddil", "tactical", "military", "classified", "cui", "il4", "il5",
		"army", "navy", "air force", "marines", "defense", "cmmc",
		"clearance", "battlefield", "combat", "mission critical",
	},
	SegmentEnterprise: {
		"enterprise", "e20", "f500", "fortune 500", "large organization",
		"sso", "saml", "ldap", "compliance", "audit", "governance",
		"10000+ users", "high availability", "clustering", "scale",
		"ha", "disaster recovery", "multi-region",
	},
	SegmentHealthcare: {
		"healthcare", "health", "hospital", "medical", "hipaa", "patient",
		"clinical", "phi", "protected health information", "healthcare provider",
	},
	SegmentFinance: {
		"financial", "bank", "fintech", "trading", "sox", "pci", "payment",
		"insurance", "financial services", "broker", "investment",
	},
	SegmentDevOps: {
		"devops", "sre", "site reliability", "ci/cd", "kubernetes", "k8s",
		"docker", "automation", "infrastructure", "monitoring", "alerts",
		"incident response", "chatops", "devsecops",
	},
	SegmentSMB: {
		"smb", "small business", "small-medium business", "small and medium",
		"startup", "small team", "small organization", "under 100 users",
		"50 users", "100 users", "small company",
	},
}

// CompetitorKeywords maps competitor names to their keyword patterns
var CompetitorKeywords = map[Competitor][]string{
	CompetitorSlack: {
		"slack", "slack.com", "slackbot",
	},
	CompetitorTeams: {
		"teams", "microsoft teams", "ms teams",
	},
	CompetitorDiscord: {
		"discord",
	},
	CompetitorRocketChat: {
		"rocket.chat", "rocket chat", "rocketchat",
	},
	CompetitorZoom: {
		"zoom", "zoom meetings",
	},
	CompetitorWebex: {
		"webex", "cisco webex",
	},
	CompetitorMicrosoftAny: {
		"microsoft", "office 365", "o365",
	},
}

// TechnicalCategoryKeywords maps technical categories to their keyword indicators
var TechnicalCategoryKeywords = map[TechnicalCategory][]string{
	CategoryPlugins: {
		"plugin", "jira plugin", "github plugin", "gitlab plugin",
		"autolink", "webhook", "integration", "custom plugin",
	},
	CategoryMobile: {
		"mobile app", "ios", "android", "mobile", "smartphone",
		"tablet", "mobile version", "mobile client",
	},
	CategoryAuthentication: {
		"ldap", "saml", "sso", "ad/ldap", "authentication",
		"login", "sign in", "sign-in", "keycloak", "oauth",
		"openid", "gitlab auth", "mfa", "2fa", "active directory",
	},
	CategoryPerformance: {
		"slow", "performance", "timeout", "hanging", "freeze",
		"latency", "unresponsive", "lag", "crash", "outage",
		"memory", "cpu", "load", "bottleneck",
	},
	CategoryDatabase: {
		"mysql", "postgres", "postgresql", "database", "db",
		"migration", "schema", "query", "sql",
	},
	CategoryCompliance: {
		"compliance", "audit", "retention", "export",
		"message export", "gdpr", "data retention",
		"e-discovery", "ediscovery", "legal hold",
	},
	CategoryChannels: {
		"channel", "thread", "message", "post", "reply",
		"scroll", "autoscroll", "sidebar", "notification",
		"mention", "dm", "direct message", "group message",
	},
	CategoryIntegrations: {
		"integration", "webhook", "incoming webhook", "outgoing webhook",
		"slash command", "bot", "api", "rest api",
	},
	CategoryPlaybooks: {
		"playbook", "run", "checklist", "workflow",
		"incident", "retrospective",
	},
	CategoryBoards: {
		"board", "card", "focalboard", "kanban",
		"project board", "task board",
	},
	CategoryCalls: {
		"calls", "call", "voice", "video call", "screen share",
		"webrtc", "calls plugin",
	},
}

// PrioritySignals contains keyword patterns for priority detection
var PrioritySignals = struct {
	High []string
	Low  []string
}{
	High: []string{
		"blocking deal", "must-have", "critical", "urgent", "asap",
		"blocker", "deal breaker", "required for", "cannot proceed without",
		"p0", "p1", "high priority", "showstopper",
	},
	Low: []string{
		"nice-to-have", "low priority", "future", "someday", "eventually",
		"nice to have", "would be nice", "p3", "p4", "backlog",
	},
}

// ExtractCustomerSegments extracts customer segment indicators from text
func ExtractCustomerSegments(text ...string) []CustomerSegment {
	searchText := strings.ToLower(strings.Join(text, " "))
	segmentMap := make(map[CustomerSegment]bool)

	for segment, keywords := range CustomerSegmentKeywords {
		for _, keyword := range keywords {
			if strings.Contains(searchText, keyword) {
				segmentMap[segment] = true
				break
			}
		}
	}

	segments := make([]CustomerSegment, 0, len(segmentMap))
	for segment := range segmentMap {
		segments = append(segments, segment)
	}

	return segments
}

// ExtractCompetitiveContext detects competitive feature context
func ExtractCompetitiveContext(text ...string) Competitor {
	searchText := strings.ToLower(strings.Join(text, " "))

	competitorOrder := []Competitor{
		CompetitorSlack,
		CompetitorDiscord,
		CompetitorRocketChat,
		CompetitorZoom,
		CompetitorWebex,
		CompetitorMicrosoftAny,
		CompetitorTeams,
	}

	for _, competitor := range competitorOrder {
		keywords := CompetitorKeywords[competitor]
		for _, keyword := range keywords {
			if strings.Contains(searchText, keyword) {
				return competitor
			}
		}
	}

	return ""
}

// EstimatePriority infers priority from text signals and state
func EstimatePriority(text string, state string) Priority {
	searchText := strings.ToLower(text)

	// Check high priority signals
	for _, signal := range PrioritySignals.High {
		if strings.Contains(searchText, signal) {
			return PriorityHigh
		}
	}

	// Check low priority signals
	for _, signal := range PrioritySignals.Low {
		if strings.Contains(searchText, signal) {
			return PriorityLow
		}
	}

	// State-based priority inference
	switch state {
	case "Delivered":
		return PriorityCompleted
	case "In Development", "Planned":
		return PriorityHigh
	case "Unlikely to Implement", "Deprecated", "Candidate for Deprecation":
		return PriorityLow
	default:
		return PriorityMedium
	}
}

// ExtractTechnicalCategories extracts technical category indicators from text
func ExtractTechnicalCategories(text ...string) []TechnicalCategory {
	searchText := strings.ToLower(strings.Join(text, " "))
	categoryMap := make(map[TechnicalCategory]bool)

	for category, keywords := range TechnicalCategoryKeywords {
		for _, keyword := range keywords {
			if strings.Contains(searchText, keyword) {
				categoryMap[category] = true
				break
			}
		}
	}

	categories := make([]TechnicalCategory, 0, len(categoryMap))
	for category := range categoryMap {
		categories = append(categories, category)
	}

	return categories
}

// FormatSegmentLabel creates a label string for a customer segment
func FormatSegmentLabel(segment CustomerSegment) string {
	return fmt.Sprintf("segment:%s", segment)
}

// FormatCompetitiveLabel creates a label string for competitive context
func FormatCompetitiveLabel(competitor Competitor) string {
	return fmt.Sprintf("competitive:%s", competitor)
}

// FormatPriorityLabel creates a label string for priority
func FormatPriorityLabel(priority Priority) string {
	return fmt.Sprintf("priority:%s", priority)
}

// FormatCategoryLabel creates a label string for technical category
func FormatCategoryLabel(category TechnicalCategory) string {
	return fmt.Sprintf("category:%s", category)
}
