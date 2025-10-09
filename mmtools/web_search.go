// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package mmtools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/k3a/html2text"

	"github.com/mattermost/mattermost-plugin-ai/config"
	"github.com/mattermost/mattermost-plugin-ai/llm"
)

const (
	// WebSearchContextKey is the key used within llm.Context.Parameters to store web search results
	WebSearchContextKey = "mm_web_search_results"

	defaultGoogleSearchEndpoint = "https://www.googleapis.com/customsearch/v1"
	minQueryLength              = 3
	// WebSearchSourceFetchDescription describes the page retrieval tool.
	WebSearchSourceFetchDescription = "Fetch the full HTML content at a given URL and convert it to plain text for analysis. Use this tool to fetch more content from a web search result, or when a link is provided to you by the user. Responses from this tool should be scrutinized for relevance, as some fetches may return generic pages as they don't allow AI Agents to access them."
)

// WebSearchService exposes the built-in web search tool if configured.
type WebSearchService interface {
	Tool() *llm.Tool
	SourceTool() *llm.Tool
}

// WebSearchLog abstracts the logging interface used by the service.
type WebSearchLog interface {
	Debug(message string, keyValuePairs ...any)
	Info(message string, keyValuePairs ...any)
	Warn(message string, keyValuePairs ...any)
	Error(message string, keyValuePairs ...any)
}

// WebSearchToolArgs represents the JSON schema for the web search tool input.
type WebSearchToolArgs struct {
	Query string `jsonschema_description:"The web search query to execute."`
}

// WebSearchSourceArgs represents the input to fetch a single web page.
type WebSearchSourceArgs struct {
	URL string `jsonschema_description:"The absolute URL of the web page to retrieve."`
}

// WebSearchResult represents a single web search result consumed by downstream components.
type WebSearchResult struct {
	Index   int    `json:"index"`
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
	Query   string `json:"query"`
}

// WebSearchContextValue stores the results produced by a single tool invocation.
type WebSearchContextValue struct {
	Query   string            `json:"query"`
	Results []WebSearchResult `json:"results"`
}

type webSearchService struct {
	cfgGetter  func() *config.Config
	logger     WebSearchLog
	httpClient *http.Client
	tool       *llm.Tool
	sourceTool *llm.Tool
	mutex      sync.RWMutex
}

// NewWebSearchService constructs a new WebSearchService implementation.
func NewWebSearchService(cfgGetter func() *config.Config, logger WebSearchLog, httpClient *http.Client) WebSearchService {
	service := &webSearchService{
		cfgGetter:  cfgGetter,
		logger:     logger,
		httpClient: httpClient,
	}

	service.tool = &llm.Tool{
		Name:        "WebSearch",
		Description: "Perform a live web search using Google's Custom Search API. Use this tool to retrieve current information. Keep your search queries generic and concise according to the user's ask. Cite sources in your final answer using markers like [1], wrapped in markdown formatting with the full URL of the source. ",
		Schema:      llm.NewJSONSchemaFromStruct[WebSearchToolArgs](),
		Resolver:    service.resolve,
	}

	service.sourceTool = &llm.Tool{
		Name:        "WebSearchFetchSource",
		Description: WebSearchSourceFetchDescription,
		Schema:      llm.NewJSONSchemaFromStruct[WebSearchSourceArgs](),
		Resolver:    service.resolveSource,
	}

	return service
}

// Tool returns the web search tool if the configuration is valid and enabled.
func (s *webSearchService) Tool() *llm.Tool {
	s.mutex.RLock()
	tool := s.tool
	s.mutex.RUnlock()
	if tool == nil {
		return nil
	}

	cfg := s.cfgGetter()
	if cfg == nil {
		return nil
	}

	webCfg := cfg.WebSearch
	if !webCfg.Enabled {
		return nil
	}

	if strings.ToLower(strings.TrimSpace(webCfg.Provider)) != "google" {
		s.logDebug("web search provider not supported", "provider", webCfg.Provider)
		return nil
	}

	if webCfg.Google.APIKey == "" || webCfg.Google.SearchEngineID == "" {
		s.logWarn("web search misconfigured: missing API credentials")
		return nil
	}

	return tool
}

