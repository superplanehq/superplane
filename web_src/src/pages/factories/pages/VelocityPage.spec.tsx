import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";

import type { FactoriesDescribeFactoryVelocityResponse, FactoriesFactory, FactoriesWorkOrder } from "@/api-client";
import type { FactoryVelocityParams } from "@/hooks/useFactoryVelocity";
import { TooltipProvider } from "@/ui/tooltip";

import { PRIMARY_FACTORY_ID, PRIMARY_FACTORY_KEY, REFUND_FACTORY } from "../__fixtures__/factoryPageResponses";
import { PEOPLE_FIRST_PAGE_SIZE, PEOPLE_LOAD_MORE_SIZE, peoplePageSizeForOffset } from "../lib/velocityPeopleSort";
import { FactoriesLayoutContext } from "../layout/factoriesLayoutContext";
import { VelocityPage } from "./VelocityPage";

type VelocityPerson = NonNullable<FactoriesDescribeFactoryVelocityResponse["people"]>[number];

interface VelocityHookState {
  data?: FactoriesDescribeFactoryVelocityResponse;
  isLoading?: boolean;
  isFetching?: boolean;
  error?: Error | null;
  /**
   * The whole People cohort behind the mock's paging, sorted the way the
   * backend would return it. Defaults to `data.people` when unset, which is
   * enough for tests with fewer than one page of people.
   */
  allPeople?: VelocityPerson[];
  /**
   * Holds the report of the previous offset, the way the query does while the
   * next page loads.
   */
  holdsPreviousReport?: boolean;
}

interface WorkOrdersHookState {
  data?: FactoriesWorkOrder[];
  isLoading?: boolean;
  isFetching?: boolean;
  error?: Error | null;
}

const velocityHookState: VelocityHookState = {};
const workOrdersHookState: WorkOrdersHookState = {};
/** Every call the page made to `useFactoryVelocity`, newest last. */
const velocityHookCalls: FactoryVelocityParams[] = [];

/** Workspace setup picks the GitHub integration and the app repository. */
const FACTORY_WITH_SETUP_REPO: FactoriesFactory = {
  ...REFUND_FACTORY,
  onboarding: { ...REFUND_FACTORY.onboarding, vcsIntegrationId: "int-1", appRepository: "acme/api" },
};

const startSync = vi.fn();
const syncHookState: { isPending?: boolean } = {};

/** A cohort large enough to exercise paging, already in default sort order. */
function manyPeople(count: number): VelocityPerson[] {
  return Array.from({ length: count }, (_, index) => ({
    id: `person-${index + 1}`,
    name: `Contributor ${String(index + 1).padStart(2, "0")}`,
    email: `contributor${index + 1}@example.com`,
    authoredMerged: 1,
    factoryMerged: 0,
    factoryWaste: 0,
    medianCycleHours: 0,
    costCents: "0",
  }));
}

vi.mock("@/hooks/useFactoryVelocity", () => ({
  useFactoryVelocity: (_organizationId: string, _factoryId: string, params: FactoryVelocityParams) => {
    velocityHookCalls.push(params);

    const base = velocityHookState.data;
    const allPeople = velocityHookState.allPeople ?? base?.people ?? [];
    const holdsPreviousReport = velocityHookState.holdsPreviousReport ?? false;
    const offset = holdsPreviousReport ? 0 : (params.peopleOffset ?? 0);
    const pageSize = holdsPreviousReport
      ? PEOPLE_FIRST_PAGE_SIZE
      : (params.peoplePageSize ?? peoplePageSizeForOffset(offset));
    const page = allPeople.slice(offset, offset + pageSize);

    return {
      data: base
        ? {
            ...base,
            people: page,
            peopleTotal: allPeople.length,
            peopleHasMore: offset + page.length < allPeople.length,
          }
        : undefined,
      isLoading: velocityHookState.isLoading ?? false,
      isFetching: velocityHookState.isFetching ?? false,
      isPlaceholderData: holdsPreviousReport,
      error: velocityHookState.error ?? null,
      refetch: vi.fn(),
    };
  },
  useSyncFactoryVelocity: () => ({
    mutate: startSync,
    isPending: syncHookState.isPending ?? false,
  }),
}));

