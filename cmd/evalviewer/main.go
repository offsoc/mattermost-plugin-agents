// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// EvalLogLine matches the structure from evals/record.go
type EvalLogLine struct {
	Name      string  `json:"name"`
	Timestamp string  `json:"timestamp"`
	RunNumber int     `json:"run_number"`
	Rubric    string  `json:"rubric"`
	Output    string  `json:"output"`
	Reasoning string  `json:"reasoning"`
	Score     float64 `json:"score"`
	Pass      bool    `json:"pass"`
}

var (
	// Flags for view command
	filename         string
	showOnlyFailures bool
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "evalviewer",
		Short: "Display evaluation results from evals.jsonl",
		Long: `evalviewer is a CLI tool to run evaluations and display results in a nice table format.

It can either run tests and display results, or view existing evaluation results.`,
	}

	var runCmd = &cobra.Command{
		Use:   "run [go test flags and args]",
		Short: "Run eval tests and display results",
		Long: `Run go test with GOEVALS=1 environment variable set, then automatically
find and display the evaluation results in a TUI.

All arguments after 'run' are passed directly to 'go test'.`,
		Example: `  evalviewer run -v ./conversations         # Run evals for conversations package
  evalviewer run -v ./...                   # Run all evals
  evalviewer run -v -cover ./conversations  # Run with test coverage`,
		DisableFlagParsing: true,
		Run: func(cmd *cobra.Command, args []string) {
			runCommand(args)
		},
	}

	var viewCmd = &cobra.Command{
		Use:   "view",
		Short: "Display existing evaluation results",
		Long:  `Display evaluation results from an existing evals.jsonl file in a TUI.`,
		Example: `  evalviewer view -file evals.jsonl         # View existing results
  evalviewer view -failures-only            # Show only failures`,
		Run: func(cmd *cobra.Command, args []string) {
			viewCommandWithFlags()
		},
	}

	var checkCmd = &cobra.Command{
		Use:   "check [go test flags and args]",
		Short: "Run eval tests and check results (CI-friendly, no TUI)",
		Long: `Run go test with GOEVALS=1 environment variable set, then check the results
and exit with status code 1 if any evaluations failed. This command is designed
for CI/CD pipelines and does not use the interactive TUI.

All arguments after 'check' are passed directly to 'go test'.`,
		Example: `  evalviewer check ./conversations           # Run and check evals for conversations
  evalviewer check ./...                     # Run and check all evals
  evalviewer check -v -cover ./...           # Run with verbose output and coverage`,
		DisableFlagParsing: true,
		Run: func(cmd *cobra.Command, args []string) {
			checkCommand(args)
		},
	}

	// Add flags to view command
	viewCmd.Flags().StringVarP(&filename, "file", "f", "evals.jsonl", "Path to the evals.jsonl file")
	viewCmd.Flags().BoolVar(&showOnlyFailures, "failures-only", false, "Show only failed evaluations")

	// Add commands to root
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(viewCmd)
	rootCmd.AddCommand(checkCmd)

	// Execute
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runCommand(args []string) {
	// Execute go test with GOEVALS=1
	fmt.Println("Running evaluations...")

	// Prepare go test command
	cmdArgs := []string{"test"}
	cmdArgs = append(cmdArgs, args...)

	cmd := exec.Command("go", cmdArgs...)
	cmd.Env = append(os.Environ(), "GOEVALS=1")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run command and show output
	if err := cmd.Run(); err != nil {
		fmt.Printf("\nTests completed with errors: %v\n", err)
	} else {
		fmt.Println("\nTests completed successfully.")
	}

	// Find and display results
	evalFile, err := findEvalsFile()
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nError finding evals.jsonl: %v\n", err)
		fmt.Println("You can view results manually with: evalviewer view -file /path/to/evals.jsonl")
		os.Exit(1)
	}

	// Display results with default settings
	results, err := loadResults(evalFile, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading results: %v\n", err)
		os.Exit(1)
	}

	// Launch TUI
	if err := runTUI(results); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}
}

func viewCommandWithFlags() {
	results, err := loadResults(filename, false) // Don't pre-filter, let TUI handle it
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading results: %v\n", err)
		os.Exit(1)
	}

	// Launch TUI
	if err := runTUI(results); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}
}

func checkCommand(args []string) {
	// Execute go test with GOEVALS=1
	fmt.Println("Running evaluations...")

	// Prepare go test command
	cmdArgs := []string{"test"}
	cmdArgs = append(cmdArgs, args...)

	cmd := exec.Command("go", cmdArgs...)
	cmd.Env = append(os.Environ(), "GOEVALS=1")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run command and show output
	testErr := cmd.Run()
	if testErr != nil {
		fmt.Printf("\nTests completed with errors: %v\n", testErr)
	} else {
		fmt.Println("\nTests completed successfully.")
	}

	// Find and check results
	evalFile, err := findEvalsFile()
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nError finding evals.jsonl: %v\n", err)
		fmt.Println("No evaluation results found to check.")
		// If tests failed but no evals file, exit with test error
		if testErr != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Load and check results
	results, err := loadResults(evalFile, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading results: %v\n", err)
		os.Exit(1)
	}

	// Print summary
	printSummary(results)

	// Exit with appropriate status code
	hasFailures := false
	for _, result := range results {
		if !result.Pass {
			hasFailures = true
			break
		}
	}

	if hasFailures || testErr != nil {
		os.Exit(1)
	}
}

func printSummary(results []EvalLogLine) {
	if len(results) == 0 {
		fmt.Println("\nNo evaluation results found.")
		return
	}

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("EVALUATION RESULTS SUMMARY")
	fmt.Println(strings.Repeat("=", 80))

	passed := 0
	failed := 0
	var failures []EvalLogLine

	for _, result := range results {
		if result.Pass {
			passed++
		} else {
			failed++
			failures = append(failures, result)
		}
	}

	fmt.Printf("\nTotal Tests: %d\n", len(results))
	fmt.Printf("Passed:      %d\n", passed)
	fmt.Printf("Failed:      %d\n", failed)

	if len(failures) > 0 {
		fmt.Println("\n" + strings.Repeat("-", 80))
		fmt.Println("FAILED EVALUATIONS:")
		fmt.Println(strings.Repeat("-", 80))

		for i, failure := range failures {
			fmt.Printf("\n%d. %s\n", i+1, failure.Name)
			fmt.Printf("   Rubric: %s\n", truncateString(failure.Rubric, 70))
			fmt.Printf("   Score:  %.2f\n", failure.Score)
			if failure.Reasoning != "" {
				fmt.Printf("   Reason: %s\n", truncateString(failure.Reasoning, 70))
			}
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 80))
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func findEvalsFile() (string, error) {
	// Look for evals.jsonl in current directory and parent directories
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		evalFile := filepath.Join(dir, "evals.jsonl")
		if _, err := os.Stat(evalFile); err == nil {
			return evalFile, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("evals.jsonl not found in current directory or parent directories")
}

func loadResults(filename string, showOnlyFailures bool) ([]EvalLogLine, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("error opening file %s: %v", filename, err)
	}
	defer file.Close()

	var results []EvalLogLine
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var result EvalLogLine
		if err := json.Unmarshal([]byte(line), &result); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing line: %v\n", err)
			continue
		}

		// Filter based on failures-only flag
		if showOnlyFailures && result.Pass {
			continue
		}

		results = append(results, result)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %v", err)
	}

	return results, nil
}
