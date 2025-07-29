// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package mcpserver

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/mattermost/mattermost-plugin-ai/mcpserver/auth"
	"github.com/mattermost/mattermost-plugin-ai/mcpserver/tools"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

// MattermostMCPServer provides a high-level interface for creating an MCP server
// with Mattermost-specific tools and authentication
type MattermostMCPServer struct {
	mcpServer    *server.MCPServer
	authProvider auth.AuthenticationProvider
	logger       *mlog.Logger
	config       ServerConfig
}

// MattermostStdioMCPServer provides STDIO transport MCP server
type MattermostStdioMCPServer struct {
	*MattermostMCPServer
	config StdioConfig
}

// MattermostHTTPMCPServer provides HTTP transport MCP server
type MattermostHTTPMCPServer struct {
	*MattermostMCPServer
	config               HTTPConfig
	sseServer            *server.SSEServer
	streamableHTTPServer *server.StreamableHTTPServer
	httpMux              *http.ServeMux
	httpServer           *http.Server
}

// Serve starts the STDIO MCP server
func (s *MattermostStdioMCPServer) Serve() error {
	return s.serveStdio()
}

// Serve starts the HTTP MCP server
func (s *MattermostHTTPMCPServer) Serve() error {
	s.logger.Info("starting HTTP MCP server with SSE support",
		mlog.String("bind_addr", s.config.HTTPBindAddr),
		mlog.Int("port", s.config.HTTPPort),
		mlog.String("server_url", s.config.GetMMServerURL()),
	)

	// Start the custom HTTP server with OAuth endpoints
	return s.httpServer.ListenAndServe()
}

// NewStdioServer creates a new STDIO transport MCP server
func NewStdioServer(serverURL, token string, logger *mlog.Logger, devMode bool) (*MattermostStdioMCPServer, error) {
	if serverURL == "" {
		return nil, fmt.Errorf("server URL cannot be empty")
	}
	if token == "" {
		return nil, fmt.Errorf("personal access token cannot be empty")
	}

	if logger == nil {
		var err error
		logger, err = createDefaultLogger()
		if err != nil {
			return nil, fmt.Errorf("failed to create default logger: %w", err)
		}
	}

	config := StdioConfig{
		BaseConfig: BaseConfig{
			MMServerURL: serverURL,
			DevMode:     devMode,
		},
		PersonalAccessToken: token,
	}

	mattermostServer := &MattermostStdioMCPServer{
		MattermostMCPServer: &MattermostMCPServer{
			logger: logger,
			config: config,
		},
		config: config,
	}

	// Create authentication provider
	mattermostServer.authProvider = auth.NewTokenAuthenticationProvider(serverURL, token, logger)

	// Create MCP server
	mattermostServer.mcpServer = server.NewMCPServer(
		"mattermost-mcp-server",
		"0.1.0",
		server.WithToolCapabilities(false),
		server.WithLogging(),
		server.WithRecovery(),
	)

	// Validate token at startup for STDIO
	if err := mattermostServer.authProvider.ValidateAuth(context.Background()); err != nil {
		return nil, fmt.Errorf("startup token validation failed: %w", err)
	}

	// Register tools
	mattermostServer.registerTools()

	return mattermostServer, nil
}

