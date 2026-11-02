import { RUN_STATUS_FILTER_OPTIONS, type RunStatusFilter } from "@/ui/Runs/runPresentation";

export const RUNS_SIDEBAR_FILTERS_STORAGE_KEY = "runs-sidebar-filters";

export type PersistedFilters = {
  statuses: Set<RunStatusFilter>;
  triggerIds: Set<string>;
};

export function loadPersistedFilters(): PersistedFilters {
  if (typeof window === "undefined") return { statuses: new Set(), triggerIds: new Set() };

  try {
    const raw = window.localStorage.getItem(RUNS_SIDEBAR_FILTERS_STORAGE_KEY);
    if (!raw) return { statuses: new Set(), triggerIds: new Set() };

    const parsed = JSON.parse(raw) as { statuses?: unknown; triggerIds?: unknown };
    const validStatuses = new Set<RunStatusFilter>(RUN_STATUS_FILTER_OPTIONS.map((option) => option.id));
    const statuses = new Set<RunStatusFilter>(
      Array.isArray(parsed.statuses)
        ? parsed.statuses.filter(
            (status: unknown): status is RunStatusFilter =>
              typeof status === "string" && validStatuses.has(status as RunStatusFilter),
          )
        : [],
    );
    const triggerIds = new Set<string>(
      Array.isArray(parsed.triggerIds)
        ? parsed.triggerIds.filter((triggerId: unknown): triggerId is string => typeof triggerId === "string")
        : [],
    );

    return { statuses, triggerIds };
  } catch {
    return { statuses: new Set(), triggerIds: new Set() };
  }
}

export function savePersistedFilters(filters: PersistedFilters): void {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.setItem(
      RUNS_SIDEBAR_FILTERS_STORAGE_KEY,
      JSON.stringify({
        statuses: Array.from(filters.statuses),
        triggerIds: Array.from(filters.triggerIds),
      }),
    );
  } catch {
    // Filter persistence is optional.
  }
}
