import type { FindingLocation } from "./types";

export function formatFindingLocation(location: FindingLocation): string {
  if (location.startLine == null) return location.path;
  if (location.endLine == null || location.endLine === location.startLine) {
    return `${location.path}:${location.startLine}`;
  }
  return `${location.path}:${location.startLine}-${location.endLine}`;
}
