import { SuperplaneConnectionGroup, SuperplaneEventSource, SuperplaneStageEvent } from "@/api-client/types.gen";
import { AllNodeType, EdgeType } from "../types/flow";
import { EventSourceWithEvents, StageWithEventQueue } from "../store/types";
import { ConnectionLineType, Edge, MarkerType } from "@xyflow/react";
import { DEFAULT_HEIGHT, DEFAULT_WIDTH } from "./constants";
import { ElkExtendedEdge, ElkNode } from "elkjs";
import { elk } from "./layoutConfig";


interface NodePositions {
  [nodeId: string]: { x: number; y: number };
}

export const transformEventSourcesToNodes = (
  eventSources: EventSourceWithEvents[],
  nodePositions: NodePositions
): AllNodeType[] => {
  return eventSources.map((es, idx) => {
    const lastEvents = es.events
      ? es.events.sort((a, b) => {
          const timeA = new Date(a.createdAt || 0).getTime();
          const timeB = new Date(b.createdAt || 0).getTime();
          return timeB - timeA;
        }).slice(0, 3)
      : [];
    
    
    return ({
      id: es.metadata?.id || '',
      type: 'githubIntegration',
      data: {
        id: es.metadata?.id || '',
        name: es.metadata?.name,
        description: es.metadata?.description,
        events: lastEvents,
        integration: es.spec?.integration,
        resource: es.spec?.resource,
      },
      position: nodePositions[es.metadata?.id || ''] || { x: 0, y: idx * 320 },
      draggable: true,
 
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
      description: st.metadata?.description || '',
      icon: "storage",
      queues: st.queue || [],
      connections: st.spec?.connections || [],
      conditions: st.spec?.conditions || [],
      outputs: st.spec?.outputs || [],
      inputs: st.spec?.inputs || [],
      secrets: st.spec?.secrets || [],
      executor: st.spec?.executor,
      approveStageEvent: (event: SuperplaneStageEvent) => {
        approveStageEvent(event.id!, st.metadata?.id || '');
      },
      isDraft: st.isDraft || false
    },
    position: nodePositions[st.metadata?.id || ''] || {
      x: 600 * ((st.spec?.connections?.length || 1)),
      y: (idx - 1) * 400
    },
    draggable: true,

  } as unknown as AllNodeType));
};

export const transformConnectionGroupsToNodes = (
  connectionGroups: SuperplaneConnectionGroup[],
  nodePositions: NodePositions
): AllNodeType[] => {
  return connectionGroups.map((g, idx) => ({
    id: g.metadata?.id || '',
    type: 'connectionGroup',
    data: {
      id: g.metadata?.id || '',
      name: g.metadata?.name || '',
      connections: g.spec?.connections || [],
      groupBy: g.spec?.groupBy || [],
    },
    position: nodePositions[g.metadata?.id || ''] || {
      x: 600 * ((g.spec?.connections?.length || 1)),
      y: (idx - 1) * 400
    },
    draggable: true,
    width: DEFAULT_WIDTH,
    height: DEFAULT_HEIGHT,
  } as unknown as AllNodeType));
};

export const transformToEdges = (
  stages: StageWithEventQueue[],
  connectionGroups: SuperplaneConnectionGroup[],
  eventSources: SuperplaneEventSource[]
): EdgeType[] => {
  const stageEdges = stages.flatMap((st) =>
    (st.spec?.connections || []).map((conn) => {
      const sourceObj =
        eventSources.find((es) => es.metadata?.name === conn.name) ||
        stages.find((s) => s.metadata?.name === conn.name) ||
        connectionGroups.find((g) => g.metadata?.name === conn.name);

      const sourceId = sourceObj?.metadata?.id ?? conn.name;
      const strokeColor = '#000000';
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

  const connectionGroupEdges = connectionGroups.flatMap((g) =>
    (g.spec?.connections || []).map((conn) => {
      const sourceObj =
        eventSources.find((es) => es.metadata?.name === conn.name) ||
        stages.find((s) => s.metadata?.name === conn.name) ||
        connectionGroups.find((g) => g.metadata?.name === conn.name);

      const sourceId = sourceObj?.metadata?.id ?? conn.name;
      const strokeColor = '#000000';
      return {
        id: `e-${conn.name}-${g.metadata?.id}`,
        source: sourceId,
        target: g.metadata?.id || '',
        type: ConnectionLineType.Bezier,
        animated: true,
        style: { stroke: strokeColor, strokeWidth: 4 },
        markerEnd: { type: MarkerType.Arrow, color: strokeColor, strokeWidth: 2 }
      } as EdgeType;
    })
  );

  return [...stageEdges, ...connectionGroupEdges];
};

export const autoLayoutNodes = async (
  nodes: AllNodeType[],
  edges: Edge[]
) => {
  const elkNodes: ElkNode[] = nodes.map((node) => ({
    id: node.id,
    width: DEFAULT_WIDTH,
    height: DEFAULT_HEIGHT,
  }));

  const elkEdges: ElkExtendedEdge[] = edges.map((edge) => ({
    id: edge.id,
    sources: [edge.source],
    targets: [edge.target],
  }));

  try {
    const layoutedGraph = await elk.layout({
      id: "root",
      children: elkNodes,
      edges: elkEdges,
    });

    const newNodes = nodes.map((node) => {
      const elkNode = layoutedGraph.children?.find((n) => n.id === node.id);
      const nodeElement: HTMLDivElement | null = document.querySelector(`[data-id="${node.id}"]`);

      if (elkNode?.x !== undefined && elkNode?.y !== undefined) {
        const newPosition = {
          x: elkNode.x + Math.random() / 1000,
          y: elkNode.y - (nodeElement?.offsetHeight || 0) / 2,
        };

        return {
          ...node,
          position: newPosition,
        };
      }

      return node;
    });

    return newNodes;
  } catch (error) {
    console.error('ELK auto-layout failed:', error);
    return nodes;
  }
};