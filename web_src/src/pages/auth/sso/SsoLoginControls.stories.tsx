import type { Meta, StoryObj } from "@storybook/react-vite";

import { ContinueWithSso, SsoOrganizationPrompt } from "./SsoLoginControls";

const meta = {
  title: "Auth/SSO login",
  component: ContinueWithSso,
} satisfies Meta<typeof ContinueWithSso>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Button: Story = { args: { redirect: "", lastUsed: false } };
export const LastUsed: Story = {
  args: { redirect: "", lastUsed: true, onLastUsedHint: "You used SSO to log in last time" },
};
export const RequiredPrompt: StoryObj = {
  render: () => <SsoOrganizationPrompt required onContinue={() => undefined} onUseAnother={() => undefined} />,
};
export const OptionalPrompt: StoryObj = {
  render: () => <SsoOrganizationPrompt required={false} onContinue={() => undefined} onUseAnother={() => undefined} />,
};
export const DomainError: StoryObj = {
  render: () => <p role="alert">SSO is not set up for this email domain.</p>,
};
