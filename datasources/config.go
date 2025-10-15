// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// LoadConfig loads configuration from a file with environment variable fallbacks
func LoadConfig(configPath string) (*Config, error) {
	config, err := loadFromFile(configPath)
	if err != nil {
		return nil, err
	}

	// Apply environment variable fallbacks for sensitive data
	ApplyEnvironmentFallbacks(config)

	return config, nil
}

// CreateDefaultConfig creates a secure default configuration with all sources disabled
// Environment variables are automatically applied for authentication tokens
func CreateDefaultConfig() *Config {
	config := &Config{
		Sources: []SourceConfig{
			{
				Name:     SourceMattermostDocs,
				Enabled:  false,
				Protocol: HTTPProtocolType,
				Endpoints: map[string]string{
					EndpointBaseURL:        MattermostDocsURL,
					EndpointAdmin:          DocsAdminPath,
					EndpointDeveloper:      DocsDeveloperPath,
					EndpointAPI:            DocsAPIPath,
					EndpointMobile:         DocsMobilePath,
					EndpointMobileApps:     DocsMobileAppsPath,
					EndpointMobileStrategy: DocsMobileStrategyPath,
				},
				Auth:           AuthConfig{Type: AuthTypeToken},
				Sections:       []string{SectionAdmin, SectionDeveloper, SectionAPI, SectionMobile, SectionMobileApps, SectionMobileStrategy},
				MaxDocsPerCall: DefaultMaxDocsPerCallHTTP,
				RateLimit: RateLimitConfig{
					RequestsPerMinute: DefaultRequestsPerMinuteHTTP,
					BurstSize:         DefaultBurstSizeHTTP,
					Enabled:           true,
				},
			},
			{
				Name:     SourceMattermostHandbook,
				Enabled:  false,
				Protocol: HTTPProtocolType,
				Endpoints: map[string]string{
					EndpointBaseURL:    MattermostHandbookURL,
					SectionCompany:     HandbookCompanyPath,
					SectionOperations:  HandbookOperationsPath,
					SectionEngineering: HandbookEngineeringPath,
					SectionPeople:      HandbookPeoplePath,
					SectionMarketing:   HandbookMarketingPath,
					SectionSales:       HandbookSalesPath,
					SectionSupport:     HandbookSupportPath,
					SectionSecurity:    HandbookSecurityPath,
				},
				Auth:           AuthConfig{Type: AuthTypeNone},
				Sections:       []string{SectionCompany, SectionOperations, SectionEngineering, SectionPeople, SectionMarketing, SectionSales, SectionSupport, SectionSecurity},
				MaxDocsPerCall: DefaultMaxDocsPerCallHTTP,
				RateLimit: RateLimitConfig{
					RequestsPerMinute: DefaultRequestsPerMinuteHTTP,
					BurstSize:         DefaultBurstSizeHTTP,
					Enabled:           true,
				},
			},
			{
				Name:     SourceMattermostForum,
				Enabled:  false,
				Protocol: DiscourseProtocolType,
				Endpoints: map[string]string{
					EndpointBaseURL: MattermostForumURL,
				},
				Auth:           AuthConfig{Type: AuthTypeNone},
				Sections:       []string{SectionAnnouncements, SectionFAQ, SectionForumTroubleshooting, SectionUserFeedback, SectionCopilotAI, SectionRecipes},
				MaxDocsPerCall: DefaultMaxDocsPerCallCommunity,
				RateLimit: RateLimitConfig{
					RequestsPerMinute: DefaultRequestsPerMinuteCommunity,
					BurstSize:         DefaultBurstSizeCommunity,
					Enabled:           true,
				},
			},
			{
				Name:     SourceMattermostBlog,
				Enabled:  false,
				Protocol: HTTPProtocolType,
				Endpoints: map[string]string{
					EndpointBaseURL:      MattermostBlogURL,
					SectionBlogPosts:     BlogPlatformPath,
					SectionTechnical:     BlogEngineeringPath,
					SectionAnnouncements: BlogCommunityPath,
				},
				Auth:           AuthConfig{Type: AuthTypeNone},
				Sections:       []string{SectionBlogPosts, SectionTechnical, SectionAnnouncements},
				MaxDocsPerCall: DefaultMaxDocsPerCallHTTP,
				RateLimit: RateLimitConfig{
					RequestsPerMinute: DefaultRequestsPerMinuteHTTP,
					BurstSize:         DefaultBurstSizeHTTP,
					Enabled:           true,
				},
			},
			{
				Name:     SourceMattermostNewsroom,
				Enabled:  false,
				Protocol: HTTPProtocolType,
				Endpoints: map[string]string{
					EndpointBaseURL:      MattermostNewsroomURL,
					SectionNews:          NewsroomNewsPath,
					SectionPressReleases: NewsroomPressReleasePath,
					SectionMediaKit:      NewsroomPressKitPath,
				},
				Auth:           AuthConfig{Type: AuthTypeNone},
				Sections:       []string{SectionNews, SectionPressReleases, SectionMediaKit},
				MaxDocsPerCall: DefaultMaxDocsPerCallHTTP,
				RateLimit: RateLimitConfig{
					RequestsPerMinute: DefaultRequestsPerMinuteHTTP,
					BurstSize:         DefaultBurstSizeHTTP,
					Enabled:           true,
				},
			},
			{
				Name:     SourceGitHubRepos,
				Enabled:  false,
				Protocol: GitHubAPIProtocolType,
				Endpoints: map[string]string{
					EndpointOwner: GitHubOwnerMattermost,
					EndpointRepos: GitHubReposList,
				},
				Auth:           AuthConfig{Type: AuthTypeToken, Key: ""},
				Sections:       []string{SectionIssues, SectionReleases, SectionPulls, SectionCode},
				MaxDocsPerCall: DefaultMaxDocsPerCallGitHub,
				RateLimit: RateLimitConfig{
					RequestsPerMinute: DefaultRequestsPerMinuteGitHub,
					BurstSize:         DefaultBurstSizeGitHub,
					Enabled:           true,
				},
			},
			{
				Name:     SourceCommunityForum,
				Enabled:  true,
				Protocol: MattermostProtocolType,
				Endpoints: map[string]string{
					EndpointBaseURL: CommunityForumURL,
				},
				Auth:           AuthConfig{Type: AuthTypeToken},
				Sections:       []string{SectionFeatureRequests, SectionTroubleshooting, SectionGeneral, SectionBugs, SectionAskAnything, SectionAskRnD, SectionAIExchange, SectionAccessibility, SectionAPIv4},
				MaxDocsPerCall: DefaultMaxDocsPerCallCommunity,
				RateLimit: RateLimitConfig{
					RequestsPerMinute: DefaultRequestsPerMinuteCommunity,
					BurstSize:         DefaultBurstSizeCommunity,
					Enabled:           true,
				},
				// ChannelMapping: nil, // Community forum uses dynamic discovery, not custom mapping
			},
			{
				Name:     SourceMattermostHub,
				Enabled:  false,
				Protocol: MattermostProtocolType,
				Endpoints: map[string]string{
					EndpointBaseURL: MattermostHubURL,
				},
				Auth:           AuthConfig{Type: AuthTypeToken},
				Sections:       []string{SectionContactSales, SectionCustomerFeedback},
				MaxDocsPerCall: DefaultMaxDocsPerCallCommunity,
				RateLimit: RateLimitConfig{
					RequestsPerMinute: DefaultRequestsPerMinuteCommunity,
					BurstSize:         DefaultBurstSizeCommunity,
					Enabled:           true,
				},
			},
			{
				Name:     SourceConfluenceDocs,
				Enabled:  false,
				Protocol: ConfluenceProtocolType,
				Endpoints: map[string]string{
					EndpointBaseURL: ConfluenceURL,
					EndpointSpaces:  ConfluenceSpaces,
					EndpointEmail:   "",
				},
				Auth:           AuthConfig{Type: AuthTypeAPIKey, Key: ""},
				Sections:       []string{SectionProductRequirements, SectionMarketResearch, SectionFeatureSpecs, SectionRoadmaps, SectionCompetitiveAnalysis, SectionCustomerInsights},
				MaxDocsPerCall: DefaultMaxDocsPerCallHTTP,
				RateLimit: RateLimitConfig{
					RequestsPerMinute: DefaultRequestsPerMinuteConfluence,
					BurstSize:         DefaultBurstSizeConfluence,
					Enabled:           true,
				},
			},
			{
				Name:     SourcePluginMarketplace,
				Enabled:  false,
				Protocol: HTTPProtocolType,
				Endpoints: map[string]string{
					EndpointBaseURL:     IntegrationsURL,
					SectionPlugins:      "/",
					SectionIntegrations: "/",
				},
				Auth:           AuthConfig{Type: AuthTypeNone},
				Sections:       []string{SectionPlugins, SectionIntegrations},
				MaxDocsPerCall: DefaultMaxDocsPerCallHTTP,
				RateLimit: RateLimitConfig{
					RequestsPerMinute: DefaultRequestsPerMinuteHTTP,
					BurstSize:         DefaultBurstSizeHTTP,
					Enabled:           true,
				},
			},
			// UserVoice (feature_requests) - DISABLED due to lack of API access
			// Data is corrupted (HTML scraping artifacts). Re-enable when proper API key is available.
			// To export/update data with API: Set USERVOICE_API_KEY env var and run `make export-uservoice`
			{
				Name:     SourceFeatureRequests,
				Enabled:  false, // DISABLED - no API access
				Protocol: FileProtocolType,
				Endpoints: map[string]string{
					EndpointFilePath: "uservoice_suggestions.json",
				},
				Auth:           AuthConfig{Type: AuthTypeNone},
				Sections:       []string{SectionFeatureRequests},
				MaxDocsPerCall: DefaultMaxDocsPerCallFile,
				RateLimit: RateLimitConfig{
					RequestsPerMinute: DefaultRequestsPerMinuteFile,
					BurstSize:         DefaultBurstSizeFile,
					Enabled:           true,
				},
			},
			{
				Name:     SourceJiraDocs,
				Enabled:  false,
				Protocol: JiraProtocolType,
				Endpoints: map[string]string{
					EndpointBaseURL: JiraURL,
				},
				Auth:           AuthConfig{Type: AuthTypeAPIKey, Key: ""},
				Sections:       []string{SectionBug, SectionTask, SectionStory, SectionEpic, SectionSpike, SectionSubtask},
				MaxDocsPerCall: DefaultMaxDocsPerCallJira,
				RateLimit: RateLimitConfig{
					RequestsPerMinute: DefaultRequestsPerMinuteJira,
					BurstSize:         DefaultBurstSizeJira,
					Enabled:           true,
				},
			},
			{
				Name:     SourceProductBoardFeatures,
				Enabled:  true,
				Protocol: FileProtocolType,
				Endpoints: map[string]string{
					EndpointFilePath: "productboard-features.json",
				},
				Auth:           AuthConfig{Type: AuthTypeNone},
				Sections:       []string{SectionFeatures, SectionDelivered, SectionIdeas, SectionInProgress},
				MaxDocsPerCall: DefaultMaxDocsPerCallFile,
				RateLimit: RateLimitConfig{
					RequestsPerMinute: DefaultRequestsPerMinuteFile,
					BurstSize:         DefaultBurstSizeFile,
					Enabled:           true,
				},
			},
			{
				Name:     SourceZendeskTickets,
				Enabled:  true,
				Protocol: FileProtocolType,
				Endpoints: map[string]string{
					EndpointFilePath: "zendesk_tickets.txt",
				},
				Auth:           AuthConfig{Type: AuthTypeNone},
				Sections:       []string{SectionGeneral},
				MaxDocsPerCall: DefaultMaxDocsPerCallFile,
				RateLimit: RateLimitConfig{
					RequestsPerMinute: DefaultRequestsPerMinuteFile,
					BurstSize:         DefaultBurstSizeFile,
					Enabled:           true,
				},
			},
		},
		AllowedDomains: DefaultAllowedDomains,
		GitHubToken:    "",
		CacheTTL:       DefaultCacheTTL,
	}

	// Apply environment variable fallbacks for authentication tokens
	ApplyEnvironmentFallbacks(config)

	return config
}

