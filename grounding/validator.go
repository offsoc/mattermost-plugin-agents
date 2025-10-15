// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package grounding

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ValidateCitationsWithClaims validates citations and their metadata claims
// Extracts claims from response, validates citations, then validates claims
func ValidateCitationsWithClaims(response string, citations []Citation, refIndex *ReferenceIndex, roleMetadata RoleMetadata) []Citation {
	var citationsWithClaims []Citation
	if roleMetadata != nil {
		extractor := NewClaimExtractor(roleMetadata)
		citationsWithClaims = extractor.ExtractMetadataClaims(response, citations)
	} else {
		citationsWithClaims = citations
	}

	validated := ValidateCitations(citationsWithClaims, refIndex)

	validated = ValidateMetadataClaims(validated, refIndex)

	return validated
}

// ValidateCitations validates citations using exact-match reference index
// Eliminates false positives from substring matching
func ValidateCitations(citations []Citation, refIndex *ReferenceIndex) []Citation {
	validated := make([]Citation, len(citations))

	for i, citation := range citations {
		validated[i] = citation

		if citation.Type == CitationMetadata {
			validated[i].IsValid = true
			validated[i].ValidationStatus = ValidationGrounded
			validated[i].ValidationDetails = "Metadata always grounded"
			continue
		}

		switch citation.Type {
		case CitationJiraTicket:
			if _, found := refIndex.JiraTickets[citation.Value]; found {
				validated[i].IsValid = true
				validated[i].ValidationStatus = ValidationGrounded
				validated[i].ValidationDetails = "Found in reference index"
				continue
			}

		case CitationGitHub:
			if _, found := refIndex.GitHubIssues[citation.Value]; found {
				validated[i].IsValid = true
				validated[i].ValidationStatus = ValidationGrounded
				validated[i].ValidationDetails = "Found in reference index"
				continue
			}

		case CitationURL:
			normalized := normalizeURL(citation.Value)
			if refIndex.URLs[normalized] {
				validated[i].IsValid = true
				validated[i].ValidationStatus = ValidationGrounded
				validated[i].ValidationDetails = "Found in reference index"
				continue
			}

		case CitationZendesk:
			ticketID := extractZendeskTicketID(citation.Value)
			if ticketID != "" {
				if _, found := refIndex.ZendeskTickets[ticketID]; found {
					validated[i].IsValid = true
					validated[i].ValidationStatus = ValidationGrounded
					validated[i].ValidationDetails = "Found in reference index"
					continue
				}
			}

		case CitationProductBoard, CitationUserVoice:
		}

		validated[i].IsValid = false
		validated[i].ValidationStatus = ValidationNotChecked
		validated[i].ValidationDetails = "Not in tool results, needs API check"
	}

	return validated
}

// normalizeURL removes protocol and trailing slashes for comparison
func normalizeURL(rawURL string) string {
	normalized := strings.ToLower(rawURL)
	normalized = strings.TrimPrefix(normalized, "https://")
	normalized = strings.TrimPrefix(normalized, "http://")
	normalized = strings.TrimSuffix(normalized, "/")
	return normalized
}

