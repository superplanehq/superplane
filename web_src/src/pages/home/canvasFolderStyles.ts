import type { CanvasFolderColor } from "@/hooks/useCanvasData";

export const CANVAS_FOLDER_SECTION_SHELL_CLASS = "w-full rounded-2xl p-4";

type FolderColorOption = {
  label: string;
  backgroundClass: string;
  swatchClass: string;
  swatchForegroundClass: string;
  foregroundClass: string;
  foregroundMutedClass: string;
  headerInteractiveClass: string;
  renameInputClass: string;
};

export const FOLDER_COLOR_OPTIONS: Record<CanvasFolderColor, FolderColorOption> = {
  blue: {
    label: "blue",
    backgroundClass: "bg-blue-500 dark:bg-blue-900/45",
    swatchClass: "bg-blue-500 dark:bg-blue-500/70",
    swatchForegroundClass: "text-white",
    foregroundClass: "text-white",
    foregroundMutedClass: "text-white/80",
    headerInteractiveClass: "hover:border-white/25 hover:bg-white/5",
    renameInputClass: "border-white/50 bg-white/5 text-white placeholder:text-white/60 focus-visible:border-white/60",
  },
  green: {
    label: "green",
    backgroundClass: "bg-green-600 dark:bg-green-900/45",
    swatchClass: "bg-green-600 dark:bg-green-600/70",
    swatchForegroundClass: "text-white",
    foregroundClass: "text-white",
    foregroundMutedClass: "text-white/80",
    headerInteractiveClass: "hover:border-white/25 hover:bg-white/5",
    renameInputClass: "border-white/50 bg-white/5 text-white placeholder:text-white/60 focus-visible:border-white/60",
  },
  purple: {
    label: "violet",
    backgroundClass: "bg-violet-500 dark:bg-violet-900/45",
    swatchClass: "bg-violet-500 dark:bg-violet-500/70",
    swatchForegroundClass: "text-white",
    foregroundClass: "text-white",
    foregroundMutedClass: "text-white/80",
    headerInteractiveClass: "hover:border-white/25 hover:bg-white/5",
    renameInputClass: "border-white/50 bg-white/5 text-white placeholder:text-white/60 focus-visible:border-white/60",
  },
  slate: {
    label: "slate",
    backgroundClass: "bg-slate-700 dark:bg-slate-600/45",
    swatchClass: "bg-slate-700 dark:bg-slate-700/70",
    swatchForegroundClass: "text-white",
    foregroundClass: "text-white",
    foregroundMutedClass: "text-white/80",
    headerInteractiveClass: "hover:border-white/25 hover:bg-white/5",
    renameInputClass: "border-white/50 bg-white/5 text-white placeholder:text-white/60 focus-visible:border-white/60",
  },
  orange: {
    label: "orange",
    backgroundClass: "bg-amber-400 dark:bg-orange-900/45",
    swatchClass: "bg-amber-400 dark:bg-amber-400/70",
    swatchForegroundClass: "text-amber-950",
    foregroundClass: "text-amber-950 dark:text-white",
    foregroundMutedClass: "text-amber-950/80 dark:text-white/80",
    headerInteractiveClass:
      "hover:border-amber-950/20 hover:bg-amber-950/5 dark:hover:border-white/25 dark:hover:bg-white/5",
    renameInputClass:
      "border-amber-950/40 bg-amber-950/5 text-amber-950 placeholder:text-amber-950/60 focus-visible:border-amber-950/60 dark:border-white/50 dark:bg-white/5 dark:text-white dark:placeholder:text-white/60 dark:focus-visible:border-white/60",
  },
};

export function folderColorStyles(backgroundColor: CanvasFolderColor) {
  return FOLDER_COLOR_OPTIONS[backgroundColor];
}
