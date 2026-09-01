import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "../../../__fixtures__/ComponentStoryShell";
import { withFactoriesTheme } from "../../../__fixtures__/factoriesStoryTheme";
import { ACCOUNT_REDESIGN_SECURE_PROFILE } from "./accountProfileRedesignMocks";
import { AccountProfileRedesignPlayground } from "./AccountProfileRedesignPlayground";

/**
 * Account settings redesign. Storybook-only. The live
 * `/settings/account/general` page is unchanged.
 *
 * Profile is editable. Tokens, password, 2FA, and sessions live on Security.
 * Theme and timezone live on Preferences.
 */
const meta = {
  title: "Factories/Pages/Account Profile Redesign",
  parameters: { layout: "fullscreen" },
  decorators: [
    withFactoriesTheme,
    (Story) => (
      <ComponentStoryShell className="h-svh bg-background p-0">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta;

export default meta;

type Story = StoryObj;

export const Profile: Story = {
  render: () => <AccountProfileRedesignPlayground />,
};

export const Security: Story = {
  render: () => <AccountProfileRedesignPlayground initialPage="security" />,
};

export const SecurityReady: Story = {
  name: "Security — 2FA and tokens",
  render: () => (
    <AccountProfileRedesignPlayground initialPage="security" initialProfile={ACCOUNT_REDESIGN_SECURE_PROFILE} />
  ),
};

export const Preferences: Story = {
  render: () => <AccountProfileRedesignPlayground initialPage="preferences" />,
};
