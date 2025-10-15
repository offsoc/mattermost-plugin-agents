// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/mattermost/mattermost-plugin-ai/datasources/queryutils"
	"github.com/mattermost/mattermost-plugin-ai/mmapi"
)

// MattermostProtocol implements the DataSourceProtocol for any Mattermost server instance
type MattermostProtocol struct {
	apiClient       *MattermostAPIClient
	transformer     *MattermostTransformer
	fallbackHandler *MattermostFallbackHandler
	rateLimiter     *RateLimiter
	pluginAPI       mmapi.Client
	topicAnalyzer   *TopicAnalyzer
}

// MattermostTeam represents a team from Mattermost API
type MattermostTeam struct {
	ID             string `json:"id"`
	CreateAt       int64  `json:"create_at"`
	UpdateAt       int64  `json:"update_at"`
	DeleteAt       int64  `json:"delete_at"`
	DisplayName    string `json:"display_name"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	Email          string `json:"email"`
	Type           string `json:"type"`
	CompanyName    string `json:"company_name"`
	AllowedDomains string `json:"allowed_domains"`
	InviteID       string `json:"invite_id"`
}

// MattermostChannel represents a channel from Mattermost API
type MattermostChannel struct {
	ID            string `json:"id"`
	CreateAt      int64  `json:"create_at"`
	UpdateAt      int64  `json:"update_at"`
	DeleteAt      int64  `json:"delete_at"`
	TeamID        string `json:"team_id"`
	Type          string `json:"type"`
	DisplayName   string `json:"display_name"`
	Name          string `json:"name"`
	Header        string `json:"header"`
	Purpose       string `json:"purpose"`
	LastPostAt    int64  `json:"last_post_at"`
	TotalMsgCount int64  `json:"total_msg_count"`
	CreatorID     string `json:"creator_id"`
}

// MattermostPost represents a post from Mattermost API
type MattermostPost struct {
	ID         string                 `json:"id"`
	CreateAt   int64                  `json:"create_at"`
	UpdateAt   int64                  `json:"update_at"`
	EditAt     int64                  `json:"edit_at"`
	DeleteAt   int64                  `json:"delete_at"`
	IsPinned   bool                   `json:"is_pinned"`
	UserID     string                 `json:"user_id"`
	ChannelID  string                 `json:"channel_id"`
	RootID     string                 `json:"root_id"`
	ParentID   string                 `json:"parent_id"`
	OriginalID string                 `json:"original_id"`
	Message    string                 `json:"message"`
	Type       string                 `json:"type"`
	Props      map[string]interface{} `json:"props"`
	Hashtags   string                 `json:"hashtags"`
	ReplyCount int64                  `json:"reply_count"`
}

// MattermostUser represents a user from Mattermost API
type MattermostUser struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	Nickname  string `json:"nickname"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Position  string `json:"position"`
}

// PostsResponse represents the response from posts API
type PostsResponse struct {
	Order []string                  `json:"order"`
	Posts map[string]MattermostPost `json:"posts"`
}

// SearchPostsResponse represents the response from posts search API
type SearchPostsResponse struct {
	Order   []string                  `json:"order"`
	Posts   map[string]MattermostPost `json:"posts"`
	Matches map[string][]string       `json:"matches,omitempty"`
}

// NewMattermostProtocol creates a new Mattermost protocol instance
func NewMattermostProtocol(httpClient *http.Client, pluginAPI mmapi.Client, fallbackDirectory string) *MattermostProtocol {
	return &MattermostProtocol{
		apiClient:       NewMattermostAPIClient(httpClient),
		transformer:     NewMattermostTransformer(),
		fallbackHandler: NewMattermostFallbackHandler(pluginAPI, fallbackDirectory),
		pluginAPI:       pluginAPI,
		topicAnalyzer:   NewTopicAnalyzer(),
	}
}

