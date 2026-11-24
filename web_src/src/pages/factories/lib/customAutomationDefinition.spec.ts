import { describe, expect, it } from "bun:test";
import {
  automationDefinitionCanvas,
  automationDefinitionTriggers,
  automationDefinitionYaml,
  parseAutomationForCanvas,
  newAutomationDefinition,
} from "./customAutomationDefinition";

function automationWithAgent(agent: unknown): string {
  return `apiVersion: v2
kind: Automation
name: Credential parsing
triggers:
  - use: manual
steps: []
agent: ${JSON.stringify(agent)}
`;
}

describe("automation YAML agent parsing", () => {
  it("preserves a model-only agent", () => {
    const parsed = parseAutomationForCanvas(automationWithAgent({ model: "custom-model" }), "backlog");
    expect(parsed.agent).toEqual({ model: "custom-model" });
  });

  it.each(["superplane", "claude-code", "codex", "openrouter"] as const)(
    "accepts %s with omitted credentials",
    (harness) => {
      const parsed = parseAutomationForCanvas(automationWithAgent({ harness }), "backlog");
      expect(parsed.agent).toEqual({ harness, model: undefined });
      const withModel = parseAutomationForCanvas(automationWithAgent({ harness, model: "custom-model" }), "backlog");
      expect(withModel.agent).toEqual({ harness, model: "custom-model" });
    },
  );

  it.each([
    {
      harness: "claude-code",
      credentials: { source: "integration", name: "Claude connection" },
    },
    { harness: "codex", credentials: { source: "subscription" } },
    { harness: "openrouter", credentials: { source: "hosted" } },
  ])("preserves $credentials.source credentials for $harness", ({ harness, credentials }) => {
    const parsed = parseAutomationForCanvas(
      automationWithAgent({ harness, model: "custom-model", credentials }),
      "backlog",
    );
    expect(parsed.agent).toEqual({ harness, model: "custom-model", credentials });
  });

  it.each([
    ["missing integration name", { source: "integration" }, "agent.credentials.name · Enter a nonblank string."],
    [
      "blank integration name",
      { source: "integration", name: "   " },
      "agent.credentials.name · Enter a nonblank string.",
    ],
    [
      "non-string integration name",
      { source: "integration", name: 42 },
      "agent.credentials.name · Enter a nonblank string.",
    ],
    [
      "multiline integration name",
      { source: "integration", name: "Claude\nconnection" },
      "agent.credentials.name · Enter a single line to use the visual editor.",
    ],
    [
      "named subscription",
      { source: "subscription", name: "ChatGPT" },
      "agent.credentials.name · Remove the subscription name.",
    ],
    [
      "named hosted credentials",
      { source: "hosted", name: "OpenRouter" },
      "agent.credentials.name · Remove the hosted connection name.",
    ],
    ["unknown source", { source: "unknown" }, "agent.credentials.source · Use integration, hosted, or subscription."],
    ["missing source", {}, "agent.credentials.source · Use integration, hosted, or subscription."],
    ["null mapping", null, "agent.credentials · Enter a mapping."],
    ["scalar mapping", "hosted", "agent.credentials · Enter a mapping."],
    ["list mapping", [], "agent.credentials · Enter a mapping."],
    [
      "unknown credential field",
      { source: "hosted", token: "unexpected" },
      "agent.credentials.token · Remove the unknown field.",
    ],
  ] as const)("rejects %s", (_label, credentials, message) => {
    expect(() => parseAutomationForCanvas(automationWithAgent({ harness: "codex", credentials }), "backlog")).toThrow(
      new Error(message),
    );
  });

  it.each([
    ["unsupported harness", { harness: "unknown" }, "agent.harness · Choose a supported agent."],
    ["missing harness", {}, "agent.harness · Choose a supported agent."],
    [
      "credentials without a harness",
      { model: "custom-model", credentials: { source: "hosted" } },
      "agent.harness · Choose a supported agent.",
    ],
    ["unknown agent field", { harness: "codex", extra: true }, "agent.extra · Remove the unknown field."],
    [
      "non-Codex subscription",
      { harness: "claude-code", credentials: { source: "subscription" } },
      "agent.credentials.source · A ChatGPT subscription requires Codex.",
    ],
    [
      "named non-Codex subscription",
      { harness: "openrouter", credentials: { source: "subscription", name: "ChatGPT" } },
      "agent.credentials.source · A ChatGPT subscription requires Codex.",
    ],
    [
      "SuperPlane credentials",
      { harness: "superplane", credentials: { source: "hosted" } },
      "agent.credentials · SuperPlane uses workspace credentials.",
    ],
    [
      "malformed SuperPlane credentials",
      { harness: "superplane", credentials: null },
      "agent.credentials · SuperPlane uses workspace credentials.",
    ],
  ] as const)("rejects %s", (_label, agent, message) => {
    expect(() => parseAutomationForCanvas(automationWithAgent(agent), "backlog")).toThrow(new Error(message));
  });
});

describe("automation Canvas agent configuration", () => {
  it("uses the workspace agent when YAML omits the agent", () => {
    const visual = automationDefinitionCanvas(newAutomationDefinition("backlog"), {
      component: "runnerCodex",
      credentialSource: "subscription",
    });
    expect(visual.draft.components[0]?.component).toBe("runnerCodex");
  });
});

// A form save must not turn a leave trigger into an enter trigger.
describe("leave trigger conversion", () => {
  it.each(["backlog", "implement", "verify", "done"] as const)("preserves leave triggers in %s", (stage) => {
    const definition = newAutomationDefinition(stage);
    definition.triggers = [{ id: "leave", use: "task_leaves_stage" }];
    const parsed = parseAutomationForCanvas(automationDefinitionYaml(definition), stage);
    const visual = automationDefinitionCanvas(parsed);
    expect(visual.triggers[0]?.mode).toBe("leave");
    expect(automationDefinitionTriggers(visual.triggers, stage)).toEqual(definition.triggers);
    expect(() =>
      parseAutomationForCanvas(automationDefinitionYaml({ ...definition, runs_on_task: false }), stage),
    ).toThrow();
  });
});
