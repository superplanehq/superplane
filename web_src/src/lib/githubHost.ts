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
  const hostname = (host ?? "").trim() || GITHUB_DOTCOM_WEB_HOST;
  return `https://${hostname}`;
}

export function githubRepoWebUrl(ownerRepo: string, host?: string | null): string | undefined {
  const match = /^([a-zA-Z0-9_.-]+)\/([a-zA-Z0-9_.-]+)$/.exec(ownerRepo.trim());
  if (!match) {
    return undefined;
  }
  return `${githubWebOrigin(host)}/${match[1]}/${match[2]}`;
}

export function isGitHubWebHostname(hostname: string, host?: string | null): boolean {
  const value = hostname.trim().toLowerCase();
  if (!value) {
    return false;
  }
  if (value === GITHUB_DOTCOM_WEB_HOST || value.endsWith(GHE_COM_SUFFIX)) {
    return true;
  }
  const expected = (host ?? "").trim().toLowerCase();
  return Boolean(expected) && value === expected;
}
