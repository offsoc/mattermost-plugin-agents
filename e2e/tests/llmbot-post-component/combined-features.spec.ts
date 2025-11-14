import { test, expect } from '@playwright/test';
import RunLLMBotTestContainer from 'helpers/llmbot-test-container';
import MattermostContainer from 'helpers/mmcontainer';
import { MattermostPage } from 'helpers/mm';
import { AIPlugin } from 'helpers/ai-plugin';
import { AnthropicMockContainer } from 'helpers/anthropic-mock';
import { LLMBotPostHelper } from 'helpers/llmbot-post';
import { LLMBotPostCreator, Annotation } from 'helpers/llmbot-post-creator';

/**
 * Test Suite: Combined Features
 *
 * Tests integration of multiple LLMBot post features:
 * 18. Reasoning + Citations Together
 * 19. Regenerate Clears All State (SKIPPED - requires working API)
 * 20. Multiple Users View Same Post
 *
 * Spec: /e2e/LLMBOT_POST_COMPONENT_TEST_PLAN.md (Tests 18-20)
 * Uses direct post creation to test combined features
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

test.describe('Combined Features', () => {
    test('Reasoning + Citations Together', async ({ page }) => {
        const { mmPage, llmBotHelper } = await setupTestPage(page);

        const reasoning = "I'll search for TypeScript information and analyze key features...";
        const text = "TypeScript provides static typing for JavaScript. More details available.";
        const annotations: Annotation[] = [
            {
                type: 'url_citation',
                start_index: 56,
                end_index: 56,
                url: 'https://www.typescriptlang.org/',
                title: 'TypeScript Official Site',
                index: 1
            },
            {
                type: 'url_citation',
                start_index: 80,
                end_index: 80,
                url: 'https://github.com/microsoft/TypeScript',
                title: 'TypeScript GitHub',
                index: 2
            }
        ];

        await postCreator.createPost({
            message: text,
            reasoning: reasoning,
            annotations: annotations,
            channelId: testChannelId,
            requesterUserId: testUserId,
        });

        await mmPage.goto('test', 'messages');
        await page.waitForTimeout(1000);

        await llmBotHelper.expectReasoningVisible(true);
        await expect(page.getByText('Thinking')).toBeVisible();

        await llmBotHelper.expectPostText(text);
        await llmBotHelper.expectCitationCount(2);

        await llmBotHelper.clickReasoningToggle();
        await llmBotHelper.expectReasoningExpanded(true);
        await llmBotHelper.expectReasoningText("I'll search for TypeScript");

        await llmBotHelper.hoverCitation(1);
        await llmBotHelper.expectCitationTooltip('typescriptlang.org');

        await page.mouse.move(0, 0);
        await page.waitForTimeout(200);

        await llmBotHelper.hoverCitation(2);
        await llmBotHelper.expectCitationTooltip('github.com');

        await page.reload();
        await mmPage.goto('test', 'messages');
        await page.waitForTimeout(1000);

        await llmBotHelper.expectReasoningVisible(true);
        await llmBotHelper.expectReasoningExpanded(false);

        await llmBotHelper.expectCitationCount(2);
        await llmBotHelper.waitForCitation(1);

        await llmBotHelper.clickReasoningToggle();
        await llmBotHelper.expectReasoningExpanded(true);
        await llmBotHelper.expectReasoningText('TypeScript information');

        await llmBotHelper.hoverCitation(1);
        await llmBotHelper.expectCitationTooltip('typescriptlang.org');
    });

    test.skip('Regenerate Clears All State', async ({ page }) => {
        // SKIPPED: requires working regenerate API
    });

    test('Multiple Users View Same Post', async ({ page, browser }) => {
        const { mmPage, llmBotHelper } = await setupTestPage(page);

        const reasoning = "Analyzing the question from multiple perspectives...";
        const text = "Based on comprehensive research, here are the key findings.";
        const annotations: Annotation[] = [
            {
                type: 'url_citation',
                start_index: 40,
                end_index: 40,
                url: 'https://www.example.com/research',
                title: 'Research Article',
                index: 1
            },
            {
                type: 'url_citation',
                start_index: 75,
                end_index: 75,
                url: 'https://www.example.com/findings',
                title: 'Key Findings',
                index: 2
            }
        ];

        await postCreator.createPost({
            message: text,
            reasoning: reasoning,
            annotations: annotations,
            channelId: testChannelId,
            requesterUserId: testUserId,
        });

        await mmPage.goto('test', 'messages');
        await page.waitForTimeout(1000);

        await llmBotHelper.expectReasoningVisible(true);
        await llmBotHelper.clickReasoningToggle();
        await llmBotHelper.expectReasoningExpanded(true);
        await llmBotHelper.expectReasoningText('Analyzing the question');

        const context2 = await browser.newContext();
        const page2 = await context2.newPage();

        const mmPage2 = new MattermostPage(page2);
        const aiPlugin2 = new AIPlugin(page2);
        const llmBotHelper2 = new LLMBotPostHelper(page2);
        const url = mattermost.url();

        await mmPage2.login(url, 'seconduser', 'seconduser');

        const secondUserClient = await mattermost.getClient('seconduser', 'seconduser');
        const secondUser = await secondUserClient.getMe();
        const secondUserChannelId = await postCreator.createDMChannel(secondUser.id);

        await postCreator.createPost({
            message: text,
            reasoning: reasoning,
            annotations: annotations,
            channelId: secondUserChannelId,
            requesterUserId: secondUser.id,
        });

        await mmPage2.goto('test', 'messages');
        await page2.waitForTimeout(1000);

        await llmBotHelper2.expectReasoningVisible(true);
        await llmBotHelper2.expectReasoningExpanded(false);

        await llmBotHelper2.clickReasoningToggle();
        await llmBotHelper2.expectReasoningExpanded(true);
        await llmBotHelper2.expectReasoningText('multiple perspectives');

        await llmBotHelper.hoverCitation(1);
        await llmBotHelper.expectCitationTooltip('example.com');

        await page.mouse.move(0, 0);
        await page.waitForTimeout(200);

        await llmBotHelper2.hoverCitation(1);
        await llmBotHelper2.expectCitationTooltip('example.com');

        await llmBotHelper.expectPostText(text);
        await llmBotHelper2.expectPostText(text);

        await llmBotHelper.expectCitationCount(2);
        await llmBotHelper2.expectCitationCount(2);

        const popup1Promise = page.waitForEvent('popup');
        await llmBotHelper.clickCitation(1);
        const popup1 = await popup1Promise;
        expect(popup1.url()).toBe(annotations[0].url);
        await popup1.close();

        const popup2Promise = page2.waitForEvent('popup');
        await llmBotHelper2.clickCitation(2);
        const popup2 = await popup2Promise;
        expect(popup2.url()).toBe(annotations[1].url);
        await popup2.close();

        await page2.close();
        await context2.close();
    });
});
