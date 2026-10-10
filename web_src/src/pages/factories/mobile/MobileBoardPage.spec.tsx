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
import { MobileMorePage } from "./MobileMorePage";
import { MobileBoardPage } from "./MobileBoardPage";

const idleBoardPage = (): {
  hasNextPage: boolean;
  isFetchingNextPage: boolean;
  isFetchNextPageError: boolean;
  fetchNextPage: () => void;
  totalCount?: number;
} => ({ hasNextPage: false, isFetchingNextPage: false, isFetchNextPageError: false, fetchNextPage: vi.fn() });
const boardWorkOrders = vi.fn((): FactoriesWorkOrder[] => []);
const backlogPage = vi.fn(idleBoardPage);
const openPage = vi.fn(idleBoardPage);
const donePage = vi.fn(idleBoardPage);
const factoryIntakes = vi.fn((): FactoriesFactoryIntake[] => []);
const prFeedbackHandlers = vi.fn((): FactoriesFactoryPrFeedbackHandler[] => []);

vi.mock("@/contexts/useAccount", () => ({ useAccount: () => ({ account: { id: "account-1", name: "Ada" } }) }));
const openCreateWorkOrder = vi.fn();
const importIntakeItem = vi.fn(async () => ({ id: "imported-task", number: "47", title: "Imported issue" }));

