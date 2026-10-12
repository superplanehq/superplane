import { fireEvent, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useLocation } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import type { FactoriesFactoryIntake, FactoriesFactoryPrFeedbackHandler, FactoriesWorkOrder } from "@/api-client";

import {
  GITHUB_ISSUES_INTAKE,
  GITHUB_ISSUES_INTAKE_ID,
  PRIMARY_FACTORY_ROUTE_SEGMENT,
  REFUND_FACTORY,
  REFUND_FACTORY_APPS,
  REFUND_LINE_HOTFIX_ID,
  REFUND_LINE_PLAN_ID,
} from "../__fixtures__/factoryPageResponses";
import { DRAFT_WORK_ORDER, createMobileBoardRenderer, stubElementHeights } from "./mobileBoardPageSpecHarness";

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
const factoryAutomations = vi.fn(() => REFUND_FACTORY_APPS);

vi.mock("@/contexts/useAccount", () => ({ useAccount: () => ({ account: { id: "account-1", name: "Ada" } }) }));
const openCreateWorkOrder = vi.fn();
const importIntakeItem = vi.fn(async () => ({ id: "imported-task", number: "47", title: "Imported issue" }));

function LocationProbe() {
  const location = useLocation();
  return <div data-testid="mobile-test-location">{`${location.pathname}${location.search}`}</div>;
}

const { boardElement, renderBoard } = createMobileBoardRenderer(openCreateWorkOrder, LocationProbe);

vi.mock("@/hooks/useFactoryData", () => ({
  useFactoryBoardWorkOrders: () => ({
    workOrders: boardWorkOrders(),
    isLoading: false,
    isPlaceholderData: false,
    backlog: backlogPage(),
    open: openPage(),
    done: donePage(),
  }),
  useFactoryAutomations: () => ({ data: factoryAutomations() }),
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

vi.mock("../pages/PlanningSettingsHost", () => ({
  PlanningSettingsHost: ({ onClose }: { onClose: () => void }) => (
    <div data-testid="planning-settings">
      <button onClick={onClose}>Close analysis</button>
    </div>
  ),
}));

vi.mock("../pages/ColumnAutomationViewPopup", () => ({
  ColumnAutomationViewHost: ({ canvasId, onClose }: { canvasId: string; onClose: () => void }) => (
    <div data-testid="column-automation-settings">
      {canvasId}
      <button onClick={onClose}>Close automation</button>
    </div>
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
    factoryAutomations.mockReturnValue([
      ...REFUND_FACTORY_APPS,
      { id: "analysis-app", name: "Ingest" },
      { id: "confidence-app", name: "Merge confidence", columnKey: "verify" },
      { id: "custom-app", name: "Notify team", columnKey: "done" },
    ]);
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

  it("shows repair and paused labels for backlog intakes", () => {
    factoryIntakes.mockReturnValue([
      GITHUB_ISSUES_INTAKE,
      {
        id: "intake-sentry",
        name: "Sentry exceptions",
        source: "SOURCE_SENTRY_EXCEPTIONS",
        healthy: false,
        paused: true,
      },
    ]);
    renderBoard(REFUND_FACTORY, true);

    const healthy = screen.getByRole("link", { name: /GitHub issues/i });
    expect(healthy).not.toHaveTextContent("Needs repair");
    expect(healthy).not.toHaveTextContent("Paused");

    const stopped = screen.getByRole("link", { name: /Sentry exceptions/i });
    expect(stopped).toHaveTextContent("Needs repair");
    expect(stopped).toHaveTextContent("Paused");
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
    expect(screen.queryByRole("tablist", { name: "Board columns" })).not.toBeInTheDocument();
  });

  it("opens task analysis settings from More and returns to the board on close", async () => {
    const user = userEvent.setup();
    renderBoard(REFUND_FACTORY, true);

    await user.click(screen.getByRole("link", { name: /Task analysis/i }));

    expect(screen.getByTestId("planning-settings")).toBeInTheDocument();
    expect(screen.queryByRole("tablist", { name: "Board columns" })).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Close analysis" }));
    expect(screen.queryByTestId("planning-settings")).not.toBeInTheDocument();
    expect(screen.getByRole("tablist", { name: "Board columns" })).toBeInTheDocument();
  });

  it.each([
    ["Refund Implementer", "app-refund-implementer"],
    ["Merge confidence", "confidence-app"],
    ["PR Closure", "app-pr-closure"],
    ["Notify team", "custom-app"],
  ])("opens %s settings from More and returns to the board on close", async (name, canvasId) => {
    const user = userEvent.setup();
    renderBoard(REFUND_FACTORY, true);

    await user.click(screen.getByRole("link", { name: new RegExp(name) }));

    expect(screen.getByTestId("column-automation-settings")).toHaveTextContent(canvasId);
    expect(screen.queryByRole("tablist", { name: "Board columns" })).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Close automation" }));
    expect(screen.queryByTestId("column-automation-settings")).not.toBeInTheDocument();
    expect(screen.getByRole("tablist", { name: "Board columns" })).toBeInTheDocument();
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
      ...idleBoardPage(),
      hasNextPage: true,
      fetchNextPage: fetchBacklog,
    });
    donePage.mockReturnValue({
      ...idleBoardPage(),
      hasNextPage: true,
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
    boardWorkOrders.mockReturnValue([DRAFT_WORK_ORDER]);
    backlogPage.mockReturnValue({
      ...idleBoardPage(),
      hasNextPage: true,
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
      boardWorkOrders.mockReturnValue([DRAFT_WORK_ORDER]);
      backlogPage.mockReturnValue({
        ...idleBoardPage(),
        hasNextPage: true,
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
      ...idleBoardPage(),
      hasNextPage: true,
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
      boardWorkOrders.mockReturnValue([DRAFT_WORK_ORDER]);
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
    boardWorkOrders.mockReturnValue([DRAFT_WORK_ORDER]);
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
    boardWorkOrders.mockReturnValue([DRAFT_WORK_ORDER]);
    backlogPage.mockReturnValue({ ...idleBoardPage(), totalCount: 80 });
    donePage.mockReturnValue({ ...idleBoardPage(), totalCount: 60 });
    renderBoard();

    expect(screen.getByTestId("mobile-board-tab-backlog")).toHaveTextContent("Backlog80");
    expect(screen.getByTestId("mobile-board-tab-done")).toHaveTextContent("Done60");
  });

  it("falls back to loaded card count when search filter is active", async () => {
    const user = userEvent.setup();
    boardWorkOrders.mockReturnValue([DRAFT_WORK_ORDER]);
    backlogPage.mockReturnValue({ ...idleBoardPage(), totalCount: 80 });
    renderBoard();

    expect(screen.getByTestId("mobile-board-tab-backlog")).toHaveTextContent("Backlog80");

    await user.click(screen.getByTestId("mobile-board-search-toggle"));
    const searchInput = screen.getByTestId("mobile-board-search-input");
    await user.type(searchInput, "refund");

    expect(screen.getByTestId("mobile-board-tab-backlog")).toHaveTextContent("Backlog1");
  });
});
