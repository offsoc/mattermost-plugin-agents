// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

//go:build integration
// +build integration

package datasources

import (
	"net/http"
	"testing"
)

func TestDiscourseProtocol_ComplexBooleanQueries(t *testing.T) {
	protocol := NewDiscourseProtocol(&http.Client{}, nil)

	source := SourceConfig{
		Name:     SourceMattermostForum,
		Protocol: DiscourseProtocolType,
		Endpoints: map[string]string{
			EndpointBaseURL: MattermostForumURL,
		},
		Sections: []string{SectionAnnouncements, SectionFAQ, SectionCopilotAI},
		Auth:     AuthConfig{Type: AuthTypeNone},
	}

	setupFunc := func() error {
		t.Log("Testing Discourse protocol with Mattermost forum - complex boolean queries")
		return nil
	}

	VerifyProtocolDiscourseBooleanQuery(t, protocol, source, setupFunc)
}
