export type CanvasLayoutMode = "manual" | "factory-auto";

export function isFactoryAutoLayout(mode: CanvasLayoutMode | undefined): boolean {
  return mode === "factory-auto";
}
