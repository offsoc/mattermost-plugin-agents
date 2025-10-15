// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"fmt"
	"strings"

	pm "github.com/mattermost/mattermost-plugin-ai/datasources/segments/pm"
)

// ZendeskTicket represents a support ticket from Zendesk
type ZendeskTicket struct {
	ID          string
	Title       string
	UpdatedBy   string
	Public      bool
	Owner       string
	Customer    string
	Tags        []string
	Content     string
	Description string
}

// fetchFromZendeskText processes Zendesk tickets from text files
func (f *FileProtocol) fetchFromZendeskText(content, sourceName string, request ProtocolRequest) ([]Doc, error) {
	tickets := f.parseTextTickets(content)
	totalTickets := len(tickets)

	if f.pluginAPI != nil {
		f.pluginAPI.LogDebug(sourceName+": starting search",
			"total_tickets", totalTickets,
			"limit", request.Limit)
	}

	var docs []Doc
	matchedCount := 0
	filteredCount := 0

	for i, ticket := range tickets {
		matches := f.matchesTicketSearchBoolean(ticket, request.Topic)

		if f.pluginAPI != nil && i < 5 {
			f.pluginAPI.LogDebug(sourceName+": ticket match check",
				"index", i,
				"id", ticket.ID,
				"title", ticket.Title,
				"matches", matches)
		}

		if matches {
			matchedCount++
			doc := f.ticketToDoc(ticket, sourceName)

			if sourceName == SourceZendeskTickets {
				docs = append(docs, doc)
				if f.pluginAPI != nil && len(docs) <= 3 {
					f.pluginAPI.LogDebug(sourceName+": doc accepted",
						"title", doc.Title,
						"count", len(docs))
				}
			} else {
				accepted, reason := f.universalScorer.IsUniversallyAcceptableWithReason(doc.Content, doc.Title, sourceName, request.Topic)
				if accepted {
					docs = append(docs, doc)
				} else {
					filteredCount++
					if f.pluginAPI != nil {
						f.pluginAPI.LogDebug(sourceName+": filtered out",
							"title", doc.Title,
							"ticket_id", ticket.ID,
							"reason", reason)
					}
				}
			}
		}
	}

	if f.pluginAPI != nil {
		f.pluginAPI.LogDebug(sourceName+": search complete",
			"total_tickets", totalTickets,
			"matched_query", matchedCount,
			"filtered_out", filteredCount,
			"final_results", len(docs))
	}

	return docs, nil
}

// parseTextTickets parses structured text content into tickets
func (f *FileProtocol) parseTextTickets(content string) []ZendeskTicket {
	var tickets []ZendeskTicket
	lines := strings.Split(content, "\n")

	var currentTicket *ZendeskTicket
	var contentLines []string

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		if strings.HasPrefix(line, "#zd") {
			if currentTicket != nil {
				currentTicket.Content = strings.TrimSpace(strings.Join(contentLines, "\n"))
				currentTicket.Description = f.extractTicketDescription(currentTicket.Content)
				tickets = append(tickets, *currentTicket)
			}

			parts := strings.SplitN(line, ":", 2)
			currentTicket = &ZendeskTicket{
				ID: strings.TrimSpace(parts[0]),
			}
			if len(parts) > 1 {
				currentTicket.Title = strings.TrimSpace(parts[1])
			}
			contentLines = []string{}
		} else if currentTicket != nil {
			switch {
			case strings.HasPrefix(line, "Updated by"):
				currentTicket.UpdatedBy = f.extractValueAfter(line, "Updated by")
			case strings.HasPrefix(line, "Public:"):
				currentTicket.Public = strings.Contains(line, "true")
			case strings.HasPrefix(line, "Ticket owner:"):
				if i+1 < len(lines) {
					currentTicket.Customer = strings.TrimSpace(lines[i+1])
					currentTicket.Owner = currentTicket.Customer
					i++
				}
			case strings.HasPrefix(line, " Tags:"):
				tags := strings.TrimPrefix(line, " Tags:")
				currentTicket.Tags = f.parseTags(tags)
			case strings.TrimSpace(line) != "" && !strings.HasPrefix(line, "Zendesk") && !strings.HasPrefix(line, "BOT"):
				contentLines = append(contentLines, line)
			}
		}
	}

	if currentTicket != nil {
		currentTicket.Content = strings.TrimSpace(strings.Join(contentLines, "\n"))
		currentTicket.Description = f.extractTicketDescription(currentTicket.Content)
		tickets = append(tickets, *currentTicket)
	}

	return tickets
}

// extractValueAfter extracts the value after a prefix in a line
func (f *FileProtocol) extractValueAfter(line, prefix string) string {
	parts := strings.SplitN(line, prefix, 2)
	if len(parts) > 1 {
		return strings.TrimSpace(parts[1])
	}
	return ""
}

// parseTags parses space-separated tags
func (f *FileProtocol) parseTags(tagStr string) []string {
	tagStr = strings.TrimSpace(tagStr)
	if tagStr == "" {
		return []string{}
	}
	tags := strings.Fields(tagStr)
	return tags
}

