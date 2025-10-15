// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import "time"

// Default configuration constants
const (
	DefaultCacheTTLHours = 24
	DefaultCacheTTL      = DefaultCacheTTLHours * time.Hour
)

// Content extraction constants
const (
	// Chunking configuration
	EnhancedChunkSize    = 5000 // Increased from default 1000 for better content extraction
	EnhancedChunkOverlap = 500  // Increased from default 200 for better context
	DefaultChunkSize     = 1000 // Original chunk size for comparison
	DefaultChunkOverlap  = 200  // Original overlap size
	MinChunkSizeFactor   = 0.5  // Allow chunks to be 50% of target size

	// Content selection
	MaxChunksToReturn           = 5                // Number of top-scored chunks to return
	SmallDocumentThreshold      = 3                // Documents with <= this many chunks are returned in full
	MinContentLengthForCitation = 2000             // Minimum characters needed for useful citations
	MaxHTMLSize                 = 10 * 1024 * 1024 // 10MB max HTML size

	// API content limits
	MaxJiraCommentsToInclude = 10 // Include more comments for better context (was 3)
)

// Source names
const (
	SourceMattermostDocs       = "mattermost_docs"
	SourceMattermostHandbook   = "mattermost_handbook"
	SourceMattermostForum      = "mattermost_forum"
	SourceMattermostBlog       = "mattermost_blog"
	SourceMattermostNewsroom   = "mattermost_newsroom"
	SourceGitHubRepos          = "github_repos"
	SourceCommunityForum       = "community_forum"
	SourceMattermostHub        = "mattermost_hub"
	SourceConfluenceDocs       = "confluence_docs"
	SourceJiraDocs             = "jira_docs"
	SourcePluginMarketplace    = "plugin_marketplace"
	SourceFeatureRequests      = "feature_requests"
	SourceProductBoardFeatures = "productboard_features"
	SourceZendeskTickets       = "zendesk_tickets"
)

const (
	AuthTypeNone   = "none"
	AuthTypeToken  = "token"
	AuthTypeAPIKey = "api_key"
)

const (
	HeaderAuthorization = "Authorization"
	HeaderContentType   = "Content-Type"
	HeaderUserAgent     = "User-Agent"
	HeaderAccept        = "Accept"
)

const (
	AuthPrefixBearer = "Bearer "
	AuthPrefixBasic  = "Basic "
	AuthPrefixToken  = "Token "
)

const (
	UserAgentMattermostPM  = "Mattermost-PM-Agent/1.0"
	UserAgentMattermostBot = "Mozilla/5.0 (compatible; MattermostBot/1.0)"
)

const (
	AcceptGitHubAPI = "application/vnd.github.v3+json"
	AcceptJSON      = "application/json"
	AcceptHTML      = "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"
)

const (
	HTTPProtocolType       ProtocolType = "http"
	GitHubAPIProtocolType  ProtocolType = "github_api"
	MattermostProtocolType ProtocolType = "mattermost"
	ConfluenceProtocolType ProtocolType = "confluence"
	UserVoiceProtocolType  ProtocolType = "uservoice"
	DiscourseProtocolType  ProtocolType = "discourse"
	JiraProtocolType       ProtocolType = "jira"
	FileProtocolType       ProtocolType = "file"
)

const (
	EndpointBaseURL        = "base_url"
	EndpointAdmin          = "admin"
	EndpointDeveloper      = "developer"
	EndpointAPI            = "api"
	EndpointMobile         = "mobile"
	EndpointMobileApps     = "mobile_apps"
	EndpointMobileStrategy = "mobile_strategy"
)

const (
	EndpointOwner = "owner"
	EndpointRepos = "repos"
)

const (
	EndpointSpaces = "spaces"
	EndpointSearch = "search"
	EndpointEmail  = "email"
)

const (
	EndpointFilePath = "file_path"
)

// Confluence space configurations
const (
	ConfluenceSpaces = "DES,FF,WD,WKFL,EN,APPS,AFC,ATE,CI,CS,CSO,EGE,GLOAB,HPN,JM,MCI,MSU,RM,SCA,Threads,Toolkit" // Mattermost Confluence spaces
)

const (
	GitHubOwnerMattermost = "mattermost"
	GitHubReposList       = "mattermost,mattermost-mobile,desktop,docs,mattermost-plugin-playbooks,mattermost-plugin-calls,mattermost-plugin-github,mattermost-plugin-jira,mattermost-plugin-zoom,mattermost-plugin-boards,mmctl,mattermost-push-proxy,mattermost-webapp"
)

const (
	MattermostDocsURL     = "https://docs.mattermost.com"
	MattermostHandbookURL = "https://handbook.mattermost.com"
	MattermostForumURL    = "https://forum.mattermost.com"
	MattermostBlogURL     = "https://mattermost.com/blog"
	MattermostNewsroomURL = "https://mattermost.com/newsroom"
	GitHubAPIURL          = "https://api.github.com"
	CommunityForumURL     = "https://community.mattermost.com"
	MattermostHubURL      = "https://hub.mattermost.com"
	ConfluenceURL         = "https://mattermost.atlassian.net/wiki"
	JiraURL               = "https://mattermost.atlassian.net"
	PluginMarketplaceURL  = "https://mattermost.com/marketplace"
	IntegrationsURL       = "https://integrations.mattermost.com"
	FeatureRequestsURL    = "https://mattermost.uservoice.com"
)