// Fetch retrieves documents from Mattermost server channels
func (m *MattermostProtocol) Fetch(ctx context.Context, request ProtocolRequest) ([]Doc, error) {
	source := request.Source

	EnsureRateLimiter(&m.rateLimiter, source.RateLimit)

	baseURL := source.Endpoints[EndpointBaseURL]
	if baseURL == "" {
		return nil, fmt.Errorf("missing required Mattermost configuration: base_url")
	}

	var allDocs []Doc

	if request.Topic != "" {
		searchDocs := m.searchPosts(ctx, baseURL, request.Topic, request.Limit, source.Name)
		if len(searchDocs) > 0 {
			searchDocs = FilterDocsByBooleanQuery(searchDocs, request.Topic)
			return searchDocs, nil
		}
	}

	for _, section := range request.Sections {
		if len(allDocs) >= request.Limit {
			break
		}

		channelDocs := m.fetchFromChannel(ctx, baseURL, section, request.Topic, request.Limit-len(allDocs), source.Name, source)
		allDocs = append(allDocs, channelDocs...)
	}

	allDocs = FilterDocsByBooleanQuery(allDocs, request.Topic)
	return allDocs, nil
}

// GetType returns the protocol type
func (m *MattermostProtocol) GetType() ProtocolType {
	return MattermostProtocolType
}

// SetAuth configures authentication for the protocol
func (m *MattermostProtocol) SetAuth(auth AuthConfig) {
	m.apiClient.SetAuth(auth)
}

// Close cleans up resources used by the protocol
func (m *MattermostProtocol) Close() error {
	CloseRateLimiter(&m.rateLimiter)
	return nil
}

// searchPosts searches for posts across the Mattermost instance
func (m *MattermostProtocol) searchPosts(ctx context.Context, baseURL, terms string, limit int, sourceName string) []Doc {
	if err := WaitRateLimiter(ctx, m.rateLimiter); err != nil {
		return nil
	}

	searchTerms := SimplifyBooleanQueryToKeywords(terms)

	searchResponse, err := m.apiClient.SearchPosts(ctx, baseURL, searchTerms, limit)
	if err != nil {
		if m.pluginAPI != nil {
			m.pluginAPI.LogWarn(sourceName+": search failed", "error", err.Error())
		}
		return nil
	}

	resultDocs := m.transformer.ConvertPostsToDoc(searchResponse.Posts, searchResponse.Order, baseURL, sourceName, terms)

	return resultDocs
}

// fetchFromChannel retrieves information about a specific channel or recent posts
func (m *MattermostProtocol) fetchFromChannel(ctx context.Context, baseURL, section, topic string, limit int, sourceName string, config SourceConfig) []Doc {
	channelName := m.mapSectionToChannelWithConfig(ctx, section, config, baseURL)

	if m.apiClient.auth.Type != AuthTypeNone && m.apiClient.auth.Key != "" {
		docs := m.fetchChannelPosts(ctx, baseURL, channelName, topic, limit, sourceName, section)
		if len(docs) > 0 {
			return docs
		}
	}

	if sourceName == SourceMattermostHub {
		return m.fallbackHandler.LoadHubMockData(section, sourceName, baseURL, channelName, topic)
	}

	return nil
}

// fetchChannelPosts retrieves actual posts from a channel (requires authentication)
func (m *MattermostProtocol) fetchChannelPosts(ctx context.Context, baseURL, channelName, topic string, limit int, sourceName, section string) []Doc {
	if err := WaitRateLimiter(ctx, m.rateLimiter); err != nil {
		return nil
	}

	teams, err := m.apiClient.GetTeams(ctx, baseURL)
	if err != nil || len(teams) == 0 {
		if m.pluginAPI != nil && err != nil {
			m.pluginAPI.LogWarn(sourceName+": get teams failed", "channel", channelName, "error", err)
		}
		return nil
	}

	var channel *MattermostChannel
	var teamName string
	for _, team := range teams {
		ch, chErr := m.apiClient.GetChannelByName(ctx, baseURL, team.ID, channelName)
		if chErr == nil && ch != nil {
			channel = ch
			teamName = team.Name
			break
		}
	}

	if channel == nil {
		if m.pluginAPI != nil {
			m.pluginAPI.LogWarn(sourceName+": channel not found", "channel", channelName)
		}
		return nil
	}

	posts, err := m.apiClient.GetChannelPosts(ctx, baseURL, channel.ID, limit)
	if err != nil {
		if m.pluginAPI != nil {
			m.pluginAPI.LogWarn(sourceName+": get posts failed", "channel", channelName, "error", err)
		}
		return nil
	}

	docs := m.transformer.FilterAndConvertPosts(posts, topic, channel, baseURL, sourceName, section, teamName)

	if len(docs) == 0 {
		return nil
	}

	return docs
}

