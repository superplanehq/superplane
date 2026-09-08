import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";

import { TooltipProvider } from "@/ui/tooltip";

import { WorkOrderDetailHeader } from "./WorkOrderDetailHeader";

function renderHeader(overrides: Partial<Parameters<typeof WorkOrderDetailHeader>[0]> = {}) {
  const onDuplicate = vi.fn();
  render(
    <MemoryRouter>
      <TooltipProvider>
        <WorkOrderDetailHeader
          orderTitle="Retry refunds"
          orderIdentifier="SP-42"
          displayStatus="waiting"
          isOpen
          isDispatchable
          isClosed={false}
          canClose
          canManage
          canCreate
          isCompleting={false}
          isRejecting={false}
          isClosing={false}
          isUpdatingStatus={false}
          onClose={vi.fn()}
          onStatusChange={vi.fn()}
          onDuplicate={onDuplicate}
          {...overrides}
        />
      </TooltipProvider>
    </MemoryRouter>,
  );
  return { onDuplicate };
}

describe("WorkOrderDetailHeader", () => {
  it("shows Duplicate in the overflow menu and calls onDuplicate", async () => {
    const user = userEvent.setup();
    const { onDuplicate } = renderHeader();

    await user.click(screen.getByTestId("work-order-actions-button"));
    await user.click(screen.getByTestId("work-order-duplicate-button"));

    expect(onDuplicate).toHaveBeenCalledTimes(1);
  });

  it("disables Duplicate without create permission", async () => {
    const user = userEvent.setup();
    const { onDuplicate } = renderHeader({ canCreate: false });

    await user.click(screen.getByTestId("work-order-actions-button"));
    expect(screen.getByTestId("work-order-duplicate-button")).toHaveAttribute("data-disabled");
    await user.click(screen.getByTestId("work-order-duplicate-button"));
    expect(onDuplicate).not.toHaveBeenCalled();
  });
});
