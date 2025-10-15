// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mattermost/mattermost-plugin-ai/mmapi"
)

// UserVoiceProtocol implements the DataSourceProtocol for UserVoice using REST API v2
type UserVoiceProtocol struct {
	client          *http.Client
	auth            AuthConfig
	rateLimiter     *RateLimiter
	pluginAPI       mmapi.Client
	topicAnalyzer   *TopicAnalyzer
	universalScorer *UniversalRelevanceScorer
	apiKey          string // UserVoice API key (client_id)
	Subdomain       string // UserVoice subdomain (e.g., "mattermost")
}

// UserVoiceSuggestion represents a feature request from UserVoice
type UserVoiceSuggestion struct {
	ID          string
	Title       string
	Description string
	Status      string
	Votes       int
	Comments    int
	Category    string
	URL         string
	CreatedAt   string
	UpdatedAt   string
	AuthorName  string
	Priority    string
}

// UserVoiceAPIResponse represents the API response from UserVoice v2 API
type UserVoiceAPIResponse struct {
	Suggestions  []UserVoiceAPISuggestion `json:"suggestions"`
	ResponseData UserVoiceResponseData    `json:"response_data"`
	Page         int                      `json:"page"`
	TotalPages   int                      `json:"total_pages"`
	TotalRecords int                      `json:"total_records"`
	PerPage      int                      `json:"per_page"`
}

// UserVoiceAPISuggestion represents a suggestion from the API
type UserVoiceAPISuggestion struct {
	ID            int                `json:"id"`
	Title         string             `json:"title"`
	Text          string             `json:"text"`
	FormattedText string             `json:"formatted_text"`
	Status        UserVoiceStatus    `json:"status"`
	VoteCount     int                `json:"vote_count"`
	CommentsCount int                `json:"comments_count"`
	Category      *UserVoiceCategory `json:"category"`
	URL           string             `json:"url"`
	CreatedAt     string             `json:"created_at"`
	UpdatedAt     string             `json:"updated_at"`
	Creator       *UserVoiceUser     `json:"creator"`
	State         string             `json:"state"`
}

// UserVoiceStatus represents suggestion status
type UserVoiceStatus struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

// UserVoiceCategory represents a suggestion category
type UserVoiceCategory struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// UserVoiceUser represents a UserVoice user
type UserVoiceUser struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// UserVoiceResponseData contains pagination metadata
type UserVoiceResponseData struct {
	Page         int `json:"page"`
	TotalRecords int `json:"total_records"`
	PerPage      int `json:"per_page"`
	TotalPages   int `json:"total_pages"`
}

// NewUserVoiceProtocol creates a new UserVoice protocol instance
func NewUserVoiceProtocol(httpClient *http.Client, pluginAPI mmapi.Client) *UserVoiceProtocol {
	apiKey := os.Getenv("USERVOICE_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("MM_AI_USERVOICE_TOKEN")
	}

	return &UserVoiceProtocol{
		client:          httpClient,
		rateLimiter:     NewRateLimiter(DefaultRequestsPerMinuteCommunity, DefaultBurstSizeCommunity),
		pluginAPI:       pluginAPI,
		topicAnalyzer:   NewTopicAnalyzer(),
		universalScorer: NewUniversalRelevanceScorer(),
		apiKey:          apiKey,
		Subdomain:       "mattermost", // Default subdomain
	}
}

