// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/mattermost/mattermost-plugin-ai/datasources"
)

func main() {
	baseURL := flag.String("url", "https://mattermost.uservoice.com", "UserVoice base URL")
	outputPath := flag.String("output", "assets/uservoice_suggestions.json", "Output JSON file path")
	days := flag.Int("days", 730, "Fetch suggestions from last N days (730 = 2 years)")
	showStats := flag.Bool("stats", false, "Show statistics for existing export file")
	flag.Parse()

	if *showStats {
		if err := datasources.UserVoiceExportStats(*outputPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	fmt.Printf("UserVoice Exporter\n")
	fmt.Printf("==================\n")
	fmt.Printf("URL: %s\n", *baseURL)
	fmt.Printf("Output: %s\n", *outputPath)
	fmt.Printf("Date Range: Last %d days\n\n", *days)
	fmt.Printf("⚠️  This may take 70+ seconds due to UserVoice rate limiting...\n\n")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)

	exporter := datasources.NewUserVoiceExporter(nil)

	if err := exporter.ExportAllSuggestions(ctx, *baseURL, *outputPath, *days); err != nil {
		cancel()
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	cancel()

	fmt.Println("\n--- Export Statistics ---")
	if err := datasources.UserVoiceExportStats(*outputPath); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to generate stats: %v\n", err)
	}

	fmt.Println("\n✅ Export complete!")
	fmt.Printf("\nTo use this data source:\n")
	fmt.Printf("1. The file is saved at: %s\n", *outputPath)
	fmt.Printf("2. Add to file_protocol.go as a new source\n")
	fmt.Printf("3. Enable the source in config\n")
}