// mapSectionToChannelWithConfig maps configuration sections to actual Mattermost channel names
// For community_forum: uses dynamic discovery → hardcoded fallback
// For other sources: uses configured mapping → hardcoded fallback
func (m *MattermostProtocol) mapSectionToChannelWithConfig(ctx context.Context, section string, config SourceConfig, baseURL string) string {
	if config.Name == SourceCommunityForum {
		if discoveredChannels, err := m.discoverRelevantChannels(ctx, baseURL); err == nil {
			if channels, exists := discoveredChannels[section]; exists && len(channels) > 0 {
				return channels[0]
			}
		}
	}

	if config.Name != SourceCommunityForum && config.ChannelMapping != nil {
		if channelName, exists := config.ChannelMapping[section]; exists {
			return channelName
		}
	}

	return m.mapSectionToChannel(section)
}

// mapSectionToChannel provides default channel mappings (fallback when no config available)
func (m *MattermostProtocol) mapSectionToChannel(section string) string {
	switch section {
	// Community Forum channels (community.mattermost.com) - Optimized based on actual content analysis
	case SectionFeatureRequests:
		return "ai-exchange" // AI and feature discussions
	case SectionTroubleshooting:
		return "bugs" // Primary channel for issues and improvements
	case SectionGeneral:
		return "ask-anything" // General questions and discussions
	case SectionBugs:
		return "bugs" // Bug reports and improvements
	case SectionAskAnything:
		return "ask-anything" // General Mattermost questions
	case SectionAskRnD:
		return "ask-r-and-d" // Questions to Product/Design/Engineering teams
	case SectionAIExchange:
		return "ai-exchange" // AI and feature discussions
	case SectionAccessibility:
		return "accessibility" // Accessibility improvements
	case SectionAPIv4:
		return "apiv4" // API discussions and feedback
	case SectionAPI:
		return "developers"
	case SectionMobile:
		return "mobile"
	// Mattermost Hub channels (hub.mattermost.com) - PM-focused channels
	case SectionContactSales:
		return "sales-feedback" // Sales team feedback and lost deal analysis
	case SectionCustomerFeedback:
		return "customer-success" // Customer success team insights and barriers
	default:
		return section
	}
}

// discoverRelevantChannels attempts to find channels that match PM-relevant patterns
func (m *MattermostProtocol) discoverRelevantChannels(ctx context.Context, baseURL string) (map[string][]string, error) {
	discoveredChannels := make(map[string][]string)

	teams, err := m.apiClient.GetTeams(ctx, baseURL)
	if err != nil {
		return nil, err
	}

	patterns := m.buildSectionPatterns()

	for _, team := range teams {
		channels, err := m.apiClient.GetTeamChannels(ctx, baseURL, team.ID)
		if err != nil {
			continue
		}

		for _, channel := range channels {
			if channel.Type != "O" {
				continue
			}

			channelNameLower := strings.ToLower(channel.Name)
			channelDisplayNameLower := strings.ToLower(channel.DisplayName)

			for section, keywords := range patterns {
				for _, keyword := range keywords {
					if strings.Contains(channelNameLower, keyword) ||
						strings.Contains(channelDisplayNameLower, keyword) {
						discoveredChannels[section] = append(discoveredChannels[section], channel.Name)
						break
					}
				}
			}
		}
	}

	return discoveredChannels, nil
}

