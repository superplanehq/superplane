import { type AppTabId, isAppTabId } from "@/lib/lastVisitedAppTab";

/** URL view flags derived from search params (see viewState.ts). */
export type UrlViewFlags = {
  isRunInspectionMode: boolean;
  isMemoryMode: boolean;
  isFilesMode: boolean;
  isConsoleMode: boolean;
};

/**
 * Query params that pin the URL to a destination. A tab-selecting `view` and
 * `run` select a destination directly; `version` (version preview), `edit`
 * (edit-session entry), `sidebar`/`node` (node selection), and `file` (file
 * selection) deep-link into a specific spot. A default-tab redirect must not
 * pull the user away from any of them.
 */
const DEEP_LINK_PARAMS = ["version", "edit", "sidebar", "node", "file"] as const;

/**
 * `view` values that actually select a tab (`dashboard` is the legacy alias
 * for Console and is rewritten to it on mount). Legacy values that select no
 * tab (`runs`, `versions`) are deleted on mount by
 * useWorkflowViewSearchParams, so they must not pin navigation: the stored-tab
 * redirect and the Console fallback should still apply for that visit.
 */
const TAB_SELECTING_VIEW_VALUES = ["console", "dashboard", "memory", "files"] as const;

function hasAnyParam(searchParams: URLSearchParams, params: readonly string[]): boolean {
  return params.some((param) => (searchParams.get(param) ?? "") !== "");
}

function viewParamSelectsTab(searchParams: URLSearchParams): boolean {
  const view = searchParams.get("view") ?? "";
  return (TAB_SELECTING_VIEW_VALUES as readonly string[]).includes(view);
}

/** Maps URL view flags to the tab identifier. Returns null while run inspection is active. */
export function urlViewFlagsToTab(flags: UrlViewFlags): AppTabId | null {
  if (flags.isRunInspectionMode) return null;
  if (flags.isConsoleMode) return "console";
  if (flags.isMemoryMode) return "memory";
  if (flags.isFilesMode) return "files";
  return "canvas";
}

/** A tab-selecting or deep-link param means there is no default tab to resolve. */
export function urlPinsNavigation(searchParams: URLSearchParams): boolean {
  return (
    viewParamSelectsTab(searchParams) ||
    hasAnyParam(searchParams, ["run"]) ||
    hasAnyParam(searchParams, DEEP_LINK_PARAMS)
  );
}

/**
 * A deep link without a tab-selecting `view` lands on a tab the user did not
 * actively pick, so persistence must not treat that landing as a tab change.
 * (`run` maps to no tab at all, and closing run inspection has its own guard.)
 */
export function urlDeepLinksWithoutTabPick(searchParams: URLSearchParams): boolean {
  return hasAnyParam(searchParams, DEEP_LINK_PARAMS) && !viewParamSelectsTab(searchParams);
}

/**
 * Builds the search params for landing on `tab`: sets/clears `view` and strips
 * selection params that only make sense on the previous tab.
 */
export function buildAppTabSearchParams(tab: AppTabId, current: URLSearchParams): URLSearchParams {
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
}

/** Minimal shape of the live-console query the resolver depends on. */
export type ConsoleQueryLike = {
  isSuccess: boolean;
  isError: boolean;
  data: { panels: unknown[] } | undefined;
};

export type DefaultTabResolution = { settled: false } | { settled: true; redirectTo: AppTabId | null };

/**
 * Decides which tab (if any) should be the destination on a bare app URL.
 * Priority: stored tab → live console (Console when the app has panels,
 * Canvas otherwise). A console read that errors settles on Canvas so the
 * caller does not wait forever.
 */
export function resolveDefaultTab({
  storedTab,
  liveConsoleQuery,
}: {
  storedTab: AppTabId | null;
  liveConsoleQuery: ConsoleQueryLike;
}): DefaultTabResolution {
  if (isAppTabId(storedTab)) return { settled: true, redirectTo: storedTab };

  if (liveConsoleQuery.isError) return { settled: true, redirectTo: null };

  if (!liveConsoleQuery.isSuccess) return { settled: false };

  const panels = liveConsoleQuery.data?.panels ?? [];
  return { settled: true, redirectTo: panels.length > 0 ? "console" : null };
}
