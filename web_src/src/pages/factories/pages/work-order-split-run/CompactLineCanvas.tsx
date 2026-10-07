import { Handle, Position, ReactFlow, Background, useReactFlow, type Node, type NodeProps } from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { Maximize2, MoreHorizontal, Pencil } from "lucide-react";
import { useCallback, useEffect, useMemo, type MouseEvent } from "react";

import { Button } from "@/components/ui/button";
import { buttonVariants } from "@/components/ui/buttonVariants";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useTheme } from "@/contexts/useTheme";
import { FACTORY_HANDLE_STYLE, factoryCanvasBackground, factoryEdgePalette } from "@/lib/factoryCanvasChrome";
import { FACTORY_SIDE_HANDLE_ID, FACTORY_SPINE_HANDLE_ID } from "@/lib/layout/factoryRunLeafLayout";
import { cn } from "@/lib/utils";
import { Link } from "@/components/Link/link";
import { FACTORY_HANDLE_OUTSET_PX } from "@/ui/CanvasPage/Block/handleStyle";
import { CustomEdge } from "@/ui/CanvasPage/CustomEdge";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import { FactoryNodeCardShell } from "@/ui/factoryNodeChrome/FactoryNodeCardShell";
import { FACTORY_NODE_SELECTED_RING_CLASSNAME } from "@/ui/factoryNodeChrome/factoryNodeSelectedRing";
import { FactoryNodeStepList } from "@/ui/factoryNodeChrome/FactoryNodeStepList";

import {
  COMPACT_CANVAS_FIT_SETTLE_MS,
  COMPACT_FIT_VIEW_OPTIONS,
  compactCanvasFitKey,
  compactCanvasNodeFocusRequest,
  shouldFitCompactCanvas,
} from "./compactCanvasFit";
import { compactLineCanvasGraph, type LineNodeData } from "./compactLineCanvasGraph";
import type { SplitRunCanvasModel } from "./splitRunCanvases";

function LineCanvasTargetHandle({ isSideTarget }: { isSideTarget: boolean }) {
  if (isSideTarget) {
    return (
      <Handle
        type="target"
        position={Position.Left}
        style={{
          ...FACTORY_HANDLE_STYLE,
          left: -FACTORY_HANDLE_OUTSET_PX,
          top: "50%",
          transform: "translateY(-50%)",
        }}
      />
    );
  }

  return (
    <Handle
      type="target"
      position={Position.Top}
      style={{
        ...FACTORY_HANDLE_STYLE,
        left: "50%",
        top: -FACTORY_HANDLE_OUTSET_PX,
        transform: "translateX(-50%)",
      }}
    />
  );
}

function LineCanvasNode({ data }: NodeProps<Node<LineNodeData>>) {
  return (
    <div
      className={cn("relative overflow-visible rounded-2xl", data.isSelected && "z-10")}
      data-testid={`split-run-canvas-node-${data.nodeId}`}
      data-selected={data.isSelected ? "true" : undefined}
      aria-selected={data.isSelected}
      onClick={() => data.onSelect?.(data.nodeId)}
    >
      <div
        className={cn("relative overflow-visible rounded-2xl", data.isSelected && FACTORY_NODE_SELECTED_RING_CLASSNAME)}
      >
        <LineCanvasTargetHandle isSideTarget={data.isSideTarget} />
        <FactoryNodeCardShell
          title={data.title}
          subtitle={data.subtitle}
          iconSlug={data.iconSlug}
          iconSrc={data.iconSrc}
          status={data.status}
          metrics={data.metrics}
          selected={data.isSelected}
          body={data.steps.length > 0 ? <FactoryNodeStepList steps={data.steps} /> : undefined}
        />
        {data.isSelected && data.editHref ? <SelectedNodeEditButton href={data.editHref} /> : null}
        <Handle
          id={FACTORY_SPINE_HANDLE_ID}
          type="source"
          position={Position.Bottom}
          style={{
            ...FACTORY_HANDLE_STYLE,
            left: "50%",
            bottom: -FACTORY_HANDLE_OUTSET_PX,
            transform: "translateX(-50%)",
          }}
        />
        <Handle
          id={FACTORY_SIDE_HANDLE_ID}
          type="source"
          position={Position.Right}
          style={{
            ...FACTORY_HANDLE_STYLE,
            right: -FACTORY_HANDLE_OUTSET_PX,
            top: "50%",
            transform: "translateY(-50%)",
          }}
        />
      </div>
    </div>
  );
}

