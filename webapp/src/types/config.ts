// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

export type WebSearchGoogleConfig = {
    apiKey: string;
    searchEngineId: string;
    resultLimit: number;
    apiURL: string;
};

export type WebSearchConfig = {
    enabled: boolean;
    provider: string;
    google: WebSearchGoogleConfig;
};

export type Config = {
    services: ServiceData[];
    bots: BotConfig[];
    defaultBotName: string;
    transcriptBackend?: string;
    enableLLMTrace: boolean;
    enableTokenUsageLogging: boolean;
    enableCallSummary: boolean;
    allowedUpstreamHostnames: string;
    embeddingSearchConfig: EmbeddingSearchConfig;
    mcp: MCPConfig;
    webSearch: WebSearchConfig;
};
