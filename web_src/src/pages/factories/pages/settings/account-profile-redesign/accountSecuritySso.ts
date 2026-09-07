import type { AccountRedesignSsoAccount } from "./accountProfileRedesignMocks";

export const SSO_PROVIDERS = [
  { provider: "github" as const, label: "GitHub" },
  { provider: "google" as const, label: "Google" },
] satisfies ReadonlyArray<{ provider: AccountRedesignSsoAccount["provider"]; label: string }>;

export type SsoProviderItem = (typeof SSO_PROVIDERS)[number];