vi.mock("@/hooks/useFactoryData", () => ({
  useFactoryBoardWorkOrders: () => ({
    workOrders: boardWorkOrders(),
    isLoading: false,
    isPlaceholderData: false,
    backlog: backlogPage(),
    open: openPage(),
    done: donePage(),
  }),
  useFactoryAutomations: () => ({ data: [] }),
  useCreateWorkOrder: () => ({ mutateAsync: vi.fn() }),
  useDispatchWorkOrder: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateWorkOrderAssignees: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

vi.mock("@/hooks/useFactoryIntakeData", () => ({
  useFactoryIntakes: () => ({ data: factoryIntakes() }),
  useImportFactoryIntakeItem: () => ({ mutateAsync: importIntakeItem }),
  useSearchFactoryIntakeItems: () => ({
    data: [{ id: "12", key: "#12", title: "Imported issue" }],
    isLoading: false,
    isFetching: false,
  }),
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

function stubElementHeights(measurements: { scrollHeight: number; clientHeight: number }) {
  for (const [name, value] of Object.entries(measurements)) {
    Object.defineProperty(HTMLElement.prototype, name, { configurable: true, value });
  }
  return () => {
    for (const name of Object.keys(measurements)) Reflect.deleteProperty(HTMLElement.prototype, name);
  };
}

function LocationProbe() {
  const location = useLocation();
  return <div data-testid="mobile-test-location">{`${location.pathname}${location.search}`}</div>;
}

function renderBoard(factory = REFUND_FACTORY, more = false) {
  return render(boardElement(factory, more));
}

function boardElement(factory = REFUND_FACTORY, more = false) {
  return (
    <QueryClientProvider client={new QueryClient()}>
      <ThemeProvider>
        <TooltipProvider>
          <MemoryRouter
            initialEntries={[
              more
                ? `/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/more?lineId=${REFUND_LINE_PLAN_ID}`
                : `/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/lines/${REFUND_LINE_PLAN_ID}`,
            ]}
          >
            <FactoriesLayoutContext.Provider
              value={{
                organizationId: "org-1",
                factoryId: PRIMARY_FACTORY_ID,
                factoryKey: PRIMARY_FACTORY_KEY,
                routeSegment: PRIMARY_FACTORY_ROUTE_SEGMENT,
                factory,
                factories: [factory],
                openCreateWorkOrder,
              }}
            >
              <Routes>
                <Route path="/org-1/workspaces/:factoryKey/lines/:lineId" element={<MobileBoardPage />} />
                <Route path="/org-1/workspaces/:factoryKey/more" element={<MobileMorePage />} />
                <Route path="/org-1/workspaces/:factoryKey/task/:orderNumber" element={<div>Task page</div>} />
              </Routes>
              <LocationProbe />
            </FactoriesLayoutContext.Provider>
          </MemoryRouter>
        </TooltipProvider>
      </ThemeProvider>
    </QueryClientProvider>
  );
}

describe("MobileBoardPage", () => {
  beforeEach(() => {
    window.localStorage.clear();
    openCreateWorkOrder.mockClear();
    importIntakeItem.mockClear();
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

  it("creates a task from the empty Backlog without a duplicate action when Done has tasks", async () => {
    boardWorkOrders.mockReturnValue([
      {
        id: "done-task",
        number: "43",
        title: "Done task",
        state: "STATE_CLOSED",
        result: "RESULT_COMPLETED",
        lineDispatches: [{ id: "done-dispatch", line: { id: REFUND_LINE_PLAN_ID } }],
      },
    ]);
    renderBoard();
    expect(screen.getAllByRole("button", { name: "Create task" })).toHaveLength(1);
    await userEvent
      .setup()
      .click(within(screen.getByTestId("mobile-board-column-backlog")).getByRole("button", { name: "Create task" }));
    await userEvent.setup().click(screen.getByRole("button", { name: "Create task manually" }));
    expect(openCreateWorkOrder).toHaveBeenCalledTimes(1);
  });

  it("imports an intake issue through Create task and opens the imported task", async () => {
    const user = userEvent.setup();
    factoryIntakes.mockReturnValue([GITHUB_ISSUES_INTAKE]);
    renderBoard();

    await user.click(screen.getByRole("button", { name: "Create task" }));
    expect(screen.getByRole("textbox", { name: "Import from GitHub issue" })).toBeInTheDocument();
    await user.click(await screen.findByTestId("lines-backlog-create-item-12"));

    expect(importIntakeItem).toHaveBeenCalledWith({ intakeId: GITHUB_ISSUES_INTAKE_ID, itemId: "12" });
    expect(await screen.findByText("Task page")).toBeInTheDocument();
    expect(screen.getByTestId("mobile-test-location")).toHaveTextContent("/task/47");
    expect(openCreateWorkOrder).not.toHaveBeenCalled();
  });

  it("saves appearance and board colors from More", async () => {
    const user = userEvent.setup();
    renderBoard(REFUND_FACTORY, true);
    expect(screen.queryByRole("button", { name: "Create task" })).not.toBeInTheDocument();
    await user.click(screen.getByRole("combobox", { name: "Appearance" }));
    await user.click(await screen.findByRole("option", { name: "Dark" }));
    expect(document.documentElement).toHaveClass("dark");
    await user.click(screen.getByRole("combobox", { name: "Column colors" }));
    await user.click(await screen.findByRole("option", { name: "Off" }));
    expect(window.localStorage.getItem("factories-column-color-view")).toBe("off");
    expect(screen.getByRole("link", { name: "Settings" })).toHaveAttribute(
      "href",
      `/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/settings`,
    );
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

  it("opens intake settings from More", async () => {
    const user = userEvent.setup();
    factoryIntakes.mockReturnValue([GITHUB_ISSUES_INTAKE]);
    renderBoard(REFUND_FACTORY, true);

    await user.click(screen.getByRole("link", { name: /GitHub issues/i }));

    expect(screen.getByTestId("mobile-test-location")).toHaveTextContent(
      `/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/lines/${REFUND_LINE_PLAN_ID}?intake=1&intakeId=${GITHUB_ISSUES_INTAKE_ID}`,
    );
    expect(screen.getByTestId("intake-source-settings")).toHaveTextContent(GITHUB_ISSUES_INTAKE_ID);
    expect(screen.queryByRole("tablist", { name: "Board columns" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Search tasks" })).not.toBeInTheDocument();
  });

  it("opens pull request feedback settings for the selected listener", async () => {
    const user = userEvent.setup();
    prFeedbackHandlers.mockReturnValue([{ id: "handler-checks", source: "SOURCE_PULL_REQUEST_CHECKS", healthy: true }]);
    renderBoard(REFUND_FACTORY, true);

    await user.click(screen.getByRole("link", { name: /pull request checks/i }));

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

  it("loads the next page of a visible empty column and leaves hidden columns alone", async () => {
    const user = userEvent.setup();
    const fetchDone = vi.fn();
    const fetchBacklog = vi.fn();
    backlogPage.mockReturnValue({
      hasNextPage: true,
      isFetchingNextPage: false,
      isFetchNextPageError: false,
      fetchNextPage: fetchBacklog,
    });
    donePage.mockReturnValue({
      hasNextPage: true,
      isFetchingNextPage: false,
      isFetchNextPageError: false,
      fetchNextPage: fetchDone,
    });
    renderBoard();

    const done = screen.getByTestId("mobile-board-column-done");
    expect(done).not.toHaveTextContent("No tasks in Done.");
    expect(fetchBacklog).toHaveBeenCalledTimes(1);
    expect(fetchDone).not.toHaveBeenCalled();
    expect(screen.queryByRole("button", { name: "Load more", hidden: true })).not.toBeInTheDocument();

    fireEvent.scroll(screen.getByTestId("mobile-board-column-scroll-done"));
    expect(fetchDone).not.toHaveBeenCalled();

    await user.click(screen.getByTestId("mobile-board-tab-done"));

    expect(fetchDone).toHaveBeenCalledTimes(1);
    expect(fetchBacklog).toHaveBeenCalledTimes(1);
  });

  it("loads the next page when the visible list does not fill the screen", () => {
    const fetchNextPage = vi.fn();
    boardWorkOrders.mockReturnValue([
      {
        id: "wo-draft",
        number: "42",
        title: "Fix refund rounding",
        state: "STATE_DRAFT",
        lineDispatches: [],
      },
    ]);
    backlogPage.mockReturnValue({
      hasNextPage: true,
      isFetchingNextPage: false,
      isFetchNextPageError: false,
      fetchNextPage,
    });
    renderBoard();

    const backlog = screen.getByTestId("mobile-board-column-backlog");
    expect(within(backlog).getByRole("button", { name: "Open Fix refund rounding" })).toBeInTheDocument();
    expect(within(backlog).queryByRole("button", { name: "Load more" })).not.toBeInTheDocument();
    expect(fetchNextPage).toHaveBeenCalledTimes(1);
  });

  it("loads the next page when the visible column is scrolled within 160px of the end", () => {
    const restoreHeights = stubElementHeights({ scrollHeight: 1000, clientHeight: 400 });
    try {
      const fetchNextPage = vi.fn();
      boardWorkOrders.mockReturnValue([
        {
          id: "wo-draft",
          number: "42",
          title: "Fix refund rounding",
          state: "STATE_DRAFT",
          lineDispatches: [],
        },
      ]);
      backlogPage.mockReturnValue({
        hasNextPage: true,
        isFetchingNextPage: false,
        isFetchNextPageError: false,
        fetchNextPage,
      });
      renderBoard();

      const list = screen.getByTestId("mobile-board-column-scroll-backlog");
      expect(fetchNextPage).not.toHaveBeenCalled();

      list.scrollTop = 439;
      fireEvent.scroll(list);
      expect(fetchNextPage).not.toHaveBeenCalled();

      list.scrollTop = 440;
      fireEvent.scroll(list);
      expect(fetchNextPage).toHaveBeenCalledTimes(1);

      fireEvent.scroll(list);
      expect(fetchNextPage).toHaveBeenCalledTimes(1);
    } finally {
      restoreHeights();
    }
  });

  it("shows a loading spinner only while the next page loads", () => {
    const fetchNextPage = vi.fn();
    backlogPage.mockReturnValue({
      hasNextPage: true,
      isFetchingNextPage: true,
      isFetchNextPageError: false,
      fetchNextPage,
    });
    const view = renderBoard();

    const backlog = screen.getByTestId("mobile-board-column-backlog");
    const spinner = within(backlog).getByRole("status", { name: "Loading more" });
    const list = within(backlog).getByTestId("mobile-board-column-scroll-backlog");
    expect(spinner.tagName).not.toBe("BUTTON");
    expect(list.compareDocumentPosition(spinner) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(within(backlog).queryByRole("button", { name: "Load more" })).not.toBeInTheDocument();
    expect(fetchNextPage).not.toHaveBeenCalled();

    backlogPage.mockReturnValue({
      hasNextPage: true,
      isFetchingNextPage: false,
      isFetchNextPageError: false,
      fetchNextPage,
    });
    view.rerender(boardElement());

    expect(screen.queryByRole("status", { name: "Loading more" })).not.toBeInTheDocument();
    expect(fetchNextPage).toHaveBeenCalledTimes(1);
  });

  it("stops automatic loads after a page error until the user retries", async () => {
    const user = userEvent.setup();
    const fetchNextPage = vi.fn();
    const observers: ResizeObserverCallback[] = [];
    const originalObserver = globalThis.ResizeObserver;
    class ImmediateResizeObserver {
      constructor(callback: ResizeObserverCallback) {
        observers.push(callback);
      }
      observe() {
        observers.at(-1)?.([], this as unknown as ResizeObserver);
      }
      unobserve() {}
      disconnect() {}
    }
    globalThis.ResizeObserver = ImmediateResizeObserver as unknown as typeof ResizeObserver;

    try {
      boardWorkOrders.mockReturnValue([
        {
          id: "wo-draft",
          number: "42",
          title: "Fix refund rounding",
          state: "STATE_DRAFT",
          lineDispatches: [],
        },
      ]);
      backlogPage.mockReturnValue({
        hasNextPage: true,
        isFetchingNextPage: false,
        isFetchNextPageError: true,
        fetchNextPage,
      });
      const view = renderBoard();
      const backlog = screen.getByTestId("mobile-board-column-backlog");

      expect(fetchNextPage).not.toHaveBeenCalled();
      expect(within(backlog).getByRole("alert")).toHaveTextContent("SuperPlane could not load more tasks.");
      expect(within(backlog).queryByRole("status", { name: "Loading more" })).not.toBeInTheDocument();

      fireEvent.scroll(within(backlog).getByTestId("mobile-board-column-scroll-backlog"));
      observers.at(-1)?.([], {} as ResizeObserver);
      expect(fetchNextPage).not.toHaveBeenCalled();

      await user.click(within(backlog).getByRole("button", { name: "Try again" }));
      expect(fetchNextPage).toHaveBeenCalledTimes(1);

      view.rerender(boardElement());
      observers.at(-1)?.([], {} as ResizeObserver);
      expect(fetchNextPage).toHaveBeenCalledTimes(1);
    } finally {
      globalThis.ResizeObserver = originalObserver;
    }
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

  it("displays server total counts on column tabs instead of loaded card counts", () => {
    boardWorkOrders.mockReturnValue([
      {
        id: "wo-draft-1",
        number: "42",
        title: "Fix refund rounding",
        state: "STATE_DRAFT",
        lineDispatches: [],
      },
    ]);
    backlogPage.mockReturnValue({
      ...idleBoardPage(),
      totalCount: 80,
    });
    donePage.mockReturnValue({
      ...idleBoardPage(),
      totalCount: 60,
    });
    renderBoard();

    expect(screen.getByTestId("mobile-board-tab-backlog")).toHaveTextContent("Backlog80");
    expect(screen.getByTestId("mobile-board-tab-done")).toHaveTextContent("Done60");
  });

  it("falls back to loaded card count when search filter is active", async () => {
    const user = userEvent.setup();
    boardWorkOrders.mockReturnValue([
      {
        id: "wo-draft-1",
        number: "42",
        title: "Fix refund rounding",
        state: "STATE_DRAFT",
        lineDispatches: [],
      },
    ]);
    backlogPage.mockReturnValue({
      ...idleBoardPage(),
      totalCount: 80,
    });
    renderBoard();

    expect(screen.getByTestId("mobile-board-tab-backlog")).toHaveTextContent("Backlog80");

    await user.click(screen.getByTestId("mobile-board-search-toggle"));
    const searchInput = screen.getByTestId("mobile-board-search-input");
    await user.type(searchInput, "refund");

    expect(screen.getByTestId("mobile-board-tab-backlog")).toHaveTextContent("Backlog1");
  });
});
