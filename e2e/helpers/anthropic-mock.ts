import {StartedTestContainer, GenericContainer, StartedNetwork, Wait} from "testcontainers";

/**
 * Anthropic Mock Container
 *
 * Uses Smocker to mock the Anthropic Messages API for testing.
 * Supports thinking/reasoning and citations.
 */

export class AnthropicMockContainer {
    container: StartedTestContainer;

    start = async (network: StartedNetwork) => {
        this.container = await new GenericContainer("thiht/smocker")
            .withExposedPorts(8081)
            .withNetwork(network)
            .withNetworkAliases("anthropic")
            .withWaitStrategy(Wait.forLogMessage("Starting mock server"))
            .start()

        await this.resetMocks();
    }

    resetMocks = async () => {
        const port = this.container.getMappedPort(8081)
        const response = await fetch(`http://localhost:${port}/reset`, {method: 'POST'})
        if (!response.ok) {
            throw new Error("Failed to reset mocks")
        }
    }

    private addMock = async (mock: any) => {
        const port = this.container.getMappedPort(8081)
        const response = await fetch(`http://localhost:${port}/mocks?reset=true`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify([mock])
        })
        if (!response.ok) {
            throw new Error(`Failed to add mock: ${response.statusText}`)
        }
    }

    /**
     * Add a mock response for Anthropic Messages API
     * @param response - SSE formatted response string
     * @param botPrefix - Optional bot prefix for multi-bot scenarios
     */
    addMessagesMock = async (response: string, botPrefix?: string) => {
        const prefix = botPrefix ? ("/" + botPrefix) : ""
        return this.addMock({
            request: {
                method: "POST",
                path: prefix + "/v1/messages",
            },
            context: {
                times: 100,
            },
            response: {
                status: 200,
                headers: {
                    "Content-Type": "text/event-stream",
                },
                body: response,
            },
        })
    }

    /**
     * Add a mock with request body matching
     * @param response - SSE formatted response
     * @param requestBodyContains - String that must be in request body
     * @param botPrefix - Optional bot prefix
     */
    addMessagesMockWithRequestBody = async (response: string, requestBodyContains: string, botPrefix?: string) => {
        const prefix = botPrefix ? ("/" + botPrefix) : ""
        return this.addMock({
            request: {
                method: "POST",
                path: prefix + "/v1/messages",
                body: {
                    matcher: "ShouldContainSubstring",
                    value: requestBodyContains
                }
            },
            context: {
                times: 100,
            },
            response: {
                status: 200,
                headers: {
                    "Content-Type": "text/event-stream",
                },
                body: response,
            },
        })
    }

    /**
     * Add error mock for testing error handling
     * @param statusCode - HTTP status code
     * @param errorMessage - Error message
     * @param botPrefix - Optional bot prefix
     */
    addErrorMock = async (statusCode: number, errorMessage: string, botPrefix?: string) => {
        const prefix = botPrefix ? ("/" + botPrefix) : ""
        return this.addMock({
            request: {
                method: "POST",
                path: prefix + "/v1/messages",
            },
            context: {
                times: 100,
            },
            response: {
                status: statusCode,
                headers: {
                    "Content-Type": "application/json",
                },
                body: JSON.stringify({
                    type: 'error',
                    error: {
                        type: 'api_error',
                        message: errorMessage
                    }
                }),
            },
        })
    }

    stop = async () => {
        await this.container.stop();
    }

    url = () => {
        const port = this.container.getMappedPort(8081)
        return `http://localhost:${port}`
    }
}

/**
 * Helper function to run Anthropic mocks
 * @param network - Docker network to attach to
 * @returns Started Anthropic mock container
 */
export async function RunAnthropicMocks(network: StartedNetwork): Promise<AnthropicMockContainer> {
    const anthropicMock = new AnthropicMockContainer();
    await anthropicMock.start(network);
    return anthropicMock;
}
