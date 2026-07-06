import { Heading } from "@/components/Heading/heading";
import type { ComponentsEdge, SuperplaneComponentsNode } from "@/api-client";
import { Link } from "react-router-dom";
import { appPath } from "@/lib/appPaths";
import { appDarkModeClasses } from "@/lib/appDarkModeClasses";
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
  canUpdateCanvases: boolean;
  canDeleteCanvases: boolean;
  permissionsLoading: boolean;
}

export function CanvasCardsGrid({
  canvases,
  canvasFolders,
  organizationId,
  onEditCanvas,
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
  canUpdateCanvases: boolean;
  canDeleteCanvases: boolean;
  permissionsLoading: boolean;
}

function CanvasCard({
  canvas,
  canvasFolders,
  organizationId,
  onEdit,
  canUpdateCanvases,
  canDeleteCanvases,
  permissionsLoading,
}: CanvasCardProps) {
  const canvasHref = appPath(organizationId, canvas.id);
  const previewNodes = canvas.nodes || [];
  const previewEdges = canvas.edges || [];

  return (
    <div
      className={cn(
        "relative flex min-h-40 h-full flex-col rounded-md bg-white shadow-sm transition-[box-shadow,outline-color] cursor-pointer hover:shadow-md",
        "outline outline-1 -outline-offset-1 outline-slate-950/10 hover:outline-slate-950/15 dark:outline-gray-600/25 dark:hover:outline-gray-600/40",
        appDarkModeClasses.surfaceRaised,
      )}
    >
      <Link to={canvasHref} aria-label={`Open canvas ${canvas.name}`} className="absolute inset-0 rounded-md" />
      <div className="pointer-events-none relative flex flex-1 flex-col">
        <div className="p-3 pb-0">
          <div className="flex items-start justify-between gap-3">
            <div className="flex flex-col flex-1 min-w-0">
              <Heading
                level={3}
                className="mb-0 line-clamp-2 !text-base font-medium text-gray-800 transition-colors !leading-6 dark:text-white"
              >
                <span className="truncate">{canvas.name}</span>
              </Heading>
            </div>
            <div className="pointer-events-auto">
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

        <div className="border-t border-gray-950/10 px-3 pb-3 pt-3 dark:border-gray-700/70">
          <p className="text-left text-[11px] leading-none text-gray-500 dark:text-gray-400">
            Created by {canvas.createdBy.name}, on {canvas.createdAt}
          </p>
        </div>
      </div>
    </div>
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
              fill="currentColor"
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