// NewHTTPServer creates a new HTTP transport MCP server
func NewHTTPServer(config HTTPConfig, logger *mlog.Logger) (*MattermostHTTPMCPServer, error) {
	if config.MMServerURL == "" {
		return nil, fmt.Errorf("server URL cannot be empty")
	}
	if config.HTTPPort <= 0 {
		return nil, fmt.Errorf("HTTP port must be greater than 0")
	}
	if config.HTTPBindAddr == "" {
		return nil, fmt.Errorf("HTTP bind address cannot be empty")
	}

	if logger == nil {
		var err error
		logger, err = createDefaultLogger()
		if err != nil {
			return nil, fmt.Errorf("failed to create default logger: %w", err)
		}
	}

	mattermostServer := &MattermostHTTPMCPServer{
		MattermostMCPServer: &MattermostMCPServer{
			logger: logger,
			config: config,
		},
		config: config,
	}

	// Create OAuth authentication provider
	mattermostServer.authProvider = auth.NewOAuthAuthenticationProvider(
		config.MMServerURL,
		config.MMServerURL, // OAuth issuer is the same as server URL
		logger,
	)

	// Create MCP server with authentication middleware
	mattermostServer.mcpServer = server.NewMCPServer(
		"mattermost-mcp-server",
		"0.1.0",
		server.WithToolCapabilities(false),
		server.WithLogging(),
		server.WithRecovery(),
		mattermostServer.withAuthenticationMiddleware(),
	)

	// Register tools
	mattermostServer.registerTools()

	// Create SSE server using the mcp-go library with authentication
	baseURL := fmt.Sprintf("http://%s:%d", config.HTTPBindAddr, config.HTTPPort)
	if config.SiteURL != "" {
		baseURL = config.SiteURL
	}

	// Create HTTP server with OAuth endpoints and MCP routing
	addr := fmt.Sprintf("%s:%d", config.HTTPBindAddr, config.HTTPPort)
	if err := mattermostServer.setupHTTPServerWithOAuth(baseURL, addr); err != nil {
		return nil, fmt.Errorf("failed to setup HTTP server: %w", err)
	}

	return mattermostServer, nil
}

// serveStdio starts the server using stdio transport
func (s *MattermostMCPServer) serveStdio() error {
	errorLogger := log.New(&mlogWriter{logger: s.logger}, "", 0)
	return server.ServeStdio(s.mcpServer, server.WithErrorLogger(errorLogger))
}

// registerTools registers all tools using the tool provider
func (s *MattermostMCPServer) registerTools() {
	toolProvider := tools.NewMattermostToolProvider(s.authProvider, s.logger, s.config.GetMMServerURL(), s.config.GetDevMode())
	toolProvider.ProvideTools(s.mcpServer)
}

// GetMCPServer returns the underlying MCP server for testing purposes
func (s *MattermostMCPServer) GetMCPServer() *server.MCPServer {
	return s.mcpServer
}

// createAuthContextFunc creates a context function for SSE authentication
func (s *MattermostHTTPMCPServer) createAuthContextFunc() server.SSEContextFunc {
	return func(ctx context.Context, r *http.Request) context.Context {
		// Add HTTP request to context so OAuth provider can access it
		return context.WithValue(ctx, auth.HTTPRequestContextKey, r)
	}
}

// createHTTPContextFunc creates a context function for Streamable HTTP authentication
func (s *MattermostHTTPMCPServer) createHTTPContextFunc() server.HTTPContextFunc {
	return func(ctx context.Context, r *http.Request) context.Context {
		// Add HTTP request to context so OAuth provider can access it
		return context.WithValue(ctx, auth.HTTPRequestContextKey, r)
	}
}

// withAuthenticationMiddleware creates authentication middleware for tool handlers
func (s *MattermostHTTPMCPServer) withAuthenticationMiddleware() server.ServerOption {
	return server.WithToolHandlerMiddleware(func(next server.ToolHandlerFunc) server.ToolHandlerFunc {
		return func(ctx context.Context, request mcp.CallToolRequest) (result *mcp.CallToolResult, err error) {
			// Validate authentication before tool execution
			if authErr := s.authProvider.ValidateAuth(ctx); authErr != nil {
				s.logger.Warn("authentication failed for tool call",
					mlog.String("tool", request.Params.Name),
					mlog.Err(authErr))
				return nil, fmt.Errorf("authentication failed: %w", authErr)
			}
			return next(ctx, request)
		}
	})
}

