import { test, expect } from '@playwright/test';
import RunLLMBotTestContainer from 'helpers/llmbot-test-container';
import MattermostContainer from 'helpers/mmcontainer';
import { MattermostPage } from 'helpers/mm';
import { AIPlugin } from 'helpers/ai-plugin';
import { AnthropicMockContainer } from 'helpers/anthropic-mock';
import { LLMBotPostHelper } from 'helpers/llmbot-post';
import { LLMBotPostCreator, Annotation } from 'helpers/llmbot-post-creator';

/**
 * Test Suite: Citations and Annotations
 *
 * Tests the citation/annotation display functionality in LLMBot posts:
 * 6. Citation Display - Single Citation
 * 7. Citation Hover - Tooltip Display
 * 8. Citation Click - Opens Link
 * 9. Multiple Citations in Response
 * 10. Citations Persistence After Refresh (CRITICAL)
 * 11. Citations with Markdown Content
 * 12. Citation Favicon Fallback
 *
 * Spec: /e2e/LLMBOT_POST_COMPONENT_TEST_PLAN.md (Tests 6-12)
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

test.describe('Citations and Annotations', () => {
    test('Citation Display - Single Citation', async ({ page }) => {
        const { mmPage, llmBotHelper } = await setupTestPage(page);

        const annotation: Annotation = {
            type: 'url_citation',
            start_index: 50,
            end_index: 50,
            url: 'https://www.typescriptlang.org/docs/',
            title: 'TypeScript Documentation',
            index: 1
        };
        const responseText = 'TypeScript is a typed superset of JavaScript. More info...';

        await postCreator.createPost({
            message: responseText,
            annotations: [annotation],
            channelId: testChannelId,
            requesterUserId: testUserId,
        });

        await mmPage.goto('test', 'messages');
        await page.waitForTimeout(1000);

        await llmBotHelper.expectCitationCount(1);
        await llmBotHelper.waitForCitation(1);

        const citationWrapper = llmBotHelper.getCitationWrapper(1);
        await expect(citationWrapper).toBeVisible();

        await llmBotHelper.expectPostText(responseText);
    });

    test('Citation Hover - Tooltip Display', async ({ page }) => {
        const { mmPage, llmBotHelper } = await setupTestPage(page);

        const annotation: Annotation = {
            type: 'url_citation',
            start_index: 50,
            end_index: 50,
            url: 'https://www.typescriptlang.org/docs/',
            title: 'TypeScript Documentation',
            index: 1
        };
        const responseText = 'TypeScript provides excellent type safety and tooling support.';

        await postCreator.createPost({
            message: responseText,
            annotations: [annotation],
            channelId: testChannelId,
            requesterUserId: testUserId,
        });

        await mmPage.goto('test', 'messages');
        await page.waitForTimeout(1000);

        await llmBotHelper.waitForCitation(1);

        await llmBotHelper.hoverCitation(1);

        const expectedDomain = 'typescriptlang.org';
        await llmBotHelper.expectCitationTooltip(expectedDomain);

        await page.mouse.move(0, 0);
        await page.waitForTimeout(500);
        const tooltip = llmBotHelper.getCitationTooltip();
        await expect(tooltip).not.toBeVisible();
    });

    test('Citation Click - Opens Link', async ({ page }) => {
        const { mmPage, llmBotHelper } = await setupTestPage(page);

        const annotation: Annotation = {
            type: 'url_citation',
            start_index: 45,
            end_index: 45,
            url: 'https://www.typescriptlang.org/docs/',
            title: 'TypeScript Documentation',
            index: 1
        };
        const responseText = 'TypeScript provides excellent type safety.';

        await postCreator.createPost({
            message: responseText,
            annotations: [annotation],
            channelId: testChannelId,
            requesterUserId: testUserId,
        });

        await mmPage.goto('test', 'messages');
        await page.waitForTimeout(1000);

        await llmBotHelper.waitForCitation(1);

        const popupPromise = page.waitForEvent('popup');
        await llmBotHelper.clickCitation(1);

        const popup = await popupPromise;
        expect(popup.url()).toBe(annotation.url);
        await popup.close();
    });

    test('Multiple Citations in Response', async ({ page }) => {
        const { mmPage, llmBotHelper } = await setupTestPage(page);

        const annotations: Annotation[] = [
            {
                type: 'url_citation',
                start_index: 45,
                end_index: 45,
                url: 'https://example.com/source1',
                title: 'First Source',
                index: 1
            },
            {
                type: 'url_citation',
                start_index: 120,
                end_index: 120,
                url: 'https://example.com/source2',
                title: 'Second Source',
                index: 2
            },
            {
                type: 'url_citation',
                start_index: 200,
                end_index: 200,
                url: 'https://example.com/source3',
                title: 'Third Source',
                index: 3
            }
        ];
        const responseText = 'TypeScript provides static typing which helps catch errors early. It has excellent IDE support and tooling. The community is large and active with many resources available.';

        await postCreator.createPost({
            message: responseText,
            annotations: annotations,
            channelId: testChannelId,
            requesterUserId: testUserId,
        });

        await mmPage.goto('test', 'messages');
        await page.waitForTimeout(1000);

        await llmBotHelper.expectCitationCount(3);

        await llmBotHelper.hoverCitation(1);
        await llmBotHelper.expectCitationTooltip('example.com');

        await page.mouse.move(0, 0);
        await page.waitForTimeout(200);

        await llmBotHelper.hoverCitation(2);
        await llmBotHelper.expectCitationTooltip('example.com');

        await page.mouse.move(0, 0);
        await page.waitForTimeout(200);

        await llmBotHelper.hoverCitation(3);
        await llmBotHelper.expectCitationTooltip('example.com');

        const citation1 = llmBotHelper.getCitationWrapper(1);
        const citation2 = llmBotHelper.getCitationWrapper(2);
        const citation3 = llmBotHelper.getCitationWrapper(3);

        await expect(citation1).toBeVisible();
        await expect(citation2).toBeVisible();
        await expect(citation3).toBeVisible();
    });

    test('Citations Persistence After Refresh (CRITICAL)', async ({ page }) => {
        const { mmPage, llmBotHelper } = await setupTestPage(page);

        const annotations: Annotation[] = [
            {
                type: 'url_citation',
                start_index: 50,
                end_index: 50,
                url: 'https://www.typescriptlang.org/',
                title: 'TypeScript Official',
                index: 1
            },
            {
                type: 'url_citation',
                start_index: 100,
                end_index: 100,
                url: 'https://github.com/microsoft/TypeScript',
                title: 'TypeScript GitHub',
                index: 2
            }
        ];
        const responseText = 'TypeScript is a powerful language with great tooling and community support available online.';

        await postCreator.createPost({
            message: responseText,
            annotations: annotations,
            channelId: testChannelId,
            requesterUserId: testUserId,
        });

        await mmPage.goto('test', 'messages');
        await page.waitForTimeout(1000);

        await llmBotHelper.expectCitationCount(2);

        await llmBotHelper.hoverCitation(1);
        await llmBotHelper.expectCitationTooltip('typescriptlang.org');

        await page.reload();
        await mmPage.goto('test', 'messages');
        await page.waitForTimeout(1000);

        await llmBotHelper.expectCitationCount(2);
        await llmBotHelper.waitForCitation(1);

        await llmBotHelper.hoverCitation(1);
        await llmBotHelper.expectCitationTooltip('typescriptlang.org');

        const popupPromise = page.waitForEvent('popup');
        await llmBotHelper.clickCitation(1);
        const popup = await popupPromise;
        expect(popup.url()).toBe(annotations[0].url);
        await popup.close();
    });

    test('Citations with Markdown Content', async ({ page }) => {
        const { mmPage, llmBotHelper } = await setupTestPage(page);

        const annotations: Annotation[] = [
            {
                type: 'url_citation',
                start_index: 30,
                end_index: 30,
                url: 'https://www.typescriptlang.org/',
                title: 'TypeScript Docs',
                index: 1
            },
            {
                type: 'url_citation',
                start_index: 85,
                end_index: 85,
                url: 'https://example.com/guide',
                title: 'TypeScript Guide',
                index: 2
            }
        ];
        const responseWithMarkdown = '**TypeScript** is a great language. Learn more about it.\n\n- Strong typing\n- Great tooling\n- Active community\n\nExample: `const x: number = 1;`';

        await postCreator.createPost({
            message: responseWithMarkdown,
            annotations: annotations,
            channelId: testChannelId,
            requesterUserId: testUserId,
        });

        await mmPage.goto('test', 'messages');
        await page.waitForTimeout(1000);

        const postText = llmBotHelper.getPostText();
        await expect(postText).toContainText('TypeScript');
        await expect(postText).toContainText('Strong typing');

        await llmBotHelper.expectCitationCount(2);

        await llmBotHelper.hoverCitation(1);
        await llmBotHelper.expectCitationTooltip('typescriptlang.org');

        await page.mouse.move(0, 0);
        await page.waitForTimeout(200);

        await llmBotHelper.hoverCitation(2);
        await llmBotHelper.expectCitationTooltip('example.com');
    });

    test('Citation Favicon Fallback', async ({ page }) => {
        const { mmPage, llmBotHelper } = await setupTestPage(page);

        const annotation: Annotation = {
            type: 'url_citation',
            start_index: 40,
            end_index: 40,
            url: 'https://nonexistent-domain-12345.com/path',
            title: 'Test Source',
            index: 1
        };
        const responseText = 'Here is some information with a citation that may not have a favicon.';

        await postCreator.createPost({
            message: responseText,
            annotations: [annotation],
            channelId: testChannelId,
            requesterUserId: testUserId,
        });

        await mmPage.goto('test', 'messages');
        await page.waitForTimeout(1000);

        await llmBotHelper.waitForCitation(1);

        await llmBotHelper.hoverCitation(1);

        const tooltip = llmBotHelper.getCitationTooltip();
        await expect(tooltip).toBeVisible();

        await llmBotHelper.expectCitationTooltip('nonexistent-domain-12345.com');

        const tooltipText = await tooltip.textContent();
        expect(tooltipText).toContain('nonexistent-domain-12345.com');
    });
});
