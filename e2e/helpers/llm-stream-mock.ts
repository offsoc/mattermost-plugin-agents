/**
 * LLM Stream Mock Generator
 *
 * Generates OpenAI-compatible Server-Sent Events (SSE) stream responses
 * with support for reasoning, annotations, and tool calls.
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
    delayMs?: number;
    chunkSize?: number;
}

/**
 * Generate SSE stream data for OpenAI mock with reasoning
 * Uses the OpenAI Realtime API format with reasoning events
 * @param text - The main response text
 * @param reasoning - The reasoning/thinking text
 * @param options - Additional options for annotations, tool calls, etc.
 * @returns SSE formatted string
 */
export function generateStreamWithReasoning(
    text: string,
    reasoning: string,
    options?: StreamOptions
): string {
    const chunks: string[] = [];

    // Response created event - Responses API format
    chunks.push(createSSEEvent('response.created', {
        id: 'resp_test_123',
        status: 'in_progress'
    }));

    // Response in progress
    chunks.push(createSSEEvent('response.in_progress', {}));

    // Add reasoning events if provided (OpenAI Responses API format)
    if (reasoning) {
        // Reasoning part added event
        chunks.push(createSSEEvent('response.reasoning_summary_part.added', {
            part_id: 'part_reasoning_1'
        }));

        // Split reasoning into chunks for realistic streaming
        const reasoningChunks = splitIntoChunks(reasoning, options?.chunkSize || 20);
        for (const chunk of reasoningChunks) {
            chunks.push(createSSEEvent('response.reasoning_summary_text.delta', {
                delta: chunk
            }));
        }

        // Reasoning text done
        chunks.push(createSSEEvent('response.reasoning_summary_text.done', {
            text: reasoning
        }));

        // Reasoning part done
        chunks.push(createSSEEvent('response.reasoning_summary_part.done', {
            part_id: 'part_reasoning_1'
        }));
    }

    // Add content part added event for text output
    if (text) {
        chunks.push(createSSEEvent('response.content_part.added', {
            part: {
                id: 'part_output_1',
                type: 'output_text'
            }
        }));

        // Add text content as output_text delta events (Responses API format)
        const textChunks = splitIntoChunks(text, options?.chunkSize || 10);
        for (const chunk of textChunks) {
            chunks.push(createSSEEvent('response.output_text.delta', {
                delta: chunk
            }));
        }

        // Output text done
        chunks.push(createSSEEvent('response.output_text.done', {
            text: text
        }));
    }

    // Add annotations if provided (sent with content_part.done event)
    if (options?.includeAnnotations && options.includeAnnotations.length > 0) {
        chunks.push(createSSEEvent('response.content_part.done', {
            part: {
                type: 'output_text',
                annotations: options.includeAnnotations
            }
        }));
    } else if (text) {
        // Content part done without annotations
        chunks.push(createSSEEvent('response.content_part.done', {
            part: {
                type: 'output_text'
            }
        }));
    }

    // Response completed event
    // The SDK expects the response field to contain the completed response object
    chunks.push(createSSEEvent('response.completed', {
        response: {
            id: 'resp_test_123',
            status: 'completed',
            usage: {
                input_tokens: 100,
                output_tokens: 50,
                total_tokens: 150
            }
        }
    }));

    chunks.push('data: [DONE]\n\n');

    return chunks.join('');
}

/**
 * Generate SSE stream with annotations only
 * @param text - The response text
 * @param annotations - Array of annotations to include
 * @returns SSE formatted string
 */
export function generateStreamWithAnnotations(
    text: string,
    annotations: Annotation[]
): string {
    return generateStreamWithReasoning(text, '', { includeAnnotations: annotations });
}

/**
 * Generate SSE stream with just text (baseline)
 * @param text - The response text
 * @returns SSE formatted string
 */
export function generateSimpleTextStream(text: string): string {
    return generateStreamWithReasoning(text, '');
}

/**
 * Generate complete stream with all features
 * @param text - Main response text
 * @param reasoning - Reasoning text
 * @param annotations - Citations/annotations
 * @returns SSE formatted string
 */
export function generateFullFeaturedStream(
    text: string,
    reasoning: string,
    annotations: Annotation[]
): string {
    return generateStreamWithReasoning(text, reasoning, { includeAnnotations: annotations });
}

/**
 * Generate stream that errors mid-way
 * @param partialText - Text before error occurs
 * @param partialReasoning - Reasoning before error occurs
 * @returns SSE formatted string with error
 */
