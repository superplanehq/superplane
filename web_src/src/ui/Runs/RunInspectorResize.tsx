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
      className={cn(
        "absolute left-0 top-0 z-30 h-full w-1 -translate-x-1/2 cursor-ew-resize transition-colors hover:bg-blue-300/60",
        isResizing && "bg-blue-400/70",
      )}
    />
  );
}
