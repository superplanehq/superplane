import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { createMemoryRouter, MemoryRouter, RouterProvider } from "react-router";
import { describe, expect, it, vi } from "vitest";

import type { FactoriesFactory, FactoriesFactoryLine, FactoriesWorkOrder } from "@/api-client";

import { workOrderDetailPath } from "../lib/factoryPagePaths";
import { buildWorkOrderListEntry } from "../lib/workOrderListModel";
import { WorkOrderCard } from "./WorkOrderCard";
import { WorkOrdersBoardView } from "./WorkOrdersBoardView";
import { WorkOrdersListView } from "./WorkOrdersListView";
import { WorkOrdersTableView } from "./WorkOrdersTableView";

const organizationId = "org-1";
const factoryKey = "RF";
const factory: FactoriesFactory = { id: "factory-1", name: "Refunds", key: factoryKey };

const entry = buildWorkOrderListEntry(
  {
    id: "wo-1",
    number: "1",
    title: "Reconcile refund batch",
    state: "STATE_OPEN",
    createdAt: "2024-06-01T00:00:00Z",
    updatedAt: "2024-06-02T00:00:00Z",
    lineDispatches: [
      {
        id: "dispatch-1",
        line: { id: "line-a", name: "hotfix" },
        state: "STATE_ACTIVE",
        stepExecutions: [{ id: "e1", step: "verify", state: "STATE_STARTED" }],
      },
    ],
    assignees: [{ id: "user-1", name: "Ada Lovelace" }],
  } satisfies FactoriesWorkOrder,
  factory,
);

const permalinkHref = workOrderDetailPath(organizationId, factoryKey, entry.order.number ?? "1");

/**
 * Board and list views group entries into lanes based on display status.
 * "wo-1" has an in-progress execution, so it displays as "Running" and
 * lands in the "running" lane, which both layouts always render.
 */
const views = [
  { name: "WorkOrdersBoardView", Component: WorkOrdersBoardView },
  { name: "WorkOrdersListView", Component: WorkOrdersListView },
  { name: "WorkOrdersTableView", Component: WorkOrdersTableView },
] as const;

const viewsWithDispatch = [
  { name: "WorkOrdersListView", Component: WorkOrdersListView },
  { name: "WorkOrdersTableView", Component: WorkOrdersTableView },
] as const;

const viewsWithAssignee = viewsWithDispatch;

/**
 * jsdom doesn't lay out elements or perform coordinate-based hit-testing,
 * so `userEvent.click()` always fires directly on the node it's given —
 * it can't reproduce the browser behavior of a click "falling through" a
 * `pointer-events: none` element to whatever is stacked underneath it.
 * The regression this ticket fixes is exactly that CSS-driven behavior, so
 * the reliable way to guard it here is to assert on the *effective*
 * `pointer-events` value (walking up from the element, since an explicit
 * class on a closer ancestor overrides one further away) rather than by
 * trying to simulate a real click-through.
 */
function effectivePointerEvents(element: Element): "auto" | "none" {
  let node: Element | null = element;
  while (node) {
    const classes = node.className.toString().split(/\s+/);
    if (classes.includes("pointer-events-none")) return "none";
    if (classes.includes("pointer-events-auto")) return "auto";
    node = node.parentElement;
  }
  return "auto";
}

function renderView(
  Component: (typeof views)[number]["Component"],
  listEntries: (typeof entry)[] = [entry],
  extras: { factoryLines?: FactoriesFactoryLine[]; preferredLineName?: string } = {},
) {
  const onDispatch = vi.fn().mockResolvedValue(undefined);
  const onAssigneesSave = vi.fn().mockResolvedValue(undefined);

  const router = createMemoryRouter([
    {
      path: "/",
      element: (
        <Component
          entries={listEntries}
          organizationId={organizationId}
          factoryKey={factoryKey}
          factoryLines={extras.factoryLines ?? []}
          preferredLineName={extras.preferredLineName}
          canDispatch={true}
          canAssign={true}
          dispatchingOrderIds={new Set()}
          isAssigneesSaving={false}
          onDispatch={onDispatch}
          onAssigneesSave={onAssigneesSave}
        />
      ),
    },
    { path: permalinkHref, element: <div>Work order</div> },
  ]);

  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });

  render(
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>,
  );

  // Scope queries to the row/card itself: board and list also render lane
  // headers whose text ("Running", etc.) can collide with the badge text.
  const first = listEntries[0];
  const row = screen.getByRole("link", { name: `Open ${first.title}` }).closest("article") as HTMLElement;

  return { router, onDispatch, onAssigneesSave, row };
}

