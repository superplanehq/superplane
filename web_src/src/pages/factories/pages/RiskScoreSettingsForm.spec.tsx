import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "bun:test";

import type { PlanningReviewDraft } from "./planningReviewMockup";
import { formatRiskScoreRules, riskScoreCategoriesFromDraft } from "./riskScoreCategories";
import { RiskScoreSettingsForm } from "./RiskScoreSettingsForm";

function riskScoreDraft(): PlanningReviewDraft {
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
            {
              name: "Review Pull Request",
              type: "prompt",
              prompt: formatRiskScoreRules([{ name: "Authorization changes", score: 4 }]),
            },
          ],
        },
        concurrency: { max: "1", key: "" },
      },
    ],
  };
}

function renderForm(props: Partial<Parameters<typeof RiskScoreSettingsForm>[0]> = {}) {
  const onSave = props.onSave ?? vi.fn();
  render(<RiskScoreSettingsForm draft={props.draft ?? riskScoreDraft()} onSave={onSave} {...props} />);
  return { onSave };
}

describe("RiskScoreSettingsForm", () => {
  it("puts delete and save on one footer", () => {
    renderForm({ onDelete: vi.fn() });

    const save = screen.getByTestId("risk-score-settings-save");
    const remove = screen.getByTestId("risk-score-settings-delete");
    expect(save.closest("footer")).toBe(remove.closest("footer"));
    expect(remove.compareDocumentPosition(save) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(screen.queryByTestId("column-automation-view-delete")).not.toBeInTheDocument();
  });

  it("asks for confirmation before deleting", async () => {
    const user = userEvent.setup();
    const onDelete = vi.fn();
    const view = render(<RiskScoreSettingsForm draft={riskScoreDraft()} onSave={vi.fn()} onDelete={onDelete} />);

    await user.click(screen.getByTestId("risk-score-settings-delete"));
    expect(screen.getByText("Delete this automation? This cannot be undone.")).toBeInTheDocument();
    expect(screen.queryByTestId("risk-score-settings-save")).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Keep automation" }));
    expect(screen.getByTestId("risk-score-settings-save")).toBeInTheDocument();
    expect(onDelete).not.toHaveBeenCalled();

    await user.click(screen.getByTestId("risk-score-settings-delete"));
    view.rerender(
      <RiskScoreSettingsForm draft={riskScoreDraft()} onSave={vi.fn()} onDelete={onDelete} deletePending />,
    );
    expect(screen.getByTestId("risk-score-settings-delete-confirm")).toBeDisabled();
    expect(screen.getByTestId("risk-score-settings-delete-confirm")).toHaveTextContent("Deleting");

    view.rerender(<RiskScoreSettingsForm draft={riskScoreDraft()} onSave={vi.fn()} onDelete={onDelete} />);
    await user.click(screen.getByTestId("risk-score-settings-delete-confirm"));
    expect(onDelete).toHaveBeenCalledTimes(1);
  });

  it("saves categories and stays disabled when none remain", async () => {
    const user = userEvent.setup();
    const onSave = vi.fn();
    renderForm({ onSave });

    expect(screen.getByTestId("risk-score-settings-save")).toBeEnabled();
    await user.click(screen.getByTestId("risk-score-settings-save"));
    expect(onSave).toHaveBeenCalledTimes(1);
    expect(riskScoreCategoriesFromDraft(onSave.mock.calls[0][0])).toEqual([
      { id: "authorization", name: "Authorization changes", score: 4 },
    ]);

    await user.click(screen.getByTestId("risk-score-category-remove-authorization"));
    expect(screen.getByTestId("risk-score-settings-save")).toBeDisabled();
  });

  it("keeps save on the right when delete is unavailable", () => {
    renderForm();

    const save = screen.getByTestId("risk-score-settings-save");
    const footer = save.closest("footer");
    expect(screen.queryByTestId("risk-score-settings-delete")).not.toBeInTheDocument();
    expect(footer).toHaveClass("justify-between");
    expect(footer?.firstElementChild?.tagName).toBe("SPAN");
    expect(footer?.lastElementChild).toBe(save);
  });
});
