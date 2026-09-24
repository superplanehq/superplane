import { useCallback, useState } from "react";

export const COLUMN_AUTOMATION_VIEW_STORAGE_KEY = "factories-column-automation-view";

export const COLUMN_AUTOMATION_VIEWS = ["names", "icons"] as const;

export type ColumnAutomationView = (typeof COLUMN_AUTOMATION_VIEWS)[number];

const DEFAULT_VIEW: ColumnAutomationView = "names";

export function isColumnAutomationView(value: unknown): value is ColumnAutomationView {
  return value === "names" || value === "icons";
}

/** Read the persisted board view. Missing or invalid values use names. */
export function readStoredColumnAutomationView(): ColumnAutomationView {
  try {
    const stored = window.localStorage.getItem(COLUMN_AUTOMATION_VIEW_STORAGE_KEY);
    return isColumnAutomationView(stored) ? stored : DEFAULT_VIEW;
  } catch {
    return DEFAULT_VIEW;
  }
}

function persistColumnAutomationView(view: ColumnAutomationView): void {
  try {
    if (view === DEFAULT_VIEW) {
      window.localStorage.removeItem(COLUMN_AUTOMATION_VIEW_STORAGE_KEY);
      return;
    }
    window.localStorage.setItem(COLUMN_AUTOMATION_VIEW_STORAGE_KEY, view);
  } catch {
    // Ignore storage failures (private mode, quota, etc.).
  }
}

/** Owns the line-board automation layout. Names is the default. Only icons is stored. */
export function useColumnAutomationViewPreference(): {
  view: ColumnAutomationView;
  setView: (view: ColumnAutomationView) => void;
} {
  const [view, setViewState] = useState<ColumnAutomationView>(readStoredColumnAutomationView);

  const setView = useCallback((next: ColumnAutomationView) => {
    setViewState(next);
    persistColumnAutomationView(next);
  }, []);

  return { view, setView };
}
