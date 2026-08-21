import { Handle, Position, ReactFlow, Background, type Edge, type Node, type NodeProps } from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { Maximize2, MoreHorizontal, Pencil } from "lucide-react";
import { useCallback, useMemo, type MouseEvent } from "react";

import type { ComponentsEdge, SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import { factoryCanvasBackground, factoryEdgePalette } from "@/lib/factoryCanvasChrome";
import { Link } from "@/components/Link/link";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import { FactoryNodeCardShell } from "@/ui/factoryNodeChrome/FactoryNodeCardShell";
import type { FactoryNodeStatus } from "@/ui/factoryNodeChrome/types";

import { componentPresentation, type SplitRunCanvasModel } from "./splitRunCanvases";

type LineNodeData = {
  title: string;
  subtitle: string;
  iconSlug: string;
  status: FactoryNodeStatus;
  metrics: string;
  nodeId: string;
  onSelect?: (id: string) => void;
} & Record<string, unknown>;

function LineCanvasNode({ data, selected }: NodeProps<Node<LineNodeData>>) {
  return (
    <div className="relative" onClick={() => data.onSelect?.(data.nodeId)}>
      <Handle type="target" position={Position.Top} className="!size-2 !border-border !bg-card" />
      <FactoryNodeCardShell
        title={data.title}
        subtitle={data.subtitle}
        iconSlug={data.iconSlug}
        status={data.status}
        metrics={data.metrics}
        selected={selected}
      />
      <Handle id="default" type="source" position={Position.Bottom} className="!size-2 !border-border !bg-card" />
      <Handle id="true" type="source" position={Position.Bottom} className="!size-2 !border-border !bg-card" />
      <Handle id="passed" type="source" position={Position.Bottom} className="!size-2 !border-border !bg-card" />
      <Handle id="false" type="source" position={Position.Right} className="!size-2 !border-border !bg-card" />
      <Handle id="failed" type="source" position={Position.Right} className="!size-2 !border-border !bg-card" />
      <Handle id="found" type="source" position={Position.Bottom} className="!size-2 !border-border !bg-card" />
    </div>
  );
}

const nodeTypes = { lineCanvas: LineCanvasNode };

function origin(nodes: ComponentsNode[]): { x: number; y: number } {
  const xs = nodes.map((node) => node.position?.x ?? 0);
  const ys = nodes.map((node) => node.position?.y ?? 0);
  return { x: Math.min(...xs, 0), y: Math.min(...ys, 0) };
}

function graphFromCanvas(
  canvas: SplitRunCanvasModel,
  selectedId: string | null,
  onSelect?: (id: string) => void,
): { nodes: Node<LineNodeData>[]; edges: Edge[] } {
  const zero = origin(canvas.nodes);
  const nodes: Node<LineNodeData>[] = canvas.nodes
    .filter((node): node is ComponentsNode & { id: string } => Boolean(node.id))
    .map((node) => {
      const presentation = componentPresentation(node.component);
      return {
        id: node.id,
        type: "lineCanvas",
        position: {
          x: (node.position?.x ?? 0) - zero.x,
          y: (node.position?.y ?? 0) - zero.y,
        },
        data: {
          title: presentation.title,
          subtitle: node.name ?? presentation.title,
          iconSlug: presentation.iconSlug,
          status: canvas.statuses[node.id] ?? "pending",
          metrics: canvas.metrics[node.id] ?? "—",
          nodeId: node.id,
          onSelect,
        },
        selected: node.id === selectedId,
        draggable: false,
      };
    });

  const edges: Edge[] = canvas.edges.map((edge, index) => toFlowEdge(edge, index));
  return { nodes, edges };
}

function toFlowEdge(edge: ComponentsEdge, index: number): Edge {
  const channel = edge.channel ?? "default";
  const sideChannel = channel === "false" || channel === "failed";
  return {
    id: edge.id ?? `e-${edge.sourceId}-${edge.targetId}-${index}`,
    source: edge.sourceId ?? "",
    target: edge.targetId ?? "",
    sourceHandle: channel,
    type: "smoothstep",
    label: channel === "default" ? undefined : channel,
    labelStyle: { fontSize: 10, fill: "#64748b" },
    labelBgStyle: { fill: "#ffffff", fillOpacity: 1 },
    labelBgPadding: [6, 2],
    labelBgBorderRadius: 6,
    style: sideChannel ? { stroke: "#cbd5e1" } : undefined,
  };
}

/**
 * Real line-app canvas in the run pane. Nodes keep factory card chrome.
 */
export function CompactLineCanvas({
  canvas,
  selectedId,
  onSelect,
  editHref,
  expandHref,
  showHeader = true,
}: {
  canvas: SplitRunCanvasModel;
  selectedId: string | null;
  onSelect: (id: string | null) => void;
  editHref?: string;
  expandHref?: string;
  showHeader?: boolean;
}) {
  const { nodes, edges } = useMemo(() => graphFromCanvas(canvas, selectedId, onSelect), [canvas, selectedId, onSelect]);
  const background = factoryCanvasBackground(false);
  const palette = factoryEdgePalette(false);

  const onNodeClick = useCallback(
    (_event: MouseEvent, node: Node) => {
      onSelect(node.id);
    },
    [onSelect],
  );

  return (
    <div className="flex h-full min-h-0 w-full flex-col">
      {showHeader ? (
        <div className="flex shrink-0 items-center justify-between gap-2 px-4 pt-3 pb-2">
          <p className="min-w-0 truncate text-[15px] font-semibold tracking-[-0.02em] text-foreground">
            {canvas.title}
          </p>
          <div className="flex shrink-0 items-center gap-0.5">
            {expandHref ? (
              <Link
                href={expandHref}
                aria-label="Open automation run"
                className="flex size-7 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
                data-testid="split-run-canvas-expand"
              >
                <Maximize2 className="size-3.5" aria-hidden />
              </Link>
            ) : null}
            <CanvasOverflowMenu title={canvas.title} editHref={editHref} />
          </div>
        </div>
      ) : null}
      <div className="min-h-0 flex-1" data-testid="run-overlay-compact-canvas">
        <ReactFlow
          nodes={nodes}
          edges={edges.map((edge) => ({ ...edge, style: { ...palette.default, ...edge.style } }))}
          nodeTypes={nodeTypes}
          fitView
          fitViewOptions={{ padding: 0.2 }}
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
          <Background
            gap={background.gap}
            size={background.size}
            color={background.color}
            bgColor={background.bgColor}
          />
        </ReactFlow>
      </div>
    </div>
  );
}

function CanvasOverflowMenu({ title, editHref }: { title: string; editHref?: string }) {
  const href = editHref || "#";
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          aria-label={`${title} menu`}
          className="flex size-7 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
          data-testid="split-run-canvas-menu"
        >
          <MoreHorizontal className="size-3.5" aria-hidden />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-40">
        <DropdownMenuItem asChild data-testid="split-run-canvas-edit">
          <a href={href}>
            <Pencil className="h-3.5 w-3.5" aria-hidden />
            Edit
          </a>
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
