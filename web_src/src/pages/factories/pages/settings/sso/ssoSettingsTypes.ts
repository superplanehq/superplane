import type { SsoConnection, SsoDomain, SsoPutConnectionRequest, SsoscimToken } from "@/api-client/types.gen";

export type SsoSettingsModel = {
  licensed: boolean;
  connection?: SsoConnection | null;
  domains: SsoDomain[];
  tokens: SsoscimToken[];
  error?: string | null;
  revealedToken?: string | null;
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

export type ConfirmRequest = { title: string; body: string; run: () => void };

export type SsoDraft = {
  canChange: boolean;
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
  ask: (title: string, body: string, run: () => void) => void;
  requestRequired: () => void;
  save: (nextEnforcement?: string) => void;
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
