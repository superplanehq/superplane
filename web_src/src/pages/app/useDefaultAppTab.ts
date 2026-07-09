import { useEffect, useRef, useState } from "react";
import type { Dispatch, SetStateAction } from "react";
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
  /**
   * Live console query result, used only when there is no stored preference
   * yet to fall back to Console when panels exist. This must read the live
   * console, not the active version's: the active version can be a draft
   * (e.g. from `?version=`) whose console is empty while the live app has
   * widgets, and first-visit defaulting is about what the app publishes.
   */
  liveConsoleQuery: ConsoleQueryLike | undefined;
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
  liveConsoleQuery,
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
  // A just-scheduled redirect: the tab it navigates to and the tab it
  // navigates away from. `setSearchParams` only lands on the next render, so
  // between scheduling and landing the URL still reports the pre-redirect
  // (`from`) tab; the record effect must not persist it. Tracking `from` lets
  // the record effect tell that window apart from the user switching to a
  // third tab before the redirect applies — that choice must be recorded.
  const pendingRedirectRef = useRef<{ from: AppTabId | null; to: AppTabId } | null>(null);
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
    pendingRedirectRef.current = null;
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

    const resolution = resolveDefaultTab({
      // The user already navigated to another tab while the stored preference
      // was loading; their explicit choice wins over the default-tab redirect.
      userSwitchedTabs: currentTab !== mountTabRef.current,
      preferenceIsPending: preferenceQuery.isPending,
      preferenceIsError: preferenceQuery.isError,
      storedTab: preferenceQuery.data?.lastVisitedTab,
      liveConsoleQuery,
    });
    if (!resolution.settled) return;

    setRedirectResolved(true);
    // No redirect when already on the target tab (e.g. a refresh landing on
    // Canvas with a stored "canvas" preference): rewriting the URL would only
    // strip unrelated params like `node`/`sidebar`/`file`, losing selection.
    if (resolution.redirectTo !== null && resolution.redirectTo !== currentTab) {
      const redirect = { from: currentTab, to: resolution.redirectTo };
      pendingRedirectRef.current = redirect;
      applyTabToSearchParams(
        resolution.redirectTo,
        setSearchParams,
        // The router can apply this update after the user already picked a
        // different tab; by then the record effect has cleared the pending
        // redirect, and rewriting the URL would replace the user's choice.
        () => pendingRedirectRef.current === redirect,
      );
    }
  }, [
    redirectResolved,
    organizationId,
    canvasId,
    currentTab,
    preferenceQuery.isPending,
    preferenceQuery.isError,
    preferenceQuery.data,
    liveConsoleQuery,
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

    const pendingRedirect = pendingRedirectRef.current;
    if (pendingRedirect !== null) {
      // The URL hasn't caught up with the scheduled redirect yet — this
      // effect can run in the same commit that scheduled it, with
      // `currentTab` still reporting the pre-redirect tab. Recording now
      // would overwrite the stored tab with the tab we are navigating away
      // from.
      if (currentTab === pendingRedirect.from) return;
      // Any other tab means the redirect is no longer pending: either it
      // landed on its target, or the user switched to a different tab first —
      // an explicit choice that must be recorded like any other tab change.
      pendingRedirectRef.current = null;
    }

    if (recordedTab === currentTab) return;

    // Avoid writing an identical value back to the server on first render.
    const storedTab = preferenceQuery.data?.lastVisitedTab;
    if (recordedTab === null && storedTab === currentTab) {
      setRecordedTab(currentTab);
      return;
    }

    persistTabWithRetryBudget({ canvasId, tab: currentTab, writeAttemptsRef, setRecordedTab, recordTab });
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

type DefaultTabResolution = { settled: false } | { settled: true; redirectTo: AppTabId | null };

type ResolveDefaultTabInput = {
  userSwitchedTabs: boolean;
  preferenceIsPending: boolean;
  preferenceIsError: boolean;
  storedTab: string | undefined;
  liveConsoleQuery: ConsoleQueryLike | undefined;
};

/**
 * Decides whether default-tab resolution can settle and, if so, which tab (if
 * any) to redirect to. Pure decision logic; the caller applies the redirect.
 */
function resolveDefaultTab({
  userSwitchedTabs,
  preferenceIsPending,
  preferenceIsError,
  storedTab,
  liveConsoleQuery,
}: ResolveDefaultTabInput): DefaultTabResolution {
  if (userSwitchedTabs) return { settled: true, redirectTo: null };

  if (preferenceIsPending) return { settled: false };

  // A failed preference load leaves the stored tab unknown; redirecting
  // (including the Console fallback) could contradict it. Stay put.
  if (preferenceIsError) return { settled: true, redirectTo: null };

  if (isAppTabId(storedTab)) return { settled: true, redirectTo: storedTab };

  // No stored tab yet: fall back to Console if the live app has panels.
  if (!liveConsoleQuery) return { settled: true, redirectTo: null };

  // A console read that ended in error settles the fallback on the current
  // tab: waiting for a success that may never come would leave the resolution
  // pending forever, blocking tab recording. Skipping the Console fallback
  // for this visit is the lesser cost.
  if (liveConsoleQuery.isError) return { settled: true, redirectTo: null };

  // While the console read is still in flight, keep waiting; settling now
  // would lock in Canvas even if the read later shows panels. An explicit tab
  // switch still settles the resolution via the user-choice branch above.
  if (!liveConsoleQuery.isSuccess) return { settled: false };

  const panels = liveConsoleQuery.data?.panels ?? [];
  return { settled: true, redirectTo: panels.length > 0 ? "console" : null };
}

type WriteAttempts = { tab: AppTabId | null; count: number };

type PersistTabOptions = {
  canvasId: string;
  tab: AppTabId;
  writeAttemptsRef: { current: WriteAttempts };
  setRecordedTab: Dispatch<SetStateAction<AppTabId | null>>;
  recordTab: ReturnType<typeof useUpdateCanvasPreference>["mutate"];
};

function persistTabWithRetryBudget({ canvasId, tab, writeAttemptsRef, setRecordedTab, recordTab }: PersistTabOptions) {
  // A fresh tab value gets a fresh attempt budget; retries of the same
  // failing value keep consuming it until the cap is hit.
  if (writeAttemptsRef.current.tab !== tab) {
    writeAttemptsRef.current = { tab, count: 0 };
  }
  if (writeAttemptsRef.current.count >= MAX_TAB_WRITE_ATTEMPTS) return;
  writeAttemptsRef.current.count += 1;

  setRecordedTab(tab);
  recordTab(
    { canvasId, lastVisitedTab: tab },
    {
      onError: () => {
        // Treating a failed write as recorded would suppress every retry;
        // clearing the guard re-runs the record effect (state change), which
        // retries the write until the attempt budget runs out.
        setRecordedTab((recorded) => (recorded === tab ? null : recorded));
      },
      onSuccess: () => {
        writeAttemptsRef.current = { tab: null, count: 0 };
      },
    },
  );
}

function applyTabToSearchParams(tab: AppTabId, setSearchParams: SetSearchParams, redirectStillPending: () => boolean) {
  setSearchParams(
    (current) => {
      if (!redirectStillPending()) return current;
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
