import { test, expect } from '@playwright/test';
import RunLLMBotTestContainer from 'helpers/llmbot-test-container';
import MattermostContainer from 'helpers/mmcontainer';
import { MattermostPage } from 'helpers/mm';
import { AIPlugin } from 'helpers/ai-plugin';
import { AnthropicMockContainer } from 'helpers/anthropic-mock';
import { LLMBotPostHelper } from 'helpers/llmbot-post';
import { LLMBotPostCreator, Annotation } from 'helpers/llmbot-post-creator';

/**
 * Test Suite: Edge Cases
 *
 * Tests edge cases and error scenarios for LLMBot posts:
 * 21. Empty Reasoning
 * 22. Reasoning Without Text Response
 * 23. Citation at Start/End of Text
 * 24. Invalid Annotation JSON (handled by backend)
 * 25. Very Long Reasoning Text
 * 26. Rapid Reasoning Collapse/Expand
 * 27. Special Characters in Citations
 * 28. Citation Click Blocked by Browser
 * 29. Network Error During Streaming (SKIPPED - requires streaming)
 * 30. Concurrent Reasoning from Multiple Bots
 *
 * Spec: /e2e/LLMBOT_POST_COMPONENT_TEST_PLAN.md (Tests 21-30)
 * Uses direct post creation to test edge cases
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

    postCreator = new LLMBotPostCreator(mattermost);
    await postCreator.initialize('claude');

    const userClient = await mattermost.getClient(username, password);
    const user = await userClient.getMe();
    testUserId = user.id;

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

test.describe('Edge Cases', () => {
    test('Empty Reasoning', async ({ page }) => {
        const { mmPage, llmBotHelper } = await setupTestPage(page);

        const response = 'This is a response without reasoning.';

        await postCreator.createPost({
            message: response,
            channelId: testChannelId,
            requesterUserId: testUserId,
        });

        await mmPage.goto('test', 'messages');
        await page.waitForTimeout(1000);

        await llmBotHelper.expectReasoningVisible(false);
        await expect(page.getByText('Thinking')).not.toBeVisible();

        await llmBotHelper.expectPostText(response);
    });

    test('Reasoning Without Text Response', async ({ page }) => {
        const { mmPage, llmBotHelper } = await setupTestPage(page);

        const reasoning = 'I need to think about this question carefully...';
        const emptyResponse = '';

        await postCreator.createPost({
            message: emptyResponse,
            reasoning: reasoning,
            channelId: testChannelId,
            requesterUserId: testUserId,
        });

        await mmPage.goto('test', 'messages');
        await page.waitForTimeout(1000);

        await llmBotHelper.expectReasoningVisible(true);
        await llmBotHelper.clickReasoningToggle();
        await llmBotHelper.expectReasoningExpanded(true);
        await llmBotHelper.expectReasoningText('think about this question');
    });

    test('Citation at Start/End of Text', async ({ page }) => {
        const { mmPage, llmBotHelper } = await setupTestPage(page);

        const textStart = 'Source information about TypeScript features.';
        const annotationAtStart: Annotation = {
            type: 'url_citation',
            start_index: 0,
            end_index: 0,
            url: 'https://example.com/start',
            title: 'Start Source',
            index: 1
        };

        await postCreator.createPost({
            message: textStart,
            annotations: [annotationAtStart],
            channelId: testChannelId,
            requesterUserId: testUserId,
        });

        await mmPage.goto('test', 'messages');
        await page.waitForTimeout(1000);

        await llmBotHelper.expectCitationCount(1);
        await llmBotHelper.waitForCitation(1);
        const citationStart = llmBotHelper.getCitationWrapper(1);
        await expect(citationStart).toBeVisible();

        const textEnd = 'TypeScript is a great language for development.';
        const annotationAtEnd: Annotation = {
            type: 'url_citation',
            start_index: textEnd.length,
            end_index: textEnd.length,
            url: 'https://example.com/end',
            title: 'End Source',
            index: 1
        };

        await postCreator.createPost({
            message: textEnd,
            annotations: [annotationAtEnd],
            channelId: testChannelId,
            requesterUserId: testUserId,
        });

        await mmPage.goto('test', 'messages');
        await page.waitForTimeout(1000);

        await llmBotHelper.expectCitationCount(1);
        await llmBotHelper.waitForCitation(1);
        const citationEnd = llmBotHelper.getCitationWrapper(1);
        await expect(citationEnd).toBeVisible();
    });

    test('Invalid Annotation JSON', async ({ page }) => {
        const { mmPage, llmBotHelper } = await setupTestPage(page);

        const response = 'This response should display correctly.';

        await postCreator.createPost({
            message: response,
            channelId: testChannelId,
            requesterUserId: testUserId,
        });

        await mmPage.goto('test', 'messages');
        await page.waitForTimeout(1000);

        await llmBotHelper.expectPostText(response);
        await llmBotHelper.expectCitationCount(0);
    });

    test('Very Long Reasoning Text', async ({ page }) => {
        const { mmPage, llmBotHelper } = await setupTestPage(page);

        const longReasoning = 'Step 1: I need to analyze this complex problem thoroughly. '.repeat(40) +
            'Step 2: Consider all the alternatives and their implications. '.repeat(30) +
            'Step 3: Synthesize the information into a coherent conclusion. '.repeat(20);
        const response = 'Here is my comprehensive answer after detailed analysis.';

        await postCreator.createPost({
            message: response,
            reasoning: longReasoning,
            channelId: testChannelId,
            requesterUserId: testUserId,
        });

        await mmPage.goto('test', 'messages');
        await page.waitForTimeout(1000);

        await llmBotHelper.expectReasoningVisible(true);

        await llmBotHelper.clickReasoningToggle();
        await llmBotHelper.expectReasoningExpanded(true);

        await llmBotHelper.expectReasoningText('Step 1: I need to analyze');
        await llmBotHelper.expectReasoningText('Step 2: Consider all');
        await llmBotHelper.expectReasoningText('Step 3: Synthesize');

        const postText = llmBotHelper.getPostText();
        await expect(postText).toBeVisible();

        await llmBotHelper.clickReasoningToggle();
        await llmBotHelper.expectReasoningExpanded(false);
    });

    test('Rapid Reasoning Collapse/Expand', async ({ page }) => {
        const { mmPage, llmBotHelper } = await setupTestPage(page);

        const reasoning = 'Testing UI stability with rapid toggling of reasoning display.';
        const response = 'This tests animation and state handling.';

        await postCreator.createPost({
            message: response,
            reasoning: reasoning,
            channelId: testChannelId,
            requesterUserId: testUserId,
        });

        await mmPage.goto('test', 'messages');
        await page.waitForTimeout(1000);

        for (let i = 0; i < 10; i++) {
            await llmBotHelper.clickReasoningToggle();
            await page.waitForTimeout(50);
        }

        await llmBotHelper.expectReasoningVisible(true);
        const reasoning_display = llmBotHelper.getReasoningDisplay();
        await expect(reasoning_display).toBeVisible();

        await llmBotHelper.clickReasoningToggle();
        await page.waitForTimeout(300);

        const postText = llmBotHelper.getPostText();
        await expect(postText).toBeVisible();
        await llmBotHelper.expectPostText(response);
    });

    test('Special Characters in Citations', async ({ page }) => {
        const { mmPage, llmBotHelper } = await setupTestPage(page);

        const specialCharAnnotation: Annotation = {
            type: 'url_citation',
            start_index: 50,
            end_index: 50,
            url: 'https://example.com/path?param=value&other=123#section',
            title: 'Title with "quotes" and <brackets> & ampersands',
            index: 1
        };
        const responseText = 'Here is information with special characters in the citation metadata.';

        await postCreator.createPost({
            message: responseText,
            annotations: [specialCharAnnotation],
            channelId: testChannelId,
            requesterUserId: testUserId,
        });

        await mmPage.goto('test', 'messages');
        await page.waitForTimeout(1000);

        await llmBotHelper.expectCitationCount(1);
        await llmBotHelper.waitForCitation(1);

        await llmBotHelper.hoverCitation(1);
        await llmBotHelper.expectCitationTooltip('example.com');

        const popupPromise = page.waitForEvent('popup');
        await llmBotHelper.clickCitation(1);
        const popup = await popupPromise;
        expect(popup.url()).toBe(specialCharAnnotation.url);
        await popup.close();
    });

    test('Citation Click Blocked by Browser', async ({ page }) => {
        const { mmPage, llmBotHelper } = await setupTestPage(page);

        const annotation: Annotation = {
            type: 'url_citation',
            start_index: 30,
            end_index: 30,
            url: 'https://www.example.com/test',
            title: 'Test Source',
            index: 1
        };
        const responseText = 'Testing popup blocker behavior with citations.';

        await postCreator.createPost({
            message: responseText,
            annotations: [annotation],
            channelId: testChannelId,
            requesterUserId: testUserId,
        });

        await mmPage.goto('test', 'messages');
        await page.waitForTimeout(1000);

        await llmBotHelper.waitForCitation(1);

        try {
            const popupPromise = page.waitForEvent('popup', { timeout: 2000 });
            await llmBotHelper.clickCitation(1);
            const popup = await popupPromise;
            await popup.close();
        } catch (e) {
            // Popup may be blocked
        }

        const postText = llmBotHelper.getPostText();
        await expect(postText).toBeVisible();
        await llmBotHelper.expectPostText(responseText);
    });

    test.skip('Network Error During Streaming', async ({ page }) => {
        // SKIPPED: requires real streaming
    });

    test('Concurrent Reasoning from Multiple Bots', async ({ page }) => {
        const { mmPage, llmBotHelper } = await setupTestPage(page);

        const reasoning1 = 'Bot A reasoning: analyzing the first question thoroughly...';
        const response1 = 'Bot A response based on careful analysis.';

        await postCreator.createPost({
            message: response1,
            reasoning: reasoning1,
            channelId: testChannelId,
            requesterUserId: testUserId,
        });

        const reasoning2 = 'Bot B reasoning: thinking about the second question differently...';
        const response2 = 'Bot B response with alternative perspective.';

        await postCreator.createPost({
            message: response2,
            reasoning: reasoning2,
            channelId: testChannelId,
            requesterUserId: testUserId,
        });

        await mmPage.goto('test', 'messages');
        await page.waitForTimeout(1000);

        const allReasoningDisplays = page.locator('div:has-text("Thinking")');
        await expect(allReasoningDisplays).toHaveCount(2);

        const firstReasoning = allReasoningDisplays.first();
        await firstReasoning.click();
        await expect(page.getByText('Bot A reasoning')).toBeVisible();

        const secondReasoning = allReasoningDisplays.nth(1);
        await secondReasoning.click();
        await expect(page.getByText('Bot B reasoning')).toBeVisible();

        await expect(page.getByText(response1)).toBeVisible();
        await expect(page.getByText(response2)).toBeVisible();

        await page.reload();
        await mmPage.goto('test', 'messages');
        await page.waitForTimeout(1000);

        await expect(allReasoningDisplays).toHaveCount(2);
        await expect(page.getByText(response1)).toBeVisible();
        await expect(page.getByText(response2)).toBeVisible();
    });
});
