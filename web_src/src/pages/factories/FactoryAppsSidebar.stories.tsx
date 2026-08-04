import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "./__fixtures__/ComponentStoryShell";
import { FACTORIES_ORGANIZATION_ID, REFUND_FACTORY_APPS } from "./__fixtures__/factoryPageResponses";
import { FactoryAppsSidebar } from "./FactoryAppsSidebar";

/**
 * Right-hand sidebar showing the apps owned by a factory plus a "+ New" CTA.
 * Accepts arbitrary `children` (used in production to slot the Lines sidebar
 * underneath).
 */
const meta = {
  title: "Factories/FactoryAppsSidebar",
  component: FactoryAppsSidebar,
  parameters: { layout: "padded" },
  decorators: [
    (Story) => (
      <ComponentStoryShell className="min-h-[320px] w-[320px] bg-white p-6 dark:bg-gray-900">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta<typeof FactoryAppsSidebar>;

export default meta;

type Story = StoryObj<typeof meta>;

/** Populated: three apps, "+ New" available. */
export const Populated: Story = {
  args: {
    organizationId: FACTORIES_ORGANIZATION_ID,
    apps: REFUND_FACTORY_APPS,
    isLoading: false,
    canCreate: true,
    permissionsLoading: false,
    onCreateClick: () => console.log("create app"),
  },
};

/** Loading: skeleton copy while apps are fetched. */
export const Loading: Story = {
  args: {
    organizationId: FACTORIES_ORGANIZATION_ID,
    apps: [],
    isLoading: true,
    canCreate: true,
    permissionsLoading: false,
    onCreateClick: () => console.log("create app"),
  },
};

/** Empty: no apps yet; only the "+ New" affordance is shown. */
export const Empty: Story = {
  args: {
    organizationId: FACTORIES_ORGANIZATION_ID,
    apps: [],
    isLoading: false,
    canCreate: true,
    permissionsLoading: false,
    onCreateClick: () => console.log("create app"),
  },
};
