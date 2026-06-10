import type { ReactNode } from "react";
import { cn } from "@/lib/utils";
import { useRunsSidebarWidth } from "./useRunsSidebarWidth";

export interface CanvasRunsSidebarProps {
  isOpen: boolean;
  children: ReactNode;
}

export function CanvasRunsSidebar({ isOpen, children }: CanvasRunsSidebarProps) {
  const { sidebarRef, width, isResizing, handleMouseDown } = useRunsSidebarWidth();

  if (!isOpen) {
    return null;
  }

  return (
    <aside
      ref={sidebarRef}
      data-testid="canvas-runs-sidebar"
      className="relative z-21 flex h-full shrink-0 flex-col border-r border-border bg-white"
      style={{ width }}
    >
      <div className="flex min-h-0 flex-1 flex-col overflow-hidden">{children}</div>
      <div
        onMouseDown={handleMouseDown}
        className="group absolute top-0 right-0 bottom-0 z-30 w-4 cursor-col-resize bg-transparent"
        style={{ marginRight: "-8px" }}
        data-testid="canvas-runs-sidebar-resize-handle"
      >
        <div
          aria-hidden
          className={cn(
            "pointer-events-none absolute top-0 bottom-0 left-1/2 w-px -translate-x-1/2 bg-transparent transition-colors group-hover:bg-slate-950/50",
            isResizing && "bg-slate-950/50",
          )}
        />
      </div>
    </aside>
  );
}
