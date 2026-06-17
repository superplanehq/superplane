import { useMemo } from "react";
import type { CanvasesCanvasRun, SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import { useEventExecutions } from "@/hooks/useCanvasData";
import { ResizableBottomPane } from "@/ui/CanvasPage/ResizableBottomPane";
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
  const rootEventId = run.rootEvent?.id || null;
  const executionsQuery = useEventExecutions(canvasId, rootEventId);
  const executions = useMemo(() => executionsQuery.data?.executions || [], [executionsQuery.data?.executions]);

  return (
    <ResizableBottomPane
      height={height}
      defaultHeight={defaultHeight}
      minHeight={minHeight}
      maxHeight={maxHeight}
      onHeightChange={onHeightChange}
      testId="run-node-detail-pane"
      resizeHandleTestId="run-node-detail-pane-resize-handle"
    >
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
    </ResizableBottomPane>
  );
}
