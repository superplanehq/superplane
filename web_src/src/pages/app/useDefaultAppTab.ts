import { useEffect, useRef, useState } from "react";
import type { UseQueryResult } from "@tanstack/react-query";
import { useCanvasConsole } from "@/hooks/useCanvasData";
import type { CanvasConsoleData } from "@/hooks/useCanvasData";
import { type AppTabId, isAppTabId, readLastVisitedAppTab, recordLastVisitedAppTab } from "@/lib/lastVisitedAppTab";

type UrlViewFlags = {
  isRunInspectionMode: boolean;
  isMemoryMode: boolean;
  isFilesMode: boolean;
  isConsoleMode: boolean;
};

/** Maps the URL view flags to the tab identifier. Returns null while run inspection is active. */
function urlViewFlagsToTab(flags: UrlViewFlags): AppTabId | null {
  if (flags.isRunInspectionMode) return null;
  if (flags.isConsoleMode) return "console";
  if (flags.isMemoryMode) return "memory";
  if (flags.isFilesMode) return "files";
  return "canvas";
}

// Query params that pin the URL to a destination. `view` and `run` select a
// tab directly; `version` (version preview), `edit` (edit-session entry),
// `sidebar`/`node` (node selection), and `file` (file selection) deep-link
// into a specific spot. A default-tab redirect would pull the user away from
// any of them — and would even delete the selection params outright.
const TAB_SELECTION_PARAMS = ["view", "run"] as const;
const DEEP_LINK_PARAMS = ["version", "edit", "sidebar", "node", "file"] as const;

function hasAnyParam(searchParams: URLSearchParams, params: readonly string[]): boolean {
  return params.some((param) => (searchParams.get(param) ?? "") !== "");
}

/** A tab-selecting or deep-link param means there is no default tab to resolve. */
function urlPinsNavigation(searchParams: URLSearchParams): boolean {
  return hasAnyParam(searchParams, TAB_SELECTION_PARAMS) || hasAnyParam(searchParams, DEEP_LINK_PARAMS);
}

/**
 * A deep link without an explicit `view` lands on a tab the user did not
 * actively pick, so that landing must not be persisted as a tab change.
 * (`run` needs no handling here: it maps to no tab at all, and closing run
 * inspection already skips recording via its own guard.)
 */
function urlDeepLinksWithoutTabPick(searchParams: URLSearchParams): boolean {
  return hasAnyParam(searchParams, DEEP_LINK_PARAMS) && !hasAnyParam(searchParams, ["view"]);
}

type ConsoleQueryLike = Pick<UseQueryResult<CanvasConsoleData | undefined>, "data" | "isSuccess" | "isError">;

type SetSearchParams = (
  next: URLSearchParams | ((prev: URLSearchParams) => URLSearchParams),
  options?: { replace?: boolean },
) => void;

type UseDefaultAppTabOptions = {
  canvasId: string | undefined;
  urlViewFlags: UrlViewFlags;
  searchParams: URLSearchParams;
  setSearchParams: SetSearchParams;
};

/**
 * Persists the current app tab to localStorage and, on initial navigation to
 * an app without a tab-selecting or deep-link query param, redirects to the
 * user's last visited tab, falling back to Console when the app has panels
 * and Canvas otherwise.
 */
