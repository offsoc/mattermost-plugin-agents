import { test, expect } from '@playwright/test';

import RunContainer from 'helpers/plugincontainer';
import MattermostContainer from 'helpers/mmcontainer';
import { MattermostPage } from 'helpers/mm';
import { AIPlugin } from 'helpers/ai-plugin';
import { OpenAIMockContainer, RunOpenAIMocks } from 'helpers/openai-mock';

// spec: /Users/nickmisasi/workspace/worktrees/mattermost-plugin-ai-agents-in-e2e/e2e/specs/action-item-extraction.md
// seed: /Users/nickmisasi/workspace/worktrees/mattermost-plugin-ai-agents-in-e2e/seed.spec.ts

// Test configuration
const username = 'regularuser';
const password = 'regularuser';

let mattermost: MattermostContainer;
let openAIMock: OpenAIMockContainer;

// Mock response for action items
const actionItemsResponse = `
data: {"id":"chatcmpl-ai-1","object":"chat.completion.chunk","created":1708124577,"model":"gpt-3.5-turbo-0613","system_fingerprint":null,"choices":[{"index":0,"delta":{"role":"assistant","content":""},"logprobs":null,"finish_reason":null}]}
data: {"id":"chatcmpl-ai-1","object":"chat.completion.chunk","created":1708124577,"model":"gpt-3.5-turbo-0613","system_fingerprint":null,"choices":[{"index":0,"delta":{"content":"Based"},"logprobs":null,"finish_reason":null}]}
data: {"id":"chatcmpl-ai-1","object":"chat.completion.chunk","created":1708124577,"model":"gpt-3.5-turbo-0613","system_fingerprint":null,"choices":[{"index":0,"delta":{"content":" on"},"logprobs":null,"finish_reason":null}]}
data: {"id":"chatcmpl-ai-1","object":"chat.completion.chunk","created":1708124577,"model":"gpt-3.5-turbo-0613","system_fingerprint":null,"choices":[{"index":0,"delta":{"content":" the"},"logprobs":null,"finish_reason":null}]}
data: {"id":"chatcmpl-ai-1","object":"chat.completion.chunk","created":1708124577,"model":"gpt-3.5-turbo-0613","system_fingerprint":null,"choices":[{"index":0,"delta":{"content":" conversation"},"logprobs":null,"finish_reason":null}]}
data: {"id":"chatcmpl-ai-1","object":"chat.completion.chunk","created":1708124577,"model":"gpt-3.5-turbo-0613","system_fingerprint":null,"choices":[{"index":0,"delta":{"content":","},"logprobs":null,"finish_reason":null}]}
data: {"id":"chatcmpl-ai-1","object":"chat.completion.chunk","created":1708124577,"model":"gpt-3.5-turbo-0613","system_fingerprint":null,"choices":[{"index":0,"delta":{"content":" here"},"logprobs":null,"finish_reason":null}]}
data: {"id":"chatcmpl-ai-1","object":"chat.completion.chunk","created":1708124577,"model":"gpt-3.5-turbo-0613","system_fingerprint":null,"choices":[{"index":0,"delta":{"content":" are"},"logprobs":null,"finish_reason":null}]}
data: {"id":"chatcmpl-ai-1","object":"chat.completion.chunk","created":1708124577,"model":"gpt-3.5-turbo-0613","system_fingerprint":null,"choices":[{"index":0,"delta":{"content":" the"},"logprobs":null,"finish_reason":null}]}
data: {"id":"chatcmpl-ai-1","object":"chat.completion.chunk","created":1708124577,"model":"gpt-3.5-turbo-0613","system_fingerprint":null,"choices":[{"index":0,"delta":{"content":" action"},"logprobs":null,"finish_reason":null}]}
data: {"id":"chatcmpl-ai-1","object":"chat.completion.chunk","created":1708124577,"model":"gpt-3.5-turbo-0613","system_fingerprint":null,"choices":[{"index":0,"delta":{"content":" items"},"logprobs":null,"finish_reason":null}]}
data: {"id":"chatcmpl-ai-1","object":"chat.completion.chunk","created":1708124577,"model":"gpt-3.5-turbo-0613","system_fingerprint":null,"choices":[{"index":0,"delta":{"content":":"},"logprobs":null,"finish_reason":null}]}
data: {"id":"chatcmpl-ai-1","object":"chat.completion.chunk","created":1708124577,"model":"gpt-3.5-turbo-0613","system_fingerprint":null,"choices":[{"index":0,"delta":{"content":" 1"},"logprobs":null,"finish_reason":null}]}
data: {"id":"chatcmpl-ai-1","object":"chat.completion.chunk","created":1708124577,"model":"gpt-3.5-turbo-0613","system_fingerprint":null,"choices":[{"index":0,"delta":{"content":"."},"logprobs":null,"finish_reason":null}]}
data: {"id":"chatcmpl-ai-1","object":"chat.completion.chunk","created":1708124577,"model":"gpt-3.5-turbo-0613","system_fingerprint":null,"choices":[{"index":0,"delta":{"content":" John"},"logprobs":null,"finish_reason":null}]}
data: {"id":"chatcmpl-ai-1","object":"chat.completion.chunk","created":1708124577,"model":"gpt-3.5-turbo-0613","system_fingerprint":null,"choices":[{"index":0,"delta":{"content":" to"},"logprobs":null,"finish_reason":null}]}
data: {"id":"chatcmpl-ai-1","object":"chat.completion.chunk","created":1708124577,"model":"gpt-3.5-turbo-0613","system_fingerprint":null,"choices":[{"index":0,"delta":{"content":" update"},"logprobs":null,"finish_reason":null}]}
data: {"id":"chatcmpl-ai-1","object":"chat.completion.chunk","created":1708124577,"model":"gpt-3.5-turbo-0613","system_fingerprint":null,"choices":[{"index":0,"delta":{"content":" project"},"logprobs":null,"finish_reason":null}]}
data: {"id":"chatcmpl-ai-1","object":"chat.completion.chunk","created":1708124577,"model":"gpt-3.5-turbo-0613","system_fingerprint":null,"choices":[{"index":0,"delta":{"content":" roadmap"},"logprobs":null,"finish_reason":null}]}
data: {"id":"chatcmpl-ai-1","object":"chat.completion.chunk","created":1708124577,"model":"gpt-3.5-turbo-0613","system_fingerprint":null,"choices":[{"index":0,"delta":{"content":" by"},"logprobs":null,"finish_reason":null}]}
data: {"id":"chatcmpl-ai-1","object":"chat.completion.chunk","created":1708124577,"model":"gpt-3.5-turbo-0613","system_fingerprint":null,"choices":[{"index":0,"delta":{"content":" Friday"},"logprobs":null,"finish_reason":null}]}
data: {"id":"chatcmpl-ai-1","object":"chat.completion.chunk","created":1708124577,"model":"gpt-3.5-turbo-0613","system_fingerprint":null,"choices":[{"index":0,"delta":{},"logprobs":null,"finish_reason":"stop"}]}
data: [DONE]
`.trim().split('\n').filter(l => l).join('\n\n') + '\n\n';

