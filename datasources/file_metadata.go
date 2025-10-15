// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"strings"

	pm "github.com/mattermost/mattermost-plugin-ai/datasources/segments/pm"
)

func extractProductBoardMetadata(feature ProductBoardFeature) EntityMetadata {
	textParts := []string{feature.Name, feature.Description, feature.Parent}
	allText := strings.Join(textParts, " ")

	pmMeta := pm.Metadata{
		Segments:    pm.ExtractCustomerSegments(textParts...),
		Categories:  pm.ExtractTechnicalCategories(textParts...),
		Competitive: pm.ExtractCompetitiveContext(textParts...),
		Priority:    pm.EstimatePriority(feature.Description+" "+feature.Name, feature.State),
	}

	meta := EntityMetadata{
		EntityType:   "productboard",
		EntityID:     feature.Name, // ProductBoard feature name as ID
		RoleMetadata: pmMeta,
	}

	// Entity linking (role-agnostic)
	meta.GitHubIssues, meta.GitHubPRs = extractGitHubReferences(allText)
	meta.JiraTickets = extractJiraReferences(allText)
	meta.Commits = extractCommitReferences(allText)
	meta.ModifiedFiles = extractFileReferences(allText)
	meta.ConfluencePages = extractConfluenceReferences(allText)
	meta.MattermostLinks = extractMattermostReferences(allText)

	return meta
}

func extractZendeskMetadata(ticket ZendeskTicket) EntityMetadata {
	var textParts []string
	textParts = append(textParts, ticket.Title, ticket.Content, ticket.Description, ticket.Customer)

	allText := strings.Join(textParts, " ")

	pmMeta := pm.Metadata{
		Segments:    pm.ExtractCustomerSegments(textParts...),
		Categories:  pm.ExtractTechnicalCategories(textParts...),
		Competitive: pm.ExtractCompetitiveContext(textParts...),
		Priority:    pm.EstimatePriority(ticket.Title+" "+ticket.Content, ""),
	}

	// Enhance with Zendesk-specific tag parsing (tag-based signals override content-based detection)
	for _, tag := range ticket.Tags {
		tagLower := strings.ToLower(tag)

		// Priority elevation from tags
		switch {
		case tag == "customer_confirmed_priority":
			pmMeta.Priority = pm.PriorityHigh
		case tag == "pagerduty_sent" || tag == "weekend_schedule" || tag == "support_webhook_sent":
			if pmMeta.Priority != pm.PriorityHigh {
				pmMeta.Priority = pm.PriorityHigh
			}
		case strings.HasPrefix(tag, "tier_"):
			switch tag {
			case "tier_1":
				if pmMeta.Priority != pm.PriorityHigh {
					pmMeta.Priority = pm.PriorityHigh
				}
			case "tier_2":
				if pmMeta.Priority == "" || pmMeta.Priority == pm.PriorityLow {
					pmMeta.Priority = pm.PriorityMedium
				}
			case "tier_3", "tier_4":
				if pmMeta.Priority == "" {
					pmMeta.Priority = pm.PriorityMedium
				}
			}
		}

		// Segment enhancement from tags
		switch tag {
		case "us_only":
			if !containsSegment(pmMeta.Segments, pm.SegmentFederal) {
				pmMeta.Segments = append(pmMeta.Segments, pm.SegmentFederal)
			}
		case "100k_customer", "premier", "professional":
			if !containsSegment(pmMeta.Segments, pm.SegmentEnterprise) {
				pmMeta.Segments = append(pmMeta.Segments, pm.SegmentEnterprise)
			}
		case "enterprise", "mid-market/enterprise":
			if !containsSegment(pmMeta.Segments, pm.SegmentEnterprise) {
				pmMeta.Segments = append(pmMeta.Segments, pm.SegmentEnterprise)
			}
		}

		// Category enhancement from tags
		if strings.Contains(tagLower, "plugin") || strings.Contains(tagLower, "jira") || strings.Contains(tagLower, "playbooks") {
			categoryName := extractCategoryFromTag(tagLower)
			if categoryName != "" {
				category := pm.TechnicalCategory(categoryName)
				if !containsCategory(pmMeta.Categories, category) {
					pmMeta.Categories = append(pmMeta.Categories, category)
				}
			}
		}
	}

	// Segment detection from customer field (email domain analysis)
	if ticket.Customer != "" {
		customerLower := strings.ToLower(ticket.Customer)
		switch {
		case strings.Contains(customerLower, ".mil") || strings.Contains(customerLower, ".gov"):
			if !containsSegment(pmMeta.Segments, pm.SegmentFederal) {
				pmMeta.Segments = append(pmMeta.Segments, pm.SegmentFederal)
			}
		case strings.Contains(customerLower, "bank") || strings.Contains(customerLower, "financial"):
			if !containsSegment(pmMeta.Segments, pm.SegmentFinance) {
				pmMeta.Segments = append(pmMeta.Segments, pm.SegmentFinance)
			}
		}
	}

	meta := EntityMetadata{
		EntityType:   EntityTypeZendesk,
		EntityID:     ticket.ID,
		RoleMetadata: pmMeta,
	}

	// Entity linking (role-agnostic)
	meta.GitHubIssues, meta.GitHubPRs = extractGitHubReferences(allText)
	meta.JiraTickets = extractJiraReferences(allText)
	meta.Commits = extractCommitReferences(allText)
	meta.ModifiedFiles = extractFileReferences(allText)
	meta.ConfluencePages = extractConfluenceReferences(allText)
	meta.ZendeskTickets = extractZendeskReferences(allText)
	meta.MattermostLinks = extractMattermostReferences(allText)

	return meta
}