export function generateErrorStream(
    partialText: string,
    partialReasoning?: string
): string {
    const chunks: string[] = [];

    // Response created event
    chunks.push(createSSEEvent('response.created', {
        id: 'resp_test_error',
        status: 'in_progress'
    }));

    chunks.push(createSSEEvent('response.in_progress', {}));

    // Add partial reasoning if provided
    if (partialReasoning) {
        chunks.push(createSSEEvent('response.reasoning_summary_part.added', {
            part_id: 'part_reasoning_1'
        }));

        chunks.push(createSSEEvent('response.reasoning_summary_text.delta', {
            delta: partialReasoning
        }));
    }

    // Add partial text if provided
    if (partialText) {
        chunks.push(createSSEEvent('response.content_part.added', {
            part: {
                id: 'part_output_1',
                type: 'output_text'
            }
        }));

        chunks.push(createSSEEvent('response.output_text.delta', {
            delta: partialText
        }));
    }

    // Add error event
    chunks.push(createSSEEvent('error', {
        message: 'Internal server error',
        type: 'server_error',
        code: 500
    }));

    return chunks.join('');
}

/**
 * Create a single SSE chunk (standard chat completion format)
 * @param data - Data object to serialize
 * @returns SSE formatted chunk
 */
function createSSEChunk(data: any): string {
    return `data: ${JSON.stringify(data)}\n\n`;
}

/**
 * Create a single SSE event (OpenAI Realtime API format)
 * @param eventType - Event type (e.g., 'response.reasoning_summary_text.delta')
 * @param data - Event data
 * @returns SSE formatted event
 */
function createSSEEvent(eventType: string, data: any): string {
    return `event: ${eventType}\ndata: ${JSON.stringify(data)}\n\n`;
}

/**
 * Split text into chunks for streaming simulation
 * @param text - Text to split
 * @param maxChunkSize - Maximum size of each chunk
 * @returns Array of text chunks
 */
function splitIntoChunks(text: string, maxChunkSize: number): string[] {
    const words = text.split(' ');
    const chunks: string[] = [];
    let currentChunk = '';

    for (const word of words) {
        if (currentChunk.length + word.length + 1 > maxChunkSize) {
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

// Pre-defined test responses for common scenarios

export const responseWithReasoning = generateStreamWithReasoning(
    'Based on my analysis, TypeScript is a strongly-typed superset of JavaScript.',
    'First, I need to analyze the question about TypeScript. I should consider its main features and benefits.'
);

export const responseWithSingleCitation = generateStreamWithAnnotations(
    'TypeScript is a great language for building scalable applications.',
    [
        {
            type: 'url_citation',
            start_index: 50,
            end_index: 50,
            url: 'https://www.typescriptlang.org/docs/',
            title: 'TypeScript Documentation',
            index: 1
        }
    ]
);

export const responseWithMultipleCitations = generateStreamWithAnnotations(
    'TypeScript provides static typing, which helps catch errors early. It also has excellent IDE support and a large community.',
    [
        {
            type: 'url_citation',
            start_index: 35,
            end_index: 35,
            url: 'https://www.typescriptlang.org/',
            title: 'TypeScript Official Site',
            index: 1
        },
        {
            type: 'url_citation',
            start_index: 90,
            end_index: 90,
            url: 'https://github.com/microsoft/TypeScript',
            title: 'TypeScript GitHub',
            index: 2
        },
        {
            type: 'url_citation',
            start_index: 130,
            end_index: 130,
            url: 'https://stackoverflow.com/questions/tagged/typescript',
            title: 'TypeScript on Stack Overflow',
            index: 3
        }
    ]
);

export const responseWithReasoningAndCitations = generateFullFeaturedStream(
    'TypeScript adds static typing to JavaScript, making it more maintainable.',
    'I should explain TypeScript benefits and provide reliable sources for further reading.',
    [
        {
            type: 'url_citation',
            start_index: 60,
            end_index: 60,
            url: 'https://www.typescriptlang.org/',
            title: 'TypeScript Home',
            index: 1
        }
    ]
);

export const longReasoningResponse = generateStreamWithReasoning(
    'Here is my detailed response.',
    'Step 1: First, I need to thoroughly analyze all aspects of the problem. This requires careful consideration of multiple factors.\n\nStep 2: Next, I should evaluate alternative approaches and their trade-offs.\n\nStep 3: Then, I need to synthesize the information and form a coherent conclusion.\n\nStep 4: Finally, I should present the findings in a clear and actionable manner.'
);
