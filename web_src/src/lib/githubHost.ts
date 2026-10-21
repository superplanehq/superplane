/** Public github.com web hostname. Empty integration host means this value. */
export const GITHUB_DOTCOM_WEB_HOST = "github.com";

const GHE_COM_SUFFIX = ".ghe.com";

type HostedGitHubIntegration = {
  metadata?: { integrationName?: string | null };
  spec?: { configuration?: Record<string, unknown> | null };
  status?: { properties?: { name?: string | null; value?: string | null }[] | null };
};

function trimmedHost(value: unknown): string {
  return typeof value === "string" ? value.trim() : "";
}

function isDotComHost(host: string): boolean {
  return !host || host.toLowerCase() === GITHUB_DOTCOM_WEB_HOST;
}

/** Hostname stored on a GitHub integration. Empty means github.com. */
export function githubHostFromIntegration(integration: HostedGitHubIntegration | null | undefined): string {
  if (integration?.metadata?.integrationName !== "github") {
    return "";
  }
  const fromProperty = trimmedHost(integration.status?.properties?.find((property) => property.name === "host")?.value);
  if (!isDotComHost(fromProperty)) {
    return fromProperty;
  }
  const fromConfig = trimmedHost(integration.spec?.configuration?.host);
  if (!isDotComHost(fromConfig)) {
    return fromConfig;
  }
  return "";
}

export function isGitHubEnterpriseServerHost(host: string | null | undefined): boolean {
  const hostname = (host ?? "").trim().toLowerCase();
  return Boolean(hostname) && hostname !== GITHUB_DOTCOM_WEB_HOST && !hostname.endsWith(GHE_COM_SUFFIX);
}

export function githubWebOrigin(host?: string | null): string {
  const hostname = githubWebHostname(host);
  return `https://${hostname}`;
}

/** REST API origin for this GitHub host. Empty host is api.github.com. */
export function githubApiOrigin(host?: string | null): string {
  const hostname = githubWebHostname(host);
  if (hostname === GITHUB_DOTCOM_WEB_HOST) {
    return "https://api.github.com";
  }
  if (hostname.endsWith(GHE_COM_SUFFIX)) {
    return `https://api.${hostname}`;
  }
  return `https://${hostname}/api/v3`;
}

export function githubProfileUrl(login: string, host?: string | null): string {
  return `${githubWebOrigin(host)}/${login.trim()}`;
}

export function githubRepoWebUrl(ownerRepo: string, host?: string | null): string | undefined {
  const match = /^([a-zA-Z0-9_.-]+)\/([a-zA-Z0-9_.-]+)$/.exec(ownerRepo.trim());
  if (!match) {
    return undefined;
  }
  return `${githubWebOrigin(host)}/${match[1]}/${match[2]}`;
}

/** Stored integration host for a GitHub web URL. Empty means github.com. */
export function githubWebHostFromUrl(url: string | undefined): string {
  const hostname = hostnameFromUrl(url);
  if (!hostname || !isGitHubWebHostname(hostname)) {
    return "";
  }
  const web = githubWebHostname(hostname);
  return web === GITHUB_DOTCOM_WEB_HOST ? "" : web;
}

/** Tenant host from a hostname, URL, or GitHub repository/user object. Empty means github.com. */
export function githubHostFromHint(value: unknown): string {
  if (typeof value === "string") {
    return hostFromStringHint(value);
  }
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return "";
  }
  const record = value as Record<string, unknown>;
  for (const key of ["html_url", "url", "clone_url"] as const) {
    const host = typeof record[key] === "string" ? hostFromStringHint(record[key]) : "";
    if (host) {
      return host;
    }
  }
  return "";
}

function hostFromStringHint(raw: string): string {
  const trimmed = raw.trim();
  if (!trimmed) {
    return "";
  }
  const hostname = trimmed.includes("://") ? hostnameFromUrl(trimmed) : trimmed.toLowerCase();
  if (!isGitHubWebHostname(hostname) && !isKnownGitHubHostname(hostname)) {
    return "";
  }
  const web = githubWebHostname(hostname);
  return web === GITHUB_DOTCOM_WEB_HOST ? "" : web;
}

export function isGitHubWebHostname(hostname: string, host?: string | null): boolean {
  const value = hostname.trim().toLowerCase();
  if (!value) {
    return false;
  }
  if (value === GITHUB_DOTCOM_WEB_HOST || value === `www.${GITHUB_DOTCOM_WEB_HOST}`) {
    return true;
  }
  if (value.endsWith(GHE_COM_SUFFIX) && value !== "ghe.com" && !value.startsWith("api.")) {
    return true;
  }
  const expected = (host ?? "").trim().toLowerCase();
  return Boolean(expected) && value === expected;
}

/** Browser or API hostname on github.com or a GHE.com tenant. */
export function isKnownGitHubHostname(hostname: string): boolean {
  const value = hostname.trim().toLowerCase();
  if (!value) {
    return false;
  }
  if (value === GITHUB_DOTCOM_WEB_HOST || value === `www.${GITHUB_DOTCOM_WEB_HOST}` || value === "api.github.com") {
    return true;
  }
  return value.endsWith(GHE_COM_SUFFIX) && value !== "ghe.com";
}

function githubWebHostname(host?: string | null): string {
  const hostname = (host ?? "").trim().toLowerCase();
  if (!hostname || hostname === GITHUB_DOTCOM_WEB_HOST || hostname === `www.${GITHUB_DOTCOM_WEB_HOST}`) {
    return GITHUB_DOTCOM_WEB_HOST;
  }
  if (hostname.startsWith("api.") && hostname.endsWith(GHE_COM_SUFFIX)) {
    return hostname.slice("api.".length);
  }
  return hostname;
}

function hostnameFromUrl(url: string | undefined): string {
  const value = url?.trim();
  if (!value) {
    return "";
  }
  try {
    return new URL(value).hostname.trim().toLowerCase();
  } catch {
    return "";
  }
}
