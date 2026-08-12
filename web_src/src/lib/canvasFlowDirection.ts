export type CanvasFlowDirection = "horizontal" | "vertical";

export function isFactoryApp(factoryId?: string | null): boolean {
  return Boolean(factoryId);
}

export function resolveCanvasFlowDirection(factoryId?: string | null): CanvasFlowDirection {
  return isFactoryApp(factoryId) ? "vertical" : "horizontal";
}

export function isVerticalCanvasFlow(direction: CanvasFlowDirection | undefined): boolean {
  return direction === "vertical";
}
