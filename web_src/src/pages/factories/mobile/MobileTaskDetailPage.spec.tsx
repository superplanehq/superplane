import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes, useLocation } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import type { FactoriesWorkOrder } from "@/api-client";
import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";

import {
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
  PRIMARY_FACTORY_ROUTE_SEGMENT,
  REFUND_FACTORY,
  REFUND_LINE_HOTFIX_ID,
  REFUND_LINE_PLAN_ID,
} from "../__fixtures__/factoryPageResponses";
import { FactoriesLayoutContext } from "../layout/factoriesLayoutContext";
import { MobileTaskDetailPage } from "./MobileTaskDetailPage";

const useWorkOrder = vi.fn((): { data: FactoriesWorkOrder | undefined; isLoading: boolean; isError: boolean } => ({
  data: undefined,
  isLoading: true,
  isError: false,
}));

vi.mock("@/hooks/useFactoryData", () => ({
  useWorkOrder: () => useWorkOrder(),
  useWorkOrderArtifacts: () => ({ data: [] }),
  useFactoryAutomations: () => ({ data: [] }),
  useWorkOrderEvents: () => ({ data: { pages: [] } }),
  useCloseWorkOrder: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useDispatchWorkOrder: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateWorkOrderStatus: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

vi.mock("@/hooks/useFactoryPRFeedbackData", () => ({
  useFactoryPRFeedbackHandlers: () => ({ data: [] }),
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
  usePermissions: () => ({ canAct: () => true, currentUserId: "user-1", isLoading: false }),
}));

vi.mock("@/hooks/useOrganizationData", () => ({
  useOrganizationUsers: () => ({ data: [], isLoading: false }),
}));

vi.mock("@/hooks/useFactoryIntakeData", () => ({
  useFactoryIntakes: () => ({ data: [] }),
}));

function LocationProbe() {
  const location = useLocation();
  return <div data-testid="mobile-task-location">{`${location.pathname}${location.search}`}</div>;
}

function renderTask(entry: string | { pathname: string; search?: string; state?: unknown }) {
  return render(
    <QueryClientProvider client={new QueryClient()}>
      <ThemeProvider>
        <TooltipProvider>
          <MemoryRouter initialEntries={[entry]}>
            <FactoriesLayoutContext.Provider
              value={{
                organizationId: "org-1",
                factoryId: PRIMARY_FACTORY_ID,
                factoryKey: PRIMARY_FACTORY_KEY,
                routeSegment: PRIMARY_FACTORY_ROUTE_SEGMENT,
                factory: REFUND_FACTORY,
                factories: [REFUND_FACTORY],
                openCreateWorkOrder: vi.fn(),
              }}
            >
              <Routes>
                <Route path="/org-1/workspaces/:factoryKey/task/:orderNumber" element={<MobileTaskDetailPage />} />
                <Route path="/org-1/workspaces/:factoryKey/lines/:lineId" element={<LocationProbe />} />
              </Routes>
            </FactoriesLayoutContext.Provider>
          </MemoryRouter>
        </TooltipProvider>
      </ThemeProvider>
    </QueryClientProvider>,
  );
}

describe("MobileTaskDetailPage back link", () => {
  beforeEach(() => {
    useWorkOrder.mockReset();
    useWorkOrder.mockReturnValue({ data: undefined, isLoading: true, isError: false });
  });

  it("returns to the clicked line when the task URL has no line id", async () => {
    const user = userEvent.setup();
    renderTask({
      pathname: `/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/task/42`,
      state: { lineId: REFUND_LINE_HOTFIX_ID },
    });

    await user.click(screen.getByTestId("mobile-task-back"));

    expect(screen.getByTestId("mobile-task-location")).toHaveTextContent(
      `/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/lines/${REFUND_LINE_HOTFIX_ID}`,
    );
  });

  it("returns to the line in the task URL when that line still exists", async () => {
    const user = userEvent.setup();
    renderTask(`/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/task/42?lineId=${REFUND_LINE_HOTFIX_ID}`);

    await user.click(screen.getByTestId("mobile-task-back"));

    expect(screen.getByTestId("mobile-task-location")).toHaveTextContent(
      `/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/lines/${REFUND_LINE_HOTFIX_ID}`,
    );
  });

  it("does not open the first line while a refreshed task is loading", async () => {
    const user = userEvent.setup();
    renderTask(`/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/task/42`);

    const back = screen.getByTestId("mobile-task-back");
    expect(back).toBeDisabled();
    await user.click(back);

    expect(screen.queryByTestId("mobile-task-location")).toBeNull();
    expect(screen.getByText("Loading task…")).toBeTruthy();
  });

  it("returns to the task dispatch line after a refresh", async () => {
    useWorkOrder.mockReturnValue({
      data: {
        id: "wo-42",
        number: "42",
        title: "Fix refund rounding",
        lineDispatches: [
          { id: "dispatch-hotfix", createdAt: "2026-09-28T00:00:00.000Z", line: { id: REFUND_LINE_HOTFIX_ID } },
        ],
      },
      isLoading: false,
      isError: false,
    });
    const user = userEvent.setup();
    renderTask(`/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/task/42`);

    await user.click(screen.getByTestId("mobile-task-back"));

    expect(screen.getByTestId("mobile-task-location")).toHaveTextContent(
      `/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/lines/${REFUND_LINE_HOTFIX_ID}`,
    );
  });

  it("ignores a deleted line id and uses the task dispatch line", async () => {
    useWorkOrder.mockReturnValue({
      data: {
        id: "wo-42",
        number: "42",
        title: "Fix refund rounding",
        lineDispatches: [
          { id: "dispatch-plan", createdAt: "2026-09-28T00:00:00.000Z", line: { id: REFUND_LINE_PLAN_ID } },
        ],
      },
      isLoading: false,
      isError: false,
    });
    const user = userEvent.setup();
    renderTask(`/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/task/42?lineId=deleted-line`);

    await user.click(screen.getByTestId("mobile-task-back"));

    expect(screen.getByTestId("mobile-task-location")).toHaveTextContent(
      `/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/lines/${REFUND_LINE_PLAN_ID}`,
    );
  });
});
