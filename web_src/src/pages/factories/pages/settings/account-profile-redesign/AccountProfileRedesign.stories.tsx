import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoriesHarness } from "../../../__fixtures__/FactoriesHarness";
import { defaultFactoriesFixture, PRIMARY_FACTORY_KEY } from "../../../__fixtures__/factoryPageResponses";
/**
 * Factory Account settings. Mounted through the live factory settings
 * chrome and the live Profile, Security, and Notifications pages.
 */
const meta = {
  title: "Factories/Pages/Account Profile Redesign",
  parameters: { layout: "fullscreen" },
} satisfies Meta;

export default meta;

type Story = StoryObj;

const accountPath = (section: string) => `workspaces/${PRIMARY_FACTORY_KEY}/settings/account/${section}`;

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