// extractTicketDescription extracts first meaningful paragraph from content
func (f *FileProtocol) extractTicketDescription(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) > 20 && !strings.HasPrefix(line, "Updated by") {
			if len(line) > 200 {
				return line[:200] + "..."
			}
			return line
		}
	}
	return content
}

// matchesTicketSearch checks if a ticket matches search terms
func (f *FileProtocol) matchesTicketSearch(ticket ZendeskTicket, searchTerms []string) bool {
	if len(searchTerms) == 0 {
		return true
	}

	searchable := strings.ToLower(
		ticket.ID + " " +
			ticket.Title + " " +
			ticket.Content + " " +
			ticket.Customer + " " +
			strings.Join(ticket.Tags, " "),
	)

	matchCount := 0
	for _, term := range searchTerms {
		if strings.Contains(searchable, term) {
			matchCount++
		}
	}

	threshold := len(searchTerms) / 2
	if threshold < 1 {
		threshold = 1
	}

	return matchCount >= threshold
}

// matchesTicketSearchBoolean checks if a ticket matches a boolean query
func (f *FileProtocol) matchesTicketSearchBoolean(ticket ZendeskTicket, topic string) bool {
	if topic == "" {
		return true
	}

	queryNode, err := ParseBooleanQuery(topic)
	if err != nil {
		searchTerms := f.extractSearchTerms(topic)
		return f.matchesTicketSearch(ticket, searchTerms)
	}

	searchable := ticket.ID + " " +
		ticket.Title + " " +
		ticket.Content + " " +
		ticket.Customer + " " +
		strings.Join(ticket.Tags, " ")

	return EvaluateBoolean(queryNode, searchable)
}

// ticketToDoc converts a Zendesk ticket to a Doc, extracting status, priority, tier, tags,
// environment, region, and comment threads while preserving ticket-requester relationships
func (f *FileProtocol) ticketToDoc(ticket ZendeskTicket, sourceName string) Doc {
	meta := extractZendeskMetadata(ticket)

	pmMeta, _ := meta.RoleMetadata.(pm.Metadata)
	priority := pmMeta.Priority

	// Extract Zendesk-specific fields from tags
	tier := ""
	environment := ""
	region := ""
	for _, tag := range ticket.Tags {
		tagLower := strings.ToLower(tag)
		switch {
		case strings.HasPrefix(tag, "tier_"):
			tier = tag
		case tag == "production":
			environment = "production"
		case tag == "staging/uat/qa":
			environment = "staging"
		case tag == "amer_schedule" || tag == "americas":
			region = "americas"
		case tag == "apac_schedule" || strings.Contains(tagLower, "asia"):
			region = "asia-pacific"
		case tag == "emea_schedule" || strings.Contains(tagLower, "europe"):
			region = "emea"
		}
	}

	content := ticket.Content
	metadataSection := formatEntityMetadata(meta)

	// Add Zendesk-specific metadata after entity metadata
	if metadataSection != "" {
		// Add Zendesk-specific fields before closing metadata section
		zendeskInfo := ""
		if ticket.Customer != "" {
			// Extract customer name (before first dash or email)
			customerName := ticket.Customer
			if dashIdx := strings.Index(customerName, " - "); dashIdx > 0 {
				customerName = customerName[:dashIdx]
			}
			zendeskInfo += fmt.Sprintf("Customer: %s\n", customerName)
		}
		if tier != "" {
			zendeskInfo += fmt.Sprintf("Tier: %s\n", tier)
		}
		if environment != "" {
			zendeskInfo += fmt.Sprintf("Environment: %s\n", environment)
		}

		// Insert Zendesk info before the closing "---"
		if zendeskInfo != "" {
			metadataSection = strings.TrimSuffix(metadataSection, "---\n")
			metadataSection += zendeskInfo + "---\n"
		}
	}

	content = metadataSection + content

	if len(ticket.Tags) > 0 {
		content = fmt.Sprintf("%s\n\nTags: %s", content, strings.Join(ticket.Tags, " "))
	}

	section := SectionGeneral
	switch {
	case priority == pm.PriorityHigh && environment == "production":
		section = SectionCritical
	case environment == "production":
		section = SectionProduction
	case strings.Contains(strings.ToLower(ticket.Title), "bug"):
		section = SectionBug
	}

	labels := ticket.Tags

	if tier != "" {
		labels = append(labels, tier)
	}

	labels = append(labels, buildLabelsFromMetadata(meta)...)

	if environment != "" {
		labels = append(labels, fmt.Sprintf("env:%s", environment))
	}

	if region != "" {
		labels = append(labels, fmt.Sprintf("region:%s", region))
	}

	url := fmt.Sprintf("zendesk://ticket/%s", strings.TrimPrefix(ticket.ID, "#"))

	return Doc{
		Title:        ticket.Title,
		Content:      content,
		URL:          url,
		Section:      section,
		Source:       sourceName,
		Author:       ticket.UpdatedBy,
		LastModified: ticket.UpdatedBy,
		Labels:       labels,
	}
}
