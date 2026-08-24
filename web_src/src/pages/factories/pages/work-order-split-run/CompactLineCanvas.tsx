import { Handle, Position, ReactFlow, Background, type Node, type NodeProps } from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { Maximize2, MoreHorizontal, Pencil } from "lucide-react";
import { useCallback, useMemo, type MouseEvent } from "react";

import { Button } from "@/components/ui/button";
import { buttonVariants } from "@/components/ui/buttonVariants";
import { useTheme } from "@/contexts/useTheme";
import { FACTORY_HANDLE_STYLE, factoryCanvasBackground, factoryEdgePalette } from "@/lib/factoryCanvasChrome";
import { FACTORY_SIDE_HANDLE_ID, FACTORY_SPINE_HANDLE_ID } from "@/lib/layout/factoryRunLeafLayout";
import { cn } from "@/lib/utils";
import { Link } from "@/components/Link/link";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import { FactoryNodeCardShell } from "@/ui/factoryNodeChrome/FactoryNodeCardShell";

import { compactLineCanvasGraph, type LineNodeData } from "./compactLineCanvasGraph";
import type { SplitRunCanvasModel } from "./splitRunCanvases";

function LineCanvasNode({ data }: NodeProps<Node<LineNodeData>>) {
  return (
    <div
      className={cn("relative rounded-2xl", data.isSelected && "z-10")}
      data-testid={`split-run-canvas-node-${data.nodeId}`}
      data-selected={data.isSelected ? "true" : undefined}
      aria-selected={data.isSelected}
      onClick={() => data.onSelect?.(data.nodeId)}
    >
      <div
        className={cn(
          "rounded-2xl",
          data.isSelected &&
            "ring-2 ring-[color:var(--status-running-dot)] ring-offset-2 ring-offset-[color:var(--status-running-bg)]",
        )}
      >
        <Handle type="target" position={Position.Top} style={FACTORY_HANDLE_STYLE} />
        <Handle type="target" position={Position.Left} style={FACTORY_HANDLE_STYLE} />
        <FactoryNodeCardShell
          title={data.title}
          subtitle={data.subtitle}
          iconSlug={data.iconSlug}
          status={data.status}
          metrics={data.metrics}
          selected={data.isSelected}
        />
        <Handle id={FACTORY_SPINE_HANDLE_ID} type="source" position={Position.Bottom} style={FACTORY_HANDLE_STYLE} />
        <Handle id={FACTORY_SIDE_HANDLE_ID} type="source" position={Position.Right} style={FACTORY_HANDLE_STYLE} />
      </div>
    </div>
  );
}

const nodeTypes = { lineCanvas: LineCanvasNode };

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
  headerEdit = "menu",
  editLabel = "Edit Automation",
  onEdit,
}: {
  canvas: SplitRunCanvasModel;
  selectedId: string | null;
  onSelect: (id: string | null) => void;
  editHref?: string;
  expandHref?: string;
  showHeader?: boolean;
  headerEdit?: "menu" | "button";
  editLabel?: string;
  onEdit?: () => void;
}) {
  const { resolvedTheme } = useTheme();
  const isDark = resolvedTheme === "dark";
  const { nodes, edges } = useMemo(
    () => compactLineCanvasGraph(canvas, selectedId, onSelect, isDark),
    [canvas, isDark, onSelect, selectedId],
  );
  const background = factoryCanvasBackground(isDark);
  const palette = factoryEdgePalette(isDark);

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
          <div className="flex shrink-0 items-center gap-2">
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
            {headerEdit === "button" ? (
              editHref ? (
                <Link href={editHref} className={buttonVariants({ size: "sm" })} data-testid="split-run-canvas-edit">
                  {editLabel}
                </Link>
              ) : (
                <Button type="button" size="sm" onClick={onEdit} data-testid="split-run-canvas-edit">
                  {editLabel}
                </Button>
              )
            ) : (
              <CanvasOverflowMenu title={canvas.title} editHref={editHref} />
            )}
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
          elementsSelectable
          selectNodesOnDrag={false}
          panOnScroll
          zoomOnScroll
          onNodeClick={onNodeClick}
          onPaneClick={() => onSelect(null)}
          proOptions={{ hideAttribution: true }}
          colorMode={isDark ? "dark" : "light"}
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
      <DropdownMenuContent align="end" className="w-max min-w-40">
        {editHref ? (
          <DropdownMenuItem asChild data-testid="split-run-canvas-edit">
            <Link href={editHref}>
              <Pencil className="h-3.5 w-3.5" aria-hidden />
              Edit Automation
            </Link>
          </DropdownMenuItem>
        ) : (
          <DropdownMenuItem disabled data-testid="split-run-canvas-edit">
            <Pencil className="h-3.5 w-3.5" aria-hidden />
            Edit Automation
          </DropdownMenuItem>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
