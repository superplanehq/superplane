import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "bun:test";

import type { FactoriesFactory, FactoriesWorkOrder } from "@/api-client";

import { buildWorkOrderListEntry } from "../lib/workOrderListModel";
import { WorkOrdersBoardView } from "./WorkOrdersBoardView";

const factory: FactoriesFactory = { id: "factory-1", name: "Refunds", key: "RF" };

function draft(id: string, title: string) {
  return buildWorkOrderListEntry(
    {
      id,
      number: id,
      title,
      state: "STATE_DRAFT",
      createdAt: "2024-06-01T00:00:00Z",
      updatedAt: "2024-06-02T00:00:00Z",
      lineDispatches: [],
      assignees: [],
    } satisfies FactoriesWorkOrder,
    factory,
  );
}

function queuedFirstStep() {
  return buildWorkOrderListEntry(
    {
      id: "wo-queued",
      number: "wo-queued",
      title: "Wait for a free slot",
      state: "STATE_OPEN",
      createdAt: "2024-06-01T00:00:00Z",
      updatedAt: "2024-06-02T00:00:00Z",
      assignees: [],
      lineDispatches: [
        {
          id: "dispatch-queued",
          line: { id: "line-a", name: "hotfix" },
          state: "STATE_ACTIVE",
          createdAt: "2024-06-02T00:00:00Z",
          stepExecutions: [],
          queueItem: {
            id: "q-1",
            stepName: "hotfix",
            stepIndex: 0,
            position: 1,
          },
        },
      ],
    } satisfies FactoriesWorkOrder,
    factory,
  );
}

function renderBoard(dispatchingOrderIds: ReadonlySet<string>, willQueueOnStart = false) {
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <WorkOrdersBoardView
          entries={[draft("wo-1", "Reconcile refund batch"), draft("wo-2", "Handle duplicate charges")]}
          organizationId="org-1"
          factoryKey="RF"
          factoryLines={[{ id: "line-a", name: "hotfix" }]}
          canDispatch
          canAssign
          dispatchingOrderIds={dispatchingOrderIds}
          isAssigneesSaving={false}
          onDispatch={vi.fn().mockResolvedValue(undefined)}
          onAssigneesSave={vi.fn().mockResolvedValue(undefined)}
          willQueueOnStart={willQueueOnStart}
        />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

function renderQueuedBoard(
  overrides: {
    onCancelQueue?: (orderId: string) => Promise<void>;
    cancelingOrderIds?: ReadonlySet<string>;
  } = {},
) {
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <WorkOrdersBoardView
          entries={[queuedFirstStep()]}
          organizationId="org-1"
          factoryKey="RF"
          factoryLines={[{ id: "line-a", name: "hotfix" }]}
          canDispatch
          canAssign
          dispatchingOrderIds={new Set()}
          cancelingOrderIds={overrides.cancelingOrderIds ?? new Set()}
          isAssigneesSaving={false}
          onDispatch={vi.fn().mockResolvedValue(undefined)}
          onAssigneesSave={vi.fn().mockResolvedValue(undefined)}
          onCancelQueue={overrides.onCancelQueue ?? vi.fn().mockResolvedValue(undefined)}
        />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("Start button on draft cards", () => {
  it("shows the busy state only on the task that dispatches", () => {
    renderBoard(new Set(["wo-1"]));

    const starting = screen.getByTestId("work-order-card-start-wo-1");
    expect(starting).toHaveTextContent("Starting...");
    expect(starting).toBeDisabled();

    const idle = screen.getByTestId("work-order-card-start-wo-2");
    expect(idle).toHaveTextContent("Start");
    expect(idle).toBeEnabled();
  });

  it("leaves every card ready when no dispatch runs", () => {
    renderBoard(new Set());

    for (const id of ["wo-1", "wo-2"]) {
      const start = screen.getByTestId(`work-order-card-start-${id}`);
      expect(start).toHaveTextContent("Start");
      expect(start).toBeEnabled();
    }
  });

  it("shows Queue when the next start would wait for a free slot", () => {
    renderBoard(new Set(), true);

    const queue = screen.getByTestId("work-order-card-start-wo-1");
    expect(queue).toHaveTextContent("Queue");
    expect(queue).toBeEnabled();
  });

  it("shows the busy Queue state only on the task that dispatches", () => {
    renderBoard(new Set(["wo-1"]), true);

    const queuing = screen.getByTestId("work-order-card-start-wo-1");
    expect(queuing).toHaveTextContent("Queuing...");
    expect(queuing).toBeDisabled();

    const idle = screen.getByTestId("work-order-card-start-wo-2");
    expect(idle).toHaveTextContent("Queue");
    expect(idle).toBeEnabled();
  });
});

describe("Queued first-step cards", () => {
  it("shows the queue place instead of Start", () => {
    renderQueuedBoard();

    expect(screen.queryByTestId("work-order-card-start-wo-queued")).not.toBeInTheDocument();
    expect(screen.getByTestId("work-order-card-queued-wo-queued")).toHaveTextContent("Queued #1");
    expect(screen.getByTestId("work-order-card-cancel-queue-wo-queued")).toHaveTextContent("Cancel");
    expect(screen.getByTestId("work-order-card-wo-queued").querySelector("[data-status-mark]")).toHaveAttribute(
      "data-status-mark",
      "queued",
    );
  });

  it("calls onCancelQueue from Cancel", async () => {
    const onCancelQueue = vi.fn().mockResolvedValue(undefined);
    renderQueuedBoard({ onCancelQueue });

    await userEvent.click(screen.getByTestId("work-order-card-cancel-queue-wo-queued"));
    expect(onCancelQueue).toHaveBeenCalledWith("wo-queued");
  });

  it("shows the busy state on Cancel while the request is in flight", () => {
    renderQueuedBoard({ cancelingOrderIds: new Set(["wo-queued"]) });

    const cancel = screen.getByTestId("work-order-card-cancel-queue-wo-queued");
    expect(cancel).toHaveTextContent("Canceling...");
    expect(cancel).toBeDisabled();
  });
});
