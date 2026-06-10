import { useAuxiliarySidebarWidth } from "@/stores/useAuxiliarySidebarWidth";

export const VERSIONS_SIDEBAR_WIDTH_STORAGE_KEY = "versions-sidebar-width";
export const VERSIONS_SIDEBAR_MIN_WIDTH = 300;
export const VERSIONS_SIDEBAR_DEFAULT_WIDTH = 380;

export function useVersionsSidebarWidth(isOpen: boolean) {
  return useAuxiliarySidebarWidth(isOpen, VERSIONS_SIDEBAR_WIDTH_STORAGE_KEY, VERSIONS_SIDEBAR_DEFAULT_WIDTH);
}
