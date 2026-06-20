import { beforeEach, describe, expect, it, vi } from "vitest";

const { init } = vi.hoisted(() => ({
  init: vi.fn(),
}));

vi.mock("@sentry/react", async () => {
  const actual = await vi.importActual<typeof import("@sentry/react")>("@sentry/react");
  return {
    ...actual,
    init,
  };
});

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

  it("calls init with ignoreErrors when DSN is set", async () => {
    (window as SentryWindow).SUPERPLANE_SENTRY_DSN = "https://example@sentry.io/1";
    (window as SentryWindow).SUPERPLANE_SENTRY_ENVIRONMENT = "test";

    await import("@/sentry");

    expect(init).toHaveBeenCalledTimes(1);
    const config = init.mock.calls[0]?.[0] as {
      dsn: string;
      environment: string;
      ignoreErrors: (string | RegExp)[];
    };
    expect(config.dsn).toBe("https://example@sentry.io/1");
    expect(config.environment).toBe("test");
    expect(config.ignoreErrors).toEqual(expect.arrayContaining([expect.any(RegExp)]));
  });

  describe("ignoredErrorPatterns", () => {
    it("matches Cardano wallet extension noise (initEternlDomAPI, initNamiDomAPI, …)", async () => {
      const { ignoredErrorPatterns } = await import("@/sentry");

      const noisyMessages = [
        "initEternlDomAPI: href https://app.superplane.com/login?redirect=%2F",
        "initNamiDomAPI: href https://app.superplane.com/",
        "initFlintDomAPI: href https://app.superplane.com/",
        "initYoroiDomAPI: href https://app.superplane.com/",
        "initLaceDomAPI: href https://app.superplane.com/",
        "initGeroDomAPI: href https://app.superplane.com/",
        "initTyphonDomAPI: href https://app.superplane.com/",
      ];

      for (const message of noisyMessages) {
        const matches = ignoredErrorPatterns.some((pattern) =>
          pattern instanceof RegExp ? pattern.test(message) : message.includes(pattern),
        );
        expect(matches, `expected pattern to match: ${message}`).toBe(true);
      }
    });

    it("does not match unrelated application errors", async () => {
      const { ignoredErrorPatterns } = await import("@/sentry");

      const realErrors = [
        "TypeError: Cannot read properties of undefined (reading 'foo')",
        "Failed to fetch",
        "Network request failed",
        "SyntaxError: Unexpected token",
        "init: something else",
        "DomAPI is broken",
      ];

      for (const message of realErrors) {
        const matches = ignoredErrorPatterns.some((pattern) =>
          pattern instanceof RegExp ? pattern.test(message) : message.includes(pattern),
        );
        expect(matches, `expected pattern NOT to match: ${message}`).toBe(false);
      }
    });
  });
});
