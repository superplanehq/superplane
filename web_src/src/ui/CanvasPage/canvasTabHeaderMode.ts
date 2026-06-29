import type { HeaderMode } from "./Header";

export function isCanvasTabHeaderMode(mode: HeaderMode | undefined): boolean {
  return !mode || mode === "default" || mode === "version-live";
}

/** Canvas and Console still surface node settings; Memory and Files do not. Run inspection hides it via isRunInspectionMode. */
export function isComponentSidebarVisibleMode(mode: HeaderMode | undefined): boolean {
  return isCanvasTabHeaderMode(mode) || mode === "console";
}
