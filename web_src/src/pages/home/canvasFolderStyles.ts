import type { CanvasFolderColor } from "@/hooks/useCanvasData";

export const FOLDER_COLOR_OPTIONS: Record<
  CanvasFolderColor,
  { label: string; backgroundClass: string; swatchClass: string }
> = {
  color_1: { label: "blue", backgroundClass: "bg-blue-500", swatchClass: "bg-blue-500" },
  color_2: { label: "green", backgroundClass: "bg-green-600", swatchClass: "bg-green-600" },
  color_3: { label: "violet", backgroundClass: "bg-violet-500", swatchClass: "bg-violet-500" },
  color_4: { label: "yellow", backgroundClass: "bg-yellow-950", swatchClass: "bg-yellow-950" },
  color_5: { label: "slate", backgroundClass: "bg-slate-700", swatchClass: "bg-slate-700" },
  color_6: { label: "orange", backgroundClass: "bg-orange-500", swatchClass: "bg-orange-500" },
};
