// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"os"
	"path/filepath"

	"github.com/mattermost/mattermost-plugin-ai/mmapi"
)

// MattermostFallbackHandler manages fallback/mock data loading for Hub channels
type MattermostFallbackHandler struct {
	pluginAPI         mmapi.Client
	transformer       *MattermostTransformer
	fallbackDirectory string
}

// NewMattermostFallbackHandler creates a new fallback handler
func NewMattermostFallbackHandler(pluginAPI mmapi.Client, fallbackDirectory string) *MattermostFallbackHandler {
	return &MattermostFallbackHandler{
		pluginAPI:         pluginAPI,
		transformer:       NewMattermostTransformer(),
		fallbackDirectory: fallbackDirectory,
	}
}

// LoadHubMockData loads Hub channel content from fallback data when API access isn't available
func (h *MattermostFallbackHandler) LoadHubMockData(section, sourceName, baseURL, channelName, topic string) []Doc {
	var fallbackFileName string
	switch section {
	case SectionContactSales:
		fallbackFileName = HubContactSalesChannelData
	case SectionCustomerFeedback:
		fallbackFileName = HubCustomerFeedbackChannelData
	default:
		if h.pluginAPI != nil {
			h.pluginAPI.LogDebug("Hub fallback section not mapped", "section", section, "source", sourceName, "reason", "no_fallback_file_for_section")
		}
		return nil
	}

	fallbackDataDirectory := h.fallbackDirectory
	if fallbackDataDirectory == "" {
		fallbackDataDirectory = FallbackDataDirectory
	}

	fallbackFilePath := filepath.Join(fallbackDataDirectory, fallbackFileName)

	content, err := os.ReadFile(fallbackFilePath)
	if err != nil {
		if h.pluginAPI != nil {
			h.pluginAPI.LogWarn(sourceName+": fallback file not found", "section", section, "file", fallbackFilePath, "error", err.Error())
		}
		return nil
	}

	if h.pluginAPI != nil {
		h.pluginAPI.LogDebug(sourceName+": loaded fallback file", "section", section, "file", fallbackFilePath, "bytes", len(content))
	}

	fileProtocol := NewFileProtocol(h.pluginAPI)
	posts := fileProtocol.parseHubPosts(string(content))

	var docs []Doc
	for _, post := range posts {
		doc := fileProtocol.hubPostToDoc(post, sourceName)
		docs = append(docs, doc)
	}

	if topic != "" {
		filteredDocs := h.transformer.FilterDocsByTopic(docs, topic)
		if h.pluginAPI != nil {
			h.pluginAPI.LogDebug(sourceName+": fallback search results", "section", section, "topic", TruncateTopicForLogging(topic), "total", len(docs), "matched", len(filteredDocs))
		}
		return filteredDocs
	}

	if h.pluginAPI != nil {
		h.pluginAPI.LogDebug(sourceName+": fallback browse results", "section", section, "docs", len(docs))
	}
	return docs
}
