import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { TooltipProvider } from "@/components/ui/tooltip";

const forkMutate = vi.fn();

vi.mock("@/hooks/useFactoryData", () => ({
  useForkWorkOrder: () => ({ mutateAsync: forkMutate, isPending: false }),
}));

import { WorkOrderDetailHeader } from "./WorkOrderDetailHeader";
import { PopupHeaderActions } from "./pages/work-order-split-run/PopupHeaderActions";

const forkTarget = {
  organizationId: "org-1",
  factoryId: "factory-1",
  factoryKey: "SP",
  orderId: "order-1",
  hasPlan: true,
  canFork: true,
};

function renderHeader(hasPlan: boolean, canFork = true) {
  return render(
    <MemoryRouter>
      <TooltipProvider>
        <WorkOrderDetailHeader
          orderTitle="Retry refunds"
          displayStatus="running"
          isOpen
          isDispatchable
          isClosed={false}
          canClose
          canManage={canFork}
          isCompleting={false}
          isRejecting={false}
          isClosing={false}
          isUpdatingStatus={false}
          onClose={() => undefined}
          onStatusChange={async () => undefined}
          fork={{ ...forkTarget, hasPlan, canFork }}
        />
      </TooltipProvider>
    </MemoryRouter>,
  );
}

describe("fork task menu", () => {
  beforeEach(() => {
    forkMutate.mockReset();
    forkMutate.mockResolvedValue({ id: "order-2", number: 8 });
  });

  it("opens both choices from the task menu", async () => {
    const user = userEvent.setup();
    renderHeader(true);

    await user.click(screen.getByRole("button", { name: "More actions" }));
    await user.click(await screen.findByRole("menuitem", { name: "Fork task" }));

    const dialog = await screen.findByRole("dialog", { name: "Fork task" });
    expect(within(dialog).getByText("The source task stays as it is.")).toBeInTheDocument();
    expect(within(dialog).getByRole("radio", { name: "Copy request" })).toBeEnabled();
    expect(within(dialog).getByText("Creates a new draft. Planning starts if it is on.")).toBeInTheDocument();
    expect(within(dialog).getByRole("radio", { name: "Copy plan" })).toBeEnabled();
    expect(within(dialog).getByText("Creates a draft with the plan. Start it when ready.")).toBeInTheDocument();

    await user.click(within(dialog).getByRole("radio", { name: "Copy plan" }));
    await user.click(within(dialog).getByRole("button", { name: "Fork task" }));
    expect(forkMutate).toHaveBeenCalledWith({ orderId: "order-1", mode: "MODE_PLAN" });
  });

  it("moves between choices with arrow keys and keeps one tab stop", async () => {
    const user = userEvent.setup();
    renderHeader(true);

    await user.click(screen.getByRole("button", { name: "More actions" }));
    await user.click(await screen.findByRole("menuitem", { name: "Fork task" }));

    const dialog = await screen.findByRole("dialog", { name: "Fork task" });
    const request = within(dialog).getByRole("radio", { name: "Copy request" });
    const plan = within(dialog).getByRole("radio", { name: "Copy plan" });
    expect(request).toHaveAttribute("tabindex", "0");
    expect(plan).toHaveAttribute("tabindex", "-1");

    request.focus();
    await user.keyboard("{ArrowDown}");
    expect(plan).toHaveFocus();
    expect(plan).toHaveAttribute("aria-checked", "true");
    expect(plan).toHaveAttribute("tabindex", "0");
    expect(request).toHaveAttribute("tabindex", "-1");

    await user.tab();
    expect(within(dialog).getByRole("button", { name: "Cancel" })).toHaveFocus();

    await user.click(within(dialog).getByRole("button", { name: "Fork task" }));
    expect(forkMutate).toHaveBeenCalledWith({ orderId: "order-1", mode: "MODE_PLAN" });
  });

  it("does not move to a disabled plan choice", async () => {
    const user = userEvent.setup();
    renderHeader(false);

    await user.click(screen.getByRole("button", { name: "More actions" }));
    await user.click(await screen.findByRole("menuitem", { name: "Fork task" }));

    const dialog = await screen.findByRole("dialog", { name: "Fork task" });
    const request = within(dialog).getByRole("radio", { name: "Copy request" });
    request.focus();
    await user.keyboard("{ArrowDown}");
    expect(request).toHaveFocus();
    expect(request).toHaveAttribute("aria-checked", "true");
  });

  it("disables plan copy when the task has no plan", async () => {
    const user = userEvent.setup();
    renderHeader(false);

    await user.click(screen.getByRole("button", { name: "More actions" }));
    await user.click(await screen.findByRole("menuitem", { name: "Fork task" }));

    const dialog = await screen.findByRole("dialog", { name: "Fork task" });
    expect(within(dialog).getByRole("radio", { name: "Copy plan" })).toBeDisabled();
    expect(within(dialog).getByText("This task has no plan.")).toBeInTheDocument();
  });

  it("opens the same choices from the popup header", async () => {
    const user = userEvent.setup();
    render(
      <MemoryRouter>
        <TooltipProvider>
          <PopupHeaderActions fork={{ ...forkTarget, canFork: false }} />
        </TooltipProvider>
      </MemoryRouter>,
    );

    await user.click(screen.getByRole("button", { name: "Task actions" }));
    await user.click(await screen.findByRole("menuitem", { name: "Fork task" }));

    const dialog = await screen.findByRole("dialog", { name: "Fork task" });
    expect(within(dialog).getByRole("radio", { name: "Copy request" })).toBeDisabled();
    expect(within(dialog).getByRole("radio", { name: "Copy plan" })).toBeDisabled();
    expect(within(dialog).getByText("You do not have permission to fork this task.")).toBeInTheDocument();
  });
});
