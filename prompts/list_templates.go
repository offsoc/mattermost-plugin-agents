// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

//go:build ignore

package main

import (
	"fmt"
	"log"
	"text/template"

	"github.com/mattermost/mattermost-plugin-ai/prompts"
)

func main() {
	// Parse templates directly
	templates, err := template.ParseFS(prompts.PromptsFolder, "*.tmpl", "pm/*.tmpl")
	if err != nil {
		log.Fatalf("Failed to parse templates: %v", err)
	}

	fmt.Println("=== LOADED TEMPLATES ===")
	for _, t := range templates.Templates() {
		fmt.Printf("- %s\n", t.Name())
	}
}
