// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// MattermostAPIClient handles all HTTP API calls to Mattermost servers
type MattermostAPIClient struct {
	client *http.Client
	auth   AuthConfig
}

// NewMattermostAPIClient creates a new API client
func NewMattermostAPIClient(httpClient *http.Client) *MattermostAPIClient {
	return &MattermostAPIClient{
		client: httpClient,
		auth:   AuthConfig{Type: AuthTypeNone},
	}
}

// SetAuth configures authentication for API calls
func (c *MattermostAPIClient) SetAuth(auth AuthConfig) {
	c.auth = auth
}

// addAuthHeaders adds Mattermost API authentication headers
func (c *MattermostAPIClient) addAuthHeaders(req *http.Request) {
	switch c.auth.Type {
	case AuthTypeToken:
		if c.auth.Key != "" {
			req.Header.Set(HeaderAuthorization, AuthPrefixBearer+c.auth.Key)
		}
	case AuthTypeAPIKey:
		if c.auth.Key != "" {
			req.Header.Set(HeaderAuthorization, AuthPrefixBearer+c.auth.Key)
		}
	}
	req.Header.Set(HeaderUserAgent, UserAgentMattermostPM)
	req.Header.Set(HeaderContentType, AcceptJSON)
}

// GetTeams retrieves available teams from Mattermost API
func (c *MattermostAPIClient) GetTeams(ctx context.Context, baseURL string) ([]MattermostTeam, error) {
	url := BuildAPIURL(baseURL, "api/v4/teams")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	c.addAuthHeaders(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get teams: HTTP %d", resp.StatusCode)
	}

	var teams []MattermostTeam
	if err := json.NewDecoder(resp.Body).Decode(&teams); err != nil {
		return nil, err
	}

	return teams, nil
}

// GetTeamChannels retrieves all channels for a team
func (c *MattermostAPIClient) GetTeamChannels(ctx context.Context, baseURL, teamID string) ([]MattermostChannel, error) {
	url := BuildAPIURL(baseURL, "api/v4/teams", teamID, "channels")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	c.addAuthHeaders(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get team channels: HTTP %d", resp.StatusCode)
	}

	var channels []MattermostChannel
	if err := json.NewDecoder(resp.Body).Decode(&channels); err != nil {
		return nil, err
	}

	return channels, nil
}

// GetChannelByName retrieves a channel by its name within a team
func (c *MattermostAPIClient) GetChannelByName(ctx context.Context, baseURL, teamID, channelName string) (*MattermostChannel, error) {
	url := BuildAPIURL(baseURL, "api/v4/teams", teamID, "channels/name", channelName)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	c.addAuthHeaders(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get channel %s: HTTP %d", channelName, resp.StatusCode)
	}

	var channel MattermostChannel
	if err := json.NewDecoder(resp.Body).Decode(&channel); err != nil {
		return nil, err
	}

	return &channel, nil
}

// GetChannelPosts retrieves recent posts from a channel
func (c *MattermostAPIClient) GetChannelPosts(ctx context.Context, baseURL, channelID string, limit int) ([]MattermostPost, error) {
	url := BuildAPIURL(baseURL, fmt.Sprintf("api/v4/channels/%s/posts?per_page=%d", channelID, limit))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	c.addAuthHeaders(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get channel posts: HTTP %d", resp.StatusCode)
	}

	var postsResponse PostsResponse
	if err := json.NewDecoder(resp.Body).Decode(&postsResponse); err != nil {
		return nil, err
	}

	var posts []MattermostPost
	for _, postID := range postsResponse.Order {
		if post, exists := postsResponse.Posts[postID]; exists {
			posts = append(posts, post)
		}
	}

	return posts, nil
}

// SearchPosts searches for posts across the Mattermost instance
func (c *MattermostAPIClient) SearchPosts(ctx context.Context, baseURL, terms string, limit int) (*SearchPostsResponse, error) {
	searchURL := BuildAPIURL(baseURL, "api/v4/posts/search")

	searchParams := map[string]interface{}{
		"terms":                    terms,
		"is_or_search":             true,
		"time_zone_offset":         0,
		"include_deleted_channels": false,
		"page":                     0,
		"per_page":                 limit,
	}

	jsonData, err := json.Marshal(searchParams)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, searchURL, strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, err
	}

	c.addAuthHeaders(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("search failed: HTTP %d", resp.StatusCode)
	}

	var searchResponse SearchPostsResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResponse); err != nil {
		return nil, err
	}

	return &searchResponse, nil
}
