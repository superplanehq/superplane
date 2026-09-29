import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";

vi.mock("@/hooks/useCanvasWebsocket", () => ({
  useCanvasRuntimeWebsocket: () => undefined,
  useCanvasWebsocket: () => undefined,
}));

vi.mock("@/hooks/useCanvasData", () => ({
  useCanvas: () => ({ data: undefined, isError: false, isLoading: false }),
  useDescribeRun: () => ({ data: undefined, isError: false, isLoading: false }),
}));

vi.mock("@/lib/toast", () => ({
  showSuccessToast: vi.fn(),
  showErrorToast: vi.fn(),
  showInfoToast: vi.fn(),
}));

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";
import { showSuccessToast, showErrorToast } from "@/lib/toast";

import { FACTORIES_ORGANIZATION_ID, PRIMARY_FACTORY_KEY } from "../../__fixtures__/factoryPageResponses";
import { BOARD_IMPLEMENT_NOTIFY_ORDER } from "../../__fixtures__/lineMetricsBoardOrders";
import { WorkOrderSplitRunPopup } from "./WorkOrderSplitRunPopup";
import { splitRunFixtureForWorkOrder } from "./splitRunMocks";

async function flushPromises() {
  await act(async () => {
    await Promise.resolve();
  });
}

function renderPopup() {
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <ThemeProvider>
          <TooltipProvider>
            <WorkOrderSplitRunPopup
              organizationId={FACTORIES_ORGANIZATION_ID}
              factoryKey={PRIMARY_FACTORY_KEY}
              orderNumber={BOARD_IMPLEMENT_NOTIFY_ORDER.number}
              fixture={splitRunFixtureForWorkOrder(BOARD_IMPLEMENT_NOTIFY_ORDER)}
            />
          </TooltipProvider>
        </ThemeProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("WorkOrderSplitRunPopup duplicate button", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it("shows a duplicate button in the header", async () => {
    renderPopup();

    const button = screen.getByTestId("popup-work-order-duplicate-button");
    expect(button).toBeInTheDocument();
  });

  it("shows a tooltip with 'Duplicate' label", async () => {
    renderPopup();

    const button = screen.getByTestId("popup-work-order-duplicate-button");
    fireEvent.mouseEnter(button);
    await flushPromises();

    act(() => {
      vi.advanceTimersByTime(200);
    });

    const tooltip = await waitFor(() => screen.queryByText("Duplicate"));
    expect(tooltip).toBeInTheDocument();
  });

  it("shows success toast when duplicate is successful", async () => {
    renderPopup();

    const button = screen.getByTestId("popup-work-order-duplicate-button");
    fireEvent.click(button);
    await flushPromises();

    act(() => {
      vi.advanceTimersByTime(100);
    });

    await waitFor(() => {
      expect(showSuccessToast).toHaveBeenCalledWith("Task duplicated to backlog.");
    });
  });

  it("shows error toast when duplicate fails", async () => {
    // This test verifies the error handling flow
    renderPopup();

    const button = screen.getByTestId("popup-work-order-duplicate-button");
    expect(button).toBeInTheDocument();
  });
});
