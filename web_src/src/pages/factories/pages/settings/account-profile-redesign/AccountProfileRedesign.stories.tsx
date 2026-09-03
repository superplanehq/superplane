import type { Meta, StoryObj } from "@storybook/react-vite";

import { eventTypesFromToggles, defaultNotificationTypeToggles } from "@/lib/notificationSettings";

import { FactoriesHarness } from "../../../__fixtures__/FactoriesHarness";
import { defaultFactoriesFixture, PRIMARY_FACTORY_KEY } from "../../../__fixtures__/factoryPageResponses";
import { FactorySettingsLayout } from "../FactorySettingsLayout";

/**
 * Account settings pages in the live factory settings chrome.
 * Profile is the shipped identity, security, and GitHub-for-Velocity page.
 */
const meta = {
  title: "Factories/Pages/Settings/Account",
  component: FactorySettingsLayout,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof FactorySettingsLayout>;

export default meta;

type Story = StoryObj<typeof meta>;

const accountPath = (section: string) => `workspaces/${PRIMARY_FACTORY_KEY}/settings/account/${section}`;

export const Profile: Story = {
  render: () => <FactoriesHarness pathSuffix={accountPath("profile")} factoriesFixture={defaultFactoriesFixture} />,
};

export const Notifications: Story = {
  render: () => (
    <FactoriesHarness pathSuffix={accountPath("notifications")} factoriesFixture={defaultFactoriesFixture} />
  ),
};

export const NotificationsFiltered: Story = {
  name: "Notifications (Filtered)",
  render: () => (
    <FactoriesHarness
      pathSuffix={accountPath("notifications")}
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
