import { AUX_SIDEBAR_MIN_WIDTH } from "@/stores/sidebarLayoutStore";
import { useAuxiliarySidebarWidth } from "@/stores/useAuxiliarySidebarWidth";

export const RUNS_SIDEBAR_WIDTH_STORAGE_KEY = "runs-sidebar-width";
export const RUNS_SIDEBAR_MIN_WIDTH = AUX_SIDEBAR_MIN_WIDTH;
export const RUNS_SIDEBAR_DEFAULT_WIDTH = 300;

export function useRunsSidebarWidth(isOpen: boolean) {
  return useAuxiliarySidebarWidth(isOpen, RUNS_SIDEBAR_WIDTH_STORAGE_KEY, RUNS_SIDEBAR_DEFAULT_WIDTH);
}
