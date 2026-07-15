import type { PointerEvent } from "react";
import { cn } from "@/lib/utils";

export function ResizeHandle({
  isResizing,
  onPointerDown,
}: {
  isResizing: boolean;
  onPointerDown: (event: PointerEvent<HTMLDivElement>) => void;
}) {
  return (
    <div
      role="separator"
      aria-orientation="vertical"
      aria-label="Resize run inspector"
      data-testid="run-inspector-resize-handle"
      onPointerDown={onPointerDown}
      className="group absolute top-0 bottom-0 left-0 z-30 w-4 cursor-col-resize bg-transparent"
      style={{ marginLeft: "-8px" }}
    >
      <div
        aria-hidden
        data-testid="run-inspector-resize-line"
        className={cn(
          "pointer-events-none absolute top-0 bottom-0 left-[calc(50%-1px)] w-px -translate-x-1/2 bg-transparent transition-colors group-hover:bg-slate-950/50 dark:group-hover:bg-gray-500/50",
          isResizing && "bg-slate-950/50 dark:bg-gray-500/50",
        )}
      />
    </div>
  );
}
