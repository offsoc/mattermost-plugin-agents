// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"strings"
)

// MattermostFeatureTaxonomy defines the centralized mapping of Mattermost features to their synonyms
var MattermostFeatureTaxonomy = map[string][]string{
	"ai":            {"ai", "artificial intelligence", "copilot", "agents", "mm agents", "mattermost agents", "assistant", "chatbot", "automation", "machine learning", "ml", "llm", "large language model", "intelligent assistant", "conversational ai", "ai-powered"},
	"channels":      {"channels", "messaging", "chat", "threads", "direct messages", "dms", "group messages", "gms", "conversations", "communication"},
	"mobile":        {"mobile", "ios", "android", "app", "smartphone", "tablet", "mobile app", "native app", "react native", "react-native", "push notifications", "push notification", "offline-first", "edge mobile"},
	"playbooks":     {"playbooks", "runbooks", "automation", "workflows", "processes", "procedures", "checklists", "templates", "incident response", "conditional playbook", "playbook run", "task assignment"},
	"boards":        {"boards", "kanban", "tasks", "project management", "cards", "focalboard", "project boards", "task management"},
	"calls":         {"calls", "voice", "video", "meetings", "conferencing", "webrtc", "audio", "screen sharing", "video calls", "voice calls"},
	"plugins":       {"plugins", "integrations", "extensions", "connectors", "third-party", "marketplace", "ecosystem", "webhooks", "api integrations"},
	"enterprise":    {"enterprise", "enterprise advanced", "commercial", "licensing", "compliance", "governance", "administration", "enterprise features"},
	"security":      {"security", "authentication", "authorization", "sso", "saml", "ldap", "mfa", "two-factor", "oauth", "encryption", "compliance", "audit"},
	"deployment":    {"deployment", "installation", "setup", "configuration", "docker", "kubernetes", "containerization", "helm", "self-hosted", "on-premise", "cloud"},
	"performance":   {"performance", "scalability", "high availability", "load balancing", "clustering", "optimization", "speed", "latency", "throughput"},
	"api":           {"api", "rest api", "webhooks", "sdk", "developer", "integration", "endpoints", "documentation", "swagger"},
	"desktop":       {"desktop", "desktop app", "electron", "windows", "mac", "linux", "native desktop"},
	"web":           {"web", "webapp", "browser", "web app", "web interface", "web client"},
	"websockets":    {"websockets", "websocket", "ws", "real-time", "realtime", "live updates", "push", "bidirectional", "socket.io"},
	"notifications": {"notifications", "push notifications", "alerts", "mentions", "badges", "email notifications"},
}

// TopicAnalyzer provides centralized topic analysis and keyword expansion utilities
type TopicAnalyzer struct{}

// NewTopicAnalyzer creates a new topic analyzer instance
func NewTopicAnalyzer() *TopicAnalyzer {
	return &TopicAnalyzer{}
}

