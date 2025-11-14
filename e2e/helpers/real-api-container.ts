import fs from 'fs';
import MattermostContainer from './mmcontainer';
import { ProviderConfig } from './api-config';

/**
 * Container setup for LLMBot tests using REAL APIs
 * No mock containers - plugin calls OpenAI/Anthropic directly
 */

export async function RunRealAPIContainer(provider: ProviderConfig): Promise<MattermostContainer> {
    let filename = "";
    fs.readdirSync("../dist/").forEach(file => {
        if (file.endsWith(".tar.gz")) {
            filename = "../dist/" + file;
        }
    });

    const botName = provider.type === 'anthropic' ? 'claude' : 'mockbot';
    const serviceConfig: any = {
        "id": `${provider.name.toLowerCase()}-service`,
        "name": `${provider.name} Service`,
        "type": provider.type,
        "apiKey": provider.apiKey,
        "apiURL": provider.apiURL,
        "defaultModel": provider.defaultModel,
    };

    // Add OpenAI-specific config
    if (provider.type === 'openaicompatible') {
        // Use Responses API for reasoning support with o3/o4 models
        serviceConfig.useResponsesAPI = true;
        if (provider.reasoningEffort) {
            serviceConfig.reasoningEffort = provider.reasoningEffort;
        }
    }

    const botConfig: any = {
        "id": `${provider.name.toLowerCase()}-bot-id`,
        "name": botName,
        "displayName": `${provider.name} Bot`,
        "customInstructions": "",
        "serviceID": serviceConfig.id,
        "reasoningEnabled": provider.reasoningEnabled,
        "enabledNativeTools": ["web_search"],
    };

    // Add Anthropic-specific config
    if (provider.type === 'anthropic') {
        if (provider.thinkingBudget) {
            botConfig.thinkingBudget = provider.thinkingBudget;
        }
        // Set maxTokens to ensure it's greater than thinkingBudget
        serviceConfig.maxTokens = 16384;
    }

    const pluginConfig = {
        "config": {
            "allowPrivateChannels": true,
            "disableFunctionCalls": false,
            "enableLLMTrace": true,
            "enableUserRestrictions": false,
            "defaultBotName": botName,
            "enableVectorIndex": true,
            "services": [serviceConfig],
            "bots": [botConfig],
        }
    };

    const mattermost = await new MattermostContainer()
        .withPlugin(filename, "mattermost-ai", pluginConfig)
        .start();

    // Create test users
    await mattermost.createUser("regularuser@sample.com", "regularuser", "regularuser");
    await mattermost.addUserToTeam("regularuser", "test");
    await mattermost.createUser("seconduser@sample.com", "seconduser", "seconduser");
    await mattermost.addUserToTeam("seconduser", "test");

    const userClient = await mattermost.getClient("regularuser", "regularuser");
    const user = await userClient.getMe();

    await userClient.savePreferences(user.id, [
        {user_id: user.id, category: 'tutorial_step', name: user.id, value: '999'},
        {user_id: user.id, category: 'onboarding_task_list', name: 'onboarding_task_list_show', value: 'false'},
        {user_id: user.id, category: 'onboarding_task_list', name: 'onboarding_task_list_open', value: 'false'},
        {
            user_id: user.id,
            category: 'drafts',
            name: 'drafts_tour_tip_showed',
            value: JSON.stringify({drafts_tour_tip_showed: true}),
        },
        {user_id: user.id, category: 'crt_thread_pane_step', name: user.id, value: '999'},
    ]);

    const adminClient = await mattermost.getAdminClient();
    const admin = await adminClient.getMe();

    await adminClient.savePreferences(admin.id, [
        {user_id: admin.id, category: 'tutorial_step', name: admin.id, value: '999'},
        {user_id: admin.id, category: 'onboarding_task_list', name: 'onboarding_task_list_show', value: 'false'},
        {user_id: admin.id, category: 'onboarding_task_list', name: 'onboarding_task_list_open', value: 'false'},
        {
            user_id: admin.id,
            category: 'drafts',
            name: 'drafts_tour_tip_showed',
            value: JSON.stringify({drafts_tour_tip_showed: true}),
        },
        {user_id: admin.id, category: 'crt_thread_pane_step', name: admin.id, value: '999'},
    ]);

    await adminClient.completeSetup({
        organization: "test",
        install_plugins: [],
    });

    await new Promise(resolve => setTimeout(resolve, 1000));

    return mattermost;
}

export default RunRealAPIContainer;