export function useDefaultAppTab({ canvasId, urlViewFlags, searchParams, setSearchParams }: UseDefaultAppTabOptions) {
  // First-visit Console defaulting must consult the live console (versionId
  // undefined), not the active version's: the active version can be a draft
  // (e.g. from `?version=`) whose console is empty while the live app has
  // widgets, and first-visit defaulting is about what the app publishes. The
  // query key matches other live-console reads, so this usually dedupes.
  const liveConsoleQuery: ConsoleQueryLike = useCanvasConsole(canvasId ?? "", undefined);

  const currentTab = urlViewFlagsToTab(urlViewFlags);
  // Whether the default-tab redirect is settled for this app instance. Held
  // in state rather than a ref: resolution can settle without a URL change
  // (e.g. a console read that errors or reports no panels), and the record
  // effect below must re-run when that happens.
  const [redirectResolved, setRedirectResolved] = useState(() => urlPinsNavigation(searchParams));
  // Whether this app instance was entered through a deep link that lands on a
  // tab without the user picking it (see urlDeepLinksWithoutTabPick). The
  // record effect consumes this once to skip persisting that landing.
  const deepLinkLandingRef = useRef(urlDeepLinksWithoutTabPick(searchParams));
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
  // tabs while the Console-fallback console query is still loading, the
  // redirect must yield to that explicit choice instead of forcing Console
  // later. Reads of the stored tab are synchronous and settle on first
  // render, so this guard only matters for the async console fallback.
  const mountTabRef = useRef(currentTab);

  // The refs above hold state for a single app. React Router reuses the same
  // AppPage instance when navigating between apps (e.g. via the command
  // palette), so reset them whenever the canvas changes; otherwise the new
  // app would skip its default-tab redirect. The render-phase setState is
  // React's sanctioned way to reset state when a prop changes.
  const refsOwnerCanvasIdRef = useRef(canvasId);
  if (refsOwnerCanvasIdRef.current !== canvasId) {
    refsOwnerCanvasIdRef.current = canvasId;
    pendingRedirectRef.current = null;
    inRunInspectionRef.current = false;
    deepLinkLandingRef.current = urlDeepLinksWithoutTabPick(searchParams);
    mountTabRef.current = currentTab;
    setRedirectResolved(urlPinsNavigation(searchParams));
  }

  // Default-tab resolution: applied at most once per mount.
  useEffect(() => {
    if (redirectResolved) return;
    if (!canvasId) return;

    // A pinning param can appear while the console read is still in flight
    // (e.g. the user opens a node, a version preview, or an edit session on
    // Canvas — none of which change the tab). Redirecting after that would
    // pull the user away and strip the selection params, so settle without a
    // redirect instead.
    if (urlPinsNavigation(searchParams)) {
      setRedirectResolved(true);
      return;
    }

    const storedTab = readLastVisitedAppTab(canvasId);
    const resolution = resolveDefaultTab({
      // The user already navigated to another tab while the Console fallback
      // was still resolving; their explicit choice wins over the redirect.
      userSwitchedTabs: currentTab !== mountTabRef.current,
      storedTab,
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
  }, [redirectResolved, canvasId, currentTab, liveConsoleQuery, searchParams, setSearchParams]);

  // Record tab changes to localStorage once the initial redirect has settled.
  useEffect(() => {
    if (!canvasId) return;
    if (currentTab === null) {
      inRunInspectionRef.current = true;
      return;
    }
    if (!redirectResolved) return;

    // Closing run inspection lands on a tab the user did not actively pick;
    // do not persist that landing as a tab change.
    if (inRunInspectionRef.current) {
      inRunInspectionRef.current = false;
      return;
    }

    // Likewise for a deep-link landing (`version`/`edit`/`sidebar`/`node`/
    // `file` without `view`): the user followed a link to a spot, not to a
    // tab, so the landing must not replace their stored tab. Later deliberate
    // tab changes are recorded as usual.
    if (deepLinkLandingRef.current) {
      deepLinkLandingRef.current = false;
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

    // Avoid rewriting an identical value.
    if (readLastVisitedAppTab(canvasId) === currentTab) return;

    recordLastVisitedAppTab(canvasId, currentTab);
  }, [redirectResolved, canvasId, currentTab]);
}

type DefaultTabResolution = { settled: false } | { settled: true; redirectTo: AppTabId | null };

type ResolveDefaultTabInput = {
  userSwitchedTabs: boolean;
  storedTab: AppTabId | null;
  liveConsoleQuery: ConsoleQueryLike;
};

/**
 * Decides whether default-tab resolution can settle and, if so, which tab (if
 * any) to redirect to. Pure decision logic; the caller applies the redirect.
 */
function resolveDefaultTab({
  userSwitchedTabs,
  storedTab,
  liveConsoleQuery,
}: ResolveDefaultTabInput): DefaultTabResolution {
  if (userSwitchedTabs) return { settled: true, redirectTo: null };

  if (isAppTabId(storedTab)) return { settled: true, redirectTo: storedTab };

  // No stored tab yet: fall back to Console if the live app has panels.
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
