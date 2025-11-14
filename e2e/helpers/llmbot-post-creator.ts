import MattermostContainer from './mmcontainer';

/**
 * Helper for creating LLMBot posts with reasoning and citations
 * Creates posts directly via Mattermost API with props already set
 * This bypasses the LLM API complexity and tests the frontend components directly
 */

export interface Annotation {
    type: string;
    start_index: number;
    end_index: number;
    url: string;
    title: string;
    cited_text?: string;
    index: number;
}

export interface CreateLLMBotPostOptions {
    message: string;
    reasoning?: string;
    reasoningSignature?: string;
    annotations?: Annotation[];
    channelId: string;
    requesterUserId: string;
    botUsername?: string;
}

export class LLMBotPostCreator {
    private mattermost: MattermostContainer;
    private botId: string | null = null;

    constructor(mattermost: MattermostContainer) {
        this.mattermost = mattermost;
    }

    /**
     * Initialize by getting the bot ID
     * Call this once before creating posts
     */
    async initialize(botUsername: string = 'claude'): Promise<void> {
        const adminClient = await this.mattermost.getAdminClient();
        const botUser = await adminClient.getUserByUsername(botUsername);
        this.botId = botUser.id;
    }

    /**
     * Create an LLMBot post with reasoning and/or citations
     * @param options - Post creation options
     * @returns The created post object
     */
    async createPost(options: CreateLLMBotPostOptions): Promise<any> {
        if (!this.botId) {
            throw new Error('LLMBotPostCreator not initialized. Call initialize() first.');
        }

        const adminClient = await this.mattermost.getAdminClient();

        const props: any = {
            llm_requester_user_id: options.requesterUserId,
        };

        if (options.reasoning) {
            props.reasoning_summary = options.reasoning;
        }

        if (options.reasoningSignature) {
            props.reasoning_signature = options.reasoningSignature;
        }

        if (options.annotations && options.annotations.length > 0) {
            props.annotations = JSON.stringify(options.annotations);
        }

        // Create post as the bot user
        const post = await adminClient.createPost({
            channel_id: options.channelId,
            message: options.message,
            user_id: this.botId,
            props: props,
        });

        return post;
    }

    /**
     * Create a DM channel between bot and user
     * @param userId - User ID to create DM with
     * @returns Channel ID
     */
    async createDMChannel(userId: string): Promise<string> {
        if (!this.botId) {
            throw new Error('LLMBotPostCreator not initialized. Call initialize() first.');
        }

        const adminClient = await this.mattermost.getAdminClient();
        const channel = await adminClient.createDirectChannel([this.botId, userId]);
        return channel.id;
    }

    /**
     * Helper to create sample annotations for testing
     */
    static createSampleAnnotation(url: string, startIndex: number, title: string, index: number): Annotation {
        return {
            type: 'url_citation',
            start_index: startIndex,
            end_index: startIndex,
            url: url,
            title: title,
            index: index
        };
    }
}
