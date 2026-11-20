import type { SsoConnection, SsoDomain, SsoPutConnectionRequest, SsoscimToken } from "@/api-client/types.gen";

export function testResultText(connection: SsoConnection | null | undefined) {
  const result = connection?.lastTest;
  if (!result) {
    return null;
  }
  const when = connection.lastTestedAt ? new Date(connection.lastTestedAt).toLocaleString() : "";
  if (result.ok) {
    const who = result.email ? ` as ${result.email}` : "";
    return when ? `Last test passed on ${when}${who}.` : `Last test passed${who}.`;
  }
  const error = result.error ?? "";
  return when ? `Last test failed on ${when}: ${error}` : `Last test failed: ${error}`;
}

export function ssoTestFailureMessage(reason: string | null) {
  switch (reason) {
    case "test_member":
    case "test_permission":
    case "test_account":
      return "Your account cannot change Single Sign-On in this organization.";
    case "provider":
      return "Add the issuer URL, client ID and client secret first.";
    case "begin":
    case "config":
    case "issuer":
      return "SuperPlane could not read the identity provider configuration. Check the issuer URL.";
    default:
      return "The sign-in test failed.";
  }
}

export function statusLabel(status: string | undefined) {
  if (status === "STATUS_ACTIVE") return "Active";
  if (status === "STATUS_DISABLED") return "Disabled";
  return "Draft";
}

export type SsoSettingsModel = {
  licensed: boolean;
  canUpdate?: boolean;
  busy?: boolean;
  secretEpoch?: number;
  connection?: SsoConnection | null;
  domains: SsoDomain[];
  tokens: SsoscimToken[];
  domainError?: { id: string; message: string } | null;
  revealedToken?: string | null;
  testFailure?: string | null;
};

export type SsoSettingsActions = {
  onAddDomain: (domain: string) => void;
  onVerifyDomain: (id: string) => void;
  onDeleteDomain: (id: string) => void;
  onSave: (input: SsoPutConnectionRequest) => void;
  onTest: () => void;
  onActivate: () => void;
  onDisable: () => void;
  onDelete: () => void;
  onCreateToken: (description: string) => void;
  onRevokeToken: (id: string) => void;
  onDismissToken: () => void;
};

export type ConfirmRequest = { title: string; body: string; actionLabel: string; run: () => void };

export type SsoDraft = {
  canChange: boolean;
  canUpdate: boolean;
  permitted: boolean;
  identityDirty: boolean;
  credentialsDirty: boolean;
  provisioningDirty: boolean;
  policyDirty: boolean;
  providerReady: boolean;
  sessionError: string | null;
  exists: boolean;
  tested: boolean;
  verified: boolean;
  domain: string;
  setDomain: (value: string) => void;
  displayName: string;
  setDisplayName: (value: string) => void;
  preset: string;
  setPreset: (value: string) => void;
  issuer: string;
  setIssuer: (value: string) => void;
  clientId: string;
  setClientId: (value: string) => void;
  clientSecret: string;
  setClientSecret: (value: string) => void;
  jitEnabled: boolean;
  setJitEnabled: (value: boolean) => void;
  defaultRole: string;
  setDefaultRole: (value: string) => void;
  enforcement: string;
  setEnforcement: (value: string) => void;
  ownerBypass: boolean;
  setOwnerBypass: (value: boolean) => void;
  sessionHours: string;
  setSessionHours: (value: string) => void;
  idpLogout: boolean;
  setIdpLogout: (value: boolean) => void;
  tokenDescription: string;
  setTokenDescription: (value: string) => void;
  ask: (title: string, body: string, actionLabel: string, run: () => void) => void;
  requestRequired: () => void;
  createConnection: () => void;
  saveProvider: () => void;
  saveProvisioning: () => void;
  savePolicy: () => void;
};

export const noopSsoActions: SsoSettingsActions = {
  onAddDomain: () => undefined,
  onVerifyDomain: () => undefined,
  onDeleteDomain: () => undefined,
  onSave: () => undefined,
  onTest: () => undefined,
  onActivate: () => undefined,
  onDisable: () => undefined,
  onDelete: () => undefined,
  onCreateToken: () => undefined,
  onRevokeToken: () => undefined,
  onDismissToken: () => undefined,
};
