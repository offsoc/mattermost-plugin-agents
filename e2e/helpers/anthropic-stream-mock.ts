/**
 * Anthropic Messages API Stream Mock Generator
 *
 * Generates Anthropic-compatible Server-Sent Events (SSE) stream responses
 * with support for thinking (reasoning), citations, and tool calls.
 *
 * Format based on: https://docs.anthropic.com/en/api/messages-streaming
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

export interface StreamOptions {
    includeAnnotations?: Annotation[];
    includeToolCalls?: boolean;
    chunkSize?: number;
}

/**
 * Generate Anthropic Messages API SSE stream with thinking/reasoning
 * @param text - The main response text
 * @param thinking - The thinking/reasoning text
 * @param options - Additional options for annotations
 * @returns SSE formatted string
 */
export function generateAnthropicStreamWithThinking(
    text: string,
    thinking: string,
    options?: StreamOptions
): string {
    const events: string[] = [];

    // Message start event
    events.push(createSSEEvent('message_start', {
        type: 'message_start',
        message: {
            id: 'msg_test_123',
            type: 'message',
            role: 'assistant',
            content: [],
            model: 'claude-3-5-sonnet-20241022',
            stop_reason: null,
            usage: {
                input_tokens: 100,
                output_tokens: 0
            }
        }
    }));

    // Thinking content block (if provided)
    if (thinking) {
        // Thinking block start
        events.push(createSSEEvent('content_block_start', {
            type: 'content_block_start',
            index: 0,
            content_block: {
                type: 'thinking'
            }
        }));

        // Thinking deltas (chunked)
        const thinkingChunks = splitIntoChunks(thinking, options?.chunkSize || 20);
        for (const chunk of thinkingChunks) {
            events.push(createSSEEvent('content_block_delta', {
                type: 'content_block_delta',
                index: 0,
                delta: {
                    type: 'thinking_delta',
                    thinking: chunk
                }
            }));
        }

        // Signature delta (opaque verification field required by Anthropic)
        events.push(createSSEEvent('content_block_delta', {
            type: 'content_block_delta',
            index: 0,
            delta: {
                type: 'signature_delta',
                signature: 'mock_signature_123'
            }
        }));

        // Thinking block stop
        events.push(createSSEEvent('content_block_stop', {
            type: 'content_block_stop',
            index: 0
        }));
    }

    // Text content block
    if (text) {
        const textBlockIndex = thinking ? 1 : 0;

        // Text block start
        events.push(createSSEEvent('content_block_start', {
            type: 'content_block_start',
            index: textBlockIndex,
            content_block: {
                type: 'text',
                text: ''
            }
        }));

        // Text deltas (chunked)
        const textChunks = splitIntoChunks(text, options?.chunkSize || 10);
        for (const chunk of textChunks) {
            events.push(createSSEEvent('content_block_delta', {
                type: 'content_block_delta',
                index: textBlockIndex,
                delta: {
                    type: 'text_delta',
                    text: chunk
                }
            }));
        }

        // Text block stop
        events.push(createSSEEvent('content_block_stop', {
            type: 'content_block_stop',
            index: textBlockIndex
        }));
    }

    // Message delta with usage
    events.push(createSSEEvent('message_delta', {
        type: 'message_delta',
        delta: {
            stop_reason: 'end_turn',
            stop_sequence: null
        },
        usage: {
            output_tokens: text.split(' ').length + thinking.split(' ').length
        }
    }));

    // Message stop event
    events.push(createSSEEvent('message_stop', {
        type: 'message_stop'
    }));

    return events.join('');
}

/**
 * Generate Anthropic stream with text only (no thinking)
 * @param text - The response text
 * @returns SSE formatted string
 */
export function generateSimpleAnthropicStream(text: string): string {
    return generateAnthropicStreamWithThinking(text, '');
}

/**
 * Generate Anthropic stream with error
 * @param partialText - Text before error
 * @param partialThinking - Thinking before error
 * @returns SSE formatted string with error
 */
export function generateAnthropicErrorStream(
    partialText: string,
    partialThinking?: string
): string {
    const events: string[] = [];

    events.push(createSSEEvent('message_start', {
        type: 'message_start',
        message: {
            id: 'msg_test_error',
            type: 'message',
            role: 'assistant',
            content: [],
            model: 'claude-3-5-sonnet-20241022',
            stop_reason: null,
            usage: {
                input_tokens: 100,
                output_tokens: 0
            }
        }
    }));

    if (partialThinking) {
        events.push(createSSEEvent('content_block_start', {
            type: 'content_block_start',
            index: 0,
            content_block: { type: 'thinking' }
        }));

        events.push(createSSEEvent('content_block_delta', {
            type: 'content_block_delta',
            index: 0,
            delta: {
                type: 'thinking_delta',
                thinking: partialThinking
            }
        }));
    }

    // Error event
    events.push(createSSEEvent('error', {
        type: 'error',
        error: {
            type: 'api_error',
            message: 'Internal server error'
        }
    }));

    return events.join('');
}

/**
 * Create a single SSE event
 * @param eventType - Event type (e.g., 'message_start', 'content_block_delta')
 * @param data - Event data object
 * @returns SSE formatted event
 */
function createSSEEvent(eventType: string, data: any): string {
    return `event: ${eventType}\ndata: ${JSON.stringify(data)}\n\n`;
}

/**
 * Split text into chunks for streaming simulation
 * @param text - Text to split
 * @param maxChunkSize - Maximum size of each chunk (in words)
 * @returns Array of text chunks
 */
function splitIntoChunks(text: string, maxChunkSize: number): string[] {
    if (!text) return [];

    const words = text.split(' ');
    const chunks: string[] = [];
    let currentChunk = '';

    for (const word of words) {
        if (currentChunk.length + word.length + 1 > maxChunkSize * 5) {
            if (currentChunk) {
                chunks.push(currentChunk);
            }
            currentChunk = word;
        } else {
            currentChunk += (currentChunk ? ' ' : '') + word;
        }
    }

    if (currentChunk) {
        chunks.push(currentChunk);
    }

    return chunks;
}

// Pre-defined test responses

export const simpleAnthropicResponse = generateSimpleAnthropicStream(
    'Hello! How can I assist you today?'
);

export const anthropicResponseWithThinking = generateAnthropicStreamWithThinking(
    'Based on my analysis, TypeScript is a strongly-typed superset of JavaScript.',
    'First, I need to analyze the question about TypeScript. I should consider its main features and benefits.'
);

export const anthropicResponseWithLongThinking = generateAnthropicStreamWithThinking(
    'Here is my comprehensive answer.',
    'Step 1: First, I need to thoroughly analyze all aspects of the problem. This requires careful consideration of multiple factors.\n\nStep 2: Next, I should evaluate alternative approaches and their trade-offs.\n\nStep 3: Then, I need to synthesize the information and form a coherent conclusion.\n\nStep 4: Finally, I should present the findings in a clear and actionable manner.'
);
