import { describe, expect, it } from "bun:test";

import { agentRunnerStepTitles } from "./agentRunnerSteps";

describe("agentRunnerStepTitles", () => {
  it("returns each configured step title in order", () => {
    expect(
      agentRunnerStepTitles({
        steps: [
          { name: "Clone repo", type: "bash" },
          { name: "Write implementation plan", type: "prompt" },
          { name: "Use plan as output", type: "bash" },
        ],
      }),
    ).toEqual(["Clone repo", "Write implementation plan", "Use plan as output"]);
  });

  it("keeps a combined review step with its saved name", () => {
    const prompt = [
      "Report each enabled check with the report_merge_check tool.",
      "Enabled checks: risk, performance.",
      "Risk. Use this section only when risk is enabled.",
      "Look at the blast radius.",
      "Performance. Use this section only when performance is enabled.",
      "Look at the hot paths.",
    ].join("\n");

    const steps = {
      steps: [
        { name: "Checkout Pull Request", type: "bash" },
        { name: "Review Pull Request", type: "prompt", prompt },
      ],
    };

    expect(agentRunnerStepTitles(steps)).toEqual(["Checkout Pull Request", "Review Pull Request"]);
  });

  it("ignores malformed and blank steps", () => {
    expect(
      agentRunnerStepTitles({
        steps: [{ name: "Clone repo" }, null, { name: " " }, { title: "Wrong field" }],
      }),
    ).toEqual(["Clone repo"]);
    expect(agentRunnerStepTitles(undefined)).toEqual([]);
  });
});
