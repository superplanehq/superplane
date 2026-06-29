export const RECENT_CANVAS_OPENS_STORAGE_KEY = "recent-canvas-opens";

const MAX_RECENT_CANVAS_OPENS = 50;

export type RecentCanvasOpen = {
  canvasId: string;
  openedAt: number;
};

type RecentCanvasOpensByOrg = Record<string, RecentCanvasOpen[]>;

function readAllRecentCanvasOpens(): RecentCanvasOpensByOrg {
  if (typeof window === "undefined") {
    return {};
  }

  try {
    const raw = window.localStorage.getItem(RECENT_CANVAS_OPENS_STORAGE_KEY);
    if (!raw) {
      return {};
    }

    const parsed = JSON.parse(raw) as unknown;
    if (!parsed || typeof parsed !== "object") {
      return {};
    }

    const result: RecentCanvasOpensByOrg = {};
    for (const [organizationId, entries] of Object.entries(parsed as Record<string, unknown>)) {
      if (!Array.isArray(entries)) {
        continue;
      }

      result[organizationId] = entries
        .map((entry): RecentCanvasOpen | null => {
          if (!entry || typeof entry !== "object") {
            return null;
          }

          const canvasId = (entry as { canvasId?: unknown }).canvasId;
          const openedAt = (entry as { openedAt?: unknown }).openedAt;
          if (typeof canvasId !== "string" || typeof openedAt !== "number") {
            return null;
          }

          return { canvasId, openedAt };
        })
        .filter((entry): entry is RecentCanvasOpen => entry !== null);
    }

    return result;
  } catch {
    return {};
  }
}

function writeAllRecentCanvasOpens(data: RecentCanvasOpensByOrg): void {
  if (typeof window === "undefined") {
    return;
  }

  try {
    window.localStorage.setItem(RECENT_CANVAS_OPENS_STORAGE_KEY, JSON.stringify(data));
  } catch {
    // Recent-open persistence is optional.
  }
}

export function loadRecentCanvasOpens(organizationId: string): RecentCanvasOpen[] {
  if (!organizationId) {
    return [];
  }

  return readAllRecentCanvasOpens()[organizationId] ?? [];
}

export function recordRecentCanvasOpen(organizationId: string, canvasId: string): RecentCanvasOpen[] {
  if (!organizationId || !canvasId) {
    return [];
  }

  const all = readAllRecentCanvasOpens();
  const existing = all[organizationId] ?? [];
  const openedAt = Date.now();
  const next = [{ canvasId, openedAt }, ...existing.filter((entry) => entry.canvasId !== canvasId)].slice(
    0,
    MAX_RECENT_CANVAS_OPENS,
  );

  all[organizationId] = next;
  writeAllRecentCanvasOpens(all);
  return next;
}

export type CanvasProjectOption = {
  id: string;
  name: string;
};

export function sortCanvasProjectsByRecentOpen(
  projects: CanvasProjectOption[],
  recentOpens: RecentCanvasOpen[],
): CanvasProjectOption[] {
  const recentOrder = new Map(recentOpens.map((entry, index) => [entry.canvasId, index]));

  return [...projects].sort((left, right) => {
    const leftRecent = recentOrder.get(left.id);
    const rightRecent = recentOrder.get(right.id);

    if (leftRecent !== undefined && rightRecent !== undefined) {
      return leftRecent - rightRecent;
    }

    if (leftRecent !== undefined) {
      return -1;
    }

    if (rightRecent !== undefined) {
      return 1;
    }

    return left.name.localeCompare(right.name);
  });
}
