import { useQuery } from "@tanstack/react-query";
import { meDescribeLastLocation, meSaveLastLocation } from "@/api-client";
import { isSafeRedirectPath, pathBelongsToOrganization } from "@/lib/safeRedirectPath";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

const lastLocationKey = (organizationRoute: string) => ["me", organizationRoute, "last-location"] as const;

/**
 * Fetches the calling user's server-saved "resume where you left off" path
 * for one organization, outside of React (e.g. right after login, before
 * the app has mounted). Returns null if there is none, the path is unsafe,
 * or the request fails.
 */
export async function fetchLastLocationPath(organizationRoute: string): Promise<string | null> {
  try {
    const response = await meDescribeLastLocation(withOrganizationHeader({ organizationId: organizationRoute }));
    const path = response.data?.lastLocation?.path;
    return path && isSafeRedirectPath(path) && pathBelongsToOrganization(path, organizationRoute) ? path : null;
  } catch {
    return null;
  }
}

/**
 * Fetches the calling user's server-saved "resume where you left off"
 * location for one organization. The backend is the source of truth once
 * this resolves; callers that need an instant, same-browser fallback should
 * also consult `readLastVisitedLocation` (see `lib/lastVisitedLocation.ts`).
 */
export function useLastLocation(organizationRoute: string | null) {
  return useQuery({
    queryKey: lastLocationKey(organizationRoute ?? ""),
    queryFn: () => fetchLastLocationPath(organizationRoute!),
    enabled: Boolean(organizationRoute),
    retry: false,
    staleTime: 0,
    gcTime: 0,
  });
}

/**
 * Saves the calling user's current location to the backend, best-effort.
 * Failures are swallowed: this is a convenience for resuming later, not a
 * user-facing action, and the local storage fallback still covers the
 * current browser if the request fails.
 */
export async function saveLastLocation(organizationRoute: string, path: string): Promise<void> {
  if (!organizationRoute || !isSafeRedirectPath(path) || !pathBelongsToOrganization(path, organizationRoute)) {
    return;
  }

  try {
    await meSaveLastLocation(
      withOrganizationHeader({
        organizationId: organizationRoute,
        body: { path },
      }),
    );
  } catch {
    // Best-effort: the local storage copy still lets this browser resume.
  }
}
