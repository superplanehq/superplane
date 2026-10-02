import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ComponentProps } from "react";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "bun:test";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";

import { BOARD_IMPLEMENT_NOTIFY_ORDER } from "../../__fixtures__/lineMetricsBoardOrders";
import { CREATE_WITH_AGENT_COPY } from "../createWithAgentCopy";
import { WorkOrderSplitRunPopup } from "./WorkOrderSplitRunPopup";
import { SPLIT_RUN_RUNNING, splitRunFixtureForWorkOrder, type SplitRunFixture } from "./splitRunMocks";

/** The running Implement phase with agent notes, so the console shows a live step log. */
const RUNNING_WITH_AGENT_NOTES: SplitRunFixture = {
  ...SPLIT_RUN_RUNNING,
  phases: SPLIT_RUN_RUNNING.phases.map((phase) =>
    phase.id === "implement"
      ? {
          ...phase,
          stream: [
            ...phase.stream,
            {
              id: "note-read-test",
              at: "12:25:40",
              componentName: "Read the failing test",
              status: "passed" as const,
              duration: "1m",
              note: true,
              componentType: "prompt",
            },
            {
              id: "note-write-fix",
              at: "12:26:40",
              componentName: "Write the reconciliation fix",
              status: "running" as const,
              note: true,
              componentType: "prompt",
            },
            {
              id: "note-write-fix-detail",
              at: "12:26:45",
              componentName: "Edits reconciliation_worker_test.go",
              status: "running" as const,
              note: true,
              noteParentId: "note-write-fix",
              componentType: "note",
            },
          ],
        }
      : phase,
  ),
};

vi.mock("@monaco-editor/react", () => ({
  default: ({ value }: { value?: string }) => <pre data-testid="monaco-stub">{value}</pre>,
}));

vi.mock("@/hooks/useExperimentalFeature", () => ({
  useExperimentalFeature: () => ({
    has: () => true,
    enabledExperimentalFeatures: [],
    isLoading: false,
  }),
}));

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
    renderPopup({ fixture: RUNNING_WITH_AGENT_NOTES });

    expect(screen.queryByRole("switch", { name: "Follow" })).not.toBeInTheDocument();
    expect(screen.getByTestId("redesign-console-variant")).toBeInTheDocument();
  });

  it("hides the pill while the log follows the latest line", () => {
    renderPopup({ fixture: RUNNING_WITH_AGENT_NOTES });

    expect(screen.queryByRole("button", { name: CREATE_WITH_AGENT_COPY.jumpToLatest })).not.toBeInTheDocument();
  });

  it("keeps the pill hidden for a finished run, since auto-scroll starts on", () => {
    renderPopup({
      fixture: splitRunFixtureForWorkOrder(BOARD_IMPLEMENT_NOTIFY_ORDER),
    });

    expect(screen.queryByRole("button", { name: CREATE_WITH_AGENT_COPY.jumpToLatest })).not.toBeInTheDocument();
  });

  it("shows jump to latest after the user scrolls up, then hides it on click", async () => {
    const user = userEvent.setup();
    renderPopup({ fixture: RUNNING_WITH_AGENT_NOTES });

    const scroller = runningStepLog();
    Object.defineProperty(scroller, "scrollHeight", { configurable: true, get: () => 400 });
    Object.defineProperty(scroller, "clientHeight", { configurable: true, get: () => 100 });
    await new Promise<void>((resolve) => {
      requestAnimationFrame(() => requestAnimationFrame(() => resolve()));
    });

    scroller.scrollTop = 0;
    fireEvent.scroll(scroller);
    expect(screen.getByRole("button", { name: CREATE_WITH_AGENT_COPY.jumpToLatest })).toBeInTheDocument();
    expect(document.querySelector("[data-testid^='redesign-step-older-']")).toBeTruthy();

    await user.click(screen.getByRole("button", { name: CREATE_WITH_AGENT_COPY.jumpToLatest }));
    expect(screen.queryByRole("button", { name: CREATE_WITH_AGENT_COPY.jumpToLatest })).not.toBeInTheDocument();
  });

  it("turns following back on when the user scrolls to the latest line", async () => {
    renderPopup({ fixture: RUNNING_WITH_AGENT_NOTES });

    const scroller = runningStepLog();
    Object.defineProperty(scroller, "scrollHeight", { configurable: true, get: () => 400 });
    Object.defineProperty(scroller, "clientHeight", { configurable: true, get: () => 100 });
    await new Promise<void>((resolve) => {
      requestAnimationFrame(() => requestAnimationFrame(() => resolve()));
    });

    scroller.scrollTop = 0;
    fireEvent.scroll(scroller);
    expect(screen.getByRole("button", { name: CREATE_WITH_AGENT_COPY.jumpToLatest })).toBeInTheDocument();

    scroller.scrollTop = 300;
    fireEvent.scroll(scroller);
    expect(screen.queryByRole("button", { name: CREATE_WITH_AGENT_COPY.jumpToLatest })).not.toBeInTheDocument();
  });
});
