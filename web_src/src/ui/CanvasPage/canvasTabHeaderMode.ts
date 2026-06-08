import type { HeaderMode } from "./Header";

export function isCanvasTabHeaderMode(mode: HeaderMode | undefined): boolean {
  return mode === "default" || mode === "version-live" || mode === "version-edit";
}

/** Canvas and Console still surface node settings; Memory, Files, and Runs do not. */
export function isComponentSidebarVisibleMode(mode: HeaderMode | undefined): boolean {
  return isCanvasTabHeaderMode(mode) || mode === "console";
}
