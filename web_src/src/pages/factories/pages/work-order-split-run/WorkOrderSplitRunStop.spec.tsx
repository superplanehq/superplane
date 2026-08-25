import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ComponentProps } from "react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

const { handleStopMock, handleRejectMock } = vi.hoisted(() => ({
  handleStopMock: vi.fn(),
  handleRejectMock: vi.fn(),
}));

vi.mock("./useSplitRunFooterActions", () => ({
  useSplitRunFooterActions: () => ({
    handleStop: handleStopMock,
    handleReject: handleRejectMock,
    busy: false,
  }),
}));

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";

import { DRAFT_WORK_ORDER } from "../../__fixtures__/factoryPageResponses";
import { BOARD_IMPLEMENT_FAILED_ORDER } from "../../__fixtures__/lineMetricsBoardOrders";
import { WorkOrderSplitRunPopup } from "./WorkOrderSplitRunPopup";
import { SPLIT_RUN_RUNNING, splitRunFixtureForWorkOrder } from "./splitRunMocks";

function renderPopup(fixture: ComponentProps<typeof WorkOrderSplitRunPopup>["fixture"]) {
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <ThemeProvider>
          <TooltipProvider>
            <WorkOrderSplitRunPopup fixture={fixture} />
          </TooltipProvider>
        </ThemeProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("WorkOrderSplitRunPopup stop actions", () => {
  beforeEach(() => {
    handleStopMock.mockReset();
    handleRejectMock.mockReset();
  });

  it("closes as rejected when Stop & Cancel is used", async () => {
    const user = userEvent.setup();
    renderPopup(SPLIT_RUN_RUNNING);

    await user.click(screen.getByRole("button", { name: "Stop & Cancel" }));
    expect(handleStopMock).toHaveBeenCalledWith("canceled", expect.objectContaining({ kind: "running" }));
  });

  it("closes as completed after that Stop outcome is chosen", async () => {
    const user = userEvent.setup();
    renderPopup(SPLIT_RUN_RUNNING);

    await user.click(screen.getByRole("button", { name: "Choose how to stop" }));
    await user.click(await screen.findByRole("menuitem", { name: /Stop as Completed/ }));
    await user.click(screen.getByRole("button", { name: "Mark as Complete" }));
    expect(handleStopMock).toHaveBeenCalledWith("completed", expect.objectContaining({ kind: "running" }));
  });

  it("returns to draft after that Stop outcome is chosen", async () => {
    const user = userEvent.setup();
    renderPopup(SPLIT_RUN_RUNNING);

    await user.click(screen.getByRole("button", { name: "Choose how to stop" }));
    await user.click(await screen.findByRole("menuitem", { name: /Stop and return to Draft/ }));
    await user.click(screen.getByRole("button", { name: "Return to Draft" }));
    expect(handleStopMock).toHaveBeenCalledWith("draft", expect.objectContaining({ kind: "running" }));
  });

  it("closes a draft as canceled from Reject", async () => {
    const user = userEvent.setup();
    renderPopup(splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER));

    await user.click(within(screen.getByTestId("split-run-review")).getByRole("button", { name: "Reject" }));
    expect(handleRejectMock).toHaveBeenCalledTimes(1);
  });

  it("reopens a closed work order from Reopen", async () => {
    const user = userEvent.setup();
    renderPopup(splitRunFixtureForWorkOrder(BOARD_IMPLEMENT_FAILED_ORDER));

    expect(screen.queryByRole("button", { name: "Return to Draft" })).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Reopen" }));
    expect(handleStopMock).toHaveBeenCalledWith(
      "reopen",
      expect.objectContaining({ kind: "failed", status: "failed" }),
    );
  });

  it("hides Return to Draft when the work order is already a draft", async () => {
    const user = userEvent.setup();
    renderPopup({
      ...SPLIT_RUN_RUNNING,
      footer: { ...SPLIT_RUN_RUNNING.footer, status: "draft" },
    });

    await user.click(screen.getByRole("button", { name: "Choose how to stop" }));
    const menu = await screen.findByRole("menu");
    expect(within(menu).queryByRole("menuitem", { name: /Stop and return to Draft/ })).not.toBeInTheDocument();
    expect(within(menu).getByRole("menuitem", { name: /Stop as Canceled/ })).toBeInTheDocument();
    expect(within(menu).getByRole("menuitem", { name: /Stop as Completed/ })).toBeInTheDocument();
  });
});
