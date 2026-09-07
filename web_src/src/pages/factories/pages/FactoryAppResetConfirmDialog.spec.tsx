import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { FactoryAppResetConfirmDialog } from "./FactoryAppResetConfirmDialog";

describe("FactoryAppResetConfirmDialog", () => {
  it("confirms the reset and calls onConfirm", async () => {
    const user = userEvent.setup();
    const onConfirm = vi.fn();
    const onOpenChange = vi.fn();

    render(<FactoryAppResetConfirmDialog open onOpenChange={onOpenChange} onConfirm={onConfirm} />);

    await user.click(screen.getByTestId("factory-app-reset-defaults-confirm"));
    expect(onConfirm).toHaveBeenCalledTimes(1);
  });

  it("cancels without calling onConfirm", async () => {
    const user = userEvent.setup();
    const onConfirm = vi.fn();
    const onOpenChange = vi.fn();

    render(<FactoryAppResetConfirmDialog open onOpenChange={onOpenChange} onConfirm={onConfirm} />);

    await user.click(screen.getByTestId("factory-app-reset-defaults-cancel"));
    expect(onConfirm).not.toHaveBeenCalled();
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it("renders nothing when closed", () => {
    render(<FactoryAppResetConfirmDialog open={false} onOpenChange={vi.fn()} onConfirm={vi.fn()} />);
    expect(screen.queryByTestId("factory-app-reset-defaults-dialog")).not.toBeInTheDocument();
  });
});
