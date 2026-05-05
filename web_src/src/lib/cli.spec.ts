import { describe, expect, it } from "vitest";
import { getCliBinaryURL, getInstallCommand, getManualInstallCommand } from "@/lib/cli";

describe("cli", () => {
  it.each([
    "darwin-arm64",
    "darwin-amd64",
    "linux-amd64",
    "linux-arm64",
    "windows-browser-with-wsl-linux-terminal",
    "unknown-platform",
  ])("uses the universal installer by default for %s", () => {
    expect(getInstallCommand()).toBe("curl -fsSL https://install.superplane.com/install.sh | sh");
  });

  it.each([
    ["darwin-arm64", "https://install.superplane.com/superplane-cli-darwin-arm64"],
    ["darwin-amd64", "https://install.superplane.com/superplane-cli-darwin-amd64"],
    ["linux-amd64", "https://install.superplane.com/superplane-cli-linux-amd64"],
    ["linux-arm64", "https://install.superplane.com/superplane-cli-linux-arm64"],
  ] as const)("builds latest manual binary URLs for %s", (platform, url) => {
    expect(getCliBinaryURL(platform)).toBe(url);
  });

  it("builds version-pinned manual binary URLs with the same platform naming", () => {
    expect(getCliBinaryURL("darwin-arm64", "v0.1.6")).toBe(
      "https://install.superplane.com/v0.1.6/superplane-cli-darwin-arm64",
    );
  });

  it("builds manual install commands for explicit advanced platform selection", () => {
    expect(getManualInstallCommand("linux-arm64")).toBe(
      "curl -L https://install.superplane.com/superplane-cli-linux-arm64 -o superplane && chmod +x superplane && sudo mv superplane /usr/local/bin/superplane",
    );
  });
});
