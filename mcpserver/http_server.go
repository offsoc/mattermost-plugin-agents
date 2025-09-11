// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package mcpserver

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/server"
	"github.com/mattermost/mattermost-plugin-ai/mcpserver/auth"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

// MattermostHTTPMCPServer wraps MattermostMCPServer for HTTP transport
type MattermostHTTPMCPServer struct {
	*MattermostMCPServer
	config               HTTPConfig
	sseServer            *server.SSEServer
	streamableHTTPServer *server.StreamableHTTPServer
	httpMux              *http.ServeMux
	httpServer           *http.Server
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

	// Require site-url when binding to all interfaces for security
	// Per MCP spec: binding to 0.0.0.0 is discouraged, but if used, must have proper external URL
	if config.HTTPBindAddr == "0.0.0.0" && config.SiteURL == "" {
		return nil, fmt.Errorf("site-url is required when http-bind-addr is 0.0.0.0 to ensure secure origin validation and proper OAuth metadata")
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
		config.GetMMServerURL(),
		config.GetMMInternalServerURL(),
		config.GetMMServerURL(), // OAuth issuer is the external server URL
		logger,
	)

	// Create MCP server
	mattermostServer.mcpServer = server.NewMCPServer(
		"mattermost-mcp-server",
		"0.1.0",
		server.WithToolCapabilities(false),
		server.WithLogging(),
		server.WithRecovery(),
	)

	mattermostServer.registerTools()

	baseURL := fmt.Sprintf("http://%s:%d", config.HTTPBindAddr, config.HTTPPort)
	if config.SiteURL != "" {
		baseURL = config.SiteURL
	}

	// Create HTTP server with OAuth endpoints and MCP routing
	addr := fmt.Sprintf("%s:%d", config.HTTPBindAddr, config.HTTPPort)
	
	// Create HTTP mux router
	mattermostServer.httpMux = http.NewServeMux()

	// Add OAuth metadata endpoints
	mattermostServer.addOAuthMetadataEndpoints()

	// Create SSE server for MCP communication (backwards compatibility)
	// Configure SSE server with custom endpoints to match our path stripping
	mattermostServer.sseServer = server.NewSSEServer(
		mattermostServer.mcpServer,
		server.WithBaseURL(baseURL),
		server.WithStaticBasePath(""),
		server.WithSSEEndpoint("/sse"),
		server.WithMessageEndpoint("/message"),
		server.WithUseFullURLForMessageEndpoint(false),
		server.WithSSEContextFunc(mattermostServer.createAuthContextFunc()),
	)

	// Create Streamable HTTP server for MCP communication (new standard)
	// Configure with SSE support for GET requests per MCP specification
	mattermostServer.streamableHTTPServer = server.NewStreamableHTTPServer(
		mattermostServer.mcpServer,
		server.WithHTTPContextFunc(mattermostServer.createHTTPContextFunc()),
	)

	// Setup MCP routes
	mattermostServer.setupMCPRoutes()

	// Apply logging and security middleware to the mux
	mainHandler := mattermostServer.loggingMiddleware(mattermostServer.httpMux)
	secureHandler := mattermostServer.securityMiddleware(mainHandler)

	// Create HTTP server with security middleware
	mattermostServer.httpServer = &http.Server{
		Addr:         addr,
		Handler:      secureHandler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return mattermostServer, nil
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


// GetTestHandler returns the HTTP handler for testing purposes
func (s *MattermostHTTPMCPServer) GetTestHandler() http.Handler {
	if s.httpServer != nil {
		return s.httpServer.Handler
	}
	return s.httpMux
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
// Per MCP spec: "Servers MUST validate the Origin header on all incoming connections to prevent DNS rebinding attacks"
func (s *MattermostHTTPMCPServer) getAllowedOrigins() []string {
	var origins []string

	// Add Mattermost server URL as allowed origin
	if mmURL := s.config.GetMMServerURL(); mmURL != "" {
		origins = append(origins, mmURL)
	}

	// Add configured site URL as allowed origin (for reverse proxy scenarios)
	if siteURL := s.config.SiteURL; siteURL != "" {
		origins = append(origins, siteURL)
	}

	// For localhost binding (recommended by MCP spec), allow legitimate localhost origins
	// For 0.0.0.0 binding (discouraged by MCP spec), respect the required site-url
	// The site-url requirement is enforced in NewHTTPServer validation
	if s.config.HTTPBindAddr == "127.0.0.1" || s.config.HTTPBindAddr == "0.0.0.0" {
		// site-url is required and already added to origins above, so we respect that for external access
		// Also allow localhost access for convenience (e.g., Docker port mapping)
		localhostURL := fmt.Sprintf("http://localhost:%d", s.config.HTTPPort)
		localhost127URL := fmt.Sprintf("http://127.0.0.1:%d", s.config.HTTPPort)
		origins = append(origins, localhostURL, localhost127URL)
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

	s.logger.Warn("origin not in allowed list",
		mlog.String("origin", origin),
		mlog.Any("allowed_origins", allowedOrigins))
	return false
}

// normalizeURL normalizes a URL by removing default ports and converting to lowercase
func normalizeURL(u *url.URL) string {
	host := strings.ToLower(u.Hostname())
	port := u.Port()

	// Remove default ports
	if (u.Scheme == "http" && port == "80") || (u.Scheme == "https" && port == "443") {
		port = ""
	}

	if port != "" {
		host = fmt.Sprintf("%s:%s", host, port)
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
			_, _ = w.Write([]byte(`{"error": {"code": -32001, "message": "Origin not allowed"}}`))
			return
		}

		// Add security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Permitted-Cross-Domain-Policies", "none")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		// CSP policy allows connections for MCP protocol while preventing most attacks
		w.Header().Set("Content-Security-Policy", "default-src 'none'; connect-src 'self'; frame-ancestors 'none'")

		// Add CORS headers for allowed origins (already validated above)
		if origin := r.Header.Get("Origin"); origin != "" {
			// Origin was already validated in validateOrigin(), safe to set CORS headers
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, Accept, Cache-Control, MCP-Protocol-Version, Mcp-Session-Id, Last-Event-ID")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Max-Age", "86400")
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
		recorder := &responseRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
			logger:         s.logger,
			requestPath:    r.URL.Path,
		}

		// Log the request
		s.logger.Debug("HTTP request received",
			mlog.String("method", r.Method),
			mlog.String("path", r.URL.Path),
			mlog.String("remote_addr", r.RemoteAddr))

		// Call the next handler
		next.ServeHTTP(recorder, r)

		// Log the response
		s.logger.Debug("HTTP request completed",
			mlog.String("method", r.Method),
			mlog.String("path", r.URL.Path),
			mlog.Int("status", recorder.statusCode))
	})
}

// responseRecorder wraps http.ResponseWriter to capture the status code
type responseRecorder struct {
	http.ResponseWriter
	statusCode    int
	headerWritten bool
	logger        *mlog.Logger
	requestPath   string
}

// Flush implements http.Flusher for SSE streaming support
func (r *responseRecorder) Flush() {
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	if r.headerWritten {
		return
	}

	r.statusCode = statusCode
	r.headerWritten = true
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *responseRecorder) Write(data []byte) (int, error) {
	return r.ResponseWriter.Write(data)
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

// setupMCPRoutes sets up the MCP communication routes
func (s *MattermostHTTPMCPServer) setupMCPRoutes() {
	// New Streamable HTTP endpoint for MCP over HTTP standard
	s.httpMux.HandleFunc("/mcp", s.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		s.streamableHTTPServer.ServeHTTP(w, r)
	}))

	// SSE handler
	sseHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.sseServer.ServeHTTP(w, r)
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
