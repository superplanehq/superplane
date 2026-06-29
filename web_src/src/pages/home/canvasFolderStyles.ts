import type { CanvasFolderColor } from "@/hooks/useCanvasData";

export const CANVAS_FOLDER_SECTION_SHELL_CLASS = "w-full rounded-2xl p-4";

export const FOLDER_COLOR_OPTIONS: Record<
  CanvasFolderColor,
  { label: string; backgroundClass: string; swatchClass: string }
> = {
  blue: { label: "blue", backgroundClass: "bg-blue-500", swatchClass: "bg-blue-500" },
  green: { label: "green", backgroundClass: "bg-green-600", swatchClass: "bg-green-600" },
  purple: { label: "purple", backgroundClass: "bg-purple-500", swatchClass: "bg-purple-500" },
  yellow: { label: "yellow", backgroundClass: "bg-yellow-950", swatchClass: "bg-yellow-950" },
  slate: { label: "slate", backgroundClass: "bg-slate-700", swatchClass: "bg-slate-700" },
  orange: { label: "orange", backgroundClass: "bg-orange-500", swatchClass: "bg-orange-500" },
};
