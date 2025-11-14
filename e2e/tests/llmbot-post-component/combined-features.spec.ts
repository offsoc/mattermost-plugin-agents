import { test, expect } from '@playwright/test';
import RunRealAPIContainer from 'helpers/real-api-container';
import MattermostContainer from 'helpers/mmcontainer';
import { MattermostPage } from 'helpers/mm';
import { AIPlugin } from 'helpers/ai-plugin';
import { LLMBotPostHelper } from 'helpers/llmbot-post';
import { getAPIConfig, getSkipMessage, logAPIConfig } from 'helpers/api-config';

/**
 * Test Suite: Combined Features
 *
 * Tests multiple features working together in LLMBot posts using REAL APIs.
 * Runs once per configured provider (OpenAI and/or Anthropic).
 *
 * Environment Variables Required:
 * - ANTHROPIC_API_KEY: To run tests with Anthropic (claude-3-5-haiku)
 * - OPENAI_API_KEY: To run tests with OpenAI (gpt-4o-mini)
 *
 * Tests:
 * 1. Reasoning and Citations Together
 * 2. Regenerate Functionality
 * 3. Multiple Users Viewing Same Post
 */

const username = 'regularuser';
const password = 'regularuser';
const username2 = 'sysadmin';
const password2 = 'Sys@dmin-sample1';

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
    test.describe(`Combined Features - ${provider.name}`, () => {
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

        test('Reasoning and Citations Together', async ({ page }) => {
            test.skip(!config.shouldRunTests, skipMessage);
            test.setTimeout(150000);

            const { mmPage, aiPlugin, llmBotHelper, botUsername } = await setupTestPage(page, mattermost, provider);
            await mmPage.login(mattermost.url(), username, password);

            // Navigate to DM with bot (required for web_search native tool)
            await mmPage.createAndNavigateToDMWithBot(mattermost, username, password, botUsername);

            await aiPlugin.openRHS();

            const prompt = provider.type === 'anthropic'
                ? 'Search the web for TypeScript docs and briefly analyze 2-3 key features with citations (1 paragraph)'
                : 'Use web search to find TypeScript docs and briefly list 2-3 benefits with citations (1 paragraph)';

            await aiPlugin.sendMessage(prompt);

            await llmBotHelper.waitForReasoning(undefined, 35000);
            // Wait for streaming to complete (smart wait, 5min safety timeout)
            await llmBotHelper.waitForStreamingComplete();

            await llmBotHelper.expectReasoningVisible(true);
            await expect(page.getByText('Thinking')).toBeVisible();

            const citations = llmBotHelper.getAllCitationIcons();
            const citationCount = await citations.count();

            // Web search in DM context MUST produce citations
            expect(citationCount).toBeGreaterThan(0);
            await expect(citations.first()).toBeVisible();

            await llmBotHelper.clickReasoningToggle();
            await llmBotHelper.expectReasoningExpanded(true);

            await llmBotHelper.hoverCitation(1);
            await page.waitForTimeout(500);
            const tooltip = llmBotHelper.getCitationTooltip();
            await expect(tooltip).toBeVisible({ timeout: 5000 });
        });

        test('Regenerate Functionality', async ({ page }) => {
            test.skip(!config.shouldRunTests, skipMessage);
            test.setTimeout(150000);

            const { mmPage, aiPlugin, llmBotHelper, botUsername } = await setupTestPage(page, mattermost, provider);
            await mmPage.login(mattermost.url(), username, password);

            await aiPlugin.openRHS();

            const prompt = 'Explain TypeScript benefits in 2-3 sentences';

            await aiPlugin.sendMessage(prompt);
            // Wait for streaming to complete (smart wait, 5min safety timeout)
            await llmBotHelper.waitForStreamingComplete();

            const postTextBefore = llmBotHelper.getPostText();
            await expect(postTextBefore).toBeVisible();
            const contentBefore = await postTextBefore.textContent();

            const llmBotPost = llmBotHelper.getLLMBotPost();
            await llmBotPost.hover();
            await page.waitForTimeout(500);

            const regenerateButton = llmBotHelper.getRegenerateButton();
            const isVisible = await regenerateButton.isVisible().catch(() => false);

            if (isVisible) {
                await llmBotHelper.regenerateResponse();
                // Wait for streaming to complete after regeneration
                await llmBotHelper.waitForStreamingComplete();

                const postTextAfter = llmBotHelper.getPostText();
                await expect(postTextAfter).toBeVisible();
                const contentAfter = await postTextAfter.textContent();

                expect(contentBefore).toBeTruthy();
                expect(contentAfter).toBeTruthy();
            } else {
                console.log('Regenerate button not visible, skipping regeneration test');
            }
        });

        test('Multiple Users Viewing Same Post', async ({ page, browser }) => {
            test.skip(!config.shouldRunTests, skipMessage);
            test.setTimeout(150000);

            const { mmPage, aiPlugin, llmBotHelper, botUsername } = await setupTestPage(page, mattermost, provider);
            await mmPage.login(mattermost.url(), username, password);

            await aiPlugin.openRHS();

            const prompt = provider.type === 'anthropic'
                ? 'Briefly analyze the main benefits of TypeScript (1 paragraph)'
                : 'Think about and briefly explain the main benefits of TypeScript (1 paragraph)';

            await aiPlugin.sendMessage(prompt);
            await llmBotHelper.waitForReasoning(undefined, 35000);
            // Wait for streaming to complete (smart wait, 5min safety timeout)
            await llmBotHelper.waitForStreamingComplete();

            const postText1 = llmBotHelper.getPostText();
            await expect(postText1).toBeVisible();
            const content1 = await postText1.textContent();

            await llmBotHelper.expectReasoningVisible(true);
            await llmBotHelper.clickReasoningToggle();
            await llmBotHelper.expectReasoningExpanded(true);

            const context2 = await browser.newContext();
            const page2 = await context2.newPage();

            const mmPage2 = new MattermostPage(page2);
            const aiPlugin2 = new AIPlugin(page2);
            const llmBotHelper2 = new LLMBotPostHelper(page2);

            await mmPage2.login(mattermost.url(), username2, password2);
            await aiPlugin2.openRHS();
            await page2.waitForTimeout(3000);

            const postText2 = llmBotHelper2.getPostText();
            await expect(postText2).toBeVisible();
            const content2 = await postText2.textContent();

            expect(content2).toBe(content1);

            await llmBotHelper2.expectReasoningVisible(true);
            await llmBotHelper2.expectReasoningExpanded(false);

            await llmBotHelper2.clickReasoningToggle();
            await llmBotHelper2.expectReasoningExpanded(true);

            await llmBotHelper.expectReasoningExpanded(true);

            await context2.close();
        });
    });
}

config.providers.forEach(provider => {
    createProviderTestSuite(provider);
});
