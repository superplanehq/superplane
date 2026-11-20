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
      agentHarness: "AGENT_HARNESS_CODEX",
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
