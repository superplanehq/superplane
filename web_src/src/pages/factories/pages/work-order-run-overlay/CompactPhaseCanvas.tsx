import { Handle, Position, ReactFlow, Background, type Node, type NodeProps, type Edge } from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { useCallback, useMemo, type MouseEvent } from "react";

import githubIcon from "@/assets/icons/integrations/github.svg";
import slackIcon from "@/assets/icons/integrations/slack.svg";
import superplaneIcon from "@/assets/superplane.svg";
import { factoryCanvasBackground, factoryEdgePalette } from "@/lib/factoryCanvasChrome";
import { FactoryNodeCardShell } from "@/ui/factoryNodeChrome/FactoryNodeCardShell";
import type { FactoryNodeStatus } from "@/ui/factoryNodeChrome/types";

import type { RunOverlayProvider, RunOverlayStep, RunOverlayStepStatus } from "./workOrderRunOverlayMocks";

function providerIcon(provider: RunOverlayProvider): string {
  if (provider === "github") return githubIcon;
  if (provider === "slack") return slackIcon;
  return superplaneIcon;
}

function toFactoryStatus(status: RunOverlayStepStatus): FactoryNodeStatus {
  return status;
}

type OverlayNodeData = RunOverlayStep & { isLast: boolean; isFirst: boolean } & Record<string, unknown>;

function OverlayStepNode({ data, selected }: NodeProps<Node<OverlayNodeData>>) {
  return (
    <div className="relative">
      {data.isFirst ? null : (
        <Handle type="target" position={Position.Top} className="!size-2 !border-border !bg-card" />
      )}
      <FactoryNodeCardShell
        title={data.componentName}
        subtitle={data.detail ?? null}
        iconSrc={providerIcon(data.provider)}
        status={toFactoryStatus(data.status)}
        metrics={data.duration ?? null}
        selected={selected}
      />
      {data.isLast ? null : (
        <Handle id="out" type="source" position={Position.Bottom} className="!size-2 !border-border !bg-card" />
      )}
    </div>
  );
}

const nodeTypes = { overlayStep: OverlayStepNode };

function graphFromSteps(steps: RunOverlayStep[]): { nodes: Node<OverlayNodeData>[]; edges: Edge[] } {
  const nodes: Node<OverlayNodeData>[] = steps.map((step, index) => ({
    id: step.id,
    type: "overlayStep",
    position: { x: 40, y: index * 150 },
    data: { ...step, isFirst: index === 0, isLast: index === steps.length - 1 },
    draggable: false,
  }));
  const edges: Edge[] = steps.slice(0, -1).map((step, index) => ({
    id: `e-${step.id}-${steps[index + 1].id}`,
    source: step.id,
    sourceHandle: "out",
    target: steps[index + 1].id,
    type: "smoothstep",
  }));
  return { nodes, edges };
}

/**
 * Minimal live-canvas slice of the current phase. Uses factory node chrome
 * so the overlay can merge the run graph without opening a second page.
 */
export function CompactPhaseCanvas({
  steps,
  selectedId,
  onSelect,
}: {
  steps: RunOverlayStep[];
  selectedId: string | null;
  onSelect: (id: string | null) => void;
}) {
  const { nodes, edges } = useMemo(() => graphFromSteps(steps), [steps]);
  const background = factoryCanvasBackground(false);
  const palette = factoryEdgePalette(false);

  const displayNodes = useMemo(
    () => nodes.map((node) => ({ ...node, selected: node.id === selectedId })),
    [nodes, selectedId],
  );

  const onNodeClick = useCallback(
    (_event: MouseEvent, node: Node) => {
      onSelect(node.id);
    },
    [onSelect],
  );

  return (
    <div className="h-full min-h-[18rem] w-full" data-testid="run-overlay-compact-canvas">
      <ReactFlow
        nodes={displayNodes}
        edges={edges.map((edge) => ({ ...edge, style: palette.default }))}
        nodeTypes={nodeTypes}
        fitView
        fitViewOptions={{ padding: 0.24 }}
        nodesDraggable={false}
        nodesConnectable={false}
        panOnScroll
        zoomOnScroll
        onNodeClick={onNodeClick}
        onPaneClick={() => onSelect(null)}
        proOptions={{ hideAttribution: true }}
        colorMode="light"
        defaultEdgeOptions={{ type: "smoothstep", style: palette.default }}
      >
        <Background gap={background.gap} size={background.size} color={background.color} bgColor={background.bgColor} />
      </ReactFlow>
    </div>
  );
}
