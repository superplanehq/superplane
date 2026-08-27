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
    busy: false,
  }),
}));

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";

import { DRAFT_WORK_ORDER } from "../../__fixtures__/factoryPageResponses";
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

describe("WorkOrderSplitRunPopup stop actions", () => {
  beforeEach(() => {
    handleStopMock.mockReset();
    handleRejectMock.mockReset();
  });

  it("closes as rejected when Stop and Close is used", async () => {
    const user = userEvent.setup();
    renderPopup(SPLIT_RUN_RUNNING);

    await user.click(screen.getByRole("button", { name: "Stop and Close" }));
    expect(handleStopMock).toHaveBeenCalledWith("canceled", expect.objectContaining({ kind: "running" }));
  });

  it("closes as completed after that Stop outcome is chosen", async () => {
    const user = userEvent.setup();
    renderPopup(SPLIT_RUN_RUNNING);

    await user.click(screen.getByRole("button", { name: "Choose how to stop" }));
    await user.click(await screen.findByRole("menuitem", { name: /Stop and Complete/ }));
    await user.click(screen.getByRole("button", { name: "Stop and Complete" }));
    expect(handleStopMock).toHaveBeenCalledWith("completed", expect.objectContaining({ kind: "running" }));
  });

  it("reruns the current step after that Stop outcome is chosen", async () => {
    const user = userEvent.setup();
    renderPopup(SPLIT_RUN_RUNNING);

    await user.click(screen.getByRole("button", { name: "Choose how to stop" }));
    await user.click(await screen.findByRole("menuitem", { name: /Rerun this step/ }));
    await user.click(screen.getByRole("button", { name: "Rerun step" }));
    expect(handleStopMock).toHaveBeenCalledWith("rerun-step", expect.objectContaining({ kind: "running" }));
  });

  it("reruns from the start after that Stop outcome is chosen", async () => {
    const user = userEvent.setup();
    renderPopup(SPLIT_RUN_RUNNING);

    await user.click(screen.getByRole("button", { name: "Choose how to stop" }));
    await user.click(await screen.findByRole("menuitem", { name: /Rerun from the start/ }));
    await user.click(screen.getByRole("button", { name: "Rerun from start" }));
    expect(handleStopMock).toHaveBeenCalledWith("rerun-start", expect.objectContaining({ kind: "running" }));
  });

  it("closes the popup after Reject deletes a draft", async () => {
    handleRejectMock.mockResolvedValue(true);
    const user = userEvent.setup();
    const onClose = vi.fn();
    renderPopup(splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER), onClose);

    await user.click(within(screen.getByTestId("split-run-review")).getByRole("button", { name: "Reject" }));
    expect(handleRejectMock).toHaveBeenCalledTimes(1);
    await waitFor(() => {
      expect(onClose).toHaveBeenCalledTimes(1);
    });
  });

  it("reopens a closed work order from Reopen", async () => {
    const user = userEvent.setup();
    renderPopup(splitRunFixtureForWorkOrder(BOARD_IMPLEMENT_FAILED_ORDER));

    expect(screen.queryByRole("button", { name: "Rerun step" })).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Reopen" }));
    expect(handleStopMock).toHaveBeenCalledWith(
      "reopen",
      expect.objectContaining({ kind: "failed", status: "failed" }),
    );
  });

  it("hides Rerun when the work order is already a draft", async () => {
    const user = userEvent.setup();
    renderPopup({
      ...SPLIT_RUN_RUNNING,
      footer: { ...SPLIT_RUN_RUNNING.footer, status: "draft" },
    });

    await user.click(screen.getByRole("button", { name: "Choose how to stop" }));
    const menu = await screen.findByRole("menu");
    expect(within(menu).queryByRole("menuitem", { name: /Rerun this step/ })).not.toBeInTheDocument();
    expect(within(menu).queryByRole("menuitem", { name: /Rerun from the start/ })).not.toBeInTheDocument();
    expect(within(menu).getByRole("menuitem", { name: /Stop and Close/ })).toBeInTheDocument();
    expect(within(menu).getByRole("menuitem", { name: /Stop and Complete/ })).toBeInTheDocument();
  });
});
