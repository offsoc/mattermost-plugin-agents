// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

func (g *GitHubProtocol) fetchRecentReleases(ctx context.Context, owner, repo string, topic string, limit int, sourceName string) []Doc {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases", owner, repo)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return []Doc{}
	}

	query := req.URL.Query()
	query.Add(QueryParamPerPage, fmt.Sprintf("%d", limit))
	req.URL.RawQuery = query.Encode()

	g.addAuthHeaders(req)

	if waitErr := WaitRateLimiter(ctx, g.rateLimiter); waitErr != nil {
		return []Doc{}
	}

	resp, err := g.client.Do(req)
	if err != nil {
		return []Doc{}
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return []Doc{}
	}

	var releases []GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return []Doc{}
	}

	var docs []Doc
	for _, release := range releases {
		if release.Draft {
			continue
		}

		if !g.topicAnalyzer.IsTopicRelevantContent(release.Name+" "+release.Body, topic) {
			continue
		}

		content := g.formatReleaseContent(release)
		if !g.universalScorer.IsUniversallyAcceptable(content, release.Name, sourceName, "") {
			continue
		}
		doc := Doc{
			Title:   fmt.Sprintf("%s - %s", release.TagName, release.Name),
			Content: content,
			URL:     release.HTMLURL,
			Section: "releases",
			Source:  sourceName,
		}
		docs = append(docs, doc)
	}

	return docs
}

func (g *GitHubProtocol) formatReleaseContent(release GitHubRelease) string {
	content := fmt.Sprintf("Release %s: %s\n", release.TagName, release.Name)
	content += fmt.Sprintf("Published: %s\n", release.PublishedAt)

	if release.Prerelease {
		content += "Type: Pre-release\n"
	} else {
		content += "Type: Stable release\n"
	}

	content += "\n"
	if release.Body != "" {
		body := release.Body
		if len(body) > 500 {
			body = body[:500] + "..."
		}
		content += body
	}

	return content
}
