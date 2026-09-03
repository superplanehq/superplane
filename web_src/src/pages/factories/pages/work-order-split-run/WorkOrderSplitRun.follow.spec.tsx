import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ComponentProps } from "react";
import { MemoryRouter } from "react-router";
import { describe, expect, it } from "vitest";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";

import { BOARD_IMPLEMENT_NOTIFY_ORDER } from "../../__fixtures__/lineMetricsBoardOrders";
import { CREATE_WITH_AGENT_COPY } from "../createWithAgentCopy";
import { WorkOrderSplitRunPopup } from "./WorkOrderSplitRunPopup";
import { SPLIT_RUN_RUNNING, splitRunFixtureForWorkOrder } from "./splitRunMocks";

function renderPopup(props: ComponentProps<typeof WorkOrderSplitRunPopup>) {
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <ThemeProvider>
          <TooltipProvider>
            <WorkOrderSplitRunPopup {...props} />
          </TooltipProvider>
        </ThemeProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("WorkOrderSplitRunPopup jump-to-latest", () => {
  it("does not show a Follow toggle in the Automations tab", async () => {
    const user = userEvent.setup();
    renderPopup({ fixture: SPLIT_RUN_RUNNING });

    await user.click(screen.getByRole("tab", { name: "Automations" }));

    expect(screen.queryByRole("switch", { name: "Follow" })).not.toBeInTheDocument();
    expect(screen.getByTestId("split-run-log-scroll")).toBeInTheDocument();
  });

  it("hides the pill while the log follows the latest line", async () => {
    const user = userEvent.setup();
    renderPopup({ fixture: SPLIT_RUN_RUNNING });

    await user.click(screen.getByRole("tab", { name: "Automations" }));

    expect(screen.queryByText(CREATE_WITH_AGENT_COPY.viewingOlder)).not.toBeInTheDocument();
  });

  it("shows the pill once the run is finished, since following starts off", async () => {
    const user = userEvent.setup();
    renderPopup({
      fixture: splitRunFixtureForWorkOrder(BOARD_IMPLEMENT_NOTIFY_ORDER),
    });
    await user.click(screen.getByRole("tab", { name: "Automations" }));

    expect(screen.getByText(CREATE_WITH_AGENT_COPY.viewingOlder)).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: CREATE_WITH_AGENT_COPY.jumpToLatest }));
    expect(screen.queryByText(CREATE_WITH_AGENT_COPY.viewingOlder)).not.toBeInTheDocument();
  });

  it("shows jump to latest after the user scrolls up, then hides it on click", async () => {
    const user = userEvent.setup();
    renderPopup({ fixture: SPLIT_RUN_RUNNING });
    await user.click(screen.getByRole("tab", { name: "Automations" }));

    const scroller = screen.getByTestId("split-run-log-scroll");
    Object.defineProperty(scroller, "scrollHeight", { configurable: true, get: () => 400 });
    Object.defineProperty(scroller, "clientHeight", { configurable: true, get: () => 100 });
    await new Promise<void>((resolve) => {
      requestAnimationFrame(() => requestAnimationFrame(() => resolve()));
    });

    scroller.scrollTop = 0;
    fireEvent.scroll(scroller);
    expect(screen.getByText(CREATE_WITH_AGENT_COPY.viewingOlder)).toBeInTheDocument();
    expect(screen.getByTestId("split-run-older")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: CREATE_WITH_AGENT_COPY.jumpToLatest }));
    expect(screen.queryByText(CREATE_WITH_AGENT_COPY.viewingOlder)).not.toBeInTheDocument();
  });

  it("turns following back on when the user scrolls to the latest line", async () => {
    const user = userEvent.setup();
    renderPopup({ fixture: SPLIT_RUN_RUNNING });
    await user.click(screen.getByRole("tab", { name: "Automations" }));

    const scroller = screen.getByTestId("split-run-log-scroll");
    Object.defineProperty(scroller, "scrollHeight", { configurable: true, get: () => 400 });
    Object.defineProperty(scroller, "clientHeight", { configurable: true, get: () => 100 });
    await new Promise<void>((resolve) => {
      requestAnimationFrame(() => requestAnimationFrame(() => resolve()));
    });

    scroller.scrollTop = 0;
    fireEvent.scroll(scroller);
    expect(screen.getByText(CREATE_WITH_AGENT_COPY.viewingOlder)).toBeInTheDocument();

    scroller.scrollTop = 300;
    fireEvent.scroll(scroller);
    expect(screen.queryByText(CREATE_WITH_AGENT_COPY.viewingOlder)).not.toBeInTheDocument();
  });
});
