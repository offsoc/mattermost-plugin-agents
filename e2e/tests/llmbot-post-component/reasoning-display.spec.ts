import { test, expect } from '@playwright/test';
import RunRealAPIContainer from 'helpers/real-api-container';
import MattermostContainer from 'helpers/mmcontainer';
import { MattermostPage } from 'helpers/mm';
import { AIPlugin } from 'helpers/ai-plugin';
import { LLMBotPostHelper } from 'helpers/llmbot-post';
import { getAPIConfig, getSkipMessage, logAPIConfig } from 'helpers/api-config';

/**
 * Test Suite: Reasoning Display
 *
 * Tests the reasoning display functionality in LLMBot posts using REAL APIs.
 * Runs once per configured provider (OpenAI and/or Anthropic).
 *
 * Environment Variables Required:
 * - ANTHROPIC_API_KEY: To run tests with Anthropic (claude-3-5-haiku)
 * - OPENAI_API_KEY: To run tests with OpenAI (gpt-4o-mini)
 *
 * Tests:
 * 1. Reasoning Display - Renders from Real API
 * 2. Reasoning Toggle - Expand and Collapse
 * 3. Reasoning Persistence After Refresh (CRITICAL)
 * 4. Reasoning States - Complete State
 * 5. Multiple Posts with Reasoning
 */

const username = 'regularuser';
const password = 'regularuser';

const config = getAPIConfig();
const skipMessage = getSkipMessage();

async function setupTestPage(page, mattermost, provider) {
    const mmPage = new MattermostPage(page);
    const aiPlugin = new AIPlugin(page);
    const llmBotHelper = new LLMBotPostHelper(page);

    // Get bot username based on provider
    const botUsername = provider.type === 'anthropic' ? 'claude' : 'mockbot';

    return { mmPage, aiPlugin, llmBotHelper, botUsername };
}

