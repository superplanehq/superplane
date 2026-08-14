import { useCallback, useEffect, useMemo, useState } from "react";
import {
  EMPTY_WORK_ORDER_FILTERS,
  WORK_ORDER_LAYOUTS,
  WORK_ORDER_ORDERINGS,
  countWorkOrderFilters,
  type WorkOrderFilters,
  type WorkOrderLayoutId,
  type WorkOrderOrdering,
  type WorkOrderScope,
} from "./workOrderListModel";

/** One of the three dimensions the Filter menu can narrow. */
export type WorkOrderFilterDimension = keyof WorkOrderFilters;

/**
 * Title-bar and view state for the Work Orders page.
 *
 * Layout and ordering are the two knobs users tune once and expect to
 * carry across sessions, so we persist them in `localStorage` under a
 * factory-agnostic key. Scope, filters, and search are intentionally
 * session-local — they express what the user is looking at right now, not
 * a long-term preference.
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

const DEFAULT_LAYOUT: WorkOrderLayoutId = "board";
const DEFAULT_ORDERING: WorkOrderOrdering = "updated";
const DEFAULT_SCOPE: WorkOrderScope = "all";

const VALID_LAYOUTS = new Set(WORK_ORDER_LAYOUTS.map((item) => item.id));
const VALID_ORDERINGS = new Set(WORK_ORDER_ORDERINGS.map((item) => item.id));

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

export function useWorkOrderListState(): WorkOrderListState {
  const [layout, setLayoutState] = useState<WorkOrderLayoutId>(() =>
    readPersisted(LAYOUT_STORAGE_KEY, VALID_LAYOUTS, DEFAULT_LAYOUT),
  );
  const [ordering, setOrderingState] = useState<WorkOrderOrdering>(() =>
    readPersisted(ORDERING_STORAGE_KEY, VALID_ORDERINGS, DEFAULT_ORDERING),
  );
  const [scope, setScope] = useState<WorkOrderScope>(DEFAULT_SCOPE);
  const [filters, setFilters] = useState<WorkOrderFilters>(EMPTY_WORK_ORDER_FILTERS);
  const [search, setSearch] = useState("");
  const [searchOpen, setSearchOpen] = useState(false);
  const [filterMenuOpen, setFilterMenuOpen] = useState(false);

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

  const toggleFilter = useCallback((dimension: WorkOrderFilterDimension, value: string) => {
    setFilters((current) => {
      const values = current[dimension] as string[];
      const next = values.includes(value) ? values.filter((entry) => entry !== value) : [...values, value];
      return { ...current, [dimension]: next } as WorkOrderFilters;
    });
  }, []);

  const removeFilter = useCallback((dimension: WorkOrderFilterDimension, value: string) => {
    setFilters((current) => {
      const values = current[dimension] as string[];
      return { ...current, [dimension]: values.filter((entry) => entry !== value) } as WorkOrderFilters;
    });
  }, []);

  const clearFilterDimension = useCallback((dimension: WorkOrderFilterDimension) => {
    setFilters((current) => ({ ...current, [dimension]: [] }) as WorkOrderFilters);
  }, []);

  const clearFilters = useCallback(() => {
    setFilters(EMPTY_WORK_ORDER_FILTERS);
  }, []);

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
  }, []);

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
