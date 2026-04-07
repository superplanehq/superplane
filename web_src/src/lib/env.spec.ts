import { afterEach, describe, expect, it, vi } from "vitest";
import { isAgentReplEnabled, isUsagePageForced } from "@/lib/env";

afterEach(() => {
  vi.unstubAllEnvs();
});

describe("env", () => {
  it("reads the agent repl flag", () => {
    vi.stubEnv("VITE_ENABLE_AGENT_REPL", "false");

    expect(isAgentReplEnabled()).toBe(false);
  });

  it("reads the usage page override flag", () => {
    vi.stubEnv("VITE_FORCE_USAGE_PAGE", "true");

    expect(isUsagePageForced()).toBe(true);
  });
});
