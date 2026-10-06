import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes, useLocation } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import type { FactoriesFactoryIntake, FactoriesFactoryPrFeedbackHandler, FactoriesWorkOrder } from "@/api-client";
import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";

import {
  GITHUB_ISSUES_INTAKE,
  GITHUB_ISSUES_INTAKE_ID,
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
  PRIMARY_FACTORY_ROUTE_SEGMENT,
  REFUND_FACTORY,
  REFUND_LINE_HOTFIX_ID,
  REFUND_LINE_PLAN_ID,
} from "../__fixtures__/factoryPageResponses";
import { FactoriesLayoutContext } from "../layout/factoriesLayoutContext";
import { MobileBoardPage } from "./MobileBoardPage";

const idleBoardPage = () => ({ hasNextPage: false, isFetchingNextPage: false, fetchNextPage: vi.fn() });
const boardWorkOrders = vi.fn((): FactoriesWorkOrder[] => []);
const backlogPage = vi.fn(idleBoardPage);
const openPage = vi.fn(idleBoardPage);
const donePage = vi.fn(idleBoardPage);
const factoryIntakes = vi.fn((): FactoriesFactoryIntake[] => []);
const prFeedbackHandlers = vi.fn((): FactoriesFactoryPrFeedbackHandler[] => []);

