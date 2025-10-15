// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

//go:build ignore

package main

import (
	"fmt"
	"log"
	"strings"
	"text/template"

	"github.com/mattermost/mattermost-plugin-ai/prompts"
)

func main() {
	// Parse templates
	tmpl, err := template.ParseFS(prompts.PromptsFolder, "*.tmpl", "pm/*.tmpl")
	if err != nil {
		log.Fatalf("Failed to parse templates: %v", err)
	}

	// Execute strategic alignment template
	var output strings.Builder
	err = tmpl.ExecuteTemplate(&output, "pm_strategic_alignment_system.tmpl", nil)
	if err != nil {
		log.Fatalf("Failed to execute template: %v", err)
	}

	result := output.String()
	lines := strings.Split(result, "\n")

	fmt.Printf("=== TEMPLATE STRUCTURE ===\n")
	fmt.Printf("Total length: %d chars, %d lines\n\n", len(result), len(lines))

	// Find where our fragment content starts
	fragmentStart := -1
	for i, line := range lines {
		if strings.Contains(line, "Effective PM Communication with Data Sources") {
			fragmentStart = i
			break
		}
	}

	if fragmentStart == -1 {
		fmt.Println("❌ Fragment not found!")
		return
	}

	fmt.Printf("Fragment starts at line %d (%.1f%% through prompt)\n\n", fragmentStart+1, float64(fragmentStart)/float64(len(lines))*100)

	// Show structure
	fmt.Println("=== PROMPT STRUCTURE ===")
	fmt.Printf("Lines 1-%d: Pre-fragment content\n", fragmentStart)
	fmt.Printf("Lines %d-%d: Fragment content\n", fragmentStart+1, len(lines))

	fmt.Println("\n=== FIRST 30 LINES (Before fragment) ===")
	for i := 0; i < 30 && i < len(lines); i++ {
		fmt.Printf("%3d: %s\n", i+1, lines[i])
	}

	fmt.Printf("\n=== FRAGMENT START (Lines %d-%d) ===\n", fragmentStart+1, fragmentStart+10)
	for i := fragmentStart; i < fragmentStart+10 && i < len(lines); i++ {
		fmt.Printf("%3d: %s\n", i+1, lines[i])
	}

	fmt.Println("\n=== LAST 10 LINES ===")
	start := len(lines) - 10
	if start < 0 {
		start = 0
	}
	for i := start; i < len(lines); i++ {
		fmt.Printf("%3d: %s\n", i+1, lines[i])
	}
}
