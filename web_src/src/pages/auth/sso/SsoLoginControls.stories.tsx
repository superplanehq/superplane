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
export const EmailStep: StoryObj = {
  args: { redirect: "", lastUsed: false, startOpen: true },
};
export const DomainError: StoryObj = {
  args: {
    redirect: "",
    lastUsed: false,
    startOpen: true,
    initialError: "SSO is not set up for this email domain.",
  },
};
export const DiscoveryError: StoryObj = {
  args: {
    redirect: "",
    lastUsed: false,
    startOpen: true,
    initialError: "Single Sign-On could not be checked. Try again.",
  },
};
