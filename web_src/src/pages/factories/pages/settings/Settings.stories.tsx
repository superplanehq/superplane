import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import {
  defaultFactoriesFixture,
  EMPTY_FACTORY_KEY,
  PRIMARY_FACTORY_KEY,
} from "../../__fixtures__/factoryPageResponses";
import { eventTypesFromToggles, defaultNotificationTypeToggles } from "@/lib/notificationSettings";
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

export const Automations: Story = {
  render: () => (
    <FactoriesHarness
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/automations`}
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

export const NotificationsFiltered: Story = {
  name: "Notifications (Filtered)",
  render: () => (
    <FactoriesHarness
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/notifications`}
      factoriesFixture={{
        ...defaultFactoriesFixture,
        notificationSettings: {
          workspaces: {
            scope: "WORKSPACE_SCOPE_FILTERED",
            filters: [
              {
                workspaceId: defaultFactoriesFixture.factories[0]?.id ?? "",
                eventTypes: eventTypesFromToggles({
                  ...defaultNotificationTypeToggles(),
                  TYPE_WORK_ORDER_COMMENT_CREATED: false,
                  TYPE_WORK_ORDER_ARTIFACT_OWNED: false,
                }),
              },
            ],
          },
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

export const Usage: Story = {
  render: () => (
    <FactoriesHarness
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/usage`}
      factoriesFixture={defaultFactoriesFixture}
    />
  ),
};

export const UsageEmpty: Story = {
  name: "Usage (empty)",
  render: () => (
    <FactoriesHarness
      pathSuffix={`workspaces/${EMPTY_FACTORY_KEY}/settings/usage`}
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
