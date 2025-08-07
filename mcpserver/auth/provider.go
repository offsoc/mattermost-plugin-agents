// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

// Context keys for passing HTTP requests through context
type contextKey string

const (
	HTTPRequestContextKey contextKey = "http_request"
)

// AuthenticationProvider handles authentication for MCP requests
type AuthenticationProvider interface {
	// ValidateAuth validates authentication from context (may contain HTTP request)
	ValidateAuth(ctx context.Context) error

	// GetAuthenticatedMattermostClient returns an authenticated Mattermost client from context
	GetAuthenticatedMattermostClient(ctx context.Context) (*model.Client4, error)
}

// TokenAuthenticationProvider provides PAT token authentication for STDIO transport
type TokenAuthenticationProvider struct {
	mmServerURL string // Mattermost server URL for API communication
	token       string
	logger      mlog.LoggerIFace
}

// NewTokenAuthenticationProvider creates a new PAT token authentication provider for STDIO transport
// Uses internalURL for API communication if provided, otherwise falls back to externalURL
func NewTokenAuthenticationProvider(externalURL, internalURL, token string, logger mlog.LoggerIFace) *TokenAuthenticationProvider {
	// Use internal URL for API communication if provided, otherwise fallback to external URL
	mmServerURL := internalURL
	if mmServerURL == "" {
		mmServerURL = externalURL
	}

	return &TokenAuthenticationProvider{
		mmServerURL: mmServerURL,
		token:       token,
		logger:      logger,
	}
}

// ValidateAuth validates authentication
func (p *TokenAuthenticationProvider) ValidateAuth(ctx context.Context) error {
	// Get authenticated client and validate token (single GetMe call)
	_, err := p.GetAuthenticatedMattermostClient(ctx)
	return err
}

// GetAuthenticatedMattermostClient returns an authenticated Mattermost client
func (p *TokenAuthenticationProvider) GetAuthenticatedMattermostClient(ctx context.Context) (*model.Client4, error) {
	if p.token == "" {
		return nil, fmt.Errorf("no authentication token available")
	}

	// Create client with configured token
	client := model.NewAPIv4Client(p.mmServerURL)
	client.SetToken(p.token)

	// Validate token by getting current user (single validation call)
	user, _, err := client.GetMe(ctx, "")
	if err != nil {
		p.logger.Error("failed to validate token", mlog.Err(err))
		return nil, fmt.Errorf("invalid authentication token: %w", err)
	}

	p.logger.Debug("validated token for user", mlog.String("user_id", user.Id), mlog.String("username", user.Username))

	return client, nil
}

// OAuthAuthenticationProvider provides OAuth authentication for HTTP transport
// As a resource server, we only need to validate tokens using Mattermost's API
type OAuthAuthenticationProvider struct {
	mmServerURL string // Mattermost server URL for API communication
	issuer      string
	logger      mlog.LoggerIFace
}

// NewOAuthAuthenticationProvider creates a new OAuth authentication provider for resource server
// Uses internalURL for API communication if provided, otherwise falls back to externalURL
func NewOAuthAuthenticationProvider(externalURL, internalURL, issuer string, logger mlog.LoggerIFace) *OAuthAuthenticationProvider {
	// Use internal URL for API communication if provided, otherwise fallback to external URL
	mmServerURL := internalURL
	if mmServerURL == "" {
		mmServerURL = externalURL
	}

	return &OAuthAuthenticationProvider{
		mmServerURL: mmServerURL,
		issuer:      issuer,
		logger:      logger,
	}
}

// ValidateAuth validates OAuth authentication from context
func (p *OAuthAuthenticationProvider) ValidateAuth(ctx context.Context) error {
	// Get authenticated client, which handles all validation
	_, err := p.GetAuthenticatedMattermostClient(ctx)
	return err
}

// GetAuthenticatedMattermostClient returns an OAuth-authenticated Mattermost client from context
func (p *OAuthAuthenticationProvider) GetAuthenticatedMattermostClient(ctx context.Context) (*model.Client4, error) {
	// Parse and validate OAuth token from context
	token, err := p.parseAndValidateOAuthToken(ctx)
	if err != nil {
		return nil, err
	}

	// Create client and set OAuth token
	client := model.NewAPIv4Client(p.mmServerURL)
	client.SetOAuthToken(token)

	return client, nil
}

// parseAndValidateOAuthToken extracts and validates OAuth token from context, returning token and user info
func (p *OAuthAuthenticationProvider) parseAndValidateOAuthToken(ctx context.Context) (string, error) {
	// Extract HTTP request from context
	httpReq, ok := ctx.Value(HTTPRequestContextKey).(*http.Request)
	if !ok || httpReq == nil {
		return "", fmt.Errorf("OAuth provider requires HTTP request in context")
	}

	// Extract Bearer token from Authorization header
	authHeader := httpReq.Header.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("missing authorization header")
	}

	// Check for Bearer token
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return "", fmt.Errorf("invalid authorization header format, expected Bearer token")
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" {
		return "", fmt.Errorf("empty bearer token")
	}

	// TODO: This is where we will call the token introspection endpoint or get user from in-memory cache
	// For now, we're skipping validation and returning the token with nil user

	return token, nil
}
