import { useEffect, useRef, useState } from "react";
import type { UseQueryResult } from "@tanstack/react-query";
import { useUpdateCanvasPreference, useCanvasPreference } from "@/hooks/useCanvasData";
import type { CanvasConsoleData } from "@/hooks/useCanvasData";

export type AppTabId = "canvas" | "console" | "memory" | "files";

const APP_TAB_VALUES: readonly AppTabId[] = ["canvas", "console", "memory", "files"] as const;

// Total write attempts allowed per tab value (initial write + retries) before
// giving up until the user switches tabs again.
const MAX_TAB_WRITE_ATTEMPTS = 3;

function isAppTabId(value: string | undefined | null): value is AppTabId {
  return typeof value === "string" && (APP_TAB_VALUES as readonly string[]).includes(value);
}

type UrlViewFlags = {
  isRunInspectionMode: boolean;
  isMemoryMode: boolean;
  isFilesMode: boolean;
  isConsoleMode: boolean;
};

/** Maps the URL view flags to the backend tab identifier. Returns null while run inspection is active. */
function urlViewFlagsToTab(flags: UrlViewFlags): AppTabId | null {
  if (flags.isRunInspectionMode) return null;
  if (flags.isConsoleMode) return "console";
  if (flags.isMemoryMode) return "memory";
  if (flags.isFilesMode) return "files";
  return "canvas";
}

/** An explicit `view` or `run` param means there is no default tab to resolve. */
function urlSelectsTabExplicitly(searchParams: URLSearchParams): boolean {
  return (searchParams.get("view") ?? "") !== "" || (searchParams.get("run") ?? "") !== "";
}

type ConsoleQueryLike = Pick<UseQueryResult<CanvasConsoleData | undefined>, "data" | "isSuccess" | "isError">;

type SetSearchParams = (
  next: URLSearchParams | ((prev: URLSearchParams) => URLSearchParams),
  options?: { replace?: boolean },
) => void;

type UseDefaultAppTabOptions = {
  organizationId: string | undefined;
  canvasId: string | undefined;
  urlViewFlags: UrlViewFlags;
  searchParams: URLSearchParams;
  setSearchParams: SetSearchParams;
  /** Console query result, used only when there is no stored preference yet to fall back to Console when panels exist. */
  consoleQuery: ConsoleQueryLike | undefined;
};

/**
 * Persists the current app tab to the backend and, on initial navigation to an
 * app without an explicit `view` or `run` query param, redirects to the user's
 * last visited tab, falling back to Console when the app has panels and Canvas
 * otherwise.
 */
