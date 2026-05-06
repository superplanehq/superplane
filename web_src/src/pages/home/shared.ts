import type { CanvasGroupColor } from "../../hooks/useCanvasData";
import { CANVAS_GROUP_COLORS } from "../../hooks/useCanvasData";
import type { SuperplaneComponentsEdge, SuperplaneComponentsNode } from "@/api-client";

export interface CanvasCardData {
  id: string;
  name: string;
  description?: string;
  createdAt: string;
  type: "canvases";
  canvasGroupId?: string;
  createdBy?: { id?: string; name?: string };
  nodes?: SuperplaneComponentsNode[];
  edges?: SuperplaneComponentsEdge[];
}

export interface CanvasGroupData {
  id: string;
  title: string;
  backgroundColor: CanvasGroupColor;
}

export const GROUP_BACKGROUND_CLASSES: Record<CanvasGroupColor, string> = {
  "blue-800": "bg-blue-800",
  "green-800": "bg-green-800",
  "slate-700": "bg-slate-700",
  "violet-800": "bg-violet-800",
  "yellow-800": "bg-yellow-800",
};

export const GROUP_SWATCH_CLASSES: Record<CanvasGroupColor, string> = {
  "blue-800": "bg-blue-800",
  "green-800": "bg-green-800",
  "slate-700": "bg-slate-700",
  "violet-800": "bg-violet-800",
  "yellow-800": "bg-yellow-800",
};

export const compareByName = <T extends { name: string }>(left: T, right: T) => left.name.localeCompare(right.name);

export function asCanvasGroupColor(value?: string): CanvasGroupColor {
  return CANVAS_GROUP_COLORS.includes(value as CanvasGroupColor) ? (value as CanvasGroupColor) : "blue-800";
}

// Strip the trailing tailwind weight suffix (e.g. "-800" or "-700") so the
// human-readable color name works for every variant we support, including
// "slate-700".
export const colorLabel = (color: CanvasGroupColor) => color.replace(/-\d+$/, "");
