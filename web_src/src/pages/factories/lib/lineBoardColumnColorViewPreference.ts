import { useCallback, useState } from "react";
import { useTheme } from "@/contexts/useTheme";

export const LINE_BOARD_COLUMN_COLOR_VIEWS = ["vivid", "dim", "borders", "off"] as const;

export type LineBoardColumnColorView = (typeof LINE_BOARD_COLUMN_COLOR_VIEWS)[number];

export const DEFAULT_LINE_BOARD_COLUMN_COLOR_VIEW: LineBoardColumnColorView = "vivid";

export const LINE_BOARD_COLUMN_COLOR_VIEW_STORAGE_KEY = "factories-column-color-view";

const LEGACY_FILL_VIEW = "fill";

export function isLineBoardColumnColorView(value: unknown): value is LineBoardColumnColorView {
  return value === "vivid" || value === "dim" || value === "borders" || value === "off";
}

function migrateStoredView(value: string | null): LineBoardColumnColorView | null {
  if (value === LEGACY_FILL_VIEW) {
    return "dim";
  }
  return isLineBoardColumnColorView(value) ? value : null;
}

/** Null means the board follows the active theme. */
export function readStoredLineBoardColumnColorView(): LineBoardColumnColorView | null {
  try {
    return migrateStoredView(window.localStorage.getItem(LINE_BOARD_COLUMN_COLOR_VIEW_STORAGE_KEY));
  } catch {
    return null;
  }
}

function persistLineBoardColumnColorView(view: LineBoardColumnColorView): void {
  try {
    window.localStorage.setItem(LINE_BOARD_COLUMN_COLOR_VIEW_STORAGE_KEY, view);
  } catch {
    // Ignore storage failures (private mode, quota, etc.).
  }
}

/** Follow the theme until the person explicitly selects a board view. */
export function useLineBoardColumnColorViewPreference(): {
  view: LineBoardColumnColorView;
  setView: (view: LineBoardColumnColorView) => void;
} {
  const { resolvedTheme } = useTheme();
  const [selectedView, setViewState] = useState(readStoredLineBoardColumnColorView);
  const view = selectedView ?? (resolvedTheme === "dark" ? "dim" : "vivid");

  const setView = useCallback((next: LineBoardColumnColorView) => {
    setViewState(next);
    persistLineBoardColumnColorView(next);
  }, []);

  return { view, setView };
}
