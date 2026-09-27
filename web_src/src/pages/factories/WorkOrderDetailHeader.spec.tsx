import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "bun:test";

import { TooltipProvider } from "@/ui/tooltip";

import { WorkOrderDetailHeader } from "./WorkOrderDetailHeader";

const handlers = {
  onClose: vi.fn(),
  onStatusChange: vi.fn(async () => undefined),
  onDuplicate: vi.fn(),
};

function renderHeader(canCreate: boolean, flags: { isOpen: boolean; isDispatchable: boolean; isClosed: boolean }) {
  render(
    <TooltipProvider>
      <WorkOrderDetailHeader
        orderTitle="Retry refunds"
        displayStatus={flags.isClosed ? "completed" : flags.isOpen ? "waiting" : "draft"}
        canClose={flags.isOpen || flags.isDispatchable}
        canManage={flags.isClosed}
        isCompleting={false}
        isRejecting={false}
        isClosing={false}
        isUpdatingStatus={false}
        canCreate={canCreate}
        {...flags}
        {...handlers}
      />
    </TooltipProvider>,
  );
}

describe("WorkOrderDetailHeader duplicate", () => {
  it("includes Duplicate above status actions when create is allowed", async () => {
    const user = userEvent.setup();
    renderHeader(true, { isOpen: true, isDispatchable: true, isClosed: false });

    await user.click(screen.getByRole("button", { name: "More actions" }));

    const duplicate = screen.getByRole("menuitem", { name: "Duplicate" });
    const complete = screen.getByRole("menuitem", { name: "Complete" });
    expect(duplicate).toBeInTheDocument();
    expect(duplicate.compareDocumentPosition(complete) & Node.DOCUMENT_POSITION_FOLLOWING).toBeGreaterThan(0);
  });

  it("omits Duplicate when create is not allowed", async () => {
    const user = userEvent.setup();
    renderHeader(false, { isOpen: true, isDispatchable: true, isClosed: false });

    await user.click(screen.getByRole("button", { name: "More actions" }));

    expect(screen.queryByRole("menuitem", { name: "Duplicate" })).not.toBeInTheDocument();
    expect(screen.getByRole("menuitem", { name: "Complete" })).toBeInTheDocument();
  });

  it("shows the menu for Duplicate when no status action is present", async () => {
    const user = userEvent.setup();
    renderHeader(true, { isOpen: false, isDispatchable: false, isClosed: false });

    await user.click(screen.getByRole("button", { name: "More actions" }));

    expect(screen.getByRole("menuitem", { name: "Duplicate" })).toBeInTheDocument();
    expect(screen.queryByRole("menuitem", { name: "Complete" })).not.toBeInTheDocument();
  });
});
