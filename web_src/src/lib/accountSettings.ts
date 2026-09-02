export async function readResponseError(response: Response, fallback: string): Promise<string> {
  const text = (await response.text()).trim();
  return text || fallback;
}

export async function updateAccountName(name: string): Promise<void> {
  const response = await fetch("/account", {
    method: "PATCH",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ name }),
  });
  if (!response.ok) {
    throw new Error(await readResponseError(response, "Failed to save profile."));
  }
}

export async function updateAccountEmail(email: string): Promise<void> {
  const response = await fetch("/account", {
    method: "PATCH",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email }),
  });
  if (!response.ok) {
    throw new Error(await readResponseError(response, "Failed to update email."));
  }
}

export type AccountEmailSource = "github" | "google" | "password";

export interface AccountEmailOption {
  email: string;
  sources: AccountEmailSource[];
}

const SOURCE_LABELS: Record<AccountEmailSource, string> = {
  github: "GitHub",
  google: "Google",
  password: "Password",
};

export function accountEmailSourceLabel(sources: AccountEmailSource[]): string {
  return sources.map((source) => SOURCE_LABELS[source]).join(" · ");
}

export function accountEmailOptions(input: {
  email: string;
  hasPassword?: boolean;
  providers?: Array<{ provider: string; email?: string }>;
}): AccountEmailOption[] {
  const byEmail = new Map<string, AccountEmailOption>();
  const add = (email: string, source: AccountEmailSource) => {
    const normalized = email.trim().toLowerCase();
    if (!normalized) {
      return;
    }
    const existing = byEmail.get(normalized);
    if (existing) {
      if (!existing.sources.includes(source)) {
        existing.sources.push(source);
      }
      return;
    }
    byEmail.set(normalized, { email: normalized, sources: [source] });
  };

  let hasPasswordSource = false;
  for (const provider of input.providers ?? []) {
    if (provider.provider === "github" || provider.provider === "google" || provider.provider === "password") {
      add(provider.email ?? "", provider.provider);
      if (provider.provider === "password") {
        hasPasswordSource = true;
      }
    }
  }
  if (input.hasPassword && !hasPasswordSource) {
    add(input.email, "password");
  }
  if (byEmail.size === 0) {
    add(input.email, "password");
  }
  return [...byEmail.values()];
}

export async function deleteAccount(email: string): Promise<void> {
  const response = await fetch("/account", {
    method: "DELETE",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email }),
  });
  if (!response.ok) {
    throw new Error(await readResponseError(response, "Failed to delete account."));
  }
}

export async function disconnectAccountProvider(provider: string): Promise<void> {
  const response = await fetch(`/account/providers/${provider}`, {
    method: "DELETE",
    credentials: "include",
  });
  if (!response.ok) {
    throw new Error(await readResponseError(response, "Failed to disconnect sign-in method."));
  }
}

export function ssoLinkHref(provider: "github" | "google", redirectPath: string): string {
  const redirect = encodeURIComponent(redirectPath);
  return `/auth/${provider}?intent=link&redirect=${redirect}`;
}