// SourceTool returns the configured web source fetch tool or nil if unavailable.
func (s *webSearchService) SourceTool() *llm.Tool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if s.sourceTool == nil {
		return nil
	}

	cfg := s.cfgGetter()
	if cfg == nil {
		return nil
	}

	webCfg := cfg.WebSearch
	if !webCfg.Enabled {
		return nil
	}

	if strings.ToLower(strings.TrimSpace(webCfg.Provider)) != "google" {
		return nil
	}

	if webCfg.Google.APIKey == "" || webCfg.Google.SearchEngineID == "" {
		return nil
	}

	return s.sourceTool
}

func (s *webSearchService) resolve(llmContext *llm.Context, argsGetter llm.ToolArgumentGetter) (string, error) {
	var args WebSearchToolArgs
	if err := argsGetter(&args); err != nil {
		return "invalid parameters to function", fmt.Errorf("failed to get arguments for WebSearch tool: %w", err)
	}

	query := strings.TrimSpace(args.Query)
	if len([]rune(query)) < minQueryLength {
		return fmt.Sprintf("query must be at least %d characters", minQueryLength), errors.New("web search query too short")
	}

	if query == "" {
		return "query cannot be empty", errors.New("query cannot be empty")
	}

	cfg := s.cfgGetter()
	if cfg == nil {
		return "web search is not configured", errors.New("web search config unavailable")
	}

	webCfg := cfg.WebSearch
	if !webCfg.Enabled {
		return "web search is disabled", errors.New("web search disabled")
	}

	previousParameters := map[string]interface{}{}
	if llmContext != nil && llmContext.Parameters != nil {
		for k, v := range llmContext.Parameters {
			previousParameters[k] = v
		}
	}

	results, err := s.googleSearch(llmContext, query, webCfg.Google, webCfg.Google.ResultLimit)
	if err != nil {
		return "unable to perform web search", err
	}

	if len(results) == 0 {
		return fmt.Sprintf("No web results found for \"%s\"", query), nil
	}

	// Persist results into the LLM context for later processing (annotations, UI rendering)
	if llmContext.Parameters == nil {
		llmContext.Parameters = map[string]interface{}{}
	}
	var offset int
	var existing []WebSearchContextValue
	if raw, ok := llmContext.Parameters[WebSearchContextKey]; ok {
		if stored, ok := raw.([]WebSearchContextValue); ok {
			existing = stored
			offset = countTotalWebResults(stored)
		}
	}
	for i := range results {
		results[i].Index = offset + i + 1
		results[i].Query = query
	}
	existing = append(existing, WebSearchContextValue{
		Query:   query,
		Results: results,
	})
	llmContext.Parameters[WebSearchContextKey] = existing

	if len(previousParameters) > 0 {
		// Restore any other parameters to their previous values
		for k, v := range previousParameters {
			if k == WebSearchContextKey {
				continue
			}
			llmContext.Parameters[k] = v
		}

		for k := range llmContext.Parameters {
			if k == WebSearchContextKey {
				continue
			}
			if _, ok := previousParameters[k]; !ok {
				delete(llmContext.Parameters, k)
			}
		}
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Live web search results for \"%s\":\n", query))
	builder.WriteString("Use the numbered references [n] when citing these sources in your reply.\n\n")
	for _, result := range results {
		builder.WriteString(fmt.Sprintf("[%d] %s\n", result.Index, result.Title))
		builder.WriteString(fmt.Sprintf("URL: %s\n", result.URL))
		if result.Snippet != "" {
			builder.WriteString(fmt.Sprintf("Snippet: %s\n", result.Snippet))
		}
		builder.WriteString("\n")
	}

	return builder.String(), nil
}

