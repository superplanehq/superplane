import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";

import type { FactoriesWorkOrder, FactoriesWorkOrderArtifact, FactoriesWorkOrderEvent } from "@/api-client";

import { ComponentStoryShell } from "./__fixtures__/ComponentStoryShell";
import {
  CLOSED_FAILED_WORK_ORDER,
  CLOSED_WORK_ORDER,
  DRAFT_WORK_ORDER,
  FACTORIES_ORGANIZATION_ID,
  FAILED_WORK_ORDER,
  OPEN_WORK_ORDER,
  OPEN_WORK_ORDER_ARTIFACTS,
  PRIMARY_FACTORY_KEY,
  REFUND_FACTORY_LINES,
  RUNNING_WORK_ORDER,
  STORYBOOK_ME_USER_ID,
} from "./__fixtures__/factoryPageResponses";
import {
  CLOSED_FAILED_WORK_ORDER_EVENTS,
  CLOSED_WORK_ORDER_EVENTS,
  DRAFT_WORK_ORDER_EVENTS,
  FAILED_WORK_ORDER_EVENTS,
  OPEN_WORK_ORDER_EVENTS,
  RICH_OPEN_WORK_ORDER_EVENTS,
  RUNNING_WORK_ORDER_EVENTS,
} from "./__fixtures__/factoryPageEventFixtures";
import { WorkOrderDetailLoadedView } from "./WorkOrderDetailLoadedView";
import { WorkOrderReactionBar, type WorkOrderReactionGroup } from "./WorkOrderReactionBar";
import { getWorkOrderDetailDerived } from "./lib/workOrderProgress";

/**
 * Direct-props stories for `WorkOrderDetailLoadedView` — the pure composed
 * layout (header, description, activity timeline, assignees sidebar).
 *
 * These render the component itself with hand-controlled permissions and
 * loading flags so reviewers can flip through states without the page's
 * data-loading wrapper. Harnessed page variants live in
 * `WorkOrderDetailPage.stories.tsx`.
 */