const actionItemsResponseText = "Based on the conversation, here are the action items: 1. John to update project roadmap by Friday";

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

// Common test setup
async function setupTestPage(page) {
    const mmPage = new MattermostPage(page);
    const aiPlugin = new AIPlugin(page);
    const url = mattermost.url();

    await mmPage.login(url, username, password);

    return { mmPage, aiPlugin };
}

test.describe('Basic Action Item Extraction', () => {
    test('Extract Action Items from Simple Thread with Clear Tasks', async ({ page }) => {
        const { mmPage, aiPlugin } = await setupTestPage(page);

        // 1. Navigate to Mattermost at the base URL (already done in setupTestPage)

        // 2. Send a channel message: "We need to discuss the project timeline"
        const rootPost = await mmPage.sendMessageAsUser(
            mattermost,
            username,
            password,
            'We need to discuss the project timeline'
        );

        // 3. Create thread replies using the API
        const userClient = await mattermost.getClient(username, password);

        // Reply 1: "John, can you update the project roadmap by Friday?"
        await userClient.createPost({
            channel_id: rootPost.channel_id,
            root_id: rootPost.id,
            message: 'John, can you update the project roadmap by Friday?'
        });

        // Reply 2: "Sarah needs to schedule the stakeholder meeting next week"
        await userClient.createPost({
            channel_id: rootPost.channel_id,
            root_id: rootPost.id,
            message: 'Sarah needs to schedule the stakeholder meeting next week'
        });

        // Reply 3: "I'll send the design mockups to the team by EOD"
        await userClient.createPost({
            channel_id: rootPost.channel_id,
            root_id: rootPost.id,
            message: "I'll send the design mockups to the team by EOD"
        });

        // Navigate to the post
        await page.goto(mattermost.url() + '/test/channels/town-square');

        // Wait for the post to be visible
        await page.locator(`#post_${rootPost.id}`).waitFor({ state: 'visible' });

        // 4. Hover over the root post to reveal the post menu
        await page.locator(`#post_${rootPost.id}`).hover();

        // 5. Click on the AI Actions menu (test ID: `ai-actions-menu`)
        await page.getByTestId(`ai-actions-menu`).click();

        // 6. Click on "Find action items" button
        await openAIMock.addCompletionMock(actionItemsResponse);
        await page.getByRole('button', { name: 'Find action items' }).click();

        // 7. Wait for the AI RHS to open with the DM conversation
        await aiPlugin.expectRHSOpenWithPost();

        // Expected Results: Action items are displayed in the response
        await expect(page.getByText('action items')).toBeVisible();
    });

    test('Extract Action Items from Thread with Mixed Content', async ({ page }) => {
        const { mmPage, aiPlugin } = await setupTestPage(page);

        // 1. Send a channel message: "Team standup discussion"
        const rootPost = await mmPage.sendMessageAsUser(
            mattermost,
            username,
            password,
            'Team standup discussion'
        );

        // 2. Create thread replies
        const userClient = await mattermost.getClient(username, password);

        // Reply 1: "Good morning everyone! How's everyone doing today?"
        await userClient.createPost({
            channel_id: rootPost.channel_id,
            root_id: rootPost.id,
            message: "Good morning everyone! How's everyone doing today?"
        });

        // Reply 2: "Alex will investigate the performance issue in the database"
        await userClient.createPost({
            channel_id: rootPost.channel_id,
            root_id: rootPost.id,
            message: 'Alex will investigate the performance issue in the database'
        });

        // Reply 3: "The weather is really nice today"
        await userClient.createPost({
            channel_id: rootPost.channel_id,
            root_id: rootPost.id,
            message: 'The weather is really nice today'
        });

        // Reply 4: "Maria, please review the PR #123 before the end of the sprint"
        await userClient.createPost({
            channel_id: rootPost.channel_id,
            root_id: rootPost.id,
            message: 'Maria, please review the PR #123 before the end of the sprint'
        });

        // Reply 5: "Thanks for the update!"
        await userClient.createPost({
            channel_id: rootPost.channel_id,
            root_id: rootPost.id,
            message: 'Thanks for the update!'
        });

        // 3. Navigate to the post
        await page.goto(mattermost.url() + '/test/channels/town-square');
        await page.locator(`#post_${rootPost.id}`).waitFor({ state: 'visible' });

        // 4. Hover over the root post
        await page.locator(`#post_${rootPost.id}`).hover();

        // 5. Open AI Actions menu
        await page.getByTestId(`ai-actions-menu`).click();

        // 6. Click "Find action items"
        await openAIMock.addCompletionMock(actionItemsResponse);
        await page.getByRole('button', { name: 'Find action items' }).click();

        // 7. Wait for RHS to display results
        await aiPlugin.expectRHSOpenWithPost();

        // Expected Results: Action items extracted (Alex and Maria tasks)
        await expect(page.getByText('action items')).toBeVisible();
    });

    test('Extract Action Items with Various Formats', async ({ page }) => {
        const { mmPage, aiPlugin } = await setupTestPage(page);

        // 1. Send a channel message: "Sprint planning outcomes"
        const rootPost = await mmPage.sendMessageAsUser(
            mattermost,
            username,
            password,
            'Sprint planning outcomes'
        );

        // 2. Create thread replies with different action item formats
        const userClient = await mattermost.getClient(username, password);

        // Reply 1: "TODO: Update documentation for the new API endpoints"
        await userClient.createPost({
            channel_id: rootPost.channel_id,
            root_id: rootPost.id,
            message: 'TODO: Update documentation for the new API endpoints'
        });

        // Reply 2: "Action item: Team lead needs to approve the budget request"
        await userClient.createPost({
            channel_id: rootPost.channel_id,
            root_id: rootPost.id,
            message: 'Action item: Team lead needs to approve the budget request'
        });

        // Reply 3: "We should migrate the legacy code to the new framework"
        await userClient.createPost({
            channel_id: rootPost.channel_id,
            root_id: rootPost.id,
            message: 'We should migrate the legacy code to the new framework'
        });

        // Reply 4: "Must complete: Security audit before deployment"
        await userClient.createPost({
            channel_id: rootPost.channel_id,
            root_id: rootPost.id,
            message: 'Must complete: Security audit before deployment'
        });

        // Reply 5: "Let's set up automated testing"
        await userClient.createPost({
            channel_id: rootPost.channel_id,
            root_id: rootPost.id,
            message: "Let's set up automated testing"
        });

        // 3. Navigate to the post
        await page.goto(mattermost.url() + '/test/channels/town-square');
        await page.locator(`#post_${rootPost.id}`).waitFor({ state: 'visible' });

        // 4. Hover over the root post
        await page.locator(`#post_${rootPost.id}`).hover();

        // 5. Open AI Actions menu
        await page.getByTestId(`ai-actions-menu`).click();

        // 6. Click "Find action items"
        await openAIMock.addCompletionMock(actionItemsResponse);
        await page.getByRole('button', { name: 'Find action items' }).click();

        // 7. Wait for results
        await aiPlugin.expectRHSOpenWithPost();

        // Expected Results: All action items identified regardless of format
        await expect(page.getByText('action items')).toBeVisible();
    });
});
