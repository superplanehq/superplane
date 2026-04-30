import type { CSSProperties } from "react";
import React, { useCallback } from "react";
import type { EdgeProps } from "@xyflow/react";
import { BaseEdge, EdgeLabelRenderer, getBezierPath, useReactFlow } from "@xyflow/react";
import { CircleX } from "lucide-react";

interface CustomEdgeData {
  isHovered?: boolean;
  canDelete?: boolean;
  onDelete?: (edgeId: string) => void;
}

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
          <div className="rounded-full bg-slate-100 p-1" data-testid="edge-delete-icon">
            <CircleX size={18} className="text-slate-500" />
          </div>
        </div>
      </EdgeLabelRenderer>
    </>
  );
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
  const edgeData = data as CustomEdgeData | undefined;
  const isHovered = edgeData?.isHovered === true;
  const canDelete = edgeData?.canDelete === true;
  const onDeleteEdge = edgeData?.onDelete;

  const [edgePath, labelX, labelY] = getBezierPath({
    sourceX,
    sourceY,
    sourcePosition,
    targetX,
    targetY,
    targetPosition,
  });

  const handleEdgeDelete = useCallback(() => {
    if (onDeleteEdge) {
      onDeleteEdge(id);
      return;
    }

    setEdges((edges) => edges.filter((edge) => edge.id !== id));
  }, [id, onDeleteEdge, setEdges]);

  // Update style based on selection and hover state
  const edgeStyle: CSSProperties = {
    ...style,
    stroke: selected || isHovered ? "#A1AEC0" : style.stroke || "#DEF3FE",
    strokeWidth: selected ? 3 : style.strokeWidth || 3,
    pointerEvents: "visibleStroke",
  };
  const shouldShowIcon = canDelete && (isHovered || selected === true);
  const handleDeletePointerDown = useCallback(
    (event: React.PointerEvent<SVGPathElement>) => {
      if (event.button > 0) return;
      event.stopPropagation();
      handleEdgeDelete();
    },
    [handleEdgeDelete],
  );

  return (
    <>
      <BaseEdge path={edgePath} style={edgeStyle} interactionWidth={20} className={isHovered ? "hovered" : undefined} />
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
