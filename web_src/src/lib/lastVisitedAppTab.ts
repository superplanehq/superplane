export const LAST_VISITED_APP_TAB_STORAGE_KEY = "superplane:last-visited-app-tab";

export const APP_TAB_VALUES = ["canvas", "console", "memory", "files"] as const;

export type AppTabId = (typeof APP_TAB_VALUES)[number];

export function isAppTabId(value: unknown): value is AppTabId {
  return typeof value === "string" && (APP_TAB_VALUES as readonly string[]).includes(value);
}

type LastVisitedAppTabByCanvas = Record<string, AppTabId>;

function readAllLastVisitedAppTabs(): LastVisitedAppTabByCanvas {
  if (typeof window === "undefined") {
    return {};
  }

  try {
    const raw = window.localStorage.getItem(LAST_VISITED_APP_TAB_STORAGE_KEY);
    if (!raw) {
      return {};
    }

    const parsed = JSON.parse(raw) as unknown;
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
      return {};
    }

    const result: LastVisitedAppTabByCanvas = {};
    for (const [canvasId, tab] of Object.entries(parsed as Record<string, unknown>)) {
      if (isAppTabId(tab)) {
        result[canvasId] = tab;
      }
    }

    return result;
  } catch {
    return {};
  }
}

export function readLastVisitedAppTab(canvasId: string): AppTabId | null {
  if (!canvasId) {
    return null;
  }

  return readAllLastVisitedAppTabs()[canvasId] ?? null;
}

export function recordLastVisitedAppTab(canvasId: string, tab: AppTabId): void {
  if (!canvasId || !isAppTabId(tab) || typeof window === "undefined") {
    return;
  }

  try {
    const all = readAllLastVisitedAppTabs();
    all[canvasId] = tab;
    window.localStorage.setItem(LAST_VISITED_APP_TAB_STORAGE_KEY, JSON.stringify(all));
  } catch {
    // Last-visited persistence is optional.
  }
}
