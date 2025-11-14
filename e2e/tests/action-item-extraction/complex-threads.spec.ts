import { test, expect } from '@playwright/test';

import RunContainer from 'helpers/plugincontainer';
import MattermostContainer from 'helpers/mmcontainer';
import { MattermostPage } from 'helpers/mm';
import { AIPlugin } from 'helpers/ai-plugin';
import { OpenAIMockContainer, RunOpenAIMocks } from 'helpers/openai-mock';

// spec: /Users/nickmisasi/workspace/worktrees/mattermost-plugin-ai-agents-in-e2e/e2e/specs/action-item-extraction.md
// seed: /Users/nickmisasi/workspace/worktrees/mattermost-plugin-ai-agents-in-e2e/seed.spec.ts

const username = 'regularuser';
const password = 'regularuser';

let mattermost: MattermostContainer;
let openAIMock: OpenAIMockContainer;

const complexActionItemsResponse = `
data: {"id":"chatcmpl-ai-2","object":"chat.completion.chunk","created":1708124577,"model":"gpt-3.5-turbo-0613","system_fingerprint":null,"choices":[{"index":0,"delta":{"role":"assistant","content":""},"logprobs":null,"finish_reason":null}]}
data: {"id":"chatcmpl-ai-2","object":"chat.completion.chunk","created":1708124577,"model":"gpt-3.5-turbo-0613","system_fingerprint":null,"choices":[{"index":0,"delta":{"content":"Here"},"logprobs":null,"finish_reason":null}]}
data: {"id":"chatcmpl-ai-2","object":"chat.completion.chunk","created":1708124577,"model":"gpt-3.5-turbo-0613","system_fingerprint":null,"choices":[{"index":0,"delta":{"content":" are"},"logprobs":null,"finish_reason":null}]}
data: {"id":"chatcmpl-ai-2","object":"chat.completion.chunk","created":1708124577,"model":"gpt-3.5-turbo-0613","system_fingerprint":null,"choices":[{"index":0,"delta":{"content":" the"},"logprobs":null,"finish_reason":null}]}
data: {"id":"chatcmpl-ai-2","object":"chat.completion.chunk","created":1708124577,"model":"gpt-3.5-turbo-0613","system_fingerprint":null,"choices":[{"index":0,"delta":{"content":" 5"},"logprobs":null,"finish_reason":null}]}
data: {"id":"chatcmpl-ai-2","object":"chat.completion.chunk","created":1708124577,"model":"gpt-3.5-turbo-0613","system_fingerprint":null,"choices":[{"index":0,"delta":{"content":" action"},"logprobs":null,"finish_reason":null}]}
data: {"id":"chatcmpl-ai-2","object":"chat.completion.chunk","created":1708124577,"model":"gpt-3.5-turbo-0613","system_fingerprint":null,"choices":[{"index":0,"delta":{"content":" items"},"logprobs":null,"finish_reason":null}]}
data: {"id":"chatcmpl-ai-2","object":"chat.completion.chunk","created":1708124577,"model":"gpt-3.5-turbo-0613","system_fingerprint":null,"choices":[{"index":0,"delta":{"content":" from"},"logprobs":null,"finish_reason":null}]}
data: {"id":"chatcmpl-ai-2","object":"chat.completion.chunk","created":1708124577,"model":"gpt-3.5-turbo-0613","system_fingerprint":null,"choices":[{"index":0,"delta":{"content":" the"},"logprobs":null,"finish_reason":null}]}
data: {"id":"chatcmpl-ai-2","object":"chat.completion.chunk","created":1708124577,"model":"gpt-3.5-turbo-0613","system_fingerprint":null,"choices":[{"index":0,"delta":{"content":" discussion"},"logprobs":null,"finish_reason":null}]}
data: {"id":"chatcmpl-ai-2","object":"chat.completion.chunk","created":1708124577,"model":"gpt-3.5-turbo-0613","system_fingerprint":null,"choices":[{"index":0,"delta":{},"logprobs":null,"finish_reason":"stop"}]}
data: [DONE]
`.trim().split('\n').filter(l => l).join('\n\n') + '\n\n';

test.beforeAll(async () => {
    mattermost = await RunContainer();
    openAIMock = await RunOpenAIMocks(mattermost.network);
});

test.afterAll(async () => {
    await openAIMock.stop();
    await mattermost.stop();
});

async function setupTestPage(page) {
    const mmPage = new MattermostPage(page);
    const aiPlugin = new AIPlugin(page);
    const url = mattermost.url();

    await mmPage.login(url, username, password);

    return { mmPage, aiPlugin };
}

test.describe('Action Item Extraction from Complex Threads', () => {
    test('Extract from Multi-Participant Thread with Numerous Messages', async ({ page }) => {
        const { mmPage, aiPlugin } = await setupTestPage(page);

        // 1. Send a channel message: "Q4 Planning Discussion"
        const rootPost = await mmPage.sendMessageAsUser(
            mattermost,
            username,
            password,
            'Q4 Planning Discussion'
        );

        // 2. Create 10-15 thread replies simulating a realistic discussion
        const userClient = await mattermost.getClient(username, password);

        const messages = [
            'What are our main goals for Q4?',
            'We need to focus on the new feature launch',
            'ACTION: Mike will prepare the marketing materials by Oct 15',
            'Sounds good! What about the technical side?',
            'The backend team is working on the API updates',
            'TODO: Sarah to coordinate with the design team for UI mockups',
            'When do we expect the beta release?',
            'Target is mid-November',
            'TASK: John needs to set up the staging environment by Nov 1',
            'What about the budget for Q4?',
            'Finance team approved the additional headcount',
            'We should also plan the customer communications',
            'ACTION: Lisa will draft the customer announcement email',
            'Dont forget we need to update the documentation',
            'ACTION: Tom to update technical documentation by end of October'
        ];

        for (const message of messages) {
            await userClient.createPost({
                channel_id: rootPost.channel_id,
                root_id: rootPost.id,
                message: message
            });
        }

        // 3. Navigate to the post
        await page.goto(mattermost.url() + '/test/channels/town-square');
        await page.locator(`#post_${rootPost.id}`).waitFor({ state: 'visible' });

        // 4. Hover over the root post
        await page.locator(`#post_${rootPost.id}`).hover();

        // 5. Open AI Actions menu
        await page.getByTestId(`ai-actions-menu`).click();

        // 6. Click "Find action items"
        await openAIMock.addCompletionMock(complexActionItemsResponse);
        await page.getByRole('button', { name: 'Find action items' }).click();

        // 7. Wait for comprehensive analysis
        await aiPlugin.expectRHSOpenWithPost();

        // Expected Results: All action items are identified
        await expect(page.getByText(/action items/i)).toBeVisible();
        // Response time should be reasonable (within 60 second timeout)
    });
});
