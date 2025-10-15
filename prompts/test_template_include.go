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
	// Parse templates directly
	tmpl, err := template.ParseFS(prompts.PromptsFolder, "*.tmpl", "pm/*.tmpl")
	if err != nil {
		log.Fatalf("Failed to parse templates: %v", err)
	}

	// Try to execute the strategic alignment template
	var output strings.Builder
	err = tmpl.ExecuteTemplate(&output, "pm_strategic_alignment_system.tmpl", nil)
	if err != nil {
		log.Fatalf("Failed to execute template: %v", err)
	}

	result := output.String()
	fmt.Println("=== RENDERED TEMPLATE ===")
	fmt.Printf("Length: %d characters\n\n", len(result))

	// Check if our fragment content appears
	checks := map[string]string{
		"Fragment header":       "Effective PM Communication with Data Sources",
		"Citations section":     "Data-Driven Citations",
		"RICE framework":        "RICE Framework for Prioritization",
		"Recommendations":       "Make Recommendations Actionable",
		"Known Unknowns":        "Acknowledge Limitations and Unknowns",
		"Executive consumption": "Structure for Executive Consumption",
	}

	fmt.Println("=== CHECKING FOR FRAGMENT CONTENT ===")
	allFound := true
	for name, searchStr := range checks {
		if strings.Contains(result, searchStr) {
			fmt.Printf("✅ %s: FOUND\n", name)
		} else {
			fmt.Printf("❌ %s: NOT FOUND\n", name)
			allFound = false
		}
	}

	if allFound {
		fmt.Println("\n🎉 SUCCESS: Fragment IS being included!")
	} else {
		fmt.Println("\n❌ FAILURE: Fragment is NOT being included")
		fmt.Println("\nFirst 500 chars of output:")
		if len(result) > 500 {
			fmt.Println(result[:500])
		} else {
			fmt.Println(result)
		}
	}
}
