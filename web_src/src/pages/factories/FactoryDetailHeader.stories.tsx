import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "./__fixtures__/ComponentStoryShell";
import {
  FACTORIES_ORGANIZATION_ID,
  PRIMARY_FACTORY_ID,
  REFUND_FACTORY,
} from "./__fixtures__/factoryPageResponses";
import { FactoryDetailHeader } from "./FactoryDetailHeader";

/**
 * Factory detail header: factory name, description, and the "New Work Order"
 * CTA (gated by permissions). Doesn't include filters — those live in the
 * work orders panel.
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

/** Populated header with a CTA. */
export const Default: Story = {
  args: {
    factory: REFUND_FACTORY,
    workOrdersCount: 3,
    canCreate: true,
    permissionsLoading: false,
    createHref,
  },
};

/** Read-only viewer — CTA disabled with a tooltip. */
export const WithoutCreatePermission: Story = {
  name: "Without Create Permission",
  args: {
    factory: REFUND_FACTORY,
    workOrdersCount: 0,
    canCreate: false,
    permissionsLoading: false,
    createHref,
  },
};
