import { beforeEach, describe, expect, it, vi } from "vitest";

const { init, captureConsoleIntegration, browserApiErrorsIntegration, globalHandlersIntegration, ErrorBoundary } =
  vi.hoisted(() => ({
    init: vi.fn(),
    captureConsoleIntegration: vi.fn((opts: unknown) => ({ name: "captureConsole", opts })),
    browserApiErrorsIntegration: vi.fn((opts: unknown) => ({ name: "browserApiErrors", opts })),
    globalHandlersIntegration: vi.fn((opts: unknown) => ({ name: "globalHandlers", opts })),
    ErrorBoundary: vi.fn(),
  }));

vi.mock("@sentry/react", () => ({
  init,
  captureConsoleIntegration,
  browserApiErrorsIntegration,
  globalHandlersIntegration,
  ErrorBoundary,
}));

type SentryWindow = Window & {
  SUPERPLANE_SENTRY_DSN?: string;
  SUPERPLANE_SENTRY_ENVIRONMENT?: string;
};

describe("sentry init", () => {
  beforeEach(() => {
    init.mockClear();
    vi.resetModules();
    delete (window as SentryWindow).SUPERPLANE_SENTRY_DSN;
    delete (window as SentryWindow).SUPERPLANE_SENTRY_ENVIRONMENT;
  });

  it("does not call init when DSN is missing", async () => {
    await import("@/sentry");
    expect(init).not.toHaveBeenCalled();
  });

  it("ignores Dash0 SDK telemetry-failure console warnings", async () => {
    (window as SentryWindow).SUPERPLANE_SENTRY_DSN = "https://public@sentry.example.com/1";
    (window as SentryWindow).SUPERPLANE_SENTRY_ENVIRONMENT = "test";

    await import("@/sentry");

    expect(init).toHaveBeenCalledTimes(1);
    const initOptions = init.mock.calls[0]?.[0] as { ignoreErrors?: RegExp[] };
    const ignoreErrors = initOptions.ignoreErrors ?? [];

    const dash0FailureMessages = [
      "Failed to send telemetry to https://ingress.us-west-2.aws.dash0.com/v1/logs: 400 Bad Request",
      "Error sending telemetry to https://ingress.us-west-2.aws.dash0.com/v1/traces: TypeError: NetworkError",
      "Unable to send telemetry, fetch is not defined",
      "Failed to transmit logs Error: timed out",
      "Failed to transmit spans Error: timed out",
    ];

    for (const message of dash0FailureMessages) {
      expect(
        ignoreErrors.some((pattern) => pattern instanceof RegExp && pattern.test(message)),
        `expected "${message}" to be ignored`,
      ).toBe(true);
    }
  });

  it("does not ignore unrelated console warnings", async () => {
    (window as SentryWindow).SUPERPLANE_SENTRY_DSN = "https://public@sentry.example.com/1";

    await import("@/sentry");

    const initOptions = init.mock.calls[0]?.[0] as { ignoreErrors?: RegExp[] };
    const ignoreErrors = initOptions.ignoreErrors ?? [];

    const realApplicationMessages = [
      "Failed to load canvas: HTTP 500",
      "Unhandled promise rejection in /signup form",
      "TypeError: Cannot read properties of undefined (reading 'foo')",
    ];

    for (const message of realApplicationMessages) {
      expect(
        ignoreErrors.some((pattern) => pattern instanceof RegExp && pattern.test(message)),
        `expected "${message}" NOT to be ignored`,
      ).toBe(false);
    }
  });
});
