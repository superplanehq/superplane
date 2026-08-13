import type { FactoriesFactory } from "@/api-client";
import { normalizeWorkspaceKey } from "./workspaceKey";

export type FactoryResolutionStatus = "loading" | "found" | "not-found";

export interface FactoryResolution {
  status: FactoryResolutionStatus;
  factory: FactoriesFactory | null;
  /**
   * How the match was made. `"key"` covers both an exact and a
   * case-insensitive match on `factory.key`; `"id"` is the legacy fallback
   * for old links that still carry the raw database id. Callers use this
   * (via `factoryRouteNeedsCanonicalRedirect`) to send the browser to the
   * canonical `key` URL instead of rendering under a stale/legacy segment.
   */
  matchedBy: "key" | "id" | null;
}

const LOADING: FactoryResolution = { status: "loading", factory: null, matchedBy: null };
const NOT_FOUND: FactoryResolution = { status: "not-found", factory: null, matchedBy: null };

/**
 * Resolves the `:factoryKey` route segment to a `FactoriesFactory` using the
 * already-fetched workspace list (no extra network round trip). Matches the
 * workspace `key` case-insensitively first, then falls back to `id` so old
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

  const normalizedRouteKey = normalizeWorkspaceKey(routeKey);
  const byKey = normalizedRouteKey
    ? factories.find((factory) => Boolean(factory.key) && normalizeWorkspaceKey(factory.key!) === normalizedRouteKey)
    : undefined;
  if (byKey) {
    return { status: "found", factory: byKey, matchedBy: "key" };
  }

  const byId = factories.find((factory) => Boolean(factory.id) && factory.id === routeKey);
  if (byId) {
    return { status: "found", factory: byId, matchedBy: "id" };
  }

  return isLoading ? LOADING : NOT_FOUND;
}

/**
 * True when the route segment does not already match the canonical
 * `factory.key` exactly — a legacy id, a lowercase/mixed-case key, or any
 * other non-canonical spelling. Callers `<Navigate replace>` to the
 * canonical URL in this case instead of rendering the page.
 */
export function factoryRouteNeedsCanonicalRedirect(resolution: FactoryResolution, routeKey: string): boolean {
  return resolution.status === "found" && Boolean(resolution.factory?.key) && resolution.factory!.key !== routeKey;
}

/**
 * Swaps a stale `:factoryKey` route segment (a legacy id or a non-canonical
 * spelling of the key) for the canonical key, leaving the rest of the path
 * untouched so deep links under the workspace (work orders, lines, apps...)
 * keep working after the redirect.
 */
export function replaceFactoryKeySegment(
  pathname: string,
  organizationId: string,
  routeKey: string,
  canonicalKey: string,
): string {
  const prefix = `/${organizationId}/workspaces/${routeKey}`;
  if (!pathname.startsWith(prefix)) {
    return `/${organizationId}/workspaces/${canonicalKey}`;
  }
  return `/${organizationId}/workspaces/${canonicalKey}${pathname.slice(prefix.length)}`;
}
