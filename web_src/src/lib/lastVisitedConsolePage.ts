/**
 * Per-canvas persistence for the console's active page id. Mirrors the
 * shape of {@link ./lastVisitedAppTab} — one localStorage entry holding
 * `Record<canvasId, pageId>` — so entering console mode without an
 * explicit `?page=` param can restore the last tab the user was on.
 *
 * The resolver ({@link resolveActiveConsolePage}) treats a stale stored
 * id (page removed or renamed) as absent and falls back to the first
 * page rather than surfacing an error. This is a UX-only convenience,
 * so every failure mode silently no-ops.
 */
export const LAST_VISITED_CONSOLE_PAGE_STORAGE_KEY = "superplane:last-visited-console-page";

type LastVisitedConsolePageByCanvas = Record<string, string>;

function readAll(): LastVisitedConsolePageByCanvas {
  if (typeof window === "undefined") return {};

  try {
    const raw = window.localStorage.getItem(LAST_VISITED_CONSOLE_PAGE_STORAGE_KEY);
    if (!raw) return {};

    const parsed = JSON.parse(raw) as unknown;
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) return {};

    const result: LastVisitedConsolePageByCanvas = {};
    for (const [canvasId, pageId] of Object.entries(parsed as Record<string, unknown>)) {
      if (typeof pageId === "string" && pageId.length > 0) {
        result[canvasId] = pageId;
      }
    }
    return result;
  } catch {
    return {};
  }
}

export function readLastVisitedConsolePage(canvasId: string): string | null {
  if (!canvasId) return null;
  return readAll()[canvasId] ?? null;
}

export function recordLastVisitedConsolePage(canvasId: string, pageId: string): void {
  if (!canvasId || !pageId || typeof window === "undefined") return;

  try {
    const all = readAll();
    all[canvasId] = pageId;
    window.localStorage.setItem(LAST_VISITED_CONSOLE_PAGE_STORAGE_KEY, JSON.stringify(all));
  } catch {
    // Last-visited persistence is optional — best-effort only.
  }
}

/**
 * Resolve the active console page id from the priority chain:
 *   1. explicit valid `?page=` param
 *   2. stored last-visited page (if it still exists)
 *   3. first page in the list
 *   4. `null` (empty console → caller decides UI)
 *
 * A `pageParam` or stored id that no longer matches any page in
 * `availablePageIds` is silently ignored so a stale URL / storage entry
 * never blocks entering console mode.
 */
export function resolveActiveConsolePage({
  canvasId,
  pageParam,
  availablePageIds,
}: {
  canvasId: string;
  pageParam: string | null | undefined;
  availablePageIds: string[];
}): string | null {
  if (availablePageIds.length === 0) return null;

  if (pageParam && availablePageIds.includes(pageParam)) {
    return pageParam;
  }

  const stored = readLastVisitedConsolePage(canvasId);
  if (stored && availablePageIds.includes(stored)) {
    return stored;
  }

  return availablePageIds[0] ?? null;
}
