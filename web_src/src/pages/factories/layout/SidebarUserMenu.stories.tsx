import type { Decorator, Meta, StoryObj } from "@storybook/react-vite";
import { useQueryClient } from "@tanstack/react-query";

import { accountOrganizationsQueryKey } from "@/hooks/useAccountOrganizations";
import { ComponentStoryShell } from "../__fixtures__/ComponentStoryShell";
import { withFactoriesTheme } from "../__fixtures__/factoriesStoryTheme";
import {
  FACTORIES_ORGANIZATION_ID,
  STORYBOOK_ME_USER_AVATAR_URL,
  STORYBOOK_ME_USER_NAME,
} from "../__fixtures__/factoryPageResponses";
import { SidebarUserMenu } from "./SidebarUserMenu";

const withAccountOrganizations: Decorator = (Story) => {
  const queryClient = useQueryClient();
  queryClient.setQueryData(accountOrganizationsQueryKey, [
    { id: FACTORIES_ORGANIZATION_ID, slug: "superplane", name: "SuperPlane" },
    { id: "org-storybook-acme", slug: "acme", name: "Acme" },
  ]);
  return <Story />;
};

const meta = {
  title: "Factories/Layout/SidebarUserMenu",
  component: SidebarUserMenu,
  parameters: { layout: "padded" },
  decorators: [
    (Story) => (
      <ComponentStoryShell className="flex min-h-[380px] w-14 flex-col justify-end border-r border-sidebar-border bg-sidebar">
        <Story />
      </ComponentStoryShell>
    ),
    withAccountOrganizations,
    withFactoriesTheme,
  ],
} satisfies Meta<typeof SidebarUserMenu>;

export default meta;

type Story = StoryObj<typeof meta>;

const defaultArgs = {
  organizationId: FACTORIES_ORGANIZATION_ID,
  factoryKey: "RF",
  userName: STORYBOOK_ME_USER_NAME,
  userAvatarUrl: STORYBOOK_ME_USER_AVATAR_URL,
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