vi.mock("@/hooks/useFactoryData", () => ({
  useFactoryWorkOrders: () => ({
    data: workOrdersHookState.data ?? [],
    isLoading: workOrdersHookState.isLoading ?? false,
    isFetching: workOrdersHookState.isFetching ?? false,
    error: workOrdersHookState.error ?? null,
  }),
}));

/** A window with output, so the page renders the report instead of a state card. */
function populatedResponse(
  overrides: Partial<FactoriesDescribeFactoryVelocityResponse> = {},
): FactoriesDescribeFactoryVelocityResponse {
  return {
    yesterday: { superplaneMerged: 3, waste: 1 },
    totals: {
      superplaneMerged: 12,
      peopleMerged: 8,
      waste: 4,
      superplaneSharePct: 60,
      wastePct: 25,
      costCents: "4200",
      tokens: "185000",
      wasteCostCents: "900",
      tasksClosed: 16,
      tasksWaste: 4,
    },
    points: [
      { day: "1", superplaneMerged: 2, peopleMerged: 1, waste: 1, costCents: "800", tokens: "20000" },
      { day: "2", superplaneMerged: 3, peopleMerged: 2, waste: 0, costCents: "1200", tokens: "30000" },
    ],
    hasPeopleCohort: true,
    ...overrides,
  };
}

function renderShell(factory: FactoriesFactory = REFUND_FACTORY) {
  return render(
    <QueryClientProvider client={new QueryClient()}>
      <TooltipProvider delayDuration={0}>
        <MemoryRouter initialEntries={["/velocity"]}>
          <FactoriesLayoutContext.Provider
            value={{
              organizationId: "org-1",
              factoryId: PRIMARY_FACTORY_ID,
              factoryKey: PRIMARY_FACTORY_KEY,
              factory,
              factories: [factory],
              openCreateWorkOrder: vi.fn(),
            }}
          >
            <VelocityPage />
          </FactoriesLayoutContext.Provider>
        </MemoryRouter>
      </TooltipProvider>
    </QueryClientProvider>,
  );
}

function resetState() {
  startSync.mockClear();
  syncHookState.isPending = false;
  velocityHookState.data = undefined;
  velocityHookState.isLoading = false;
  velocityHookState.isFetching = false;
  velocityHookState.error = null;
  velocityHookState.allPeople = undefined;
  velocityHookState.holdsPreviousReport = false;
  velocityHookCalls.length = 0;
  workOrdersHookState.data = [];
  workOrdersHookState.isLoading = false;
  workOrdersHookState.isFetching = false;
  workOrdersHookState.error = null;
}

