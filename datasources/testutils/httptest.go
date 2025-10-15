// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package testutils

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// NewMockHTTPServer creates a test HTTP server with automatic cleanup.
// The server will be automatically closed when the test completes, even if it panics.
//
// Example:
//
//	server := testutils.NewMockHTTPServer(t, func(w http.ResponseWriter, r *http.Request) {
//	    w.WriteHeader(http.StatusOK)
//	    w.Write([]byte(`{"status": "ok"}`))
//	})
//	// No need for defer server.Close() - handled automatically
func NewMockHTTPServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server
}
