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

  it("rewrites stored 1 to 5 instructions when the categories are saved", () => {
    const draft = draftWithPrompt(
      [
        "score is an integer from 1 to 5.",
        "choose a score from 1 to 5.",
        "Use Higher risk for high or critical. Use Moderate risk for medium. Use Lower risk for low or very low.",
        "Documentation only = 1 (very_low). Secrets and credentials = 5 (critical).",
      ].join("\n"),
    );
    draft.components[0].configuration.steps.push({
      name: "Performance",
      type: "prompt",
      prompt: [
        "score is an integer from 1 to 5.",
        "5 means the change follows every practice that applies.",
        "4 means the change follows the practices that apply, with a small gap.",
        "3 means the change follows some practices and breaks one that applies.",
        "2 means the change breaks a practice that applies.",
        "1 means the change breaks more than one practice that applies.",
        "If no performance practice applies, report the check with 5.",
      ].join("\n"),
    });

    const next = draftWithRiskScoreCategories(draft, [{ id: "secrets", name: "Secrets and credentials", score: 3 }]);
    const steps = next!.components[0].configuration.steps as Array<{ prompt?: string }>;

    expect(steps[1].prompt).toContain("score is an integer from 1 to 3.");
    expect(steps[1].prompt).toContain("choose a score from 1 to 3.");
    expect(steps[1].prompt).toContain("Use Higher risk for critical. Use Moderate risk for caution. Use Lower risk for healthy.");
    expect(steps[1].prompt).not.toContain("1 to 5");
    expect(steps[2].prompt).toContain("If no performance practice applies, report 3.");
    expect(steps[2].prompt).not.toContain("report the check with 5");
    expect(steps[2].prompt).not.toContain("5 means");
  });

  it("leaves a custom 5-point rubric on its original scale", () => {
    const draft = draftWithPrompt(
      ["score is an integer from 1 to 5.", "5 means the owner asked for a manual score.", "Cache changes = 4 (high)."].join(
        "\n",
      ),
    );

    const next = draftWithRiskScoreCategories(draft, [{ id: "custom-1", name: "Cache changes", score: 2 }]);
    const steps = next!.components[0].configuration.steps as Array<{ prompt?: string }>;

    expect(steps[1].prompt).toContain("score is an integer from 1 to 5.");
    expect(steps[1].prompt).toContain("5 means the owner asked for a manual score.");
    expect(steps[1].prompt).toContain("Cache changes = 2 (caution).");
  });
});