describe("WorkOrdersBoardView layout", () => {
  it("uses a horizontal kanban row", () => {
    renderView(WorkOrdersBoardView);

    expect(screen.getByTestId("work-orders-board").className).toContain("overflow-x-auto");
    expect(screen.getByTestId("work-orders-board-lane-running").className).toContain("min-w-72");
    expect(screen.getByTestId("work-orders-board-lane-running").className).toContain("shrink-0");
  });

  it("does not render a send-to-line control on the card", () => {
    const { row } = renderView(WorkOrdersBoardView);

    expect(within(row).queryByTestId(`work-order-row-dispatch-${entry.id}`)).not.toBeInTheDocument();
    expect(within(row).queryByRole("button", { name: "Dispatch to line" })).not.toBeInTheDocument();
    expect(within(row).queryByRole("button", { name: "Start" })).not.toBeInTheDocument();
  });

  it("shows a status dot, title, start time, and owner", () => {
    const { row } = renderView(WorkOrdersBoardView);

    expect(within(row).getByLabelText("Running")).toBeInTheDocument();
    expect(within(row).queryByText("Running")).not.toBeInTheDocument();
    expect(within(row).getByText(entry.title)).toBeInTheDocument();
    expect(within(row).getByText(/\d+[smhd] ago$/)).toBeInTheDocument();
    const owner = within(row).getByTestId(`work-order-row-assignees-${entry.id}`);
    expect(owner).toBeInTheDocument();
    expect(within(row).queryByRole("button", { name: "Change owner" })).not.toBeInTheDocument();
    expect(effectivePointerEvents(owner)).toBe("none");
    expect(within(row).queryByText(entry.displayKey)).not.toBeInTheDocument();
    expect(within(row).queryByText(/verify/i)).not.toBeInTheDocument();
  });

  it("shows a Run failed label on a waiting card after a failed step", () => {
    const waiting = buildWorkOrderListEntry(
      {
        id: "wo-waiting",
        number: "6",
        title: "Ship idempotent refund retries",
        state: "STATE_OPEN",
        createdAt: "2024-06-01T00:00:00Z",
        updatedAt: "2024-06-02T00:00:00Z",
        lineDispatches: [
          {
            id: "dispatch-1",
            line: { id: "line-a", name: "hotfix" },
            state: "STATE_FINISHED",
            stepExecutions: [{ id: "e1", step: "implement", state: "STATE_FINISHED", result: "RESULT_FAILED" }],
          },
        ],
        assignees: [{ id: "user-2", name: "Arnold Schwarzenegger" }],
      },
      factory,
    );

    const { row } = renderView(WorkOrdersBoardView, [waiting]);

    const chip = within(row).getByText("Run failed");
    expect(chip.querySelector("svg")).toBeTruthy();
    expect(within(row).queryByRole("button", { name: "Start" })).not.toBeInTheDocument();
  });

  it("shows a Start button on a draft backlog card", () => {
    const draft = buildWorkOrderListEntry(
      {
        id: "wo-draft",
        number: "5",
        title: "Draft: rework refund telemetry",
        state: "STATE_DRAFT",
        createdAt: "2024-06-01T00:00:00Z",
        updatedAt: "2024-06-02T00:00:00Z",
        lineDispatches: [],
        assignees: [],
      },
      factory,
    );

    const { row } = renderView(WorkOrdersBoardView, [draft], {
      factoryLines: [{ id: "line-a", name: "hotfix" }],
    });

    const start = within(row).getByRole("button", { name: "Start" });
    expect(start).toBeInTheDocument();
    expect(effectivePointerEvents(start)).toBe("auto");
    expect(within(row).queryByTestId("work-order-row-assignees-wo-draft")).not.toBeInTheDocument();
  });

  it("starts a draft on the preferred line without opening the card", async () => {
    const user = userEvent.setup();
    const draft = buildWorkOrderListEntry(
      {
        id: "wo-draft",
        number: "5",
        title: "Draft: rework refund telemetry",
        state: "STATE_DRAFT",
        createdAt: "2024-06-01T00:00:00Z",
        updatedAt: "2024-06-02T00:00:00Z",
        lineDispatches: [],
      },
      factory,
    );

    const { router, onDispatch, row } = renderView(WorkOrdersBoardView, [draft], {
      factoryLines: [
        { id: "line-a", name: "plan-and-implement" },
        { id: "line-b", name: "hotfix" },
      ],
      preferredLineName: "plan-and-implement",
    });

    await user.click(within(row).getByRole("button", { name: "Start" }));

    expect(router.state.location.pathname).toBe("/");
    expect(onDispatch).toHaveBeenCalledWith("wo-draft", { lineName: "plan-and-implement" });
  });
});

