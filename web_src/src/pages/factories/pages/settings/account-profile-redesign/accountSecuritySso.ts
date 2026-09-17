import type { AccountRedesignSsoAccount } from "./accountProfileRedesignMocks";

export const SSO_PROVIDERS = [
  { provider: "github" as const, label: "GitHub" },
  { provider: "google" as const, label: "Google" },
] satisfies ReadonlyArray<{ provider: AccountRedesignSsoAccount["provider"]; label: string }>;

export type SsoProviderItem = (typeof SSO_PROVIDERS)[number];

const GITHUB_PURPOSE =
  "Used to sign in and to credit pull requests. This identity can also sign in to another SuperPlane account.";

export function ssoProviderDescription(
  provider: AccountRedesignSsoAccount["provider"],
  identity: string | null,
): string {
  if (provider === "github") {
    return identity ? `Connected as ${identity}. ${GITHUB_PURPOSE}` : GITHUB_PURPOSE;
  }
  return identity ? `Connected as ${identity}` : "Not connected";
}
