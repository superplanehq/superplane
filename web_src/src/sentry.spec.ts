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

  it("configures Dash0 telemetry noise filters when DSN is present", async () => {
    (window as SentryWindow).SUPERPLANE_SENTRY_DSN = "https://public@sentry.example.com/1";
    (window as SentryWindow).SUPERPLANE_SENTRY_ENVIRONMENT = "test";

    await import("@/sentry");

    expect(init).toHaveBeenCalledTimes(1);
    const initOptions = init.mock.calls[0]?.[0] as {
      ignoreErrors?: RegExp[];
      beforeSend?: (event: unknown, hint: unknown) => unknown;
    };

    expect(initOptions.ignoreErrors?.length).toBeGreaterThan(0);
    expect(typeof initOptions.beforeSend).toBe("function");
  });
});