// DefaultAllowedDomains contains the default list of allowed domains for datasource access
var DefaultAllowedDomains = []string{
	"docs.mattermost.com",
	"handbook.mattermost.com",
	"forum.mattermost.com",
	"mattermost.com",
	"api.github.com",
	"community.mattermost.com",
	"hub.mattermost.com",
	"mattermost.atlassian.net",
	"integrations.mattermost.com",
}

// Documentation paths
const (
	DocsAdminPath          = "/administration-guide/administration-guide-index.html"
	DocsDeveloperPath      = "/deployment-guide/deployment-guide-index.html"
	DocsAPIPath            = "/administration/changelog.html"
	DocsMobilePath         = "/deployment-guide/mobile/mobile-app-deployment.html"
	DocsMobileAppsPath     = "/deployment-guide/mobile/mobile-faq.html"
	DocsMobileStrategyPath = "/product-overview/mattermost-mobile-releases.html"
)

// Handbook paths
const (
	HandbookCompanyPath     = "/company"
	HandbookOperationsPath  = "/operations"
	HandbookEngineeringPath = "/operations/research-and-development"
	HandbookPeoplePath      = "/operations/workplace/people"
	HandbookMarketingPath   = "/operations/messaging-and-math"
	HandbookSalesPath       = "/operations/sales"
	HandbookSupportPath     = "/operations/customer-success"
	HandbookSecurityPath    = "/operations/security"
)

const (
	NewsroomNewsPath         = "/#news-section"
	NewsroomPressReleasePath = "/#press-releases-section"
	NewsroomPressKitPath     = "/#press-kit-section"
)

const (
	BlogPlatformPath    = "/category/platform/"
	BlogEngineeringPath = "/category/engineering/"
	BlogCommunityPath   = "/category/community/"
)

const (
	SectionAdmin                = "admin"
	SectionDeveloper            = "developer"
	SectionAPI                  = "api"
	SectionMobile               = "mobile"
	SectionMobileApps           = "mobile_apps"
	SectionMobileStrategy       = "mobile_strategy"
	SectionIssues               = "issues"
	SectionReleases             = "releases"
	SectionPulls                = "pulls"
	SectionCode                 = "code"
	SectionFeatureRequests      = "feature-requests"
	SectionTroubleshooting      = "troubleshooting"
	SectionGeneral              = "general"
	SectionBugs                 = "bugs"
	SectionAskAnything          = "ask-anything"
	SectionAskRnD               = "ask-r-and-d"
	SectionAIExchange           = "ai-exchange"
	SectionAccessibility        = "accessibility"
	SectionAPIv4                = "apiv4"
	SectionPlugins              = "plugins"
	SectionIntegrations         = "integrations"
	SectionContactSales         = "contact-sales"
	SectionCustomerFeedback     = "customer-feedback"
	SectionChannels             = "channels"
	SectionCompany              = "company"
	SectionOperations           = "operations"
	SectionEngineering          = "engineering"
	SectionPeople               = "people"
	SectionMarketing            = "marketing"
	SectionSales                = "sales"
	SectionSupport              = "support"
	SectionSecurity             = "security"
	SectionAnnouncements        = "announce"
	SectionFAQ                  = "faq"
	SectionForumTroubleshooting = "trouble-shoot"
	SectionUserFeedback         = "feedback"
	SectionCopilotAI            = "copilot-ai"
	SectionRecipes              = "recipes"
	SectionBlogPosts            = "blog-posts"
	SectionTechnical            = "technical"
	SectionNews                 = "news"
	SectionPressReleases        = "press-releases"
	SectionMediaKit             = "media-kit"
	SectionProductRequirements  = "product-requirements"
	SectionMarketResearch       = "market-research"
	SectionFeatureSpecs         = "feature-specs"
	SectionRoadmaps             = "roadmaps"
	SectionCompetitiveAnalysis  = "competitive-analysis"
	SectionCustomerInsights     = "customer-insights"
	SectionBug                  = "bug"
	SectionTask                 = "task"
	SectionStory                = "story"
	SectionEpic                 = "epic"
	SectionSpike                = "spike"
	SectionSubtask              = "subtask"
	SectionFeatures             = "features"
	SectionDelivered            = "delivered"
	SectionIdeas                = "ideas"
	SectionInProgress           = "in-progress"
	SectionCritical             = "critical"
	SectionProduction           = "production"
)

