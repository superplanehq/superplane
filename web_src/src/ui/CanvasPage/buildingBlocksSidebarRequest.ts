import { useEffect } from "react";

export const BUILDING_BLOCKS_SIDEBAR_REQUEST_EVENT = "canvas:building-blocks-sidebar-request";
export const BUILDING_BLOCKS_SIDEBAR_CHANGED_EVENT = "canvas:building-blocks-sidebar-changed";

type BuildingBlocksSidebarDetail = {
  canvasId: string;
  open: boolean;
};

const lastBuildingBlocksRequestByCanvas = new Map<string, boolean>();

export function requestBuildingBlocksSidebar(canvasId: string, open: boolean): void {
  if (!canvasId) return;
  lastBuildingBlocksRequestByCanvas.set(canvasId, open);
  dispatchBuildingBlocksSidebarEvent(BUILDING_BLOCKS_SIDEBAR_REQUEST_EVENT, canvasId, open);
}

export function publishBuildingBlocksSidebarChanged(canvasId: string, open: boolean): void {
  dispatchBuildingBlocksSidebarEvent(BUILDING_BLOCKS_SIDEBAR_CHANGED_EVENT, canvasId, open);
}

export function subscribeBuildingBlocksSidebarRequest(onChange: (canvasId: string, open: boolean) => void): () => void {
  return subscribeBuildingBlocksSidebarEvent(BUILDING_BLOCKS_SIDEBAR_REQUEST_EVENT, onChange);
}

export function subscribeBuildingBlocksSidebarChanged(onChange: (canvasId: string, open: boolean) => void): () => void {
  return subscribeBuildingBlocksSidebarEvent(BUILDING_BLOCKS_SIDEBAR_CHANGED_EVENT, onChange);
}

export function useBuildingBlocksSidebarRequest(canvasId: string | undefined, onToggle: (open: boolean) => void): void {
  useEffect(() => {
    if (!canvasId) return;
    const pending = lastBuildingBlocksRequestByCanvas.get(canvasId);
    if (pending !== undefined) {
      onToggle(pending);
    }
    return subscribeBuildingBlocksSidebarRequest((requestedCanvasId, open) => {
      if (requestedCanvasId !== canvasId) return;
      onToggle(open);
    });
  }, [canvasId, onToggle]);
}

function dispatchBuildingBlocksSidebarEvent(name: string, canvasId: string, open: boolean): void {
  if (!canvasId || typeof window === "undefined") return;
  window.dispatchEvent(new CustomEvent<BuildingBlocksSidebarDetail>(name, { detail: { canvasId, open } }));
}

function subscribeBuildingBlocksSidebarEvent(
  name: string,
  onChange: (canvasId: string, open: boolean) => void,
): () => void {
  if (typeof window === "undefined") return () => undefined;

  const listener = (event: Event) => {
    const detail = (event as CustomEvent<BuildingBlocksSidebarDetail>).detail;
    if (detail?.canvasId) onChange(detail.canvasId, detail.open);
  };

  window.addEventListener(name, listener);
  return () => window.removeEventListener(name, listener);
}
