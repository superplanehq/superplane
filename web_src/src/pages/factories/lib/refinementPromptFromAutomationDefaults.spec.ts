import { describe, expect, it } from "bun:test";

import { refinementPromptFromAutomationDefaults } from "./refinementPromptFromAutomationDefaults";

const FACTORY_DEFAULT = "Plan the task.\n\nTask:\n{{ root().data.workOrder }}";

function defaultsYaml(nodes: string): string {
  return `apiVersion: v1
kind: Canvas
metadata:
  name: Backlog
spec:
  edges: []
  nodes:
${nodes}`;
}

const refineTaskNode = `    - id: refine-task
      name: Refine Task
      type: TYPE_ACTION
      component: runnerClaudeCode
      configuration:
        model: sonnet
        steps:
          - name: Clone repository
            type: bash
            command: git clone
            workingDirectory: repo
          - name: Refine Task
            type: prompt
            workingDirectory: repo
            prompt: |-
              Plan the task.

              Task:
              {{ root().data.workOrder }}
`;

describe("refinementPromptFromAutomationDefaults", () => {
  it("reads the Refine Task prompt from defaults output, including the task template line", () => {
    const prompt = refinementPromptFromAutomationDefaults(defaultsYaml(refineTaskNode));

    expect(prompt).toBe(FACTORY_DEFAULT);
  });

  it("returns an empty result when the refine-task node is missing", () => {
    const yaml = defaultsYaml(`    - id: analyze
      name: Analyze
      type: TYPE_ACTION
      component: runnerClaudeCode
      configuration:
        steps:
          - name: Refine Task
            type: prompt
            prompt: ${JSON.stringify(FACTORY_DEFAULT)}
`);

    expect(refinementPromptFromAutomationDefaults(yaml)).toBe("");
  });
});
