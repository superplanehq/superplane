import { useCallback, useState } from "react";

export const REFINE_LAYOUT_STORAGE_KEY = "sp:refine:layout";

/** Which score summary is open in the composer stack. One drawer is shared. */
export type RefineSummaryKind = "clarity" | "confidence";

export type RefineLayoutPreference = {
  openSummary: RefineSummaryKind | null;
  planOpen: boolean;
};

const DEFAULT_LAYOUT: RefineLayoutPreference = {
  openSummary: "clarity",
  planOpen: false,
};

function isBoolean(value: unknown): value is boolean {
  return value === true || value === false;
}

function isSummaryKind(value: unknown): value is RefineSummaryKind {
  return value === "clarity" || value === "confidence";
}

/**
 * Read the stored summary choice. Older layouts stored `clarityExpanded`;
 * map that to the Clarity drawer being open or closed.
 */
function readOpenSummary(record: Record<string, unknown>): RefineSummaryKind | null {
  if (isSummaryKind(record.openSummary) || record.openSummary === null) {
    return record.openSummary;
  }
  if (isBoolean(record.clarityExpanded)) {
    return record.clarityExpanded ? "clarity" : null;
  }
  return DEFAULT_LAYOUT.openSummary;
}

/** Read the refine pane layout. Missing or invalid values use the defaults. */
export function readStoredRefineLayout(): RefineLayoutPreference {
  try {
    const stored = window.localStorage.getItem(REFINE_LAYOUT_STORAGE_KEY);
    if (stored == null) {
      return DEFAULT_LAYOUT;
    }
    const parsed: unknown = JSON.parse(stored);
    if (parsed == null || typeof parsed !== "object") {
      return DEFAULT_LAYOUT;
    }
    const record = parsed as Record<string, unknown>;
    return {
      openSummary: readOpenSummary(record),
      planOpen: isBoolean(record.planOpen) ? record.planOpen : DEFAULT_LAYOUT.planOpen,
    };
  } catch {
    return DEFAULT_LAYOUT;
  }
}

function persistRefineLayout(layout: RefineLayoutPreference): void {
  try {
    window.localStorage.setItem(REFINE_LAYOUT_STORAGE_KEY, JSON.stringify(layout));
  } catch {
    // Ignore storage failures (private mode, quota, etc.).
  }
}

/** Owns the open score summary and the Plan-pane open state for refine chat. */
export function useRefineLayoutPreference(): {
  openSummary: RefineSummaryKind | null;
  planOpen: boolean;
  toggleSummary: (kind: RefineSummaryKind) => void;
  togglePlan: () => void;
} {
  const [layout, setLayout] = useState<RefineLayoutPreference>(readStoredRefineLayout);

  const toggleSummary = useCallback((kind: RefineSummaryKind) => {
    setLayout((current) => {
      const next = { ...current, openSummary: current.openSummary === kind ? null : kind };
      persistRefineLayout(next);
      return next;
    });
  }, []);

  const togglePlan = useCallback(() => {
    setLayout((current) => {
      const next = { ...current, planOpen: !current.planOpen };
      persistRefineLayout(next);
      return next;
    });
  }, []);

  return {
    openSummary: layout.openSummary,
    planOpen: layout.planOpen,
    toggleSummary,
    togglePlan,
  };
}
