import { SuperplaneConnectionGroup, SuperplaneEventSource, SuperplaneStageEvent } from "@/api-client/types.gen";
import { AllNodeType, EdgeType } from "../types/flow";
import { EventSourceWithEvents, Stage } from "../store/types";
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
          const timeA = new Date(a.receivedAt || 0).getTime();
          const timeB = new Date(b.receivedAt || 0).getTime();
          return timeB - timeA;
        }).slice(0, 3)
      : [];
      
    return ({
      id: es.metadata?.id || '',
      type: 'event_source',
      data: {
        id: es.metadata?.id || '',
        name: es.metadata?.name,
        description: es.metadata?.description,
        eventFilters: es.spec?.events,
        events: lastEvents,
        integration: es.spec?.integration,
        resource: es.spec?.resource,
        schedule: es.spec?.schedule,
      },
      position: nodePositions[es.metadata?.id || ''] || { x: 0, y: idx * 320 },
    }) as unknown as AllNodeType;
  });
};

export const transformStagesToNodes = (
  stages: Stage[],
  nodePositions: NodePositions,
  approveStageEvent: (eventId: string, stageId: string) => void
): AllNodeType[] => {
  return stages.map((st, idx) => ({
    id: st.metadata?.id || '',
    type: 'stage',
    data: {
      name: st.metadata?.name || '',
      labels: [],
      status: "",
      description: st.metadata?.description || '',
      icon: "storage",
      queues: st.queue || [],
      connections: st.spec?.connections || [],
      conditions: st.spec?.conditions || [],
      outputs: st.spec?.outputs || [],
      inputs: st.spec?.inputs || [],
      inputMappings: st.spec?.inputMappings || [],
      secrets: st.spec?.secrets || [],
      executor: st.spec?.executor,
      dryRun: st.spec?.dryRun || false,
      approveStageEvent: (event: SuperplaneStageEvent) => {
        approveStageEvent(event.id!, st.metadata?.id || '');
      },
      isDraft: st.isDraft || false
    },
    position: nodePositions[st.metadata?.id || ''] || {
      x: 600 * ((st.spec?.connections?.length || 1)),
      y: (idx - 1) * 400
    },
  } as unknown as AllNodeType));
};

export const transformConnectionGroupsToNodes = (
  connectionGroups: SuperplaneConnectionGroup[],
  nodePositions: NodePositions
): AllNodeType[] => {
  return connectionGroups.map((g, idx) => ({
    id: g.metadata?.id || '',
    type: 'connection_group',
    data: {
      id: g.metadata?.id || '',
      name: g.metadata?.name || '',
      description: g.metadata?.description || '',
      connections: g.spec?.connections || [],
      groupBy: g.spec?.groupBy || [],
    },
    position: nodePositions[g.metadata?.id || ''] || {
      x: 600 * ((g.spec?.connections?.length || 1)),
      y: (idx - 1) * 400
    },
    width: DEFAULT_WIDTH,
    height: DEFAULT_HEIGHT,
  } as unknown as AllNodeType));
};

export const transformToEdges = (
  stages: Stage[],
  connectionGroups: SuperplaneConnectionGroup[],
  eventSources: SuperplaneEventSource[]
): EdgeType[] => {
  const allEdges: EdgeType[] = [];
  const edgeIdSet = new Set<string>();

  stages.forEach((st) => {
    (st.spec?.connections || []).forEach((conn) => {
      const edgeId = `e-${conn.name}-${st.metadata?.id}`;
      
      if (edgeIdSet.has(edgeId)) {
        return;
      }

      const sourceObj =
        eventSources.find((es) => es.metadata?.name === conn.name) ||
        stages.find((s) => s.metadata?.name === conn.name) ||
        connectionGroups.find((g) => g.metadata?.name === conn.name);

      const sourceId = sourceObj?.metadata?.id ?? conn.name;
      const strokeColor = '#707070';
      
      const edge: EdgeType = {
        id: edgeId,
        source: sourceId || '',
        target: st.metadata?.id || '',
        type: ConnectionLineType.Bezier,
        animated: false,
        style: { stroke: strokeColor, strokeWidth: 2 },
        markerEnd: { type: MarkerType.ArrowClosed, color: strokeColor, strokeWidth: 2 }
      };

      edgeIdSet.add(edgeId);
      allEdges.push(edge);
    });
  });

  connectionGroups.forEach((g) => {
    (g.spec?.connections || []).forEach((conn) => {
      const edgeId = `e-${conn.name}-${g.metadata?.id}`;
      
      if (edgeIdSet.has(edgeId)) {
        return;
      }

      const sourceObj =
        eventSources.find((es) => es.metadata?.name === conn.name) ||
        stages.find((s) => s.metadata?.name === conn.name) ||
        connectionGroups.find((g) => g.metadata?.name === conn.name);

      const sourceId = sourceObj?.metadata?.id ?? conn.name;
      const strokeColor = '#707070';
      
      const edge: EdgeType = {
        id: edgeId,
        source: sourceId || '',
        target: g.metadata?.id || '',
        type: ConnectionLineType.Bezier,
        animated: false,
        style: { stroke: strokeColor, strokeWidth: 2 },
        markerEnd: { type: MarkerType.ArrowClosed, color: strokeColor, strokeWidth: 2 }
      };

      edgeIdSet.add(edgeId);
      allEdges.push(edge);
    });
  });

  return allEdges;
};

const filterEdgesByExistingNodes = (edges: Edge[], nodeIds: Set<string>): Edge[] => {
  return edges.filter(edge => nodeIds.has(edge.source) && nodeIds.has(edge.target));
};

export const autoLayoutNodes = async (
  nodes: AllNodeType[],
  edges: Edge[]
) => {
  let elkNodes: ElkNode[] = nodes.map((node) => ({
    id: node.id,
    width: DEFAULT_WIDTH,
    height: DEFAULT_HEIGHT,
  }));

  elkNodes = Array.from(new Map(elkNodes.map((node) => [node.id, node])).values());
  
  const nodeIdSet = new Set(elkNodes.map(node => node.id));
  const filteredEdges = filterEdgesByExistingNodes(edges, nodeIdSet);

  let elkEdges: ElkExtendedEdge[] = filteredEdges.map((edge) => ({
    id: edge.id,
    sources: [edge.source],
    targets: [edge.target],
  }));

  elkEdges = Array.from(new Map(elkEdges.map((edge) => [edge.id, edge])).values());

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