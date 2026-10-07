import type { CSSProperties } from "react";
import React, { useCallback } from "react";
import type { EdgeProps } from "@xyflow/react";
import { BaseEdge, EdgeLabelRenderer, useReactFlow } from "@xyflow/react";
import { useTheme } from "@/contexts/useTheme";
import {
  DEFAULT_FACTORY_EDGE_STROKE,
  LONG_EDGE_PATH_LENGTH,
  TOUCHING_EDGE_STROKE,
  TOUCHING_EDGE_STROKE_DARK,
  estimatePathLength,
  getCanvasEdgePath,
  getFlowArrowPoints,
  getPointAlongPath,
} from "./edgePath";
import { CircleX } from "lucide-react";

interface CustomEdgeData {
  isHovered?: boolean;
  canDelete?: boolean;
  onDelete?: (edgeId: string) => void;
  /** Factory run display: original channel name shown as an edge badge. */
  channelLabel?: string;
  /** Factory run long, cross-column, or feedback edge: route clear of node cards. */
  routeGutterX?: number;
  /** Feedback-only Y stagger for adjacent return routes. */
  routeOffsetY?: number;
  /** Factory run: this edge geometrically touches/crosses another. */
  touchesOtherEdge?: boolean;
  /** Factory run: longer edge in a touch pair — darker stroke + label border. */
  contrastStroke?: boolean;
}

/** Near source on the stroke — on-edge, not mid-path / merge. */
const CHANNEL_LABEL_PATH_FRACTION = 0.32;

function DeleteEdgeControls({
  canDelete,
  edgePath,
  labelX,
  labelY,
  shouldShowIcon,
  onDelete,
}: {
  canDelete: boolean;
  edgePath: string;
  labelX: number;
  labelY: number;
  shouldShowIcon: boolean;
  onDelete: (event: React.PointerEvent<SVGPathElement>) => void;
}) {
  if (!canDelete) {
    return null;
  }

  return (
    <>
      <path
        data-testid="edge-delete-hit-area"
        d={edgePath}
        fill="none"
        stroke="transparent"
        strokeWidth={20}
        style={{ cursor: "pointer", pointerEvents: "stroke" }}
        onPointerDown={onDelete}
      />
      <EdgeLabelRenderer>
        <div
          style={{
            position: "absolute",
            transform: `translate(-50%, -50%) translate(${labelX}px,${labelY}px)`,
            width: "32px",
            height: "32px",
            zIndex: 1001,
            pointerEvents: "none",
            opacity: shouldShowIcon ? 1 : 0,
            transition: "opacity 150ms ease",
          }}
          className="edge-label nodrag nopan flex items-center justify-center"
        >
          <div className="rounded-full bg-slate-100 p-1 dark:bg-gray-800" data-testid="edge-delete-icon">
            <CircleX size={18} className="text-slate-500 dark:text-gray-400" />
          </div>
        </div>
      </EdgeLabelRenderer>
    </>
  );
}

/** Open chevron (two strokes), not a filled triangle. */
function FlowChevronMarkers({
  arrows,
  color,
}: {
  arrows: Array<{ x: number; y: number; angleDeg: number }>;
  color: string;
}) {
  if (arrows.length === 0) return null;

  return (
    <EdgeLabelRenderer>
      {arrows.map((arrow, index) => (
        <div
          key={`${arrow.x}-${arrow.y}-${index}`}
          data-testid="edge-flow-arrow"
          style={{
            position: "absolute",
            transform: `translate(-50%, -50%) translate(${arrow.x}px,${arrow.y}px) rotate(${arrow.angleDeg}deg)`,
            pointerEvents: "none",
            zIndex: 0,
            color,
            opacity: 0.95,
          }}
          className="nodrag nopan select-none"
        >
          <svg width="16" height="16" viewBox="0 0 12 12" aria-hidden="true">
            <path
              d="M4 2.5 L8.5 6 L4 9.5"
              fill="none"
              stroke="currentColor"
              strokeWidth="1.6"
              strokeLinecap="round"
              strokeLinejoin="round"
            />
          </svg>
        </div>
      ))}
    </EdgeLabelRenderer>
  );
}

function readCustomEdgeData(data: EdgeProps["data"]) {
  const edgeData = data as CustomEdgeData | undefined;
  const routeGutterX =
    typeof edgeData?.routeGutterX === "number" && Number.isFinite(edgeData.routeGutterX)
      ? edgeData.routeGutterX
      : undefined;
  const routeOffsetY =
    typeof edgeData?.routeOffsetY === "number" && Number.isFinite(edgeData.routeOffsetY)
      ? edgeData.routeOffsetY
      : undefined;
  return {
    isHovered: edgeData?.isHovered === true,
    canDelete: edgeData?.canDelete === true,
    onDeleteEdge: edgeData?.onDelete,
    channelLabel: typeof edgeData?.channelLabel === "string" ? edgeData.channelLabel.trim() : "",
    touchesOtherEdge: edgeData?.touchesOtherEdge === true,
    contrastStroke: edgeData?.contrastStroke === true,
    routeGutterX,
    routeOffsetY,
  };
}

