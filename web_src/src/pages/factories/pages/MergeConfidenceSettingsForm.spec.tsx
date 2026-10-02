import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { MergeConfidenceSettingsForm } from "./MergeConfidenceSettingsForm";
import { formatEnabledChecksLine, mergeConfidenceFromDraft } from "./mergeConfidenceChecks";
import type { PlanningReviewDraft } from "./planningReviewMockup";
import { defaultRiskScoreCategories, formatRiskScoreRules } from "./riskScoreCategories";

function draft(): PlanningReviewDraft {
  return {
    title: "Assess Merge Confidence",
    components: [
      {
        id: "assess-risk",
        title: "Assess Merge Confidence",
        description: "",
        expanded: true,
        configuration: {
          steps: [
            {
              name: "Review Pull Request",
              type: "prompt",
              prompt: [
                "Review the pull request diff.",
                formatEnabledChecksLine(["risk", "performance", "security", "drift"]),
                formatRiskScoreRules(defaultRiskScoreCategories()),
              ].join("\n"),
            },
          ],
        },
        concurrency: { max: "1", key: "" },
      },
    ],
  };
}

describe("MergeConfidenceSettingsForm", () => {
  it("turns a check off and saves that choice in the prompt", async () => {
    const user = userEvent.setup();
    const onSave = vi.fn();
    render(<MergeConfidenceSettingsForm draft={draft()} onSave={onSave} />);

    await user.click(screen.getByRole("switch", { name: "Drift from Specification" }));
    await user.click(screen.getByRole("switch", { name: "Blast radius" }));

    await user.click(screen.getByTestId("merge-confidence-settings-save"));

    expect(onSave).toHaveBeenCalledTimes(1);
    const saved = mergeConfidenceFromDraft(onSave.mock.calls[0][0] as PlanningReviewDraft);
    expect(saved?.checks).toEqual(["performance", "security"]);
    expect(saved?.categories.map((category) => category.name)).toContain("Authorization changes");
  });
});
