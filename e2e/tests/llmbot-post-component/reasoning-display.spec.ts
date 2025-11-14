import { test, expect } from '@playwright/test';
import RunLLMBotTestContainer from 'helpers/llmbot-test-container';
import MattermostContainer from 'helpers/mmcontainer';
import { MattermostPage } from 'helpers/mm';
import { AIPlugin } from 'helpers/ai-plugin';
import { AnthropicMockContainer } from 'helpers/anthropic-mock';
import { LLMBotPostHelper } from 'helpers/llmbot-post';
import { LLMBotPostCreator } from 'helpers/llmbot-post-creator';

/**
 * Test Suite: Reasoning Display
 *
 * Tests the reasoning display functionality in LLMBot posts:
 * 1. Reasoning Display - Initial Stream
 * 2. Reasoning Toggle - Expand and Collapse
 * 3. Reasoning Persistence After Refresh (CRITICAL)
 * 4. Reasoning States - Loading, Complete, Error
 * 5. Multiple Posts with Reasoning
 *
 * Spec: /e2e/LLMBOT_POST_COMPONENT_TEST_PLAN.md
 * Uses direct post creation to test frontend components and persistence
 */

const username = 'regularuser';
const password = 'regularuser';

let mattermost: MattermostContainer;
let anthropicMock: AnthropicMockContainer;
let postCreator: LLMBotPostCreator;
let testUserId: string;
let testChannelId: string;

test.beforeAll(async () => {
    const containers = await RunLLMBotTestContainer();
    mattermost = containers.mattermost;
    anthropicMock = containers.anthropicMock;

    // Initialize post creator
    postCreator = new LLMBotPostCreator(mattermost);
    await postCreator.initialize('claude');

    // Get test user ID
    const userClient = await mattermost.getClient(username, password);
    const user = await userClient.getMe();
    testUserId = user.id;

    // Create DM channel for tests
    testChannelId = await postCreator.createDMChannel(testUserId);
});

test.afterAll(async () => {
    await anthropicMock.stop();
    await mattermost.stop();
});

async function setupTestPage(page) {
    const mmPage = new MattermostPage(page);
    const aiPlugin = new AIPlugin(page);
    const llmBotHelper = new LLMBotPostHelper(page);
    const url = mattermost.url();

    await mmPage.login(url, username, password);

    return { mmPage, aiPlugin, llmBotHelper };
}

