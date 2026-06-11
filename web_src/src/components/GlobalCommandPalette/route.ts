import { PUBLIC_TOP_LEVEL_SEGMENTS } from "./constants";
import { isAppRouteId } from "@/lib/appPaths";
import type { CommandPage } from "./types";

export function getRouteContext(pathname: string): {
  organizationId: string | null;
  canvasId: string | null;
} {
  const segments = pathname.split("/").filter(Boolean);
  const firstSegment = segments[0] || "";
  const organizationId = PUBLIC_TOP_LEVEL_SEGMENTS.has(firstSegment) ? null : firstSegment;
  const canvasSegmentIndex = segments.findIndex((segment) => segment === "apps" || segment === "canvases");
  const parentSegment = canvasSegmentIndex >= 0 ? segments[canvasSegmentIndex] : null;
  const rawCanvasId = canvasSegmentIndex >= 0 ? segments[canvasSegmentIndex + 1] || null : null;
  const canvasId = parentSegment === "apps" ? (isAppRouteId(rawCanvasId) ? rawCanvasId : null) : rawCanvasId;
  return { organizationId, canvasId };
}

export function pageTitle(page: CommandPage): string {
  switch (page) {
    case "organization-settings":
      return "Search organization settings";
    case "canvas-settings":
      return "Search app settings";
    case "open-canvas":
      return "Search apps";
    case "admin":
      return "Search admin pages";
    default:
      return "What can we help with?";
  }
}

export function isEditableTarget(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) return false;
  if (target.isContentEditable) return true;
  const tagName = target.tagName.toLowerCase();
  return tagName === "input" || tagName === "textarea" || tagName === "select";
}
