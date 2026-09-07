import { useEffect, useRef } from "react";
import { recordLastVisitedLocation } from "@/lib/lastVisitedLocation";
import { saveLastLocation } from "./useLastLocation";

// Debounced so navigating quickly through a canvas (selecting nodes,
// switching tabs) does not fire a backend write per keystroke-equivalent
// route change. Local storage still updates on every change for an instant,
// same-browser fallback.
const SAVE_DEBOUNCE_MS = 1500;

/**
 * Records `path` as the account's "resume where you left off" screen for
 * the given organization: instantly to local storage, and (debounced) to
 * the backend so it follows the user across devices and survives closing
 * the browser. Mount this once, near the top of the organization-scoped
 * route tree, passing the current `location.pathname + location.search`.
 */
export function useRecordLastLocation(
  organizationRoute: string | null,
  accountId: string | null | undefined,
  path: string,
): void {
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    if (!organizationRoute || !accountId) {
      return;
    }

    recordLastVisitedLocation(accountId, organizationRoute, path);

    if (timerRef.current) {
      clearTimeout(timerRef.current);
    }
    timerRef.current = setTimeout(() => {
      void saveLastLocation(organizationRoute, path);
    }, SAVE_DEBOUNCE_MS);

    return () => {
      if (timerRef.current) {
        clearTimeout(timerRef.current);
      }
    };
  }, [organizationRoute, accountId, path]);
}