export function useDefaultAppTab({
  organizationId,
  canvasId,
  urlViewFlags,
  searchParams,
  setSearchParams,
  consoleQuery,
}: UseDefaultAppTabOptions) {
  const preferenceQuery = useCanvasPreference(organizationId ?? "", canvasId ?? "");
  const updatePreferenceMutation = useUpdateCanvasPreference(organizationId ?? "");
  // `mutate` is referentially stable, unlike the mutation result object, so
  // depending on it keeps the record effect from re-running on every mutation
  // state change (which could turn a failed write into a retry loop).
  const recordTab = updatePreferenceMutation.mutate;

  const currentTab = urlViewFlagsToTab(urlViewFlags);
  // Tab most recently persisted (or adopted as baseline). Held in state
  // rather than a ref: a failed write clears it, and that alone must re-run
  // the record effect so the write is retried without waiting for the user
  // to switch tabs.
  const [recordedTab, setRecordedTab] = useState<AppTabId | null>(null);
  // Bounds retries of failed writes for a given tab value; without a cap,
  // clearing the guard on error would retry a permanently failing PUT forever.
  const writeAttemptsRef = useRef<{ tab: AppTabId | null; count: number }>({ tab: null, count: 0 });
  // Whether the default-tab redirect is settled for this app instance. Held
  // in state rather than a ref: resolution can settle without a URL change
  // (e.g. a console read that errors or reports no panels), and the record
  // effect below must re-run when that happens.
  const [redirectResolved, setRedirectResolved] = useState(() => urlSelectsTabExplicitly(searchParams));
  // Tab a just-scheduled redirect is navigating to. `setSearchParams` only
  // lands on the next render, so between scheduling and landing the URL still
  // reports the pre-redirect tab; the record effect must not persist it.
  const pendingRedirectTabRef = useRef<AppTabId | null>(null);
  // Whether the previous render was in run inspection (`?run=`). Closing a
  // run lands the user on Canvas without them picking a tab, so that landing
  // must not be persisted as a tab change.
  const inRunInspectionRef = useRef(false);
  // Tab the URL reported when this app instance started. If the user switches
  // tabs while the stored preference is still loading, the redirect must
  // yield to that explicit choice instead of forcing the stored tab later.
  const mountTabRef = useRef(currentTab);

  // The refs above hold state for a single app. React Router reuses the same
  // AppPage instance when navigating between apps (e.g. via the command
  // palette), so reset them whenever the canvas changes; otherwise the new
  // app would skip its default-tab redirect and could record the previous
  // app's tab against the wrong canvas. The render-phase setState is React's
  // sanctioned way to reset state when a prop changes.
  const refsOwnerCanvasIdRef = useRef(canvasId);
  if (refsOwnerCanvasIdRef.current !== canvasId) {
    refsOwnerCanvasIdRef.current = canvasId;
    pendingRedirectTabRef.current = null;
    inRunInspectionRef.current = false;
    mountTabRef.current = currentTab;
    writeAttemptsRef.current = { tab: null, count: 0 };
    setRecordedTab(null);
    setRedirectResolved(urlSelectsTabExplicitly(searchParams));
  }

  // Default-tab resolution: applied at most once per mount.
  useEffect(() => {
    if (redirectResolved) return;
    if (!organizationId || !canvasId) return;

    // The user already navigated to another tab while the stored preference
    // was loading; their explicit choice wins over the default-tab redirect.
    if (currentTab !== mountTabRef.current) {
      setRedirectResolved(true);
      return;
    }

    if (preferenceQuery.isPending) return;

    // A failed preference load leaves the stored tab unknown; redirecting
    // (including the Console fallback) could contradict it. Stay put.
    if (preferenceQuery.isError) {
      setRedirectResolved(true);
      return;
    }

    const storedTab = preferenceQuery.data?.lastVisitedTab;
    if (isAppTabId(storedTab)) {
      setRedirectResolved(true);
      // Already on the stored tab (e.g. a refresh landing on Canvas with a
      // stored "canvas" preference): rewriting the URL would only strip
      // unrelated params like `node`/`sidebar`/`file`, losing selection state.
      if (storedTab !== currentTab) {
        pendingRedirectTabRef.current = storedTab;
        applyTabToSearchParams(storedTab, setSearchParams);
      }
      return;
    }

    // No stored tab yet: fall back to Console if the app has panels.
    if (!consoleQuery) {
      setRedirectResolved(true);
      return;
    }
    // A console read that ended in error settles the fallback on the current
    // tab: waiting for a success that may never come would leave the
    // resolution pending forever, blocking tab recording below. Skipping the
    // Console fallback for this visit is the lesser cost.
    if (consoleQuery.isError) {
      setRedirectResolved(true);
      return;
    }
    // While the console read is still in flight, keep waiting; resolving now
    // would lock in Canvas even if the read later shows panels. An explicit
    // tab switch still resolves the redirect via the user-choice branch above.
    if (!consoleQuery.isSuccess) return;

    setRedirectResolved(true);
    const panels = consoleQuery.data?.panels ?? [];
    if (panels.length > 0) {
      pendingRedirectTabRef.current = "console";
      applyTabToSearchParams("console", setSearchParams);
    }
  }, [
    redirectResolved,
    organizationId,
    canvasId,
    currentTab,
    preferenceQuery.isPending,
    preferenceQuery.isError,
    preferenceQuery.data,
    consoleQuery,
    setSearchParams,
  ]);

  // Record tab changes to the backend once the initial redirect has settled.
  useEffect(() => {
    if (!organizationId || !canvasId) return;
    if (currentTab === null) {
      inRunInspectionRef.current = true;
      return;
    }
    if (!redirectResolved) return;

    // Recording requires knowing the stored tab: a preference query that is
    // still loading or failed reports no data, which is indistinguishable
    // from "no stored tab", and writing then could overwrite a real one.
    if (!preferenceQuery.isSuccess) return;

    // Closing run inspection lands on a tab the user did not actively pick;
    // adopt it as the recording baseline without overwriting the stored tab.
    if (inRunInspectionRef.current) {
      inRunInspectionRef.current = false;
      setRecordedTab(currentTab);
      return;
    }

    // A redirect was scheduled but the URL hasn't caught up yet — this effect
    // can run in the same commit that scheduled it, with `currentTab` still
    // reporting the pre-redirect tab. Recording now would overwrite the
    // stored tab with the tab we are navigating away from.
    if (pendingRedirectTabRef.current !== null) {
      if (currentTab !== pendingRedirectTabRef.current) return;
      pendingRedirectTabRef.current = null;
    }

    if (recordedTab === currentTab) return;

    // Avoid writing an identical value back to the server on first render.
    const storedTab = preferenceQuery.data?.lastVisitedTab;
    if (recordedTab === null && storedTab === currentTab) {
      setRecordedTab(currentTab);
      return;
    }

    // A fresh tab value gets a fresh attempt budget; retries of the same
    // failing value keep consuming it until the cap is hit.
    if (writeAttemptsRef.current.tab !== currentTab) {
      writeAttemptsRef.current = { tab: currentTab, count: 0 };
    }
    if (writeAttemptsRef.current.count >= MAX_TAB_WRITE_ATTEMPTS) return;
    writeAttemptsRef.current.count += 1;

    setRecordedTab(currentTab);
    recordTab(
      { canvasId, lastVisitedTab: currentTab },
      {
        onError: () => {
          // Treating a failed write as recorded would suppress every retry;
          // clearing the guard re-runs this effect (state change), which
          // retries the write until the attempt budget runs out.
          setRecordedTab((recorded) => (recorded === currentTab ? null : recorded));
        },
        onSuccess: () => {
          writeAttemptsRef.current = { tab: null, count: 0 };
        },
      },
    );
  }, [
    redirectResolved,
    recordedTab,
    canvasId,
    currentTab,
    organizationId,
    preferenceQuery.data,
    preferenceQuery.isSuccess,
    recordTab,
  ]);
}

function applyTabToSearchParams(tab: AppTabId, setSearchParams: SetSearchParams) {
  setSearchParams(
    (current) => {
      const next = new URLSearchParams(current);
      if (tab === "canvas") {
        next.delete("view");
      } else {
        next.set("view", tab);
      }
      next.delete("run");
      next.delete("sidebar");
      next.delete("node");
      next.delete("file");
      return next;
    },
    { replace: true },
  );
}
