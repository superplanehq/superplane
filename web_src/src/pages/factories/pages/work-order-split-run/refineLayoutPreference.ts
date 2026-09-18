import { useCallback, useState } from "react";

export const REFINE_LAYOUT_STORAGE_KEY = "sp:refine:layout";

export type RefineLayoutPreference = {
  planOpen: boolean;
};

const DEFAULT_LAYOUT: RefineLayoutPreference = {
  planOpen: false,
};

function isBoolean(value: unknown): value is boolean {
  return value === true || value === false;
}

/**
 * Read the refine pane layout. Missing or invalid values use the defaults.
 * Older layouts also stored a score summary choice; those keys are ignored.
 */
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

/** Owns the Plan-pane open state for refine chat. */
export function useRefineLayoutPreference(): {
  planOpen: boolean;
  togglePlan: () => void;
} {
  const [layout, setLayout] = useState<RefineLayoutPreference>(readStoredRefineLayout);

  const togglePlan = useCallback(() => {
    setLayout((current) => {
      const next = { ...current, planOpen: !current.planOpen };
      persistRefineLayout(next);
      return next;
    });
  }, []);

  return {
    planOpen: layout.planOpen,
    togglePlan,
  };
}
