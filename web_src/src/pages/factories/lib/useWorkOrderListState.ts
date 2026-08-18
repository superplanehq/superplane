import { useCallback, useEffect, useMemo, useRef, useState, type Dispatch, type SetStateAction } from "react";
import {
  EMPTY_WORK_ORDER_FILTERS,
  WORK_ORDER_LAYOUTS,
  WORK_ORDER_ORDERINGS,
  WORK_ORDER_SCOPES,
  countWorkOrderFilters,
  type WorkOrderFilters,
  type WorkOrderLayoutId,
  type WorkOrderOrdering,
  type WorkOrderScope,
} from "./workOrderListModel";
import { WORK_ORDER_DISPLAY_STATUSES, type WorkOrderDisplayStatus } from "./workOrderProgress";

/** One of the three dimensions the Filter menu can narrow. */
export type WorkOrderFilterDimension = keyof WorkOrderFilters;

/**
 * Title-bar and view state for the Work Orders page.
 *
 * Layout and ordering are pure display preferences that don't reference
 * factory-specific data, so they're persisted in `localStorage` under a
 * single factory-agnostic key and follow the user everywhere.
 *
 * Scope (All/Active/My) and filters are also persisted, but namespaced per
 * factory: filters can reference factory-specific data (lines), so a value
 * chosen in one factory shouldn't silently apply — and likely hide
 * everything — in another. Search stays session-local; it expresses what
 * the user is looking for right now, not a long-term preference.
 */
export interface WorkOrderListState {
  layout: WorkOrderLayoutId;
  setLayout: (layout: WorkOrderLayoutId) => void;
  ordering: WorkOrderOrdering;
  setOrdering: (ordering: WorkOrderOrdering) => void;
  scope: WorkOrderScope;
  setScope: (scope: WorkOrderScope) => void;
  filters: WorkOrderFilters;
  toggleFilter: (dimension: WorkOrderFilterDimension, value: string) => void;
  removeFilter: (dimension: WorkOrderFilterDimension, value: string) => void;
  clearFilterDimension: (dimension: WorkOrderFilterDimension) => void;
  clearFilters: () => void;
  filterCount: number;
  search: string;
  setSearch: (value: string) => void;
  clearSearch: () => void;
  searchOpen: boolean;
  openSearch: () => void;
  closeSearch: () => void;
  filterMenuOpen: boolean;
  setFilterMenuOpen: (open: boolean) => void;
  hasActiveFilters: boolean;
  resetView: () => void;
}

const LAYOUT_STORAGE_KEY = "sp:work-orders:layout";
const ORDERING_STORAGE_KEY = "sp:work-orders:ordering";
const SCOPE_STORAGE_PREFIX = "sp:work-orders:scope";
const FILTERS_STORAGE_PREFIX = "sp:work-orders:filters";

const DEFAULT_LAYOUT: WorkOrderLayoutId = "board";
const DEFAULT_ORDERING: WorkOrderOrdering = "updated";
const DEFAULT_SCOPE: WorkOrderScope = "all";

const VALID_LAYOUTS = new Set(WORK_ORDER_LAYOUTS.map((item) => item.id));
const VALID_ORDERINGS = new Set(WORK_ORDER_ORDERINGS.map((item) => item.id));
const VALID_SCOPES = new Set(WORK_ORDER_SCOPES.map((item) => item.id));
const VALID_DISPLAY_STATUSES = new Set<string>(WORK_ORDER_DISPLAY_STATUSES);

/** Per-factory key, falling back to a bare key when `factoryId` is unavailable. */
function scopedStorageKey(prefix: string, factoryId: string): string {
  return factoryId ? `${prefix}:${factoryId}` : prefix;
}

function readPersisted<T>(key: string, valid: Set<T>, fallback: T): T {
  if (typeof window === "undefined") {
    return fallback;
  }
  try {
    const stored = window.localStorage.getItem(key);
    if (!stored) {
      return fallback;
    }
    return valid.has(stored as T) ? (stored as T) : fallback;
  } catch {
    return fallback;
  }
}

function writePersisted(key: string, value: string) {
  if (typeof window === "undefined") {
    return;
  }
  try {
    window.localStorage.setItem(key, value);
  } catch {
    // Persistence is best-effort; ignore quota or privacy-mode failures.
  }
}

