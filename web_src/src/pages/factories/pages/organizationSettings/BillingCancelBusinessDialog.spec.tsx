import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { BillingCancelBusinessDialog } from "./BillingCancelBusinessDialog";

describe("BillingCancelBusinessDialog", () => {
  it("keeps the dialog open when cancel fails", async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();
    const onConfirm = vi.fn().mockRejectedValue(new Error("Unable to cancel Business."));

    render(
      <BillingCancelBusinessDialog
        open
        pending={false}
        currentPeriodEnd="2026-10-09T12:00:00.000Z"
        onOpenChange={onOpenChange}
        onConfirm={onConfirm}
      />,
    );

    await user.click(screen.getByTestId("billing-cancel-subscription-confirm"));
    expect(onConfirm).toHaveBeenCalledTimes(1);
    await waitFor(() => {
      expect(onOpenChange).not.toHaveBeenCalled();
    });
    expect(screen.getByTestId("billing-cancel-subscription-dialog")).toBeInTheDocument();
  });

  it("closes the dialog after cancel succeeds", async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();
    const onConfirm = vi.fn().mockResolvedValue(undefined);

    render(
      <BillingCancelBusinessDialog
        open
        pending={false}
        currentPeriodEnd="2026-10-09T12:00:00.000Z"
        onOpenChange={onOpenChange}
        onConfirm={onConfirm}
      />,
    );

    await user.click(screen.getByTestId("billing-cancel-subscription-confirm"));
    await waitFor(() => {
      expect(onOpenChange).toHaveBeenCalledWith(false);
    });
  });
});
