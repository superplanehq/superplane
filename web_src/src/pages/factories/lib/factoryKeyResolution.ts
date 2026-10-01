import type { FactoriesFactory } from "@/api-client";
import { isValidWorkspaceKey } from "./workspaceKey";

export type FactoryResolutionStatus = "loading" | "found" | "not-found";

export interface FactoryResolution {
  status: FactoryResolutionStatus;
  factory: FactoriesFactory | null;
  /**
   * How the match was made. `"urlId"` is the stable workspace URL id.
   * `"key"` is the current workspace key while that slug is still current.
   * `"id"` is the legacy fallback for old links that still carry the raw
   * database id. Callers use this (via `factoryRouteNeedsCanonicalRedirect`)
   * to send the browser to the canonical `key-urlId` URL.
   */
  matchedBy: "urlId" | "key" | "id" | null;
}

const LOADING: FactoryResolution = { status: "loading", factory: null, matchedBy: null };
const NOT_FOUND: FactoryResolution = { status: "not-found", factory: null, matchedBy: null };

const WORKSPACE_ROUTE_SEGMENT = /^([a-zA-Z]{2,5})-([a-z0-9]{8})$/;

/**
 * Canonical workspace URL segment: lowercase key, hyphen, then the stable
 * url id. Example: `eng-k7m2xqab`. When `urlId` is missing, returns the
 * lowercase key so incomplete fixtures still build a path.
 */
export function workspaceRouteSegment(key: string | undefined | null, urlId?: string | null): string {
  if (!key) {
    return "";
  }
  const lower = key.toLowerCase();
  if (!urlId) {
    return lower;
  }
  return `${lower}-${urlId}`;
}

export function factoryRouteSegment(factory: { key?: string; urlId?: string } | null | undefined): string {
  return workspaceRouteSegment(factory?.key, factory?.urlId);
}

export function parseWorkspaceRouteSegment(segment: string): { prefix: string; urlId: string } | null {
  const match = WORKSPACE_ROUTE_SEGMENT.exec(segment);
  if (!match) {
    return null;
  }
  return { prefix: match[1], urlId: match[2] };
}

/**
 * Resolves the `:factoryKey` route segment to a `FactoriesFactory` using the
 * already-fetched workspace list (no extra network round trip). Matches the
 * stable url id first, then the current workspace key, then `id` so old
 * UUID-based links keep working.
 *
 * Returns `"loading"` (rather than `"not-found"`) whenever the list hasn't
 * resolved yet, so callers don't flash an error state on every deep link.
 */
export function resolveFactoryByKey(
  factories: FactoriesFactory[],
  routeKey: string | undefined,
  isLoading: boolean,
): FactoryResolution {
  if (!routeKey) {
    return isLoading ? LOADING : NOT_FOUND;
  }

  const parsed = parseWorkspaceRouteSegment(routeKey);
  if (parsed) {
    const byUrlId = factories.find((factory) => factory.urlId === parsed.urlId);
    if (byUrlId) {
      return { status: "found", factory: byUrlId, matchedBy: "urlId" };
    }
  }

  const candidateKey = routeKey.toUpperCase();
  if (isValidWorkspaceKey(candidateKey)) {
    const byKey = factories.find((factory) => Boolean(factory.key) && factory.key!.toUpperCase() === candidateKey);
    if (byKey) {
      return { status: "found", factory: byKey, matchedBy: "key" };
    }
  }

  const byId = factories.find((factory) => Boolean(factory.id) && factory.id === routeKey);
  if (byId) {
    return { status: "found", factory: byId, matchedBy: "id" };
  }

  return isLoading ? LOADING : NOT_FOUND;
}

/**
 * True when the route segment is not the canonical `key-urlId` form — a
 * bare current key, a stale key prefix, a legacy id, or mixed case. Callers
 * `<Navigate replace>` to the canonical URL instead of rendering the page.
 */
export function factoryRouteNeedsCanonicalRedirect(resolution: FactoryResolution, routeKey: string): boolean {
  if (resolution.status !== "found") {
    return false;
  }
  const canonical = factoryRouteSegment(resolution.factory);
  return Boolean(canonical) && canonical !== routeKey;
}

/**
 * Swaps a stale `:factoryKey` route segment (a legacy id, a bare key, or a
 * non-canonical spelling) for the canonical segment, leaving the rest of
 * the path untouched so deep links under the workspace keep working.
 */
export function replaceFactoryKeySegment(
  pathname: string,
  organizationId: string,
  routeKey: string,
  canonicalKey: string,
): string {
  const lowerCanonicalKey = canonicalKey.toLowerCase();
  const prefix = `/${organizationId}/workspaces/${routeKey}`;
  if (pathname !== prefix && !pathname.startsWith(`${prefix}/`)) {
    return `/${organizationId}/workspaces/${lowerCanonicalKey}`;
  }
  return `/${organizationId}/workspaces/${lowerCanonicalKey}${pathname.slice(prefix.length)}`;
}
