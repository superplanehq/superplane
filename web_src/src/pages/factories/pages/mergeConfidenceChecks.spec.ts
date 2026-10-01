import { describe, expect, it } from "bun:test";

import type { PlanningReviewDraft } from "./planningReviewMockup";
import {
  defaultMergeConfidenceChecks,
  draftWithMergeConfidence,
  formatEnabledChecksLine,
  formatEnabledChecksValue,
  mergeConfidenceFromDraft,
  parseEnabledChecks,
  type MergeConfidenceCheck,
} from "./mergeConfidenceChecks";
import { defaultRiskScoreCategories, formatRiskScoreRules } from "./riskScoreCategories";

function promptWithSettings(checks: readonly MergeConfidenceCheck[], rules: string): string {
  return ["Review the pull request diff.", "", formatEnabledChecksLine(checks), "", rules, ""].join("\n");
}

function draftWithPrompt(prompt: string): PlanningReviewDraft {
  return {
    title: "Assess Merge Confidence",
    components: [
      {
        id: "assess-risk",
        title: "Assess Merge Confidence",
        description: "",
        expanded: true,
        configuration: {
          steps: [{ name: "Review Pull Request", type: "prompt", prompt }],
        },
        concurrency: { max: "1", key: "" },
      },
    ],
  };
}

describe("parseEnabledChecks", () => {
  it("reads the four checks in canonical order", () => {
    const prompt = promptWithSettings(
      defaultMergeConfidenceChecks(),
      formatRiskScoreRules(defaultRiskScoreCategories()),
    );

    expect(parseEnabledChecks(prompt)).toEqual(["risk", "performance", "security", "drift"]);
    expect(formatEnabledChecksValue(["drift", "risk"])).toBe("risk, drift");
  });

  it("reads none as no checks", () => {
    expect(parseEnabledChecks("Enabled checks: none.\n")).toEqual([]);
  });

  it("returns null when the line is missing", () => {
    expect(parseEnabledChecks("Review the pull request diff.")).toBeNull();
  });
});

describe("draftWithMergeConfidence", () => {
  it("rewrites the checks line and keeps the risk rules", () => {
    const draft = draftWithPrompt(
      [
        "Review the pull request diff.",
        formatEnabledChecksLine(defaultMergeConfidenceChecks()),
        formatRiskScoreRules(defaultRiskScoreCategories()),
      ].join("\n"),
    );

    const next = draftWithMergeConfidence(draft, {
      checks: ["security", "drift"],
      categories: [{ id: "authorization", name: "Authorization changes", score: 5 }],
    });

    expect(mergeConfidenceFromDraft(next ?? undefined)).toEqual({
      checks: ["security", "drift"],
      categories: [{ id: "authorization", name: "Authorization changes", score: 5 }],
    });
    const prompt = (next?.components[0].configuration.steps as Array<{ prompt: string }>)[0].prompt;
    expect(prompt).toContain("Review the pull request diff.");
    expect(prompt).not.toContain("Documentation only");
  });
});
