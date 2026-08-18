import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import { defaultFactoriesFixture, PRIMARY_FACTORY_KEY } from "../../__fixtures__/factoryPageResponses";
import { FactorySettingsLayout } from "./FactorySettingsLayout";

/**
 * Factory settings — dedicated layout with its own left nav and General page content.
 */
const meta = {
  title: "Factories/Pages/Settings",
  component: FactorySettingsLayout,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof FactorySettingsLayout>;

export default meta;

type Story = StoryObj<typeof meta>;

export const General: Story = {
  render: () => (
    <FactoriesHarness
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/general`}
      factoriesFixture={defaultFactoriesFixture}
    />
  ),
};

export const Profile: Story = {
  render: () => (
    <FactoriesHarness
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/profile`}
      factoriesFixture={defaultFactoriesFixture}
    />
  ),
};

export const Notifications: Story = {
  render: () => (
    <FactoriesHarness
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/notifications`}
      factoriesFixture={defaultFactoriesFixture}
    />
  ),
};

export const NotificationsEnabled: Story = {
  name: "Notifications (Enabled)",
  render: () => (
    <FactoriesHarness
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/notifications`}
      factoriesFixture={{
        ...defaultFactoriesFixture,
        notificationSettings: {
          enabled: true,
          workspaceScope: "WORKSPACE_SCOPE_SELECTED",
          factoryIds: [defaultFactoriesFixture.factories[0]?.id ?? ""],
          workOrderAssigned: true,
          workOrderCommentOwned: true,
          workOrderCommentCreated: false,
          workOrderStatusOwned: true,
          workOrderArtifactOwned: false,
          workOrderMentioned: true,
        },
      }}
    />
  ),
};

export const RepositoriesSoon: Story = {
  name: "Repositories (Soon)",
  render: () => (
    <FactoriesHarness
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/repositories`}
      factoriesFixture={defaultFactoriesFixture}
    />
  ),
};

export const MembersSoon: Story = {
  name: "Members (Soon)",
  render: () => (
    <FactoriesHarness
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/members`}
      factoriesFixture={defaultFactoriesFixture}
    />
  ),
};
