import type { Meta, StoryObj } from "@storybook/react-vite";

import { SsoSettingsView } from "./SsoSettingsView";

const meta = {
  title: "Settings/Single Sign-On",
  component: SsoSettingsView,
} satisfies Meta<typeof SsoSettingsView>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Locked: Story = { args: { model: { licensed: false, domains: [], tokens: [] } } };
export const Draft: Story = {
  args: {
    model: {
      licensed: true,
      domains: [],
      tokens: [],
      connection: { id: "conn", status: "STATUS_DRAFT", displayName: "Okta" },
    },
  },
};
export const Tested: Story = {
  args: {
    model: {
      licensed: true,
      domains: [{ id: "domain", domain: "example.com", verifiedAt: "2026-11-14T00:00:00Z" }],
      tokens: [],
      connection: { id: "conn", status: "STATUS_DRAFT", lastTest: { ok: true, email: "ada@example.com" } },
    },
  },
};
export const Active: Story = {
  args: {
    model: {
      licensed: true,
      domains: [{ id: "domain", domain: "example.com", verifiedAt: "2026-11-14T00:00:00Z" }],
      tokens: [{ id: "token", description: "Okta" }],
      connection: {
        id: "conn",
        status: "STATUS_ACTIVE",
        clientSecretSet: true,
        scimBaseUrl: "https://app.example.com/scim/v2/conn",
      },
    },
  },
};
export const ErrorState: Story = {
  args: { model: { licensed: true, domains: [], tokens: [], error: "The sign-in test failed." } },
};
