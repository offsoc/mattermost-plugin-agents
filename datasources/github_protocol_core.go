// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/mattermost/mattermost-plugin-ai/mmapi"
)

type GitHubProtocol struct {
	client          *http.Client
	token           string
	auth            AuthConfig
	rateLimiter     *RateLimiter
	pluginAPI       mmapi.Client
	topicAnalyzer   *TopicAnalyzer
	universalScorer *UniversalRelevanceScorer
	circuitBreaker  *HTTPCircuitBreaker
}

type GitHubIssue struct {
	ID        int    `json:"id"`
	Number    int    `json:"number"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	HTMLURL   string `json:"html_url"`
	State     string `json:"state"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	User      struct {
		Login string `json:"login"`
	} `json:"user"`
	Labels []struct {
		Name string `json:"name"`
	} `json:"labels"`
	PullRequest *struct {
		URL     string `json:"url"`
		HTMLURL string `json:"html_url"`
	} `json:"pull_request,omitempty"`
}

type GitHubRelease struct {
	ID          int    `json:"id"`
	TagName     string `json:"tag_name"`
	Name        string `json:"name"`
	Body        string `json:"body"`
	HTMLURL     string `json:"html_url"`
	PublishedAt string `json:"published_at"`
	Draft       bool   `json:"draft"`
	Prerelease  bool   `json:"prerelease"`
}

type GitHubIssueComment struct {
	Body string `json:"body"`
	User struct {
		Login string `json:"login"`
	} `json:"user"`
}

func NewGitHubProtocol(token string, pluginAPI mmapi.Client) *GitHubProtocol {
	return &GitHubProtocol{
		client:          &http.Client{Timeout: DefaultHTTPClientTimeout},
		token:           token,
		auth:            AuthConfig{Type: AuthTypeToken, Key: token},
		pluginAPI:       pluginAPI,
		topicAnalyzer:   NewTopicAnalyzer(),
		universalScorer: NewUniversalRelevanceScorer(),
		circuitBreaker:  newHTTPCircuitBreaker(),
	}
}

func (g *GitHubProtocol) Fetch(ctx context.Context, request ProtocolRequest) ([]Doc, error) {
	source := request.Source

	EnsureRateLimiter(&g.rateLimiter, source.RateLimit)

	owner := source.Endpoints["owner"]
	reposList := source.Endpoints["repos"]
	if owner == "" || reposList == "" {
		return []Doc{}, fmt.Errorf("missing required GitHub configuration: owner and repos")
	}

	repos := strings.Split(reposList, ",")
	if request.Topic != "" {
		var filtered []string
		var scores []struct {
			repo  string
			score int
		}

		for _, r := range repos {
			r = strings.TrimSpace(r)
			score := g.topicAnalyzer.GetRepoRelevanceScore(r, request.Topic)
			if score > 0 {
				scores = append(scores, struct {
					repo  string
					score int
				}{r, score})
			} else {
				filtered = append(filtered, r)
			}
		}

		for i := 0; i < len(scores); i++ {
			for j := i + 1; j < len(scores); j++ {
				if scores[j].score > scores[i].score {
					scores[i], scores[j] = scores[j], scores[i]
				}
			}
		}

		var prioritized []string
		for _, s := range scores {
			prioritized = append(prioritized, s.repo)
		}

		if len(prioritized) > 0 {
			repos = append(repos[:0], append(prioritized, filtered...)...)
		}
	}
	var allDocs []Doc

	for _, repoName := range repos {
		if len(allDocs) >= request.Limit {
			break
		}

		repoName = strings.TrimSpace(repoName)
		docs, err := g.fetchFromRepository(ctx, owner, repoName, request)
		if err != nil {
			if g.pluginAPI != nil {
				g.pluginAPI.LogWarn(request.Source.Name+": fetch failed", "owner", owner, "repo", repoName, "error", err)
			}
			continue
		}

		allDocs = append(allDocs, docs...)
	}

	return allDocs, nil
}

func (g *GitHubProtocol) GetType() ProtocolType {
	return GitHubAPIProtocolType
}

