import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { FeedbackDialog } from "@/components/FeedbackDialog";

const { submitFeedback, showSuccessToast, showErrorToast } = vi.hoisted(() => ({
  submitFeedback: vi.fn(),
  showSuccessToast: vi.fn(),
  showErrorToast: vi.fn(),
}));

vi.mock("@/lib/submitFeedback", async () => {
  const actual = await vi.importActual("@/lib/submitFeedback");
  return {
    ...actual,
    submitFeedback,
  };
});

vi.mock("@/lib/toast", () => ({
  showSuccessToast,
  showErrorToast,
}));

function renderDialog(initialCategory?: "bug" | "feature" | "other") {
  return render(
    <MemoryRouter initialEntries={["/acme/apps/deploy"]}>
      <FeedbackDialog open onOpenChange={vi.fn()} organizationId="org-1" initialCategory={initialCategory} />
    </MemoryRouter>,
  );
}

describe("FeedbackDialog", () => {
  beforeEach(() => {
    submitFeedback.mockReset();
    showSuccessToast.mockReset();
    showErrorToast.mockReset();
    submitFeedback.mockResolvedValue(undefined);
  });

  it("shows the three feedback categories", () => {
    renderDialog();
    expect(screen.getByRole("button", { name: /Report a bug/ })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Request a feature/ })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Something else/ })).toBeInTheDocument();
  });

  it("sends the selected category and details", async () => {
    const user = userEvent.setup();
    renderDialog("bug");

    await user.type(screen.getByTestId("feedback-details"), "The canvas did not load.");
    await user.click(screen.getByTestId("feedback-send"));

    expect(submitFeedback).toHaveBeenCalledWith({
      organizationId: "org-1",
      category: "bug",
      details: "The canvas did not load.",
      pagePath: "/acme/apps/deploy",
      file: undefined,
    });
    expect(showSuccessToast).toHaveBeenCalledWith("Feedback sent.");
  });
});
