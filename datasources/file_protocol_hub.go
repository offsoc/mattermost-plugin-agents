// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"fmt"
	"strings"

	pm "github.com/mattermost/mattermost-plugin-ai/datasources/segments/pm"
)

// MattermostHubPost represents a post from Mattermost Hub channels (contact-sales, customer-feedback)
type MattermostHubPost struct {
	FirstName    string
	LastName     string
	Company      string
	Email        string
	Phone        string
	Owner        string
	CompanyType  string
	Message      string
	Notes        string
	Tags         []string
	SalesforceID string
	Licenses     string
	Team         string
	Replacement  string
	Timestamp    string
}

// fetchFromHubText processes Mattermost Hub posts from text files
func (f *FileProtocol) fetchFromHubText(content, sourceName string, request ProtocolRequest) ([]Doc, error) {
	posts := f.parseHubPosts(content)
	totalPosts := len(posts)
	var docs []Doc

	for _, post := range posts {
		if f.matchesHubPostSearch(post, request.Topic) {
			doc := f.hubPostToDoc(post, sourceName)
			docs = append(docs, doc)
		}
	}

	if f.pluginAPI != nil && request.Topic != "" {
		f.pluginAPI.LogDebug(sourceName+": search results", "total", totalPosts, "matched", len(docs))
	}

	return docs, nil
}

// matchesHubPostSearch checks if a Hub post matches a search query
func (f *FileProtocol) matchesHubPostSearch(post MattermostHubPost, topic string) bool {
	if topic == "" {
		return true
	}

	searchable := strings.ToLower(
		post.FirstName + " " +
			post.LastName + " " +
			post.Company + " " +
			post.Email + " " +
			post.Phone + " " +
			post.Timestamp + " " +
			post.Message)

	queryNode, err := ParseBooleanQuery(topic)
	if err == nil {
		return EvaluateBoolean(queryNode, searchable)
	}

	searchTerms := f.extractSearchTerms(topic)
	matchCount := 0
	for _, term := range searchTerms {
		if strings.Contains(searchable, strings.ToLower(term)) {
			matchCount++
		}
	}

	threshold := len(searchTerms) / 2
	if threshold < 1 {
		threshold = 1
	}

	return matchCount >= threshold
}

// parseHubPosts parses Mattermost Hub channel content into structured posts
func (f *FileProtocol) parseHubPosts(content string) []MattermostHubPost {
	var posts []MattermostHubPost
	lines := strings.Split(content, "\n")

	var currentPost *MattermostHubPost
	var messageLines []string
	inMessage := false

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		if (strings.Contains(line, "Bot") || strings.Contains(line, "BOT")) && i+1 < len(lines) {
			nextLine := strings.TrimSpace(lines[i+1])
			if nextLine == "BOT" || strings.HasSuffix(line, "BOT") {
				if currentPost != nil {
					currentPost.Message = strings.TrimSpace(strings.Join(messageLines, "\n"))
					if currentPost.Company != "" || currentPost.Message != "" {
						posts = append(posts, *currentPost)
					}
				}

				currentPost = &MattermostHubPost{}
				messageLines = []string{}
				inMessage = false
				continue
			}
		}

		if currentPost == nil {
			continue
		}

		switch {
		case strings.HasPrefix(line, "First Name:"):
			currentPost.FirstName = f.extractValueAfter(line, "First Name:")
		case strings.HasPrefix(line, "Last Name:"):
			currentPost.LastName = f.extractValueAfter(line, "Last Name:")
		case strings.HasPrefix(line, "Company:"):
			currentPost.Company = f.extractValueAfter(line, "Company:")
		case strings.HasPrefix(line, "Email:"):
			currentPost.Email = f.extractValueAfter(line, "Email:")
		case strings.HasPrefix(line, "Phone:"):
			currentPost.Phone = f.extractValueAfter(line, "Phone:")
		case strings.HasPrefix(line, "Owner:"):
			currentPost.Owner = f.extractValueAfter(line, "Owner:")
		case strings.HasPrefix(line, "Company Type:"):
			companyTypeRaw := f.extractValueAfter(line, "Company Type:")
			currentPost.CompanyType = companyTypeRaw
			if strings.Contains(companyTypeRaw, "#") {
				parts := strings.Split(companyTypeRaw, "#")
				for _, part := range parts {
					tag := strings.TrimSpace(part)
					if tag != "" {
						currentPost.Tags = append(currentPost.Tags, tag)
					}
				}
			}
		case strings.HasPrefix(line, "Message:") || strings.HasPrefix(line, "Notes:"):
			inMessage = true
			if strings.HasPrefix(line, "Message:") {
				initialMsg := f.extractValueAfter(line, "Message:")
				if initialMsg != "" {
					messageLines = append(messageLines, initialMsg)
				}
			} else {
				initialNotes := f.extractValueAfter(line, "Notes:")
				if initialNotes != "" {
					currentPost.Notes = initialNotes
					messageLines = append(messageLines, initialNotes)
				}
			}
		case strings.HasPrefix(line, "Link to SFDC:"):
			currentPost.SalesforceID = f.extractValueAfter(line, "Link to SFDC:")
			inMessage = false
		case strings.HasPrefix(line, "Number of licenses:"):
			currentPost.Licenses = f.extractValueAfter(line, "Number of licenses:")
		case strings.HasPrefix(line, "Team using MM:"):
			currentPost.Team = f.extractValueAfter(line, "Team using MM:")
		case strings.HasPrefix(line, "Tech replacement:"):
			currentPost.Replacement = f.extractValueAfter(line, "Tech replacement:")
		case strings.HasPrefix(line, "Clarifying Info:"):
			inMessage = false
		case inMessage && strings.TrimSpace(line) != "":
			messageLines = append(messageLines, line)
		}
	}

	if currentPost != nil {
		currentPost.Message = strings.TrimSpace(strings.Join(messageLines, "\n"))
		if currentPost.Company != "" || currentPost.Message != "" {
			posts = append(posts, *currentPost)
		}
	}

	return posts
}