// Rate limiting defaults
const (
	DefaultRequestsPerMinuteHTTP       = 30
	DefaultBurstSizeHTTP               = 5
	DefaultRequestsPerMinuteGitHub     = 60
	DefaultBurstSizeGitHub             = 10
	DefaultRequestsPerMinuteCommunity  = 20
	DefaultBurstSizeCommunity          = 3
	DefaultRequestsPerMinuteConfluence = 15
	DefaultBurstSizeConfluence         = 3
	DefaultRequestsPerMinuteJira       = 15
	DefaultBurstSizeJira               = 3
	DefaultRequestsPerMinuteFile       = 100
	DefaultBurstSizeFile               = 20
)

// Document limits
const (
	DefaultMaxDocsPerCallHTTP      = 5
	DefaultMaxDocsPerCallGitHub    = 10
	DefaultMaxDocsPerCallCommunity = 5
	DefaultMaxDocsPerCallJira      = 10
	DefaultMaxDocsPerCallFile      = 20
)

// Environment variable names
const (
	EnvGitHubToken = "MM_AI_GITHUB_TOKEN" // #nosec G101 - this is an environment variable name, not a credential
)

// Environment variable pattern for source tokens
const (
	EnvSourceTokenPrefix = "MM_AI_"
	EnvSourceTokenSuffix = "_TOKEN"
)

// Cache key format components
const (
	CacheKeySeparator = ":"
)

// Topic expansion limits
const (
	DefaultMaxExpandedTerms    = 10
	MaxExpandedTermsConfluence = 8  // Conservative for CQL character limits
	MaxExpandedTermsGitHub     = 12 // GitHub API can handle more terms
	MaxExpandedTermsMattermost = 10 // Balanced for Mattermost search
	MaxExpandedTermsHTTP       = 6  // Conservative for generic HTTP endpoints
	MaxExpandedTermsJira       = 8  // Conservative for JQL character limits
)

// Universal relevance scoring thresholds
const (
	MinContentQualityScore    = 20 // Minimum content quality score (out of 40)
	MinSemanticRelevanceScore = 15 // Minimum semantic relevance score (out of 40)
	MinSourceAuthorityScore   = 5  // Minimum source authority score (out of 20)
	MinTotalRelevanceScore    = 50 // Minimum total score to pass filter (out of 100)

	// Content quality scoring weights
	ContentLengthWeight      = 15 // Points for adequate content length
	InformationDensityWeight = 10 // Points for information density
	LanguageQualityWeight    = 10 // Points for language quality
	StructureQualityWeight   = 5  // Points for content structure

	// Basic content requirements
	MinContentLength          = 80   // Minimum character count for acceptable content
	MaxRepetitionRatio        = 0.4  // Maximum ratio of repeated content
	MinUniqueWordsRatio       = 0.6  // Minimum ratio of unique words
	MaxNavigationKeywordRatio = 0.05 // Maximum ratio of navigation keyword hits vs. characters

	// Documentation quality improvements
	DocumentationTitleBonus = 2     // Bonus points for documentation-style titles
	LongContentThreshold    = 10000 // Character threshold for relaxed quality checks

	// Content size thresholds for processing decisions
	ShortContentThreshold = 500 // Content below this length gets different processing (less strict)
	ContentPreviewLength  = 200 // Length for content/HTML previews in logging
)

// Fallback data constants - for when we don't have access to real sources
const (
	FallbackDataDirectory          = "../assets/fallback-data"
	HubContactSalesChannelData     = "hub-contact-sales.txt"
	HubCustomerFeedbackChannelData = "hub-customer-feedback.txt"
)

// Documentation keyword sets for quality assessment
var (
	DocumentationTitleKeywords = []string{
		"guide", "faq", "documentation", "tutorial", "how-to", "reference",
		"troubleshooting", "deployment", "administration", "setup", "installation",
		"configuration", "manual", "handbook", "walkthrough", "instructions",
	}
)

// Test timeouts
const (
	DefaultTestTimeout     = 30 * time.Second
	ExtendedTestTimeout    = 60 * time.Second
	LongRunningTestTimeout = 300 * time.Second
)

// HTTP client configuration
const (
	DefaultHTTPClientTimeout = 30 * time.Second
)

// Query limits and thresholds
const (
	DefaultMaxQueryLength = 256
	DefaultMaxOperators   = 10
	DisplayTruncateLength = 100
	MaxBooleanQueryDepth  = 20 // Maximum nesting depth for boolean query parser to prevent stack exhaustion
)

// Query parameter names used across multiple API protocols
const (
	QueryParamPerPage = "per_page"
	QueryParamPage    = "page"
	QueryParamSort    = "sort"
	QueryParamState   = "state"
	QueryParamAll     = "all"
)

// Status and state values used across protocols
const (
	StatusOpen       = "open"
	StatusClosed     = "closed"
	StatusCompleted  = "completed"
	StatusShipped    = "shipped"
	StatusInProgress = "in_progress"
	StatusPlanned    = "planned"
	StatusDeclined   = "declined"
	StatusRejected   = "rejected"
)

// Error messages
const (
	ErrorSourceNotFound       = "source %s not found or not enabled"
	ErrorProtocolNotSupported = "protocol %s not supported"
	ErrorFailedToFetch        = "failed to fetch from %s: %w"
)
