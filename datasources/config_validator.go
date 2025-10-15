// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"fmt"
	"net/url"
	"strings"
)

// validateConfig validates a configuration struct to prevent injection attacks
func validateConfig(config *Config) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	if len(config.AllowedDomains) == 0 {
		return fmt.Errorf("AllowedDomains cannot be empty - this prevents SSRF attacks")
	}

	for _, domain := range config.AllowedDomains {
		if domain == "" || domain == "*" || strings.Contains(domain, "*") {
			return fmt.Errorf("AllowedDomains cannot contain wildcards or be empty: %q", domain)
		}

		if strings.Contains(domain, "://") {
			return fmt.Errorf("AllowedDomains should contain domain names only, not URLs: %q", domain)
		}
	}

	for i, source := range config.Sources {
		if err := validateSourceConfig(&source, i); err != nil {
			return fmt.Errorf("source[%d] (%s): %w", i, source.Name, err)
		}
	}

	return nil
}

// validateSourceConfig validates an individual source configuration
func validateSourceConfig(source *SourceConfig, index int) error {
	if source.Name == "" {
		return fmt.Errorf("source name cannot be empty")
	}

	validProtocols := map[ProtocolType]bool{
		HTTPProtocolType:       true,
		GitHubAPIProtocolType:  true,
		MattermostProtocolType: true,
		ConfluenceProtocolType: true,
		UserVoiceProtocolType:  true,
		DiscourseProtocolType:  true,
		JiraProtocolType:       true,
		FileProtocolType:       true,
	}

	if !validProtocols[source.Protocol] {
		return fmt.Errorf("invalid protocol type: %q", source.Protocol)
	}

	validAuthTypes := map[string]bool{
		AuthTypeNone:   true,
		AuthTypeToken:  true,
		AuthTypeAPIKey: true,
	}

	if !validAuthTypes[source.Auth.Type] {
		return fmt.Errorf("invalid auth type: %q", source.Auth.Type)
	}

	for key, endpoint := range source.Endpoints {
		if endpoint == "" {
			continue // Empty endpoints are allowed
		}

		// If endpoint looks like a URL, validate it
		if strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://") {
			parsedURL, err := url.Parse(endpoint)
			if err != nil {
				return fmt.Errorf("endpoint %q has invalid URL %q: %w", key, endpoint, err)
			}

			// Enforce HTTPS except for localhost
			if parsedURL.Scheme == "http" {
				hostname := parsedURL.Hostname()
				if hostname != "localhost" && hostname != "127.0.0.1" && hostname != "::1" {
					return fmt.Errorf("endpoint %q must use https, not http: %q", key, endpoint)
				}
			}

			// Prevent private/internal IP ranges (basic check)
			hostname := parsedURL.Hostname()
			if strings.HasPrefix(hostname, "10.") ||
				strings.HasPrefix(hostname, "192.168.") ||
				strings.HasPrefix(hostname, "172.16.") ||
				strings.HasPrefix(hostname, "172.17.") ||
				strings.HasPrefix(hostname, "172.18.") ||
				strings.HasPrefix(hostname, "172.19.") ||
				strings.HasPrefix(hostname, "172.20.") ||
				strings.HasPrefix(hostname, "172.21.") ||
				strings.HasPrefix(hostname, "172.22.") ||
				strings.HasPrefix(hostname, "172.23.") ||
				strings.HasPrefix(hostname, "172.24.") ||
				strings.HasPrefix(hostname, "172.25.") ||
				strings.HasPrefix(hostname, "172.26.") ||
				strings.HasPrefix(hostname, "172.27.") ||
				strings.HasPrefix(hostname, "172.28.") ||
				strings.HasPrefix(hostname, "172.29.") ||
				strings.HasPrefix(hostname, "172.30.") ||
				strings.HasPrefix(hostname, "172.31.") ||
				strings.HasPrefix(hostname, "169.254.") {
				// Allow localhost/loopback for testing
				if hostname != "localhost" && hostname != "127.0.0.1" && hostname != "::1" {
					return fmt.Errorf("endpoint %q cannot use private IP range: %q", key, endpoint)
				}
			}
		}
	}

	if source.RateLimit.Enabled {
		if source.RateLimit.RequestsPerMinute < 1 || source.RateLimit.RequestsPerMinute > 10000 {
			return fmt.Errorf("RequestsPerMinute must be between 1 and 10000, got %d", source.RateLimit.RequestsPerMinute)
		}
		if source.RateLimit.BurstSize < 1 || source.RateLimit.BurstSize > 1000 {
			return fmt.Errorf("BurstSize must be between 1 and 1000, got %d", source.RateLimit.BurstSize)
		}
	}

	if source.MaxDocsPerCall < 1 || source.MaxDocsPerCall > 1000 {
		return fmt.Errorf("MaxDocsPerCall must be between 1 and 1000, got %d", source.MaxDocsPerCall)
	}

	return nil
}
