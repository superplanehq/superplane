import { Loader2 } from "lucide-react";
import { appDarkModeClasses } from "@/lib/appDarkModeClasses";
import { cn } from "@/lib/utils";
import { RunInspectorChrome } from "./RunInspectorChrome";
import { ResizeHandle, useResizableInspectorWidth } from "./RunInspectorResize";

export function RunInspectorLoadingPanel({ onClose }: { onClose: () => void }) {
  const inspectorWidth = useResizableInspectorWidth();

  return (
    <aside
      className={cn(
        "relative z-20 flex h-full shrink-0 flex-col border-l border-slate-950/10 bg-white text-gray-900 shadow-xl dark:border-gray-800 dark:bg-gray-950 dark:text-gray-100",
        appDarkModeClasses,
      )}
      style={{ width: inspectorWidth.width }}
      aria-label="Run inspector"
    >
      <ResizeHandle onPointerDown={inspectorWidth.startResize} isResizing={inspectorWidth.isResizing} />
      <RunInspectorChrome onClose={onClose} />
      <div className="flex min-h-0 flex-1 items-center justify-center">
        <Loader2 className="h-5 w-5 animate-spin text-slate-500 dark:text-gray-400" />
      </div>
    </aside>
  );
}
