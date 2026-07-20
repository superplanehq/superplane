import { describe, expect, it } from "vitest";

import { isIgnoredConsoleMessage } from "./sentry";

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
