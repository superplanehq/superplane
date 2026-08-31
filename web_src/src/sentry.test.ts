import type * as Sentry from "@sentry/react";
import { describe, expect, it } from "vitest";

import { isIgnoredConsoleMessage, isMonacoCanceledEvent } from "./sentry";

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
