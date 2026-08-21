import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "../__fixtures__/ComponentStoryShell";
import { withFactoriesTheme } from "../__fixtures__/factoriesStoryTheme";
import { EMPTY_FACTORY, FACTORIES_ORGANIZATION_ID, REFUND_FACTORY } from "../__fixtures__/factoryPageResponses";
import { WorkspaceSwitcher } from "./WorkspaceSwitcher";

const meta = {
  title: "Factories/Layout/WorkspaceSwitcher",
  component: WorkspaceSwitcher,
  parameters: {
    layout: "padded",
    docs: {
      description: {
        component: "The workspace initials open the switcher. The gear opens settings.",
      },
    },
  },
  decorators: [
    (Story) => (
      <ComponentStoryShell className="min-h-40 w-14 border-r border-sidebar-border bg-sidebar">
        <Story />
      </ComponentStoryShell>
    ),
    withFactoriesTheme,
  ],
} satisfies Meta<typeof WorkspaceSwitcher>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    organizationId: FACTORIES_ORGANIZATION_ID,
    factory: REFUND_FACTORY,
    factories: [REFUND_FACTORY, EMPTY_FACTORY],
    canOpenSettings: true,
    canCreateFactory: true,
    permissionsLoading: false,
    onCreateFactory: () => console.log("create workspace"),
  },
};

export const ReadOnly: Story = {
  args: {
    ...Default.args!,
    canOpenSettings: false,
    canCreateFactory: false,
  },
};