// ValidateSearchSyntax tests search queries against the Mattermost search API to validate syntax
func (m *MattermostProtocol) ValidateSearchSyntax(ctx context.Context, request ProtocolRequest) (*SyntaxValidationResult, error) {
	result := &SyntaxValidationResult{
		OriginalQuery:    request.Topic,
		IsValidSyntax:    true,
		SyntaxErrors:     []string{},
		SupportsFeatures: []string{"simple terms", "quotes", "from:", "in:", "hashtags"},
	}

	if request.Topic == "" {
		result.RecommendedQuery = "mobile"
		return result, nil
	}

	baseURL := request.Source.Endpoints[EndpointBaseURL]
	if baseURL == "" {
		result.IsValidSyntax = false
		result.SyntaxErrors = append(result.SyntaxErrors, "Mattermost search requires base_url configuration")
		result.RecommendedQuery = request.Topic
		return result, nil
	}

	searchCount := m.testMattermostSearchQuery(ctx, baseURL, request.Topic)
	result.TestResultCount = searchCount

	if searchCount == 0 {
		simplifiedQuery := m.simplifyMattermostQuery(request.Topic)
		simpleCount := m.testMattermostSearchQuery(ctx, baseURL, simplifiedQuery)

		if simpleCount > 0 {
			result.IsValidSyntax = false
			result.SyntaxErrors = append(result.SyntaxErrors,
				fmt.Sprintf("Complex query returned 0 results, but simplified query returned %d results", simpleCount))
			result.RecommendedQuery = simplifiedQuery
		} else {
			verySimpleQuery := "mobile"
			verySimpleCount := m.testMattermostSearchQuery(ctx, baseURL, verySimpleQuery)
			if verySimpleCount > 0 {
				result.IsValidSyntax = false
				result.SyntaxErrors = append(result.SyntaxErrors, "Query may be too complex for Mattermost search")
				result.RecommendedQuery = verySimpleQuery
			} else {
				result.SyntaxErrors = append(result.SyntaxErrors, "No search results found - may require authentication or instance has no content")
				result.RecommendedQuery = request.Topic
			}
		}
	} else {
		result.RecommendedQuery = request.Topic
	}

	return result, nil
}

// testMattermostSearchQuery performs a lightweight search test to validate query syntax
func (m *MattermostProtocol) testMattermostSearchQuery(ctx context.Context, baseURL, query string) int {
	if err := WaitRateLimiter(ctx, m.rateLimiter); err != nil {
		return 0
	}

	searchResponse, err := m.apiClient.SearchPosts(ctx, baseURL, query, 1)
	if err != nil {
		return 0
	}

	return len(searchResponse.Order)
}

// simplifyMattermostQuery creates a Mattermost-friendly version of a complex query
func (m *MattermostProtocol) simplifyMattermostQuery(query string) string {
	return queryutils.SimplifyQueryToKeywords(query, 2, "mobile")
}

// buildSectionPatterns builds section patterns using centralized topic analysis
func (m *MattermostProtocol) buildSectionPatterns() map[string][]string {
	basePatterns := map[string][]string{
		SectionFeatureRequests:  {"feature", "idea", "request", "suggestion"},
		SectionTroubleshooting:  {"trouble", "support", "help", "issue"},
		SectionGeneral:          {"general", "town-hall", "announcements"},
		SectionAPI:              {"developer", "api", "dev", "technical"},
		SectionContactSales:     {"sales", "business", "enterprise", "deals"},
		SectionCustomerFeedback: {"customer", "success", "feedback", "support"},
	}

	commonTopics := []string{"mobile", "web", "desktop", "server", "enterprise", "security", "performance"}
	for _, topic := range commonTopics {
		synonyms := m.topicAnalyzer.GetTopicSynonyms(topic)
		if len(synonyms) > 0 {
			switch topic {
			case "mobile":
				basePatterns[SectionMobile] = append([]string{topic}, synonyms...)
			case "api", "developer":
				basePatterns[SectionAPI] = append(basePatterns[SectionAPI], synonyms...)
			case "enterprise", "security":
				basePatterns[SectionContactSales] = append(basePatterns[SectionContactSales], synonyms...)
			default:
				basePatterns[SectionGeneral] = append(basePatterns[SectionGeneral], synonyms...)
			}
		}
	}

	return basePatterns
}
