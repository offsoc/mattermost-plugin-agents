// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package roles

import (
	"github.com/mattermost/mattermost-plugin-ai/config"
	devconfig "github.com/mattermost/mattermost-plugin-ai/config/dev"
	pmconfig "github.com/mattermost/mattermost-plugin-ai/config/pm"
	"github.com/mattermost/mattermost-plugin-ai/conversations"
	"github.com/mattermost/mattermost-plugin-ai/llm"
	"github.com/mattermost/mattermost-plugin-ai/mmapi"
	"github.com/mattermost/mattermost-plugin-ai/roles/dev"
	"github.com/mattermost/mattermost-plugin-ai/roles/pm"
)

// RoleService manages specialized bot roles (PM, Dev, etc.)
type RoleService struct {
	mmClient        mmapi.Client
	prompts         *llm.Prompts
	configContainer *config.Container
}

// NewRoleService creates a new role service
func NewRoleService(mmClient mmapi.Client, prompts *llm.Prompts, configContainer *config.Container) *RoleService {
	return &RoleService{
		mmClient:        mmClient,
		prompts:         prompts,
		configContainer: configContainer,
	}
}

// RegisterAll registers all enabled bot roles with the conversations system
func (rs *RoleService) RegisterAll(conv *conversations.Conversations) {
	// Register PM role if enabled in config
	var pmConfig pmconfig.Config
	if err := rs.configContainer.GetRoleConfig("pm", &pmConfig); err == nil && pmConfig.Enabled {
		// Register PM validator when PM role is enabled
		pmconfig.RegisterValidator()

		pmService := pm.NewService(rs.mmClient, rs.prompts)
		pmService.RegisterWithConversations(conv)
	}

	// Register Dev role if enabled in config
	var devConfig devconfig.Config
	if err := rs.configContainer.GetRoleConfig("dev", &devConfig); err == nil && devConfig.Enabled {
		// Register Dev validator when Dev role is enabled
		devconfig.RegisterValidator()

		devService := dev.NewService(rs.mmClient, rs.prompts)
		devService.RegisterWithConversations(conv)
	}

	// Register SRE role if enabled (future)
	// if os.Getenv("ENABLE_SRE_BOT") == "true" {
	//     sreService := sre.NewService(rs.mmClient, rs.prompts)
	//     sreService.RegisterWithConversations(conv)
	// }
}
