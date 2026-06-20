import { beforeEach, describe, expect, it, vi } from "vitest";

const { init, captureConsoleIntegration, browserApiErrorsIntegration, globalHandlersIntegration } = vi.hoisted(() => ({
  init: vi.fn(),
  captureConsoleIntegration: vi.fn(() => ({ name: "captureConsole" })),
  browserApiErrorsIntegration: vi.fn(() => ({ name: "browserApiErrors" })),
  globalHandlersIntegration: vi.fn(() => ({ name: "globalHandlers" })),
}));

vi.mock("@sentry/react", () => ({
  init,
  captureConsoleIntegration,
  browserApiErrorsIntegration,
  globalHandlersIntegration,
  ErrorBoundary: () => null,
}));

type SentryWindow = Window & {
  SUPERPLANE_SENTRY_DSN?: string;
  SUPERPLANE_SENTRY_ENVIRONMENT?: string;
};

const matchesAny = (patterns: (string | RegExp)[], value: string): boolean =>
  patterns.some((pattern) => (typeof pattern === "string" ? value.includes(pattern) : pattern.test(value)));

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

  it("calls init with ignoreErrors and denyUrls when DSN is set", async () => {
    (window as SentryWindow).SUPERPLANE_SENTRY_DSN = "https://example@sentry.io/1";
    (window as SentryWindow).SUPERPLANE_SENTRY_ENVIRONMENT = "test";

    await import("@/sentry");

    expect(init).toHaveBeenCalledTimes(1);
    expect(init).toHaveBeenCalledWith(
      expect.objectContaining({
        dsn: "https://example@sentry.io/1",
        environment: "test",
        ignoreErrors: expect.any(Array),
        denyUrls: expect.any(Array),
      }),
    );
  });

  it("ignores known browser-extension noise messages", async () => {
    const { ignoreErrors } = await import("@/sentry");

    const noisyMessages = [
      "initEternlDomAPI: domId 178652-50895-174938 false",
      "initCardanoDomAPI something",
      "initNamiDomAPI: foo",
      "initLaceDomAPI bar",
      "Extension context invalidated.",
      "ResizeObserver loop limit exceeded",
      "ResizeObserver loop completed with undelivered notifications.",
    ];

    for (const message of noisyMessages) {
      expect(matchesAny(ignoreErrors, message)).toBe(true);
    }
  });

  it("does not match legitimate application errors", async () => {
    const { ignoreErrors } = await import("@/sentry");

    const legitimateMessages = [
      "TypeError: Cannot read properties of undefined",
      "Failed to fetch canvas data",
      "User not authorized",
    ];

    for (const message of legitimateMessages) {
      expect(matchesAny(ignoreErrors, message)).toBe(false);
    }
  });

  it("denies events originating from browser extension URLs", async () => {
    const { denyUrls } = await import("@/sentry");

    const extensionUrls = [
      "chrome-extension://abcdef/content.js",
      "moz-extension://abcdef/content.js",
      "safari-extension://abcdef/content.js",
      "safari-web-extension://abcdef/content.js",
    ];

    for (const url of extensionUrls) {
      expect(matchesAny(denyUrls, url)).toBe(true);
    }

    expect(matchesAny(denyUrls, "https://app.superplane.com/static/index.js")).toBe(false);
  });
});
