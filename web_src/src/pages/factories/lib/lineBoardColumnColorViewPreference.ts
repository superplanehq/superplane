import { useCallback, useState } from "react";

export const LINE_BOARD_COLUMN_COLOR_VIEWS = ["fill", "off", "borders"] as const;

export type LineBoardColumnColorView = (typeof LINE_BOARD_COLUMN_COLOR_VIEWS)[number];

export const DEFAULT_LINE_BOARD_COLUMN_COLOR_VIEW: LineBoardColumnColorView = "fill";

export const LINE_BOARD_COLUMN_COLOR_VIEW_STORAGE_KEY = "factories-column-color-view";

export function isLineBoardColumnColorView(value: unknown): value is LineBoardColumnColorView {
  return value === "fill" || value === "off" || value === "borders";
}

/** Read the persisted column-color view. Missing or invalid values use fill. */
export function readStoredLineBoardColumnColorView(): LineBoardColumnColorView {
  try {
    const stored = window.localStorage.getItem(LINE_BOARD_COLUMN_COLOR_VIEW_STORAGE_KEY);
    return isLineBoardColumnColorView(stored) ? stored : DEFAULT_LINE_BOARD_COLUMN_COLOR_VIEW;
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

/** Owns the line-board column-color layout. Fill is the default and is not stored. */
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
