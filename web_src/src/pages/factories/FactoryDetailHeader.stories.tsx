import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "./__fixtures__/ComponentStoryShell";
import { FACTORIES_ORGANIZATION_ID, PRIMARY_FACTORY_ID, REFUND_FACTORY } from "./__fixtures__/factoryPageResponses";
import { FactoryDetailHeader } from "./FactoryDetailHeader";

/**
 * Factory detail header: factory name, description, actions menu (edit/delete),
 * and the "New Work Order" CTA (gated by permissions). Doesn't include filters —
 * those live in the work orders panel.
 */
const meta = {
  title: "Factories/FactoryDetailHeader",
  component: FactoryDetailHeader,
  parameters: { layout: "padded" },
  decorators: [
    (Story) => (
      <ComponentStoryShell className="min-h-[220px] max-w-5xl bg-gray-50 p-6 dark:bg-gray-950">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta<typeof FactoryDetailHeader>;

export default meta;

type Story = StoryObj<typeof meta>;

const createHref = `/${FACTORIES_ORGANIZATION_ID}/factories/${PRIMARY_FACTORY_ID}/orders/new`;

const baseArgs = {
  factory: REFUND_FACTORY,
  workOrdersCount: 3,
  canCreate: true,
  canUpdate: true,
  canDelete: true,
  permissionsLoading: false,
  createHref,
  isUpdating: false,
  isDeleting: false,
  onUpdate: async () => undefined,
  onDelete: async () => undefined,
};

/** Populated header with a CTA and manage actions. */
export const Default: Story = {
  args: baseArgs,
};

/** Read-only viewer — CTA and actions disabled with tooltips. */
export const WithoutManagePermission: Story = {
  name: "Without Manage Permission",
  args: {
    ...baseArgs,
    workOrdersCount: 0,
    canCreate: false,
    canUpdate: false,
    canDelete: false,
  },
};
