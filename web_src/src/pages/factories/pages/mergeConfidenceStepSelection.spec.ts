import { describe, expect, it } from "bun:test";
import { enabledMergeChecks, stepsWithMergeChecks } from "./mergeConfidenceStepSelection";
import type { PlanningReviewStep } from "./planningReviewMockup";

const risk: PlanningReviewStep = {
  name: "Custom blast radius",
  type: "prompt",
  prompt: "Merge check: risk.\nKeep my rules.",
};
const security: PlanningReviewStep = {
  name: "Security",
  type: "prompt",
  prompt: "Merge check: security.\nDefault security.",
};
const custom: PlanningReviewStep = { name: "Custom", type: "prompt", prompt: "Merge check: new-check.\nCustom rules." };
const setup: PlanningReviewStep = { name: "Setup", type: "bash", command: "git status" };

describe("merge confidence step selection", () => {
  it("reads only known check headers in canonical order", () => {
    expect(enabledMergeChecks([security, custom, risk, setup])).toEqual(["risk", "security"]);
  });

  it("orders known checks while preserving custom checks and other step slots", () => {
    expect(stepsWithMergeChecks([setup, security, custom, risk], ["risk", "security"], [])).toEqual([
      setup,
      risk,
      custom,
      security,
    ]);
    expect(stepsWithMergeChecks([setup, risk, custom], ["risk", "security"], [security])).toEqual([
      setup,
      risk,
      security,
      custom,
    ]);
  });

  it("adds a runner placeholder only when no merge check remains and removes it when enabling a check", () => {
    const disabled = stepsWithMergeChecks([risk], [], []);
    expect(disabled).toEqual([
      {
        name: "No checks enabled",
        type: "prompt",
        prompt:
          "Enabled checks: none.\nNo merge confidence checks are enabled. Do not inspect the repository or call the report_merge_check tool.",
      },
    ]);
    expect(stepsWithMergeChecks(disabled, ["security"], [security])).toEqual([security]);
    expect(stepsWithMergeChecks([risk, custom], [], [])).toEqual([custom]);
  });

  it("rejects enabling a check missing from the bundled defaults", () => {
    expect(() => stepsWithMergeChecks([risk], ["risk", "security"], [])).toThrow("Security");
  });
});
