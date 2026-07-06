import { Heading } from "@/components/Heading/heading";
import type { ComponentsEdge, SuperplaneComponentsNode } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Link } from "react-router-dom";
import { Pin, Star } from "lucide-react";
import type { ReactNode } from "react";
import { appPath } from "@/lib/appPaths";
import { cn } from "@/lib/utils";
import { CanvasActionsMenu } from "./CanvasActionsMenu";
import { CanvasCardDescription } from "./CanvasCardDescription";
import type { CanvasCardData, CanvasFolderData } from "./types";

type CanvasMiniMapPoint = {
  x: number;
  y: number;
};

type PositionedCanvasNode = SuperplaneComponentsNode & {
  id: string;
  position: CanvasMiniMapPoint;
};

interface CanvasCardsGridProps {
  canvases: CanvasCardData[];
  canvasFolders: CanvasFolderData[];
  organizationId: string;
  onEditCanvas: (canvas: CanvasCardData) => void;
  onTogglePin: (canvasId: string, pinned: boolean) => void;
  onToggleStar: (canvasId: string, starred: boolean) => void;
  canUpdateCanvases: boolean;
  canDeleteCanvases: boolean;
  permissionsLoading: boolean;
}

export function CanvasCardsGrid({
  canvases,
  canvasFolders,
  organizationId,
  onEditCanvas,
  onTogglePin,
  onToggleStar,
  canUpdateCanvases,
  canDeleteCanvases,
  permissionsLoading,
}: CanvasCardsGridProps) {
  return (
    <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
      {canvases.map((canvas) => (
        <CanvasCard
          key={canvas.id}
          canvas={canvas}
          canvasFolders={canvasFolders}
          organizationId={organizationId}
          onEdit={onEditCanvas}
          onTogglePin={onTogglePin}
          onToggleStar={onToggleStar}
          canUpdateCanvases={canUpdateCanvases}
          canDeleteCanvases={canDeleteCanvases}
          permissionsLoading={permissionsLoading}
        />
      ))}
    </div>
  );
}

interface CanvasCardProps {
  canvas: CanvasCardData;
  canvasFolders: CanvasFolderData[];
  organizationId: string;
  onEdit: (canvas: CanvasCardData) => void;
  onTogglePin: (canvasId: string, pinned: boolean) => void;
  onToggleStar: (canvasId: string, starred: boolean) => void;
  canUpdateCanvases: boolean;
  canDeleteCanvases: boolean;
  permissionsLoading: boolean;
}

function CanvasCard({
  canvas,
  canvasFolders,
  organizationId,
  onEdit,
  onTogglePin,
  onToggleStar,
  canUpdateCanvases,
  canDeleteCanvases,
  permissionsLoading,
}: CanvasCardProps) {
  const canvasHref = appPath(organizationId, canvas.id);
  const previewNodes = canvas.nodes || [];
  const previewEdges = canvas.edges || [];

  return (
    <div className="relative flex min-h-40 h-full flex-col bg-white dark:bg-gray-800 rounded-md outline outline-gray-950/15 hover:shadow-md transition-shadow cursor-pointer">
      <Link to={canvasHref} aria-label={`Open canvas ${canvas.name}`} className="absolute inset-0 rounded-md" />
      <div className="pointer-events-none relative flex flex-1 flex-col">
        <div className="p-3 pb-0">
          <div className="flex items-start justify-between gap-3">
            <div className="flex flex-col flex-1 min-w-0">
              <Heading
                level={3}
                className="mb-0 line-clamp-2 !text-base font-medium text-gray-800 transition-colors !leading-6"
              >
                <span className="truncate">{canvas.name}</span>
              </Heading>
            </div>
            <div className="pointer-events-auto flex items-center gap-1">
              <CanvasPreferenceButton
                active={Boolean(canvas.isPinned)}
                activeLabel={`Unpin app ${canvas.name}`}
                inactiveLabel={`Pin app ${canvas.name}`}
                activeTooltip="Unpin"
                inactiveTooltip="Pin"
                onClick={() => onTogglePin(canvas.id, !canvas.isPinned)}
                icon={<Pin size={15} className={cn(canvas.isPinned && "fill-current")} aria-hidden />}
              />
              <CanvasPreferenceButton
                active={Boolean(canvas.isStarred)}
                activeLabel={`Unstar app ${canvas.name}`}
                inactiveLabel={`Star app ${canvas.name}`}
                activeTooltip="Unstar"
                inactiveTooltip="Star"
                onClick={() => onToggleStar(canvas.id, !canvas.isStarred)}
                icon={<Star size={15} className={cn(canvas.isStarred && "fill-current")} aria-hidden />}
              />
              <CanvasActionsMenu
                canvas={canvas}
                canvasFolders={canvasFolders}
                organizationId={organizationId}
                onEdit={onEdit}
                canUpdateCanvases={canUpdateCanvases}
                canDeleteCanvases={canDeleteCanvases}
                permissionsLoading={permissionsLoading}
              />
            </div>
          </div>

          {canvas.description ? <CanvasCardDescription description={canvas.description} /> : null}
        </div>

        <div className="flex-1">
          <CanvasMiniMap nodes={previewNodes} edges={previewEdges} />
        </div>

        <div className="border-t border-gray-950/10 px-3 pb-3 pt-3 dark:border-white/10">
          <p className="text-left text-[11px] leading-none text-gray-500 dark:text-gray-400">
            Created by {canvas.createdBy.name}, on {canvas.createdAt}
          </p>
        </div>
      </div>
    </div>
  );
}

