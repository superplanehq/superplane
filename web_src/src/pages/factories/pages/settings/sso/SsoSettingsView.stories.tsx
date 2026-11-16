import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { RequiredDialog, TokenDialog } from "./SsoSettingsSections";
import { SsoSettingsView } from "./SsoSettingsView";
import { noopSsoActions, type SsoSettingsModel } from "./ssoSettingsTypes";

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

export const Interactive: Story = {
  render: function InteractiveSSO() {
    const [model, setModel] = useState<SsoSettingsModel>({
      licensed: true,
      domains: [{ id: "domain", domain: "example.com", verifiedAt: "2026-11-14T00:00:00Z" }],
      tokens: [],
      connection: {
        id: "conn",
        status: "STATUS_DRAFT",
        displayName: "Okta",
        issuer: "https://example.okta.com",
        clientId: "client",
        lastTest: { ok: true, email: "ada@example.com" },
      },
    });
    return (
      <SsoSettingsView
        model={model}
        actions={{
          ...noopSsoActions,
          onSave: (input) => {
            console.log("save", input);
            setModel((current) => ({
              ...current,
              notice: "Saved",
              connection: { ...current.connection, displayName: input.displayName, issuer: input.issuer },
            }));
          },
          onActivate: () => {
            console.log("activate");
            setModel((current) => ({
              ...current,
              connection: { ...current.connection, status: "STATUS_ACTIVE" },
            }));
          },
          onAddDomain: (domain) => {
            console.log("add domain", domain);
            setModel((current) => ({
              ...current,
              domains: [...current.domains, { id: domain, domain }],
            }));
          },
          onCreateToken: (description) => {
            console.log("create token", description);
            setModel((current) => ({
              ...current,
              revealedToken: "scim-token-secret",
              tokens: [...current.tokens, { id: description || "token", description }],
            }));
          },
          onDismissToken: () => setModel((current) => ({ ...current, revealedToken: null })),
        }}
      />
    );
  },
};
