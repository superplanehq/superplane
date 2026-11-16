import { useState, type Dispatch, type SetStateAction } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { RequiredDialog, TokenDialog } from "./SsoSettingsSections";
import { SsoSettingsView } from "./SsoSettingsView";
import { type SsoSettingsActions, type SsoSettingsModel } from "./ssoSettingsTypes";

const meta = {
  title: "Settings/Single Sign-On",
  component: SsoSettingsView,
} satisfies Meta<typeof SsoSettingsView>;

export default meta;
type Story = StoryObj<typeof meta>;

function storyActions(setModel: Dispatch<SetStateAction<SsoSettingsModel>>): SsoSettingsActions {
  return {
    onAddDomain: (domain) => {
      console.log("add domain", domain);
      setModel((current) => ({ ...current, domains: [...current.domains, { id: domain, domain }] }));
    },
    onVerifyDomain: (id) => {
      console.log("verify domain", id);
      setModel((current) => ({
        ...current,
        domains: current.domains.map((item) =>
          item.id === id ? { ...item, verifiedAt: "2026-11-14T00:00:00Z" } : item,
        ),
      }));
    },
    onDeleteDomain: (id) => {
      console.log("delete domain", id);
      setModel((current) => ({ ...current, domains: current.domains.filter((item) => item.id !== id) }));
    },
    onSave: (input) => {
      console.log("save", input);
      setModel((current) => ({
        ...current,
        notice: "Saved",
        connection: {
          ...current.connection,
          displayName: input.displayName,
          issuer: input.issuer,
          clientId: input.clientId,
        },
      }));
    },
    onTest: () => {
      console.log("test");
      setModel((current) => ({
        ...current,
        notice: "The sign-in test succeeded.",
        connection: {
          ...current.connection,
          lastTestedAt: "2026-11-14T12:00:00Z",
          lastTest: { ok: true, email: "ada@example.com" },
        },
      }));
    },
    onActivate: () => {
      console.log("activate");
      setModel((current) => ({ ...current, connection: { ...current.connection, status: "STATUS_ACTIVE" } }));
    },
    onDisable: () => {
      console.log("disable");
      setModel((current) => ({ ...current, connection: { ...current.connection, status: "STATUS_DISABLED" } }));
    },
    onDelete: () => {
      console.log("delete");
      setModel((current) => ({ ...current, connection: null, tokens: [] }));
    },
    onCreateToken: (description) => {
      console.log("create token", description);
      setModel((current) => ({
        ...current,
        revealedToken: "scim-token-secret",
        tokens: [...current.tokens, { id: description || "token", description }],
      }));
    },
    onRevokeToken: (id) => {
      console.log("revoke token", id);
      setModel((current) => ({ ...current, tokens: current.tokens.filter((item) => item.id !== id) }));
    },
    onDismissToken: () => {
      console.log("dismiss token");
      setModel((current) => ({ ...current, revealedToken: null }));
    },
  };
}

function SSOStory({ initial }: { initial: SsoSettingsModel }) {
  const [model, setModel] = useState(initial);
  return <SsoSettingsView model={model} actions={storyActions(setModel)} />;
}

export const Locked: Story = { render: () => <SSOStory initial={{ licensed: false, domains: [], tokens: [] }} /> };
export const Draft: Story = {
  render: () => (
    <SSOStory
      initial={{
        licensed: true,
        domains: [],
        tokens: [],
        connection: { id: "conn", status: "STATUS_DRAFT", displayName: "Okta" },
      }}
    />
  ),
};
export const Tested: Story = {
  render: () => (
    <SSOStory
      initial={{
        licensed: true,
        domains: [{ id: "domain", domain: "example.com", verifiedAt: "2026-11-14T00:00:00Z" }],
        tokens: [],
        connection: { id: "conn", status: "STATUS_DRAFT", lastTest: { ok: true, email: "ada@example.com" } },
      }}
    />
  ),
};
export const Active: Story = {
  render: () => (
    <SSOStory
      initial={{
        licensed: true,
        domains: [{ id: "domain", domain: "example.com", verifiedAt: "2026-11-14T00:00:00Z" }],
        tokens: [{ id: "token", description: "Okta" }],
        connection: {
          id: "conn",
          status: "STATUS_ACTIVE",
          clientSecretSet: true,
          scimBaseUrl: "https://app.example.com/scim/v2/conn",
        },
      }}
    />
  ),
};
export const ErrorState: Story = {
  render: () => <SSOStory initial={{ licensed: true, domains: [], tokens: [], error: "The sign-in test failed." }} />,
};
export const UnlicensedConnection: Story = {
  render: () => (
    <SSOStory
      initial={{
        licensed: false,
        domains: [{ id: "domain", domain: "example.com", verifiedAt: "2026-11-14T00:00:00Z" }],
        tokens: [],
        connection: { id: "conn", status: "STATUS_ACTIVE", displayName: "Okta" },
      }}
    />
  ),
};
export const FailedTest: Story = {
  render: () => (
    <SSOStory
      initial={{
        licensed: true,
        domains: [],
        tokens: [],
        connection: {
          id: "conn",
          status: "STATUS_DRAFT",
          lastTestedAt: "2026-11-14T12:00:00Z",
          lastTest: { ok: false, error: "The issuer did not respond." },
        },
      }}
    />
  ),
};
export const PendingDomain: Story = {
  render: () => (
    <SSOStory
      initial={{
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
      }}
    />
  ),
};
export const TokenShownOnce: StoryObj = {
  render: () => <TokenDialog secret="scim-token-secret" onDismiss={() => undefined} />,
};
export const RequiredConfirmation: StoryObj = {
  render: () => <RequiredDialog open onOpenChange={() => undefined} onConfirm={() => undefined} />,
};

export const Interactive: Story = {
  render: () => (
    <SSOStory
      initial={{
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
      }}
    />
  ),
};