/** Narrows an unknown array down to the statuses recognized today, dropping the rest. */
function sanitizeStatuses(value: unknown): WorkOrderDisplayStatus[] {
  if (!Array.isArray(value)) {
    return [];
  }
  return value.filter(
    (entry): entry is WorkOrderDisplayStatus => typeof entry === "string" && VALID_DISPLAY_STATUSES.has(entry),
  );
}

/** Opaque string arrays: kept as-is, stale IDs just render an "Unknown" chip. */
function sanitizeIds(value: unknown): string[] {
  if (!Array.isArray(value)) {
    return [];
  }
  return value.filter((entry): entry is string => typeof entry === "string");
}

function readPersistedFilters(key: string): WorkOrderFilters {
  if (typeof window === "undefined") {
    return EMPTY_WORK_ORDER_FILTERS;
  }
  try {
    const stored = window.localStorage.getItem(key);
    if (!stored) {
      return EMPTY_WORK_ORDER_FILTERS;
    }
    const parsed = JSON.parse(stored) as Partial<WorkOrderFilters> | null;
    if (!parsed || typeof parsed !== "object") {
      return EMPTY_WORK_ORDER_FILTERS;
    }
    return {
      statuses: sanitizeStatuses(parsed.statuses),
      lineIds: sanitizeIds(parsed.lineIds),
      assigneeIds: sanitizeIds(parsed.assigneeIds),
    };
  } catch {
    return EMPTY_WORK_ORDER_FILTERS;
  }
}

function writePersistedFilters(key: string, filters: WorkOrderFilters) {
  if (typeof window === "undefined") {
    return;
  }
  try {
    window.localStorage.setItem(key, JSON.stringify(filters));
  } catch {
    // Persistence is best-effort; ignore quota or privacy-mode failures.
  }
}

/**
 * Owns `scope` and `filters`, namespacing their storage per `factoryId` and
 * resetting `search`/`searchOpen`/`filterMenuOpen` whenever `factoryId`
 * changes (the Work Orders route can be revisited across factories without
 * necessarily remounting its page component).
 *
 * Reset-on-factory-change and persist-on-value-change are both driven by
 * effects, so a "suppress" ref pair guards against the two racing in the
 * commit where `factoryId` changes: without it, the persist effects would
 * see the *old* factory's scope/filters values (state hasn't re-rendered
 * yet) paired with the *new* `factoryId`, and clobber the new factory's
 * storage before the reset effect's reads land.
 */
