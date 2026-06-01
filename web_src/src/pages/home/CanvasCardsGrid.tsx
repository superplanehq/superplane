import { Heading } from "@/components/Heading/heading";
import type { SuperplaneComponentsEdge, SuperplaneComponentsNode } from "@/api-client";
import { cn } from "@/lib/utils";
import { useCallback, useEffect, useLayoutEffect, useRef, useState, type MouseEvent, type ReactNode } from "react";
import { Link } from "react-router-dom";
import { CanvasActionsMenu } from "./CanvasActionsMenu";
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
  const canvasHref = `/${organizationId}/canvases/${canvas.id}`;
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
                className="mb-0 line-clamp-2 !text-lg font-medium text-gray-800 transition-colors !leading-6"
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

        <div className="border-t border-gray-950/10 px-3 pb-3 pt-3 dark:border-white/10">
          <p className="text-left text-[11px] leading-none text-gray-500 dark:text-gray-400">
            Created by {canvas.createdBy.name}, on {canvas.createdAt}
          </p>
        </div>
      </div>
    </div>
  );
}

function CanvasCardDescription({ description }: { description: string }) {
  const [isExpanded, setIsExpanded] = useState(false);
  const [isOverflowing, setIsOverflowing] = useState(false);
  const [truncateAt, setTruncateAt] = useState(0);
  const containerRef = useRef<HTMLDivElement>(null);
  const measureRef = useRef<HTMLParagraphElement>(null);

  const remeasure = useCallback(() => {
    const element = measureRef.current;
    if (!element) {
      return;
    }

    const lineHeight = Number.parseFloat(getComputedStyle(element).lineHeight);
    const maxHeight = lineHeight * 2;

    const fitsContent = (length: number) => {
      element.replaceChildren();

      const visibleText = length >= description.length ? description : `${description.slice(0, length).trimEnd()} `;
      element.append(document.createTextNode(visibleText));

      if (length < description.length) {
        const toggle = document.createElement("span");
        toggle.textContent = "…more";
        element.append(toggle);
      }

      return element.scrollHeight <= maxHeight + 1;
    };

    if (fitsContent(description.length)) {
      setIsOverflowing(false);
      return;
    }

    setIsOverflowing(true);

    if (isExpanded) {
      return;
    }

    let low = 0;
    let high = description.length;
    let best = 0;

    while (low <= high) {
      const mid = Math.floor((low + high) / 2);
      if (fitsContent(mid)) {
        best = mid;
        low = mid + 1;
      } else {
        high = mid - 1;
      }
    }

    setTruncateAt(best);
  }, [description, isExpanded]);

  useLayoutEffect(() => {
    remeasure();
  }, [remeasure]);

  useEffect(() => {
    const element = containerRef.current;
    if (!element) {
      return;
    }

    const observer = new ResizeObserver(remeasure);
    observer.observe(element);

    return () => observer.disconnect();
  }, [remeasure]);

  const handleToggle = (event: MouseEvent<HTMLButtonElement>) => {
    event.preventDefault();
    event.stopPropagation();
    setIsExpanded((current) => !current);
  };

  const descriptionClassName = "text-left text-sm leading-normal text-gray-800 dark:text-gray-400";

  if (isExpanded) {
    return (
      <div ref={containerRef} className="pointer-events-auto mt-1 mb-3">
        <p ref={measureRef} aria-hidden className={cn("invisible absolute w-full", descriptionClassName)} />
        <p className={descriptionClassName}>
          {isOverflowing ? (
            <>
              {`${description} `}
              <DescriptionToggle onClick={handleToggle}>/ show less</DescriptionToggle>
            </>
          ) : (
            description
          )}
        </p>
      </div>
    );
  }

  return (
    <div ref={containerRef} className="pointer-events-auto relative mt-1 mb-3">
      <p ref={measureRef} aria-hidden className={cn("invisible absolute w-full", descriptionClassName)} />
      <p className={descriptionClassName}>
        {isOverflowing ? (
          <>
            {`${description.slice(0, truncateAt).trimEnd()} `}
            <DescriptionToggle onClick={handleToggle}>…more</DescriptionToggle>
          </>
        ) : (
          description
        )}
      </p>
    </div>
  );
}

function DescriptionToggle({
  children,
  onClick,
}: {
  children: ReactNode;
  onClick: (event: MouseEvent<HTMLButtonElement>) => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="inline text-gray-500 hover:text-gray-700 dark:hover:text-gray-400"
    >
      {children}
    </button>
  );
}

interface CanvasMiniMapProps {
  nodes?: SuperplaneComponentsNode[];
  edges?: SuperplaneComponentsEdge[];
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
