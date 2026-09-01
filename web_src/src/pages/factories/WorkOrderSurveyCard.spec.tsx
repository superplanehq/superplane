import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { WORK_ORDER_SURVEY_SUBMIT, WORK_ORDER_SURVEY_TITLE, WorkOrderSurveyCard } from "./WorkOrderSurveyCard";

describe("WorkOrderSurveyCard", () => {
  it("submits selected answers and skipped questions", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    render(
      <WorkOrderSurveyCard
        survey={{
          id: "s-1",
          status: "pending",
          questions: [
            { id: "scope", prompt: "Where?", options: ["A", "B"], allowFreeText: false },
            { id: "notes", prompt: "Notes?", options: [], allowFreeText: true },
          ],
        }}
        onSubmit={onSubmit}
      />,
    );

    expect(screen.getByLabelText(WORK_ORDER_SURVEY_TITLE)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "A" }));
    await user.click(screen.getByRole("button", { name: "Next" }));
    await user.click(screen.getByRole("button", { name: WORK_ORDER_SURVEY_SUBMIT }));

    expect(onSubmit).toHaveBeenCalledWith([
      { id: "scope", value: "A" },
      { id: "notes", value: "skipped" },
    ]);
  });
});
