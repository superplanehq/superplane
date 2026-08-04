import type { Meta, StoryObj } from "@storybook/react-vite";

import type { FactoriesWorkOrder } from "@/api-client";

import { ComponentStoryShell } from "./__fixtures__/ComponentStoryShell";
import {
  CLOSED_WORK_ORDER,
  FACTORIES_ORGANIZATION_ID,
  FAILED_WORK_ORDER,
  OPEN_WORK_ORDER,
  PRIMARY_FACTORY_ID,
  REFUND_FACTORY,
  REFUND_FACTORY_LINES,
  RUNNING_WORK_ORDER,
} from "./__fixtures__/factoryPageResponses";
import { WorkOrderDetailLoadedView } from "./WorkOrderDetailLoadedView";
import { getWorkOrderDetailDerived } from "./workOrderProgress";

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
  title: "Factories/WorkOrderDetailLoadedView",
  component: WorkOrderDetailLoadedView,
  parameters: { layout: "fullscreen" },
  decorators: [
    (Story) => (
      <ComponentStoryShell
        initialPath={`/${FACTORIES_ORGANIZATION_ID}/factories/${PRIMARY_FACTORY_ID}`}
        className="min-h-screen w-full bg-gray-50 dark:bg-gray-950"
      >
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta<typeof WorkOrderDetailLoadedView>;

export default meta;

type Story = StoryObj<typeof meta>;

const factoryHref = `/${FACTORIES_ORGANIZATION_ID}/factories/${PRIMARY_FACTORY_ID}`;

function buildLoadedViewArgs(order: FactoriesWorkOrder) {
  const derived = getWorkOrderDetailDerived(order);
  return {
    factory: REFUND_FACTORY,
    factoryHref,
    organizationId: FACTORIES_ORGANIZATION_ID,
    order,
    displayStatus: derived.displayStatus!,
    statusMeta: derived.statusMeta!,
    assigneeIds: derived.assigneeIds,
    assigneeNames: derived.assigneeNames,
    factoryLines: REFUND_FACTORY_LINES,
    isOpen: derived.isOpen,
    canDispatch: true,
    canClose: true,
    canAssign: true,
    permissionsLoading: false,
    isDispatching: false,
    isCompleting: false,
    isRejecting: false,
    isClosing: false,
    isAssigneesSaving: false,
    onDispatch: async (lineName: string) => {
      console.log("dispatch", lineName);
    },
    onClose: (result: "RESULT_COMPLETED" | "RESULT_REJECTED") => {
      console.log("close", result);
    },
    onAssigneesSave: async (assigneeIds: string[]) => {
      console.log("save assignees", assigneeIds);
    },
  };
}

/** Open — assignees + full action row (Dispatch / Complete / Reject). */
export const Open: Story = {
  args: buildLoadedViewArgs(OPEN_WORK_ORDER),
};

/** Running — action row still visible; badge shows the running spinner. */
export const Running: Story = {
  args: buildLoadedViewArgs(RUNNING_WORK_ORDER),
};

/** Failed — failed step surfaced in the timeline. */
export const Failed: Story = {
  args: buildLoadedViewArgs(FAILED_WORK_ORDER),
};

/** Closed — action row hidden, "Closed as completed" footer visible. */
export const Closed: Story = {
  args: buildLoadedViewArgs(CLOSED_WORK_ORDER),
};

/** Read-only viewer — permissions off; assignees edit button disabled. */
export const ReadOnly: Story = {
  name: "Read Only",
  args: {
    ...buildLoadedViewArgs(OPEN_WORK_ORDER),
    canDispatch: false,
    canClose: false,
    canAssign: false,
  },
};
