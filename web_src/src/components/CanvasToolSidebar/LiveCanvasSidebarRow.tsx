import { cn } from "@/lib/utils";
import { Radio } from "lucide-react";
import { RUNS_SIDEBAR_ROW_CLASS } from "./runsSidebarRowLayout";

interface LiveCanvasSidebarRowProps {
  isSelected: boolean;
  onSelect: () => void;
}

export function LiveCanvasSidebarRow({ isSelected, onSelect }: LiveCanvasSidebarRowProps) {
  return (
    <button
      type="button"
      data-testid="runs-sidebar-live-canvas"
      aria-label="Live Canvas"
      aria-current={isSelected ? "true" : undefined}
      className={cn(
        RUNS_SIDEBAR_ROW_CLASS,
        "w-full text-left transition-colors",
        isSelected ? "bg-sky-100 dark:bg-indigo-950" : "hover:bg-gray-50 dark:hover:bg-gray-800",
      )}
      onClick={onSelect}
    >
      <Radio
        className={cn(
          "h-3.5 w-3.5 shrink-0",
          isSelected ? "text-sky-800 dark:text-indigo-300" : "text-gray-500 dark:text-gray-400",
        )}
        aria-hidden
      />
      <span
        className={cn(
          "min-w-0 flex-1 truncate text-[13px]",
          isSelected
            ? "font-semibold text-sky-900 dark:text-indigo-300"
            : "font-medium text-gray-800 dark:text-gray-100",
        )}
      >
        Live Canvas
      </span>
    </button>
  );
}
