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
    handleStopAutomation: vi.fn(),
    busy: false,
  }),
}));

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";

import { DRAFT_WORK_ORDER, FAILED_WORK_ORDER, OPEN_WORK_ORDER } from "../../__fixtures__/factoryPageResponses";
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

describe("WorkOrderSplitRunPopup decision footer", () => {
  beforeEach(() => {
    handleStopMock.mockReset();
    handleRejectMock.mockReset();
  });

  it("keeps Reject and Approve off a running work order", () => {
    renderPopup(SPLIT_RUN_RUNNING);

    expect(screen.queryByRole("button", { name: "Reject" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Approve" })).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-header-actions")).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-review")).not.toBeInTheDocument();
  });

  it("rejects and approves a waiting work order from the note", async () => {
    const user = userEvent.setup();
    renderPopup(splitRunFixtureForWorkOrder(OPEN_WORK_ORDER));

    const note = screen.getByTestId("split-run-attention-note");
    expect(screen.queryByTestId("split-run-header-actions")).not.toBeInTheDocument();
    await user.click(within(note).getByRole("button", { name: "Reject" }));
    expect(handleRejectMock).toHaveBeenCalledTimes(1);
    await user.click(within(note).getByRole("button", { name: "Approve" }));
    expect(handleStopMock).toHaveBeenCalledWith(
      "completed",
      expect.objectContaining({ kind: "waiting", status: "waiting" }),
    );
  });

  it("starts and rejects a draft from the note", async () => {
    const user = userEvent.setup();
    const onDispatch = vi.fn();
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <MemoryRouter>
          <ThemeProvider>
            <TooltipProvider>
              <WorkOrderSplitRunPopup
                fixture={splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER)}
                onDispatch={onDispatch}
                canDispatch
              />
            </TooltipProvider>
          </ThemeProvider>
        </MemoryRouter>
      </QueryClientProvider>,
    );

    const note = screen.getByTestId("split-run-attention-note");
    expect(screen.queryByTestId("split-run-header-actions")).not.toBeInTheDocument();
    await user.click(within(note).getByRole("button", { name: "Start" }));
    expect(onDispatch).toHaveBeenCalledTimes(1);
    await user.click(within(note).getByRole("button", { name: "Reject" }));
    expect(handleRejectMock).toHaveBeenCalledTimes(1);
  });

  it("reruns a failed open work order from the note", async () => {
    const user = userEvent.setup();
    renderPopup(splitRunFixtureForWorkOrder(FAILED_WORK_ORDER));

    const note = screen.getByTestId("split-run-attention-note");
    expect(within(note).queryByRole("button", { name: "Approve" })).not.toBeInTheDocument();
    await user.click(within(note).getByRole("button", { name: "Rerun" }));
    expect(handleStopMock).toHaveBeenCalledWith("rerun-step", expect.objectContaining({ kind: "failed" }));
  });

  it("reopens a closed work order from the note", async () => {
    const user = userEvent.setup();
    renderPopup(splitRunFixtureForWorkOrder(BOARD_IMPLEMENT_FAILED_ORDER));

    const note = screen.getByTestId("split-run-attention-note");
    expect(screen.queryByTestId("split-run-header-actions")).not.toBeInTheDocument();
    expect(within(note).queryByRole("button", { name: "Reject" })).not.toBeInTheDocument();
    await user.click(within(note).getByRole("button", { name: "Reopen" }));
    expect(handleStopMock).toHaveBeenCalledWith(
      "reopen",
      expect.objectContaining({ kind: "failed", status: "failed" }),
    );
  });
});
