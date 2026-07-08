import { useEffect, useRef } from "react";
import type { UseQueryResult } from "@tanstack/react-query";
import { useUpdateCanvasPreference, useCanvasPreference } from "@/hooks/useCanvasData";
import type { CanvasConsoleData } from "@/hooks/useCanvasData";

export type AppTabId = "canvas" | "console" | "memory" | "files";

const APP_TAB_VALUES: readonly AppTabId[] = ["canvas", "console", "memory", "files"] as const;

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

type ConsoleQueryLike = Pick<UseQueryResult<CanvasConsoleData | undefined>, "data" | "isPending" | "isLoading">;

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

  const currentTab = urlViewFlagsToTab(urlViewFlags);
  const lastRecordedTabRef = useRef<AppTabId | null>(null);
  const redirectResolvedRef = useRef<boolean | null>(null);
  // Tab a just-scheduled redirect is navigating to. `setSearchParams` only
  // lands on the next render, so between scheduling and landing the URL still
  // reports the pre-redirect tab; the record effect must not persist it.
  const pendingRedirectTabRef = useRef<AppTabId | null>(null);

  // Snapshot on mount whether the URL already selected a tab or a run; if it
  // did, the default-tab redirect is skipped for this app instance. Deriving
  // the initial value lazily keeps this a single mount-time check without
  // needing to watch `searchParams` changes.
  if (redirectResolvedRef.current === null) {
    const hasView = (searchParams.get("view") ?? "") !== "";
    const hasRun = (searchParams.get("run") ?? "") !== "";
    redirectResolvedRef.current = hasView || hasRun;
  }

  // Default-tab resolution: applied at most once per mount.
  useEffect(() => {
    if (redirectResolvedRef.current) return;
    if (!organizationId || !canvasId) return;
    if (preferenceQuery.isPending) return;

    const storedTab = preferenceQuery.data?.lastVisitedTab;
    if (isAppTabId(storedTab)) {
      redirectResolvedRef.current = true;
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
      redirectResolvedRef.current = true;
      return;
    }
    if (consoleQuery.isPending || consoleQuery.isLoading) return;

    redirectResolvedRef.current = true;
    const panels = consoleQuery.data?.panels ?? [];
    if (panels.length > 0) {
      pendingRedirectTabRef.current = "console";
      applyTabToSearchParams("console", setSearchParams);
    }
  }, [
    organizationId,
    canvasId,
    currentTab,
    preferenceQuery.isPending,
    preferenceQuery.data,
    consoleQuery,
    setSearchParams,
  ]);

  // Record tab changes to the backend once the initial redirect has settled.
  useEffect(() => {
    if (!organizationId || !canvasId) return;
    if (currentTab === null) return;
    if (!redirectResolvedRef.current) return;

    // A redirect was scheduled but the URL hasn't caught up yet — this effect
    // can run in the same commit that scheduled it, with `currentTab` still
    // reporting the pre-redirect tab. Recording now would overwrite the
    // stored tab with the tab we are navigating away from.
    if (pendingRedirectTabRef.current !== null) {
      if (currentTab !== pendingRedirectTabRef.current) return;
      pendingRedirectTabRef.current = null;
    }

    if (lastRecordedTabRef.current === currentTab) return;

    // Avoid writing an identical value back to the server on first render.
    const storedTab = preferenceQuery.data?.lastVisitedTab;
    if (lastRecordedTabRef.current === null && storedTab === currentTab) {
      lastRecordedTabRef.current = currentTab;
      return;
    }

    lastRecordedTabRef.current = currentTab;
    updatePreferenceMutation.mutate({ canvasId, lastVisitedTab: currentTab });
  }, [canvasId, currentTab, organizationId, preferenceQuery.data, updatePreferenceMutation]);
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
