import { describe, expect, it } from "bun:test";
import {
  automationDefinitionCanvas,
  automationDefinitionTriggers,
  automationDefinitionYaml,
  parseAutomationForCanvas,
  newAutomationDefinition,
} from "./customAutomationDefinition";

describe("automation Canvas agent configuration", () => {
  it.each([
    ["private-models", "custom"],
    ["router", undefined],
  ])("uses the selected %s connection's model catalog", (name, provider) => {
    const definition = newAutomationDefinition("backlog");
    definition.agent = {
      harness: "openrouter",
      model: "selected-model",
      credentials: { source: "integration", name },
    };
    const visual = automationDefinitionCanvas(definition, undefined, [
      { metadata: { name: "router", integrationName: "openrouter" } },
      { metadata: { name: "private-models", integrationName: "customLlm" } },
    ]);

    expect(visual.draft.components[0]?.configuration).toMatchObject({
      model: "selected-model",
      credentials: { source: "integration", name },
    });
    expect(visual.draft.components[0]?.configuration.llmProvider).toBe(provider);
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
