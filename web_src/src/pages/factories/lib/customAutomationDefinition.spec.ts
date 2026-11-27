import { describe, expect, it } from "bun:test";
import {
  automationDefinitionCanvas,
  automationDefinitionTriggers,
  automationDefinitionYaml,
  parseAutomationForCanvas,
  newAutomationDefinition,
  type AutomationDefinition,
} from "./customAutomationDefinition";
import type { CustomAutomationColumn } from "./customAutomation";

describe("automation YAML validation", () => {
  it.each([undefined, "0s", "30s", "90m", "2h", "24h"])("preserves an optional timeout of %s", (timeout) => {
    const definition = { ...newAutomationDefinition("backlog"), timeout: timeout };
    expect(parseAutomationForCanvas(automationDefinitionYaml(definition), "backlog").timeout).toBe(timeout);
  });

  it.each(["-1s", "24h1s", "1.5s", "3600", "100000000000000000000h", 3600, null])(
    "rejects an invalid timeout of %s",
    (timeout) => {
      const source =
        automationDefinitionYaml(newAutomationDefinition("backlog")) + `timeout: ${JSON.stringify(timeout)}\n`;
      expect(() => parseAutomationForCanvas(source, "backlog")).toThrow("timeout");
    },
  );

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

describe("canonical execution options", () => {
  it("keeps nested false values and explicit choices while projecting durations", () => {
    const agent = {
      additional_instructions: "",
      thinking_level: "high",
      timeout: "120m",
      skip_setup_commands: false,
      skip_repository_checkout: false,
    };
    const definition = parseAutomationForCanvas(
      automationDefinitionYaml({ ...newAutomationDefinition("backlog"), agent }),
      "backlog",
    );
    expect(definition.agent).toEqual(agent);
    expect(automationDefinitionCanvas(definition).draft.components[0]?.configuration.executionTimeoutSeconds).toBe(
      7200,
    );
  });

  it("rejects conflicting timeout scopes at the agent field", () => {
    const definition = { ...newAutomationDefinition("backlog"), timeout: "2h", agent: { timeout: "90m" } };
    expect(() => parseAutomationForCanvas(automationDefinitionYaml(definition), "backlog")).toThrow("agent.timeout");
  });

  it.each([
    ["timeout_seconds: 3600", "timeout"],
    ["runs_on_task: false", "run_context"],
    ["skip_setup_commands: false", "agent.skip_setup_commands"],
    ["skip_repository_checkout: false", "agent.skip_repository_checkout"],
    ["triggers: [{use: manual}]", "uses"],
    ['triggers: [{uses: schedule, config: {cron: "0 9 * * *", utc_offset: 1}}]', "timezone"],
  ])("rejects replaced syntax %s", (field, replacement) => {
    const original = automationDefinitionYaml(newAutomationDefinition("backlog"));
    const source = field.startsWith("triggers:")
      ? original.replace(/triggers:[\s\S]*?steps:/, field + "\nsteps:")
      : original + field + "\n";
    expect(() => parseAutomationForCanvas(source, "backlog")).toThrow(replacement);
  });

  it("keeps schedule plus manual context free of a task", () => {
    const definition: AutomationDefinition = {
      ...newAutomationDefinition("verify"),
      triggers: [{ uses: "schedule", config: { cron: "0 9 * * *", timezone: "-3.5" } }, { uses: "manual" }],
    };
    const parsed = parseAutomationForCanvas(automationDefinitionYaml(definition), "verify");
    const visual = automationDefinitionCanvas(parsed);
    expect(visual.runsOnTask).toBe(false);
    expect(automationDefinitionTriggers(visual.triggers, "verify")).toEqual(definition.triggers);
  });
});

// A form save must not turn a leave trigger into an enter trigger.
describe("leave trigger conversion", () => {
  it.each(["backlog", "implement", "verify", "done"] as const)("preserves leave triggers in %s", (stage) => {
    const definition = newAutomationDefinition(stage);
    definition.triggers = [{ id: "leave", uses: "task_leaves_stage" }];
    const parsed = parseAutomationForCanvas(automationDefinitionYaml(definition), stage);
    const visual = automationDefinitionCanvas(parsed);
    expect(visual.triggers[0]?.mode).toBe("leave");
    expect(automationDefinitionTriggers(visual.triggers, stage)).toEqual(definition.triggers);
    expect(() =>
      parseAutomationForCanvas(automationDefinitionYaml({ ...definition, run_context: "workspace" }), stage),
    ).toThrow();
  });
});

describe("webhook trigger conversion", () => {
  const webhookDefinition = (): AutomationDefinition => ({
    ...newAutomationDefinition("backlog"),
    triggers: [{ id: "intake", uses: "webhook", config: { webhook_id: "webhook-1" } }],
  });

  it("preserves a Backlog webhook trigger and runs it with no task", () => {
    const definition = webhookDefinition();
    const visual = automationDefinitionCanvas(
      parseAutomationForCanvas(automationDefinitionYaml(definition), "backlog"),
    );

    expect(visual.runsOnTask).toBe(false);
    expect(visual.triggers[0]).toEqual(expect.objectContaining({ mode: "webhook", webhookId: "webhook-1" }));
    expect(automationDefinitionTriggers(visual.triggers, "backlog")).toEqual(definition.triggers);
  });

  it.each<CustomAutomationColumn>(["implement", "verify", "done"])("accepts a webhook trigger in %s", (stage) => {
    const visual = automationDefinitionCanvas(
      parseAutomationForCanvas(automationDefinitionYaml(webhookDefinition()), stage),
    );
    expect(visual.runsOnTask).toBe(false);
    expect(visual.triggers[0]?.mode).toBe("webhook");
  });

  it.each(["task_enters_stage", "manual"] as const)("keeps %s next to a webhook on a custom stage", (uses) => {
    const stage: CustomAutomationColumn = "custom-qa";
    const definition = webhookDefinition();
    definition.triggers.unshift({ id: "task-trigger", uses });
    const visual = automationDefinitionCanvas(parseAutomationForCanvas(automationDefinitionYaml(definition), stage));
    expect(visual.runsOnTask).toBeUndefined();
    expect(automationDefinitionTriggers(visual.triggers, stage)).toEqual(definition.triggers);
  });

  it("accepts a custom stage automation that only a webhook starts", () => {
    const stage: CustomAutomationColumn = "custom-qa";
    const definition = webhookDefinition();
    const visual = automationDefinitionCanvas(parseAutomationForCanvas(automationDefinitionYaml(definition), stage));
    expect(automationDefinitionTriggers(visual.triggers, stage)).toEqual(definition.triggers);
  });

  const duplicate = webhookDefinition();
  duplicate.triggers.push({ id: "again", uses: "webhook", config: { webhook_id: "webhook-1" } });
  it.each<[CustomAutomationColumn, AutomationDefinition, string]>([
    ["backlog", { ...webhookDefinition(), run_context: "task" }, "run_context"],
    ["backlog", duplicate, "triggers[1].config.webhook_id"],
  ])("rejects an invalid webhook trigger in %s", (stage, definition, path) => {
    expect(() => parseAutomationForCanvas(automationDefinitionYaml(definition), stage)).toThrow(path);
  });
});
