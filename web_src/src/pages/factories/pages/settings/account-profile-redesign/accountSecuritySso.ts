import type { AccountRedesignSsoAccount, AccountRedesignSsoProvider } from "./accountProfileRedesignMocks";

export const SSO_PROVIDERS = [
  { provider: "github" as const, label: "GitHub" },
  { provider: "google" as const, label: "Google" },
] satisfies ReadonlyArray<{ provider: AccountRedesignSsoAccount["provider"]; label: string }>;

export type SsoProviderItem = (typeof SSO_PROVIDERS)[number];

const GITHUB_PURPOSE = "Used to sign in and to credit pull requests.";

export function ssoProviderDescription(provider: AccountRedesignSsoProvider, identity: string | null): string {
  if (provider === "github") {
    return identity ? `Connected as ${identity}. ${GITHUB_PURPOSE}` : GITHUB_PURPOSE;
  }
  return identity ? `Connected as ${identity}` : "Not connected";
}

export function ssoDisconnectDescription(provider: AccountRedesignSsoProvider, label: string): string {
  if (provider === "github") {
    return "You cannot sign in with GitHub until you connect it again. Velocity also stops crediting your pull requests.";
  }
  return `You cannot sign in with ${label} until you connect it again.`;
}
