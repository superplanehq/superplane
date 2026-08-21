import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { createMemoryRouter, RouterProvider } from "react-router";
import { describe, expect, it, vi } from "vitest";

import type { FactoriesFactory, FactoriesFactoryLine, FactoriesWorkOrder } from "@/api-client";

import { workOrderDetailPath } from "../lib/factoryPagePaths";
import { buildWorkOrderListEntry } from "../lib/workOrderListModel";
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

const detailHref = workOrderDetailPath(organizationId, factoryKey, "1");

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
          isDispatching={false}
          isAssigneesSaving={false}
          onDispatch={onDispatch}
          onAssigneesSave={onAssigneesSave}
        />
      ),
    },
    { path: detailHref, element: <div>Work order detail</div> },
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

  it("shows one owner on the card even when two people are listed", () => {
    const twoOwners = buildWorkOrderListEntry(
      {
        id: "wo-1",
        number: "1",
        title: "Reconcile refund batch",
        state: "STATE_OPEN",
        createdAt: "2024-06-01T00:00:00Z",
        updatedAt: "2024-06-02T00:00:00Z",
        lineDispatches: entry.order.lineDispatches,
        assignees: [
          { id: "user-1", name: "Ada Lovelace" },
          { id: "user-2", name: "Grace Hopper" },
        ],
      },
      factory,
    );

    const { row } = renderView(WorkOrdersBoardView, [twoOwners]);

    expect(within(row).queryByText("+1")).not.toBeInTheDocument();
    expect(within(row).queryByLabelText(/more owners/i)).not.toBeInTheDocument();
    expect(within(row).getByTestId(`work-order-row-assignees-${twoOwners.id}`)).toBeInTheDocument();
  });

  it("reveals a right-quarter Start arrow on a draft backlog card", () => {
    const draft = buildWorkOrderListEntry(
      {
        id: "wo-draft",
        number: "5",
        title: "Draft: rework refund telemetry",
        state: "STATE_DRAFT",
        createdAt: "2024-06-01T00:00:00Z",
        updatedAt: "2024-06-02T00:00:00Z",
        lineDispatches: [],
        assignees: [{ id: "user-1", name: "Ada Lovelace" }],
      },
      factory,
    );

    const { row } = renderView(WorkOrdersBoardView, [draft], {
      factoryLines: [{ id: "line-a", name: "hotfix" }],
    });

    const hoverActions = within(row).getByTestId("work-order-card-hover-actions-wo-draft");
    expect(hoverActions).toHaveClass("opacity-0");
    expect(hoverActions.className).toMatch(/group-hover:opacity-100/);
    expect(hoverActions.className).toMatch(/w-1\/4/);
    expect(within(hoverActions).getByRole("button", { name: "Start" })).toBeInTheDocument();
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

    const hoverActions = within(row).getByTestId("work-order-card-hover-actions-wo-draft");
    expect(effectivePointerEvents(hoverActions)).toBe("none");

    await user.click(within(hoverActions).getByRole("button", { name: "Start" }));

    expect(router.state.location.pathname).toBe("/");
    expect(onDispatch).toHaveBeenCalledWith("wo-draft", { lineName: "plan-and-implement" });
  });
});

describe.each(views)("$name click handling", ({ Component }) => {
  it("lets clicks on the title and status badge pass through to the overlay link", () => {
    const { row } = renderView(Component);
    const link = within(row).getByRole("link", { name: `Open ${entry.title}` });

    expect(effectivePointerEvents(link)).toBe("auto");
    expect(effectivePointerEvents(within(row).getByText(entry.title))).toBe("none");
    expect(effectivePointerEvents(within(row).getByText("Running"))).toBe("none");
  });

  it("keeps the assignee control clickable", () => {
    const { row } = renderView(Component);

    expect(effectivePointerEvents(within(row).getByTestId(`work-order-row-assignees-${entry.id}`))).toBe("auto");
  });

  it("navigates to the detail page when the overlay link is activated", async () => {
    const user = userEvent.setup();
    const { router, row } = renderView(Component);

    await user.click(within(row).getByRole("link", { name: `Open ${entry.title}` }));

    expect(router.state.location.pathname).toBe(detailHref);
  });

  it("does not navigate when the assignee control is clicked", async () => {
    const user = userEvent.setup();
    const { router, row } = renderView(Component);

    await user.click(within(row).getByTestId(`work-order-row-assignees-${entry.id}`));

    expect(router.state.location.pathname).toBe("/");
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
