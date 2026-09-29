import { useCallback, useState } from "react";

export const LINE_BOARD_COLUMN_COLOR_VIEWS = ["vivid", "dim", "borders", "off"] as const;

export type LineBoardColumnColorView = (typeof LINE_BOARD_COLUMN_COLOR_VIEWS)[number];

export const DEFAULT_LINE_BOARD_COLUMN_COLOR_VIEW: LineBoardColumnColorView = "vivid";

export const LINE_BOARD_COLUMN_COLOR_VIEW_STORAGE_KEY = "factories-column-color-view";

const LEGACY_FILL_VIEW = "fill";

export function isLineBoardColumnColorView(value: unknown): value is LineBoardColumnColorView {
  return value === "vivid" || value === "dim" || value === "borders" || value === "off";
}

function migrateStoredView(value: string | null): LineBoardColumnColorView {
  if (value === LEGACY_FILL_VIEW) {
    return "dim";
  }
  return isLineBoardColumnColorView(value) ? value : DEFAULT_LINE_BOARD_COLUMN_COLOR_VIEW;
}

/** Read the persisted column-color view. Missing or invalid values use vivid. */
export function readStoredLineBoardColumnColorView(): LineBoardColumnColorView {
  try {
    return migrateStoredView(window.localStorage.getItem(LINE_BOARD_COLUMN_COLOR_VIEW_STORAGE_KEY));
  } catch {
    return DEFAULT_LINE_BOARD_COLUMN_COLOR_VIEW;
  }
}

function persistLineBoardColumnColorView(view: LineBoardColumnColorView): void {
  try {
    if (view === DEFAULT_LINE_BOARD_COLUMN_COLOR_VIEW) {
      window.localStorage.removeItem(LINE_BOARD_COLUMN_COLOR_VIEW_STORAGE_KEY);
      return;
    }
    window.localStorage.setItem(LINE_BOARD_COLUMN_COLOR_VIEW_STORAGE_KEY, view);
  } catch {
    // Ignore storage failures (private mode, quota, etc.).
  }
}

/** Owns the line-board column-color layout. Vivid is the default and is not stored. */
export function useLineBoardColumnColorViewPreference(): {
  view: LineBoardColumnColorView;
  setView: (view: LineBoardColumnColorView) => void;
} {
  const [view, setViewState] = useState<LineBoardColumnColorView>(readStoredLineBoardColumnColorView);

  const setView = useCallback((next: LineBoardColumnColorView) => {
    setViewState(next);
    persistLineBoardColumnColorView(next);
  }, []);

  return { view, setView };
}
