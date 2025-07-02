// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package mcpserver

import (
	"context"
	"fmt"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

// AuthenticationProvider handles authentication for MCP requests
type AuthenticationProvider interface {
	// ValidateAuth validates authentication from context and returns user ID
	ValidateAuth(ctx context.Context) (string, error)

	// GetAuthenticatedMattermostClient returns an authenticated Mattermost client
	GetAuthenticatedMattermostClient(ctx context.Context) (*model.Client4, error)
}

// TokenAuthenticationProvider provides PAT token authentication for STDIO transport
type TokenAuthenticationProvider struct {
	serverURL string
	token     string
	logger    mlog.LoggerIFace
}

// NewTokenAuthenticationProvider creates a new PAT token authentication provider for STDIO transport
func NewTokenAuthenticationProvider(serverURL, token string, logger mlog.LoggerIFace) *TokenAuthenticationProvider {
	return &TokenAuthenticationProvider{
		serverURL: serverURL,
		token:     token,
		logger:    logger,
	}
}

// ValidateAuth validates authentication and returns user ID
func (p *TokenAuthenticationProvider) ValidateAuth(ctx context.Context) (string, error) {
	// Get authenticated client (reuses the authentication logic)
	client, err := p.GetAuthenticatedMattermostClient(ctx)
	if err != nil {
		return "", err
	}

	// Get current user to validate token
	user, _, err := client.GetMe(ctx, "")
	if err != nil {
		p.logger.Error("failed to validate token", mlog.Err(err))
		return "", fmt.Errorf("invalid authentication token: %w", err)
	}

	p.logger.Debug("validated token for user", mlog.String("user_id", user.Id), mlog.String("username", user.Username))

	return user.Id, nil
}

// GetAuthenticatedMattermostClient returns an authenticated Mattermost client
func (p *TokenAuthenticationProvider) GetAuthenticatedMattermostClient(ctx context.Context) (*model.Client4, error) {
	if p.token == "" {
		return nil, fmt.Errorf("no authentication token available")
	}

	// Create client with configured token
	client := model.NewAPIv4Client(p.serverURL)
	client.SetToken(p.token)

	return client, nil
}

// OAuthAuthenticationProvider will provide OAuth authentication for HTTP transport
// TODO: Implement when HTTP transport is added
type OAuthAuthenticationProvider struct {
	clientID     string
	clientSecret string
	redirectURL  string
	serverURL    string
	logger       mlog.LoggerIFace
}

// NewOAuthAuthenticationProvider creates a new OAuth authentication provider
// TODO: Implement when HTTP transport is added
func NewOAuthAuthenticationProvider(clientID, clientSecret, redirectURL, serverURL string, logger mlog.LoggerIFace) *OAuthAuthenticationProvider {
	return &OAuthAuthenticationProvider{
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURL:  redirectURL,
		serverURL:    serverURL,
		logger:       logger,
	}
}

// ValidateAuth validates OAuth authentication from context and returns user ID
// TODO: Implement when HTTP transport is added
func (p *OAuthAuthenticationProvider) ValidateAuth(ctx context.Context) (string, error) {
	return "", fmt.Errorf("OAuth authentication not yet implemented")
}

// GetAuthenticatedMattermostClient returns an OAuth-authenticated Mattermost client
// TODO: Implement when HTTP transport is added
func (p *OAuthAuthenticationProvider) GetAuthenticatedMattermostClient(ctx context.Context) (*model.Client4, error) {
	return nil, fmt.Errorf("OAuth authentication not yet implemented")
}
