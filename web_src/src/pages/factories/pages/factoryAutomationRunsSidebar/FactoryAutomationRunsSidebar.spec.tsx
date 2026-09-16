import type { CanvasesCanvasRun, FactoriesWorkOrder } from "@/api-client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";

import { FactoriesLayoutContext } from "../../layout/factoriesLayoutContext";
import { FactoryAutomationRunsSidebar } from "./FactoryAutomationRunsSidebar";

const { useInfiniteCanvasRuns, useFactoryWorkOrders } = vi.hoisted(() => ({
  useInfiniteCanvasRuns: vi.fn(),
  useFactoryWorkOrders: vi.fn(),
}));

vi.mock("@/hooks/useCanvasData", () => ({
  useInfiniteCanvasRuns,
}));

vi.mock("@/hooks/useFactoryData", () => ({
  useFactoryWorkOrders: (...args: unknown[]) => useFactoryWorkOrders(...args),
  useFactoryPullRequests: () => ({ data: [] }),
  useDispatchWorkOrder: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateWorkOrderAssignees: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

vi.mock("@/hooks/useWorkOrderCardActions", () => ({
  useWorkOrderCardActions: () => ({
    dispatchingOrderIds: new Set<string>(),
    isAssigneesSaving: false,
    onDispatch: vi.fn(),
    onAssigneesSave: vi.fn(),
  }),
}));

function makeRun(overrides: Partial<CanvasesCanvasRun> = {}): CanvasesCanvasRun {
  return {
    id: "run-1",
    canvasId: "app-github-issues-intake",
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
    createdAt: "2026-05-01T12:00:00Z",
    rootEvent: {
      id: "event-1",
      nodeId: "trigger",
      customName: "feat: Add console empty-state",
      createdAt: "2026-05-01T12:00:00Z",
    },
    executions: [],
    ...overrides,
  };
}

function mockRuns(runs: CanvasesCanvasRun[]) {
  useInfiniteCanvasRuns.mockReturnValue({
    data: { pages: [{ runs, totalCount: runs.length }] },
    isPending: false,
    isError: false,
    hasNextPage: false,
    isFetchingNextPage: false,
    fetchNextPage: vi.fn(),
    refetch: vi.fn(),
  });
}

function mockWorkOrders(orders: FactoriesWorkOrder[]) {
  useFactoryWorkOrders.mockReturnValue({ data: orders });
}

function renderSidebar(
  props: Partial<Parameters<typeof FactoryAutomationRunsSidebar>[0]> = {},
  options?: { withLayout?: boolean; workOrders?: FactoriesWorkOrder[] },
) {
  mockWorkOrders(options?.workOrders ?? []);
  const sidebar = (
    <FactoryAutomationRunsSidebar
      canvasId="app-github-issues-intake"
      selectedRunId={null}
      onSelectRun={vi.fn()}
      {...props}
    />
  );
  const tree = options?.withLayout ? (
    <FactoriesLayoutContext.Provider
      value={{
        organizationId: "org-1",
        factoryId: "factory-1",
        factoryKey: "RF",
        factory: { id: "factory-1", key: "RF", name: "Refunds", lines: [] },
        factories: [],
        openCreateWorkOrder: () => {},
      }}
    >
      {sidebar}
    </FactoriesLayoutContext.Provider>
  ) : (
    sidebar
  );

  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <ThemeProvider>
          <TooltipProvider>
            <div className="flex h-[32rem]">{tree}</div>
          </TooltipProvider>
        </ThemeProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  useFactoryWorkOrders.mockReturnValue({ data: [] });
});

afterEach(() => {
  localStorage.clear();
});

describe("FactoryAutomationRunsSidebar", () => {
  it("lists factory runs without using the standalone canvas sidebar", () => {
    mockRuns([
      makeRun(),
      makeRun({
        id: "run-2",
        state: "STATE_STARTED",
        result: "RESULT_UNKNOWN",
        rootEvent: { id: "event-2", nodeId: "trigger", customName: "On Issue Labeled" },
      }),
    ]);

    renderSidebar();

    expect(useInfiniteCanvasRuns).toHaveBeenCalledWith("app-github-issues-intake", {}, true);
    const sidebar = screen.getByTestId("factory-automation-runs-sidebar");
    expect(screen.queryByTestId("canvas-runs-sidebar")).not.toBeInTheDocument();
    expect(within(sidebar).getByText("Runs")).toBeInTheDocument();
    expect(within(sidebar).getByLabelText("Search runs")).toBeInTheDocument();
    expect(within(sidebar).getByText("feat: Add console empty-state")).toBeInTheDocument();
    expect(within(sidebar).getByText("On Issue Labeled")).toBeInTheDocument();
  });

  it("keeps the selected run in the sidebar without leaving the page", async () => {
    const user = userEvent.setup();
    mockRuns([makeRun()]);

    function ControlledSidebar() {
      const [selectedRunId, setSelectedRunId] = useState<string | null>(null);
      return (
        <FactoryAutomationRunsSidebar
          canvasId="app-github-issues-intake"
          selectedRunId={selectedRunId}
          onSelectRun={setSelectedRunId}
          runHrefFor={(runId) => `/org-1/workspaces/RF/apps/app-github-issues-intake?run=${runId}&from=lines`}
        />
      );
    }

    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <MemoryRouter>
          <ThemeProvider>
            <TooltipProvider>
              <div className="flex h-[32rem]">
                <ControlledSidebar />
              </div>
            </TooltipProvider>
          </ThemeProvider>
        </MemoryRouter>
      </QueryClientProvider>,
    );

    const row = screen.getByTestId("factory-automation-runs-row");
    const link = within(row).getByRole("link", { name: "feat: Add console empty-state" });
    expect(link).toHaveAttribute("href", "/org-1/workspaces/RF/apps/app-github-issues-intake?run=run-1&from=lines");

    await user.click(link);
    expect(row.className).not.toMatch(/bg-sky-50/);
    expect(link.className).toMatch(/bg-slate-200/);
    expect(link).toHaveAttribute("data-selected", "true");
  });

  it("shows the Kanban task card when the run belongs to a task", () => {
    mockRuns([makeRun({ id: "run-intake" })]);
    renderSidebar(
      { organizationId: "org-1", factoryId: "factory-1", factoryKey: "RF" },
      {
        workOrders: [
          {
            id: "wo-1",
            number: "12",
            key: "RF-12",
            title: "feat: Add console empty-state",
            state: "STATE_DRAFT",
            sourceRunId: "run-intake",
          },
        ],
      },
    );

    const card = screen.getByTestId("work-order-card-wo-1");
    expect(card).toHaveTextContent("feat: Add console empty-state");
    expect(card).not.toHaveAttribute("data-selected");
    expect(screen.getByTestId("factory-automation-runs-row").className).not.toMatch(/bg-sky-50/);
    expect(screen.queryByTestId("factory-automation-run-run-intake")).not.toBeInTheDocument();
  });

  it("shows an empty state when the canvas has no runs", () => {
    mockRuns([]);

    renderSidebar();

    expect(screen.getByText("No runs")).toBeInTheDocument();
  });
});