describe("VelocityPage shell", () => {
  it("sets the document title from the page and workspace name", () => {
    resetState();
    renderShell();

    expect(document.title).toBe(`Velocity · ${REFUND_FACTORY.name} · SuperPlane`);
  });

  it("shows the loading state while velocity is loading", () => {
    resetState();
    velocityHookState.isLoading = true;

    renderShell();

    expect(screen.getByTestId("velocity-loading-state")).toBeInTheDocument();
    expect(screen.queryByTestId("velocity-summary")).not.toBeInTheDocument();
  });

  it("shows an error state with retry when velocity fails to load", () => {
    resetState();
    velocityHookState.error = new Error("network");

    renderShell();

    expect(screen.getByTestId("velocity-error-state")).toBeInTheDocument();
    expect(screen.getByTestId("velocity-error-retry")).toBeInTheDocument();
    expect(screen.queryByTestId("velocity-summary")).not.toBeInTheDocument();
  });

  // Merge counts come from a background sync, so a user who just merged
  // something needs a way to ask for a fresh read.
  it("starts a sync when the workspace has a repository", async () => {
    resetState();
    velocityHookState.data = populatedResponse();

    renderShell(FACTORY_WITH_SETUP_REPO);
    await userEvent.click(screen.getByTestId("velocity-overflow-menu"));
    await userEvent.click(screen.getByRole("menuitem", { name: "Refresh data" }));

    expect(startSync).toHaveBeenCalledTimes(1);
  });

  // A sync rebuilds sixty days of history, which takes long enough that the
  // page has to say the work is running.
  it("shows progress while a sync runs, and keeps the report readable", () => {
    resetState();
    velocityHookState.data = populatedResponse();
    syncHookState.isPending = true;

    renderShell(FACTORY_WITH_SETUP_REPO);

    expect(screen.getByTestId("velocity-sync-progress")).toBeInTheDocument();
    expect(screen.getByTestId("velocity-summary")).toBeInTheDocument();
  });

  it("shows no progress bar when no sync is running", () => {
    resetState();
    velocityHookState.data = populatedResponse();

    renderShell(FACTORY_WITH_SETUP_REPO);

    expect(screen.queryByTestId("velocity-sync-progress")).not.toBeInTheDocument();
  });

  it("hides the overflow menu when there is no repository to read", () => {
    resetState();
    velocityHookState.data = populatedResponse();

    renderShell();

    expect(screen.queryByTestId("velocity-overflow-menu")).not.toBeInTheDocument();
  });

  it("keeps the report when a refetch fails and cached data remains", () => {
    resetState();
    velocityHookState.data = populatedResponse();
    velocityHookState.error = new Error("network");

    renderShell();

    expect(screen.getByTestId("velocity-summary")).toBeInTheDocument();
    expect(screen.queryByTestId("velocity-error-state")).not.toBeInTheDocument();
  });

  it("shows the zero state when the window holds no output", () => {
    resetState();
    velocityHookState.data = {
      yesterday: { superplaneMerged: 0, waste: 0 },
      totals: { superplaneMerged: 0, peopleMerged: 0, waste: 0, superplaneSharePct: 0, wastePct: 0 },
      points: [{ day: "1", superplaneMerged: 0, peopleMerged: 0, waste: 0 }],
      hasPeopleCohort: false,
    };

    renderShell();

    expect(screen.getByTestId("velocity-zero-state")).toBeInTheDocument();
    expect(screen.queryByTestId("velocity-summary")).not.toBeInTheDocument();
  });

  it("names the repository from workspace setup in the header", () => {
    resetState();
    velocityHookState.data = populatedResponse();

    renderShell(FACTORY_WITH_SETUP_REPO);

    expect(screen.getByTestId("workspace-page-header-subtitle")).toHaveTextContent("acme/api");
  });

  it("leads with the tasks that closed and the share of them that wasted", () => {
    resetState();
    velocityHookState.data = populatedResponse();

    renderShell();

    const summary = screen.getByTestId("velocity-summary");
    expect(summary).toHaveTextContent("Tasks closed");
    expect(summary).toHaveTextContent("16");
    expect(summary).toHaveTextContent("Task waste");
    expect(summary).toHaveTextContent("25%");
    expect(summary).not.toHaveTextContent("4 tasks closed without a merge");
    expect(screen.getByRole("button", { name: "About Tasks closed" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "About Task waste" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "About Median cycle time" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "About Cost per task" })).toBeInTheDocument();
  });

  it("hides the metric explanations behind info tooltips", async () => {
    resetState();
    velocityHookState.data = populatedResponse();
    const user = userEvent.setup();

    renderShell();

    expect(screen.queryByRole("tooltip")).not.toBeInTheDocument();

    await user.hover(screen.getByRole("button", { name: "About Task waste" }));

    expect(await screen.findByRole("tooltip")).toHaveTextContent("4 tasks closed without a merge");
  });

  it("spreads tracked spend over the tasks that closed", () => {
    resetState();
    velocityHookState.data = populatedResponse();

    renderShell();

    const summary = screen.getByTestId("velocity-summary");
    expect(summary).toHaveTextContent("Cost per task");
    // $42.00 of spend over 16 closed tasks.
    expect(summary).toHaveTextContent("$2.63");
  });

  it("compares against the previous window when it holds output", () => {
    resetState();
    velocityHookState.data = populatedResponse({
      hasPreviousWindow: true,
      previousTotals: {
        superplaneMerged: 6,
        peopleMerged: 6,
        waste: 6,
        superplaneSharePct: 50,
        wastePct: 50,
        costCents: "3000",
        tasksClosed: 12,
        tasksWaste: 6,
      },
    });

    renderShell();

    const summary = screen.getByTestId("velocity-summary");
    expect(summary).toHaveTextContent("Compared with the previous 14 days");
    // Waste fell from 50% to 25% of the tasks that closed.
    expect(summary).toHaveTextContent("25 pp");
  });

  it("drops the comparison for a workspace without an earlier period", () => {
    resetState();
    velocityHookState.data = populatedResponse({ hasPreviousWindow: false });

    renderShell();

    const summary = screen.getByTestId("velocity-summary");
    expect(summary).toHaveTextContent("There is no earlier period to compare with yet.");
    expect(summary).not.toHaveTextContent("No change");
  });

  it("lists people with their authored and SuperPlane merges", () => {
    resetState();
    velocityHookState.data = populatedResponse({
      people: [
        {
          id: "user-1",
          name: "Ada Lovelace",
          email: "ada@example.com",
          authoredMerged: 5,
          factoryMerged: 3,
          factoryWaste: 1,
          medianCycleHours: 12,
          costCents: "1500",
        },
      ],
    });

    renderShell();

    const people = screen.getByTestId("velocity-people");
    expect(people).toHaveTextContent("Ada Lovelace");
    expect(people).toHaveTextContent("1 person with activity in this period");
  });

  it("shows only the first page of people and the true total, with a Show more control", () => {
    resetState();
    velocityHookState.data = populatedResponse();
    velocityHookState.allPeople = manyPeople(12);

    renderShell();

    const people = screen.getByTestId("velocity-people");
    expect(people).toHaveTextContent("12 people with activity in this period");
    expect(within(people).getAllByRole("row")).toHaveLength(1 + PEOPLE_FIRST_PAGE_SIZE);
    expect(within(people).getByText("Contributor 01", { selector: "p" })).toBeInTheDocument();
    expect(within(people).queryByText("Contributor 06", { selector: "p" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: /show more/i })).toBeInTheDocument();
  });

  it("keeps the report and the rows already shown while the next page loads", async () => {
    resetState();
    velocityHookState.data = populatedResponse();
    velocityHookState.allPeople = manyPeople(12);
    const user = userEvent.setup();

    renderShell();

    velocityHookState.holdsPreviousReport = true;
    velocityHookState.isFetching = true;
    await user.click(screen.getByRole("button", { name: /show more/i }));

    const people = screen.getByTestId("velocity-people");
    expect(screen.queryByTestId("velocity-loading-state")).not.toBeInTheDocument();
    expect(within(people).getAllByRole("row")).toHaveLength(1 + PEOPLE_FIRST_PAGE_SIZE);
  });

  it("loads more people, keeping ranks sequential, and hides the control once every row is shown", async () => {
    resetState();
    velocityHookState.data = populatedResponse();
    velocityHookState.allPeople = manyPeople(12);
    const user = userEvent.setup();

    renderShell();
    await user.click(screen.getByRole("button", { name: /show more/i }));

    const people = screen.getByTestId("velocity-people");
    expect(within(people).getByText("Contributor 12", { selector: "p" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /show more/i })).not.toBeInTheDocument();
    expect(velocityHookCalls.at(-1)).toMatchObject({
      peopleOffset: PEOPLE_FIRST_PAGE_SIZE,
      peoplePageSize: PEOPLE_LOAD_MORE_SIZE,
    });

    const ranks = within(people)
      .getAllByRole("row")
      .slice(1)
      .map((row) => within(row).getAllByRole("cell")[0]?.textContent);
    expect(ranks).toEqual(Array.from({ length: 12 }, (_, index) => String(index + 1)));
  });

  it("asks for 20 more people on each Show more click after the first page", async () => {
    resetState();
    velocityHookState.data = populatedResponse();
    velocityHookState.allPeople = manyPeople(50);
    const user = userEvent.setup();

    renderShell();
    await user.click(screen.getByRole("button", { name: /show more/i }));
    expect(velocityHookCalls.at(-1)).toMatchObject({
      peopleOffset: PEOPLE_FIRST_PAGE_SIZE,
      peoplePageSize: PEOPLE_LOAD_MORE_SIZE,
    });

    await user.click(screen.getByRole("button", { name: /show more/i }));
    expect(velocityHookCalls.at(-1)).toMatchObject({
      peopleOffset: PEOPLE_FIRST_PAGE_SIZE + PEOPLE_LOAD_MORE_SIZE,
      peoplePageSize: PEOPLE_LOAD_MORE_SIZE,
    });

    const people = screen.getByTestId("velocity-people");
    expect(within(people).getAllByRole("row")).toHaveLength(1 + PEOPLE_FIRST_PAGE_SIZE + PEOPLE_LOAD_MORE_SIZE * 2);
    expect(screen.getByRole("button", { name: /show more/i })).toBeInTheDocument();
  });

  it("sorts through the backend and resets paging to the first page", async () => {
    resetState();
    velocityHookState.data = populatedResponse();
    velocityHookState.allPeople = manyPeople(12);
    const user = userEvent.setup();

    renderShell();
    await user.click(screen.getByRole("button", { name: /show more/i }));
    expect(screen.getByTestId("velocity-people")).toHaveTextContent("Contributor 12");

    await user.click(screen.getByRole("button", { name: "Costs" }));

    expect(velocityHookCalls.at(-1)).toMatchObject({
      peopleSort: "costUsd",
      peopleSortDirection: "desc",
      peopleOffset: 0,
    });
    // Paging restarted, so the control is back and the second page is gone.
    expect(screen.getByRole("button", { name: /show more/i })).toBeInTheDocument();
    expect(screen.getByTestId("velocity-people")).not.toHaveTextContent("Contributor 12");

    await user.click(screen.getByRole("button", { name: "Costs" }));
    expect(velocityHookCalls.at(-1)).toMatchObject({ peopleSort: "costUsd", peopleSortDirection: "asc" });
  });

  it("hides Show more while a sort reset still holds the previous report", async () => {
    resetState();
    velocityHookState.data = populatedResponse();
    velocityHookState.allPeople = manyPeople(12);
    const user = userEvent.setup();

    renderShell();
    await user.click(screen.getByRole("button", { name: /show more/i }));
    expect(screen.getByTestId("velocity-people")).toHaveTextContent("Contributor 12");

    velocityHookState.holdsPreviousReport = true;
    velocityHookState.isFetching = true;
    await user.click(screen.getByRole("button", { name: "Costs" }));

    expect(screen.queryByRole("button", { name: /show more/i })).not.toBeInTheDocument();
    expect(velocityHookCalls.at(-1)).toMatchObject({ peopleOffset: 0, peopleSort: "costUsd" });
  });

  it("explains an empty Manual work column when GitHub is not connected", () => {
    resetState();
    velocityHookState.data = populatedResponse({
      hasPeopleCohort: false,
      people: [{ id: "user-1", name: "Ada Lovelace", factoryMerged: 3 }],
    });

    renderShell();

    expect(screen.getByTestId("velocity-people")).toHaveTextContent(
      "Connect GitHub in workspace setup to count the pull requests people created.",
    );
  });

  it("hides the intake split when no intake source produced a merge", () => {
    resetState();
    velocityHookState.data = populatedResponse({ intakeSources: [] });

    renderShell();

    expect(screen.queryByText("Intake source")).not.toBeInTheDocument();
    expect(screen.getByText("Who created")).toBeInTheDocument();
  });

  it("offers the intake split when the response names its sources", () => {
    resetState();
    velocityHookState.data = populatedResponse({
      intakeSources: [{ key: "github-issues", label: "GitHub issue", merged: 7 }],
    });

    renderShell();

    expect(screen.getByText("Intake source")).toBeInTheDocument();
  });

  it("explains when task time could not be loaded", () => {
    resetState();
    velocityHookState.data = populatedResponse();
    workOrdersHookState.error = new Error("network");

    renderShell();

    const taskTime = screen.getByTestId("velocity-task-time");
    expect(taskTime).toHaveTextContent("We could not load task time.");
  });

  it("reports tracked spend split between tokens and compute", () => {
    resetState();
    velocityHookState.data = populatedResponse();

    renderShell();

    const cost = screen.getByTestId("velocity-cost");
    expect(cost).toHaveTextContent("Total cost");
    expect(cost).toHaveTextContent("$42.00");
    expect(cost).toHaveTextContent("Tokens");
    expect(cost).toHaveTextContent("Compute");
  });
});