const nodeTypes = { lineCanvas: LineCanvasNode };
const edgeTypes = { custom: CustomEdge };

const EDIT_COMPONENT_LABEL = "Edit component";

function SelectedNodeEditButton({ href }: { href: string }) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Link
          href={href}
          aria-label={EDIT_COMPONENT_LABEL}
          data-testid="split-run-canvas-node-edit"
          className="nodrag nopan absolute top-2 right-2 z-20 flex size-7 items-center justify-center rounded-md bg-foreground text-background shadow-sm transition-colors hover:bg-foreground/90"
          onClick={(event) => event.stopPropagation()}
        >
          <Pencil className="size-3.5" aria-hidden />
        </Link>
      </TooltipTrigger>
      <TooltipContent side="top">{EDIT_COMPONENT_LABEL}</TooltipContent>
    </Tooltip>
  );
}

function CompactCanvasViewport({ contentKey, selectedId }: { contentKey: string; selectedId: string | null }) {
  const { fitView, getNode } = useReactFlow();

  useEffect(() => {
    if (!shouldFitCompactCanvas(contentKey) || selectedId) {
      return;
    }
    const timer = window.setTimeout(() => {
      void fitView(COMPACT_FIT_VIEW_OPTIONS);
    }, COMPACT_CANVAS_FIT_SETTLE_MS);
    return () => window.clearTimeout(timer);
  }, [contentKey, fitView, selectedId]);

  useEffect(() => {
    if (!selectedId) {
      return;
    }
    const timer = window.setTimeout(() => {
      const request = compactCanvasNodeFocusRequest(getNode(selectedId));
      if (!request) {
        return;
      }
      void fitView(request);
    }, COMPACT_CANVAS_FIT_SETTLE_MS);
    return () => window.clearTimeout(timer);
  }, [contentKey, fitView, getNode, selectedId]);

  return null;
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
  headerEdit = "menu",
  editLabel = "Edit Automation",
  onEdit,
  nodeEditHref,
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
  nodeEditHref?: (nodeId: string) => string;
}) {
  const { resolvedTheme } = useTheme();
  const isDark = resolvedTheme === "dark";
  const { nodes, edges } = useMemo(
    () => compactLineCanvasGraph(canvas, selectedId, onSelect, isDark, nodeEditHref),
    [canvas, isDark, nodeEditHref, onSelect, selectedId],
  );
  const background = factoryCanvasBackground(isDark);
  const palette = factoryEdgePalette(isDark);
  const contentKey = compactCanvasFitKey(nodes.map((node) => node.id));

  const onNodeClick = useCallback(
    (_event: MouseEvent, node: Node) => {
      onSelect(node.id);
    },
    [onSelect],
  );

  return (
    <div className="flex h-full min-h-0 w-full flex-col overflow-hidden">
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
      <div className="relative h-full min-h-[18rem] w-full flex-1" data-testid="run-overlay-compact-canvas">
        <ReactFlow
          key={contentKey}
          nodes={nodes}
          edges={edges.map((edge) => ({ ...edge, style: { ...palette.default, ...edge.style } }))}
          nodeTypes={nodeTypes}
          edgeTypes={edgeTypes}
          fitView
          fitViewOptions={COMPACT_FIT_VIEW_OPTIONS}
          minZoom={0.1}
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
          defaultEdgeOptions={{ type: "custom", style: palette.default }}
        >
          <CompactCanvasViewport contentKey={contentKey} selectedId={selectedId} />
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
