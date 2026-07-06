import type { CanvasCardData } from "./types";

export function applyCanvasAppPreferences(canvases: CanvasCardData[]): CanvasCardData[] {
  return [...canvases].sort(compareCanvasPreferenceOrder);
}

function compareCanvasPreferenceOrder(left: CanvasCardData, right: CanvasCardData): number {
  const rankDiff = canvasPreferenceRank(left) - canvasPreferenceRank(right);
  if (rankDiff !== 0) {
    return rankDiff;
  }

  const preferenceTimeDiff = preferenceTime(right) - preferenceTime(left);
  if (preferenceTimeDiff !== 0) {
    return preferenceTimeDiff;
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

function preferenceTime(canvas: CanvasCardData): number {
  if (canvas.isPinned) {
    return timestampValue(canvas.pinnedAt);
  }

  if (canvas.isStarred) {
    return timestampValue(canvas.starredAt);
  }

  return 0;
}

function timestampValue(value?: string): number {
  if (!value) {
    return 0;
  }

  const parsed = Date.parse(value);
  return Number.isNaN(parsed) ? 0 : parsed;
}