func extractUserVoiceMetadata(suggestion UserVoiceSuggestion) EntityMetadata {
	var textParts []string
	textParts = append(textParts, suggestion.Title, suggestion.Description)

	allText := strings.Join(textParts, " ")

	pmMeta := pm.Metadata{
		Segments:    pm.ExtractCustomerSegments(textParts...),
		Categories:  pm.ExtractTechnicalCategories(textParts...),
		Competitive: pm.ExtractCompetitiveContext(textParts...),
	}

	// Priority based on votes
	switch {
	case suggestion.Votes >= 100:
		pmMeta.Priority = pm.PriorityHigh
	case suggestion.Votes >= 20:
		pmMeta.Priority = pm.PriorityMedium
	default:
		pmMeta.Priority = pm.PriorityLow
	}

	meta := EntityMetadata{
		EntityType:   "uservoice",
		EntityID:     suggestion.ID,
		RoleMetadata: pmMeta,
	}

	// Entity linking (role-agnostic)
	meta.GitHubIssues, meta.GitHubPRs = extractGitHubReferences(allText)
	meta.JiraTickets = extractJiraReferences(allText)
	meta.Commits = extractCommitReferences(allText)
	meta.ModifiedFiles = extractFileReferences(allText)
	meta.ConfluencePages = extractConfluenceReferences(allText)
	meta.MattermostLinks = extractMattermostReferences(allText)

	return meta
}

// extractCategoryFromTag extracts technical category from Zendesk tag
func extractCategoryFromTag(tag string) string {
	switch {
	case strings.Contains(tag, "jira"):
		return "plugins"
	case strings.Contains(tag, "playbooks"):
		return "playbooks"
	case strings.Contains(tag, "plugin"):
		return "plugins"
	case strings.Contains(tag, "mobile"):
		return "mobile"
	case strings.Contains(tag, "database"):
		return "database"
	case strings.Contains(tag, "channel"):
		return "channels"
	case strings.Contains(tag, "authentication") || strings.Contains(tag, "saml") || strings.Contains(tag, "ldap"):
		return "authentication"
	default:
		return ""
	}
}