// extractZendeskTicketID extracts ticket ID from Zendesk citation
func extractZendeskTicketID(citation string) string {
	// Zendesk citations like "12345" or "https://support.mattermost.com/hc/requests/12345"
	parts := strings.Split(citation, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return citation
}

// ValidateCitationsWithAPI validates citations with optional API verification for ungrounded citations
func ValidateCitationsWithAPI(
	ctx context.Context,
	citations []Citation,
	refIndex *ReferenceIndex,
	apiClients *APIClients,
) []Citation {
	results := make([]Citation, len(citations))

	for i, citation := range citations {
		citation = ValidateCitations([]Citation{citation}, refIndex)[0]

		if citation.ValidationStatus == ValidationGrounded {
			results[i] = citation
			continue
		}

		if apiClients != nil && citation.ValidationStatus == ValidationNotChecked {
			citation = verifyViaAPI(ctx, citation, apiClients)
		}

		results[i] = citation
	}

	return results
}

// verifyViaAPI attempts to verify citation via external API
func verifyViaAPI(
	ctx context.Context,
	citation Citation,
	clients *APIClients,
) Citation {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var exists bool
	var err error

	switch citation.Type {
	case CitationJiraTicket:
		if clients.Jira != nil {
			exists, err = clients.Jira.IssueExists(ctx, citation.Value)
		} else {
			return citation
		}

	case CitationGitHub:
		if clients.GitHub != nil {
			owner, repo, number, parseErr := parseGitHubRef(citation.Value)
			if parseErr != nil {
				citation.ValidationStatus = ValidationAPIError
				citation.ValidationDetails = fmt.Sprintf("Parse error: %v", parseErr)
				citation.VerifiedViaAPI = true
				return citation
			}
			exists, err = clients.GitHub.IssueExists(ctx, owner, repo, number)
		} else {
			return citation
		}

	case CitationURL:
		exists, err = verifyURL(ctx, citation.Value)

	default:
		citation.ValidationStatus = ValidationNotChecked
		return citation
	}

	citation.VerifiedViaAPI = true

	if err != nil {
		citation.ValidationStatus = ValidationAPIError
		citation.ValidationDetails = fmt.Sprintf("API error: %v", err)
		return citation
	}

	if exists {
		citation.IsValid = true
		citation.ValidationStatus = ValidationUngroundedValid
		citation.ValidationDetails = "Verified via API"
	} else {
		citation.IsValid = false
		citation.ValidationStatus = ValidationFabricated
		citation.ValidationDetails = "Does not exist in external system"
	}

	return citation
}

// parseGitHubRef parses GitHub reference like "owner/repo#123" or "#123"
func parseGitHubRef(ref string) (owner, repo string, number int, err error) {
	if strings.Contains(ref, "/") && strings.Contains(ref, "#") {
		parts := strings.Split(ref, "/")
		if len(parts) != 2 {
			return "", "", 0, fmt.Errorf("invalid GitHub reference format: %s", ref)
		}
		owner = parts[0]
		repoParts := strings.Split(parts[1], "#")
		if len(repoParts) != 2 {
			return "", "", 0, fmt.Errorf("invalid GitHub reference format: %s", ref)
		}
		repo = repoParts[0]
		number, err = strconv.Atoi(repoParts[1])
		if err != nil {
			return "", "", 0, fmt.Errorf("invalid issue number: %s", repoParts[1])
		}
		return owner, repo, number, nil
	}

	if strings.HasPrefix(ref, "#") {
		return "", "", 0, fmt.Errorf("cannot verify short GitHub reference without owner/repo: %s", ref)
	}

	return "", "", 0, fmt.Errorf("invalid GitHub reference format: %s", ref)
}

// verifyURL checks URL accessibility with HEAD first, GET fallback
func verifyURL(ctx context.Context, urlStr string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, "HEAD", urlStr, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 400 {
			return true, nil
		}
		if resp.StatusCode == 404 {
			return false, nil
		}
	}

	req, err = http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create GET request: %w", err)
	}

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("GET request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return true, nil
	}
	if resp.StatusCode == 404 {
		return false, nil
	}

	return false, fmt.Errorf("HTTP %d", resp.StatusCode)
}

// ValidateURLAccessibility checks if ungrounded URL citations are actually accessible
// Uses parallel requests with semaphore to limit concurrency
func ValidateURLAccessibility(citations []Citation) []Citation {
	validated := make([]Citation, len(citations))
	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	copy(validated, citations)

	var urlIndices []int
	for i, citation := range citations {
		if citation.Type == CitationURL &&
			(citation.ValidationStatus == ValidationNotChecked || citation.ValidationStatus == "") {
			urlIndices = append(urlIndices, i)
		}
	}

	if len(urlIndices) == 0 {
		return validated
	}

	const maxConcurrent = 10
	semaphore := make(chan struct{}, maxConcurrent)
	done := make(chan bool, len(urlIndices))

	for _, idx := range urlIndices {
		go func(i int) {
			semaphore <- struct{}{}
			defer func() {
				<-semaphore
				done <- true
			}()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			req, err := http.NewRequestWithContext(ctx, "HEAD", validated[i].Value, nil)
			if err != nil {
				validated[i].ValidationStatus = ValidationUngroundedBroken
				validated[i].HTTPStatusCode = 0
				return
			}

			resp, err := client.Do(req)
			if err != nil {
				validated[i].ValidationStatus = ValidationUngroundedBroken
				validated[i].HTTPStatusCode = 0
				return
			}
			defer resp.Body.Close()

			validated[i].HTTPStatusCode = resp.StatusCode

			if resp.StatusCode >= 200 && resp.StatusCode < 400 {
				validated[i].ValidationStatus = ValidationUngroundedValid
			} else {
				validated[i].ValidationStatus = ValidationUngroundedBroken
			}
		}(idx)
	}

	for range urlIndices {
		<-done
	}

	return validated
}
