import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";
import { appDarkModeClasses } from "@/lib/appDarkModeClasses";
import { cn } from "@/lib/utils";

export interface ResizableBottomPaneProps {
  children: ReactNode;
  height?: number;
  defaultHeight?: number;
  minHeight?: number;
  maxHeight?: number;
  onHeightChange?: (height: number) => void;
  testId?: string;
  resizeHandleTestId?: string;
}

export function ResizableBottomPane({
  children,
  height,
  defaultHeight = 320,
  minHeight = 240,
  maxHeight = 820,
  onHeightChange,
  testId = "resizable-bottom-pane",
  resizeHandleTestId = "resizable-bottom-pane-resize-handle",
}: ResizableBottomPaneProps) {
  const [internalHeight, setInternalHeight] = useState(defaultHeight);
  const [isResizing, setIsResizing] = useState(false);
  const dragStartRef = useRef<{ y: number; height: number } | null>(null);

  const paneHeight = height ?? internalHeight;
  const clampHeight = useCallback(
    (value: number) => {
      const overrideMaxHeight = Math.min(document.body.clientHeight - 100, maxHeight);
      return Math.max(minHeight, Math.min(overrideMaxHeight, value));
    },
    [minHeight, maxHeight],
  );

  const setPaneHeight = useCallback(
    (value: number) => {
      const nextHeight = clampHeight(value);
      if (height === undefined) {
        setInternalHeight(nextHeight);
      }
      onHeightChange?.(nextHeight);
    },
    [clampHeight, height, onHeightChange],
  );

  const handleResizeStart = useCallback(
    (event: React.MouseEvent<HTMLDivElement>) => {
      dragStartRef.current = { y: event.clientY, height: paneHeight };
      setIsResizing(true);
    },
    [paneHeight],
  );

  useEffect(() => {
    if (!isResizing) {
      return;
    }

    const handleMouseMove = (moveEvent: MouseEvent) => {
      if (!dragStartRef.current) return;
      const delta = dragStartRef.current.y - moveEvent.clientY;
      setPaneHeight(dragStartRef.current.height + delta);
    };

    const handleMouseUp = () => {
      dragStartRef.current = null;
      setIsResizing(false);
    };

    window.addEventListener("mousemove", handleMouseMove);
    window.addEventListener("mouseup", handleMouseUp);
    document.body.style.userSelect = "none";
    document.body.style.cursor = "ns-resize";

    return () => {
      window.removeEventListener("mousemove", handleMouseMove);
      window.removeEventListener("mouseup", handleMouseUp);
      document.body.style.userSelect = "";
      document.body.style.cursor = "";
    };
  }, [isResizing, setPaneHeight]);

  return (
    <aside
      className={cn(
        "relative z-20 flex min-h-0 shrink-0 flex-col border-t bg-white dark:bg-gray-900",
        appDarkModeClasses.sidebarEdge,
      )}
      data-testid={testId}
      style={{ height: paneHeight, minHeight, maxHeight }}
    >
      <div
        onMouseDown={handleResizeStart}
        className="group absolute left-0 right-0 top-0 z-50 h-4 cursor-row-resize bg-transparent"
        style={{ marginTop: "-8px" }}
        data-testid={resizeHandleTestId}
      >
        <div
          aria-hidden
          data-testid={`${resizeHandleTestId}-line`}
          className={cn(
            "pointer-events-none absolute left-0 right-0 top-[calc(50%-1px)] h-px -translate-y-1/2 bg-transparent transition-colors group-hover:bg-slate-950/50 dark:group-hover:bg-gray-600/50",
            isResizing && "bg-slate-950/50 dark:bg-gray-600/50",
          )}
        />
      </div>
      <div className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden">{children}</div>
    </aside>
  );
}