// setupHTTPServerWithOAuth sets up the HTTP server with OAuth metadata endpoints and MCP routing
func (s *MattermostHTTPMCPServer) setupHTTPServerWithOAuth(baseURL, addr string) error {
	// Create HTTP mux router
	s.httpMux = http.NewServeMux()

	// Add OAuth metadata endpoints
	s.addOAuthMetadataEndpoints()

	// Create SSE server for MCP communication (backwards compatibility)
	// Configure SSE server with custom endpoints to match our path stripping
	s.sseServer = server.NewSSEServer(
		s.mcpServer,
		server.WithBaseURL(baseURL),
		server.WithStaticBasePath(""),
		server.WithSSEEndpoint("/sse"),
		server.WithMessageEndpoint("/message"),
		server.WithUseFullURLForMessageEndpoint(false),
		server.WithSSEContextFunc(s.createAuthContextFunc()),
	)

	// Create Streamable HTTP server for MCP communication (new standard)
	s.streamableHTTPServer = server.NewStreamableHTTPServer(
		s.mcpServer,
		server.WithHTTPContextFunc(s.createHTTPContextFunc()),
	)

	// Setup MCP routes
	s.setupMCPRoutes()

	// Apply logging and security middleware to the mux
	mainHandler := s.loggingMiddleware(s.httpMux)
	secureHandler := s.securityMiddleware(mainHandler)

	// Create HTTP server with security middleware
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      secureHandler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return nil
}

// getResourceMetadataURL returns the URL for the protected resource metadata endpoint
func (s *MattermostHTTPMCPServer) getResourceMetadataURL() string {
	baseURL := s.config.GetMMServerURL()
	if s.config.SiteURL != "" {
		baseURL = s.config.SiteURL
	}
	return fmt.Sprintf("%s/.well-known/oauth-protected-resource", baseURL)
}

// getAllowedOrigins returns the list of allowed origins for CORS and DNS rebinding protection
func (s *MattermostHTTPMCPServer) getAllowedOrigins() []string {
	var origins []string

	// Add Mattermost server URL as allowed origin
	if mmURL := s.config.GetMMServerURL(); mmURL != "" {
		origins = append(origins, mmURL)
	}

	// Add configured site URL as allowed origin
	if siteURL := s.config.SiteURL; siteURL != "" {
		origins = append(origins, siteURL)
	}

	return origins
}

// validateOrigin validates the Origin header to prevent DNS rebinding attacks
func (s *MattermostHTTPMCPServer) validateOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		// No Origin header - allow for direct API calls (non-browser requests)
		// This is common for server-to-server communication
		return true
	}

	allowedOrigins := s.getAllowedOrigins()

	// Parse the origin URL to normalize it
	originURL, err := url.Parse(origin)
	if err != nil {
		s.logger.Warn("invalid origin header",
			mlog.String("origin", origin),
			mlog.Err(err))
		return false
	}

	// Normalize origin (remove default ports)
	normalizedOrigin := normalizeURL(originURL)

	// Check against allowed origins
	for _, allowedOrigin := range allowedOrigins {
		if allowedURL, err := url.Parse(allowedOrigin); err == nil {
			normalizedAllowed := normalizeURL(allowedURL)
			if normalizedOrigin == normalizedAllowed {
				return true
			}
		}
	}

	s.logger.Warn("origin not allowed",
		mlog.String("origin", origin),
		mlog.String("allowed_origins", strings.Join(allowedOrigins, ", ")))
	return false
}

// normalizeURL removes default ports and normalizes URL for comparison
func normalizeURL(u *url.URL) string {
	host := u.Host

	// Remove default ports
	if u.Scheme == "https" && strings.HasSuffix(host, ":443") {
		host = strings.TrimSuffix(host, ":443")
	} else if u.Scheme == "http" && strings.HasSuffix(host, ":80") {
		host = strings.TrimSuffix(host, ":80")
	}

	return fmt.Sprintf("%s://%s", u.Scheme, host)
}

