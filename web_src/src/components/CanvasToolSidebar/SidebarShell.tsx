import type { ReactNode } from "react";
import { cn } from "@/lib/utils";
import { useSidebarWidth } from "./useSidebarWidth";

export function SidebarShell({ children }: { children: ReactNode }) {
  const { sidebarRef, width, isResizing, handleMouseDown } = useSidebarWidth();

  return (
    <aside
      ref={sidebarRef}
      data-testid="canvas-tool-sidebar"
      className="relative z-21 flex h-full shrink-0 flex-col border-r border-border bg-white"
      style={{ width }}
    >
      <div className="flex min-h-0 flex-1 flex-col overflow-hidden">{children}</div>
      <div
        onMouseDown={handleMouseDown}
        className={cn(
          "absolute top-0 right-0 bottom-0 z-10 w-1 translate-x-1/2 cursor-col-resize bg-transparent transition-colors duration-150 ease-out delay-0",
          "hover:delay-300 hover:bg-slate-950/10",
          isResizing && "bg-slate-950/10 delay-0",
        )}
        aria-hidden
        data-testid="canvas-tool-sidebar-resize-handle"
      />
    </aside>
  );
}