// Fetch retrieves documents from UserVoice using REST API
func (u *UserVoiceProtocol) Fetch(ctx context.Context, request ProtocolRequest) ([]Doc, error) {
	if err := WaitRateLimiter(ctx, u.rateLimiter); err != nil {
		return nil, fmt.Errorf("rate limiting error: %w", err)
	}

	if u.apiKey == "" {
		return nil, fmt.Errorf("UserVoice API key not configured. Set USERVOICE_API_KEY or MM_AI_USERVOICE_TOKEN environment variable")
	}

	baseURL := request.Source.Endpoints[EndpointBaseURL]
	if baseURL == "" {
		baseURL = "https://mattermost.uservoice.com"
	}

	u.ExtractSubdomain(baseURL)

	var allDocs []Doc
	page := 1
	perPage := 100

	for len(allDocs) < request.Limit {
		suggestions, hasMore, err := u.FetchSuggestionsAPI(ctx, request.Topic, page, perPage)
		if err != nil {
			if u.pluginAPI != nil {
				u.pluginAPI.LogWarn(request.Source.Name+": API fetch failed", "page", page, "error", err)
			}
			break
		}

		if len(suggestions) == 0 {
			break
		}

		for i := range suggestions {
			if len(allDocs) >= request.Limit {
				break
			}

			if request.Topic != "" && !u.topicAnalyzer.IsTopicRelevantContent(
				suggestions[i].Title+" "+suggestions[i].Description, request.Topic) {
				continue
			}

			doc := u.suggestionToDoc(suggestions[i], request.Source.Name)

			if !u.universalScorer.IsUniversallyAcceptable(doc.Content, doc.Title, request.Source.Name, request.Topic) {
				continue
			}

			allDocs = append(allDocs, doc)
		}

		if !hasMore {
			break
		}
		page++
	}

	return allDocs, nil
}

// FetchSuggestionsAPI retrieves suggestions from UserVoice REST API v2
// Note: UserVoice has both public and admin APIs. The admin API requires OAuth.
// For public access, we try the suggestions endpoint with client_id authentication
func (u *UserVoiceProtocol) FetchSuggestionsAPI(ctx context.Context, query string, page, perPage int) ([]UserVoiceSuggestion, bool, error) {
	apiURL := fmt.Sprintf("https://%s.uservoice.com/api/v2/suggestions", u.Subdomain)

	params := url.Values{}
	params.Add("client_id", u.apiKey)
	params.Add(QueryParamPage, strconv.Itoa(page))
	params.Add(QueryParamPerPage, strconv.Itoa(perPage))
	params.Add(QueryParamSort, "votes") // Sort by votes descending

	if query != "" {
		params.Add("query", query)
	}

	fullURL := apiURL + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, false, fmt.Errorf("failed to create request: %w", err)
	}

	u.addAuthHeaders(req)

	resp, err := u.client.Do(req)
	if err != nil {
		return nil, false, fmt.Errorf("failed to fetch API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return nil, false, fmt.Errorf("unauthorized: UserVoice API key is invalid or missing. Set USERVOICE_API_KEY environment variable")
	}

	if resp.StatusCode == 404 {
		return nil, false, fmt.Errorf("UserVoice API endpoint not found. The API may have changed or require admin access")
	}

	if resp.StatusCode != 200 {
		return nil, false, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var apiResponse UserVoiceAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, false, fmt.Errorf("failed to parse API response: %w", err)
	}

	suggestions := make([]UserVoiceSuggestion, 0, len(apiResponse.Suggestions))
	for _, apiSugg := range apiResponse.Suggestions {
		suggestion := UserVoiceSuggestion{
			ID:          strconv.Itoa(apiSugg.ID),
			Title:       apiSugg.Title,
			Description: apiSugg.Text,
			Status:      u.normalizeStatus(apiSugg.Status.Name),
			Votes:       apiSugg.VoteCount,
			Comments:    apiSugg.CommentsCount,
			URL:         apiSugg.URL,
			CreatedAt:   apiSugg.CreatedAt,
			UpdatedAt:   apiSugg.UpdatedAt,
		}

		if apiSugg.Category != nil {
			suggestion.Category = apiSugg.Category.Name
		}

		if apiSugg.Creator != nil {
			suggestion.AuthorName = apiSugg.Creator.Name
		}

		suggestion.Priority = u.mapVotesToPriority(apiSugg.VoteCount)

		suggestions = append(suggestions, suggestion)
	}

	hasMore := page < apiResponse.TotalPages
	return suggestions, hasMore, nil
}

