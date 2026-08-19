import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "./__fixtures__/ComponentStoryShell";
import {
  CLOSED_FAILED_WORK_ORDER,
  CLOSED_WORK_ORDER,
  DRAFT_WORK_ORDER,
  FACTORIES_ORGANIZATION_ID,
  FAILED_WORK_ORDER,
  OPEN_WORK_ORDER,
  PRIMARY_FACTORY_KEY,
  RUNNING_WORK_ORDER,
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
import { WorkOrderActivityTimeline } from "./WorkOrderActivityTimeline";
import { WorkOrderCommentComposer } from "./WorkOrderCommentComposer";

const composerFooter = (
  <WorkOrderCommentComposer
    organizationId={FACTORIES_ORGANIZATION_ID}
    canComment
    isSubmitting={false}
    onSubmit={async () => undefined}
  />
);

/**
 * Vertical timeline of the work order lifecycle: `created` marker, dispatch
 * batches per line, and (when closed) a "Closed as …" footer.
 */
const meta = {
  title: "Factories/Components/WorkOrderActivityTimeline",
  component: WorkOrderActivityTimeline,
  parameters: { layout: "padded" },
  args: {
    factoryKey: PRIMARY_FACTORY_KEY,
  },
  decorators: [
    (Story) => (
      <ComponentStoryShell className="min-h-[380px] max-w-2xl bg-white p-6 dark:bg-gray-900">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta<typeof WorkOrderActivityTimeline>;

export default meta;

type Story = StoryObj<typeof meta>;

/** Freshly opened order — only a `created` marker. */
export const JustCreated: Story = {
  args: {
    organizationId: FACTORIES_ORGANIZATION_ID,
    order: OPEN_WORK_ORDER,
    events: OPEN_WORK_ORDER_EVENTS,
  },
};

/** Running — dispatch batch with a step in progress. */
export const Running: Story = {
  args: {
    organizationId: FACTORIES_ORGANIZATION_ID,
    order: RUNNING_WORK_ORDER,
    events: RUNNING_WORK_ORDER_EVENTS,
  },
};

/** Failed — dispatch batch with a failed step surfaced. */
export const Failed: Story = {
  args: {
    organizationId: FACTORIES_ORGANIZATION_ID,
    order: FAILED_WORK_ORDER,
    events: FAILED_WORK_ORDER_EVENTS,
  },
};

/** Closed — full history + "Closed as completed" footer. */
export const Closed: Story = {
  args: {
    organizationId: FACTORIES_ORGANIZATION_ID,
    order: CLOSED_WORK_ORDER,
    events: CLOSED_WORK_ORDER_EVENTS,
  },
};

/** Draft — scoping notes + a status transition into `draft`. */
export const Draft: Story = {
  args: {
    organizationId: FACTORIES_ORGANIZATION_ID,
    order: DRAFT_WORK_ORDER,
    events: DRAFT_WORK_ORDER_EVENTS,
  },
};

/** Rich open order — comments (user + LLM) and both artifact kinds inline. */
export const WithCommentsAndArtifacts: Story = {
  name: "With Comments & Artifacts",
  args: {
    organizationId: FACTORIES_ORGANIZATION_ID,
    order: OPEN_WORK_ORDER,
    events: RICH_OPEN_WORK_ORDER_EVENTS,
  },
};

/** Closed as failed — markdown artifact + failed close footer. */
export const ClosedFailed: Story = {
  name: "Closed (failed)",
  args: {
    organizationId: FACTORIES_ORGANIZATION_ID,
    order: CLOSED_FAILED_WORK_ORDER,
    events: CLOSED_FAILED_WORK_ORDER_EVENTS,
  },
};

/** Older events are available but not loaded yet. */
export const WithLoadMore: Story = {
  args: {
    organizationId: FACTORIES_ORGANIZATION_ID,
    order: RUNNING_WORK_ORDER,
    events: RUNNING_WORK_ORDER_EVENTS,
    hasMoreEvents: true,
    onLoadMoreEvents: () => undefined,
  },
};

export const Loading: Story = {
  args: {
    organizationId: FACTORIES_ORGANIZATION_ID,
    order: OPEN_WORK_ORDER,
    isLoading: true,
    footer: composerFooter,
  },
};

export const ErrorState: Story = {
  args: {
    organizationId: FACTORIES_ORGANIZATION_ID,
    order: OPEN_WORK_ORDER,
    eventsError: new Error("Failed to load activity"),
    onRetryEvents: () => undefined,
    footer: composerFooter,
  },
};
