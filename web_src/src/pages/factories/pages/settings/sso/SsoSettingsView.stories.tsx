import type { Meta, StoryObj } from "@storybook/react-vite";

import { RequiredDialog, TokenDialog } from "./SsoSettingsSections";
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
export const UnlicensedConnection: Story = {
  args: {
    model: {
      licensed: false,
      domains: [{ id: "domain", domain: "example.com", verifiedAt: "2026-11-14T00:00:00Z" }],
      tokens: [],
      connection: { id: "conn", status: "STATUS_ACTIVE", displayName: "Okta" },
    },
  },
};
export const FailedTest: Story = {
  args: {
    model: {
      licensed: true,
      domains: [],
      tokens: [],
      connection: {
        id: "conn",
        status: "STATUS_DRAFT",
        lastTestedAt: "2026-11-14T12:00:00Z",
        lastTest: { ok: false, error: "The issuer did not respond." },
      },
    },
  },
};
export const PendingDomain: Story = {
  args: {
    model: {
      licensed: true,
      domains: [
        {
          id: "domain",
          domain: "example.com",
          txtName: "_superplane-verification.example.com",
          txtValue: "superplane-domain-verification=token",
        },
      ],
      tokens: [],
      connection: { id: "conn", status: "STATUS_DRAFT" },
    },
  },
};
export const TokenShownOnce: StoryObj = {
  render: () => <TokenDialog secret="scim-token-secret" onDismiss={() => undefined} />,
};
export const RequiredConfirmation: StoryObj = {
  render: () => <RequiredDialog open onOpenChange={() => undefined} onConfirm={() => undefined} />,
};
