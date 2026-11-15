export type SsoDiscovery = {
  ssoAvailable: boolean;
  startUrl?: string;
  required?: boolean;
};

export type EmailFirstSsoDecision = { kind: "continue" } | { kind: "prompt"; startUrl: string; required: boolean };

export async function discoverSso(email: string): Promise<SsoDiscovery | null> {
  try {
    const response = await fetch("/auth/sso/discover", {
      method: "POST",
      headers: { "Content-Type": "application/json", Accept: "application/json" },
      body: JSON.stringify({ email }),
    });
    if (!response.ok) {
      return null;
    }
    return (await response.json()) as SsoDiscovery;
  } catch {
    return null;
  }
}

export async function resolveEmailFirstSso(input: {
  email: string;
  signup: boolean;
  ssoEnabled: boolean;
}): Promise<EmailFirstSsoDecision> {
  if (input.signup || !input.ssoEnabled) {
    return { kind: "continue" };
  }
  const discovery = await discoverSso(input.email);
  if (!discovery?.ssoAvailable || !discovery.startUrl) {
    return { kind: "continue" };
  }
  return { kind: "prompt", startUrl: discovery.startUrl, required: Boolean(discovery.required) };
}

export function ssoStartHref(startUrl: string, email: string, redirect: string): string {
  const url = new URL(startUrl, window.location.origin);
  if (email) {
    url.searchParams.set("login_hint", email);
  }
  if (redirect) {
    url.searchParams.set("redirect", redirect);
  }
  return `${url.pathname}${url.search}`;
}
