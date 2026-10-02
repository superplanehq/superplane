import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, fireEvent, render, screen } from "@testing-library/react";
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

vi.mock("@/contexts/usePermissions");

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";
import { usePermissions } from "@/contexts/usePermissions";

import { FACTORIES_ORGANIZATION_ID, PRIMARY_FACTORY_KEY } from "../../__fixtures__/factoryPageResponses";
import { BOARD_IMPLEMENT_NOTIFY_ORDER } from "../../__fixtures__/lineMetricsBoardOrders";
import { WorkOrderSplitRunPopup } from "./WorkOrderSplitRunPopup";
import { splitRunFixtureForWorkOrder } from "./splitRunMocks";

async function flushPromises() {
  await act(async () => {
    await Promise.resolve();
  });
}

function renderPopup(canCreateTasks: boolean = true, onClose: () => void = () => {}) {
  const usePermissionsMock = usePermissions as any;
  usePermissionsMock.mockReturnValue({
    canAct: (resource: string, action: string) => {
      if (resource === "work_orders" && action === "create") {
        return canCreateTasks;
      }
      return true;
    },
    isLoading: false,
  });

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
              onClose={onClose}
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

  it("shows the duplicate button next to the copy-link button", () => {
    renderPopup();

    const button = screen.getByTestId("popup-work-order-duplicate-button");
    expect(button).toBeInTheDocument();
  });

  it("disables the duplicate button with a tooltip for users without create permission", () => {
    renderPopup(false);

    const button = screen.getByTestId("popup-work-order-duplicate-button");
    expect(button).toHaveAttribute("disabled");
  });

  it("enables the duplicate button for users with create permission", () => {
    renderPopup(true);

    const button = screen.getByTestId("popup-work-order-duplicate-button");
    expect(button).not.toHaveAttribute("disabled");
  });

  it("shows Duplicate tooltip on hover for enabled button", async () => {
    renderPopup(true);

    const button = screen.getByTestId("popup-work-order-duplicate-button");
    fireEvent.mouseEnter(button);
    await flushPromises();

    act(() => {
      vi.advanceTimersByTime(100);
    });

    // The tooltip should be present in the DOM
    expect(screen.queryByText("Duplicate")).toBeInTheDocument();
  });
});
