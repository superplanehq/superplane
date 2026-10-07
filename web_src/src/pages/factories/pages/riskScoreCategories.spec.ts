import { describe, expect, it } from "bun:test";

import type { PlanningReviewDraft } from "./planningReviewMockup";
import {
  defaultRiskScoreCategories,
  draftWithRiskScoreCategories,
  formatRiskScoreRules,
  isRiskScoreCategoryName,
  parseRiskScoreRules,
  riskScoreCategoriesFromDraft,
} from "./riskScoreCategories";

function promptWithRules(rules: string): string {
  return [
    "Review the pull request diff.",
    "If several categories match, use the highest score.",
    "",
    rules,
    "",
    "Write the JSON to /tmp/risk.json.",
  ].join("\n");
}

function draftWithPrompt(prompt: string): PlanningReviewDraft {
  return {
    title: "Assess Risk",
    components: [
      {
        id: "assess-risk",
        title: "Assess Risk",
        description: "",
        expanded: true,
        configuration: {
          steps: [
            { name: "Checkout Pull Request", type: "bash", command: "git clone" },
            { name: "Review Pull Request", type: "prompt", prompt },
          ],
        },
        concurrency: { max: "1", key: "" },
      },
    ],
  };
}

describe("formatRiskScoreRules", () => {
  it("uses caution for additive database changes and critical for authorization changes", () => {
    const rules = formatRiskScoreRules(defaultRiskScoreCategories());

    expect(rules).toContain("Additive database changes = 2 (caution).");
    expect(rules).toContain("Authorization changes = 3 (critical).");
    expect(rules).toContain("Documentation only = 1 (healthy).");
    expect(rules).toContain("Secrets and credentials = 3 (critical).");
  });

  it("uses the current list, including added and removed categories", () => {
    const categories = defaultRiskScoreCategories()
      .filter((category) => category.id !== "documentation")
      .map((category) => (category.id === "authorization" ? { ...category, score: 1 as const } : category));
    categories.push({ id: "custom-1", name: "Cache changes", score: 2 });

    const rules = formatRiskScoreRules(categories);

    expect(rules).not.toContain("Documentation only");
    expect(rules).toContain("Authorization changes = 1 (healthy).");
    expect(rules).toContain("Cache changes = 2 (caution).");
  });

  it("skips a name that contains an equals sign", () => {
    expect(isRiskScoreCategoryName("Connection = pool")).toBe(false);
    expect(formatRiskScoreRules([{ name: "Connection = pool", score: 3 }])).toBe("");
  });
});

describe("parseRiskScoreRules", () => {
  it("reads the default categories back from the prompt", () => {
    const prompt = promptWithRules(formatRiskScoreRules(defaultRiskScoreCategories()));

    expect(parseRiskScoreRules(prompt)).toEqual(defaultRiskScoreCategories());
  });

  it("folds an old rules line and keeps a new score", () => {
    const prompt = promptWithRules(
      "Cache changes = 2 (low). Queue changes = 4 (high). Billing changes = 3 (critical). API changes = 2 (caution).",
    );

    expect(parseRiskScoreRules(prompt)).toEqual([
      { id: "custom-1", name: "Cache changes", score: 1 },
      { id: "custom-2", name: "Queue changes", score: 3 },
      { id: "custom-3", name: "Billing changes", score: 3 },
      { id: "custom-4", name: "API changes", score: 2 },
    ]);
  });

  it("returns null when the prompt has no rules line", () => {
    expect(parseRiskScoreRules("Review the pull request diff.")).toBeNull();
  });
});

describe("draftWithRiskScoreCategories", () => {
  it("rewrites only the rules line in the review prompt", () => {
    const draft = draftWithPrompt(promptWithRules(formatRiskScoreRules(defaultRiskScoreCategories())));
    const next = draftWithRiskScoreCategories(draft, [
      { id: "authorization", name: "Authorization changes", score: 3 },
      { id: "custom-1", name: "Cache changes", score: 2 },
    ]);

    expect(next).not.toBeNull();
    expect(riskScoreCategoriesFromDraft(next!)).toEqual([
      { id: "authorization", name: "Authorization changes", score: 3 },
      { id: "custom-1", name: "Cache changes", score: 2 },
    ]);
    const steps = next!.components[0].configuration.steps as Array<{ prompt?: string; command?: string }>;
    expect(steps[0].command).toBe("git clone");
    expect(steps[1].prompt).toContain("Write the JSON to /tmp/risk.json.");
    expect(steps[1].prompt).not.toContain("Documentation only");
  });

  it("returns null when no step has a rules line", () => {
    expect(draftWithRiskScoreCategories(draftWithPrompt("Review the diff."), defaultRiskScoreCategories())).toBeNull();
  });
});
