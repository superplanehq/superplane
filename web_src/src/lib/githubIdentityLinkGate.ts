const GITHUB_CONNECT_RESUME_PARAM = "githubConnect";
const GITHUB_CONNECT_RESUME_VALUE = "resume";

type AuthConfigResponse = { providers?: string[] };

type AccountIdentityResponse = {
  providers?: Array<{ provider?: string }>;
  linked_accounts?: Array<{ provider?: string; username?: string }>;
};

/** True when the account proves a GitHub identity: a linked account or a GitHub sign-in. */
export function hasGitHubIdentity(account: AccountIdentityResponse | undefined): boolean {
  const linked = account?.linked_accounts?.some((item) => item.provider === "github" && Boolean(item.username));
  const signIn = account?.providers?.some((item) => item.provider === "github");
  return Boolean(linked || signIn);
}

export function githubIdentityLinkPath(returnTo: string): string {
  return githubConnectLinkPath(returnTo, false);
}

/**
 * Link flow that forces GitHub's account picker, so a member signed in to
 * more than one GitHub account can link a different one.
 */
export function githubSwitchAccountPath(returnTo: string): string {
  return githubConnectLinkPath(returnTo, true);
}

function githubConnectLinkPath(returnTo: string, selectAccount: boolean): string {
  const redirect = encodeURIComponent(withGitHubConnectResumeMarker(returnTo));
  const picker = selectAccount ? "&select_account=1" : "";
  return `/auth/github?intent=connect${picker}&redirect=${redirect}`;
}

/** Marks the return path so the page continues the connect after the link flow. */
export function withGitHubConnectResumeMarker(returnTo: string): string {
  const [pathname, search = ""] = returnTo.split("?");
  const params = new URLSearchParams(search);
  params.set(GITHUB_CONNECT_RESUME_PARAM, GITHUB_CONNECT_RESUME_VALUE);
  return `${pathname}?${params.toString()}`;
}

/** True when the page loads with the marker plus an auth result, i.e. the link flow just returned. */
export function isGitHubConnectResumeReturn(search: string): boolean {
  const params = new URLSearchParams(search);
  if (params.get(GITHUB_CONNECT_RESUME_PARAM) !== GITHUB_CONNECT_RESUME_VALUE) {
    return false;
  }
  return params.has("linked_account") || params.has("auth_error");
}

export function githubConnectResumeError(search: string): string {
  return new URLSearchParams(search).get("auth_error") ?? "";
}

export function stripGitHubConnectResumeParams(search: string): string {
  const params = new URLSearchParams(search);
  params.delete(GITHUB_CONNECT_RESUME_PARAM);
  params.delete("linked_account");
  params.delete("auth_error");
  params.delete("provider");
  return params.toString();
}

/**
 * Sends the member through the GitHub identity link flow before a hosted
 * connect, so installation discovery can prepopulate the account picker.
 * The gate never blocks a connect: with a known identity, with GitHub
 * sign-in not configured, or on any lookup error the connect proceeds
 * exactly as before.
 */
export async function redirectToGitHubIdentityLink(returnTo: string): Promise<boolean> {
  if (!(await needsGitHubIdentityLink())) {
    return false;
  }
  window.location.assign(githubIdentityLinkPath(returnTo));
  return true;
}

async function needsGitHubIdentityLink(): Promise<boolean> {
  try {
    const [config, account] = await Promise.all([
      fetchJson<AuthConfigResponse>("/auth/config"),
      fetchJson<AccountIdentityResponse>("/account"),
    ]);
    if (!config?.providers?.includes("github")) {
      return false;
    }
    return !hasGitHubIdentity(account);
  } catch {
    return false;
  }
}

async function fetchJson<T>(path: string): Promise<T | undefined> {
  const response = await fetch(path, { credentials: "include" });
  if (!response.ok) {
    throw new Error(`${path} responded with ${response.status}`);
  }
  return (await response.json()) as T;
}
