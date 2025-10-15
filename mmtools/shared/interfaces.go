// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package shared

// MockLoader provides mock data for internal search results during testing/development
type MockLoader interface {
	IsEnabled() bool
	LoadMockResponse(key string) (string, bool)
}
