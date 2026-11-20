import { describe, expect, it } from "bun:test";
import { automationDefinitionCanvas, newAutomationDefinition } from "./customAutomationDefinition";

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
