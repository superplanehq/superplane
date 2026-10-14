import { beforeEach, describe, expect, it, vi } from "bun:test";

import { DefaultPromptUnavailableError } from "./defaultAgentPrompt";
import { loadDefaultAgentPrompt } from "./loadDefaultAgentPrompt";

const { materializeDefaults } = vi.hoisted(() => ({ materializeDefaults: vi.fn() }));

vi.mock("@/api-client", () => ({
  factoriesMaterializeFactoryAutomationDefaults: materializeDefaults,
}));

const YAML_WITHOUT_PROMPT = `apiVersion: v1
kind: Canvas
metadata:
  name: Implement
spec:
  nodes:
    - id: implementation-agent-no-issue
      name: Agent
      type: TYPE_ACTION
      component: runnerClaudeCode
      configuration:
        steps:
          - name: Clone Repo
            type: bash
            command: git clone
`;

const input = {
  organizationId: "org-1",
  factoryId: "factory-1",
  automationId: "canvas-1",
  agentNodeId: "implementation-agent-no-issue",
};

describe("loadDefaultAgentPrompt", () => {
  beforeEach(() => {
    materializeDefaults.mockReset();
  });

  it("rejects with DefaultPromptUnavailableError when the default has no prompt step", async () => {
    materializeDefaults.mockResolvedValue({ data: { canvasYaml: YAML_WITHOUT_PROMPT } });

    await expect(loadDefaultAgentPrompt(input)).rejects.toBeInstanceOf(DefaultPromptUnavailableError);
    expect(materializeDefaults).toHaveBeenCalled();
  });

  it("keeps an API request failure as its own error", async () => {
    const failure = new Error("Network down");
    materializeDefaults.mockRejectedValue(failure);

    await expect(loadDefaultAgentPrompt(input)).rejects.toBe(failure);
  });
});