test.describe('Reasoning Display', () => {
    test('Reasoning Display - Renders from Props', async ({ page }) => {
        const { mmPage, llmBotHelper } = await setupTestPage(page);

        // Create post with reasoning via props
        const thinking = "First, I need to understand the context. Then I'll analyze the key points...";
        const response = "Based on my analysis, here are the findings...";

        const post = await postCreator.createPost({
            message: response,
            reasoning: thinking,
            reasoningSignature: 'mock_sig_123',
            channelId: testChannelId,
            requesterUserId: testUserId,
        });

        // Navigate to the AI RHS to see the post
        await mmPage.goto('test', 'messages');
        await page.waitForTimeout(1000);

        // Verify reasoning displays
        await llmBotHelper.expectReasoningVisible(true);
        await expect(page.getByText('Thinking')).toBeVisible();
        await llmBotHelper.expectReasoningExpanded(false);

        // Verify post text is visible
        await llmBotHelper.expectPostText(response);
    });

    test('Reasoning Toggle - Expand and Collapse', async ({ page }) => {
        const { mmPage, llmBotHelper } = await setupTestPage(page);

        const thinking = "Step 1: Analyze the requirements...\n\nStep 2: Consider alternatives...\n\nStep 3: Draw conclusions...";
        const response = "Here is the solution based on my reasoning.";

        await postCreator.createPost({
            message: response,
            reasoning: thinking,
            channelId: testChannelId,
            requesterUserId: testUserId,
        });

        await mmPage.goto('test', 'messages');
        await page.waitForTimeout(1000);

        // Verify reasoning is initially collapsed
        await llmBotHelper.expectReasoningVisible(true);
        await llmBotHelper.expectReasoningExpanded(false);

        // Click to expand
        await llmBotHelper.clickReasoningToggle();
        await llmBotHelper.expectReasoningExpanded(true);
        await llmBotHelper.expectReasoningText('Step 1: Analyze the requirements');

        // Click to collapse
        await llmBotHelper.clickReasoningToggle();
        await llmBotHelper.expectReasoningExpanded(false);
        await expect(page.getByText('Thinking')).toBeVisible();
    });

    test('Reasoning Persistence After Refresh (CRITICAL)', async ({ page }) => {
        const { mmPage, llmBotHelper } = await setupTestPage(page);

        const thinking = "Let me think through this problem step by step...";
        const response = "Based on my analysis, here is the answer.";

        await postCreator.createPost({
            message: response,
            reasoning: thinking,
            channelId: testChannelId,
            requesterUserId: testUserId,
        });

        await mmPage.goto('test', 'messages');
        await page.waitForTimeout(1000);

        // Verify reasoning is visible
        await llmBotHelper.expectReasoningVisible(true);
        await llmBotHelper.expectReasoningExpanded(false);

        // Expand to see full content
        await llmBotHelper.clickReasoningToggle();
        await llmBotHelper.expectReasoningExpanded(true);
        await llmBotHelper.expectReasoningText('Let me think through this problem');

        // CRITICAL: Refresh page and verify reasoning persists
        await page.reload();
        await mmPage.goto('test', 'messages');
        await page.waitForTimeout(1000);

        // Reasoning should still be visible after refresh
        await llmBotHelper.expectReasoningVisible(true);
        await llmBotHelper.expectReasoningExpanded(false);

        // Expand again and verify content is preserved
        await llmBotHelper.clickReasoningToggle();
        await llmBotHelper.expectReasoningExpanded(true);
        await llmBotHelper.expectReasoningText('Let me think through this problem');
        await llmBotHelper.expectReasoningText('step by step');
    });

    test('Reasoning States - Complete State', async ({ page }) => {
        const { mmPage, llmBotHelper } = await setupTestPage(page);

        const thinking = "Analyzing the question...";
        const response = "Here is the complete answer.";

        await postCreator.createPost({
            message: response,
            reasoning: thinking,
            channelId: testChannelId,
            requesterUserId: testUserId,
        });

        await mmPage.goto('test', 'messages');
        await page.waitForTimeout(1000);

        // Verify thinking label is present
        await expect(page.getByText('Thinking')).toBeVisible();

        // Expand and verify content
        await llmBotHelper.clickReasoningToggle();
        await llmBotHelper.expectReasoningExpanded(true);
        await llmBotHelper.expectReasoningText('Analyzing the question');

        // Collapse and verify still accessible
        await llmBotHelper.clickReasoningToggle();
        await llmBotHelper.expectReasoningExpanded(false);
        await llmBotHelper.expectReasoningVisible(true);
    });

    test('Multiple Posts with Reasoning', async ({ page }) => {
        const { mmPage, llmBotHelper } = await setupTestPage(page);

        // Create first post with reasoning
        const thinking1 = "First reasoning: analyzing the first question...";
        const response1 = "First response based on reasoning.";

        await postCreator.createPost({
            message: response1,
            reasoning: thinking1,
            channelId: testChannelId,
            requesterUserId: testUserId,
        });

        // Create second post with reasoning
        const thinking2 = "Second reasoning: thinking about the second question...";
        const response2 = "Second response with different reasoning.";

        await postCreator.createPost({
            message: response2,
            reasoning: thinking2,
            channelId: testChannelId,
            requesterUserId: testUserId,
        });

        await mmPage.goto('test', 'messages');
        await page.waitForTimeout(1000);

        // Verify both posts have independent reasoning displays
        const allReasoningDisplays = page.locator('div:has-text("Thinking")');
        await expect(allReasoningDisplays).toHaveCount(2);

        // Expand first post's reasoning
        const firstReasoning = allReasoningDisplays.first();
        await firstReasoning.click();
        await expect(allReasoningDisplays).toHaveCount(2);

        // Expand second post's reasoning
        const secondReasoning = allReasoningDisplays.nth(1);
        await secondReasoning.click();
        await expect(allReasoningDisplays).toHaveCount(2);

        // Verify no cross-contamination - both should still be visible
        await expect(page.getByText(response1)).toBeVisible();
        await expect(page.getByText(response2)).toBeVisible();
    });
});
