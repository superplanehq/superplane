import { SuperplaneEventSource, SuperplaneStageEvent } from "@/api-client/types.gen";
import { DEFAULT_WIDTH, DEFAULT_HEIGHT, LAYOUT_SPACING } from "./constants";
import { AllNodeType, EdgeType } from "../types/flow";
import { EventSourceWithEvents, StageWithEventQueue } from "../store/types";
import { ConnectionLineType, MarkerType } from "@xyflow/react";


interface NodePositions {
  [nodeId: string]: { x: number; y: number };
}

export const transformEventSourcesToNodes = (
  eventSources: EventSourceWithEvents[],
  nodePositions: NodePositions
): AllNodeType[] => {
  return eventSources.map((es, idx) => {
    const lastEvent = es.events && es.events.length > 0 
      ? es.events.sort((a, b) => {
          const timeA = new Date(a.createdAt || 0).getTime();
          const timeB = new Date(b.createdAt || 0).getTime();
          return timeB - timeA;
        })[0]
      : null;
    
    const lastEventTimestamp = new Date(lastEvent?.createdAt || '').toLocaleString() || 'n/a';
    
    return ({
      id: es.metadata?.id || '',
      type: 'githubIntegration',
      data: {
        id: es.metadata?.name || '',
        name: es.metadata?.name,
        timestamp: lastEventTimestamp
      },
      position: nodePositions[es.metadata?.id || ''] || { x: 0, y: idx * 320 },
      draggable: true
    }) as unknown as AllNodeType;
  });
};

export const transformStagesToNodes = (
  stages: StageWithEventQueue[],
  nodePositions: NodePositions,
  approveStageEvent: (eventId: string, stageId: string) => void
): AllNodeType[] => {
  return stages.map((st, idx) => ({
    id: st.metadata?.id || '',
      type: 'deploymentCard',
      data: {
        label: st.metadata?.name || '',
          labels: [],
          status: "",
          icon: "storage",
          queues: st.queue || [],
        connections: st.spec?.connections || [],
        conditions: st.spec?.conditions || [],
        outputs: st.spec?.outputs || [],
        inputs: st.spec?.inputs || [],
        executor: st.spec?.executor,
          approveStageEvent: (event: SuperplaneStageEvent) => {
            approveStageEvent(event.id!, st.metadata?.id || '');
          }
      },
    position: nodePositions[st.metadata?.id || ''] || {
      x: 600 * ((st.spec?.connections?.length || 1)),
          y: (idx - 1) * 400
      },
      draggable: true
  } as unknown as AllNodeType));
};

export const transformToEdges = (
  stages: StageWithEventQueue[],
  eventSources: SuperplaneEventSource[]
): EdgeType[] => {
  return stages.flatMap((st) =>
    (st.spec?.connections || []).map((conn) => {
      const isEvent = eventSources.some((es) => es.metadata?.name === conn.name);
      const sourceObj =
        eventSources.find((es) => es.metadata?.name === conn.name) ||
        stages.find((s) => s.metadata?.name === conn.name);
      const sourceId = sourceObj?.metadata?.id ?? conn.name;
      const strokeColor = isEvent ? '#FF0000' : '#000000';
      return { 
        id: `e-${conn.name}-${st.metadata?.id}`, 
        source: sourceId, 
        target: st.metadata?.id || '', 
        type: ConnectionLineType.Bezier, 
        animated: true, 
        style: { stroke: strokeColor, strokeWidth: 4 },
        markerEnd: { type: MarkerType.Arrow, color: strokeColor, strokeWidth: 2 }
      } as EdgeType;
    })
  );
};

export const applyGridLayout = (
  nodes: AllNodeType[],
  nodePositions: NodePositions
): AllNodeType[] => {
  return nodes.map((node, index) => {
    if (nodePositions[node.id]) {
      return node;
    }
    
    const cols = Math.ceil(Math.sqrt(nodes.length));
    const row = Math.floor(index / cols);
    const col = index % cols;
    
    return {
      ...node,
      position: {
        x: col * (DEFAULT_WIDTH + LAYOUT_SPACING.GRID_OFFSET),
        y: row * (DEFAULT_HEIGHT + LAYOUT_SPACING.GRID_OFFSET)
      }
    };
  });
};