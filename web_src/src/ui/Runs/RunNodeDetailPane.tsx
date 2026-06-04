import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { CanvasesCanvasRun, SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import { useEventExecutions } from "@/hooks/useCanvasData";
import { cn } from "@/lib/utils";
import { RunNodeDetailContent } from "./RunNodeDetailContent";

export interface RunNodeDetailPaneProps {
  canvasId: string;
  run: CanvasesCanvasRun;
  nodeId: string;
  workflowNodes?: ComponentsNode[];
  componentIconMap?: Record<string, string>;
  onClose: () => void;
  onNavigateNode?: (nodeId: string) => void;
  height?: number;
  defaultHeight?: number;
  minHeight?: number;
  maxHeight?: number;
  onHeightChange?: (height: number) => void;
}

export function RunNodeDetailPane({
  canvasId,
  run,
  nodeId,
  workflowNodes = [],
  componentIconMap = {},
  onClose,
  onNavigateNode,
  height,
  defaultHeight = 320,
  minHeight = 240,
  maxHeight = 820,
  onHeightChange,
}: RunNodeDetailPaneProps) {
  const [internalHeight, setInternalHeight] = useState(defaultHeight);
  const [isResizing, setIsResizing] = useState(false);
  const dragStartRef = useRef<{ y: number; height: number } | null>(null);

  const rootEventId = run.rootEvent?.id || null;
  const executionsQuery = useEventExecutions(canvasId, rootEventId);
  const executions = useMemo(() => executionsQuery.data?.executions || [], [executionsQuery.data?.executions]);

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
      document.body.style.userSelect = "none";
      document.body.style.cursor = "ns-resize";
      setIsResizing(true);

      const handleMouseMove = (moveEvent: MouseEvent) => {
        if (!dragStartRef.current) return;
        const delta = dragStartRef.current.y - moveEvent.clientY;
        setPaneHeight(dragStartRef.current.height + delta);
      };

      const handleMouseUp = () => {
        dragStartRef.current = null;
        document.body.style.userSelect = "";
        document.body.style.cursor = "";
        setIsResizing(false);
        window.removeEventListener("mousemove", handleMouseMove);
        window.removeEventListener("mouseup", handleMouseUp);
      };

      window.addEventListener("mousemove", handleMouseMove);
      window.addEventListener("mouseup", handleMouseUp);
    },
    [paneHeight, setPaneHeight],
  );

  useEffect(() => {
    return () => {
      document.body.style.userSelect = "";
      document.body.style.cursor = "";
    };
  }, []);

  return (
    <aside
      className="relative z-31 shrink-0 border-t border-border bg-white"
      data-testid="run-node-detail-pane"
      style={{ height: paneHeight, minHeight, maxHeight }}
    >
      <div
        onMouseDown={handleResizeStart}
        className="group absolute left-0 right-0 top-0 z-30 h-4 cursor-row-resize bg-transparent"
        style={{ marginTop: "-8px" }}
        data-testid="run-node-detail-pane-resize-handle"
      >
        <div
          aria-hidden
          className={cn(
            "pointer-events-none absolute left-0 right-0 top-1/2 h-px -translate-y-1/2 bg-transparent transition-colors group-hover:bg-slate-950/50",
            isResizing && "bg-slate-950/50",
          )}
        />
      </div>
      <RunNodeDetailContent
        run={run}
        nodeId={nodeId}
        workflowNodes={workflowNodes}
        componentIconMap={componentIconMap}
        executions={executions}
        isExecutionsLoading={executionsQuery.isLoading}
        onClose={onClose}
        onNavigateNode={onNavigateNode}
        testId="run-node-detail-modal"
      />
    </aside>
  );
}