function resolveTouchingEdgeStroke(isDark: boolean): string {
  return isDark ? TOUCHING_EDGE_STROKE_DARK : TOUCHING_EDGE_STROKE;
}

function resolveEdgeLineStroke(contrastStroke: boolean, styleStroke: CSSProperties["stroke"], isDark: boolean): string {
  if (contrastStroke) {
    return resolveTouchingEdgeStroke(isDark);
  }
  if (typeof styleStroke === "string") {
    return styleStroke;
  }
  return DEFAULT_FACTORY_EDGE_STROKE;
}

function buildEdgeStrokeStyle(style: CSSProperties, selected: boolean | undefined, isHovered: boolean): CSSProperties {
  return {
    strokeWidth: selected ? 3 : style.strokeWidth || 3,
    pointerEvents: "visibleStroke",
    ...(style.strokeDasharray ? { strokeDasharray: style.strokeDasharray } : {}),
    ...(selected || isHovered ? { strokeOpacity: 1 } : {}),
  };
}

export const CustomEdge = React.memo(function CustomEdge({
  id,
  sourceX,
  sourceY,
  targetX,
  targetY,
  sourcePosition,
  targetPosition,
  style = {},
  selected,
  data,
}: EdgeProps) {
  const { setEdges } = useReactFlow();
  const { resolvedTheme } = useTheme();
  const isDark = resolvedTheme === "dark";
  const {
    isHovered,
    canDelete,
    onDeleteEdge,
    channelLabel,
    touchesOtherEdge,
    contrastStroke,
    routeGutterX,
    routeOffsetY,
  } = readCustomEdgeData(data);

  const [edgePath, labelX, labelY] = getCanvasEdgePath({
    sourceX,
    sourceY,
    sourcePosition,
    targetX,
    targetY,
    targetPosition,
    routeGutterX,
    routeOffsetY,
  });

  const pathLength = estimatePathLength(edgePath);
  const isLongEdge = pathLength >= LONG_EDGE_PATH_LENGTH || routeGutterX != null;
  const flowArrows = isLongEdge ? getFlowArrowPoints(edgePath) : [];
  // Near-source only when this edge crosses/touches another — otherwise keep mid-path label.
  const channelPoint =
    channelLabel && touchesOtherEdge ? getPointAlongPath(edgePath, CHANNEL_LABEL_PATH_FRACTION) : null;
  const channelLabelX = channelPoint?.x ?? labelX;
  const channelLabelY = channelPoint?.y ?? labelY;

  const handleEdgeDelete = useCallback(() => {
    if (onDeleteEdge) {
      onDeleteEdge(id);
      return;
    }

    setEdges((edges) => edges.filter((edge) => edge.id !== id));
  }, [id, onDeleteEdge, setEdges]);

  const edgeStyle = buildEdgeStrokeStyle(style, selected, isHovered);
  const shouldShowIcon = canDelete && (isHovered || selected === true);
  const handleDeletePointerDown = useCallback(
    (event: React.PointerEvent<SVGPathElement>) => {
      if (event.button > 0) return;
      event.stopPropagation();
      handleEdgeDelete();
    },
    [handleEdgeDelete],
  );

  const lineStroke = resolveEdgeLineStroke(contrastStroke, style.stroke, isDark);
  const touchingStroke = resolveTouchingEdgeStroke(isDark);

  return (
    <>
      <BaseEdge path={edgePath} style={edgeStyle} interactionWidth={20} />
      <FlowChevronMarkers arrows={flowArrows} color={lineStroke} />
      {channelLabel && !shouldShowIcon ? (
        <EdgeLabelRenderer>
          <div
            data-testid="edge-channel-label"
            style={{
              position: "absolute",
              transform: `translate(-50%, -50%) translate(${channelLabelX}px,${channelLabelY}px)`,
              pointerEvents: "none",
              zIndex: 1000,
              ...(contrastStroke ? { borderColor: touchingStroke, color: touchingStroke } : {}),
            }}
            className="nodrag nopan rounded-md border border-border bg-card px-1.5 py-0.5 text-[10px] font-medium text-muted-foreground shadow-sm"
          >
            {channelLabel}
          </div>
        </EdgeLabelRenderer>
      ) : null}
      <DeleteEdgeControls
        canDelete={canDelete}
        edgePath={edgePath}
        labelX={labelX}
        labelY={labelY}
        shouldShowIcon={shouldShowIcon}
        onDelete={handleDeletePointerDown}
      />
    </>
  );
});