// ExtractSubdomain extracts the subdomain from a UserVoice URL
func (u *UserVoiceProtocol) ExtractSubdomain(baseURL string) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return
	}

	parts := strings.Split(parsed.Hostname(), ".")
	if len(parts) > 0 {
		u.Subdomain = parts[0]
	}
}

// mapVotesToPriority converts vote count to priority: high (100+), medium (50-99), low (10-49), minimal (<10)
func (u *UserVoiceProtocol) mapVotesToPriority(votes int) string {
	switch {
	case votes >= 100:
		return "high"
	case votes >= 50:
		return "medium"
	case votes >= 10:
		return "low"
	default:
		return "minimal"
	}
}

// GetType returns the protocol type
func (u *UserVoiceProtocol) GetType() ProtocolType {
	return UserVoiceProtocolType
}

// SetAuth configures authentication for the protocol
func (u *UserVoiceProtocol) SetAuth(auth AuthConfig) {
	u.auth = auth
	if auth.Key != "" {
		u.apiKey = auth.Key
	}
}

// Close cleans up resources used by the protocol
func (u *UserVoiceProtocol) Close() error {
	CloseRateLimiter(&u.rateLimiter)
	return nil
}

// normalizeStatus normalizes status strings: "completed"/"done"/"shipped" -> "completed",
// "in progress"/"development"/"started" -> "in_progress", "planned"/"roadmap" -> "planned",
// "declined"/"rejected"/"closed" -> "declined", "under review" -> "under_review", default -> "open"
func (u *UserVoiceProtocol) normalizeStatus(status string) string {
	status = strings.ToLower(strings.TrimSpace(status))

	switch {
	case strings.Contains(status, StatusCompleted) || strings.Contains(status, "done") || strings.Contains(status, StatusShipped):
		return StatusCompleted
	case strings.Contains(status, "in progress") || strings.Contains(status, "development") || strings.Contains(status, "working") || strings.Contains(status, "started"):
		return StatusInProgress
	case strings.Contains(status, StatusPlanned) || strings.Contains(status, "roadmap") || strings.Contains(status, "accepted"):
		return StatusPlanned
	case strings.Contains(status, StatusDeclined) || strings.Contains(status, StatusRejected) || strings.Contains(status, StatusClosed) || strings.Contains(status, "wont"):
		return StatusDeclined
	case strings.Contains(status, "under review") || strings.Contains(status, "reviewing") || strings.Contains(status, "consideration"):
		return "under_review"
	default:
		return StatusOpen
	}
}

// addAuthHeaders adds UserVoice API authentication headers
func (u *UserVoiceProtocol) addAuthHeaders(req *http.Request) {
	req.Header.Set(HeaderAccept, AcceptJSON)
	req.Header.Set(HeaderUserAgent, UserAgentMattermostPM)

	if u.auth.Type == AuthTypeAPIKey || u.auth.Type == AuthTypeToken {
		if u.apiKey != "" && !strings.Contains(req.URL.String(), "client_id=") {
			req.Header.Set(HeaderAuthorization, AuthPrefixBearer+u.apiKey)
		}
	}
}

// ValidateSearchSyntax validates search queries for UserVoice API
func (u *UserVoiceProtocol) ValidateSearchSyntax(ctx context.Context, request ProtocolRequest) (*SyntaxValidationResult, error) {
	result := &SyntaxValidationResult{
		OriginalQuery:    request.Topic,
		IsValidSyntax:    true,
		SyntaxErrors:     []string{},
		SupportsFeatures: []string{"text search", "sort by votes", "pagination"},
	}

	if request.Topic == "" {
		result.RecommendedQuery = "mobile"
		return result, nil
	}

	if u.apiKey == "" {
		result.IsValidSyntax = false
		result.SyntaxErrors = append(result.SyntaxErrors, "UserVoice API key not configured")
		result.RecommendedQuery = request.Topic
		return result, nil
	}

	suggestions, _, err := u.FetchSuggestionsAPI(ctx, request.Topic, 1, 1)
	if err != nil {
		result.IsValidSyntax = false
		result.SyntaxErrors = append(result.SyntaxErrors, err.Error())
		result.TestResultCount = 0
	} else {
		result.TestResultCount = len(suggestions)
	}

	result.RecommendedQuery = request.Topic
	return result, nil
}

