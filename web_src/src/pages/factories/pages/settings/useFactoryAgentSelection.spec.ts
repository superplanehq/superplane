import { describe, expect, it } from "vitest";

import { providerFor } from "./useFactoryAgentSelection";

describe("providerFor", () => {
  it("uses the persisted Anthropic provider before legacy heuristics", () => {
    expect(
      providerFor({
        agentProvider: "AGENT_PROVIDER_ANTHROPIC",
        agentHarness: "AGENT_HARNESS_CODEX",
      }),
    ).toBe("anthropic");
  });
});
