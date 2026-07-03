import type { RunDisplayMode } from "./RunPanel";

export const RUN_PANEL_SIZE_STORAGE_KEY = "run-panel-size";

const VALID_MODES: RunDisplayMode[] = ["full", "split", "min"];
const DEFAULT_MODE: RunDisplayMode = "split";

export function loadRunPanelSize(): RunDisplayMode {
  if (typeof window === "undefined") return DEFAULT_MODE;
  try {
    const raw = window.localStorage.getItem(RUN_PANEL_SIZE_STORAGE_KEY);
    if (raw && VALID_MODES.includes(raw as RunDisplayMode)) return raw as RunDisplayMode;
  } catch {
    // Size preference is optional.
  }
  return DEFAULT_MODE;
}

export function saveRunPanelSize(mode: RunDisplayMode): void {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.setItem(RUN_PANEL_SIZE_STORAGE_KEY, mode);
  } catch {
    // Size preference is optional.
  }
}