// CreateEnabledConfig creates a configuration with all sources enabled
// This is useful for testing and development where all data sources should be available
// Callers should set FallbackDirectory after creation if needed
func CreateEnabledConfig() *Config {
	config := CreateDefaultConfig()
	for i := range config.Sources {
		config.Sources[i].Enabled = true
	}
	return config
}

// IsEnabled checks if external docs are globally enabled
func (c *Config) IsEnabled() bool {
	if c == nil {
		return false
	}

	for _, source := range c.Sources {
		if source.Enabled {
			return true
		}
	}
	return false
}

// GetEnabledSources returns only the enabled sources
func (c *Config) GetEnabledSources() []SourceConfig {
	var enabled []SourceConfig
	for _, source := range c.Sources {
		if source.Enabled {
			enabled = append(enabled, source)
		}
	}
	return enabled
}

// EnableSource enables a specific source by name for testing
func (c *Config) EnableSource(sourceName string) bool {
	for i := range c.Sources {
		if c.Sources[i].Name == sourceName {
			c.Sources[i].Enabled = true
			return true
		}
	}
	return false
}

// EnableSources enables multiple sources by name for testing
func (c *Config) EnableSources(sourceNames ...string) {
	for _, name := range sourceNames {
		c.EnableSource(name)
	}
}

