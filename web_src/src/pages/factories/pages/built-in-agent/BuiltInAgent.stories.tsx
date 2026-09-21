import type { FactoriesWorkOrder } from "@/api-client";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "../../__fixtures__/ComponentStoryShell";
import { factoryAgentChatMessages } from "../../__fixtures__/factoryAgentChatMessages";
import {
  CLOSED_WORK_ORDER,
  DRAFT_WORK_ORDER,
  INGEST_DRAFT_WORK_ORDER,
  OPEN_WORK_ORDER,
  OPEN_WORK_ORDER_SECONDARY,
  RUNNING_WORK_ORDER,
} from "../../__fixtures__/factoryPageWorkOrders";
import { withFactoriesTheme } from "../../__fixtures__/factoriesStoryTheme";
import {
  WORK_ORDER_BOARD_LANES,
  getWorkOrderDisplayStatus,
  type WorkOrderDisplayStatus,
} from "../../lib/workOrderProgress";
import { BuiltInAgentPlayground } from "./BuiltInAgentPlayground";
import {
  buildProposedPlan,
  type BuiltInAgentLane,
  type BuiltInAgentMessage,
  type BuiltInAgentTask,
} from "./builtInAgentMocks";
import type { BuiltInAgentPrototypeSeed } from "./useBuiltInAgentPrototype";

const LANE_BY_STATUS = new Map<WorkOrderDisplayStatus, BuiltInAgentLane>(
  WORK_ORDER_BOARD_LANES.flatMap((lane) => lane.statuses.map((status) => [status, lane.id] as const)),
);

function taskFromWorkOrder(order: FactoriesWorkOrder): BuiltInAgentTask {
  const displayStatus = getWorkOrderDisplayStatus(order);
  return {
    id: order.id ?? "",
    title: order.title?.trim() || "Untitled task",
    state: displayStatus,
    lane: LANE_BY_STATUS.get(displayStatus) ?? "backlog",
    owner: order.assignees?.[0]?.name ?? null,
    updatedAt: order.updatedAt ?? order.createdAt ?? new Date().toISOString(),
  };
}

const SEED_TASKS: BuiltInAgentTask[] = [
  DRAFT_WORK_ORDER,
  INGEST_DRAFT_WORK_ORDER,
  OPEN_WORK_ORDER,
  OPEN_WORK_ORDER_SECONDARY,
  RUNNING_WORK_ORDER,
  CLOSED_WORK_ORDER,
].map(taskFromWorkOrder);

const chatFixture = factoryAgentChatMessages();

const SEED_TRANSCRIPT: BuiltInAgentMessage[] = chatFixture.messages.map((message, index) => ({
  id: message.id,
  role: message.role as BuiltInAgentMessage["role"],
  content:
    index === 0
      ? "Create a task for the duplicate refund work."
      : "Created the task. Tell me if you want a plan for the board.",
  createdAt: message.createdAt,
}));

const withTasksSeed: BuiltInAgentPrototypeSeed = {
  tasks: SEED_TASKS,
  transcript: SEED_TRANSCRIPT,
};

/**
 * Storybook-only planning agent. The panel opens from the left rail and
 * pushes the board. Production navigation is unchanged.
 */
const meta = {
  title: "Factories/Prototypes/Built-in Agent",
  parameters: { layout: "fullscreen" },
  decorators: [
    withFactoriesTheme,
    (Story) => (
      <ComponentStoryShell className="min-h-svh bg-background p-0">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta;

export default meta;

type Story = StoryObj;

export const Closed: Story = {
  render: () => <BuiltInAgentPlayground seed={{ ...withTasksSeed, panelOpen: false }} />,
};

export const Empty: Story = {
  render: () => <BuiltInAgentPlayground seed={{ tasks: [], transcript: [] }} />,
};

export const WithTasks: Story = {
  name: "With tasks",
  render: () => <BuiltInAgentPlayground seed={withTasksSeed} />,
};

export const PlanProposed: Story = {
  name: "Plan proposed",
  render: () => <BuiltInAgentPlayground seed={{ ...withTasksSeed, pendingPlan: buildProposedPlan(SEED_TASKS) }} />,
};

export const DeleteConfirm: Story = {
  name: "Delete confirm",
  render: () => <BuiltInAgentPlayground seed={{ ...withTasksSeed, pendingDeleteId: SEED_TASKS[0]?.id ?? null }} />,
};

export const AgentError: Story = {
  name: "Agent error",
  render: () => <BuiltInAgentPlayground seed={{ ...withTasksSeed, hasError: true }} />,
};
