import { describe, expect, it } from "bun:test";

import { workspaceAgentModelSource } from "./workspaceAgentModelSource";

describe("workspaceAgentModelSource", () => {
  it("uses the source from the factory response when another provider is also connected", () => {
    expect(workspaceAgentModelSource("AGENT_HARNESS_CLAUDE_CODE", "openrouter")).toBe("openrouter");
    expect(workspaceAgentModelSource("AGENT_HARNESS_CLAUDE_CODE", "custom")).toBe("custom");
  });

  it("does not guess a provider when the factory does not report one", () => {
    expect(workspaceAgentModelSource("AGENT_HARNESS_CLAUDE_CODE", "")).toBeNull();
    expect(workspaceAgentModelSource("AGENT_HARNESS_CLAUDE_CODE", undefined)).toBeNull();
    expect(workspaceAgentModelSource(undefined, undefined)).toBeNull();
  });

  it("keeps harness sources that do not use a saved connection", () => {
    expect(workspaceAgentModelSource("AGENT_HARNESS_SUPERPLANE", undefined)).toBe("hosted");
    expect(workspaceAgentModelSource("AGENT_HARNESS_CODEX", undefined)).toBe("openai");
    expect(workspaceAgentModelSource("AGENT_HARNESS_CURSOR", "openrouter")).toBeNull();
    expect(workspaceAgentModelSource("AGENT_HARNESS_CURSOR", undefined)).toBeNull();
  });
});
