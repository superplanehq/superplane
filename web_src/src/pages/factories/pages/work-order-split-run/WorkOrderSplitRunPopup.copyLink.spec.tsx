import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";

import { FACTORIES_ORGANIZATION_ID, PRIMARY_FACTORY_KEY } from "../../__fixtures__/factoryPageResponses";
import { BOARD_IMPLEMENT_NOTIFY_ORDER } from "../../__fixtures__/lineMetricsBoardOrders";
import { workOrderDetailPath } from "../../lib/factoryPagePaths";
import { WorkOrderSplitRunPopup } from "./WorkOrderSplitRunPopup";
import { splitRunFixtureForWorkOrder } from "./splitRunMocks";

function mockClipboard() {
  const writeText = vi.fn(() => Promise.resolve());
  Object.defineProperty(navigator, "clipboard", {
    configurable: true,
    value: { writeText },
  });
  return writeText;
}

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

describe("WorkOrderSplitRunPopup copy-link button", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it("shows a link icon in the header that copies the work-order permalink", async () => {
    const writeText = mockClipboard();

    renderPopup();

    const button = screen.getByTestId("popup-work-order-copy-link-button");
    expect(button).toBeInTheDocument();

    fireEvent.click(button);
    await flushPromises();

    const expectedUrl =
      window.location.origin +
      workOrderDetailPath(FACTORIES_ORGANIZATION_ID, PRIMARY_FACTORY_KEY, BOARD_IMPLEMENT_NOTIFY_ORDER.number!);
    expect(writeText).toHaveBeenCalledWith(expectedUrl);
    expect(screen.getAllByText("Copied").length).toBeGreaterThan(0);

    act(() => {
      vi.advanceTimersByTime(1600);
    });

    expect(screen.queryByText("Copied")).not.toBeInTheDocument();
  });
});
