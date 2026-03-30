import { afterEach, describe, expect, it } from "vitest";
import { detectPlatform, getInstallCommand } from "@/lib/cli";

const originalUserAgent = navigator.userAgent;

function setUserAgent(userAgent: string) {
  Object.defineProperty(window.navigator, "userAgent", {
    configurable: true,
    value: userAgent,
  });
}

afterEach(() => {
  setUserAgent(originalUserAgent);
});

describe("cli", () => {
  it("detects linux arm platforms from the user agent", () => {
    setUserAgent("Mozilla/5.0 (X11; Linux aarch64)");

    expect(detectPlatform()).toBe("linux-arm64");
  });

  it("defaults to darwin amd64 when the user agent is not linux arm", () => {
    setUserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 14_0)");

    expect(detectPlatform()).toBe("darwin-amd64");
  });

  it("builds the install command for the detected platform", () => {
    expect(getInstallCommand("linux-arm64")).toContain("superplane-cli-linux-arm64");
  });
});
