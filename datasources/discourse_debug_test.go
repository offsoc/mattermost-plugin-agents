// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

// TestDiscourseProtocol_DebugTerms helps identify common terms in Discourse data
func TestDiscourseProtocol_DebugTerms(t *testing.T) {
	protocol := NewDiscourseProtocol(&http.Client{}, nil)

	source := SourceConfig{
		Name:     SourceMattermostForum,
		Protocol: DiscourseProtocolType,
		Endpoints: map[string]string{
			EndpointBaseURL:        MattermostForumURL,
			SectionTroubleshooting: "general",
		},
		Sections: []string{SectionTroubleshooting},
		Auth:     AuthConfig{Type: AuthTypeNone},
	}

	ctx := context.Background()

	// Test simple queries to find common terms
	simpleQueries := []string{
		"mattermost",
		"server",
		"mobile",
		"plugin",
		"installation",
		"configuration",
		"authentication",
		"notifications",
		"channels",
		"performance",
		"deployment",
		"upgrade",
		"docker",
		"database",
		"error",
	}

	t.Log("\n=== Testing simple queries ===")
	for _, query := range simpleQueries {
		request := ProtocolRequest{
			Source:   source,
			Topic:    query,
			Sections: source.Sections,
			Limit:    3,
		}

		docs, err := protocol.Fetch(ctx, request)
		if err != nil {
			t.Logf("Query '%s' failed: %v", query, err)
			continue
		}

		t.Logf("Query '%s' returned %d docs", query, len(docs))
		if len(docs) > 0 {
			t.Logf("  First doc: %s", docs[0].Title)
		}
	}

	// Test AND combinations
	t.Log("\n=== Testing AND combinations ===")
	andCombos := []string{
		"mattermost AND server",
		"mattermost AND plugin",
		"server AND configuration",
		"mobile AND notifications",
		"docker AND installation",
		"database AND performance",
		"authentication AND error",
	}

	for _, query := range andCombos {
		request := ProtocolRequest{
			Source:   source,
			Topic:    query,
			Sections: source.Sections,
			Limit:    3,
		}

		docs, err := protocol.Fetch(ctx, request)
		if err != nil {
			t.Logf("Query '%s' failed: %v", query, err)
			continue
		}

		if len(docs) > 0 {
			// Check if first doc actually contains both terms
			content := strings.ToLower(docs[0].Title + " " + docs[0].Content)
			parts := strings.Split(query, " AND ")
			term1 := strings.ToLower(strings.TrimSpace(parts[0]))
			term2 := strings.ToLower(strings.TrimSpace(parts[1]))
			has1 := strings.Contains(content, term1)
			has2 := strings.Contains(content, term2)

			if has1 && has2 {
				t.Logf("✓ Query '%s' returned %d docs - first doc has both terms", query, len(docs))
				t.Logf("  Title: %s", docs[0].Title)
			} else {
				t.Logf("✗ Query '%s' returned %d docs - first doc missing terms (has %s: %v, has %s: %v)",
					query, len(docs), term1, has1, term2, has2)
			}
		} else {
			t.Logf("✗ Query '%s' returned 0 docs", query)
		}
	}

	// Test complex combinations
	t.Log("\n=== Testing complex (A OR B) AND C combinations ===")
	complexCombos := []string{
		"(mattermost OR server) AND plugin",
		"(docker OR installation) AND configuration",
		"(mobile OR plugin) AND error",
		"(database OR performance) AND server",
	}

	for _, query := range complexCombos {
		request := ProtocolRequest{
			Source:   source,
			Topic:    query,
			Sections: source.Sections,
			Limit:    3,
		}

		docs, err := protocol.Fetch(ctx, request)
		if err != nil {
			t.Logf("Query '%s' failed: %v", query, err)
			continue
		}

		if len(docs) > 0 {
			t.Logf("✓ Query '%s' returned %d docs", query, len(docs))
			t.Logf("  Title: %s", docs[0].Title)
		} else {
			t.Logf("✗ Query '%s' returned 0 docs", query)
		}
	}
}
