// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package config

import (
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/mattermost/mattermost-plugin-ai/embeddings"
	"github.com/mattermost/mattermost-plugin-ai/llm"
	"github.com/mattermost/mattermost-plugin-ai/mcp"
	"github.com/mattermost/mattermost-plugin-ai/openai"
)

type Config struct {
	Services                 []llm.ServiceConfig              `json:"services"`
	Bots                     []llm.BotConfig                  `json:"bots"`
	DefaultBotName           string                           `json:"defaultBotName"`
	TranscriptGenerator      string                           `json:"transcriptBackend"`
	EnableLLMTrace           bool                             `json:"enableLLMTrace"`
	EnableTokenUsageLogging  bool                             `json:"enableTokenUsageLogging"`
	AllowedUpstreamHostnames string                           `json:"allowedUpstreamHostnames"`
	EmbeddingSearchConfig    embeddings.EmbeddingSearchConfig `json:"embeddingSearchConfig"`
	MCP                      mcp.Config                       `json:"mcp"`

	// Role-specific configurations - extensible without changing core
	// Shared resources (like datasources) should be stored in RoleConfigs["shared"]
	RoleConfigs map[string]json.RawMessage `json:"roleConfigs,omitempty"`
}

func (c *Config) Clone() *Config {
	clone, err := DeepCopyJSON(*c)
	if err != nil {
		panic(fmt.Sprintf("failed to clone configuration: %v", err))
	}

	return &clone
}

type UpdateListener func()

type Container struct {
	cfg       atomic.Pointer[Config]
	listeners []UpdateListener
}

// Config retruns the whole configuration readonly.
// Avoid using this method, prefer using config though interfaces.
func (c *Container) Config() *Config {
	return c.cfg.Load()
}

func (c *Container) GetEnableLLMTrace() bool {
	cfg := c.cfg.Load()
	if cfg == nil {
		return false
	}
	return cfg.EnableLLMTrace
}

func (c *Container) GetTranscriptGenerator() string {
	cfg := c.cfg.Load()
	if cfg == nil {
		return ""
	}
	return cfg.TranscriptGenerator
}

func (c *Container) GetBots() []llm.BotConfig {
	cfg := c.cfg.Load()
	if cfg == nil {
		return nil
	}
	return cfg.Bots
}

func (c *Container) GetDefaultBotName() string {
	cfg := c.cfg.Load()
	if cfg == nil {
		return ""
	}
	return cfg.DefaultBotName
}

func (c *Container) EnableLLMLogging() bool {
	cfg := c.cfg.Load()
	if cfg == nil {
		return false
	}
	return cfg.EnableLLMTrace
}

func (c *Container) EnableTokenUsageLogging() bool {
	return c.cfg.Load().EnableTokenUsageLogging
}

func (c *Container) MCP() mcp.Config {
	cfg := c.cfg.Load()
	if cfg == nil {
		return mcp.Config{}
	}
	return cfg.MCP
}

// GetRoleConfig retrieves and unmarshals a role-specific configuration
func (c *Container) GetRoleConfig(roleName string, target interface{}) error {
	cfg := c.cfg.Load()
	if cfg == nil || cfg.RoleConfigs == nil {
		return fmt.Errorf("no role configurations available")
	}

	rawConfig, exists := cfg.RoleConfigs[roleName]
	if !exists {
		return fmt.Errorf("no configuration found for role: %s", roleName)
	}

	return json.Unmarshal(rawConfig, target)
}

func (c *Container) RegisterUpdateListener(listener UpdateListener) {
	c.listeners = append(c.listeners, listener)
}

func (c *Container) EmbeddingSearchConfig() embeddings.EmbeddingSearchConfig {
	cfg := c.cfg.Load()
	if cfg == nil {
		return embeddings.EmbeddingSearchConfig{}
	}
	return cfg.EmbeddingSearchConfig
}

// Updates the current configuration
// The new configuration is deep-copied to ensure the new and old
// configurations are independent of each other.
func (c *Container) Update(newConfig *Config) {
	if newConfig == nil {
		c.cfg.Store(nil)
		return
	}

	// Create a deep copy of the new configuration
	clone, err := DeepCopyJSON(*newConfig)
	if err != nil {
		panic(fmt.Sprintf("failed to deep copy configuration: %v", err))
	}

	// Update the atomic pointer with the new configuration
	c.cfg.Store(&clone)

	// Notify all listeners about the configuration change
	for _, listener := range c.listeners {
		listener()
	}
}

// DeepCopyJSON creates a deep copy of JSON-serializable structs
func DeepCopyJSON[T any](src T) (T, error) {
	var dst T
	data, err := json.Marshal(src)
	if err != nil {
		return dst, err
	}
	err = json.Unmarshal(data, &dst)
	return dst, err
}

func OpenAIConfigFromServiceConfig(serviceConfig llm.ServiceConfig) openai.Config {
	streamingTimeout := time.Second * 30
	if serviceConfig.StreamingTimeoutSeconds > 0 {
		streamingTimeout = time.Duration(serviceConfig.StreamingTimeoutSeconds) * time.Second
	}

	return openai.Config{
		APIKey:             serviceConfig.APIKey,
		APIURL:             serviceConfig.APIURL,
		OrgID:              serviceConfig.OrgID,
		DefaultModel:       serviceConfig.DefaultModel,
		InputTokenLimit:    serviceConfig.InputTokenLimit,
		OutputTokenLimit:   serviceConfig.OutputTokenLimit,
		StreamingTimeout:   streamingTimeout,
		SendUserID:         serviceConfig.SendUserID,
		UseResponsesAPI:    serviceConfig.UseResponsesAPI,
		EnabledNativeTools: serviceConfig.EnabledNativeTools,
		DefaultTemperature: serviceConfig.DefaultTemperature,
	}
}
