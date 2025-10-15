// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/mattermost/mattermost-plugin-ai/mmapi"
)

// UserVoiceExporter exports all UserVoice suggestions to a JSON file
type UserVoiceExporter struct {
	protocol *UserVoiceProtocol
}

// NewUserVoiceExporter creates a new exporter
func NewUserVoiceExporter(pluginAPI mmapi.Client) *UserVoiceExporter {
	return &UserVoiceExporter{
		protocol: NewUserVoiceProtocol(&http.Client{}, pluginAPI),
	}
}

// ExportAllSuggestions fetches ALL suggestions from UserVoice REST API and saves to JSON
// This uses the UserVoice API v2 to get clean JSON data
// dateFilter: fetch suggestions from last N days (0 = all time)
func (e *UserVoiceExporter) ExportAllSuggestions(ctx context.Context, baseURL string, outputPath string, dateFilter int) error {
	if dateFilter == 0 {
		dateFilter = 730 // Default: 2 years
	}

	cutoffDate := time.Now().AddDate(0, 0, -dateFilter)

	// Extract subdomain from base URL
	e.protocol.ExtractSubdomain(baseURL)

	var allSuggestions []UserVoiceSuggestion

	fmt.Printf("Fetching suggestions from UserVoice API: %s\n", baseURL)
	fmt.Printf("Using subdomain: %s\n", e.protocol.Subdomain)

	page := 1
	perPage := 100

	for {
		// Use the new API-based fetch method
		suggestions, hasMore, err := e.protocol.FetchSuggestionsAPI(ctx, "", page, perPage)
		if err != nil {
			fmt.Printf("Warning: Failed to fetch page %d: %v\n", page, err)
			break
		}

		if len(suggestions) == 0 {
			break // No more suggestions on this page
		}

		// Apply date filter
		filteredCount := 0
		for _, suggestion := range suggestions {
			// Apply date filter if we have created date
			if suggestion.CreatedAt != "" {
				createdAt, err := time.Parse(time.RFC3339, suggestion.CreatedAt)
				if err == nil && createdAt.Before(cutoffDate) {
					continue // Skip suggestions older than cutoff
				}
			}

			allSuggestions = append(allSuggestions, suggestion)
			filteredCount++
		}

		fmt.Printf("  Fetched page %d: %d suggestions (%d after date filter, %d total so far)\n",
			page, len(suggestions), filteredCount, len(allSuggestions))

		if !hasMore {
			break // API indicates no more pages
		}

		page++

		// Safety check: max 1000 pages
		if page > 1000 {
			fmt.Printf("Warning: Reached max page limit (1000)\n")
			break
		}
	}

	if len(allSuggestions) == 0 {
		return fmt.Errorf("no suggestions found from API (check API key: USERVOICE_API_KEY or MM_AI_USERVOICE_TOKEN)")
	}

	// Save to JSON file
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(allSuggestions); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	fmt.Printf("\n✅ Successfully exported %d suggestions to %s\n", len(allSuggestions), outputPath)
	fmt.Printf("Date range: Last %d days (since %s)\n", dateFilter, cutoffDate.Format("2006-01-02"))

	return nil
}

// UserVoiceExportStats returns statistics about the exported file
func UserVoiceExportStats(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var suggestions []UserVoiceSuggestion
	if err := json.Unmarshal(data, &suggestions); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	statusCounts := make(map[string]int)
	voteCounts := make(map[string]int) // buckets: 0-10, 11-50, 51-100, 100+
	categoryCounts := make(map[string]int)

	for _, s := range suggestions {
		statusCounts[s.Status]++

		switch {
		case s.Votes <= 10:
			voteCounts["0-10"]++
		case s.Votes <= 50:
			voteCounts["11-50"]++
		case s.Votes <= 100:
			voteCounts["51-100"]++
		default:
			voteCounts["100+"]++
		}

		if s.Category != "" {
			categoryCounts[s.Category]++
		}
	}

	fmt.Printf("\n=== UserVoice Export Statistics ===\n")
	fmt.Printf("Total suggestions: %d\n\n", len(suggestions))

	fmt.Printf("By Status:\n")
	for status, count := range statusCounts {
		fmt.Printf("  %-20s: %d\n", status, count)
	}

	fmt.Printf("\nBy Vote Count:\n")
	for bucket, count := range voteCounts {
		fmt.Printf("  %-10s votes: %d\n", bucket, count)
	}

	fmt.Printf("\nBy Category:\n")
	for cat, count := range categoryCounts {
		if cat == "" {
			continue
		}
		fmt.Printf("  %-30s: %d\n", cat, count)
	}

	return nil
}