// containsWord checks if a word exists as a complete word (not substring)
// This prevents "web" from matching inside "websocket"
func containsWord(text, word string) bool {
	// For multi-word phrases, use simple substring matching
	if strings.Contains(word, " ") {
		return strings.Contains(text, word)
	}

	// For single words, check word boundaries
	words := strings.FieldsFunc(text, func(r rune) bool {
		// Split on non-alphanumeric characters except hyphens (to keep "ai-powered" intact)
		return (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-'
	})

	for _, w := range words {
		if w == word {
			return true
		}
	}
	return false
}

// ExtractTopicKeywords extracts and expands keywords from a topic string
func (t *TopicAnalyzer) ExtractTopicKeywords(topic string) []string {
	if topic == "" {
		return nil
	}

	topicLower := strings.ToLower(topic)

	// Try to parse as boolean query first
	var keywords []string
	if queryNode, err := ParseBooleanQuery(topic); err == nil && queryNode != nil {
		keywords = ExtractKeywords(queryNode)
		for i := range keywords {
			keywords[i] = strings.ToLower(keywords[i])
		}
	} else {
		keywords = strings.Fields(topicLower)
	}

	// Expand with synonyms and related terms
	var expanded []string
	expanded = append(expanded, keywords...)

	if len(keywords) > 1 {
		expanded = append(expanded, topicLower)
	}

	for _, keyword := range keywords {
		synonyms := t.GetTopicSynonyms(keyword)
		expanded = append(expanded, synonyms...)
	}

	return t.deduplicateStrings(expanded)
}

// GetTopicSynonyms returns related terms and synonyms for a given keyword
func (t *TopicAnalyzer) GetTopicSynonyms(keyword string) []string {
	if synonyms, exists := MattermostFeatureTaxonomy[keyword]; exists {
		return synonyms
	}
	synonymMap := map[string][]string{
		// Federal & Military
		"federal":  {"government", "military", "defense", "dod", "army", "navy", "air force", "marines", "coast guard", "federal agency", "clearance", "classified", "unclassified", "cui", "fisma", "fedramp"},
		"military": {"defense", "army", "navy", "air force", "marines", "tactical", "combat", "battlefield", "mission critical", "secure communications", "command and control", "c2"},
		"ddil":     {"disconnected", "intermittent", "limited", "constrained", "degraded network", "poor connectivity", "edge environment", "tactical environment", "contested environment"},

		// Network & Connectivity
		"network":   {"connectivity", "bandwidth", "latency", "throughput", "mesh", "lte", "satellite", "tactical network", "degraded network", "constrained network", "edge network", "low bandwidth", "high latency"},
		"offline":   {"disconnected", "intermittent", "store-and-forward", "offline-first", "cached", "local storage", "sync", "eventual consistency", "offline mode", "disconnected operations"},
		"bandwidth": {"throughput", "data rate", "mbps", "kbps", "network capacity", "data limit", "traffic", "congestion", "low bandwidth", "high bandwidth", "constrained bandwidth"},
		"mesh":      {"manet", "ad-hoc network", "peer-to-peer", "p2p", "mesh network", "mesh radio", "tactical mesh", "mobile ad-hoc network", "mesh topology", "self-healing network"},
		"lte":       {"4g", "cellular", "mobile network", "wireless", "bonded lte", "carrier aggregation", "tak lte", "commercial lte", "tactical lte", "lte mesh"},
		"satellite": {"sat", "satcom", "iridium", "certus", "leo sat", "geostationary", "low earth orbit", "satellite communication", "sat transceiver", "sat link"},

		// Infrastructure & Deployment
		"docker":     {"container", "containerization", "dockerfile", "docker-compose", "image", "registry", "orchestration", "microservices", "containerized", "docker swarm"},
		"kubernetes": {"k8s", "container orchestration", "pod", "deployment", "service", "ingress", "helm", "operator", "eks", "aks", "gke", "openshift", "cluster"},
		"ha":         {"high availability", "redundancy", "failover", "clustering", "load balancing", "fault tolerance", "disaster recovery", "uptime", "availability", "reliability"},

		// Specialized Terms
		"iridium":     {"satellite", "certus", "leo sat", "sat transceiver", "portable hardware", "burst mode", "sbd", "satellite phone", "global coverage"},
		"tactical":    {"military", "combat", "battlefield", "mission critical", "secure", "tactical network", "tactical communications", "tactical environment", "field operations"},
		"manet":       {"mobile ad-hoc network", "mesh", "tactical network", "ad-hoc", "mesh network", "peer-to-peer", "self-organizing", "dynamic topology"},
		"jitter":      {"network jitter", "packet delay variation", "latency variation", "network instability", "timing variation", "quality degradation"},
		"constrained": {"limited", "degraded", "restricted", "edge", "tactical", "contested", "ddil", "low bandwidth", "high latency", "unreliable"},

		// Common Infrastructure (general terms not specific to Mattermost)
		"user":       {"ui", "ux", "interface", "experience", "usability", "accessibility", "user interface", "user experience", "human computer interaction", "hci", "end user"},
		"testing":    {"qa", "quality assurance", "automation", "unit test", "integration test", "e2e", "end to end", "test automation", "quality control"},
		"monitoring": {"metrics", "logging", "alerts", "observability", "telemetry", "analytics", "tracking", "debugging", "error tracking", "apm"},

		// Content filtering keywords (for quality detection)
		"promotional": {"hacktoberfest", "contribute, collaborate & earn rewards", "contribute, collaborate", "earn rewards", "special offer", "limited time", "subscribe now", "sign up", "contact sales", "free trial", "get started free", "buy now", "upgrade now", "newsletter signup", "follow us", "social media"},
		"technical":   {"window.datalayer", "gtag(", "--primary-1:", "--contrast-primary-1:", "analytics_storage", "ad_storage", "consent", "default", "github.com/mattermost/docs/edit/master", "bluesky icon"},
	}

	if synonyms, exists := synonymMap[keyword]; exists {
		return synonyms
	}
	return nil
}

// GetMattermostFeatures detects Mattermost features mentioned in a topic string
func (t *TopicAnalyzer) GetMattermostFeatures(topic string) []string {
	if topic == "" {
		return nil
	}

	topicLower := strings.ToLower(topic)
	var foundFeatures []string
	seenFeatures := make(map[string]bool)

	type match struct {
		canonical string
		synonym   string
		length    int
	}
	var matches []match

	for canonical, synonyms := range MattermostFeatureTaxonomy {
		for _, synonym := range synonyms {
			synonymLower := strings.ToLower(synonym)
			// Use word boundary matching for more accurate detection
			// This prevents "web" from matching inside "websocket"
			if containsWord(topicLower, synonymLower) {
				matches = append(matches, match{
					canonical: canonical,
					synonym:   synonym,
					length:    len(synonym),
				})
			}
		}
	}

	// Sort matches by length (longest first) to prioritize more specific terms
	// This ensures "websocket" is matched before "web"
	for i := 0; i < len(matches); i++ {
		for j := i + 1; j < len(matches); j++ {
			if matches[j].length > matches[i].length {
				matches[i], matches[j] = matches[j], matches[i]
			}
		}
	}

	for _, m := range matches {
		if !seenFeatures[m.canonical] {
			foundFeatures = append(foundFeatures, m.canonical)
			seenFeatures[m.canonical] = true
		}
	}

	return foundFeatures
}

// ExpandMattermostFeatureSynonyms expands canonical feature names to all their synonyms
func (t *TopicAnalyzer) ExpandMattermostFeatureSynonyms(features []string) []string {
	var allSynonyms []string
	seen := make(map[string]bool)

	for _, feature := range features {
		if synonyms, exists := MattermostFeatureTaxonomy[feature]; exists {
			for _, synonym := range synonyms {
				if !seen[synonym] {
					allSynonyms = append(allSynonyms, synonym)
					seen[synonym] = true
				}
			}
		} else if !seen[feature] {
			// If feature not in taxonomy, add it literally
			allSynonyms = append(allSynonyms, feature)
			seen[feature] = true
		}
	}

	return allSynonyms
}

// ScoreContentRelevance scores content based on keyword relevance
func (t *TopicAnalyzer) ScoreContentRelevance(content string, keywords []string) int {
	return t.ScoreContentRelevanceWithTitle(content, keywords, "")
}

// ScoreContentRelevanceWithTitle scores content and title based on keyword relevance
func (t *TopicAnalyzer) ScoreContentRelevanceWithTitle(content string, keywords []string, title string) int {
	if len(keywords) == 0 {
		return 0
	}

	contentScore := t.scoreText(strings.ToLower(content), keywords, 1.0)

	// Title matches get 3x weight since titles are more indicative of document topic
	titleScore := 0
	if title != "" {
		titleScore = t.scoreText(strings.ToLower(title), keywords, 3.0)

		// Only add bonus for documentation-style titles if there's already some topic relevance
		// This prevents irrelevant documentation from getting a false positive boost
		if titleScore > 0 || contentScore > 0 {
			lowerTitle := strings.ToLower(title)
			for _, docKeyword := range DocumentationTitleKeywords {
				if strings.Contains(lowerTitle, docKeyword) {
					titleScore += DocumentationTitleBonus
					break
				}
			}
		}
	}

	return contentScore + titleScore
}

// scoreText scores a text string based on keyword matches with weighting
func (t *TopicAnalyzer) scoreText(lowerText string, keywords []string, multiplier float64) int {
	score := 0

	// Define keyword weights for generic vs specific terms
	genericTerms := map[string]bool{
		"device": true, "phone": true, "tablet": true, "app": true, "application": true,
		"system": true, "service": true, "tool": true, "feature": true, "function": true,
		"interface": true, "experience": true, "user": true, "admin": true, "config": true,
	}

	for _, kw := range keywords {
		if kw == "" {
			continue
		}

		if strings.Contains(lowerText, kw) {
			// Base score for keyword match
			keywordScore := 1.0

			// Boost exact multi-word phrase matches
			switch {
			case strings.Contains(kw, " ") && len(strings.Fields(kw)) > 1:
				keywordScore *= 3.0 // Multi-word phrases get 3x boost
			case genericTerms[kw]:
				keywordScore *= 0.5 // Generic terms get 0.5x weight
			case len(kw) > 8:
				keywordScore *= 2.0 // Specific long terms get 2x weight
			case len(kw) > 6:
				keywordScore *= 1.5 // Medium terms get 1.5x weight
			}

			// Count occurrences for frequency boost
			occurrences := strings.Count(lowerText, kw)
			if occurrences > 1 {
				keywordScore += float64(min(occurrences-1, 2)) // Max boost of 2
			}

			score += int(keywordScore * multiplier)
		}
	}

	return score
}

// IsTopicRelevantContent checks if content contains topic-relevant keywords
func (t *TopicAnalyzer) IsTopicRelevantContent(content, topic string) bool {
	return t.IsTopicRelevantContentWithTitle(content, topic, "")
}

// IsTopicRelevantContentWithTitle checks if content and title contain topic-relevant keywords
func (t *TopicAnalyzer) IsTopicRelevantContentWithTitle(content, topic, title string) bool {
	if topic == "" {
		return true
	}

	keywords := t.ExtractTopicKeywords(topic)
	score := t.ScoreContentRelevanceWithTitle(content, keywords, title)

	// Use adaptive threshold based on topic complexity
	threshold := t.getMinimumThreshold(topic)
	return score >= threshold
}

// getMinimumThreshold returns appropriate threshold based on topic complexity
func (t *TopicAnalyzer) getMinimumThreshold(topic string) int {
	words := strings.Fields(topic)

	// Check if this is an AI-related topic using the existing taxonomy
	mattermostFeatures := t.GetMattermostFeatures(topic)
	for _, feature := range mattermostFeatures {
		if feature == "ai" {
			// Very lenient threshold for AI topics since they're highly sought after
			return 1
		}
	}

	// Single word topics: lower threshold
	if len(words) == 1 {
		return 2
	}

	// Multi-word topics need matches from multiple concepts
	if len(words) >= 2 {
		return len(words) // e.g., "mobile strategy" needs score >= 2
	}

	return 2
}

// GetTopicRelevantSections returns sections most relevant to a given topic
func (t *TopicAnalyzer) GetTopicRelevantSections(topic string) []string {
	if topic == "" {
		return nil
	}

	topicLower := strings.ToLower(topic)

	// Map topic keywords to relevant sections
	sectionMap := map[string][]string{
		"mobile":            {SectionMobile, SectionMobileApps, SectionMobileStrategy},
		"ios":               {SectionMobile, SectionMobileApps},
		"android":           {SectionMobile, SectionMobileApps},
		"web":               {SectionDeveloper, SectionAPI},
		"server":            {SectionAdmin, SectionAPI, SectionDeveloper},
		"desktop":           {SectionDeveloper},
		"enterprise":        {SectionAdmin},
		"admin":             {SectionAdmin},
		"developer":         {SectionDeveloper, SectionAPI},
		"api":               {SectionAPI, SectionDeveloper},
		"security":          {SectionAdmin},
		"deployment":        {SectionAdmin},
		"plugin":            {SectionPlugins, SectionDeveloper},
		"integration":       {SectionIntegrations, SectionDeveloper},
		"ai":                {SectionPlugins, SectionIntegrations, SectionDeveloper, SectionAPI},
		"copilot":           {SectionPlugins, SectionIntegrations, SectionDeveloper, SectionAPI},
		"agents":            {SectionPlugins, SectionIntegrations, SectionDeveloper, SectionAPI},
		"assistant":         {SectionPlugins, SectionIntegrations, SectionDeveloper, SectionAPI},
		"chatbot":           {SectionPlugins, SectionIntegrations, SectionDeveloper, SectionAPI},
		"automation":        {SectionPlugins, SectionIntegrations, SectionDeveloper, SectionAPI},
		"ldap":              {SectionAdmin},
		"saml":              {SectionAdmin},
		"sso":               {SectionAdmin},
		"auth":              {SectionAdmin},
		"authentication":    {SectionAdmin},
		"authorization":     {SectionAdmin},
		"oauth":             {SectionAdmin},
		"mfa":               {SectionAdmin},
		"active directory":  {SectionAdmin},
		"ad":                {SectionAdmin},
		"config":            {SectionAdmin},
		"configuration":     {SectionAdmin},
		"clustering":        {SectionAdmin},
		"cluster":           {SectionAdmin},
		"high availability": {SectionAdmin},
		"ha":                {SectionAdmin},
	}

	var relevantSections []string
	for keyword, sections := range sectionMap {
		if strings.Contains(topicLower, keyword) {
			relevantSections = append(relevantSections, sections...)
		}
	}

	return t.deduplicateStrings(relevantSections)
}

// GetRepoRelevanceScore calculates how relevant a repository name is to a given topic
func (t *TopicAnalyzer) GetRepoRelevanceScore(repo, topic string) int {
	if topic == "" {
		return 1 // Default relevance when no topic specified
	}

	repoLower := strings.ToLower(repo)
	topicLower := strings.ToLower(topic)

	score := 0

	keywords := strings.Fields(topicLower)
	for _, keyword := range keywords {
		if keyword == "" {
			continue
		}

		if strings.Contains(repoLower, keyword) {
			score += 3
		}

		synonyms := t.GetTopicSynonyms(keyword)
		for _, synonym := range synonyms {
			if strings.Contains(repoLower, synonym) {
				score += 2 // Medium score for synonym matches
			}
		}
	}

	// Special cases for well-known repo patterns
	specialPatterns := map[string][]string{
		"mobile":    {"mobile"},
		"web":       {"webapp", "web"},
		"desktop":   {"desktop"},
		"server":    {"server"},
		"ai":        {"ai", "copilot", "agents", "assistant"},
		"copilot":   {"copilot", "ai", "agents", "assistant"},
		"agents":    {"agents", "copilot", "ai", "assistant"},
		"assistant": {"assistant", "copilot", "ai", "agents"},
	}

	for topicKey, repoPatterns := range specialPatterns {
		if strings.Contains(topicLower, topicKey) {
			for _, pattern := range repoPatterns {
				if strings.Contains(repoLower, pattern) {
					score += 5
				}
			}
		}
	}

	return score
}

// SelectBestChunk selects the most relevant chunk based on topic keywords
func (t *TopicAnalyzer) SelectBestChunk(chunks []string, topic string) string {
	if len(chunks) == 0 {
		return ""
	}

	if len(chunks) == 1 || topic == "" {
		return chunks[0]
	}

	keywords := t.ExtractTopicKeywords(topic)
	if len(keywords) == 0 {
		return chunks[0]
	}

	bestScore := -1
	bestChunk := chunks[0]

	for _, chunk := range chunks {
		score := t.ScoreContentRelevance(chunk, keywords)
		if score > bestScore {
			bestScore = score
			bestChunk = chunk
		}
	}

	return bestChunk
}

// SelectBestChunkWithContext selects the most relevant chunks and includes more content for better citations
func (t *TopicAnalyzer) SelectBestChunkWithContext(chunks []string, topic string) string {
	if len(chunks) == 0 {
		return ""
	}

	// For small documents, return everything
	if len(chunks) <= SmallDocumentThreshold {
		return strings.Join(chunks, "\n\n")
	}

	// If no topic provided, return first few chunks
	if topic == "" {
		maxChunks := MaxChunksToReturn
		if len(chunks) < maxChunks {
			maxChunks = len(chunks)
		}
		return strings.Join(chunks[:maxChunks], "\n\n")
	}

	keywords := t.ExtractTopicKeywords(topic)
	if len(keywords) == 0 {
		// No keywords, return first chunks up to max
		maxChunks := MaxChunksToReturn
		if len(chunks) < maxChunks {
			maxChunks = len(chunks)
		}
		return strings.Join(chunks[:maxChunks], "\n\n")
	}

	// Score all chunks and create a slice of indices with scores
	type chunkScore struct {
		index int
		score int
	}
	scores := make([]chunkScore, len(chunks))
	for i, chunk := range chunks {
		scores[i] = chunkScore{
			index: i,
			score: t.ScoreContentRelevance(chunk, keywords),
		}
	}

	for i := 0; i < len(scores); i++ {
		for j := i + 1; j < len(scores); j++ {
			if scores[j].score > scores[i].score {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}

	topCount := MaxChunksToReturn
	if len(scores) < topCount {
		topCount = len(scores)
	}

	topIndices := make([]int, topCount)
	for i := 0; i < topCount; i++ {
		topIndices[i] = scores[i].index
	}

	// Sort indices to maintain document order
	for i := 0; i < len(topIndices); i++ {
		for j := i + 1; j < len(topIndices); j++ {
			if topIndices[j] < topIndices[i] {
				topIndices[i], topIndices[j] = topIndices[j], topIndices[i]
			}
		}
	}

	var result strings.Builder
	for i, idx := range topIndices {
		if i > 0 {
			result.WriteString("\n\n")
		}
		result.WriteString(chunks[idx])
	}

	finalContent := result.String()
	if len(finalContent) < MinContentLengthForCitation && len(chunks) > topCount {
		for _, cs := range scores[topCount:] {
			finalContent = finalContent + "\n\n" + chunks[cs.index]
			if len(finalContent) >= MinContentLengthForCitation {
				break
			}
		}
	}

	return finalContent
}

// BuildExpandedSearchTerms creates a prioritized list of search terms with synonym expansion
func (t *TopicAnalyzer) BuildExpandedSearchTerms(topic string, maxTerms int) []string {
	if topic == "" {
		return nil
	}

	if maxTerms <= 0 {
		maxTerms = 10 // Default reasonable limit
	}

	// Start with original keywords
	keywords := t.ExtractTopicKeywords(topic)
	if len(keywords) == 0 {
		return nil
	}

	var expandedTerms []string

	topicLower := strings.ToLower(topic)
	expandedTerms = append(expandedTerms, topicLower)

	for _, keyword := range keywords {
		if keyword != topicLower { // Avoid duplicating full topic
			expandedTerms = append(expandedTerms, keyword)
		}
	}

	for _, keyword := range keywords {
		synonyms := t.GetTopicSynonyms(keyword)
		for _, synonym := range synonyms {
			// Prioritize multi-word synonyms as they're more specific
			if strings.Contains(synonym, " ") {
				expandedTerms = append(expandedTerms, synonym)
			}
		}
	}

	for _, keyword := range keywords {
		synonyms := t.GetTopicSynonyms(keyword)
		for _, synonym := range synonyms {
			if !strings.Contains(synonym, " ") {
				expandedTerms = append(expandedTerms, synonym)
			}
		}
	}

	// Deduplicate while preserving priority order
	deduplicated := t.deduplicateStrings(expandedTerms)

	// Limit to requested number of terms
	if len(deduplicated) > maxTerms {
		deduplicated = deduplicated[:maxTerms]
	}

	return deduplicated
}

// deduplicateStrings removes duplicate strings from a slice
func (t *TopicAnalyzer) deduplicateStrings(strs []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, s := range strs {
		if s != "" && !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}

	return result
}
