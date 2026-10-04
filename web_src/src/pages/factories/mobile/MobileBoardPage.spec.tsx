import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes, useLocation } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import type { FactoriesFactoryIntake, FactoriesWorkOrder } from "@/api-client";
import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";

import {
  GITHUB_ISSUES_INTAKE,
  GITHUB_ISSUES_INTAKE_ID,
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
  PRIMARY_FACTORY_ROUTE_SEGMENT,
  REFUND_FACTORY,
  REFUND_LINE_PLAN_ID,
} from "../__fixtures__/factoryPageResponses";
import { FactoriesLayoutContext } from "../layout/factoriesLayoutContext";
import { MobileBoardPage } from "./MobileBoardPage";

const idleBoardPage = () => ({ hasNextPage: false, isFetchingNextPage: false, fetchNextPage: vi.fn() });
const boardWorkOrders = vi.fn((): FactoriesWorkOrder[] => []);
const factoryIntakes = vi.fn((): FactoriesFactoryIntake[] => []);

vi.mock("@/hooks/useFactoryData", () => ({
  useFactoryBoardWorkOrders: () => ({
    workOrders: boardWorkOrders(),
    isLoading: false,
    isPlaceholderData: false,
    backlog: idleBoardPage(),
    open: idleBoardPage(),
    done: idleBoardPage(),
  }),
  useFactoryAutomations: () => ({ data: [] }),
  useDispatchWorkOrder: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateWorkOrderAssignees: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

vi.mock("@/hooks/useFactoryIntakeData", () => ({
  useFactoryIntakes: () => ({ data: factoryIntakes() }),
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

vi.mock("@/hooks/usePageTitle", () => ({
  usePageTitle: () => undefined,
}));

function LocationProbe() {
  const location = useLocation();
  return <div data-testid="mobile-test-location">{`${location.pathname}${location.search}`}</div>;
}

function renderBoard() {
  return render(
    <QueryClientProvider client={new QueryClient()}>
      <ThemeProvider>
        <TooltipProvider>
          <MemoryRouter
            initialEntries={[`/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/lines/${REFUND_LINE_PLAN_ID}`]}
          >
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
                <Route path="/org-1/workspaces/:factoryKey/lines/:lineId" element={<MobileBoardPage />} />
                <Route path="/org-1/workspaces/:factoryKey/task/:orderNumber" element={<div>Task page</div>} />
              </Routes>
              <LocationProbe />
            </FactoriesLayoutContext.Provider>
          </MemoryRouter>
        </TooltipProvider>
      </ThemeProvider>
    </QueryClientProvider>,
  );
}

describe("MobileBoardPage", () => {
  beforeEach(() => {
    window.localStorage.clear();
    boardWorkOrders.mockReturnValue([]);
    factoryIntakes.mockReturnValue([]);
  });

  it("renders every board column as its own full-width snap section with a tab", () => {
    renderBoard();

    const track = screen.getByTestId("mobile-board-track");
    const sections = within(track).getAllByRole("region", { hidden: true });
    expect(sections.map((section) => section.getAttribute("data-testid"))).toEqual([
      "mobile-board-column-backlog",
      "mobile-board-column-phase-0",
      "mobile-board-column-phase-1",
      "mobile-board-column-verify",
      "mobile-board-column-done",
    ]);
    expect(screen.getByTestId("mobile-board-tab-backlog")).toHaveTextContent("Backlog");
    expect(screen.getByTestId("mobile-board-tab-verify")).toHaveTextContent("Verify");
    expect(screen.getByTestId("mobile-board-tab-backlog")).toHaveAttribute("aria-selected", "true");
    expect(screen.getByTestId("mobile-board-tab-done")).toHaveAttribute("aria-selected", "false");
    expect(screen.getByTestId("mobile-board-column-done")).toHaveTextContent("No tasks in Done.");
  });

  it("moves the selected column when a tab is tapped", async () => {
    const user = userEvent.setup();
    renderBoard();

    await user.click(screen.getByTestId("mobile-board-tab-verify"));

    expect(screen.getByTestId("mobile-board-tab-verify")).toHaveAttribute("aria-selected", "true");
    expect(screen.getByTestId("mobile-board-tab-backlog")).toHaveAttribute("aria-selected", "false");
    expect(screen.getByTestId("mobile-board-column-backlog")).toHaveAttribute("aria-hidden", "true");
    expect(screen.getByTestId("mobile-board-column-verify")).not.toHaveAttribute("aria-hidden");
  });

  it("keeps intakes out of the board and lists them under the three-dots menu", async () => {
    const user = userEvent.setup();
    factoryIntakes.mockReturnValue([GITHUB_ISSUES_INTAKE]);
    renderBoard();

    expect(screen.queryByTestId(`mobile-board-intake-${GITHUB_ISSUES_INTAKE_ID}`)).not.toBeInTheDocument();

    await user.click(screen.getByTestId("mobile-board-menu"));
    await user.click(await screen.findByTestId(`mobile-board-intake-${GITHUB_ISSUES_INTAKE_ID}`));

    expect(screen.getByTestId("mobile-test-location")).toHaveTextContent(
      `/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/lines/${REFUND_LINE_PLAN_ID}?intake=1&intakeId=${GITHUB_ISSUES_INTAKE_ID}`,
    );
  });

  it("opens search as a full-width row from the top bar", async () => {
    const user = userEvent.setup();
    renderBoard();

    expect(screen.queryByTestId("mobile-board-search-input")).not.toBeInTheDocument();

    await user.click(screen.getByTestId("mobile-board-search-toggle"));

    expect(screen.getByTestId("mobile-board-search-input")).toBeInTheDocument();
  });

  it("opens a task as a full-screen route with the board line in the URL", async () => {
    const user = userEvent.setup();
    boardWorkOrders.mockReturnValue([
      {
        id: "wo-draft",
        number: "42",
        title: "Fix refund rounding",
        state: "STATE_DRAFT",
        lineDispatches: [],
      },
    ]);
    renderBoard();

    await user.click(
      within(screen.getByTestId("mobile-board-column-backlog")).getByRole("button", {
        name: "Open Fix refund rounding",
      }),
    );

    expect(screen.getByTestId("mobile-test-location")).toHaveTextContent(
      `/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/task/42?lineId=${REFUND_LINE_PLAN_ID}`,
    );
  });
});
