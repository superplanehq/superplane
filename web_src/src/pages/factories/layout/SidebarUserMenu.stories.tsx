import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "../__fixtures__/ComponentStoryShell";
import { withFactoriesTheme } from "../__fixtures__/factoriesStoryTheme";
import { FACTORIES_ORGANIZATION_ID } from "../__fixtures__/factoryPageResponses";
import { SidebarUserMenu } from "./SidebarUserMenu";

const meta = {
  title: "Factories/Layout/SidebarUserMenu",
  component: SidebarUserMenu,
  parameters: { layout: "padded" },
  decorators: [
    (Story) => (
      <ComponentStoryShell className="flex min-h-[380px] w-[240px] flex-col justify-end border-r border-sidebar-border bg-sidebar">
        <Story />
      </ComponentStoryShell>
    ),
    withFactoriesTheme,
  ],
} satisfies Meta<typeof SidebarUserMenu>;

export default meta;

type Story = StoryObj<typeof meta>;

const defaultArgs = {
  organizationId: FACTORIES_ORGANIZATION_ID,
  factoryKey: "RF",
  userName: "Storybook User",
  userEmail: "storybook@superplane.dev",
  userAvatarUrl: null,
  organizationName: "SuperPlane",
};

export const Default: Story = {
  args: defaultArgs,
};

export const Open: Story = {
  args: {
    ...defaultArgs,
    defaultOpen: true,
  },
};
