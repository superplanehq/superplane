/**
 * Apps API client.
 *
 * Implements typed fetch wrappers for the Apps gRPC-gateway REST endpoints.
 * This file lives in src/lib/ (not in src/api-client/) so it is committed
 * alongside the rest of the source code rather than being gitignored as a
 * generated artifact.
 */

// ─── Types ──────────────────────────────────────────────────────────────────

export interface AppsAppMetadata {
  id?: string;
  organizationId?: string;
  displayName?: string;
  slug?: string;
  description?: string;
  canvasId?: string;
  createdAt?: string;
  updatedAt?: string;
  createdById?: string;
}

export interface AppsAppSyncState {
  status?: string;
  error?: string;
  liveCommitSha?: string;
  editSessionBranch?: string;
  defaultBranch?: string;
  codeStorageRemoteUrl?: string;
  codeStorageRepoId?: string;
}

export interface AppsApp {
  metadata?: AppsAppMetadata;
  syncState?: AppsAppSyncState;
}

export interface AppsAppDoc {
  id?: string;
  appId?: string;
  path?: string;
  content?: string;
  sha?: string;
  updatedAt?: string;
}

export interface DashboardPanel {
  id?: string;
  type?: string;
  content?: Record<string, unknown>;
}

export interface DashboardLayoutItem {
  i?: string;
  x?: number;
  y?: number;
  w?: number;
  h?: number;
  minW?: number;
  minH?: number;
}

export interface AppsDashboard {
  canvasId?: string;
  panels?: DashboardPanel[];
  layout?: DashboardLayoutItem[];
  updatedAt?: string;
}

export interface AppsCanvas {
  metadata?: { id?: string; name?: string; [k: string]: unknown };
  spec?: Record<string, unknown>;
  status?: Record<string, unknown>;
}

// ─── API call helper ─────────────────────────────────────────────────────────

function getOrganizationIdFromUrl(): string | null {
  const pathSegments = window.location.pathname.split("/");
  if (pathSegments[1] && pathSegments[1] !== "auth" && pathSegments[1] !== "login") {
    return pathSegments[1];
  }
  return null;
}

async function apiFetch<T>(url: string, options: RequestInit & { organizationId?: string } = {}): Promise<T> {
  const { organizationId, ...fetchOptions } = options;
  const orgId = organizationId ?? getOrganizationIdFromUrl();

  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(fetchOptions.headers as Record<string, string> | undefined),
  };
  if (orgId) {
    headers["x-organization-id"] = orgId;
  }

  const response = await fetch(url, { ...fetchOptions, headers });
  if (!response.ok) {
    const body = await response.text().catch(() => "");
    throw new Error(body || `HTTP ${response.status}`);
  }
  return response.json() as Promise<T>;
}

// ─── App endpoints ───────────────────────────────────────────────────────────

export async function listApps(organizationId: string): Promise<{ apps: AppsApp[] }> {
  return apiFetch("/api/v1/apps", { organizationId });
}

export async function describeApp(appId: string, organizationId: string): Promise<{ app: AppsApp }> {
  return apiFetch(`/api/v1/apps/${appId}`, { organizationId });
}

export async function createApp(
  input: { displayName: string; appSlug: string; description?: string },
  organizationId: string,
): Promise<{ app: AppsApp }> {
  return apiFetch("/api/v1/apps", {
    method: "POST",
    body: JSON.stringify(input),
    organizationId,
  });
}

export async function deleteApp(appId: string, organizationId: string): Promise<void> {
  await apiFetch(`/api/v1/apps/${appId}`, { method: "DELETE", organizationId });
}

export async function syncApp(appId: string, organizationId: string): Promise<{ app: AppsApp }> {
  return apiFetch(`/api/v1/apps/${appId}/sync`, {
    method: "POST",
    body: JSON.stringify({}),
    organizationId,
  });
}

export async function getAppDashboard(appId: string, organizationId: string): Promise<{ dashboard: AppsDashboard }> {
  return apiFetch(`/api/v1/apps/${appId}/dashboard`, { organizationId });
}

export async function updateAppDashboard(
  appId: string,
  input: { panels: DashboardPanel[]; layout: DashboardLayoutItem[] },
  organizationId: string,
): Promise<{ dashboard: AppsDashboard }> {
  return apiFetch(`/api/v1/apps/${appId}/dashboard`, {
    method: "PUT",
    body: JSON.stringify(input),
    organizationId,
  });
}

export async function getAppCanvas(appId: string, organizationId: string): Promise<{ canvas: AppsCanvas }> {
  return apiFetch(`/api/v1/apps/${appId}/canvas`, { organizationId });
}

export async function listAppDocs(appId: string, organizationId: string): Promise<{ docs: AppsAppDoc[] }> {
  return apiFetch(`/api/v1/apps/${appId}/docs`, { organizationId });
}

export async function getAppDoc(appId: string, path: string, organizationId: string): Promise<{ doc: AppsAppDoc }> {
  return apiFetch(`/api/v1/apps/${appId}/docs/${encodeURIComponent(path)}`, { organizationId });
}

export async function updateAppDoc(
  appId: string,
  path: string,
  content: string,
  organizationId: string,
): Promise<{ doc: AppsAppDoc }> {
  return apiFetch(`/api/v1/apps/${appId}/docs/${encodeURIComponent(path)}`, {
    method: "PUT",
    body: JSON.stringify({ content }),
    organizationId,
  });
}
