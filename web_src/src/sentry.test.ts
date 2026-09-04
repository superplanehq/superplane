import type * as Sentry from "@sentry/react";
import { describe, expect, it } from "vitest";

import {
  isIgnoredConsoleMessage,
  isMonacoCanceledEvent,
  normalizeApiRouteTemplate,
  normalizeSentryFingerprint,
} from "./sentry";

function buildEvent(value: string | undefined, filenames: string[]): Sentry.ErrorEvent {
  return {
    exception: {
      values: [
        {
          value,
          stacktrace: {
            frames: filenames.map((filename) => ({ filename })),
          },
        },
      ],
    },
  } as Sentry.ErrorEvent;
}

describe("isIgnoredConsoleMessage", () => {
  it("ignores Dash0 telemetry export failures", () => {
    expect(isIgnoredConsoleMessage("Failed to send telemetry to https://dash0")).toBe(true);
    expect(isIgnoredConsoleMessage("Error sending telemetry to https://dash0")).toBe(true);
    expect(isIgnoredConsoleMessage("Failed to fetch")).toBe(true);
  });

  it("ignores the Vue Devtools browser-extension conflict message", () => {
    expect(
      isIgnoredConsoleMessage(
        "Another version of Vue Devtools seems to be installed. Please enable only one version at a time.",
      ),
    ).toBe(true);
  });

  it("keeps genuine application console messages", () => {
    expect(isIgnoredConsoleMessage("Something actually broke")).toBe(false);
    expect(isIgnoredConsoleMessage("Unexpected token in JSON")).toBe(false);
  });

  it("returns false for non-string messages", () => {
    expect(isIgnoredConsoleMessage(undefined)).toBe(false);
    expect(isIgnoredConsoleMessage(null)).toBe(false);
    expect(isIgnoredConsoleMessage(42)).toBe(false);
  });
});

describe("isMonacoCanceledEvent", () => {
  it("ignores a Canceled rejection whose frames all come from monaco-editor", () => {
    const event = buildEvent("Canceled", [
      "https://cdn.jsdelivr.net/npm/monaco-editor@0.52.2/esm/vs/base/common/async.js",
      "https://cdn.jsdelivr.net/npm/monaco-editor@0.52.2/esm/vs/editor/browser/services/clipboardService.js",
    ]);

    expect(isMonacoCanceledEvent(event)).toBe(true);
  });

  it("ignores a Canceled rejection with a mix of monaco-editor and @sentry frames", () => {
    const event = buildEvent("Canceled", [
      "https://cdn.jsdelivr.net/npm/monaco-editor@0.52.2/esm/vs/base/common/async.js",
      "node_modules/@sentry/browser/build/npm/esm/instrument.js",
    ]);

    expect(isMonacoCanceledEvent(event)).toBe(true);
  });

  it("keeps a Canceled rejection that includes an application frame", () => {
    const event = buildEvent("Canceled", [
      "https://cdn.jsdelivr.net/npm/monaco-editor@0.52.2/esm/vs/base/common/async.js",
      "web_src/src/components/TextFieldRenderer.tsx",
    ]);

    expect(isMonacoCanceledEvent(event)).toBe(false);
  });

  it("keeps a Canceled rejection with no frames", () => {
    const event = buildEvent("Canceled", []);

    expect(isMonacoCanceledEvent(event)).toBe(false);
  });

  it("keeps events whose exception value is not exactly Canceled", () => {
    const event = buildEvent("Some other error", [
      "https://cdn.jsdelivr.net/npm/monaco-editor@0.52.2/esm/vs/base/common/async.js",
    ]);

    expect(isMonacoCanceledEvent(event)).toBe(false);
  });
});

function buildMessageEvent(message: string, requestUrl?: string): Sentry.Event {
  return {
    message,
    request: requestUrl ? { url: requestUrl } : undefined,
  } as Sentry.Event;
}

describe("normalizeApiRouteTemplate", () => {
  it("replaces the organization slug with a placeholder", () => {
    expect(normalizeApiRouteTemplate("/api/v1/organizations/superplane/byok-models")).toBe(
      "/api/v1/organizations/{org}/byok-models",
    );
  });

  it("replaces a UUID organization id with the same placeholder", () => {
    expect(normalizeApiRouteTemplate("/api/v1/organizations/3ee1aa47-3a60-4c1f-b645-0b9859ab91f8/byok-models")).toBe(
      "/api/v1/organizations/{org}/byok-models",
    );
  });

  it("collapses UUID segments in other api paths", () => {
    expect(normalizeApiRouteTemplate("/api/v1/canvases/3ee1aa47-3a60-4c1f-b645-0b9859ab91f8/runs")).toBe(
      "/api/v1/canvases/{id}/runs",
    );
  });

  it("returns undefined for paths outside /api/v1/", () => {
    expect(normalizeApiRouteTemplate("/login")).toBeUndefined();
  });
});

describe("normalizeSentryFingerprint", () => {
  it("groups a slug URL and a UUID URL for the same route into one fingerprint", () => {
    const slugEvent = buildMessageEvent(
      "GET /api/v1/organizations/superplane/byok-models failed: 500",
      "https://app.superplane.com/api/v1/organizations/superplane/byok-models",
    );
    const uuidEvent = buildMessageEvent(
      "GET /api/v1/organizations/3ee1aa47-3a60-4c1f-b645-0b9859ab91f8/byok-models failed: 500",
      "https://app.superplane.com/api/v1/organizations/3ee1aa47-3a60-4c1f-b645-0b9859ab91f8/byok-models",
    );

    const normalizedSlug = normalizeSentryFingerprint(slugEvent);
    const normalizedUuid = normalizeSentryFingerprint(uuidEvent);

    expect(normalizedSlug.fingerprint).toBeDefined();
    expect(normalizedSlug.fingerprint).toEqual(normalizedUuid.fingerprint);
  });

  it("derives the api path from the message when no request url is set", () => {
    const event = buildMessageEvent("GET /api/v1/organizations/superplane/byok-models failed: 500");

    const normalized = normalizeSentryFingerprint(event);

    expect(normalized.fingerprint).toEqual(["http-error", "UNKNOWN", "/api/v1/organizations/{org}/byok-models", "500"]);
  });

  it("leaves unrelated events unchanged", () => {
    const event = buildMessageEvent("Something unrelated went wrong");

    const normalized = normalizeSentryFingerprint(event);

    expect(normalized.fingerprint).toBeUndefined();
    expect(normalized).toEqual(event);
  });

  it("does not override an existing fingerprint", () => {
    const event = {
      ...buildMessageEvent(
        "GET /api/v1/organizations/superplane/byok-models failed: 500",
        "/api/v1/organizations/superplane/byok-models",
      ),
      fingerprint: ["custom"],
    } as Sentry.Event;

    const normalized = normalizeSentryFingerprint(event);

    expect(normalized.fingerprint).toEqual(["custom"]);
  });
});
