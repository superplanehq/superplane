import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "../__fixtures__/ComponentStoryShell";
import { FACTORIES_ORGANIZATION_ID } from "../__fixtures__/factoryPageResponses";
import { SidebarUserMenu } from "./SidebarUserMenu";

const meta = {
  title: "Factories/Layout/SidebarUserMenu",
  component: SidebarUserMenu,
  parameters: { layout: "padded" },
  decorators: [
    (Story) => (
      <ComponentStoryShell className="w-64 bg-white dark:bg-gray-900">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta<typeof SidebarUserMenu>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    organizationId: FACTORIES_ORGANIZATION_ID,
    userName: "Storybook User",
    userEmail: "storybook@superplane.dev",
    userAvatarUrl: null,
    organizationName: "SuperPlane",
  },
};
