import { test, expect } from '@playwright/test';
import RunRealAPIContainer from 'helpers/real-api-container';
import MattermostContainer from 'helpers/mmcontainer';
import { MattermostPage } from 'helpers/mm';
import { AIPlugin } from 'helpers/ai-plugin';
import { LLMBotPostHelper } from 'helpers/llmbot-post';
import { getAPIConfig, getSkipMessage, logAPIConfig } from 'helpers/api-config';

/**
 * Test Suite: Citations and Annotations
 *
 * Tests the citation/annotation display functionality in LLMBot posts using REAL APIs.
 * Runs once per configured provider (OpenAI and/or Anthropic).
 *
 * Environment Variables Required:
 * - ANTHROPIC_API_KEY: To run tests with Anthropic (claude-3-5-haiku)
 * - OPENAI_API_KEY: To run tests with OpenAI (gpt-4o-mini)
 *
 * Tests:
 * 1. Citation Display - Renders from Real API
 * 2. Citation Hover Tooltip
 * 3. Citation Click Link
 * 4. Multiple Citations
 * 5. Citation Persistence After Refresh
 * 6. Citations with Markdown Content
 * 7. Citation Favicon Display
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
    test.describe(`Citations and Annotations - ${provider.name}`, () => {
        let mattermost: MattermostContainer;

        test.beforeAll(async () => {
            if (!config.shouldRunTests) return;
            provider.reasoningEnabled = false;
            mattermost = await RunRealAPIContainer(provider);
        });

        test.afterAll(async () => {
            if (mattermost) {
                await mattermost.stop();
            }
        });

        test('Citation Display - Renders from Real API', async ({ page }) => {
            test.skip(!config.shouldRunTests, skipMessage);
            test.setTimeout(90000);

            const { mmPage, aiPlugin, llmBotHelper, botUsername } = await setupTestPage(page, mattermost, provider);
            await mmPage.login(mattermost.url(), username, password);

            // Navigate to DM with bot (required for web_search native tool)
            await mmPage.createAndNavigateToDMWithBot(mattermost, username, password, botUsername);

            await aiPlugin.openRHS();

            const prompt = provider.type === 'anthropic'
                ? 'Search the web for TypeScript documentation and briefly summarize 2-3 key features'
                : 'Use web search to find TypeScript best practices and briefly list 2-3 points with citations';

            await aiPlugin.sendMessage(prompt);

            // Wait for streaming to complete (returns early when done, 5min safety timeout)
            await llmBotHelper.waitForStreamingComplete();

            const citations = llmBotHelper.getAllCitationIcons();
            const count = await citations.count();

            // Web search in DM context MUST produce citations
            expect(count).toBeGreaterThan(0);
            await llmBotHelper.expectCitationCount(count);
            await expect(citations.first()).toBeVisible();
        });

        test('Citation Hover Tooltip', async ({ page }) => {
            test.skip(!config.shouldRunTests, skipMessage);
            test.setTimeout(120000);

            const { mmPage, aiPlugin, llmBotHelper, botUsername } = await setupTestPage(page, mattermost, provider);
            await mmPage.login(mattermost.url(), username, password);

            // Navigate to DM with bot (required for web_search native tool)
            await mmPage.createAndNavigateToDMWithBot(mattermost, username, password, botUsername);

            await aiPlugin.openRHS();

            const prompt = 'Search the web for TypeScript documentation and briefly summarize with citations (2-3 sentences)';

            await aiPlugin.sendMessage(prompt);

            // Wait for streaming to complete (returns early when done, 5min safety timeout)
            await llmBotHelper.waitForStreamingComplete();

            const citations = llmBotHelper.getAllCitationIcons();
            const count = await citations.count();

            // Web search in DM context MUST produce citations
            expect(count).toBeGreaterThan(0);
            await llmBotHelper.hoverCitation(1);
            await page.waitForTimeout(1500); // Longer wait for tooltip

            const tooltip = llmBotHelper.getCitationTooltip();
            await expect(tooltip).toBeVisible({ timeout: 15000 }); // Increased timeout
        });

        test('Citation Click Link', async ({ page, context }) => {
            test.skip(!config.shouldRunTests, skipMessage);
            test.setTimeout(120000);

            const { mmPage, aiPlugin, llmBotHelper, botUsername } = await setupTestPage(page, mattermost, provider);
            await mmPage.login(mattermost.url(), username, password);

            // Navigate to DM with bot (required for web_search native tool)
            await mmPage.createAndNavigateToDMWithBot(mattermost, username, password, botUsername);

            await aiPlugin.openRHS();

            const prompt = 'Search the web for TypeScript official website and cite it';

            await aiPlugin.sendMessage(prompt);

            // Wait for streaming to complete (returns early when done, 5min safety timeout)
            await llmBotHelper.waitForStreamingComplete();

            const citations = llmBotHelper.getAllCitationIcons();
            const count = await citations.count();

            // Web search in DM context MUST produce citations
            expect(count).toBeGreaterThan(0);
            const pagePromise = context.waitForEvent('page');
            await llmBotHelper.clickCitation(1);

            const newPage = await pagePromise;
            await newPage.waitForLoadState();
            await expect(newPage.url()).toContain('http');
            await newPage.close();
        });

        test('Multiple Citations', async ({ page }) => {
            test.skip(!config.shouldRunTests, skipMessage);
            test.setTimeout(150000);

            const { mmPage, aiPlugin, llmBotHelper, botUsername } = await setupTestPage(page, mattermost, provider);
            await mmPage.login(mattermost.url(), username, password);

            // Navigate to DM with bot (required for web_search native tool)
            await mmPage.createAndNavigateToDMWithBot(mattermost, username, password, botUsername);

            await aiPlugin.openRHS();

            const prompt = provider.type === 'anthropic'
                ? 'Search the web for TypeScript, JavaScript, and React and briefly compare them with citations (1 paragraph)'
                : 'Use web search to find TypeScript, JavaScript, React info and briefly compare with citations (1 paragraph)';

            await aiPlugin.sendMessage(prompt);

            // Wait for streaming to complete (smart wait, 5min safety timeout)
            await llmBotHelper.waitForStreamingComplete();

            const citations = llmBotHelper.getAllCitationIcons();
            const count = await citations.count();

            // Web search with multiple topics should produce multiple citations
            expect(count).toBeGreaterThanOrEqual(2);
            await expect(citations.first()).toBeVisible();
            await expect(citations.nth(1)).toBeVisible();

            await llmBotHelper.hoverCitation(1);
            await page.waitForTimeout(500);
            const tooltip1 = llmBotHelper.getCitationTooltip();
            await expect(tooltip1).toBeVisible({ timeout: 5000 });

            await page.mouse.move(0, 0);
            await page.waitForTimeout(300);

            await llmBotHelper.hoverCitation(2);
            await page.waitForTimeout(500);
            const tooltip2 = llmBotHelper.getCitationTooltip();
            await expect(tooltip2).toBeVisible({ timeout: 5000 });
        });

        test('Citation Persistence After Refresh', async ({ page }) => {
            test.skip(!config.shouldRunTests, skipMessage);
            test.setTimeout(120000);

            const { mmPage, aiPlugin, llmBotHelper, botUsername } = await setupTestPage(page, mattermost, provider);
            await mmPage.login(mattermost.url(), username, password);

            // Navigate to DM with bot (required for web_search native tool)
            await mmPage.createAndNavigateToDMWithBot(mattermost, username, password, botUsername);

            await aiPlugin.openRHS();

            const prompt = 'Search the web for TypeScript documentation and briefly describe it with citations (1 paragraph)';

            await aiPlugin.sendMessage(prompt);

            // Wait for streaming to complete (smart wait, 5min safety timeout)
            await llmBotHelper.waitForStreamingComplete();

            const citationsBefore = llmBotHelper.getAllCitationIcons();
            const countBefore = await citationsBefore.count();

            // Web search in DM context MUST produce citations
            expect(countBefore).toBeGreaterThan(0);

            await page.reload();
            await aiPlugin.openRHS();
            await page.waitForTimeout(2000);

            // After refresh, RHS shows fresh conversation - must navigate to chat history
            await aiPlugin.openChatHistory();
            await page.waitForTimeout(1000);
            await aiPlugin.clickChatHistoryItem(0); // Select most recent conversation
            await page.waitForTimeout(2000);

            // Verify citations persist in loaded conversation
            const citationsAfter = llmBotHelper.getAllCitationIcons();
            const countAfter = await citationsAfter.count();

            expect(countAfter).toBe(countBefore);
            await expect(citationsAfter.first()).toBeVisible();
        });

        test('Citations with Markdown Content', async ({ page }) => {
            test.skip(!config.shouldRunTests, skipMessage);
            test.setTimeout(120000);

            const { mmPage, aiPlugin, llmBotHelper, botUsername } = await setupTestPage(page, mattermost, provider);
            await mmPage.login(mattermost.url(), username, password);

            // Navigate to DM with bot (required for web_search native tool)
            await mmPage.createAndNavigateToDMWithBot(mattermost, username, password, botUsername);

            await aiPlugin.openRHS();

            const prompt = 'Search the web for 1-2 TypeScript code examples with markdown formatting and citations (brief)';

            await aiPlugin.sendMessage(prompt);

            // Wait for streaming to complete (smart wait, 5min safety timeout)
            await llmBotHelper.waitForStreamingComplete();

            const postText = llmBotHelper.getPostText();
            await expect(postText).toBeVisible();

            const citations = llmBotHelper.getAllCitationIcons();
            const count = await citations.count();

            // Web search in DM context MUST produce citations
            expect(count).toBeGreaterThan(0);
            await expect(citations.first()).toBeVisible();
        });

        test('Citation Favicon Display', async ({ page }) => {
            test.skip(!config.shouldRunTests, skipMessage);
            test.setTimeout(120000);

            const { mmPage, aiPlugin, llmBotHelper, botUsername } = await setupTestPage(page, mattermost, provider);
            await mmPage.login(mattermost.url(), username, password);

            // Navigate to DM with bot (required for web_search native tool)
            await mmPage.createAndNavigateToDMWithBot(mattermost, username, password, botUsername);

            await aiPlugin.openRHS();

            const prompt = 'Search the web for TypeScript official documentation and cite it';

            await aiPlugin.sendMessage(prompt);

            // Wait for streaming to complete (returns early when done, 5min safety timeout)
            await llmBotHelper.waitForStreamingComplete();

            const citations = llmBotHelper.getAllCitationIcons();
            const count = await citations.count();

            // Web search in DM context MUST produce citations
            expect(count).toBeGreaterThan(0);
            await llmBotHelper.hoverCitation(1);
            await page.waitForTimeout(1500); // Longer wait for tooltip

            const tooltip = llmBotHelper.getCitationTooltip();
            await expect(tooltip).toBeVisible({ timeout: 15000 }); // Increased timeout

            // Favicon display is optional - some sites may not have favicons
            const favicon = tooltip.locator('img[src*="favicon"], svg');
            const faviconCount = await favicon.count();

            if (faviconCount > 0) {
                await expect(favicon.first()).toBeVisible();
            }
        });
    });
}

config.providers.forEach(provider => {
    createProviderTestSuite(provider);
});
