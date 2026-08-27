import { describe, expect, it } from "vitest";

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

  it("ignores malformed and blank steps", () => {
    expect(
      agentRunnerStepTitles({
        steps: [{ name: "Clone repo" }, null, { name: " " }, { title: "Wrong field" }],
      }),
    ).toEqual(["Clone repo"]);
    expect(agentRunnerStepTitles(undefined)).toEqual([]);
  });
});