func (s *webSearchService) resolveSource(llmContext *llm.Context, argsGetter llm.ToolArgumentGetter) (string, error) {
	var args WebSearchSourceArgs
	if err := argsGetter(&args); err != nil {
		return "invalid parameters to function", fmt.Errorf("failed to get arguments for WebSearchFetchSource tool: %w", err)
	}

	pageURL := strings.TrimSpace(args.URL)
	if pageURL == "" {
		return "url cannot be empty", errors.New("source fetch url empty")
	}

	if !strings.HasPrefix(pageURL, "http://") && !strings.HasPrefix(pageURL, "https://") {
		return "url must be absolute", errors.New("source fetch url must be absolute")
	}

	client := s.httpClient
	if client == nil {
		client = http.DefaultClient
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, pageURL, nil)
	if err != nil {
		s.logError("failed to create source fetch request", "error", err)
		return "unable to create request", err
	}
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("User-Agent", "Mattermost-AI-Plugin/1.0")

	resp, err := client.Do(req)
	if err != nil {
		s.logError("source fetch request failed", "error", err, "url", pageURL)
		return "unable to fetch the requested URL", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		s.logWarn("source fetch non-success status", "status", resp.Status, "url", pageURL)
		return fmt.Sprintf("failed to fetch URL: %s", resp.Status), fmt.Errorf("source fetch failed: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.logError("failed to read source fetch response", "error", err, "url", pageURL)
		return "unable to read the response", err
	}

	text := strings.TrimSpace(html2text.HTML2Text(string(body)))
	if text == "" {
		s.logWarn("source fetch resulted in empty text", "url", pageURL)
		return "fetched page contained no readable text", nil
	}

	return text, nil
}

func (s *webSearchService) googleSearch(ctx *llm.Context, query string, cfg config.WebSearchGoogleConfig, desiredLimit int) ([]WebSearchResult, error) {
	endpoint := strings.TrimSpace(cfg.APIURL)
	if endpoint == "" {
		endpoint = defaultGoogleSearchEndpoint
	}

	limit := desiredLimit
	if limit <= 0 {
		limit = 5
	}
	if limit > 10 {
		limit = 10
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create web search request: %w", err)
	}

	values := url.Values{}
	values.Set("key", cfg.APIKey)
	values.Set("cx", cfg.SearchEngineID)
	values.Set("q", query)
	values.Set("num", strconv.Itoa(limit))
	req.URL.RawQuery = values.Encode()
	req.Header.Set("Accept", "application/json")

	client := s.httpClient
	if client == nil {
		client = http.DefaultClient
	}

	resp, err := client.Do(req)
	if err != nil {
		s.logError("web search request failed", "error", err)
		return nil, fmt.Errorf("web search request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("web search request failed: status %s", resp.Status)
	}

	var payload googleSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("failed to decode web search response: %w", err)
	}

	results := make([]WebSearchResult, 0, len(payload.Items))
	for _, item := range payload.Items {
		results = append(results, WebSearchResult{
			Title:   strings.TrimSpace(item.Title),
			URL:     strings.TrimSpace(item.Link),
			Snippet: strings.TrimSpace(item.Snippet),
		})
	}

	return results, nil
}

type googleSearchResponse struct {
	Items []struct {
		Title   string `json:"title"`
		Link    string `json:"link"`
		Snippet string `json:"snippet"`
	} `json:"items"`
}

func (s *webSearchService) logDebug(msg string, keyValuePairs ...any) {
	if s.logger != nil {
		s.logger.Debug(msg, keyValuePairs...)
	}
}

func (s *webSearchService) logWarn(msg string, keyValuePairs ...any) {
	if s.logger != nil {
		s.logger.Warn(msg, keyValuePairs...)
	}
}

func (s *webSearchService) logError(msg string, keyValuePairs ...any) {
	if s.logger != nil {
		s.logger.Error(msg, keyValuePairs...)
	}
}

func countTotalWebResults(values []WebSearchContextValue) int {
	count := 0
	for _, v := range values {
		count += len(v.Results)
	}
	return count
}

// ConsumeWebSearchContexts extracts and removes the stored search context values.
func ConsumeWebSearchContexts(ctx *llm.Context) []WebSearchContextValue {
	if ctx == nil || ctx.Parameters == nil {
		return nil
	}

	raw, ok := ctx.Parameters[WebSearchContextKey]
	if !ok {
		return nil
	}

	values, ok := raw.([]WebSearchContextValue)
	if !ok {
		delete(ctx.Parameters, WebSearchContextKey)
		return nil
	}

	delete(ctx.Parameters, WebSearchContextKey)
	return values
}

// FlattenWebSearchResults flattens the result sets from multiple tool executions into a single slice.
func FlattenWebSearchResults(values []WebSearchContextValue) []WebSearchResult {
	if len(values) == 0 {
		return nil
	}

	flat := make([]WebSearchResult, 0)
	for _, value := range values {
		for _, result := range value.Results {
			flat = append(flat, result)
		}
	}

	return flat
}

// DecorateStreamWithAnnotations attaches annotation events based on search results to the provided stream.
func DecorateStreamWithAnnotations(result *llm.TextStreamResult, searchData []WebSearchContextValue) *llm.TextStreamResult {
	if result == nil || len(searchData) == 0 {
		return result
	}

	flat := FlattenWebSearchResults(searchData)
	if len(flat) == 0 {
		return result
	}

	output := make(chan llm.TextStreamEvent)
	go func() {
		defer close(output)
		var builder strings.Builder
		for event := range result.Stream {
			switch event.Type {
			case llm.EventTypeText:
				if text, ok := event.Value.(string); ok {
					builder.WriteString(text)
				}
				output <- event
			case llm.EventTypeEnd:
				annotations := buildWebSearchAnnotations(builder.String(), flat)
				if len(annotations) > 0 {
					output <- llm.TextStreamEvent{
						Type:  llm.EventTypeAnnotations,
						Value: annotations,
					}
				}
				output <- event
			default:
				output <- event
			}
		}
	}()

	return &llm.TextStreamResult{Stream: output}
}

func buildWebSearchAnnotations(message string, results []WebSearchResult) []llm.Annotation {
	if len(message) == 0 || len(results) == 0 {
		return nil
	}

	indexMap := make(map[int]WebSearchResult, len(results))
	for _, res := range results {
		indexMap[res.Index] = res
	}

	annotations := []llm.Annotation{}
	pos := 0
	runeIndex := 0
	for pos < len(message) {
		r, size := utf8.DecodeRuneInString(message[pos:])
		if r == '[' {
			startRuneIndex := runeIndex
			pos += size
			runeIndex++

			numBuilder := strings.Builder{}
			digitCursor := pos
			digitRuneIndex := runeIndex
			for digitCursor < len(message) {
				digitRune, digitSize := utf8.DecodeRuneInString(message[digitCursor:])
				if digitRune < '0' || digitRune > '9' {
					break
				}
				numBuilder.WriteRune(digitRune)
				digitCursor += digitSize
				digitRuneIndex++
			}

			if numBuilder.Len() == 0 {
				pos = digitCursor
				runeIndex = digitRuneIndex
				continue
			}

			if digitCursor >= len(message) {
				pos = digitCursor
				runeIndex = digitRuneIndex
				continue
			}

			closeRune, closeSize := utf8.DecodeRuneInString(message[digitCursor:])
			closingRuneIndex := digitRuneIndex + 1
			nextPos := digitCursor + closeSize
			if closeRune != ']' {
				pos = nextPos
				runeIndex = closingRuneIndex
				continue
			}

			idx, err := strconv.Atoi(numBuilder.String())
			if err == nil {
				if res, ok := indexMap[idx]; ok {
					annotations = append(annotations, llm.Annotation{
						Type:       llm.AnnotationTypeURLCitation,
						StartIndex: startRuneIndex,
						EndIndex:   closingRuneIndex,
						URL:        res.URL,
						Title:      res.Title,
						CitedText:  res.Snippet,
						Index:      idx,
					})
				}
			}

			pos = nextPos
			runeIndex = closingRuneIndex
			continue
		}

		pos += size
		runeIndex++
	}

	return annotations
}

const footnoteTemplate = "[%d] %s — %s"

func formatWebSearchResults(results []WebSearchResult) string {
	if len(results) == 0 {
		return ""
	}

	builder := strings.Builder{}
	builder.WriteString("\nSources:\n")
	for _, result := range results {
		builder.WriteString(fmt.Sprintf(footnoteTemplate, result.Index, result.Title, result.URL))
		builder.WriteString("\n")
	}

	return builder.String()
}