// hubPostToDoc converts a MattermostHubPost to a Doc, extracting category, customer segments,
// priority, license counts, company type, and owner info from Hub's structured contact/feedback format
func (f *FileProtocol) hubPostToDoc(post MattermostHubPost, sourceName string) Doc {
	segments, categories, competitive, priority, crossRefs := extractHubMetadata(post)

	var metadataLines []string

	if len(segments) > 0 {
		segmentLabels := make([]string, len(segments))
		for i, seg := range segments {
			segmentLabels[i] = string(seg)
		}
		metadataLines = append(metadataLines, fmt.Sprintf("Segments: %s", strings.Join(segmentLabels, ", ")))
	}

	if len(categories) > 0 {
		categoryLabels := make([]string, len(categories))
		for i, cat := range categories {
			categoryLabels[i] = string(cat)
		}
		metadataLines = append(metadataLines, fmt.Sprintf("Categories: %s", strings.Join(categoryLabels, ", ")))
	}

	if competitive != "" {
		metadataLines = append(metadataLines, fmt.Sprintf("Competitive Context: %s", competitive))
	}

	if priority != "" {
		metadataLines = append(metadataLines, fmt.Sprintf("Priority: %s", priority))
	}

	if len(crossRefs) > 0 {
		metadataLines = append(metadataLines, fmt.Sprintf("References: %s", strings.Join(crossRefs, ", ")))
	}

	content := ""
	if len(metadataLines) > 0 {
		content = strings.Join(metadataLines, "\n") + "\n\n"
	}

	content += fmt.Sprintf("Company: %s\n", post.Company)
	if post.FirstName != "" || post.LastName != "" {
		content += fmt.Sprintf("Contact: %s %s\n", post.FirstName, post.LastName)
	}
	if post.Email != "" {
		content += fmt.Sprintf("Email: %s\n", post.Email)
	}
	if post.Owner != "" {
		content += fmt.Sprintf("Owner: %s\n", post.Owner)
	}
	if post.CompanyType != "" {
		content += fmt.Sprintf("Type: %s\n", post.CompanyType)
	}
	if post.Licenses != "" {
		content += fmt.Sprintf("Licenses: %s\n", post.Licenses)
	}
	if post.Replacement != "" {
		content += fmt.Sprintf("Replacing: %s\n", post.Replacement)
	}
	content += "\n" + post.Message

	section := SectionGeneral
	if priority == pm.PriorityHigh || strings.Contains(strings.ToLower(post.CompanyType), "federal") {
		section = SectionCritical
	}

	labels := make([]string, 0)
	for _, seg := range segments {
		labels = append(labels, pm.FormatSegmentLabel(seg))
	}
	for _, cat := range categories {
		labels = append(labels, pm.FormatCategoryLabel(cat))
	}
	if competitive != "" {
		labels = append(labels, pm.FormatCompetitiveLabel(competitive))
	}
	if priority != "" {
		labels = append(labels, pm.FormatPriorityLabel(priority))
	}

	if post.Licenses != "" {
		licenseCount := parseLicenseCount(post.Licenses)
		if licenseCount > 0 {
			labels = append(labels, fmt.Sprintf("licenses:%d", licenseCount))
			switch {
			case licenseCount >= 10000:
				labels = append(labels, "enterprise_large")
			case licenseCount >= 1000:
				labels = append(labels, "enterprise_medium")
			case licenseCount >= 100:
				labels = append(labels, "smb_large")
			}
		}
	}

	daysCreated := DaysSince(post.Timestamp)
	if recencyLabel := FormatRecencyLabel(daysCreated); recencyLabel != "" {
		labels = append(labels, recencyLabel+"_created")
	}

	title := fmt.Sprintf("%s - %s", post.Company, post.CompanyType)
	if post.FirstName != "" || post.LastName != "" {
		title = fmt.Sprintf("%s %s - %s", post.FirstName, post.LastName, post.Company)
	}

	url := fmt.Sprintf("hub://%s/%s", sourceName, post.Company)
	if post.SalesforceID != "" {
		url = post.SalesforceID
	}

	author := post.Owner
	if author == "" && (post.FirstName != "" || post.LastName != "") {
		author = fmt.Sprintf("%s %s", post.FirstName, post.LastName)
	}

	return Doc{
		Title:        title,
		Content:      content,
		URL:          url,
		Section:      section,
		Source:       sourceName,
		Labels:       labels,
		Author:       author,
		CreatedDate:  post.Timestamp,
		LastModified: post.Timestamp,
	}
}

// extractHubMetadata extracts metadata from Hub posts
func extractHubMetadata(post MattermostHubPost) (
	segments []pm.CustomerSegment,
	categories []pm.TechnicalCategory,
	competitive pm.Competitor,
	priority pm.Priority,
	crossRefs []string,
) {
	searchText := strings.ToLower(
		post.Company + " " +
			post.Message + " " +
			post.Notes + " " +
			post.CompanyType + " " +
			post.Replacement + " " +
			strings.Join(post.Tags, " "),
	)

	segments = pm.ExtractCustomerSegments(searchText)
	categories = pm.ExtractTechnicalCategories(searchText)
	competitive = pm.ExtractCompetitiveContext(searchText)
	priority = pm.EstimatePriority(searchText, post.CompanyType)
	crossRefs = ExtractCrossReferences(post.Message + " " + post.Notes + " " + post.SalesforceID)

	return segments, categories, competitive, priority, crossRefs
}
