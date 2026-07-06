import type { CanvasAppPreferences } from "@/lib/canvasAppPreferences";
import type { CanvasCardData } from "./types";

export function applyCanvasAppPreferences(
  canvases: CanvasCardData[],
  preferences: CanvasAppPreferences,
): CanvasCardData[] {
  const preferenceOrder = {
    pinned: buildPreferenceOrder(preferences.pinnedCanvasIds),
    starred: buildPreferenceOrder(preferences.starredCanvasIds),
  };

  return canvases
    .map((canvas) => ({
      ...canvas,
      isPinned: preferenceOrder.pinned.has(canvas.id),
      isStarred: preferenceOrder.starred.has(canvas.id),
    }))
    .sort((left, right) => compareCanvasPreferenceOrder(left, right, preferenceOrder));
}

function compareCanvasPreferenceOrder(
  left: CanvasCardData,
  right: CanvasCardData,
  preferenceOrder: {
    pinned: Map<string, number>;
    starred: Map<string, number>;
  },
): number {
  const rankDiff = canvasPreferenceRank(left) - canvasPreferenceRank(right);
  if (rankDiff !== 0) {
    return rankDiff;
  }

  const orderDiff = preferenceIndex(left, preferenceOrder) - preferenceIndex(right, preferenceOrder);
  if (orderDiff !== 0) {
    return orderDiff;
  }

  return left.name.localeCompare(right.name);
}

function canvasPreferenceRank(canvas: CanvasCardData): number {
  if (canvas.isPinned) {
    return 0;
  }

  if (canvas.isStarred) {
    return 1;
  }

  return 2;
}

function preferenceIndex(
  canvas: CanvasCardData,
  preferenceOrder: {
    pinned: Map<string, number>;
    starred: Map<string, number>;
  },
): number {
  if (canvas.isPinned) {
    return preferenceOrder.pinned.get(canvas.id) ?? Number.MAX_SAFE_INTEGER;
  }

  if (canvas.isStarred) {
    return preferenceOrder.starred.get(canvas.id) ?? Number.MAX_SAFE_INTEGER;
  }

  return Number.MAX_SAFE_INTEGER;
}

function buildPreferenceOrder(canvasIds: string[]): Map<string, number> {
  return new Map(canvasIds.map((canvasId, index) => [canvasId, index]));
}
