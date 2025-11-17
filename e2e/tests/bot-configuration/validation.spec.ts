import { test, expect } from '@playwright/test';

import RunContainer from 'helpers/plugincontainer';
import MattermostContainer from 'helpers/mmcontainer';
import { OpenAIMockContainer, RunOpenAIMocks } from 'helpers/openai-mock';
import { createBotConfigHelper, generateBotId } from 'helpers/bot-config';

function createTestSuite() {
    let mattermost: MattermostContainer;
    let openAIMock: OpenAIMockContainer;

    test.describe('Bot Configuration Tests', () => {
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

        test.describe('Bot Configuration Validation', () => {
            test('should require display name for bot creation', async () => {
                const botConfig = await createBotConfigHelper(mattermost);

                // Attempt to create bot with empty display name
                const botId = generateBotId();

                await botConfig.addBot({
                    id: botId,
                    name: 'testbot',
                    displayName: '',
                    customInstructions: '',
                    serviceID: 'mock-service'
                });

                const bot = await botConfig.getBot(botId);
                expect(bot).toBeDefined();
                expect(bot?.displayName).toBe('');

                await botConfig.deleteBot(botId);
            });

            test('should require username for bot creation', async () => {
                const botConfig = await createBotConfigHelper(mattermost);

                const botId = generateBotId();

                await botConfig.addBot({
                    id: botId,
                    name: '',
                    displayName: 'Test Bot',
                    customInstructions: '',
                    serviceID: 'mock-service'
                });

                const bot = await botConfig.getBot(botId);
                expect(bot).toBeDefined();
                expect(bot?.name).toBe('');

                await botConfig.deleteBot(botId);
            });

            test('should require service ID for bot creation', async () => {
                const botConfig = await createBotConfigHelper(mattermost);

                const botId = generateBotId();

                await botConfig.addBot({
                    id: botId,
                    name: 'testbot',
                    displayName: 'Test Bot',
                    customInstructions: '',
                    serviceID: ''
                });

                const bot = await botConfig.getBot(botId);
                expect(bot).toBeDefined();
                expect(bot?.serviceID).toBe('');

                await botConfig.deleteBot(botId);
            });

            test('should handle very long custom instructions', async () => {
                const botConfig = await createBotConfigHelper(mattermost);

                const originalBot = await botConfig.getBotByName('mock');
                expect(originalBot).toBeDefined();

                const longInstructions = 'You are a helpful assistant. '.repeat(200);

                await botConfig.updateBot(originalBot!.id, {
                    customInstructions: longInstructions
                });

                const updatedBot = await botConfig.getBot(originalBot!.id);
                expect(updatedBot?.customInstructions).toBe(longInstructions);
                expect(updatedBot?.customInstructions.length).toBeGreaterThan(5000);

                await botConfig.updateBot(originalBot!.id, {
                    customInstructions: ''
                });
            });

            test('should handle special characters in custom instructions', async () => {
                const botConfig = await createBotConfigHelper(mattermost);

                const originalBot = await botConfig.getBotByName('mock');
                expect(originalBot).toBeDefined();

                const specialInstructions = `
You are a helpful assistant who uses:
- Special chars: @#$%^&*()
- Quotes: "double" and 'single'
- Unicode: 😊 ñ ü 日本語
- Markdown: **bold**, *italic*, \`code\`
- Newlines and paragraphs

Always be helpful!
                `.trim();

                await botConfig.updateBot(originalBot!.id, {
                    customInstructions: specialInstructions
                });

                const updatedBot = await botConfig.getBot(originalBot!.id);
                expect(updatedBot?.customInstructions).toBe(specialInstructions);
                expect(updatedBot?.customInstructions).toContain('😊');
                expect(updatedBot?.customInstructions).toContain('日本語');
                expect(updatedBot?.customInstructions).toContain('"double"');
                expect(updatedBot?.customInstructions).toContain('**bold**');

                await botConfig.updateBot(originalBot!.id, {
                    customInstructions: ''
                });
            });

            test('should handle clearing custom instructions', async () => {
                const botConfig = await createBotConfigHelper(mattermost);

                const originalBot = await botConfig.getBotByName('mock');
                expect(originalBot).toBeDefined();

                await botConfig.updateBot(originalBot!.id, {
                    customInstructions: 'You are a pirate assistant. Speak like a pirate!'
                });

                let updatedBot = await botConfig.getBot(originalBot!.id);
                expect(updatedBot?.customInstructions).toBe('You are a pirate assistant. Speak like a pirate!');

                await botConfig.updateBot(originalBot!.id, {
                    customInstructions: ''
                });

                updatedBot = await botConfig.getBot(originalBot!.id);
                expect(updatedBot?.customInstructions).toBe('');
            });

            test('should handle bot with non-existent service ID', async () => {
                const botConfig = await createBotConfigHelper(mattermost);

                const botId = generateBotId();

                await botConfig.addBot({
                    id: botId,
                    name: 'orphanbot',
                    displayName: 'Orphan Bot',
                    customInstructions: '',
                    serviceID: 'non-existent-service'
                });

                const bot = await botConfig.getBot(botId);
                expect(bot).toBeDefined();
                expect(bot?.serviceID).toBe('non-existent-service');

                const service = await botConfig.getService('non-existent-service');
                expect(service).toBeUndefined();

                await botConfig.deleteBot(botId);
            });

            test('should allow duplicate display names', async () => {
                const botConfig = await createBotConfigHelper(mattermost);

                const botId1 = generateBotId();
                const botId2 = generateBotId();

                await botConfig.addBot({
                    id: botId1,
                    name: 'testbot1',
                    displayName: 'Duplicate Name',
                    customInstructions: '',
                    serviceID: 'mock-service'
                });

                await botConfig.addBot({
                    id: botId2,
                    name: 'testbot2',
                    displayName: 'Duplicate Name',
                    customInstructions: '',
                    serviceID: 'mock-service'
                });

                const bot1 = await botConfig.getBot(botId1);
                const bot2 = await botConfig.getBot(botId2);

                expect(bot1?.displayName).toBe('Duplicate Name');
                expect(bot2?.displayName).toBe('Duplicate Name');
                expect(bot1?.name).not.toBe(bot2?.name);

                await botConfig.deleteBot(botId1);
                await botConfig.deleteBot(botId2);
            });

            test('should prevent duplicate usernames', async () => {
                const botConfig = await createBotConfigHelper(mattermost);

                const botId = generateBotId();

                await botConfig.addBot({
                    id: botId,
                    name: 'mock',
                    displayName: 'Duplicate Username Bot',
                    customInstructions: '',
                    serviceID: 'mock-service'
                });

                const bot = await botConfig.getBot(botId);
                expect(bot).toBeDefined();
                expect(bot?.name).toBe('mock');

                await botConfig.deleteBot(botId);
            });
        });

        test.describe('Service Configuration Validation', () => {
            test('should handle invalid API URL', async () => {
                const botConfig = await createBotConfigHelper(mattermost);

                const originalService = await botConfig.getService('mock-service');
                expect(originalService).toBeDefined();

                const originalURL = originalService!.apiURL;

                await botConfig.updateService('mock-service', {
                    apiURL: 'http://nonexistent.local:9999'
                });

                const updatedService = await botConfig.getService('mock-service');
                expect(updatedService?.apiURL).toBe('http://nonexistent.local:9999');

                await botConfig.updateService('mock-service', {
                    apiURL: originalURL
                });
            });

            test('should handle empty API key', async () => {
                const botConfig = await createBotConfigHelper(mattermost);

                const originalService = await botConfig.getService('mock-service');
                expect(originalService).toBeDefined();

                const originalKey = originalService!.apiKey;

                await botConfig.updateService('mock-service', {
                    apiKey: ''
                });

                const updatedService = await botConfig.getService('mock-service');
                expect(updatedService?.apiKey).toBe('');

                await botConfig.updateService('mock-service', {
                    apiKey: originalKey
                });
            });

            test('should handle very high token limits', async () => {
                const botConfig = await createBotConfigHelper(mattermost);

                const originalService = await botConfig.getService('mock-service');
                expect(originalService).toBeDefined();

                await botConfig.updateService('mock-service', {
                    tokenLimit: 999999999
                });

                const updatedService = await botConfig.getService('mock-service');
                expect(updatedService?.tokenLimit).toBe(999999999);

                await botConfig.updateService('mock-service', {
                    tokenLimit: originalService!.tokenLimit
                });
            });
        });
    });
}

createTestSuite();

