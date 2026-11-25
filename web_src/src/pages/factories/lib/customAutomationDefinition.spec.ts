import { describe, expect, it } from "bun:test";
import {
  automationDefinitionCanvas,
  automationDefinitionTriggers,
  automationDefinitionYaml,
  parseAutomationForCanvas,
  newAutomationDefinition,
} from "./customAutomationDefinition";

describe("automation YAML validation", () => {
  it.each([undefined, 1, 86400])("preserves an optional timeout of %s", (timeout) => {
    const definition = { ...newAutomationDefinition("backlog"), timeout_seconds: timeout };
    expect(parseAutomationForCanvas(automationDefinitionYaml(definition), "backlog").timeout_seconds).toBe(timeout);
  });

  it.each([0, 86401, 1.5, "3600", null])("rejects an invalid timeout of %s", (timeout) => {
    const source =
      automationDefinitionYaml(newAutomationDefinition("backlog")) + `timeout_seconds: ${JSON.stringify(timeout)}\n`;
    expect(() => parseAutomationForCanvas(source, "backlog")).toThrow("timeout_seconds");
  });

  it.each([
    { model: "workspace-model" },
    { harness: "superplane" as const },
    { harness: "claude-code" as const, credentials: { source: "integration" as const, name: "team" } },
    { harness: "codex" as const, credentials: { source: "subscription" as const } },
    { harness: "openrouter" as const, credentials: { source: "hosted" as const } },
  ])("preserves supported agent configuration %j", (agent) => {
    const definition = { ...newAutomationDefinition("backlog"), agent };
    expect(parseAutomationForCanvas(automationDefinitionYaml(definition), "backlog").agent).toEqual(agent);
  });

  it.each([
    ["harness: superplane\n  credentials: {source: hosted}", "agent.credentials"],
    ["harness: claude-code\n  credentials: {source: subscription}", "agent.credentials.source"],
    ["harness: codex\n  credentials: {source: subscription, name: team}", "agent.credentials.name"],
    ["harness: openrouter\n  credentials: {source: hosted, name: team}", "agent.credentials.name"],
    ['harness: claude-code\n  credentials: {source: integration, name: ""}', "agent.credentials.name"],
    ["harness: codex\n  credentials: {source: unknown}", "agent.credentials.source"],
  ])("rejects unsupported credentials %s", (agent, path) => {
    const source = automationDefinitionYaml(newAutomationDefinition("backlog")) + `agent:\n  ${agent}\n`;
    expect(() => parseAutomationForCanvas(source, "backlog")).toThrow(path);
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
