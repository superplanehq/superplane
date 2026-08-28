import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { FactoriesFactory, FactoriesWorkOrder } from "@/api-client";
import type * as canvasData from "@/hooks/useCanvasData";
import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";
import {
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
  REFUND_FACTORY,
  REFUND_LINE_PLAN_ID,
} from "../__fixtures__/factoryPageResponses";
import { BOARD_DONE_REJECTED_ORDER } from "../__fixtures__/lineMetricsBoardOrders";
import { withPlanLinePhases } from "../__fixtures__/lineMetricsPlanLine";
import { FactoriesLayoutContext } from "../layout/factoriesLayoutContext";
import { LinesPage } from "./LinesPage";

const useFactoryWorkOrders = vi.fn(() => ({ data: [] as FactoriesWorkOrder[] }));

vi.mock("@/hooks/useFactoryData", () => ({
  useFactoryWorkOrders: () => useFactoryWorkOrders(),
  useFactoryApps: () => ({ data: [] }),
  useCreateFactoryLine: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateFactoryLine: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useWorkOrderEvents: () => ({ data: { pages: [] } }),
  useWorkOrderArtifacts: () => ({ data: [] }),
  useFactoryPullRequests: () => ({ data: [] }),
  useCloseWorkOrder: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useDispatchWorkOrder: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateWorkOrder: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateWorkOrderAssignees: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateWorkOrderStatus: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useCreateWorkOrder: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

vi.mock("@/hooks/useFactoryIntakeData", () => ({
  useFactoryIntakes: () => ({ data: [] }),
  useFactoryIntakeRuns: () => ({ data: [], isLoading: false, isError: false, refetch: vi.fn() }),
  useCreateFactoryIntake: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateFactoryIntake: () => ({ mutateAsync: vi.fn(), isPending: false, error: null }),
  useSearchFactoryIntakeItems: () => ({ data: [], isLoading: false, isError: false }),
  useImportFactoryIntakeItem: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

vi.mock("@/hooks/useWorkOrderCardActions", () => ({
  useWorkOrderCardActions: () => ({
    dispatchingOrderIds: new Set<string>(),
    isAssigneesSaving: false,
    onDispatch: vi.fn(),
    onAssigneesSave: vi.fn(),
  }),
}));

vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({ canAct: () => true, isLoading: false }),
}));

vi.mock("@/hooks/usePageTitle", () => ({
  usePageTitle: () => undefined,
}));

vi.mock("@/hooks/useMe", () => ({
  useMe: () => ({ data: { id: "storybook-user" } }),
}));

vi.mock("@/hooks/useWorkOrderChecks", () => ({
  useWorkOrderChecks: () => ({ data: [] }),
}));

vi.mock("@/hooks/useFactoryPRFeedbackData", () => ({
  useFactoryPRFeedbackHandlers: () => ({ data: [] }),
}));

vi.mock("@/hooks/useCanvasData", async (importOriginal) => {
  const actual = await importOriginal<typeof canvasData>();
  return {
    ...actual,
    useCanvas: () => ({ data: { spec: { nodes: [] } }, isPending: false, isError: false }),
    useUpdateCanvasVersion: () => ({ mutateAsync: vi.fn(), isPending: false }),
    useCommitCanvasStaging: () => ({ mutateAsync: vi.fn(), isPending: false }),
  };
});

function renderBoard(factory: FactoriesFactory = REFUND_FACTORY) {
  return render(
    <QueryClientProvider client={new QueryClient()}>
      <ThemeProvider>
        <TooltipProvider>
          <MemoryRouter initialEntries={[`/org-1/workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_PLAN_ID}`]}>
            <FactoriesLayoutContext.Provider
              value={{
                organizationId: "org-1",
                factoryId: factory.id ?? PRIMARY_FACTORY_ID,
                factoryKey: factory.key ?? PRIMARY_FACTORY_KEY,
                factory,
                factories: [factory],
                openCreateWorkOrder: vi.fn(),
              }}
            >
              <Routes>
                <Route path="/org-1/workspaces/:factoryKey/lines/:lineId" element={<LinesPage />} />
              </Routes>
            </FactoriesLayoutContext.Provider>
          </MemoryRouter>
        </TooltipProvider>
      </ThemeProvider>
    </QueryClientProvider>,
  );
}

describe("LinesPage Done column", () => {
  beforeEach(() => {
    window.localStorage.clear();
    useFactoryWorkOrders.mockReturnValue({ data: [] });
  });

  it("always shows a Done column after the line stages", () => {
    renderBoard();

    expect(screen.getByTestId("lines-column-title-backlog")).toHaveTextContent("Backlog");
    expect(screen.getByTestId("lines-column-title-phase-0")).toBeInTheDocument();
    expect(screen.getByTestId("lines-column-title-phase-1")).toBeInTheDocument();
    expect(screen.getByTestId("lines-column-title-verify")).toHaveTextContent("Verify");
    expect(screen.getByTestId("lines-column-title-done")).toHaveTextContent("Done");
    expect(screen.getByTestId("lines-done-column")).toHaveTextContent("No tasks in Done.");
    expect(screen.getByTestId("lines-verify-column")).toHaveTextContent("No tasks in Verify.");
    expect(screen.getByTestId("lines-phase-column-1")).toHaveTextContent("Nothing here.");
    expect(screen.queryByTestId("lines-column-title-phase-2")).not.toBeInTheDocument();
  });

  it("puts a closed work order in Done instead of the last stage", () => {
    useFactoryWorkOrders.mockReturnValue({
      data: [BOARD_DONE_REJECTED_ORDER],
    });
    renderBoard();

    expect(
      within(screen.getByTestId("lines-done-column")).getByText("Replace the refund batch exporter"),
    ).toBeInTheDocument();
    expect(
      within(screen.getByTestId("lines-phase-column-0")).queryByText("Replace the refund batch exporter"),
    ).not.toBeInTheDocument();
    expect(
      within(screen.getByTestId("lines-phase-column-1")).queryByText("Replace the refund batch exporter"),
    ).not.toBeInTheDocument();
    expect(
      within(screen.getByTestId("lines-verify-column")).queryByText("Replace the refund batch exporter"),
    ).not.toBeInTheDocument();
  });

  it("collects finished work orders in the Done column", () => {
    useFactoryWorkOrders.mockReturnValue({
      data: [
        {
          id: "wo-completed",
          title: "Publish refund SLA dashboard",
          state: "STATE_CLOSED",
          result: "RESULT_COMPLETED",
          lineDispatches: [{ id: "dispatch-1", line: { id: REFUND_LINE_PLAN_ID } }],
        },
      ] as FactoriesWorkOrder[],
    });
    renderBoard();

    const done = screen.getByTestId("lines-done-column");
    expect(within(done).getByRole("button", { name: "Open Publish refund SLA dashboard" })).toBeInTheDocument();
    expect(screen.getByTestId("lines-phase-column-1")).toHaveTextContent("Nothing here.");
    expect(screen.getByTestId("lines-verify-column")).toHaveTextContent("No tasks in Verify.");
  });

  it("keeps the bookend Done column when the line has its own Done automation", () => {
    const factory: FactoriesFactory = {
      ...REFUND_FACTORY,
      lines: (REFUND_FACTORY.lines ?? []).map(withPlanLinePhases),
    };
    renderBoard(factory);

    expect(screen.getByTestId("lines-done-column")).toBeInTheDocument();
    expect(screen.queryByTestId("lines-phase-column-2")).not.toBeInTheDocument();
  });
});
