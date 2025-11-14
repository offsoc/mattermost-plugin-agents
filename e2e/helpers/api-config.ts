/**
 * API Configuration for E2E Tests
 *
 * Manages configuration for running tests with real APIs.
 * Used exclusively by llmbot-post-component tests.
 */

export interface ProviderConfig {
    name: string;
    type: 'openaicompatible' | 'anthropic';
    apiKey: string;
    apiURL: string;
    defaultModel: string;
    reasoningEnabled: boolean;
    thinkingBudget?: number;
    reasoningEffort?: string;
}

export interface APITestConfig {
    hasAnthropicKey: boolean;
    hasOpenAIKey: boolean;
    shouldRunTests: boolean;
    providers: ProviderConfig[];
}

/**
 * Get API configuration from environment variables
 * @returns Configuration object with available providers
 */
export function getAPIConfig(): APITestConfig {
    const anthropicKey = process.env.ANTHROPIC_API_KEY;
    const openaiKey = process.env.OPENAI_API_KEY;

    const hasAnthropicKey = !!anthropicKey && anthropicKey.length > 0;
    const hasOpenAIKey = !!openaiKey && openaiKey.length > 0;
    const shouldRunTests = hasAnthropicKey || hasOpenAIKey;

    const providers: ProviderConfig[] = [];

    if (hasAnthropicKey) {
        providers.push({
            name: 'Anthropic',
            type: 'anthropic',
            apiKey: anthropicKey!,
            apiURL: 'https://api.anthropic.com',
            defaultModel: 'claude-3-7-sonnet-20250219',
            reasoningEnabled: true,
            thinkingBudget: 8192,
        });
    }

    if (hasOpenAIKey) {
        providers.push({
            name: 'OpenAI',
            type: 'openaicompatible',
            apiKey: openaiKey!,
            apiURL: 'https://api.openai.com/v1',
            defaultModel: 'o4-mini',
            reasoningEnabled: true,
            reasoningEffort: 'high',
        });
    }

    return {
        hasAnthropicKey,
        hasOpenAIKey,
        shouldRunTests,
        providers,
    };
}

/**
 * Get skip message for tests when no API keys are present
 * @returns Skip message or null if tests should run
 */
export function getSkipMessage(): string | null {
    const config = getAPIConfig();

    if (!config.shouldRunTests) {
        return 'Skipping llmbot-post-component tests: No ANTHROPIC_API_KEY or OPENAI_API_KEY found in environment. Set one to run these tests with real APIs.';
    }

    return null;
}

/**
 * Log API configuration for debugging
 */
export function logAPIConfig(): void {
    const config = getAPIConfig();

    if (!config.shouldRunTests) {
        console.log('⚠️  LLMBot tests SKIPPED - No API keys configured');
        console.log('   Set ANTHROPIC_API_KEY or OPENAI_API_KEY to enable');
        return;
    }

    console.log('🔴 LLMBot tests using REAL APIs:');
    config.providers.forEach(provider => {
        console.log(`   - ${provider.name}: ${provider.defaultModel}`);
    });
    console.log('   ⚠️  This will incur API costs (~$0.05 per run)');
}
