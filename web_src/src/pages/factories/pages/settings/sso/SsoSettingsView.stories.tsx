import { useState, type Dispatch, type SetStateAction } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { SsoSettingsView } from "./SsoSettingsView";
import { type SsoSettingsActions, type SsoSettingsModel } from "./ssoSettingsTypes";

const meta = {
  title: "Settings/Single Sign-On",
  component: SsoSettingsView,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof SsoSettingsView>;

export default meta;
type Story = StoryObj<typeof meta>;

const pendingDomain = {
  id: "domain",
  domain: "example.com",
  txtName: "_superplane-verification.example.com",
  txtValue: "superplane-domain-verification=token",
};

const verifiedDomain = { ...pendingDomain, verifiedAt: "2026-11-14T00:00:00Z" };

const urls = {
  redirectUri: "https://app.example.com/auth/sso/conn/callback",
  initiateLoginUri: "https://app.example.com/auth/sso/conn/start",
  postLogoutRedirectUri: "https://app.example.com/login",
  backChannelLogoutUri: "https://app.example.com/auth/sso/conn/backchannel-logout",
  scimBaseUrl: "https://app.example.com/scim/v2/conn",
};

function storyActions(setModel: Dispatch<SetStateAction<SsoSettingsModel>>): SsoSettingsActions {
  return {
    onAddDomain: (domain) =>
      setModel((current) => ({ ...current, domains: [...current.domains, { id: domain, domain }] })),
    onVerifyDomain: (id) =>
      setModel((current) => ({
        ...current,
        domainError: null,
        domains: current.domains.map((item) =>
          item.id === id ? { ...item, verifiedAt: "2026-11-14T00:00:00Z" } : item,
        ),
      })),
    onDeleteDomain: (id) =>
      setModel((current) => ({ ...current, domains: current.domains.filter((item) => item.id !== id) })),
    onSave: (input) =>
      setModel((current) => ({
        ...current,
        connection: {
          id: current.connection?.id ?? "conn",
          status: current.connection?.status ?? "STATUS_DRAFT",
          ...current.connection,
          ...urls,
          displayName: input.displayName,
          preset: input.preset,
          issuer: input.issuer,
          clientId: input.clientId,
          clientSecretSet: Boolean(input.clientSecret) || current.connection?.clientSecretSet,
          jitEnabled: input.jitEnabled,
          defaultRole: input.defaultRole,
          enforcement: input.enforcement,
          allowOwnerBypass: input.allowOwnerBypass,
          maxSessionAgeSeconds: input.maxSessionAgeSeconds,
          idpLogoutEnabled: input.idpLogoutEnabled,
        },
      })),
    onTest: () => undefined,
    onActivate: () =>
      setModel((current) => ({ ...current, connection: { ...current.connection, status: "STATUS_ACTIVE" } })),
    onDisable: () =>
      setModel((current) => ({ ...current, connection: { ...current.connection, status: "STATUS_DISABLED" } })),
    onDelete: () => setModel((current) => ({ ...current, connection: null, tokens: [] })),
    onCreateToken: (description) =>
      setModel((current) => ({
        ...current,
        revealedToken: "scim-token-secret",
        tokens: [...current.tokens, { id: description || "token", description, createdAt: "2026-11-14T00:00:00Z" }],
      })),
    onRevokeToken: (id) =>
      setModel((current) => ({ ...current, tokens: current.tokens.filter((item) => item.id !== id) })),
    onDismissToken: () => setModel((current) => ({ ...current, revealedToken: null })),
  };
}

function SSOStory({ initial }: { initial: SsoSettingsModel }) {
  const [model, setModel] = useState(initial);
  return <SsoSettingsView model={model} actions={storyActions(setModel)} />;
}

export const Locked: Story = {
  render: () => <SSOStory initial={{ licensed: false, domains: [], tokens: [] }} />,
};

export const NoConnection: Story = {
  render: () => <SSOStory initial={{ licensed: true, domains: [], tokens: [] }} />,
};

export const DraftPendingDomain: Story = {
  render: () => (
    <SSOStory
      initial={{
        licensed: true,
        domains: [pendingDomain],
        tokens: [],
        connection: {
          id: "conn",
          status: "STATUS_DRAFT",
          displayName: "Okta",
          preset: "PRESET_OKTA",
          ...urls,
        },
      }}
    />
  ),
};

export const DraftReadyToTest: Story = {
  render: () => (
    <SSOStory
      initial={{
        licensed: true,
        domains: [verifiedDomain],
        tokens: [],
        connection: {
          id: "conn",
          status: "STATUS_DRAFT",
          displayName: "Okta",
          preset: "PRESET_OKTA",
          issuer: "https://example.okta.com",
          clientId: "0oa123example",
          clientSecretSet: true,
          ...urls,
        },
      }}
    />
  ),
};

export const Active: Story = {
  render: () => (
    <SSOStory
      initial={{
        licensed: true,
        domains: [verifiedDomain],
        tokens: [{ id: "token", description: "Okta", createdAt: "2026-11-14T00:00:00Z" }],
        connection: {
          id: "conn",
          status: "STATUS_ACTIVE",
          displayName: "Okta",
          preset: "PRESET_OKTA",
          issuer: "https://example.okta.com",
          clientId: "0oa123example",
          clientSecretSet: true,
          lastTestedAt: "2026-11-14T12:00:00Z",
          lastTest: { ok: true, email: "ada@example.com" },
          ...urls,
        },
      }}
    />
  ),
};

export const DomainVerificationError: Story = {
  render: () => (
    <SSOStory
      initial={{
        licensed: true,
        domains: [pendingDomain],
        tokens: [],
        domainError: { id: "domain", message: "The TXT record was not found." },
        connection: {
          id: "conn",
          status: "STATUS_DRAFT",
          displayName: "Okta",
          preset: "PRESET_OKTA",
          ...urls,
        },
      }}
    />
  ),
};
