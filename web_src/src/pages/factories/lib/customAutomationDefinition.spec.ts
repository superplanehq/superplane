import type { PlanningReviewStep } from "../pages/planningReviewMockup";
import type { AutomationDefinition } from "./customAutomationDefinition";
import { describe, expect, it } from "bun:test";
import {
  automationDefinitionCanvas,
  automationDefinitionSteps,
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

describe("chat continuation", () => {
  it("preserves the final Chat step through the visual editor", () => {
    const definition = newAutomationDefinition("done");
    definition.steps = [{ name: "Discuss", chat: "Discuss the result." }];
    const parsed = parseAutomationForCanvas(automationDefinitionYaml(definition), "done");
    const visual = automationDefinitionCanvas(parsed);
    expect(
      automationDefinitionSteps(visual.draft.components[0].configuration.steps as PlanningReviewStep[])[0].chat,
    ).toBe("Discuss the result.");
  });

  it.each([
    { runs_on_task: false },
    { triggers: [{ use: "schedule", config: { cron: "0 * * * *" } }] },
    { steps: [{ name: "Discuss", bash: "echo hello", chat: "Hello" }] },
    {
      steps: [
        { name: "Discuss", chat: "Hello" },
        { name: "Finish", bash: "true" },
      ],
    },
  ])("rejects chat without a task or outside the final prompt", (patch) => {
    const definition = {
      ...newAutomationDefinition("backlog"),
      triggers: [{ use: "manual" }],
      steps: [{ name: "Discuss", chat: "Hello" }],
      ...patch,
    };
    expect(() =>
      parseAutomationForCanvas(automationDefinitionYaml(definition as AutomationDefinition), "backlog"),
    ).toThrow();
  });
});
