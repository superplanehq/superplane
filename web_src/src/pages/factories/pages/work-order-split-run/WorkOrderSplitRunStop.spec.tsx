import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
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
    handleStopAutomation: vi.fn(),
    busy: false,
  }),
}));

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";

import { DRAFT_WORK_ORDER, OPEN_WORK_ORDER } from "../../__fixtures__/factoryPageResponses";
import { BOARD_IMPLEMENT_FAILED_ORDER } from "../../__fixtures__/lineMetricsBoardOrders";
import { WorkOrderSplitRunPopup } from "./WorkOrderSplitRunPopup";
import { SPLIT_RUN_RUNNING, splitRunFixtureForWorkOrder } from "./splitRunMocks";

function renderPopup(fixture: ComponentProps<typeof WorkOrderSplitRunPopup>["fixture"], onClose?: () => void) {
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <ThemeProvider>
          <TooltipProvider>
            <WorkOrderSplitRunPopup fixture={fixture} onClose={onClose} />
          </TooltipProvider>
        </ThemeProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

function headerActions() {
  return screen.getByTestId("split-run-header-actions");
}

describe("WorkOrderSplitRunPopup header actions", () => {
  beforeEach(() => {
    handleStopMock.mockReset();
    handleRejectMock.mockReset();
  });

  it("asks before Reject closes a running work order", async () => {
    const user = userEvent.setup();
    renderPopup(SPLIT_RUN_RUNNING);

    expect(screen.queryByRole("button", { name: "Stop and Close" })).not.toBeInTheDocument();
    await user.click(within(headerActions()).getByRole("button", { name: "Reject" }));
    expect(handleStopMock).not.toHaveBeenCalled();

    const dialog = await screen.findByRole("alertdialog");
    expect(within(dialog).getByRole("heading", { name: "Stop running automations?" })).toBeInTheDocument();
    expect(
      within(dialog).getByText(/This action stops all running automations on this work order/),
    ).toBeInTheDocument();
    await user.click(within(dialog).getByRole("button", { name: "Reject" }));
    expect(handleStopMock).toHaveBeenCalledWith("canceled", expect.objectContaining({ kind: "running" }));
  });

  it("asks before Approve closes a running work order", async () => {
    const user = userEvent.setup();
    renderPopup(SPLIT_RUN_RUNNING);

    await user.click(within(headerActions()).getByRole("button", { name: "Approve" }));
    expect(handleStopMock).not.toHaveBeenCalled();

    const dialog = await screen.findByRole("alertdialog");
    await user.click(within(dialog).getByRole("button", { name: "Approve" }));
    expect(handleStopMock).toHaveBeenCalledWith("completed", expect.objectContaining({ kind: "running" }));
  });

  it("does not close when Cancel is used on the running confirm", async () => {
    const user = userEvent.setup();
    renderPopup(SPLIT_RUN_RUNNING);

    await user.click(within(headerActions()).getByRole("button", { name: "Reject" }));
    await user.click(within(await screen.findByRole("alertdialog")).getByRole("button", { name: "Cancel" }));
    expect(handleStopMock).not.toHaveBeenCalled();
    expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument();
  });

  it("rejects a waiting work order without a confirm", async () => {
    const user = userEvent.setup();
    renderPopup(splitRunFixtureForWorkOrder(OPEN_WORK_ORDER));

    await user.click(within(headerActions()).getByRole("button", { name: "Reject" }));
    expect(handleStopMock).toHaveBeenCalledWith("canceled", expect.objectContaining({ kind: "waiting" }));
    expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument();
  });

  it("approves a waiting work order without a confirm", async () => {
    const user = userEvent.setup();
    renderPopup(splitRunFixtureForWorkOrder(OPEN_WORK_ORDER));

    await user.click(within(headerActions()).getByRole("button", { name: "Approve" }));
    expect(handleStopMock).toHaveBeenCalledWith("completed", expect.objectContaining({ kind: "waiting" }));
    expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument();
  });

  it("closes the popup after Reject deletes a draft", async () => {
    handleRejectMock.mockResolvedValue(true);
    const user = userEvent.setup();
    const onClose = vi.fn();
    renderPopup(splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER), onClose);

    await user.click(within(headerActions()).getByRole("button", { name: "Reject" }));
    expect(handleRejectMock).toHaveBeenCalledTimes(1);
    await waitFor(() => {
      expect(onClose).toHaveBeenCalledTimes(1);
    });
  });

  it("reopens a closed work order from Reopen", async () => {
    const user = userEvent.setup();
    renderPopup(splitRunFixtureForWorkOrder(BOARD_IMPLEMENT_FAILED_ORDER));

    expect(within(headerActions()).queryByRole("button", { name: "Reject" })).not.toBeInTheDocument();
    await user.click(within(headerActions()).getByRole("button", { name: "Reopen" }));
    expect(handleStopMock).toHaveBeenCalledWith(
      "reopen",
      expect.objectContaining({ kind: "failed", status: "failed" }),
    );
  });
});
