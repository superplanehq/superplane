import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ComponentProps } from "react";
import { MemoryRouter } from "react-router";
import { describe, expect, it } from "bun:test";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";

import { BOARD_IMPLEMENT_NOTIFY_ORDER } from "../../__fixtures__/lineMetricsBoardOrders";
import { CREATE_WITH_AGENT_COPY } from "../createWithAgentCopy";
import { WorkOrderSplitRunPopup } from "./WorkOrderSplitRunPopup";
import { splitRunFixtureForWorkOrder } from "./splitRunMocks";
import { SPLIT_RUN_SUPER503_RUNNING } from "./splitRunSuper503RunningFixture";

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

function runningStepLog() {
  const node = document.querySelector("[data-testid^='redesign-step-log-']");
  if (!(node instanceof HTMLElement)) {
    throw new Error("running step log not found");
  }
  return node;
}

describe("WorkOrderSplitRunPopup jump-to-latest", () => {
  it("does not show a Follow toggle on the console", () => {
    renderPopup({ fixture: SPLIT_RUN_SUPER503_RUNNING });

    expect(screen.queryByRole("switch", { name: "Follow" })).not.toBeInTheDocument();
    expect(screen.getByTestId("redesign-console-variant")).toBeInTheDocument();
  });

  it("hides the pill while the log follows the latest line", () => {
    renderPopup({ fixture: SPLIT_RUN_SUPER503_RUNNING });

    expect(screen.queryByText(CREATE_WITH_AGENT_COPY.viewingOlder)).not.toBeInTheDocument();
  });

  it("keeps the pill hidden for a finished run, since auto-scroll starts on", () => {
    renderPopup({
      fixture: splitRunFixtureForWorkOrder(BOARD_IMPLEMENT_NOTIFY_ORDER),
    });

    expect(screen.queryByText(CREATE_WITH_AGENT_COPY.viewingOlder)).not.toBeInTheDocument();
  });

  it("shows jump to latest after the user scrolls up, then hides it on click", async () => {
    const user = userEvent.setup();
    renderPopup({ fixture: SPLIT_RUN_SUPER503_RUNNING });

    const scroller = runningStepLog();
    Object.defineProperty(scroller, "scrollHeight", { configurable: true, get: () => 400 });
    Object.defineProperty(scroller, "clientHeight", { configurable: true, get: () => 100 });
    await new Promise<void>((resolve) => {
      requestAnimationFrame(() => requestAnimationFrame(() => resolve()));
    });

    scroller.scrollTop = 0;
    fireEvent.scroll(scroller);
    expect(screen.getByText(CREATE_WITH_AGENT_COPY.viewingOlder)).toBeInTheDocument();
    expect(document.querySelector("[data-testid^='redesign-step-older-']")).toBeTruthy();

    await user.click(screen.getByRole("button", { name: CREATE_WITH_AGENT_COPY.jumpToLatest }));
    expect(screen.queryByText(CREATE_WITH_AGENT_COPY.viewingOlder)).not.toBeInTheDocument();
  });

  it("turns following back on when the user scrolls to the latest line", async () => {
    renderPopup({ fixture: SPLIT_RUN_SUPER503_RUNNING });

    const scroller = runningStepLog();
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
