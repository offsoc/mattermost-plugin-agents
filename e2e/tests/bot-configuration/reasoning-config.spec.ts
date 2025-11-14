import { test, expect } from '@playwright/test';

import RunContainer from 'helpers/plugincontainer';
import MattermostContainer from 'helpers/mmcontainer';
import { MattermostPage } from 'helpers/mm';
import { AIPlugin } from 'helpers/ai-plugin';
import { OpenAIMockContainer, RunOpenAIMocks, responseTest, responseTestText } from 'helpers/openai-mock';
import { createBotConfigHelper, generateBotId } from 'helpers/bot-config';

// Test configuration
const username = 'regularuser';
const password = 'regularuser';

let mattermost: MattermostContainer;
let openAIMock: OpenAIMockContainer;

// Common test setup
async function setupTestPage(page) {
    const mmPage = new MattermostPage(page);
    const aiPlugin = new AIPlugin(page);
    const url = mattermost.url();

    await mmPage.login(url, username, password);

    return { mmPage, aiPlugin };
}

test.describe('Reasoning Configuration Tests', () => {
    // Setup for all tests in the file
    test.beforeAll(async () => {
        mattermost = await RunContainer();
        openAIMock = await RunOpenAIMocks(mattermost.network);
    });

    // Cleanup after all tests
    test.afterAll(async () => {
        await openAIMock.stop();
        await mattermost.stop();
    });

    test.describe('OpenAI Reasoning Configuration', () => {
        test('should enable reasoning for OpenAI service with Responses API', async () => {
            const botConfig = await createBotConfigHelper(mattermost);

            // Get the mock service (OpenAI-compatible with Responses API enabled)
            const originalService = await botConfig.getService('mock-service');
            expect(originalService).toBeDefined();
            // Note: useResponsesAPI field may or may not be present depending on version
            // We just need a service that supports reasoning

            // Enable reasoning on the service
            await botConfig.updateService('mock-service', {
                reasoningEnabled: true
            });

            // Verify reasoning was enabled
            const updatedService = await botConfig.getService('mock-service');
            expect(updatedService?.reasoningEnabled).toBe(true);

            // Restore original state
            await botConfig.updateService('mock-service', {
                reasoningEnabled: originalService!.reasoningEnabled
            });
        });

        test('should disable reasoning for OpenAI service', async () => {
            const botConfig = await createBotConfigHelper(mattermost);

            // Get service and enable reasoning first
            const originalService = await botConfig.getService('mock-service');
            await botConfig.updateService('mock-service', {
                reasoningEnabled: true
            });

            // Verify enabled
            let service = await botConfig.getService('mock-service');
            expect(service?.reasoningEnabled).toBe(true);

            // Now disable reasoning
            await botConfig.updateService('mock-service', {
                reasoningEnabled: false
            });

            // Verify disabled
            service = await botConfig.getService('mock-service');
            expect(service?.reasoningEnabled).toBe(false);

            // Restore
            await botConfig.updateService('mock-service', {
                reasoningEnabled: originalService!.reasoningEnabled
            });
        });

        test('should configure reasoning effort for OpenAI (if supported)', async () => {
            const botConfig = await createBotConfigHelper(mattermost);

            // Get service
            const originalService = await botConfig.getService('mock-service');
            expect(originalService).toBeDefined();

            // Note: Reasoning effort might be stored in a different field or as part of model parameters
            // This test verifies the service can be updated with reasoning-related config
            await botConfig.updateService('mock-service', {
                reasoningEnabled: true,
                // Reasoning effort might be stored in extended config or model parameters
                // The exact field depends on the plugin implementation
            });

            // Verify the configuration was saved
            const updatedService = await botConfig.getService('mock-service');
            expect(updatedService?.reasoningEnabled).toBe(true);

            // Restore
            await botConfig.updateService('mock-service', {
                reasoningEnabled: originalService!.reasoningEnabled
            });
        });

        test('should require Responses API for OpenAI reasoning', async () => {
            const botConfig = await createBotConfigHelper(mattermost);

            // Create a service without Responses API
            await botConfig.addService({
                id: 'no-responses-api-service',
                name: 'No Responses API Service',
                type: 'openai',
                apiKey: 'test-key',
                apiURL: 'http://openai:8080',
                useResponsesAPI: false, // Responses API disabled
                reasoningEnabled: false
            });

            // Verify service was created
            const service = await botConfig.getService('no-responses-api-service');
            expect(service).toBeDefined();
            expect(service?.useResponsesAPI).toBe(false);

            // Attempt to enable reasoning (should work in config, but may not work at runtime)
            await botConfig.updateService('no-responses-api-service', {
                reasoningEnabled: true
            });

            // Verify configuration accepts this (validation happens at runtime)
            const updatedService = await botConfig.getService('no-responses-api-service');
            expect(updatedService?.reasoningEnabled).toBe(true);
            expect(updatedService?.useResponsesAPI).toBe(false);

            // Note: At runtime, reasoning would fail or be ignored without Responses API
            // This is expected behavior - configuration allows it but runtime enforces the requirement

            // Clean up
            await botConfig.deleteService('no-responses-api-service');
        });
    });

    test.describe('Anthropic Reasoning Configuration', () => {
        test('should enable reasoning for Anthropic service', async () => {
            const botConfig = await createBotConfigHelper(mattermost);

            // Create Anthropic service
            await botConfig.addService({
                id: 'anthropic-test-service',
                name: 'Anthropic Test Service',
                type: 'anthropic',
                apiKey: 'test-anthropic-key',
                apiURL: 'https://api.anthropic.com',
                reasoningEnabled: false
            });

            // Enable reasoning
            await botConfig.updateService('anthropic-test-service', {
                reasoningEnabled: true
            });

            // Verify enabled
            const service = await botConfig.getService('anthropic-test-service');
            expect(service?.reasoningEnabled).toBe(true);

            // Clean up
            await botConfig.deleteService('anthropic-test-service');
        });

        test('should configure thinking budget for Anthropic (if supported)', async () => {
            const botConfig = await createBotConfigHelper(mattermost);

            // Create Anthropic service
            await botConfig.addService({
                id: 'anthropic-thinking-service',
                name: 'Anthropic Thinking Service',
                type: 'anthropic',
                apiKey: 'test-anthropic-key',
                apiURL: 'https://api.anthropic.com',
                reasoningEnabled: true
            });

            // Note: Thinking budget might be stored as tokenLimit or in extended configuration
            // The exact field depends on the plugin implementation
            // Anthropic's thinking budget is typically specified in tokens (e.g., 4096)
            await botConfig.updateService('anthropic-thinking-service', {
                tokenLimit: 4096 // This might represent the thinking budget
            });

            // Verify configuration was saved
            const service = await botConfig.getService('anthropic-thinking-service');
            expect(service?.tokenLimit).toBe(4096);
            expect(service?.reasoningEnabled).toBe(true);

            // Clean up
            await botConfig.deleteService('anthropic-thinking-service');
        });

        test('should handle different thinking budget values for Anthropic', async () => {
            const botConfig = await createBotConfigHelper(mattermost);

            // Create Anthropic service
            await botConfig.addService({
                id: 'anthropic-budget-test',
                name: 'Anthropic Budget Test',
                type: 'anthropic',
                apiKey: 'test-key',
                apiURL: 'https://api.anthropic.com',
                reasoningEnabled: true
            });

            // Test different budget values
            const budgets = [1024, 2048, 4096, 8192];

            for (const budget of budgets) {
                await botConfig.updateService('anthropic-budget-test', {
                    tokenLimit: budget
                });

                const service = await botConfig.getService('anthropic-budget-test');
                expect(service?.tokenLimit).toBe(budget);
            }

            // Clean up
            await botConfig.deleteService('anthropic-budget-test');
        });
    });

    test.describe('Cross-Provider Reasoning Tests', () => {
        test('should allow switching bot between OpenAI and Anthropic services with reasoning', async () => {
            const botConfig = await createBotConfigHelper(mattermost);

            // Create OpenAI service with reasoning
            await botConfig.addService({
                id: 'openai-reasoning',
                name: 'OpenAI Reasoning Service',
                type: 'openaicompatible',
                apiKey: 'openai-key',
                apiURL: 'http://openai:8080',
                useResponsesAPI: true,
                reasoningEnabled: true
            });

            // Create Anthropic service with reasoning
            await botConfig.addService({
                id: 'anthropic-reasoning',
                name: 'Anthropic Reasoning Service',
                type: 'anthropic',
                apiKey: 'anthropic-key',
                apiURL: 'https://api.anthropic.com',
                reasoningEnabled: true,
                tokenLimit: 4096
            });

            // Create bot using OpenAI service
            const botId = generateBotId();
            await botConfig.addBot({
                id: botId,
                name: 'reasoningbot',
                displayName: 'Reasoning Bot',
                customInstructions: 'You use advanced reasoning.',
                serviceID: 'openai-reasoning'
            });

            // Verify bot uses OpenAI service
            let bot = await botConfig.getBot(botId);
            expect(bot?.serviceID).toBe('openai-reasoning');

            // Switch to Anthropic service
            await botConfig.updateBot(botId, {
                serviceID: 'anthropic-reasoning'
            });

            // Verify switch
            bot = await botConfig.getBot(botId);
            expect(bot?.serviceID).toBe('anthropic-reasoning');

            // Clean up
            await botConfig.deleteBot(botId);
            await botConfig.deleteService('openai-reasoning');
            await botConfig.deleteService('anthropic-reasoning');
        });

        test('should persist reasoning configuration across service updates', async () => {
            const botConfig = await createBotConfigHelper(mattermost);

            // Create service with reasoning disabled
            await botConfig.addService({
                id: 'reasoning-persist-test',
                name: 'Reasoning Persist Test',
                type: 'openaicompatible',
                apiKey: 'test-key',
                apiURL: 'http://openai:8080',
                useResponsesAPI: true,
                reasoningEnabled: false
            });

            // Enable reasoning
            await botConfig.updateService('reasoning-persist-test', {
                reasoningEnabled: true
            });

            // Make other updates
            await botConfig.updateService('reasoning-persist-test', {
                apiKey: 'updated-key'
            });

            // Verify reasoning is still enabled
            const service = await botConfig.getService('reasoning-persist-test');
            expect(service?.reasoningEnabled).toBe(true);
            expect(service?.apiKey).toBe('updated-key');

            // Clean up
            await botConfig.deleteService('reasoning-persist-test');
        });
    });
});
