import { useEffect, useRef } from "react";
import { recordLastVisitedLocation } from "@/lib/lastVisitedLocation";
import { saveLastLocation } from "./useLastLocation";

// Debounced so navigating quickly through a canvas (selecting nodes,
// switching tabs) does not fire a backend write per keystroke-equivalent
// route change. Local storage still updates on every change for an instant,
// same-browser fallback.
const SAVE_DEBOUNCE_MS = 1500;

type LastLocationSnapshot = {
  organizationRoute: string | null;
  accountId: string | null | undefined;
  path: string;
};

function flushLastLocation(snapshot: LastLocationSnapshot): void {
  if (!snapshot.organizationRoute || !snapshot.accountId) {
    return;
  }
  void saveLastLocation(snapshot.organizationRoute, snapshot.path);
}

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
  const latestRef = useRef<LastLocationSnapshot>({ organizationRoute, accountId, path });
  latestRef.current = { organizationRoute, accountId, path };

  useEffect(() => {
    if (!organizationRoute || !accountId) {
      return;
    }

    recordLastVisitedLocation(accountId, organizationRoute, path);

    if (timerRef.current) {
      clearTimeout(timerRef.current);
    }
    timerRef.current = setTimeout(() => {
      timerRef.current = null;
      flushLastLocation({ organizationRoute, accountId, path });
    }, SAVE_DEBOUNCE_MS);

    return () => {
      if (timerRef.current) {
        clearTimeout(timerRef.current);
        timerRef.current = null;
      }
    };
  }, [organizationRoute, accountId, path]);

  useEffect(() => {
    const onLeave = () => {
      flushLastLocation(latestRef.current);
    };
    window.addEventListener("pagehide", onLeave);
    return () => {
      window.removeEventListener("pagehide", onLeave);
      flushLastLocation(latestRef.current);
    };
  }, []);
}
