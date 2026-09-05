const STORAGE_PREFIX = "integration-setup-return";
const MAX_AGE_MS = 15 * 60 * 1000;

/** Sent with the GitHub callback so setup can return to onboarding in one hop. */
export const INTEGRATION_SETUP_RETURN_COOKIE = "sp_integration_setup_return";

/** Keeps the hosted GitHub account picker on Integrations instead of bouncing back. */
export const INTEGRATION_SETUP_STAY_PARAM = "setupStay";

/** GitHub returned setup_action=request. A GitHub admin must approve the install. */
export const GITHUB_SETUP_REQUEST_PARAM = "githubSetup";
export const GITHUB_SETUP_REQUEST_VALUE = "request";
/** GitHub organization the member asked an admin to approve. */
export const GITHUB_SETUP_ORG_PARAM = "githubOrg";

interface StoredReturn {
  path: string;
  createdAt: number;
}

// Keyed by organization only, not by integration id: the legacy GitHub connect
// creates a new integration during the round trip to the provider, so the id the
// caller knows before leaving does not match the id the provider redirects to.
function storageKey(organizationId: string): string {
  return `${STORAGE_PREFIX}:${organizationId}`;
}

function isSafePath(path: string, organizationId: string): boolean {
  const pathname = path.split("?")[0] ?? path;
  const isOrganizationPath = path.startsWith(`/${organizationId}/`);
  return (isOrganizationPath || pathname === "/onboarding") && !path.startsWith("//");
}

export function rememberIntegrationSetupReturn(organizationId: string, path: string | undefined): void {
  if (!organizationId || !path || !isSafePath(path, organizationId)) return;

  const value: StoredReturn = { path, createdAt: Date.now() };
  window.localStorage.setItem(storageKey(organizationId), JSON.stringify(value));
  writeSetupReturnCookie(path);
}

export function peekIntegrationSetupReturn(organizationId: string): string | null {
  if (!organizationId) return null;

  const key = storageKey(organizationId);
  const raw = window.localStorage.getItem(key);
  if (!raw) return null;

  try {
    const value = JSON.parse(raw) as Partial<StoredReturn>;
    if (
      typeof value.path !== "string" ||
      typeof value.createdAt !== "number" ||
      !isSafePath(value.path, organizationId) ||
      Date.now() - value.createdAt > MAX_AGE_MS
    ) {
      window.localStorage.removeItem(key);
      return null;
    }
    return value.path;
  } catch {
    window.localStorage.removeItem(key);
    return null;
  }
}

export function consumeIntegrationSetupReturn(organizationId: string): void {
  window.localStorage.removeItem(storageKey(organizationId));
  clearSetupReturnCookie();
}

function writeSetupReturnCookie(path: string): void {
  const maxAge = Math.floor(MAX_AGE_MS / 1000);
  document.cookie = `${INTEGRATION_SETUP_RETURN_COOKIE}=${encodeURIComponent(path)}; Path=/; Max-Age=${maxAge}; SameSite=Lax`;
}

function clearSetupReturnCookie(): void {
  document.cookie = `${INTEGRATION_SETUP_RETURN_COOKIE}=; Path=/; Max-Age=0; SameSite=Lax`;
}

function pathnameOf(path: string): string {
  return path.split("?")[0] ?? path;
}

/** Deletes the marker after the browser lands on the stored return page. */
export function consumeIntegrationSetupReturnIfArrived(organizationId: string, currentPathname: string): void {
  const stored = peekIntegrationSetupReturn(organizationId);
  if (!stored) return;
  if (pathnameOf(stored) !== pathnameOf(currentPathname)) return;
  consumeIntegrationSetupReturn(organizationId);
}

export function hasIntegrationSetupStay(search: string): boolean {
  const query = search.startsWith("?") ? search.slice(1) : search;
  return new URLSearchParams(query).get(INTEGRATION_SETUP_STAY_PARAM) === "1";
}

export function hasGitHubSetupRequest(search: string): boolean {
  const query = search.startsWith("?") ? search.slice(1) : search;
  return new URLSearchParams(query).get(GITHUB_SETUP_REQUEST_PARAM) === GITHUB_SETUP_REQUEST_VALUE;
}

export function githubSetupRequestedOrganization(search: string): string {
  const query = search.startsWith("?") ? search.slice(1) : search;
  return new URLSearchParams(query).get(GITHUB_SETUP_ORG_PARAM)?.trim() ?? "";
}

/** Copies githubSetup=request from the provider callback onto the stored return path. */
export function withGitHubSetupRequest(path: string, search: string): string {
  if (!hasGitHubSetupRequest(search)) return path;

  const [pathname, existing = ""] = path.split("?");
  const params = new URLSearchParams(existing);
  params.set(GITHUB_SETUP_REQUEST_PARAM, GITHUB_SETUP_REQUEST_VALUE);
  const organization = githubSetupRequestedOrganization(search);
  if (organization !== "") {
    params.set(GITHUB_SETUP_ORG_PARAM, organization);
  }
  return `${pathname}?${params.toString()}`;
}
