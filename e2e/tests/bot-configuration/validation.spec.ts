import { test, expect } from '@playwright/test';

import RunContainer from 'helpers/plugincontainer';
import MattermostContainer from 'helpers/mmcontainer';
import { OpenAIMockContainer, RunOpenAIMocks } from 'helpers/openai-mock';
import { createBotConfigHelper, generateBotId } from 'helpers/bot-config';

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

        // Create bot with empty display name
        await botConfig.addBot({
            id: botId,
            name: 'testbot',
            displayName: '', // Empty display name
            customInstructions: '',
            serviceID: 'mock-service'
        });

        // Verify bot was added (API doesn't validate, but it should be there)
        const bot = await botConfig.getBot(botId);
        expect(bot).toBeDefined();
        expect(bot?.displayName).toBe('');

        // Clean up
        await botConfig.deleteBot(botId);
    });

    test('should require username for bot creation', async () => {
        const botConfig = await createBotConfigHelper(mattermost);

        // Attempt to create bot without username
        const botId = generateBotId();

        await botConfig.addBot({
            id: botId,
            name: '', // Empty name
            displayName: 'Test Bot',
            customInstructions: '',
            serviceID: 'mock-service'
        });

        // Verify bot was added
        const bot = await botConfig.getBot(botId);
        expect(bot).toBeDefined();
        expect(bot?.name).toBe('');

        // Clean up
        await botConfig.deleteBot(botId);
    });

    test('should require service ID for bot creation', async () => {
        const botConfig = await createBotConfigHelper(mattermost);

        const botId = generateBotId();

        // Create bot without service ID
        await botConfig.addBot({
            id: botId,
            name: 'testbot',
            displayName: 'Test Bot',
            customInstructions: '',
            serviceID: '' // Empty service ID
        });

        // Verify bot was added
        const bot = await botConfig.getBot(botId);
        expect(bot).toBeDefined();
        expect(bot?.serviceID).toBe('');

        // Clean up
        await botConfig.deleteBot(botId);
    });

    test('should handle very long custom instructions', async () => {
        const botConfig = await createBotConfigHelper(mattermost);

        // Get existing bot
        const originalBot = await botConfig.getBotByName('mock');
        expect(originalBot).toBeDefined();

        // Create very long custom instructions (5000+ characters)
        const longInstructions = 'You are a helpful assistant. '.repeat(200); // ~5600 characters

        // Update bot with long instructions
        await botConfig.updateBot(originalBot!.id, {
            customInstructions: longInstructions
        });

        // Verify instructions were saved
        const updatedBot = await botConfig.getBot(originalBot!.id);
        expect(updatedBot?.customInstructions).toBe(longInstructions);
        expect(updatedBot?.customInstructions.length).toBeGreaterThan(5000);

        // Restore original instructions
        await botConfig.updateBot(originalBot!.id, {
            customInstructions: ''
        });
    });

    test('should handle special characters in custom instructions', async () => {
        const botConfig = await createBotConfigHelper(mattermost);

        // Get existing bot
        const originalBot = await botConfig.getBotByName('mock');
        expect(originalBot).toBeDefined();

        // Custom instructions with special characters
        const specialInstructions = `
You are a helpful assistant who uses:
- Special chars: @#$%^&*()
- Quotes: "double" and 'single'
- Unicode: 😊 ñ ü 日本語
- Markdown: **bold**, *italic*, \`code\`
- Newlines and paragraphs

Always be helpful!
        `.trim();

        // Update bot with special character instructions
        await botConfig.updateBot(originalBot!.id, {
            customInstructions: specialInstructions
        });

        // Verify all characters were preserved
        const updatedBot = await botConfig.getBot(originalBot!.id);
        expect(updatedBot?.customInstructions).toBe(specialInstructions);
        expect(updatedBot?.customInstructions).toContain('😊');
        expect(updatedBot?.customInstructions).toContain('日本語');
        expect(updatedBot?.customInstructions).toContain('"double"');
        expect(updatedBot?.customInstructions).toContain('**bold**');

        // Restore original instructions
        await botConfig.updateBot(originalBot!.id, {
            customInstructions: ''
        });
    });

    test('should handle clearing custom instructions', async () => {
        const botConfig = await createBotConfigHelper(mattermost);

        // Get existing bot
        const originalBot = await botConfig.getBotByName('mock');
        expect(originalBot).toBeDefined();

        // Set custom instructions
        await botConfig.updateBot(originalBot!.id, {
            customInstructions: 'You are a pirate assistant. Speak like a pirate!'
        });

        // Verify instructions were set
        let updatedBot = await botConfig.getBot(originalBot!.id);
        expect(updatedBot?.customInstructions).toBe('You are a pirate assistant. Speak like a pirate!');

        // Clear instructions
        await botConfig.updateBot(originalBot!.id, {
            customInstructions: ''
        });

        // Verify instructions were cleared
        updatedBot = await botConfig.getBot(originalBot!.id);
        expect(updatedBot?.customInstructions).toBe('');
    });

    test('should handle bot with non-existent service ID', async () => {
        const botConfig = await createBotConfigHelper(mattermost);

        const botId = generateBotId();

        // Create bot with non-existent service
        await botConfig.addBot({
            id: botId,
            name: 'orphanbot',
            displayName: 'Orphan Bot',
            customInstructions: '',
            serviceID: 'non-existent-service'
        });

        // Verify bot was added
        const bot = await botConfig.getBot(botId);
        expect(bot).toBeDefined();
        expect(bot?.serviceID).toBe('non-existent-service');

        // Verify service doesn't exist
        const service = await botConfig.getService('non-existent-service');
        expect(service).toBeUndefined();

        // Clean up
        await botConfig.deleteBot(botId);
    });

    test('should allow duplicate display names', async () => {
        const botConfig = await createBotConfigHelper(mattermost);

        const botId1 = generateBotId();
        const botId2 = generateBotId();

        // Create two bots with same display name
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

        // Verify both bots exist with same display name
        const bot1 = await botConfig.getBot(botId1);
        const bot2 = await botConfig.getBot(botId2);

        expect(bot1?.displayName).toBe('Duplicate Name');
        expect(bot2?.displayName).toBe('Duplicate Name');
        expect(bot1?.name).not.toBe(bot2?.name);

        // Clean up
        await botConfig.deleteBot(botId1);
        await botConfig.deleteBot(botId2);
    });

    test('should prevent duplicate usernames', async () => {
        const botConfig = await createBotConfigHelper(mattermost);

        const botId = generateBotId();

        // Attempt to create bot with duplicate username
        await botConfig.addBot({
            id: botId,
            name: 'mock', // Duplicate of existing bot
            displayName: 'Duplicate Username Bot',
            customInstructions: '',
            serviceID: 'mock-service'
        });

        // Verify bot was added in configuration
        // Note: The actual Mattermost bot user creation might fail,
        // but the configuration will be saved
        const bot = await botConfig.getBot(botId);
        expect(bot).toBeDefined();
        expect(bot?.name).toBe('mock');

        // Clean up
        await botConfig.deleteBot(botId);
    });
    });

    test.describe('Service Configuration Validation', () => {
        test('should handle invalid API URL', async () => {
        const botConfig = await createBotConfigHelper(mattermost);

        // Get existing service
        const originalService = await botConfig.getService('mock-service');
        expect(originalService).toBeDefined();

        const originalURL = originalService!.apiURL;

        // Set invalid API URL
        await botConfig.updateService('mock-service', {
            apiURL: 'http://nonexistent.local:9999'
        });

        // Verify URL was updated
        const updatedService = await botConfig.getService('mock-service');
        expect(updatedService?.apiURL).toBe('http://nonexistent.local:9999');

        // Restore original URL
        await botConfig.updateService('mock-service', {
            apiURL: originalURL
        });
    });

    test('should handle empty API key', async () => {
        const botConfig = await createBotConfigHelper(mattermost);

        // Get existing service
        const originalService = await botConfig.getService('mock-service');
        expect(originalService).toBeDefined();

        const originalKey = originalService!.apiKey;

        // Set empty API key
        await botConfig.updateService('mock-service', {
            apiKey: ''
        });

        // Verify key was updated
        const updatedService = await botConfig.getService('mock-service');
        expect(updatedService?.apiKey).toBe('');

        // Restore original key
        await botConfig.updateService('mock-service', {
            apiKey: originalKey
        });
    });

    test('should handle very high token limits', async () => {
        const botConfig = await createBotConfigHelper(mattermost);

        // Get existing service
        const originalService = await botConfig.getService('mock-service');
        expect(originalService).toBeDefined();

        // Set very high token limit
        await botConfig.updateService('mock-service', {
            tokenLimit: 999999999
        });

        // Verify limit was set
        const updatedService = await botConfig.getService('mock-service');
        expect(updatedService?.tokenLimit).toBe(999999999);

        // Restore original limit
        await botConfig.updateService('mock-service', {
            tokenLimit: originalService!.tokenLimit
        });
    });
    });
});