function usePersistedScopeAndFilters(
  factoryId: string,
  onFactoryChange: () => void,
): {
  scope: WorkOrderScope;
  setScope: (scope: WorkOrderScope) => void;
  filters: WorkOrderFilters;
  setFilters: Dispatch<SetStateAction<WorkOrderFilters>>;
} {
  const [scope, setScope] = useState<WorkOrderScope>(() =>
    readPersisted(scopedStorageKey(SCOPE_STORAGE_PREFIX, factoryId), VALID_SCOPES, DEFAULT_SCOPE),
  );
  const [filters, setFilters] = useState<WorkOrderFilters>(() =>
    readPersistedFilters(scopedStorageKey(FILTERS_STORAGE_PREFIX, factoryId)),
  );

  const previousFactoryIdRef = useRef(factoryId);
  const suppressScopeWriteRef = useRef(false);
  const suppressFiltersWriteRef = useRef(false);

  useEffect(() => {
    if (previousFactoryIdRef.current === factoryId) {
      return;
    }
    previousFactoryIdRef.current = factoryId;
    suppressScopeWriteRef.current = true;
    suppressFiltersWriteRef.current = true;
    setScope(readPersisted(scopedStorageKey(SCOPE_STORAGE_PREFIX, factoryId), VALID_SCOPES, DEFAULT_SCOPE));
    setFilters(readPersistedFilters(scopedStorageKey(FILTERS_STORAGE_PREFIX, factoryId)));
    onFactoryChange();
    // `onFactoryChange` intentionally excluded: callers pass a fresh closure
    // each render, and re-running this effect for that alone would defeat
    // the factoryId-transition check above.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [factoryId]);

  useEffect(() => {
    if (suppressScopeWriteRef.current) {
      suppressScopeWriteRef.current = false;
      return;
    }
    writePersisted(scopedStorageKey(SCOPE_STORAGE_PREFIX, factoryId), scope);
  }, [scope, factoryId]);

  useEffect(() => {
    if (suppressFiltersWriteRef.current) {
      suppressFiltersWriteRef.current = false;
      return;
    }
    writePersistedFilters(scopedStorageKey(FILTERS_STORAGE_PREFIX, factoryId), filters);
  }, [filters, factoryId]);

  return { scope, setScope, filters, setFilters };
}

export function useWorkOrderListState(factoryId: string): WorkOrderListState {
  const [layout, setLayoutState] = useState<WorkOrderLayoutId>(() =>
    readPersisted(LAYOUT_STORAGE_KEY, VALID_LAYOUTS, DEFAULT_LAYOUT),
  );
  const [ordering, setOrderingState] = useState<WorkOrderOrdering>(() =>
    readPersisted(ORDERING_STORAGE_KEY, VALID_ORDERINGS, DEFAULT_ORDERING),
  );
  const [search, setSearch] = useState("");
  const [searchOpen, setSearchOpen] = useState(false);
  const [filterMenuOpen, setFilterMenuOpen] = useState(false);

  const { scope, setScope, filters, setFilters } = usePersistedScopeAndFilters(factoryId, () => {
    setSearch("");
    setSearchOpen(false);
    setFilterMenuOpen(false);
  });

  useEffect(() => {
    writePersisted(LAYOUT_STORAGE_KEY, layout);
  }, [layout]);

  useEffect(() => {
    writePersisted(ORDERING_STORAGE_KEY, ordering);
  }, [ordering]);

  const setLayout = useCallback((next: WorkOrderLayoutId) => {
    setLayoutState(next);
  }, []);

  const setOrdering = useCallback((next: WorkOrderOrdering) => {
    setOrderingState(next);
  }, []);

  const toggleFilter = useCallback(
    (dimension: WorkOrderFilterDimension, value: string) => {
      setFilters((current) => {
        const values = current[dimension] as string[];
        const next = values.includes(value) ? values.filter((entry) => entry !== value) : [...values, value];
        return { ...current, [dimension]: next } as WorkOrderFilters;
      });
    },
    [setFilters],
  );

  const removeFilter = useCallback(
    (dimension: WorkOrderFilterDimension, value: string) => {
      setFilters((current) => {
        const values = current[dimension] as string[];
        return { ...current, [dimension]: values.filter((entry) => entry !== value) } as WorkOrderFilters;
      });
    },
    [setFilters],
  );

  const clearFilterDimension = useCallback(
    (dimension: WorkOrderFilterDimension) => {
      setFilters((current) => ({ ...current, [dimension]: [] }) as WorkOrderFilters);
    },
    [setFilters],
  );

  const clearFilters = useCallback(() => {
    setFilters(EMPTY_WORK_ORDER_FILTERS);
  }, [setFilters]);

  const clearSearch = useCallback(() => {
    setSearch("");
  }, []);

  const openSearch = useCallback(() => {
    setSearchOpen(true);
  }, []);

  const closeSearch = useCallback(() => {
    setSearchOpen(false);
    setSearch("");
  }, []);

  const resetView = useCallback(() => {
    setScope(DEFAULT_SCOPE);
    setFilters(EMPTY_WORK_ORDER_FILTERS);
    setSearch("");
  }, [setScope, setFilters]);

  const filterCount = countWorkOrderFilters(filters);

  const hasActiveFilters = useMemo(
    () => scope !== DEFAULT_SCOPE || filterCount > 0 || search.trim().length > 0,
    [scope, filterCount, search],
  );

  return {
    layout,
    setLayout,
    ordering,
    setOrdering,
    scope,
    setScope,
    filters,
    toggleFilter,
    removeFilter,
    clearFilterDimension,
    clearFilters,
    filterCount,
    search,
    setSearch,
    clearSearch,
    searchOpen,
    openSearch,
    closeSearch,
    filterMenuOpen,
    setFilterMenuOpen,
    hasActiveFilters,
    resetView,
  };
}
