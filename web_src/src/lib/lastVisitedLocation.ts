// Instant, same-browser fallback for "resume where you left off". The
// backend (see `useLastLocation`) is the source of truth once it responds,
// so a user waiting on something like a pending approval still resumes
// correctly in a fresh browser or on another device. This local copy only
// covers the gap before that response arrives, and offline/unauthenticated
// first paint.
//
// Entries are keyed by `${accountId}:${organizationRoute}` because the same
// account can belong to several organizations, each with its own last
// screen. `organizationRoute` is whatever segment the app used in the URL
// (slug or UID), matching `organizationRouteId`.
import { isSafeRedirectPath } from "./safeRedirectPath";

export const LAST_VISITED_LOCATION_STORAGE_KEY = "superplane:last-visited-location";

type LastVisitedLocationByKey = Record<string, string>;

function storageKey(accountId: string, organizationRoute: string): string {
  return `${accountId}:${organizationRoute}`;
}

function readAllLastVisitedLocations(): LastVisitedLocationByKey {
  if (typeof window === "undefined") {
    return {};
  }

  try {
    const raw = window.localStorage.getItem(LAST_VISITED_LOCATION_STORAGE_KEY);
    if (!raw) {
      return {};
    }

    const parsed = JSON.parse(raw) as unknown;
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
      return {};
    }

    const result: LastVisitedLocationByKey = {};
    for (const [key, path] of Object.entries(parsed as Record<string, unknown>)) {
      if (typeof path === "string" && isSafeRedirectPath(path)) {
        result[key] = path;
      }
    }

    return result;
  } catch {
    return {};
  }
}

/** Returns the last path visited for this account within this organization, if any. */
export function readLastVisitedLocation(accountId: string, organizationRoute: string): string | null {
  if (!accountId || !organizationRoute) {
    return null;
  }

  return readAllLastVisitedLocations()[storageKey(accountId, organizationRoute)] ?? null;
}

/** Records the path the account last visited within an organization. */
export function recordLastVisitedLocation(accountId: string, organizationRoute: string, path: string): void {
  if (!accountId || !organizationRoute || !isSafeRedirectPath(path) || typeof window === "undefined") {
    return;
  }

  try {
    const all = readAllLastVisitedLocations();
    all[storageKey(accountId, organizationRoute)] = path;
    window.localStorage.setItem(LAST_VISITED_LOCATION_STORAGE_KEY, JSON.stringify(all));
  } catch {
    // Last-visited persistence is optional.
  }
}
