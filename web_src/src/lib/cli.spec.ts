import { describe, expect, it } from "bun:test";
import { getInstallCommand } from "@/lib/cli";

describe("cli", () => {
  it("uses the universal installer", () => {
    expect(getInstallCommand()).toBe("curl -fsSL https://install.superplane.com/install.sh | sh");
  });
});
