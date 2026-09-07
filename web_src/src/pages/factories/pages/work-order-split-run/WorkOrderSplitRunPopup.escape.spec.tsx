import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

const { handleStopMock, handleRejectMock, handleBackToDraftMock } = vi.hoisted(() => ({
  handleStopMock: vi.fn(),
  handleRejectMock: vi.fn(),
  handleBackToDraftMock: vi.fn(),
}));

vi.mock("./useSplitRunFooterActions", () => ({
  useSplitRunFooterActions: () => ({
    handleStop: handleStopMock,
    handleReject: handleRejectMock,
    handleBackToDraft: handleBackToDraftMock,
    handleStopAutomation: vi.fn(),
    busy: false,
  }),
}));

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";

import { OPEN_WORK_ORDER } from "../../__fixtures__/factoryPageResponses";
import { WorkOrderSplitRunPopup } from "./WorkOrderSplitRunPopup";
import { splitRunFixtureForWorkOrder } from "./splitRunMocks";

function renderPopup(onClose: () => void) {
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <ThemeProvider>
          <TooltipProvider>
            <WorkOrderSplitRunPopup fixture={splitRunFixtureForWorkOrder(OPEN_WORK_ORDER)} onClose={onClose} />
          </TooltipProvider>
        </ThemeProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("WorkOrderSplitRunPopup Escape handling", () => {
  beforeEach(() => {
    handleStopMock.mockReset();
    handleRejectMock.mockReset();
    handleBackToDraftMock.mockReset().mockResolvedValue(true);
  });

  it("closes the popup when Escape is pressed", async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    renderPopup(onClose);

    await user.keyboard("{Escape}");

    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("cancels an in-progress title edit on the first Escape, and closes on the second", async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    renderPopup(onClose);

    const title = screen.getByTestId("popup-work-order-title");
    await user.click(title);
    const input = screen.getByTestId("popup-work-order-title-input");
    await waitFor(() => expect(input).toHaveFocus());

    await user.keyboard("{Escape}");
    expect(onClose).not.toHaveBeenCalled();
    expect(screen.queryByTestId("popup-work-order-title-input")).not.toBeInTheDocument();

    await user.keyboard("{Escape}");
    expect(onClose).toHaveBeenCalledTimes(1);
  });
});
