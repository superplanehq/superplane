import { describe, expect, it } from "bun:test";

import {
  expandMergeConfidenceSteps,
  mergeCheckPromptBody,
  newMergeConfidenceStep,
  withMergeCheckHeader,
} from "./mergeConfidenceSteps";
import type { PlanningReviewStep } from "./planningReviewMockup";

const REVIEW = [
  "Review this pull request. The diff is in /tmp/pr.diff.",
  "Do not change files.",
  "",
  "Report each enabled check with the report_merge_check tool.",
  "Call the tool once for every check named in Enabled checks.",
  "Do not write a file.",
  "",
  "Enabled checks: risk, security.",
  "",
  "Risk. Use this section only when risk is enabled.",
  "A higher score means more risk.",
  "",
  "Performance. Use this section only when performance is enabled.",
  "Read the performance practices.",
  "",
  "Security. Use this section only when security is enabled.",
  "Read the security practices.",
  "",
].join("\n");

describe("expandMergeConfidenceSteps", () => {
  it("turns each enabled check into a step and keeps checkout", () => {
    const steps: PlanningReviewStep[] = [
      { name: "Checkout Pull Request", type: "bash", command: "git clone" },
      { name: "Review Pull Request", type: "prompt", prompt: REVIEW, workingDirectory: "repo" },
    ];

    const expanded = expandMergeConfidenceSteps(steps);

    expect(expanded.map((step) => step.name)).toEqual(["Checkout Pull Request", "Blast radius", "Security"]);
    expect(expanded[1]?.prompt?.startsWith("Merge check: risk.\n")).toBe(true);
    expect(expanded[1]?.prompt).toContain("A higher score means more risk.");
    expect(expanded[1]?.prompt).toContain("The diff is in /tmp/pr.diff.");
    expect(expanded[1]?.prompt).not.toContain("Enabled checks:");
    expect(expanded[2]?.prompt).toContain("check is security.");
    expect(expanded.map((step) => step.name)).not.toContain("Performance");
  });

  it("keeps a disabled check out of the next step", () => {
    const prompt = REVIEW.replace("Enabled checks: risk, security.", "Enabled checks: security.");
    const expanded = expandMergeConfidenceSteps([
      { name: "Review Pull Request", type: "prompt", prompt, workingDirectory: "repo" },
    ]);

    expect(expanded.map((step) => step.name)).toEqual(["Security"]);
    expect(expanded[0]?.prompt).not.toContain("more risk");
  });

  it("leaves a step list that is already split unchanged", () => {
    const steps: PlanningReviewStep[] = [
      { name: "Checkout Pull Request", type: "bash", command: "git clone" },
      { name: "Blast radius", type: "prompt", prompt: "Merge check: risk.\nScore the change.\n" },
    ];

    expect(expandMergeConfidenceSteps(steps)).toEqual(steps);
  });
});

describe("newMergeConfidenceStep", () => {
  it("adds a check the person can name, and keeps the id when the text changes", () => {
    const first = newMergeConfidenceStep([]);
    const second = newMergeConfidenceStep([first]);

    expect(first.name).toBe("New check");
    expect(first.type).toBe("prompt");
    expect(mergeCheckPromptBody(first.prompt)).toContain("Describe what this check looks for.");
    expect(mergeCheckPromptBody(first.prompt)).toContain("report_merge_check");
    expect(mergeCheckPromptBody(first.prompt)).toContain("check is new-check.");
    expect(mergeCheckPromptBody(first.prompt)).not.toContain("Merge check:");
    expect(second.prompt?.startsWith("Merge check: new-check-2.\n")).toBe(true);

    const edited = withMergeCheckHeader(first.prompt, "Look at the database migration.");
    expect(edited.startsWith("Merge check: new-check.\n")).toBe(true);
    expect(edited).toContain("Look at the database migration.");
  });
});
