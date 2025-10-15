// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import "strings"

// BuildAPIURL constructs an API URL from base URL and path segments.
// It ensures proper slash handling by trimming trailing slashes from the base
// and leading/trailing slashes from path segments, then joining them.
//
// Example:
//
//	BuildAPIURL("https://api.example.com/", "users", "/123/")
//	// Returns: "https://api.example.com/users/123"
func BuildAPIURL(baseURL string, pathSegments ...string) string {
	if baseURL == "" {
		return ""
	}

	base := strings.TrimRight(baseURL, "/")

	if len(pathSegments) == 0 {
		return base
	}

	for i, segment := range pathSegments {
		pathSegments[i] = strings.Trim(segment, "/")
	}

	return base + "/" + strings.Join(pathSegments, "/")
}
