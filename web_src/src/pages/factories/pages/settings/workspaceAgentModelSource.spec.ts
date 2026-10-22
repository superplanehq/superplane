import { describe, expect, it } from "bun:test";

import { modelSourceFromIntegrationName, workspaceAgentModelSource } from "./workspaceAgentModelSource";

const anthropicAndOpenRouter = [
  { provider: "anthropic" as const, connected: true },
  { provider: "openai" as const, connected: false },
  { provider: "openrouter" as const, connected: true },
  { provider: "custom" as const, connected: false },
];

describe("workspaceAgentModelSource", () => {
  it("uses the saved connection provider when another provider is also connected", () => {
    expect(workspaceAgentModelSource("AGENT_HARNESS_CLAUDE_CODE", "openrouter", anthropicAndOpenRouter)).toBe(
      "openrouter",
    );
    expect(workspaceAgentModelSource("AGENT_HARNESS_CLAUDE_CODE", "custom", anthropicAndOpenRouter)).toBe("custom");
  });

  it("falls back to the first connected provider when no saved connection resolves", () => {
    expect(workspaceAgentModelSource("AGENT_HARNESS_CLAUDE_CODE", null, anthropicAndOpenRouter)).toBe("anthropic");
    expect(workspaceAgentModelSource(undefined, null, anthropicAndOpenRouter)).toBe("anthropic");
  });

  it("keeps harness sources that do not use a saved connection", () => {
    expect(workspaceAgentModelSource("AGENT_HARNESS_SUPERPLANE", "openrouter", anthropicAndOpenRouter)).toBe("hosted");
    expect(workspaceAgentModelSource("AGENT_HARNESS_CODEX", null, [])).toBe("openai");
    expect(workspaceAgentModelSource("AGENT_HARNESS_CURSOR", "openrouter", anthropicAndOpenRouter)).toBeNull();
  });
});

describe("modelSourceFromIntegrationName", () => {
  it("maps integration app names to model sources", () => {
    expect(modelSourceFromIntegrationName("claude")).toBe("anthropic");
    expect(modelSourceFromIntegrationName("openai")).toBe("openai");
    expect(modelSourceFromIntegrationName("openrouter")).toBe("openrouter");
    expect(modelSourceFromIntegrationName("customLlm")).toBe("custom");
    expect(modelSourceFromIntegrationName("github")).toBeNull();
  });
});
