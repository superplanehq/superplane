import type * as SentryReact from "@sentry/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const { init } = vi.hoisted(() => ({
  init: vi.fn(),
}));

vi.mock("@sentry/react", async () => {
  const actual = await vi.importActual<typeof SentryReact>("@sentry/react");
  return {
    ...actual,
    init,
  };
});

type SentryWindow = Window & {
  SUPERPLANE_SENTRY_DSN?: string;
  SUPERPLANE_SENTRY_ENVIRONMENT?: string;
};

function matchesAnyIgnoredError(message: string, patterns: (string | RegExp)[]): boolean {
  return patterns.some((pattern) => (typeof pattern === "string" ? message.includes(pattern) : pattern.test(message)));
}

describe("sentry init", () => {
  beforeEach(() => {
    init.mockClear();
    vi.resetModules();
    delete (window as SentryWindow).SUPERPLANE_SENTRY_DSN;
    delete (window as SentryWindow).SUPERPLANE_SENTRY_ENVIRONMENT;
  });

  it("calls init with ignoreErrors when DSN is configured", async () => {
    (window as SentryWindow).SUPERPLANE_SENTRY_DSN = "https://test@sentry.example/1";
    (window as SentryWindow).SUPERPLANE_SENTRY_ENVIRONMENT = "test-env";

    const { IGNORED_ERRORS } = await import("@/sentry");

    expect(init).toHaveBeenCalledTimes(1);
    expect(init).toHaveBeenCalledWith(
      expect.objectContaining({
        dsn: "https://test@sentry.example/1",
        environment: "test-env",
        ignoreErrors: IGNORED_ERRORS,
      }),
    );
  });

  it("does not call init when DSN is missing", async () => {
    await import("@/sentry");
    expect(init).not.toHaveBeenCalled();
  });
});

describe("sentry IGNORED_ERRORS", () => {
  it("drops minified-vendor ReferenceErrors like 'Can't find variable: J4'", async () => {
    const { IGNORED_ERRORS } = await import("@/sentry");

    // The exact error reported by Sentry for this regression.
    expect(matchesAnyIgnoredError("Can't find variable: J4", IGNORED_ERRORS)).toBe(true);
    expect(matchesAnyIgnoredError("ReferenceError: Can't find variable: J4", IGNORED_ERRORS)).toBe(true);

    // Other typical Terser-style minified identifiers.
    expect(matchesAnyIgnoredError("Can't find variable: ABC12", IGNORED_ERRORS)).toBe(true);
    expect(matchesAnyIgnoredError("Can't find variable: $a9", IGNORED_ERRORS)).toBe(true);
    expect(matchesAnyIgnoredError("K1 is not defined", IGNORED_ERRORS)).toBe(true);
  });

  it("does not drop ReferenceErrors that look like real application identifiers", async () => {
    const { IGNORED_ERRORS } = await import("@/sentry");

    expect(matchesAnyIgnoredError("Can't find variable: userProfile", IGNORED_ERRORS)).toBe(false);
    expect(matchesAnyIgnoredError("Can't find variable: counterValue", IGNORED_ERRORS)).toBe(false);
    expect(matchesAnyIgnoredError("ReferenceError: handleSubmit is not defined", IGNORED_ERRORS)).toBe(false);
  });

  it("drops common browser-extension and browser-quirk noise", async () => {
    const { IGNORED_ERRORS } = await import("@/sentry");

    expect(matchesAnyIgnoredError("Can't find variable: __AutoFillPopupClose__", IGNORED_ERRORS)).toBe(true);
    expect(matchesAnyIgnoredError("Can't find variable: _AutofillCallbackHandler", IGNORED_ERRORS)).toBe(true);
    expect(matchesAnyIgnoredError("ResizeObserver loop limit exceeded", IGNORED_ERRORS)).toBe(true);
    expect(matchesAnyIgnoredError("Load failed", IGNORED_ERRORS)).toBe(true);
    expect(matchesAnyIgnoredError("Failed to fetch", IGNORED_ERRORS)).toBe(true);
    expect(matchesAnyIgnoredError("Script error.", IGNORED_ERRORS)).toBe(true);
  });
});
