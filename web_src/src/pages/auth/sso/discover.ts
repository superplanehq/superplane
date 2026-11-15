export type SsoDiscovery = {
  ssoAvailable: boolean;
  startUrl?: string;
  required?: boolean;
};

export type SsoDiscoveryResult =
  | { status: "available"; startUrl: string; required: boolean }
  | { status: "unavailable" }
  | { status: "error" };

export type EmailFirstSsoDecision = { kind: "continue" } | { kind: "prompt"; startUrl: string; required: boolean };

export async function discoverSso(email: string): Promise<SsoDiscoveryResult> {
  try {
    const response = await fetch("/auth/sso/discover", {
      method: "POST",
      headers: { "Content-Type": "application/json", Accept: "application/json" },
      body: JSON.stringify({ email }),
    });
    if (!response.ok) {
      return { status: "error" };
    }
    const body = (await response.json()) as SsoDiscovery;
    if (body.ssoAvailable && body.startUrl) {
      return { status: "available", startUrl: body.startUrl, required: Boolean(body.required) };
    }
    return { status: "unavailable" };
  } catch {
    return { status: "error" };
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
  if (discovery.status !== "available") {
    return { kind: "continue" };
  }
  return { kind: "prompt", startUrl: discovery.startUrl, required: discovery.required };
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
