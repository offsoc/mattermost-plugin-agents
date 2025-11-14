import { Page } from '@playwright/test';

/**
 * WebSocket Event Simulator
 *
 * Simulates WebSocket events for LLMBot post streaming updates.
 * These events are sent from the backend to update posts in real-time.
 */

export interface PostUpdateWebsocketMessage {
    post_id: string;
    next?: string;
    control?: string;
    tool_call?: string;
    reasoning?: string;
    annotations?: string;
}

export interface Annotation {
    type: string;
    start_index: number;
    end_index: number;
    url: string;
    title: string;
    cited_text?: string;
    index: number;
}

/**
 * Simulate reasoning stream events via WebSocket
 * This mimics the backend sending reasoning chunks during LLM generation
 *
 * @param page - Playwright page object
 * @param postId - ID of the post being updated
 * @param reasoning - The reasoning text to stream
 */
export async function simulateReasoningStream(
    page: Page,
    postId: string,
    reasoning: string
): Promise<void> {
    // Simulate start event
    await page.evaluate(
        ({ id, control }) => {
            window.dispatchEvent(
                new CustomEvent('websocket_event', {
                    detail: {
                        event: 'posted',
                        data: {
                            post_id: id,
                            control: control
                        }
                    }
                })
            );
        },
        { id: postId, control: 'start' }
    );

    // Split reasoning into chunks for realistic streaming
    const chunks = reasoning.match(/.{1,50}/g) || [reasoning];
    let accumulatedReasoning = '';

    for (const chunk of chunks) {
        accumulatedReasoning += chunk;

        await page.evaluate(
            ({ id, text }) => {
                window.dispatchEvent(
                    new CustomEvent('websocket_event', {
                        detail: {
                            event: 'posted',
                            data: {
                                post_id: id,
                                control: 'reasoning_summary',
                                reasoning: text
                            }
                        }
                    })
                );
            },
            { id: postId, text: accumulatedReasoning }
        );

        // Small delay to simulate network latency
        await page.waitForTimeout(50);
    }

    // Send reasoning done event
    await page.evaluate(
        ({ id, text }) => {
            window.dispatchEvent(
                new CustomEvent('websocket_event', {
                    detail: {
                        event: 'posted',
                        data: {
                            post_id: id,
                            control: 'reasoning_summary_done',
                            reasoning: text
                        }
                    }
                })
            );
        },
        { id: postId, text: accumulatedReasoning }
    );
}

/**
 * Simulate text content stream events
 *
 * @param page - Playwright page object
 * @param postId - ID of the post being updated
 * @param text - The text content to stream
 */
export async function simulateTextStream(
    page: Page,
    postId: string,
    text: string
): Promise<void> {
    const words = text.split(' ');
    let accumulatedText = '';

    for (const word of words) {
        accumulatedText += (accumulatedText ? ' ' : '') + word;

        await page.evaluate(
            ({ id, content }) => {
                window.dispatchEvent(
                    new CustomEvent('websocket_event', {
                        detail: {
                            event: 'posted',
                            data: {
                                post_id: id,
                                next: content
                            }
                        }
                    })
                );
            },
            { id: postId, content: accumulatedText }
        );

        await page.waitForTimeout(50);
    }
}

/**
 * Simulate annotation/citation events
 *
 * @param page - Playwright page object
 * @param postId - ID of the post being updated
 * @param annotations - Array of annotations to add
 */
export async function simulateAnnotationEvent(
    page: Page,
    postId: string,
    annotations: Annotation[]
): Promise<void> {
    await page.evaluate(
        ({ id, annotationsJson }) => {
            window.dispatchEvent(
                new CustomEvent('websocket_event', {
                    detail: {
                        event: 'posted',
                        data: {
                            post_id: id,
                            control: 'annotations',
                            annotations: annotationsJson
                        }
                    }
                })
            );
        },
        { id: postId, annotationsJson: JSON.stringify(annotations) }
    );
}

/**
 * Simulate stream end event
 *
 * @param page - Playwright page object
 * @param postId - ID of the post being updated
 */
export async function simulateStreamEnd(
    page: Page,
    postId: string
): Promise<void> {
    await page.evaluate(
        ({ id }) => {
            window.dispatchEvent(
                new CustomEvent('websocket_event', {
                    detail: {
                        event: 'posted',
                        data: {
                            post_id: id,
                            control: 'end'
                        }
                    }
                })
            );
        },
        { id: postId }
    );
}

/**
 * Simulate stream cancellation
 *
 * @param page - Playwright page object
 * @param postId - ID of the post being updated
 */
export async function simulateStreamCancel(
    page: Page,
    postId: string
): Promise<void> {
    await page.evaluate(
        ({ id }) => {
            window.dispatchEvent(
                new CustomEvent('websocket_event', {
                    detail: {
                        event: 'posted',
                        data: {
                            post_id: id,
                            control: 'cancel'
                        }
                    }
                })
            );
        },
        { id: postId }
    );
}

/**
 * Simulate complete stream with reasoning, text, and annotations
 *
 * @param page - Playwright page object
 * @param postId - ID of the post being updated
 * @param reasoning - Reasoning text
 * @param text - Response text
 * @param annotations - Array of annotations
 */
export async function simulateCompleteStream(
    page: Page,
    postId: string,
    reasoning?: string,
    text?: string,
    annotations?: Annotation[]
): Promise<void> {
    // Start
    await page.evaluate(
        ({ id }) => {
            window.dispatchEvent(
                new CustomEvent('websocket_event', {
                    detail: {
                        event: 'posted',
                        data: {
                            post_id: id,
                            control: 'start'
                        }
                    }
                })
            );
        },
        { id: postId }
    );

    // Reasoning (if provided)
    if (reasoning) {
        await simulateReasoningStream(page, postId, reasoning);
    }

    // Text content (if provided)
    if (text) {
        await simulateTextStream(page, postId, text);
    }

    // Annotations (if provided)
    if (annotations && annotations.length > 0) {
        await simulateAnnotationEvent(page, postId, annotations);
    }

    // End
    await simulateStreamEnd(page, postId);
}

/**
 * Simulate stream error
 *
 * @param page - Playwright page object
 * @param postId - ID of the post being updated
 * @param errorMessage - Error message to display
 */
export async function simulateStreamError(
    page: Page,
    postId: string,
    errorMessage: string
): Promise<void> {
    await page.evaluate(
        ({ id, error }) => {
            window.dispatchEvent(
                new CustomEvent('websocket_event', {
                    detail: {
                        event: 'post_error',
                        data: {
                            post_id: id,
                            error: error
                        }
                    }
                })
            );
        },
        { id: postId, error: errorMessage }
    );
}