interface CanvasPreferenceButtonProps {
  active: boolean;
  activeLabel: string;
  inactiveLabel: string;
  activeTooltip: string;
  inactiveTooltip: string;
  icon: ReactNode;
  onClick: () => void;
}

function CanvasPreferenceButton({
  active,
  activeLabel,
  inactiveLabel,
  activeTooltip,
  inactiveTooltip,
  icon,
  onClick,
}: CanvasPreferenceButtonProps) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          size="icon-xs"
          aria-label={active ? activeLabel : inactiveLabel}
          onClick={(event) => {
            event.preventDefault();
            event.stopPropagation();
            onClick();
          }}
          className={cn(
            "rounded-md text-gray-400 hover:bg-gray-100 hover:text-gray-800 dark:hover:bg-gray-700 dark:hover:text-white",
            active &&
              "bg-blue-50 text-blue-600 hover:bg-blue-100 hover:text-blue-700 dark:bg-blue-950 dark:text-blue-300",
          )}
        >
          {icon}
        </Button>
      </TooltipTrigger>
      <TooltipContent>{active ? activeTooltip : inactiveTooltip}</TooltipContent>
    </Tooltip>
  );
}

interface CanvasMiniMapProps {
  nodes?: SuperplaneComponentsNode[];
  edges?: ComponentsEdge[];
}

function CanvasMiniMap({ nodes = [], edges = [] }: CanvasMiniMapProps) {
  const positionedNodes = nodes.filter(hasMiniMapPosition);

  if (!positionedNodes.length) {
    return <div className="h-24 w-full p-4" />;
  }

  const xs = positionedNodes.map((node) => node.position.x);
  const ys = positionedNodes.map((node) => node.position.y);
  const minX = Math.min(...xs);
  const maxX = Math.max(...xs);
  const minY = Math.min(...ys);
  const maxY = Math.max(...ys);
  const padding = 80;
  const width = Math.max(maxX - minX, 200) + padding * 2;
  const height = Math.max(maxY - minY, 200) + padding * 2;
  const viewBox = `${minX - padding} ${minY - padding} ${width} ${height}`;
  const nodeWidth = Math.min(Math.max(width * 0.08, 30), 80);
  const nodeHeight = nodeWidth * 0.45;

  const nodePositions = new Map<string, CanvasMiniMapPoint>();
  positionedNodes.forEach((node) => {
    nodePositions.set(node.id, { x: node.position.x, y: node.position.y });
  });

  const drawableEdges =
    edges?.filter(
      (edge) => edge.sourceId && edge.targetId && nodePositions.has(edge.sourceId) && nodePositions.has(edge.targetId),
    ) || [];

  return (
    <div className="w-full overflow-hidden p-4">
      <svg
        viewBox={viewBox}
        preserveAspectRatio="xMidYMid meet"
        className="h-24 w-full text-gray-500 dark:text-gray-400"
      >
        {drawableEdges.map((edge) => {
          const source = nodePositions.get(edge.sourceId!);
          const target = nodePositions.get(edge.targetId!);
          if (!source || !target) return null;
          return (
            <line
              key={`${edge.sourceId}-${edge.targetId}`}
              x1={source.x}
              y1={source.y}
              x2={target.x}
              y2={target.y}
              stroke="currentColor"
              strokeWidth={6}
              strokeLinecap="round"
              opacity={0.25}
            />
          );
        })}
        {positionedNodes.map((node) => {
          return (
            <rect
              key={node.id}
              x={node.position.x - nodeWidth / 2}
              y={node.position.y - nodeHeight / 2}
              width={nodeWidth}
              height={nodeHeight}
              rx={8}
              ry={8}
              fill="#1f2937"
              opacity={1}
            />
          );
        })}
      </svg>
    </div>
  );
}

function hasMiniMapPosition(node: SuperplaneComponentsNode): node is PositionedCanvasNode {
  return Boolean(node.id) && typeof node.position?.x === "number" && typeof node.position?.y === "number";
}