function createProviderTestSuite(provider) {
    test.describe(`Reasoning Display - ${provider.name}`, () => {
        let mattermost: MattermostContainer;

        test.beforeAll(async () => {
            if (!config.shouldRunTests) return;
            mattermost = await RunRealAPIContainer(provider);
        });

        test.afterAll(async () => {
            if (mattermost) {
                await mattermost.stop();
            }
        });

        test('Reasoning Display - Renders from Real API', async ({ page }) => {
            test.skip(!config.shouldRunTests, skipMessage);
            test.setTimeout(60000);

            const { mmPage, aiPlugin, llmBotHelper, botUsername } = await setupTestPage(page, mattermost, provider);
            await mmPage.login(mattermost.url(), username, password);

            await aiPlugin.openRHS();

            const prompt = provider.type === 'anthropic'
                ? 'What letter is missing from the following sequence: A, C, E, G, I, K, M, O, Q, S, U, W, Y, ?. Think HARD.'
                : 'Think step by step about the benefits of TypeScript compared to JavaScript. Consider multiple angles. Keep response brief (1-2 paragraphs).';

            await aiPlugin.sendMessage(prompt);

            await llmBotHelper.waitForReasoning(undefined, 30000);

            await llmBotHelper.expectReasoningVisible(true);
            await expect(page.getByText('Thinking')).toBeVisible();
            await llmBotHelper.expectReasoningExpanded(false);

            await page.waitForTimeout(5000);
            await llmBotHelper.expectReasoningVisible(true);
        });

        test('Reasoning Toggle - Expand and Collapse', async ({ page }) => {
            test.skip(!config.shouldRunTests, skipMessage);
            test.setTimeout(60000);

            const { mmPage, aiPlugin, llmBotHelper, botUsername } = await setupTestPage(page, mattermost, provider);
            await mmPage.login(mattermost.url(), username, password);

            await aiPlugin.openRHS();

            const prompt = provider.type === 'anthropic'
                ? 'Compare TypeScript and JavaScript for code quality. What are the key differences and trade-offs? (2-3 sentences). Think HARD.'
                : 'Think carefully: compare TypeScript and JavaScript for code quality. What are the key differences? (2-3 sentences)';

            await aiPlugin.sendMessage(prompt);
            await llmBotHelper.waitForReasoning(undefined, 30000);
            await page.waitForTimeout(5000);

            await llmBotHelper.expectReasoningVisible(true);
            await llmBotHelper.expectReasoningExpanded(false);

            await llmBotHelper.clickReasoningToggle();
            await llmBotHelper.expectReasoningExpanded(true);

            await llmBotHelper.clickReasoningToggle();
            await llmBotHelper.expectReasoningExpanded(false);
            await expect(page.getByText('Thinking')).toBeVisible();
        });

        test('Reasoning Persistence After Refresh (CRITICAL)', async ({ page }) => {
            test.skip(!config.shouldRunTests, skipMessage);
            test.setTimeout(60000);

            const { mmPage, aiPlugin, llmBotHelper, botUsername } = await setupTestPage(page, mattermost, provider);
            await mmPage.login(mattermost.url(), username, password);

            await aiPlugin.openRHS();

            const prompt = 'Analyze: What are the main advantages of TypeScript over JavaScript? Consider developer experience, maintainability, and type safety. (1 paragraph)';

            await aiPlugin.sendMessage(prompt);
            await llmBotHelper.waitForReasoning(undefined, 30000);
            await page.waitForTimeout(5000);

            await llmBotHelper.expectReasoningVisible(true);
            await llmBotHelper.expectReasoningExpanded(false);

            await llmBotHelper.clickReasoningToggle();
            await llmBotHelper.expectReasoningExpanded(true);

            await page.reload();
            await aiPlugin.openRHS();
            await page.waitForTimeout(2000);

            // After refresh, RHS shows fresh conversation - must navigate to chat history
            await aiPlugin.openChatHistory();
            await page.waitForTimeout(1000);
            await aiPlugin.clickChatHistoryItem(0); // Select most recent conversation
            await page.waitForTimeout(2000);

            // Verify reasoning persists in loaded conversation
            await llmBotHelper.expectReasoningVisible(true);
            await llmBotHelper.expectReasoningExpanded(false);

            await llmBotHelper.clickReasoningToggle();
            await llmBotHelper.expectReasoningExpanded(true);
        });

        test('Reasoning States - Complete State', async ({ page }) => {
            test.skip(!config.shouldRunTests, skipMessage);
            test.setTimeout(60000);

            const { mmPage, aiPlugin, llmBotHelper, botUsername } = await setupTestPage(page, mattermost, provider);
            await mmPage.login(mattermost.url(), username, password);

            await aiPlugin.openRHS();

            await aiPlugin.sendMessage('Evaluate the benefits of TypeScript from multiple perspectives: developer productivity, code maintainability, and team collaboration. (3-4 sentences)');
            await llmBotHelper.waitForReasoning(undefined, 30000);
            await page.waitForTimeout(3000);

            await expect(page.getByText('Thinking')).toBeVisible();

            await llmBotHelper.clickReasoningToggle();
            await llmBotHelper.expectReasoningExpanded(true);

            await llmBotHelper.clickReasoningToggle();
            await llmBotHelper.expectReasoningExpanded(false);
            await llmBotHelper.expectReasoningVisible(true);
        });

        test('Multiple Posts with Reasoning', async ({ page }) => {
            test.skip(!config.shouldRunTests, skipMessage);
            test.setTimeout(120000);

            const { mmPage, aiPlugin, llmBotHelper, botUsername } = await setupTestPage(page, mattermost, provider);
            await mmPage.login(mattermost.url(), username, password);

            await aiPlugin.openRHS();

            // First message with reasoning
            await aiPlugin.sendMessage('Compare and analyze: TypeScript vs JavaScript for large projects. What are the trade-offs?');
            await llmBotHelper.waitForReasoning(undefined, 30000);
            await page.waitForTimeout(3000);

            // Second message with reasoning - wait for it to complete
            await aiPlugin.sendMessage('Evaluate JavaScript limitations when it comes to type safety and refactoring. How do these impact development?');

            // Wait for second reasoning to appear with longer timeout
            const allReasoningDisplays = page.locator('div:has-text("Thinking")');
            await expect(allReasoningDisplays).toHaveCount(2, { timeout: 45000 });
            await page.waitForTimeout(2000);

            // Verify both reasoning displays are interactive
            const firstReasoning = allReasoningDisplays.first();
            await firstReasoning.click();
            await expect(allReasoningDisplays).toHaveCount(2);

            const secondReasoning = allReasoningDisplays.nth(1);
            await secondReasoning.click();
            await expect(allReasoningDisplays).toHaveCount(2);
        });
    });
}

config.providers.forEach(provider => {
    createProviderTestSuite(provider);
});
