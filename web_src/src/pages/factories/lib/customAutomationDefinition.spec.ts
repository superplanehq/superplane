import { describe, expect, it } from "bun:test";
import {
  automationDefinitionCanvas,
  automationDefinitionTriggers,
  automationDefinitionYaml,
  parseAutomationForCanvas,
  newAutomationDefinition,
} from "./customAutomationDefinition";

describe("automation Canvas agent configuration", () => {
  it("uses the workspace agent when YAML omits the agent", () => {
    const visual = automationDefinitionCanvas(newAutomationDefinition("backlog"), {
      component: "runnerCodex",
      credentialSource: "subscription",
    });
    expect(visual.draft.components[0]?.component).toBe("runnerCodex");
  });

  it.each([
    { harness: "claude-code", credentials: { source: "integration", name: "team" } },
    { harness: "codex", credentials: { source: "subscription" } },
    { harness: "openrouter", credentials: { source: "hosted" } },
  ] as const)("preserves credentials for $harness", (agent) => {
    const definition = { ...newAutomationDefinition("backlog"), agent };
    expect(parseAutomationForCanvas(automationDefinitionYaml(definition), "backlog").agent).toEqual(agent);
  });

  it.each([
    { harness: "superplane", credentials: { source: "integration", name: "team" } },
    { harness: "claude-code", credentials: { source: "subscription" } },
    { harness: "codex", credentials: { source: "subscription", name: "team" } },
    { harness: "openrouter", credentials: { source: "hosted", name: "team" } },
    { harness: "codex", credentials: { source: "unknown" } },
  ])("rejects invalid credentials: %j", (agent) => {
    const yaml = automationDefinitionYaml(newAutomationDefinition("backlog"));
    expect(() => parseAutomationForCanvas(`${yaml}\nagent: ${JSON.stringify(agent)}`, "backlog")).toThrow(
      "agent.credentials",
    );
  });
});

describe("automation execution timeout", () => {
  it.each([undefined, 1, 86400])("accepts timeout %s", (timeout_seconds) => {
    const definition = { ...newAutomationDefinition("backlog"), timeout_seconds };
    expect(parseAutomationForCanvas(automationDefinitionYaml(definition), "backlog").timeout_seconds).toBe(
      timeout_seconds,
    );
  });

  it.each([0, 86401, 1.5, "60", null])("rejects timeout %j", (timeout) => {
    const yaml = automationDefinitionYaml(newAutomationDefinition("backlog"));
    expect(() => parseAutomationForCanvas(`${yaml}\ntimeout_seconds: ${JSON.stringify(timeout)}`, "backlog")).toThrow(
      "timeout_seconds",
    );
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
