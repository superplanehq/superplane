import { useMemo } from "react";
import {
  transformEventSourcesToNodes,
  transformStagesToNodes,
  transformToEdges,
  applyGridLayout
} from "../utils/flowTransformers";
import { AllNodeType, LayoutedFlowData } from "../types/flow";
import { useCanvasStore } from "../store/canvasStore";

export const useFlowData = (): LayoutedFlowData => {
  const { stages, event_sources, nodePositions, approveStageEvent } = useCanvasStore();

  return useMemo(() => {
    const eventSourceNodes = transformEventSourcesToNodes(event_sources, nodePositions);
    const stageNodes = transformStagesToNodes(stages, nodePositions, approveStageEvent);
    const rawNodes = [...eventSourceNodes, ...stageNodes];
    const flowEdges = transformToEdges(stages, event_sources);

    const needsAutoLayout = rawNodes.some(node => !nodePositions[node.id]);
    const layoutedNodes: AllNodeType[] = needsAutoLayout && rawNodes.length > 0 
      ? applyGridLayout(rawNodes, nodePositions)
      : rawNodes;

    return { layoutedNodes, flowEdges };
  }, [event_sources, stages, nodePositions, approveStageEvent]);
};