// loadFromFile loads configuration from a JSON file
func loadFromFile(configPath string) (*Config, error) {
	if configPath == "" {
		return CreateDefaultConfig(), nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return CreateDefaultConfig(), nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	if config.CacheTTL == 0 {
		config.CacheTTL = DefaultCacheTTL
	}

	return &config, nil
}

// ApplyEnvironmentFallbacks applies environment variable fallbacks for sensitive configuration
func ApplyEnvironmentFallbacks(config *Config) {
	// GitHub token fallback
	if config.GitHubToken == "" {
		if envToken := os.Getenv(EnvGitHubToken); envToken != "" {
			config.GitHubToken = envToken
		}
	}

	// Source-specific auth token fallbacks
	for i, source := range config.Sources {
		if source.Auth.Key == "" && source.Auth.Type != AuthTypeNone {
			envVar := fmt.Sprintf(EnvSourceTokenPrefix+"%s"+EnvSourceTokenSuffix, strings.ToUpper(source.Name))
			envToken := os.Getenv(envVar)

			switch {
			case envToken != "":
				config.Sources[i].Auth.Key = envToken
			case source.Name == SourceJiraDocs:
				// Also check for generic Jira token for consistency with MM tools
				jiraToken := os.Getenv("MM_AI_JIRA_TOKEN")
				if jiraToken != "" {
					config.Sources[i].Auth.Key = jiraToken
				}
			case source.Protocol == GitHubAPIProtocolType:
				// For GitHub sources, fall back to config.GitHubToken if available
				if config.GitHubToken != "" {
					config.Sources[i].Auth.Key = config.GitHubToken
				}
			}
		}
	}
}
