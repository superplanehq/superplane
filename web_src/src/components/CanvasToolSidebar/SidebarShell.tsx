import type { ReactNode } from "react";
import { cn } from "@/lib/utils";
import { appDarkModeClasses } from "@/lib/appDarkModeClasses";
import { useSidebarWidth } from "./useSidebarWidth";

export function SidebarShell({ children }: { children: ReactNode }) {
  const { sidebarRef, width, isResizing, handleMouseDown } = useSidebarWidth();

  return (
    <aside
      ref={sidebarRef}
      data-testid="canvas-tool-sidebar"
      className={cn(
        "relative z-21 flex h-full shrink-0 flex-col border-r bg-white dark:bg-gray-900",
        appDarkModeClasses.sidebarEdge,
      )}
      style={{ width }}
    >
      <div className="flex min-h-0 flex-1 flex-col overflow-hidden">{children}</div>
      <div
        onMouseDown={handleMouseDown}
        className="group absolute top-0 right-0 bottom-0 z-30 w-4 cursor-col-resize bg-transparent"
        style={{ marginRight: "-8px" }}
        data-testid="canvas-tool-sidebar-resize-handle"
      >
        <div
          aria-hidden
          className={cn(
            "pointer-events-none absolute top-0 bottom-0 left-1/2 w-px -translate-x-1/2 bg-transparent transition-colors group-hover:bg-slate-950/50 dark:group-hover:bg-gray-600/50",
            isResizing && "bg-slate-950/50 dark:bg-gray-600/50",
          )}
        />
      </div>
    </aside>
  );
}
