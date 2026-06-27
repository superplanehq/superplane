import { beforeEach, describe, expect, it, vi } from "vitest";

const { init } = vi.hoisted(() => ({
  init: vi.fn(),
}));

vi.mock("@sentry/react", () => ({
  init,
  captureConsoleIntegration: vi.fn(() => ({ name: "captureConsole" })),
  browserApiErrorsIntegration: vi.fn(() => ({ name: "browserApiErrors" })),
  globalHandlersIntegration: vi.fn(() => ({ name: "globalHandlers" })),
}));

interface SentryWindow extends Window {
  SUPERPLANE_SENTRY_DSN?: string;
  SUPERPLANE_SENTRY_ENVIRONMENT?: string;
}

function clearSentryWindow() {
  const sentryWindow = window as SentryWindow;
  delete sentryWindow.SUPERPLANE_SENTRY_DSN;
  delete sentryWindow.SUPERPLANE_SENTRY_ENVIRONMENT;
}

function makeMonacoCancellationError(): Error {
  const error = new Error("Canceled");
  error.name = "Canceled";
  return error;
}

describe("sentry init", () => {
  beforeEach(() => {
    init.mockClear();
    vi.resetModules();
    clearSentryWindow();
  });

  it("does not call Sentry.init when no DSN is configured", async () => {
    await import("@/sentry");
    expect(init).not.toHaveBeenCalled();
  });

  it("initializes Sentry when DSN is configured", async () => {
    (window as SentryWindow).SUPERPLANE_SENTRY_DSN = "https://example@sentry.io/1";
    (window as SentryWindow).SUPERPLANE_SENTRY_ENVIRONMENT = "test";

    await import("@/sentry");

    expect(init).toHaveBeenCalledTimes(1);
    const config = init.mock.calls[0][0];
    expect(config.dsn).toBe("https://example@sentry.io/1");
    expect(config.environment).toBe("test");
    expect(typeof config.beforeSend).toBe("function");
    expect(config.ignoreErrors).toEqual(expect.arrayContaining(["Canceled: Canceled"]));
  });
});

describe("isIgnoredCancellationError", () => {
  it("returns true for Monaco CancellationError shape", async () => {
    const { isIgnoredCancellationError } = await import("@/sentry");
    expect(isIgnoredCancellationError(makeMonacoCancellationError())).toBe(true);
  });

  it("returns true for DOMException AbortError", async () => {
    const { isIgnoredCancellationError } = await import("@/sentry");
    const error = new Error("The user aborted a request.");
    error.name = "AbortError";
    expect(isIgnoredCancellationError(error)).toBe(true);
  });

  it("returns false for unrelated errors", async () => {
    const { isIgnoredCancellationError } = await import("@/sentry");
    expect(isIgnoredCancellationError(new Error("boom"))).toBe(false);
    expect(isIgnoredCancellationError(new TypeError("nope"))).toBe(false);
    expect(isIgnoredCancellationError(null)).toBe(false);
    expect(isIgnoredCancellationError(undefined)).toBe(false);
    expect(isIgnoredCancellationError("Canceled")).toBe(false);
  });

  it("returns false when only the message matches", async () => {
    const { isIgnoredCancellationError } = await import("@/sentry");
    const error = new Error("Canceled");
    expect(isIgnoredCancellationError(error)).toBe(false);
  });
});

describe("beforeSend filter", () => {
  beforeEach(() => {
    init.mockClear();
    vi.resetModules();
    clearSentryWindow();
  });

  async function loadBeforeSend() {
    (window as SentryWindow).SUPERPLANE_SENTRY_DSN = "https://example@sentry.io/1";
    await import("@/sentry");
    const config = init.mock.calls[0][0];
    return config.beforeSend as (event: unknown, hint: unknown) => unknown;
  }

  it("drops events whose originalException is a Monaco CancellationError", async () => {
    const beforeSend = await loadBeforeSend();
    const result = beforeSend(
      { exception: { values: [{ type: "Error", value: "boom" }] } },
      { originalException: makeMonacoCancellationError() },
    );
    expect(result).toBeNull();
  });

  it("drops events whose serialized exception matches Canceled: Canceled", async () => {
    const beforeSend = await loadBeforeSend();
    const result = beforeSend(
      { exception: { values: [{ type: "Canceled", value: "Canceled" }] } },
      { originalException: undefined },
    );
    expect(result).toBeNull();
  });

  it("drops events with AbortError in the exception payload", async () => {
    const beforeSend = await loadBeforeSend();
    const result = beforeSend(
      { exception: { values: [{ type: "AbortError", value: "The user aborted a request." }] } },
      { originalException: undefined },
    );
    expect(result).toBeNull();
  });

  it("passes through unrelated errors unchanged", async () => {
    const beforeSend = await loadBeforeSend();
    const event = { exception: { values: [{ type: "TypeError", value: "x is not a function" }] } };
    const result = beforeSend(event, { originalException: new TypeError("x is not a function") });
    expect(result).toBe(event);
  });

  it("passes through events with no exception payload", async () => {
    const beforeSend = await loadBeforeSend();
    const event = { message: "log message" };
    const result = beforeSend(event, { originalException: undefined });
    expect(result).toBe(event);
  });
});
