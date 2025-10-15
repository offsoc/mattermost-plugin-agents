// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

//go:build ignore

package main

import (
	"fmt"
	"log"

	"github.com/mattermost/mattermost-plugin-ai/llm"
	"github.com/mattermost/mattermost-plugin-ai/prompts"
)

func main() {
	// Load the prompts system
	p, err := llm.NewPrompts(prompts.PromptsFolder)
	if err != nil {
		log.Fatalf("Failed to load prompts: %v", err)
	}

	// Create empty context
	ctx := &llm.Context{}

	// Test rendering the strategic alignment prompt
	promptName := "pm_strategic_alignment_system"

	result, err := p.Format(promptName, ctx)
	if err != nil {
		log.Fatalf("Failed to format prompt '%s': %v", promptName, err)
	}

	fmt.Println("=== RENDERED PROMPT ===")
	fmt.Println(result)
	fmt.Println("\n=== LENGTH ===")
	fmt.Printf("Total characters: %d\n", len(result))

	// Check if our fragment content appears
	if containsString(result, "CRITICAL: CALL ALL REQUIRED TOOLS AT ONCE") {
		fmt.Println("\n✅ Multi-tool policy fragment IS included")
	} else {
		fmt.Println("\n❌ Multi-tool policy fragment NOT included")
	}

	if containsString(result, "IMPORTANT - Call Multiple Tools Together When Needed") {
		fmt.Println("✅ Role-specific tool examples ARE included")
	} else {
		fmt.Println("❌ Role-specific tool examples NOT included")
	}

	if containsString(result, "Known Unknowns") {
		fmt.Println("✅ Response structure fragment IS included - found 'Known Unknowns' section")
	} else {
		fmt.Println("❌ Response structure fragment NOT included")
	}

	if containsString(result, "RICE Framework for Prioritization") {
		fmt.Println("✅ RICE framework instructions found")
	} else {
		fmt.Println("❌ RICE framework instructions not found")
	}
}

func containsString(haystack, needle string) bool {
	return len(haystack) >= len(needle) && findSubstring(haystack, needle)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
