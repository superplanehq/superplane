import { useMemo } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { Background, ReactFlow, ReactFlowProvider, type Edge, type Node, type NodeProps } from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import "@/ui/CanvasPage/canvas-reset.css";
import type { CanvasesCanvasNodeExecution, CanvasesCanvasRun } from "@/api-client";
import { prepareComponentBaseNode, prepareTriggerNode } from "@/pages/app/lib/canvas-node-preparation";
import { Block, type CanvasBlockData } from "@/ui/CanvasPage/Block";
import { getRunExecutions, RUNS_STORY_CANVAS_ID, mockWorkflowNodes } from "./fixtures";

function BlockNode({ id, data, selected }: NodeProps) {
  return <Block nodeId={id} selected={selected} data={data as unknown as CanvasBlockData} canvasMode="live" />;
}

const nodeTypes = { default: BlockNode };

export function RunCanvas({
  run,
  selectedNodeId,
  onSelectNode,
}: {
  run: CanvasesCanvasRun;
  selectedNodeId: string | null;
  onSelectNode: (nodeId: string) => void;
}) {
  const queryClient = useQueryClient();

  const { nodes, edges } = useMemo(() => {
    const executions = getRunExecutions(run.id!);

    const nodeExecutionsMap: Record<string, CanvasesCanvasNodeExecution[]> = {};
    for (const execution of executions) {
      if (!execution.nodeId) {
        continue;
      }
      (nodeExecutionsMap[execution.nodeId] ??= []).push(execution);
    }

    const participantIds: string[] = [];
    const addParticipant = (nodeId?: string) => {
      if (nodeId && !participantIds.includes(nodeId)) {
        participantIds.push(nodeId);
      }
    };
    addParticipant(run.rootEvent?.nodeId);
    executions.forEach((execution) => addParticipant(execution.nodeId));

    const canvasNodes: Node[] = participantIds
      .map((nodeId) => mockWorkflowNodes.find((node) => node.id === nodeId))
      .filter((node): node is (typeof mockWorkflowNodes)[number] => Boolean(node))
      .map((node, index) => {
        const canvasNode =
          node.type === "TYPE_TRIGGER"
            ? prepareTriggerNode(node, [], {}, "live", { canvasId: RUNS_STORY_CANVAS_ID })
            : prepareComponentBaseNode({
                nodes: mockWorkflowNodes,
                node,
                components: [],
                nodeExecutionsMap,
                nodeQueueItemsMap: {},
                canvasId: RUNS_STORY_CANVAS_ID,
                queryClient,
                canvasMode: "live",
              });

        return {
          ...canvasNode,
          type: "default",
          position: { x: index * 440, y: (index % 2) * 60 },
          selected: canvasNode.id === selectedNodeId,
        };
      });

    const canvasEdges: Edge[] = canvasNodes.slice(1).map((node, index) => ({
      id: `edge-${canvasNodes[index].id}-${node.id}`,
      source: canvasNodes[index].id,
      target: node.id,
    }));

    return { nodes: canvasNodes, edges: canvasEdges };
  }, [run, selectedNodeId, queryClient]);

  return (
    <div className="relative min-h-0 flex-1">
      <ReactFlowProvider>
        <ReactFlow
          nodes={nodes}
          edges={edges}
          nodeTypes={nodeTypes}
          fitView
          minZoom={0.2}
          proOptions={{ hideAttribution: true }}
          onNodeClick={(_, node) => onSelectNode(node.id)}
          className="sp-canvas h-full w-full"
        >
          <Background gap={16} size={1.5} bgColor="#F8FAFC" color="#cbd5e1" />
        </ReactFlow>
      </ReactFlowProvider>
    </div>
  );
}
