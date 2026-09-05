import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { ArchiveStepDialog } from "./ArchiveStepDialog";

describe("ArchiveStepDialog", () => {
  it("blocks archive when the column still has tasks", async () => {
    const user = userEvent.setup();
    const onConfirm = vi.fn();
    const onOpenChange = vi.fn();

    render(<ArchiveStepDialog open stepName="Implement" hasTasks onOpenChange={onOpenChange} onConfirm={onConfirm} />);

    expect(screen.getByRole("heading", { name: "Archive this step" })).toBeInTheDocument();
    expect(screen.getByText("Make sure that the column does not have any tasks in it.")).toBeInTheDocument();
    expect(screen.queryByTestId("lines-archive-step-confirm")).not.toBeInTheDocument();

    await user.click(screen.getByTestId("lines-archive-step-close"));
    expect(onConfirm).not.toHaveBeenCalled();
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it("asks for confirmation when the column is empty", async () => {
    const user = userEvent.setup();
    const onConfirm = vi.fn();
    const onOpenChange = vi.fn();

    render(
      <ArchiveStepDialog
        open
        stepName="Implement"
        hasTasks={false}
        onOpenChange={onOpenChange}
        onConfirm={onConfirm}
      />,
    );

    expect(screen.getByRole("heading", { name: "Archive this step?" })).toBeInTheDocument();
    expect(screen.getByText("This archives Implement and the automation.")).toBeInTheDocument();

    await user.click(screen.getByTestId("lines-archive-step-confirm"));
    expect(onConfirm).toHaveBeenCalledTimes(1);
  });

  it("blocks archive when the line has only one step", async () => {
    const user = userEvent.setup();
    const onConfirm = vi.fn();
    const onOpenChange = vi.fn();

    render(
      <ArchiveStepDialog
        open
        stepName="Implement"
        hasTasks={false}
        isLastStep
        onOpenChange={onOpenChange}
        onConfirm={onConfirm}
      />,
    );

    expect(screen.getByRole("heading", { name: "Archive this step" })).toBeInTheDocument();
    expect(screen.getByText("A line must have at least one step.")).toBeInTheDocument();
    expect(screen.queryByTestId("lines-archive-step-confirm")).not.toBeInTheDocument();

    await user.click(screen.getByTestId("lines-archive-step-close"));
    expect(onConfirm).not.toHaveBeenCalled();
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it("cancels without archiving", async () => {
    const user = userEvent.setup();
    const onConfirm = vi.fn();
    const onOpenChange = vi.fn();

    render(
      <ArchiveStepDialog
        open
        stepName="Implement"
        hasTasks={false}
        onOpenChange={onOpenChange}
        onConfirm={onConfirm}
      />,
    );

    await user.click(screen.getByTestId("lines-archive-step-cancel"));
    expect(onConfirm).not.toHaveBeenCalled();
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it("renders nothing when closed", () => {
    render(
      <ArchiveStepDialog
        open={false}
        stepName="Implement"
        hasTasks={false}
        onOpenChange={vi.fn()}
        onConfirm={vi.fn()}
      />,
    );
    expect(screen.queryByTestId("lines-archive-step-dialog")).not.toBeInTheDocument();
  });
});