func (g *GitHubProtocol) SetAuth(auth AuthConfig) {
	g.auth = auth
	if auth.Key != "" {
		g.token = auth.Key
	}
}

func (g *GitHubProtocol) Close() error {
	CloseRateLimiter(&g.rateLimiter)
	return nil
}

func (g *GitHubProtocol) fetchFromRepository(ctx context.Context, owner, repo string, request ProtocolRequest) ([]Doc, error) {
	var docs []Doc

	sections := g.prioritizeSections(request.Sections, request.Topic)

	for _, section := range sections {
		if len(docs) >= request.Limit {
			break
		}

		switch section {
		case "issues":
			issueDocs := g.fetchRecentIssues(ctx, owner, repo, request.Topic, request.Limit-len(docs), request.Source.Name)
			docs = append(docs, issueDocs...)
		case "releases":
			releaseDocs := g.fetchRecentReleases(ctx, owner, repo, request.Topic, request.Limit-len(docs), request.Source.Name)
			docs = append(docs, releaseDocs...)
		case "pulls":
			prDocs := g.fetchRecentPRs(ctx, owner, repo, request.Topic, request.Limit-len(docs), request.Source.Name)
			docs = append(docs, prDocs...)
		case "code":
			topicLower := strings.ToLower(request.Topic)
			useIssuesSearch := strings.Contains(topicLower, "issue") ||
				strings.Contains(topicLower, "feature") ||
				strings.Contains(topicLower, "bug") ||
				strings.Contains(topicLower, "enhancement") ||
				strings.Contains(topicLower, "request") ||
				strings.Contains(topicLower, "gap") ||
				strings.Contains(topicLower, "limitation") ||
				strings.Contains(topicLower, "missing") ||
				strings.Contains(topicLower, "feedback") ||
				strings.Contains(topicLower, "problem") ||
				strings.Contains(topicLower, "suggestion")

			if useIssuesSearch {
				issuesDocs := g.searchIssues(ctx, owner, repo, request.Topic, request.Limit-len(docs), request.Source.Name)
				docs = append(docs, issuesDocs...)
			} else {
				language := detectLanguageFromExtension(request.Topic)
				codeDocs := g.searchCode(ctx, owner, repo, request.Topic, language, request.Limit-len(docs), request.Source.Name)
				docs = append(docs, codeDocs...)
			}
		}
	}

	docs = FilterDocsByBooleanQuery(docs, request.Topic)
	return docs, nil
}

// prioritizeSections reorders sections to prioritize most relevant ones for the topic
func (g *GitHubProtocol) prioritizeSections(sections []string, topic string) []string {
	if topic == "" {
		return sections
	}

	topicLower := strings.ToLower(topic)

	releaseKeywords := []string{"release", "version", "changelog", "what's new", "release notes"}
	isReleaseQuery := false
	for _, keyword := range releaseKeywords {
		if strings.Contains(topicLower, keyword) {
			isReleaseQuery = true
			break
		}
	}

	if isReleaseQuery {
		return sections
	}

	prioritized := make([]string, 0, len(sections))
	remaining := make([]string, 0)

	for _, section := range sections {
		if section == "issues" || section == "pulls" {
			prioritized = append(prioritized, section)
		} else {
			remaining = append(remaining, section)
		}
	}

	prioritized = append(prioritized, remaining...)

	return prioritized
}

func (g *GitHubProtocol) addAuthHeaders(req *http.Request) {
	if g.token != "" {
		req.Header.Set(HeaderAuthorization, AuthPrefixBearer+g.token)
	}
	req.Header.Set(HeaderAccept, AcceptGitHubAPI)
	req.Header.Set(HeaderUserAgent, UserAgentMattermostPM)
}

func (g *GitHubProtocol) isRelevantToTopic(title, body, topic string) bool {
	if topic == "" {
		return true
	}

	content := strings.ToLower(title + " " + body)
	topicLower := strings.ToLower(topic)
	topicKeywords := strings.Fields(topicLower)

	for _, keyword := range topicKeywords {
		if strings.Contains(content, keyword) {
			return true
		}
	}

	return false
}

func isPullRequest(issue GitHubIssue) bool {
	return issue.PullRequest != nil
}
