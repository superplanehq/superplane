import { describe, expect, it } from "vitest";

import { isIgnoredConsoleMessage } from "@/sentry";

describe("isIgnoredConsoleMessage", () => {
  it("ignores Dash0 telemetry export failures", () => {
    expect(isIgnoredConsoleMessage("Failed to send telemetry to https://ingress.dash0.com")).toBe(true);
    expect(isIgnoredConsoleMessage("Error sending telemetry to endpoint")).toBe(true);
    expect(isIgnoredConsoleMessage("Failed to fetch")).toBe(true);
  });

  it("ignores the Vue Devtools browser-extension conflict message", () => {
    expect(
      isIgnoredConsoleMessage(
        "Another version of Vue Devtools seems to be installed. Please enable only one version at a time.",
      ),
    ).toBe(true);
  });

  it("does not ignore genuine application console messages", () => {
    expect(isIgnoredConsoleMessage("Something broke in the app")).toBe(false);
    expect(isIgnoredConsoleMessage("Unexpected token in JSON")).toBe(false);
  });
});