const meta = {
  title: "Factories/Components/WorkOrderDetailLoadedView",
  component: WorkOrderDetailLoadedView,
  parameters: { layout: "fullscreen" },
  decorators: [
    (Story) => (
      <ComponentStoryShell
        initialPath={`/${FACTORIES_ORGANIZATION_ID}/workspaces/${PRIMARY_FACTORY_KEY}`}
        className="min-h-screen w-full bg-gray-50 dark:bg-gray-950"
      >
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta<typeof WorkOrderDetailLoadedView>;

export default meta;

type Story = StoryObj<typeof meta>;

interface BuildLoadedViewOverrides {
  events?: FactoriesWorkOrderEvent[];
  artifacts?: FactoriesWorkOrderArtifact[];
}

function buildLoadedViewArgs(order: FactoriesWorkOrder, overrides: BuildLoadedViewOverrides = {}) {
  const derived = getWorkOrderDetailDerived(order);
  return {
    organizationId: FACTORIES_ORGANIZATION_ID,
    factoryKey: PRIMARY_FACTORY_KEY,
    order,
    events: overrides.events ?? [],
    artifacts: overrides.artifacts ?? [],
    isArtifactsLoading: false,
    displayStatus: derived.displayStatus!,
    statusMeta: derived.statusMeta!,
    assigneeIds: derived.assigneeIds,
    assigneeNames: derived.assigneeNames,
    factoryLines: REFUND_FACTORY_LINES,
    isOpen: derived.isOpen,
    isDispatchable: derived.isDispatchable,
    isClosed: derived.isClosed,
    canDispatch: true,
    canClose: true,
    canAssign: true,
    canManage: true,
    permissionsLoading: false,
    isDispatching: false,
    isCompleting: false,
    isRejecting: false,
    isClosing: false,
    isAssigneesSaving: false,
    isUpdatingStatus: false,
    isAddingComment: false,
    onDispatch: async (input: { lineName: string }) => {
      console.log("dispatch", input);
    },
    onClose: (result: "RESULT_COMPLETED" | "RESULT_REJECTED" | "RESULT_FAILED") => {
      console.log("close", result);
    },
    onAssigneesSave: async (assigneeIds: string[]) => {
      console.log("save assignees", assigneeIds);
    },
    onStatusChange: async (state: string, result?: string) => {
      console.log("status change", state, result);
    },
    onAddComment: async (body: string) => {
      console.log("comment", body);
    },
  };
}

/** Open — assignees + full action row (Dispatch / Complete / Reject). */
export const Open: Story = {
  args: buildLoadedViewArgs(OPEN_WORK_ORDER, { events: OPEN_WORK_ORDER_EVENTS }),
};

/** Running — action row still visible; badge shows the running spinner. */
export const Running: Story = {
  args: buildLoadedViewArgs(RUNNING_WORK_ORDER, { events: RUNNING_WORK_ORDER_EVENTS }),
};

/** Failed — failed step surfaced in the timeline. */
export const Failed: Story = {
  args: buildLoadedViewArgs(FAILED_WORK_ORDER, { events: FAILED_WORK_ORDER_EVENTS }),
};

/** Closed — reopen actions available, "Closed as completed" footer visible. */
export const Closed: Story = {
  args: buildLoadedViewArgs(CLOSED_WORK_ORDER, { events: CLOSED_WORK_ORDER_EVENTS }),
};

/** Draft — Dispatch surfaces alongside the scoping notes. */
export const Draft: Story = {
  args: buildLoadedViewArgs(DRAFT_WORK_ORDER, { events: DRAFT_WORK_ORDER_EVENTS }),
};

/** Rich detail — inline comments, both artifact kinds, and the sidebar list. */
export const WithCommentsAndArtifacts: Story = {
  name: "With Comments & Artifacts",
  args: buildLoadedViewArgs(OPEN_WORK_ORDER, {
    events: RICH_OPEN_WORK_ORDER_EVENTS,
    artifacts: OPEN_WORK_ORDER_ARTIFACTS,
  }),
};

/** Closed as failed — failed badge, reopen actions, markdown artifact + failed close. */
export const ClosedFailed: Story = {
  name: "Closed (failed)",
  args: buildLoadedViewArgs(CLOSED_FAILED_WORK_ORDER, { events: CLOSED_FAILED_WORK_ORDER_EVENTS }),
};

/** Read-only viewer — permissions off; every action button is disabled. */
export const ReadOnly: Story = {
  name: "Read Only",
  args: {
    ...buildLoadedViewArgs(OPEN_WORK_ORDER, { events: OPEN_WORK_ORDER_EVENTS }),
    canDispatch: false,
    canClose: false,
    canAssign: false,
    canManage: false,
  },
};

// --- Reaction bar prototype (issue #6680) ---------------------------------
//
// `reactionBar` is a Storybook-only slot on `WorkOrderDetailLoadedView` — it
// stays `undefined` on the live page (see `WorkOrderDetailPage.tsx`), so
// nothing below touches production rendering.

const DEMO_REACTIONS: WorkOrderReactionGroup[] = [
  {
    emoji: "👍",
    users: [
      { id: STORYBOOK_ME_USER_ID, name: "Storybook User" },
      { id: "user-alex", name: "Alex Chen" },
    ],
  },
  { emoji: "👀", users: [{ id: "user-priya", name: "Priya Patel" }] },
];

/** Local, interactive stand-in for the reaction bar's future data source. */
function ReactionBarDemo({ canReact }: { canReact: boolean }) {
  const [reactions, setReactions] = useState<WorkOrderReactionGroup[]>(DEMO_REACTIONS);

  const onToggleReaction = (emoji: string) => {
    setReactions((current) => {
      const group = current.find((item) => item.emoji === emoji);
      const hasMine = group?.users.some((user) => user.id === STORYBOOK_ME_USER_ID) ?? false;

      if (hasMine) {
        return current
          .map((item) =>
            item.emoji === emoji
              ? { ...item, users: item.users.filter((user) => user.id !== STORYBOOK_ME_USER_ID) }
              : item,
          )
          .filter((item) => item.users.length > 0);
      }

      const me = { id: STORYBOOK_ME_USER_ID, name: "Storybook User" };
      return group
        ? current.map((item) => (item.emoji === emoji ? { ...item, users: [...item.users, me] } : item))
        : [...current, { emoji, users: [me] }];
    });
  };

  return (
    <WorkOrderReactionBar
      reactions={reactions}
      currentUserId={STORYBOOK_ME_USER_ID}
      canReact={canReact}
      onToggleReaction={onToggleReaction}
    />
  );
}

/** Open — reaction bar sits below the description, above Activity; try toggling a pill. */
export const WithReactionsOpen: Story = {
  name: "With Reactions (Open)",
  args: buildLoadedViewArgs(OPEN_WORK_ORDER, { events: OPEN_WORK_ORDER_EVENTS }),
  render: (args) => <WorkOrderDetailLoadedView {...args} reactionBar={<ReactionBarDemo canReact />} />,
};

/** Running — same reaction bar composed alongside the in-progress status badge. */
export const WithReactionsRunning: Story = {
  name: "With Reactions (Running)",
  args: buildLoadedViewArgs(RUNNING_WORK_ORDER, { events: RUNNING_WORK_ORDER_EVENTS }),
  render: (args) => <WorkOrderDetailLoadedView {...args} reactionBar={<ReactionBarDemo canReact />} />,
};

/** Closed — read-only order; the "+" trigger is disabled with a permission tooltip. */
export const WithReactionsClosed: Story = {
  name: "With Reactions (Closed)",
  args: buildLoadedViewArgs(CLOSED_WORK_ORDER, { events: CLOSED_WORK_ORDER_EVENTS }),
  render: (args) => <WorkOrderDetailLoadedView {...args} reactionBar={<ReactionBarDemo canReact={false} />} />,
};