describe.each(views)("$name click handling", ({ Component }) => {
  it("lets clicks on the title and status pass through to the overlay link", () => {
    const { row } = renderView(Component);
    const link = within(row).getByRole("link", { name: `Open ${entry.title}` });
    const status =
      Component === WorkOrdersBoardView ? within(row).getByLabelText("Running") : within(row).getByText("Running");

    expect(effectivePointerEvents(link)).toBe("auto");
    expect(effectivePointerEvents(within(row).getByText(entry.title))).toBe("none");
    expect(effectivePointerEvents(status)).toBe("none");
  });

  it("navigates to the work order permalink when the overlay link is activated", async () => {
    const user = userEvent.setup();
    const { router, row } = renderView(Component);

    await user.click(within(row).getByRole("link", { name: `Open ${entry.title}` }));

    expect(router.state.location.pathname).toBe(permalinkHref);
  });
});

describe.each(viewsWithAssignee)("$name assignee control", ({ Component }) => {
  it("shows the owner without a change control", () => {
    const { row } = renderView(Component);

    expect(within(row).getByTestId(`work-order-row-assignees-${entry.id}`)).toBeInTheDocument();
    expect(within(row).queryByRole("button", { name: "Change owner" })).not.toBeInTheDocument();
    expect(effectivePointerEvents(within(row).getByTestId(`work-order-row-assignees-${entry.id}`))).toBe("none");
  });
});

describe.each(viewsWithDispatch)("$name dispatch control", ({ Component }) => {
  it("keeps the dispatch button clickable and does not navigate when it is clicked", async () => {
    const user = userEvent.setup();
    const { router, onDispatch, row } = renderView(Component);

    const dispatchButton = within(row).getByTestId(`work-order-row-dispatch-${entry.id}`);
    expect(effectivePointerEvents(dispatchButton)).toBe("auto");

    await user.click(dispatchButton);

    expect(router.state.location.pathname).toBe("/");
    expect(onDispatch).not.toHaveBeenCalled();
  });
});

