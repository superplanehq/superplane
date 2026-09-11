import type { Meta, StoryObj } from "@storybook/react-vite";
import { useMemo, useState } from "react";

import type { FactoriesFactory, FactoriesFactoryLine, FactoriesWorkOrder } from "@/api-client";
import { Button } from "@/components/ui/button";

import { ComponentStoryShell } from "../__fixtures__/ComponentStoryShell";
import { withFactoriesTheme } from "../__fixtures__/factoriesStoryTheme";
import { buildWorkOrderListEntry } from "../lib/workOrderListModel";
import { KanbanCardMotionItem } from "./KanbanCardMotionItem";
import { useKanbanDisplayedBoard, type KanbanCardPlacement } from "./kanbanCardMotion";
import { WorkOrderBoardLane, WorkOrderKanbanBoard, workOrderKanbanLaneScrollClassName } from "./WorkOrderBoardChrome";
import { WorkOrderCard } from "./WorkOrderCard";

const factory: FactoriesFactory = { id: "factory-1", name: "Refunds", key: "RF" };
const factoryLines: FactoriesFactoryLine[] = [{ id: "line-a", name: "hotfix" }];

type PlaygroundOrder = {
  id: string;
  column: "backlog" | "implement";
  title: string;
  number: string;
};

const INITIAL_ORDERS: PlaygroundOrder[] = [
  { id: "wo-1", column: "backlog", title: "Unclear scope of Viewer role", number: "41" },
  { id: "wo-2", column: "backlog", title: "Make factory work-order dispatch idempotent", number: "40" },
  { id: "wo-3", column: "implement", title: "PR Closure: Close source issue", number: "38" },
];

function toWorkOrder(order: PlaygroundOrder): FactoriesWorkOrder {
  return {
    id: order.id,
    number: order.number,
    title: order.title,
    state: order.column === "backlog" ? "STATE_DRAFT" : "STATE_OPEN",
    createdAt: "2026-08-30T10:00:00Z",
    updatedAt: "2026-09-01T10:00:00Z",
    lineDispatches: [],
    assignees: [{ id: "user-1", name: "Ada Lovelace" }],
  };
}

function BoardMotionPlayground() {
  const [orders, setOrders] = useState<PlaygroundOrder[]>(INITIAL_ORDERS);
  const [nextNumber, setNextNumber] = useState(42);
  const placements = useMemo<KanbanCardPlacement[]>(
    () => orders.map((order) => ({ id: order.id, column: order.column })),
    [orders],
  );
  const displayed = useKanbanDisplayedBoard(orders, placements);
  const backlog = displayed.filter((order) => order.column === "backlog");
  const implement = displayed.filter((order) => order.column === "implement");

  const cardContext = {
    organizationId: "org-1",
    factoryKey: "RF",
    factoryLines,
    canDispatch: false,
    canAssign: false,
    dispatchingOrderIds: new Set<string>(),
    isAssigneesSaving: false,
    onDispatch: async () => {},
    onAssigneesSave: async () => {},
  };

  return (
    <div className="flex h-[32rem] w-[48rem] flex-col gap-3">
      <div className="flex gap-2">
        <Button
          type="button"
          size="sm"
          data-testid="story-add-backlog-card"
          onClick={() => {
            const id = `wo-${nextNumber}`;
            setOrders((current) => [
              {
                id,
                column: "backlog",
                title: `New intake ${nextNumber}`,
                number: String(nextNumber),
              },
              ...current,
            ]);
            setNextNumber((value) => value + 1);
          }}
        >
          Add Backlog card
        </Button>
        <Button
          type="button"
          size="sm"
          variant="secondary"
          data-testid="story-move-to-implement"
          onClick={() => {
            const moving = backlog[0];
            if (!moving) {
              return;
            }
            setOrders((current) =>
              current.map((order) => (order.id === moving.id ? { ...order, column: "implement" } : order)),
            );
          }}
        >
          Move first Backlog card
        </Button>
      </div>
      <WorkOrderKanbanBoard testId="story-kanban-motion-board">
        <WorkOrderBoardLane title="Backlog" count={backlog.length} emptyDescription="No tasks in the backlog.">
          <ul className={workOrderKanbanLaneScrollClassName}>
            {backlog.map((order) => (
              <KanbanCardMotionItem key={order.id} id={order.id}>
                <WorkOrderCard
                  {...cardContext}
                  entry={buildWorkOrderListEntry(toWorkOrder(order), factory)}
                  onOpen={() => {}}
                />
              </KanbanCardMotionItem>
            ))}
          </ul>
        </WorkOrderBoardLane>
        <WorkOrderBoardLane title="Implement" count={implement.length} emptyDescription="Nothing here." tone="running">
          <ul className={workOrderKanbanLaneScrollClassName}>
            {implement.map((order) => (
              <KanbanCardMotionItem key={order.id} id={order.id}>
                <WorkOrderCard
                  {...cardContext}
                  entry={buildWorkOrderListEntry(toWorkOrder(order), factory)}
                  onOpen={() => {}}
                />
              </KanbanCardMotionItem>
            ))}
          </ul>
        </WorkOrderBoardLane>
      </WorkOrderKanbanBoard>
    </div>
  );
}

const meta = {
  title: "Factories/Components/KanbanCardMotion",
  component: BoardMotionPlayground,
  parameters: { layout: "fullscreen" },
  decorators: [
    withFactoriesTheme,
    (Story) => (
      <ComponentStoryShell className="bg-background p-6">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta<typeof BoardMotionPlayground>;

export default meta;

type Story = StoryObj<typeof meta>;

/**
 * Click Add Backlog card to ease a ticket in. Click Move first Backlog
 * card to fly it into Implement. Uses the same view-transition path as
 * the Lines board.
 */
export const Playground: Story = {};
