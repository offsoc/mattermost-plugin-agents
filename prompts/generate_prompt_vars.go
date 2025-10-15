// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

//go:build ignore

package main

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	var output bytes.Buffer
	output.WriteString("// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.\n// See LICENSE.txt for license information.\n\npackage prompts\n\n// Automatically generated convenience vars for the filenames in prompts/\nconst (\n")

	// Scan root prompts directory
	scanTemplatesInDir(".", &output)

	// Scan role subdirectories
	scanTemplatesInDir("pm", &output)
	scanTemplatesInDir("dev", &output)

	output.WriteString(")\n")

	// Format the output using gofmt
	formattedOutput, err := format.Source(output.Bytes())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
		os.Exit(1)
	}

	// Write the formatted output to the file
	err = os.WriteFile("prompts_vars.go", formattedOutput, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing output file: %v\n", err)
		os.Exit(1)
	}
}

func scanTemplatesInDir(dir string, output *bytes.Buffer) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading %s directory: %v\n", dir, err)
		os.Exit(1)
	}

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".tmpl" {
			// Skip template fragments (files starting with underscore)
			// These are included by other templates and not used directly
			if strings.HasPrefix(entry.Name(), "_") {
				continue
			}
			baseName := strings.TrimSuffix(entry.Name(), ".tmpl")
			// Note: Go's template.ParseFS registers templates by basename only, not full path
			varName := "Prompt" + toCamelCase(baseName)
			fmt.Fprintf(output, "\t%s = %q\n", varName, baseName)
		}
	}
}

func toCamelCase(s string) string {
	// Common acronyms that should be all uppercase
	acronyms := map[string]bool{
		"api": true,
		"pr":  true,
		"id":  true,
		"url": true,
		"uri": true,
		"sql": true,
		"db":  true,
		"ui":  true,
		"ci":  true,
		"cd":  true,
	}

	words := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-'
	})
	for i, word := range words {
		lower := strings.ToLower(word)
		if acronyms[lower] {
			words[i] = strings.ToUpper(word)
		} else {
			words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}
	return strings.Join(words, "")
}
