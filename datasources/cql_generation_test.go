// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"net/http"
	"testing"
)

func TestCQLGeneration(t *testing.T) {
	protocol := NewConfluenceProtocol(&http.Client{}, nil)

	tests := []string{
		"channels",
		"mobile",
		"playbooks",
		"deployment OR installation",
		"security OR authentication",
		"api",
		"enterprise",
	}

	for _, topic := range tests {
		cql := protocol.buildExpandedCQL("CLOUD", topic)
		t.Logf("\n=== Topic: %s ===", topic)
		t.Logf("CQL (%d chars):\n%s", len(cql), cql)
	}
}