// securityMiddleware applies security headers and validation
func (s *MattermostHTTPMCPServer) securityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Validate Origin header to prevent DNS rebinding attacks
		if !s.validateOrigin(r) {
			s.logger.Warn("request blocked due to invalid origin",
				mlog.String("origin", r.Header.Get("Origin")),
				mlog.String("remote_addr", r.RemoteAddr),
				mlog.String("user_agent", r.UserAgent()))

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error": {"code": -32603, "message": "Origin not allowed"}}`))
			return
		}

		// Add security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Security-Policy", "default-src 'none'")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Add CORS headers for allowed origins (already validated above)
		if origin := r.Header.Get("Origin"); origin != "" {
			// Origin was already validated in validateOrigin(), safe to set CORS headers
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		}

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware creates a standard HTTP logging middleware
func (s *MattermostHTTPMCPServer) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Create a response writer wrapper to capture status code
		recorder := &responseRecorder{ResponseWriter: w, statusCode: http.StatusOK}

		// Process the request
		next.ServeHTTP(recorder, r)

		// Log all requests with more detail for debugging
		s.logger.Debug("HTTP request processed",
			mlog.String("method", r.Method),
			mlog.String("path", r.URL.Path),
			mlog.String("query", r.URL.RawQuery),
			mlog.Int("status", recorder.statusCode),
			mlog.String("user_agent", r.UserAgent()),
			mlog.String("content_type", r.Header.Get("Content-Type")),
		)

		// Log 404s specifically to help debug routing issues
		if recorder.statusCode == 404 {
			s.logger.Warn("Route not found",
				mlog.String("method", r.Method),
				mlog.String("path", r.URL.Path),
				mlog.String("full_url", r.URL.String()),
			)
		}
	})
}

// responseRecorder is a wrapper around http.ResponseWriter to capture status code
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *responseRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

// Flush implements http.Flusher by delegating to the wrapped writer if it supports flushing
func (r *responseRecorder) Flush() {
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// requireAuth creates HTTP middleware that requires OAuth authentication
func (s *MattermostHTTPMCPServer) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Add HTTP request to context for OAuth provider
		ctx := context.WithValue(r.Context(), auth.HTTPRequestContextKey, r)

		if err := s.authProvider.ValidateAuth(ctx); err != nil {
			s.logger.Warn("authentication failed for MCP endpoint",
				mlog.String("path", r.URL.Path),
				mlog.Err(err))

			// Return 401 Unauthorized with WWW-Authenticate header (RFC 9728)
			resourceMetadataURL := s.getResourceMetadataURL()
			w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Bearer resource_metadata="%s"`, resourceMetadataURL))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error": {"code": -32600, "message": "Authentication required"}}`))
			return
		}

		// Set the authenticated context for downstream handlers
		r = r.WithContext(ctx)
		next(w, r)
	}
}

// setupMCPRoutes configures HTTP routes for MCP endpoints
func (s *MattermostHTTPMCPServer) setupMCPRoutes() {
	// New Streamable HTTP endpoint (MCP specification 2025-06-18)
	s.httpMux.HandleFunc("/mcp", s.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		s.streamableHTTPServer.ServeHTTP(w, r)
	}))

	// Legacy SSE endpoint for backwards compatibility with MCP clients that use Server-Sent Events
	// This is the standard MCP SSE transport endpoint - "/sse" is the correct path per MCP spec
	sseHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.logger.Info("MCP SSE request received",
			mlog.String("method", r.Method),
			mlog.String("path", r.URL.Path),
			mlog.String("query", r.URL.RawQuery),
			mlog.String("content_type", r.Header.Get("Content-Type")),
			mlog.String("accept", r.Header.Get("Accept")),
			mlog.String("user_agent", r.UserAgent()))

		// Call the SSE server directly - don't wrap with recorder as it breaks streaming
		s.sseServer.ServeHTTP(w, r)

		s.logger.Info("MCP SSE request completed",
			mlog.String("method", r.Method),
			mlog.String("path", r.URL.Path))
	})

	// Handle /sse and all sub-paths with authentication
	s.httpMux.Handle("/sse/", s.requireAuth(sseHandler.ServeHTTP))
	s.httpMux.Handle("/sse", s.requireAuth(sseHandler.ServeHTTP))

	// Handle /message endpoint for MCP SSE transport (backwards compatibility)
	s.httpMux.Handle("/message", s.requireAuth(sseHandler.ServeHTTP))

	// Default 404 handler for any other unmatched paths
	s.httpMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		s.logger.Debug("Request to unmatched path", mlog.String("path", r.URL.Path))
		http.NotFound(w, r)
	})
}
