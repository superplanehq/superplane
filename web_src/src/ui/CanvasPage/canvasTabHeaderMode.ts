import type { HeaderMode } from "./Header";

export function isCanvasTabHeaderMode(mode: HeaderMode | undefined): boolean {
  return mode === "default" || mode === "version-live" || mode === "version-edit";
}
