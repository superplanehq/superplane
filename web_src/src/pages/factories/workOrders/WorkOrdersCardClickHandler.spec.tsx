import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { createMemoryRouter, RouterProvider } from "react-router";
import { describe, expect, it, vi } from "vitest";

import type { FactoriesFactory, FactoriesWorkOrder } from "@/api-client";

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
    executions: [{ id: "e1", step: "verify", state: "STATE_STARTED", line: { id: "line-a", name: "hotfix" } }],
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

function renderView(Component: (typeof views)[number]["Component"]) {
  const onDispatch = vi.fn().mockResolvedValue(undefined);
  const onAssigneesSave = vi.fn().mockResolvedValue(undefined);

  const router = createMemoryRouter([
    {
      path: "/",
      element: (
        <Component
          entries={[entry]}
          organizationId={organizationId}
          factoryKey={factoryKey}
          factoryLines={[]}
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
  const row = screen.getByRole("link", { name: `Open ${entry.title}` }).closest("article") as HTMLElement;

  return { router, onDispatch, onAssigneesSave, row };
}

describe("WorkOrdersBoardView layout", () => {
  it("uses a horizontal kanban row", () => {
    renderView(WorkOrdersBoardView);

    expect(screen.getByTestId("work-orders-board").className).toContain("overflow-x-auto");
    expect(screen.getByTestId("work-orders-board-lane-running").className).toContain("min-w-72");
    expect(screen.getByTestId("work-orders-board-lane-running").className).toContain("shrink-0");
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

  it("keeps the dispatch button and assignee control clickable", () => {
    const { row } = renderView(Component);

    expect(effectivePointerEvents(within(row).getByTestId(`work-order-row-dispatch-${entry.id}`))).toBe("auto");
    expect(effectivePointerEvents(within(row).getByTestId(`work-order-row-assignees-${entry.id}`))).toBe("auto");
  });

  it("navigates to the detail page when the overlay link is activated", async () => {
    const user = userEvent.setup();
    const { router, row } = renderView(Component);

    await user.click(within(row).getByRole("link", { name: `Open ${entry.title}` }));

    expect(router.state.location.pathname).toBe(detailHref);
  });

  it("does not navigate when the dispatch button is clicked", async () => {
    const user = userEvent.setup();
    const { router, onDispatch, row } = renderView(Component);

    await user.click(within(row).getByTestId(`work-order-row-dispatch-${entry.id}`));

    expect(router.state.location.pathname).toBe("/");
    expect(onDispatch).not.toHaveBeenCalled();
  });

  it("does not navigate when the assignee control is clicked", async () => {
    const user = userEvent.setup();
    const { router, row } = renderView(Component);

    await user.click(within(row).getByTestId(`work-order-row-assignees-${entry.id}`));

    expect(router.state.location.pathname).toBe("/");
  });
});
