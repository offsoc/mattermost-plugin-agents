import { test, expect } from '@playwright/test';
import RunLLMBotTestContainer from 'helpers/llmbot-test-container';
import MattermostContainer from 'helpers/mmcontainer';
import { MattermostPage } from 'helpers/mm';
import { AIPlugin } from 'helpers/ai-plugin';
import { AnthropicMockContainer } from 'helpers/anthropic-mock';
import { LLMBotPostHelper } from 'helpers/llmbot-post';
import { LLMBotPostCreator, Annotation } from 'helpers/llmbot-post-creator';

/**
 * Test Suite: Streaming and Persistence
 *
 * Tests content persistence in LLMBot posts (streaming tests skipped):
 * 13. Text Streaming - Cursor Animation (SKIPPED - requires real streaming)
 * 14. Streaming States - Start to End (SKIPPED - requires real streaming)
 * 15. Streaming Stop/Cancel (SKIPPED - requires real streaming)
 * 16. Persistence After Page Navigation
 * 17. Persistence in Thread View
 *
 * Spec: /e2e/LLMBOT_POST_COMPONENT_TEST_PLAN.md (Tests 13-17)
 * Uses direct post creation to test persistence after page navigation
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

test.describe('Streaming and Persistence', () => {
    test.skip('Text Streaming - Cursor Animation', async ({ page }) => {
        // SKIPPED: requires real streaming
    });

    test.skip('Streaming States - Start to End', async ({ page }) => {
        // SKIPPED: requires real streaming
    });

    test.skip('Streaming Stop/Cancel', async ({ page }) => {
        // SKIPPED: requires real streaming
    });

    test('Persistence After Page Navigation', async ({ page }) => {
        const { mmPage, llmBotHelper } = await setupTestPage(page);

        const reasoning = 'Let me analyze this question carefully and provide a comprehensive answer...';
        const annotations: Annotation[] = [
            {
                type: 'url_citation',
                start_index: 50,
                end_index: 50,
                url: 'https://www.example.com/source',
                title: 'Example Source',
                index: 1
            }
        ];
        const response = 'TypeScript is a powerful language with many benefits including static typing and excellent tooling support.';

        await postCreator.createPost({
            message: response,
            reasoning: reasoning,
            annotations: annotations,
            channelId: testChannelId,
            requesterUserId: testUserId,
        });

        await mmPage.goto('test', 'messages');
        await page.waitForTimeout(1000);

        await llmBotHelper.expectPostText(response);
        await llmBotHelper.expectReasoningVisible(true);
        await llmBotHelper.expectCitationCount(1);

        await page.goto(mattermost.url() + '/test/channels/town-square');
        await page.waitForTimeout(1000);

        await mmPage.goto('test', 'messages');
        await page.waitForTimeout(1000);

        await llmBotHelper.expectPostText(response);

        await llmBotHelper.expectReasoningVisible(true);
        await llmBotHelper.expectReasoningExpanded(false);

        await llmBotHelper.clickReasoningToggle();
        await llmBotHelper.expectReasoningExpanded(true);
        await llmBotHelper.expectReasoningText('analyze this question');

        await llmBotHelper.expectCitationCount(1);
        await llmBotHelper.waitForCitation(1);

        await llmBotHelper.hoverCitation(1);
        await llmBotHelper.expectCitationTooltip('example.com');

        const popupPromise = page.waitForEvent('popup');
        await llmBotHelper.clickCitation(1);
        const popup = await popupPromise;
        expect(popup.url()).toBe(annotations[0].url);
        await popup.close();
    });

    test('Persistence in Thread View', async ({ page }) => {
        const { mmPage, llmBotHelper } = await setupTestPage(page);

        const reasoning = 'Analyzing the request and gathering relevant information to provide a helpful response...';
        const annotations: Annotation[] = [
            {
                type: 'url_citation',
                start_index: 70,
                end_index: 70,
                url: 'https://www.typescriptlang.org/',
                title: 'TypeScript Official',
                index: 1
            }
        ];
        const response = 'TypeScript provides excellent type safety and developer experience with modern JavaScript features.';

        await postCreator.createPost({
            message: response,
            reasoning: reasoning,
            annotations: annotations,
            channelId: testChannelId,
            requesterUserId: testUserId,
        });

        await mmPage.goto('test', 'messages');
        await page.waitForTimeout(1000);

        await llmBotHelper.expectReasoningVisible(true);
        await llmBotHelper.expectCitationCount(1);

        await page.keyboard.press('Escape');
        await page.waitForTimeout(500);

        await mmPage.goto('test', 'messages');
        await page.waitForTimeout(1000);

        await llmBotHelper.expectPostText(response);
        await llmBotHelper.expectReasoningVisible(true);

        await llmBotHelper.expectReasoningExpanded(false);
        await llmBotHelper.clickReasoningToggle();
        await llmBotHelper.expectReasoningExpanded(true);
        await llmBotHelper.expectReasoningText('Analyzing the request');

        await llmBotHelper.expectCitationCount(1);
        await llmBotHelper.hoverCitation(1);
        await llmBotHelper.expectCitationTooltip('typescriptlang.org');

        await page.reload();
        await page.waitForTimeout(1000);
        await mmPage.goto('test', 'messages');
        await page.waitForTimeout(1000);

        await llmBotHelper.expectPostText(response);
        await llmBotHelper.expectReasoningVisible(true);
        await llmBotHelper.expectCitationCount(1);

        await llmBotHelper.clickReasoningToggle();
        await llmBotHelper.expectReasoningExpanded(true);

        await llmBotHelper.hoverCitation(1);
        await llmBotHelper.expectCitationTooltip('typescriptlang.org');
    });
});