// suggestionToDoc converts a UserVoice suggestion to a Doc, mapping votes to priority labels
// (high: 100+, medium: 50-99, low: 10-49) and extracting category, status, engagement metrics
func (u *UserVoiceProtocol) suggestionToDoc(suggestion UserVoiceSuggestion, sourceName string) Doc {
	var contentParts []string

	contentParts = append(contentParts, fmt.Sprintf("**Title:** %s", suggestion.Title))

	if suggestion.Description != "" {
		contentParts = append(contentParts, fmt.Sprintf("\n**Description:**\n%s", suggestion.Description))
	}

	if suggestion.Status != "" {
		contentParts = append(contentParts, fmt.Sprintf("\n**Status:** %s", suggestion.Status))
	}

	if suggestion.Votes > 0 {
		contentParts = append(contentParts, fmt.Sprintf("**Votes:** %d", suggestion.Votes))
	}

	if suggestion.Comments > 0 {
		contentParts = append(contentParts, fmt.Sprintf("**Comments:** %d", suggestion.Comments))
	}

	if suggestion.Category != "" {
		contentParts = append(contentParts, fmt.Sprintf("**Category:** %s", suggestion.Category))
	}

	if suggestion.Priority != "" {
		contentParts = append(contentParts, fmt.Sprintf("**Priority:** %s", suggestion.Priority))
	}

	if suggestion.AuthorName != "" {
		contentParts = append(contentParts, fmt.Sprintf("**Author:** %s", suggestion.AuthorName))
	}

	content := strings.Join(contentParts, "\n")

	labels := []string{}
	if suggestion.Status != "" {
		labels = append(labels, fmt.Sprintf("status:%s", suggestion.Status))
	}
	if suggestion.Category != "" {
		labels = append(labels, fmt.Sprintf("category:%s", strings.ToLower(strings.ReplaceAll(suggestion.Category, " ", "_"))))
	}
	if suggestion.Priority != "" {
		labels = append(labels, fmt.Sprintf("priority:%s", suggestion.Priority))
	}

	switch {
	case suggestion.Votes >= 100:
		labels = append(labels, "votes:100+")
	case suggestion.Votes >= 50:
		labels = append(labels, "votes:50-99")
	case suggestion.Votes >= 10:
		labels = append(labels, "votes:10-49")
	}

	if suggestion.Comments > 20 {
		labels = append(labels, "high_engagement")
	} else if suggestion.Comments > 5 {
		labels = append(labels, "medium_engagement")
	}

	if suggestion.CreatedAt != "" {
		if createdAt, err := time.Parse(time.RFC3339, suggestion.CreatedAt); err == nil {
			daysAgo := int(time.Since(createdAt).Hours() / 24)
			if recencyLabel := FormatRecencyLabel(daysAgo); recencyLabel != "" {
				labels = append(labels, recencyLabel+"_created")
			}
		}
	}

	if suggestion.UpdatedAt != "" {
		if updatedAt, err := time.Parse(time.RFC3339, suggestion.UpdatedAt); err == nil {
			daysAgo := int(time.Since(updatedAt).Hours() / 24)
			if recencyLabel := FormatRecencyLabel(daysAgo); recencyLabel != "" {
				labels = append(labels, recencyLabel+"_updated")
			}
		}
	}

	return Doc{
		Title:        suggestion.Title,
		Content:      content,
		URL:          suggestion.URL,
		Source:       sourceName,
		Section:      SectionFeatureRequests,
		Labels:       labels,
		Author:       suggestion.AuthorName,
		CreatedDate:  suggestion.CreatedAt,
		LastModified: suggestion.UpdatedAt,
	}
}