vi.mock("@/hooks/useFactoryData", () => ({
  useFactoryBoardWorkOrders: () => ({
    workOrders: boardWorkOrders(),
    isLoading: false,
    isPlaceholderData: false,
    backlog: backlogPage(),
    open: openPage(),
    done: donePage(),
  }),
  useFactoryWorkOrdersPage: () => ({
    orders: [],
    isLoading: false,
    isPlaceholderData: false,
    hasNextPage: false,
    fetchNextPage: vi.fn(),
    isFetchingNextPage: false,
    isFetchNextPageError: false,
  }),
  useFactoryAutomations: () => ({ data: [] }),
  useDispatchWorkOrder: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateWorkOrderAssignees: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

vi.mock("@/hooks/useFactoryIntakeData", () => ({
  useFactoryIntakes: () => ({ data: factoryIntakes() }),
}));

vi.mock("@/hooks/useFactoryPRFeedbackData", () => ({
  useFactoryPRFeedbackHandlers: () => ({ data: prFeedbackHandlers() }),
}));

vi.mock("../pages/IntakeSettingsHost", () => ({
  IntakeSettingsHost: ({ intake }: { intake: { intakeId: string } }) => (
    <div data-testid="intake-source-settings">{intake.intakeId}</div>
  ),
}));

vi.mock("../pages/PRFeedbackSettingsHost", () => ({
  PRFeedbackSettingsHost: ({ handlerId }: { handlerId?: string | null }) => (
    <div data-testid="pr-feedback-settings">{handlerId}</div>
  ),
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

function LocationProbe() {
  const location = useLocation();
  return <div data-testid="mobile-test-location">{`${location.pathname}${location.search}`}</div>;
}

function renderBoard(factory = REFUND_FACTORY) {
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
                factory,
                factories: [factory],
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
    backlogPage.mockReset();
    openPage.mockReset();
    donePage.mockReset();
    backlogPage.mockImplementation(idleBoardPage);
    openPage.mockImplementation(idleBoardPage);
    donePage.mockImplementation(idleBoardPage);
    factoryIntakes.mockReturnValue([]);
    prFeedbackHandlers.mockReturnValue([]);
  });

  it("sets the tab title from the workspace name, not the line name", () => {
    renderBoard({ ...REFUND_FACTORY, name: "SuperPlane Prod" });

    expect(document.title).toBe("SuperPlane Prod · SuperPlane");
    expect(document.title).not.toContain("Plan and Implement");
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
    expect(screen.getByTestId("mobile-board-column-backlog")).toHaveAttribute("inert");
    expect(screen.getByTestId("mobile-board-column-verify")).not.toHaveAttribute("aria-hidden");
    expect(screen.getByTestId("mobile-board-column-verify")).not.toHaveAttribute("inert");
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
    expect(screen.getByTestId("intake-source-settings")).toHaveTextContent(GITHUB_ISSUES_INTAKE_ID);
  });

  it("opens pull request feedback settings for the selected listener", async () => {
    const user = userEvent.setup();
    prFeedbackHandlers.mockReturnValue([{ id: "handler-checks", source: "SOURCE_PULL_REQUEST_CHECKS", healthy: true }]);
    renderBoard();

    await user.click(screen.getByTestId("mobile-board-menu"));
    await user.click(await screen.findByTestId("mobile-board-listener-handler-checks"));

    expect(screen.getByTestId("pr-feedback-settings")).toHaveTextContent("handler-checks");
  });

  it("switches to another line from the board header", async () => {
    const user = userEvent.setup();
    renderBoard();

    await user.click(screen.getByTestId("mobile-board-line-switcher"));
    await user.click(await screen.findByTestId(`mobile-board-line-${REFUND_LINE_HOTFIX_ID}`));

    expect(screen.getByTestId("mobile-test-location")).toHaveTextContent(
      `/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/lines/${REFUND_LINE_HOTFIX_ID}`,
    );
    expect(screen.getByTestId("mobile-board-line-switcher")).toHaveTextContent("Hotfix");
  });

  it("loads the next page when a column has no cards yet", async () => {
    const user = userEvent.setup();
    const fetchNextPage = vi.fn();
    donePage.mockReturnValue({ hasNextPage: true, isFetchingNextPage: false, fetchNextPage });
    renderBoard();

    await user.click(screen.getByTestId("mobile-board-tab-done"));
    const done = screen.getByTestId("mobile-board-column-done");
    expect(done).not.toHaveTextContent("No tasks in Done.");
    await user.click(within(done).getByRole("button", { name: "Load more" }));

    expect(fetchNextPage).toHaveBeenCalledTimes(1);
  });

  it("offers load more when the visible cards do not fill the column", () => {
    boardWorkOrders.mockReturnValue([
      {
        id: "wo-draft",
        number: "42",
        title: "Fix refund rounding",
        state: "STATE_DRAFT",
        lineDispatches: [],
      },
    ]);
    backlogPage.mockReturnValue({ hasNextPage: true, isFetchingNextPage: false, fetchNextPage: vi.fn() });
    renderBoard();

    const backlog = screen.getByTestId("mobile-board-column-backlog");
    expect(within(backlog).getByRole("button", { name: "Open Fix refund rounding" })).toBeInTheDocument();
    expect(within(backlog).getByRole("button", { name: "Load more" })).toBeInTheDocument();
  });

  it("opens the closed-task dialog from a status that needs it", async () => {
    const user = userEvent.setup();
    renderBoard();

    await user.click(screen.getByTestId("work-orders-filter-trigger"));
    await user.hover(screen.getByTestId("work-orders-filter-statuses"));
    fireEvent.click(await screen.findByTestId("work-orders-filter-statuses-rejected"));

    expect(await screen.findByTestId("work-order-closed-status-dialog")).toBeInTheDocument();
  });

  it("opens search as a full-width row from the top bar", async () => {
    const user = userEvent.setup();
    renderBoard();

    expect(screen.queryByTestId("mobile-board-search-input")).not.toBeInTheDocument();

    await user.click(screen.getByTestId("mobile-board-search-toggle"));

    expect(screen.getByTestId("mobile-board-search-input")).toBeInTheDocument();
  });

  it("opens a task without a line id in the URL", async () => {
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

    const location = screen.getByTestId("mobile-test-location");
    expect(location).toHaveTextContent(`/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/task/42`);
    expect(location).not.toHaveTextContent("lineId=");
  });
});
