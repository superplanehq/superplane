import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { AddPRFeedbackPicker } from "./AddPRFeedbackPicker";

function renderPicker(onSelect = vi.fn(), onClose = vi.fn()) {
  render(<AddPRFeedbackPicker open onClose={onClose} onSelect={onSelect} />);
  return { onSelect, onClose };
}

describe("AddPRFeedbackPicker", () => {
  it("offers discussion and check cards", () => {
    renderPicker();

    const picker = screen.getByTestId("add-pr-feedback-picker");
    expect(within(picker).getByRole("heading", { name: "Add feedback handler" })).toBeInTheDocument();
    expect(within(picker).getByTestId("add-pr-feedback-template-discussion")).toHaveTextContent(
      "Pull request discussion",
    );
    expect(within(picker).getByTestId("add-pr-feedback-template-checks")).toHaveTextContent("Pull request checks");
  });

  it("reports the chosen source to the caller", async () => {
    const user = userEvent.setup();
    const { onSelect } = renderPicker();

    await user.click(screen.getByTestId("add-pr-feedback-template-checks"));

    expect(onSelect).toHaveBeenCalledWith(expect.objectContaining({ id: "checks" }));
  });

  it("does not offer a source that already has a handler", async () => {
    const user = userEvent.setup();
    const onSelect = vi.fn();
    render(<AddPRFeedbackPicker open onClose={vi.fn()} onSelect={onSelect} takenSourceIds={["discussion"]} />);

    const discussion = screen.getByTestId("add-pr-feedback-template-discussion");
    expect(discussion).toBeDisabled();
    expect(discussion).toHaveTextContent("A handler for this source already exists.");
    await user.click(discussion);
    expect(onSelect).not.toHaveBeenCalled();

    await user.click(screen.getByTestId("add-pr-feedback-template-checks"));
    expect(onSelect).toHaveBeenCalledWith(expect.objectContaining({ id: "checks" }));
  });
});
