import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "../__fixtures__/ComponentStoryShell";
import { withFactoriesTheme } from "../__fixtures__/factoriesStoryTheme";
import {
  DEFAULT_WORK_ORDERS,
  FACTORIES_ORGANIZATION_ID,
  PRIMARY_FACTORY_KEY,
} from "../__fixtures__/factoryPageResponses";
import { FactoriesNav } from "./FactoriesNav";

const meta = {
  title: "Factories/Layout/FactoriesNav",
  component: FactoriesNav,
  parameters: { layout: "padded" },
  decorators: [
    (Story) => (
      <ComponentStoryShell
        initialPath={`/${FACTORIES_ORGANIZATION_ID}/workspaces/${PRIMARY_FACTORY_KEY}/overview`}
        className="min-h-[420px] w-[240px] border-r border-sidebar-border bg-sidebar"
      >
        <Story />
      </ComponentStoryShell>
    ),
    withFactoriesTheme,
  ],
} satisfies Meta<typeof FactoriesNav>;

export default meta;

type Story = StoryObj<typeof meta>;

export const WithRecents: Story = {
  args: {
    organizationId: FACTORIES_ORGANIZATION_ID,
    factoryKey: PRIMARY_FACTORY_KEY,
    recentWorkOrders: DEFAULT_WORK_ORDERS.slice(0, 5),
  },
};

export const NoRecents: Story = {
  args: {
    organizationId: FACTORIES_ORGANIZATION_ID,
    factoryKey: PRIMARY_FACTORY_KEY,
    recentWorkOrders: [],
  },
};
