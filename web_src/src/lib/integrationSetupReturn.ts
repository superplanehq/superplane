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
/** GitHub connection that received the installation request callback. */
export const GITHUB_SETUP_INTEGRATION_PARAM = "githubIntegrationId";

interface StoredReturn {
  path: string;
  createdAt: number;
  preferredIntegrationId?: string;
}

// Legacy GitHub setup uses organization-wide local storage because it can
// change integration ids during its provider round trip. Other provider flows
// use tab-scoped session storage so concurrent setup tabs cannot overwrite one
// another.
function storageKey(organizationId: string): string {
  return `${STORAGE_PREFIX}:${organizationId}`;
}

function isSafePath(path: string, organizationId: string): boolean {
  const pathname = path.split("?")[0] ?? path;
  const isOrganizationPath = path.startsWith(`/${organizationId}/`);
  return (isOrganizationPath || pathname === "/onboarding") && !path.startsWith("//");
}

export function rememberIntegrationSetupReturn(
  organizationId: string,
  path: string | undefined,
  preferredIntegrationId?: string,
): void {
  if (!organizationId || !path || !isSafePath(path, organizationId)) return;

  const value: StoredReturn = {
    path,
    createdAt: Date.now(),
    ...(preferredIntegrationId?.trim() ? { preferredIntegrationId: preferredIntegrationId.trim() } : {}),
  };
  const key = storageKey(organizationId);
  if (value.preferredIntegrationId) {
    window.sessionStorage.setItem(key, JSON.stringify(value));
    return;
  }

  window.sessionStorage.removeItem(key);
  window.localStorage.setItem(key, JSON.stringify(value));
  writeSetupReturnCookie(path);
}

function readStoredReturn(organizationId: string, storage: Storage): StoredReturn | null {
  const key = storageKey(organizationId);
  const raw = storage.getItem(key);
  if (!raw) return null;

  try {
    const value = JSON.parse(raw) as Partial<StoredReturn>;
    if (
      typeof value.path !== "string" ||
      typeof value.createdAt !== "number" ||
      !isSafePath(value.path, organizationId) ||
      Date.now() - value.createdAt > MAX_AGE_MS
    ) {
      storage.removeItem(key);
      return null;
    }
    const preferredIntegrationId =
      typeof value.preferredIntegrationId === "string" && value.preferredIntegrationId.trim()
        ? value.preferredIntegrationId.trim()
        : undefined;
    return {
      path: value.path,
      createdAt: value.createdAt,
      ...(preferredIntegrationId ? { preferredIntegrationId } : {}),
    };
  } catch {
    storage.removeItem(key);
    return null;
  }
}

function readIntegrationSetupReturn(organizationId: string): StoredReturn | null {
  if (!organizationId) return null;

  const key = storageKey(organizationId);
  if (window.sessionStorage.getItem(key)) {
    return readStoredReturn(organizationId, window.sessionStorage);
  }
  return readStoredReturn(organizationId, window.localStorage);
}

export function peekIntegrationSetupReturn(organizationId: string): string | null {
  return readIntegrationSetupReturn(organizationId)?.path ?? null;
}

export function peekIntegrationSetupReturnPreferredIntegration(organizationId: string): string | null {
  return readIntegrationSetupReturn(organizationId)?.preferredIntegrationId ?? null;
}

export function consumeIntegrationSetupReturn(organizationId: string): void {
  const key = storageKey(organizationId);
  if (window.sessionStorage.getItem(key)) {
    window.sessionStorage.removeItem(key);
    return;
  }

  window.localStorage.removeItem(key);
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

export function githubSetupRequestedIntegration(search: string): string {
  const query = search.startsWith("?") ? search.slice(1) : search;
  return new URLSearchParams(query).get(GITHUB_SETUP_INTEGRATION_PARAM)?.trim() ?? "";
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
  const integrationId = githubSetupRequestedIntegration(search);
  if (integrationId !== "") {
    params.set(GITHUB_SETUP_INTEGRATION_PARAM, integrationId);
  }
  return `${pathname}?${params.toString()}`;
}
