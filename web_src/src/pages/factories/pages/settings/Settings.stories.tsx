import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import {
  defaultFactoriesFixture,
  EMPTY_FACTORY_KEY,
  PRIMARY_FACTORY_KEY,
} from "../../__fixtures__/factoryPageResponses";
import { eventTypesFromToggles, defaultNotificationTypeToggles } from "@/lib/notificationSettings";
import { FactorySettingsLayout } from "./FactorySettingsLayout";
import {
  CONNECTED_SETUP_INTEGRATIONS,
  SETUP_ANSWERS,
  factoriesFixtureWithSetupAnswers,
} from "../../__fixtures__/setupStoryFixtures";

/** Factory settings with Account, Workspace, and Organization navigation. */
const meta = {
  title: "Factories/Pages/Settings",
  component: FactorySettingsLayout,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof FactorySettingsLayout>;

export default meta;

type Story = StoryObj<typeof meta>;

export const WorkspaceGeneral: Story = {
  render: () => (
    <FactoriesHarness
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/workspace/general`}
      factoriesFixture={defaultFactoriesFixture}
    />
  ),
};

export const Automations: Story = {
  render: () => (
    <FactoriesHarness
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/workspace/automations`}
      factoriesFixture={defaultFactoriesFixture}
    />
  ),
};

/** Live Account General page. Redesign mockup: Factories/Pages/Account Profile Redesign. */
export const AccountGeneral: Story = {
  render: () => (
    <FactoriesHarness
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/account/general`}
      factoriesFixture={defaultFactoriesFixture}
    />
  ),
};

export const Notifications: Story = {
  render: () => (
    <FactoriesHarness
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/account/notifications`}
      factoriesFixture={defaultFactoriesFixture}
    />
  ),
};

export const NotificationsFiltered: Story = {
  name: "Notifications (Filtered)",
  render: () => (
    <FactoriesHarness
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/account/notifications`}
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

export const Repository: Story = {
  render: () => (
    <FactoriesHarness
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/workspace/repository`}
      factoriesFixture={factoriesFixtureWithSetupAnswers(SETUP_ANSWERS.agent)}
      orgIntegrations={CONNECTED_SETUP_INTEGRATIONS}
    />
  ),
};

export const Models: Story = {
  render: () => (
    <FactoriesHarness
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/workspace/models`}
      factoriesFixture={defaultFactoriesFixture}
    />
  ),
};

export const Spending: Story = {
  render: () => (
    <FactoriesHarness
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/workspace/spending`}
      factoriesFixture={defaultFactoriesFixture}
    />
  ),
};

export const SpendingEmpty: Story = {
  name: "Spending (empty)",
  render: () => (
    <FactoriesHarness
      pathSuffix={`workspaces/${EMPTY_FACTORY_KEY}/settings/workspace/spending`}
      factoriesFixture={defaultFactoriesFixture}
    />
  ),
};