describe("WorkOrderCard scores", () => {
  it("shows a score and a Start button to the right of the score", () => {
    const draft = buildWorkOrderListEntry(
      {
        id: "wo-draft-scored",
        number: "842",
        title: "Add retry handling to webhook delivery",
        state: "STATE_DRAFT",
        createdAt: "2024-06-01T00:00:00Z",
        updatedAt: "2024-06-02T00:00:00Z",
        lineDispatches: [],
        assignees: [],
      },
      factory,
    );

    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <MemoryRouter>
          <WorkOrderCard
            entry={draft}
            organizationId={organizationId}
            factoryKey={factoryKey}
            factoryLines={[{ id: "line-a", name: "hotfix" }]}
            canDispatch
            canAssign
            dispatchingOrderIds={new Set()}
            isAssigneesSaving={false}
            onDispatch={vi.fn()}
            onAssigneesSave={vi.fn()}
            confidenceScore={5}
            onOpen={vi.fn()}
          />
        </MemoryRouter>
      </QueryClientProvider>,
    );

    const score = screen.getByTestId("work-order-card-score-wo-draft-scored");
    expect(score).toHaveAttribute("aria-valuenow", "5");
    expect(score).toHaveAttribute("aria-valuemax", "5");
    expect(score.querySelectorAll("[data-filled='true']")).toHaveLength(5);
    expect(score.querySelectorAll("[data-filled='false']")).toHaveLength(0);
    const start = screen.getByRole("button", { name: "Start" });
    expect(start).toBeInTheDocument();
    expect(score.compareDocumentPosition(start) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  });

  it("shows the check name and score when the bars are hovered", async () => {
    const user = userEvent.setup();
    const draft = buildWorkOrderListEntry(
      {
        id: "wo-draft-scored",
        number: "842",
        title: "Add retry handling to webhook delivery",
        state: "STATE_DRAFT",
        createdAt: "2024-06-01T00:00:00Z",
        updatedAt: "2024-06-02T00:00:00Z",
        lineDispatches: [],
        assignees: [],
      },
      factory,
    );

    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <MemoryRouter>
          <WorkOrderCard
            entry={draft}
            organizationId={organizationId}
            factoryKey={factoryKey}
            factoryLines={[{ id: "line-a", name: "hotfix" }]}
            canDispatch
            canAssign
            dispatchingOrderIds={new Set()}
            isAssigneesSaving={false}
            onDispatch={vi.fn()}
            onAssigneesSave={vi.fn()}
            confidenceScore={4}
            onOpen={vi.fn()}
          />
        </MemoryRouter>
      </QueryClientProvider>,
    );

    const score = screen.getByTestId("work-order-card-score-wo-draft-scored");
    expect(effectivePointerEvents(score)).toBe("auto");

    await user.hover(score);

    const tip = await screen.findByRole("tooltip");
    expect(tip).toHaveTextContent("Confidence score");
    expect(tip).toHaveTextContent("4/5");
  });
});

describe("WorkOrderCard attention", () => {
  const waitingOrder: FactoriesWorkOrder = {
    id: "wo-waiting",
    number: "12",
    title: "Ship refund retries",
    state: "STATE_OPEN",
    createdAt: "2024-06-01T00:00:00Z",
    updatedAt: "2024-06-02T00:00:00Z",
    statusNotes: [{ key: "pr-closure", headline: "Waiting for user review", body: "Tag the agent." }],
    lineDispatches: [],
    assignees: [],
  };

  const cardProps = {
    organizationId,
    factoryKey,
    factoryLines: [{ id: "line-a", name: "hotfix" }] as FactoriesFactoryLine[],
    canDispatch: true,
    canAssign: true,
    dispatchingOrderIds: new Set<string>(),
    isAssigneesSaving: false,
    onDispatch: vi.fn(),
    onAssigneesSave: vi.fn(),
    onOpen: vi.fn(),
  };

  it("shows Waiting for user review when the work order has a status note", () => {
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <MemoryRouter>
          <WorkOrderCard entry={buildWorkOrderListEntry(waitingOrder, factory)} {...cardProps} />
        </MemoryRouter>
      </QueryClientProvider>,
    );

    expect(screen.getByText("Waiting for user review")).toBeInTheDocument();
    expect(screen.queryByText("Addressing user feedback")).not.toBeInTheDocument();
  });

  it("shows Addressing user feedback when a PR-feedback run is active", () => {
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <MemoryRouter>
          <WorkOrderCard
            entry={buildWorkOrderListEntry(waitingOrder, factory)}
            {...cardProps}
            addressingFeedbackOrderIds={new Set(["wo-waiting"])}
          />
        </MemoryRouter>
      </QueryClientProvider>,
    );

    expect(screen.getByText("Addressing user feedback")).toBeInTheDocument();
    expect(screen.queryByText("Waiting for user review")).not.toBeInTheDocument();
  });
});
