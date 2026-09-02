import type { Meta, StoryObj } from "@storybook/react-vite";

import { eventTypesFromToggles, defaultNotificationTypeToggles } from "@/lib/notificationSettings";

import { FactoriesHarness } from "../../../__fixtures__/FactoriesHarness";
import {
  FactoriesPaletteStoryShell,
  type FactoriesSettingsPalette,
} from "../../../__fixtures__/factoriesPaletteStoryShell";
import { defaultFactoriesFixture, PRIMARY_FACTORY_KEY } from "../../../__fixtures__/factoryPageResponses";
import { FactorySettingsLayout } from "../FactorySettingsLayout";

/**
 * Account settings pages in the live factory settings chrome.
 * Profile is the shipped identity, appearance, and GitHub-for-Velocity page.
 *
 * Dark palette stories compare two Mobbin-backed options:
 * - Cursor: warm canvas, warm field fills (shipped).
 * - GitHub: cool Primer canvas so field fills match the page hue.
 */
const meta = {
  title: "Factories/Pages/Settings/Account",
  component: FactorySettingsLayout,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof FactorySettingsLayout>;

export default meta;

type Story = StoryObj<typeof meta>;

const accountPath = (section: string) => `workspaces/${PRIMARY_FACTORY_KEY}/settings/account/${section}`;

const darkGlobals = { theme: "dark" as const, backgrounds: { value: "dark" } };

function accountPaletteStory(section: "profile" | "security", palette: FactoriesSettingsPalette): Story {
  return {
    globals: darkGlobals,
    render: () => (
      <FactoriesPaletteStoryShell palette={palette}>
        <FactoriesHarness pathSuffix={accountPath(section)} factoriesFixture={defaultFactoriesFixture} />
      </FactoriesPaletteStoryShell>
    ),
  };
}

export const Profile: Story = {
  render: () => <FactoriesHarness pathSuffix={accountPath("profile")} factoriesFixture={defaultFactoriesFixture} />,
};

export const Security: Story = {
  render: () => <FactoriesHarness pathSuffix={accountPath("security")} factoriesFixture={defaultFactoriesFixture} />,
};

export const SecurityReady: Story = {
  name: "Security — SSO and tokens",
  render: () => <FactoriesHarness pathSuffix={accountPath("security")} factoriesFixture={defaultFactoriesFixture} />,
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

export const ProfileDarkCursor: Story = {
  name: "Profile — Dark (Cursor)",
  ...accountPaletteStory("profile", "cursor"),
};

export const ProfileDarkGitHub: Story = {
  name: "Profile — Dark (GitHub)",
  ...accountPaletteStory("profile", "github"),
};

export const SecurityDarkCursor: Story = {
  name: "Security — Dark (Cursor)",
  ...accountPaletteStory("security", "cursor"),
};

export const SecurityDarkGitHub: Story = {
  name: "Security — Dark (GitHub)",
  ...accountPaletteStory("security", "github"),
};
